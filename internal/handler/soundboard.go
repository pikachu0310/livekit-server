package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/go-audio/wav"
	"github.com/google/uuid"
	"github.com/hajimehoshi/go-mp3"
	"github.com/jfreymuth/oggvorbis"
	"github.com/labstack/echo/v4"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pikachu0310/livekit-server/internal/pkg/util"
	"github.com/pikachu0310/livekit-server/openapi/models"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// PostSoundboard handles uploading a short audio file (<=20s) to S3, storing metadata in DB
// POST /soundboard
func (h *Handler) PostSoundboard(c echo.Context) error {
	userId, err := util.GetTraqUserID(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error: AuthTraQClient": err.Error(),
		})
	}

	// multipart フォームから取得
	file, err := c.FormFile("audio")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "audio file is required (multipart form field: 'audio')",
		})
	}

	// soundName
	soundName := c.FormValue("soundName")
	if soundName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "soundName is required (multipart form field: 'soundName')",
		})
	}

	// stampId
	stampId := c.FormValue("stampId")
	if stampId == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "stampId is required (multipart form field: 'stampId')",
		})
	}
	if !h.repo.CheckStampExistence(stampId) {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "stampId is invalid",
		})
	}

	// Content-Type が audio/ で始まらない場合は 400
	contentType := file.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "audio/") {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("invalid content-type: %s", contentType),
		})
	}

	// ファイルを読み込み
	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to open uploaded file: %v", err),
		})
	}
	defer src.Close()
	fileBytes, err := io.ReadAll(src)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to read file bytes: %v", err),
		})
	}

	// 拡張子を取得し、小文字化
	// 例: .mp3, .wav, .ogg など
	ext := strings.ToLower(filepath.Ext(file.Filename))

	// 音声ファイルであり、20秒以内か判定
	if err := checkAudioDuration(fileBytes, ext, 20.0); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	// soundId を生成
	soundId := uuid.NewString()

	// S3へアップロード
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := h.FileService.UploadFile(ctx, fileBytes, soundId); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to upload file: %v", err),
		})
	}

	// DB保存
	if err := h.repo.InsertSoundboardItem(soundId, soundName, stampId, userId); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to insert soundboard item: %v", err),
		})
	}

	resp := models.SoundboardUploadResponse{SoundId: soundId}
	return c.JSON(http.StatusOK, resp)
}

// checkAudioDuration は拡張子(ext)に基づいて対応ライブラリを使い、秒数をチェックする
// mp3 / wav / ogg に対応し、それ以外は "we only support mp3, wav, ogg" エラー
func checkAudioDuration(fileBytes []byte, ext string, maxSeconds float64) error {
	switch ext {
	case ".mp3":
		dur, err := getMp3Duration(fileBytes)
		if err != nil {
			return fmt.Errorf("mp3 decode error: %w", err)
		}
		if dur > maxSeconds {
			return fmt.Errorf("audio is too long (%.1f sec). Must be <= %.0f", dur, maxSeconds)
		}
		return nil

	case ".wav":
		dur, err := getWavDuration(fileBytes)
		if err != nil {
			return fmt.Errorf("wav decode error: %w", err)
		}
		if dur > maxSeconds {
			return fmt.Errorf("audio is too long (%.1f sec). Must be <= %.0f", dur, maxSeconds)
		}
		return nil

	case ".ogg":
		dur, err := getOggDuration(fileBytes)
		if err != nil {
			return fmt.Errorf("ogg decode error: %w", err)
		}
		if dur > maxSeconds {
			return fmt.Errorf("audio is too long (%.1f sec). Must be <= %.0f", dur, maxSeconds)
		}
		return nil

	default:
		return errors.New("we only support .mp3, .wav, .ogg")
	}
}

// getMp3Duration returns duration in seconds for MP3
func getMp3Duration(data []byte) (float64, error) {
	r := bytes.NewReader(data)
	decoder, err := mp3.NewDecoder(r)
	if err != nil {
		return 0, err
	}
	// decoder.Length() はサンプル数
	// decoder.SampleRate() はサンプリングレート(例: 44100)
	sampleRate := float64(decoder.SampleRate())
	totalSamples := float64(decoder.Length())
	if sampleRate <= 0 {
		return 0, errors.New("invalid mp3 sample rate")
	}
	seconds := totalSamples / sampleRate
	return seconds, nil
}

// getWavDuration returns duration in seconds for WAV
func getWavDuration(data []byte) (float64, error) {
	r := bytes.NewReader(data)
	wavDecoder := wav.NewDecoder(r)
	buf, err := wavDecoder.FullPCMBuffer()
	if err != nil {
		return 0, err
	}
	if buf == nil || buf.Format == nil {
		return 0, errors.New("invalid wav format or buffer")
	}
	sampleRate := float64(buf.Format.SampleRate)
	sampleCount := float64(len(buf.Data)) // PCMBufferのサンプル数
	if sampleRate <= 0 {
		return 0, errors.New("invalid wav sample rate")
	}
	seconds := sampleCount / sampleRate
	return seconds, nil
}

// getOggDuration returns duration in seconds for OGG(Vorbis)
func getOggDuration(data []byte) (float64, error) {
	r := bytes.NewReader(data)
	stream, err := oggvorbis.NewReader(r)
	if err != nil {
		return 0, err
	}
	sampleRate := float64(stream.SampleRate())
	sampleCount := float64(stream.Length()) // Total samples
	if sampleRate <= 0 {
		return 0, errors.New("invalid ogg sample rate")
	}
	seconds := sampleCount / sampleRate
	return seconds, nil
}

// PostSoundboardPlay triggers playing an uploaded audio file via LiveKit Ingress
// POST /soundboard/play
func (h *Handler) PostSoundboardPlay(c echo.Context) error {
	// 1) JSON ボディをパース
	var req models.SoundboardPlayRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}
	RoomId, err := uuid.Parse(req.RoomName.String())
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid roomName (must be UUID)",
		})
	}
	if req.SoundId == "" || RoomId == uuid.Nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "soundId and roomName are required",
		})
	}

	// 2) 認証トークンからユーザ情報を取得 (実装は省略/仮)
	//    例: userID, err := h.extractUserIDFromBearer(c)
	//    ここで "ユーザが roomName に参加しているか" を確認
	//    もし参加していないなら 400や403を返す
	//    例: if !h.repo.IsUserInRoom(userID, req.RoomName) { ... }

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 3) S3ファイルキー = soundId として署名付きURL生成
	audioURL, err := h.FileService.GeneratePresignedURL(ctx, req.SoundId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to generate presigned URL: %v", err),
		})
	}

	// 4) Ingressクライアント
	ingressClient := lksdk.NewIngressClient(h.repo.LiveKitHost, h.repo.ApiKey, h.repo.ApiSecret)

	// 5) Ingress リクエスト作成
	ingReq := &livekit.CreateIngressRequest{
		InputType: livekit.IngressInput_URL_INPUT,
		// ここでは SFU内の participantIdentity等をどう扱うかは任意
		// たとえば "soundboard-user" など固定でOK
		Name:                "soundboard-ingress",
		RoomName:            req.RoomName.String(), // openapi_types.UUID → string
		ParticipantIdentity: "soundboard-" + req.SoundId,
		ParticipantName:     "Soundboard " + req.SoundId,
		Url:                 audioURL,
	}

	info, err := ingressClient.CreateIngress(ctx, ingReq)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to create ingress: %v", err),
		})
	}

	// 6) SoundboardPlayResponse にマッピングして返す
	resp := models.SoundboardPlayResponse{
		IngressId: info.IngressId,
		Url:       &info.Url,
		StreamKey: &info.StreamKey,
	}
	return c.JSON(http.StatusOK, resp)
}

// GetSoundboardList returns an array of SoundboardItem (soundId, soundName, stampId)
// GET /soundboard
func (h *Handler) GetSoundboardList(c echo.Context) error {
	// 1) DBからサウンドボード一覧を取得
	items, err := h.repo.GetAllSoundboards()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to get soundboard list: %v", err),
		})
	}

	// 2) models.SoundboardListResponse = []SoundboardItem に変換
	var resp models.SoundboardListResponse
	for _, it := range items {
		resp = append(resp, models.SoundboardItem{
			SoundId:   it.SoundID,
			SoundName: it.SoundName,
			StampId:   it.StampID,
			CreatorId: it.CreatorID,
		})
	}

	return c.JSON(http.StatusOK, resp)
}

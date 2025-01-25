package handler

import (
	"context"
	"fmt"
	"github.com/pikachu0310/livekit-server/internal/pkg/util"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"

	"github.com/pikachu0310/livekit-server/openapi/models"
)

// PostSoundboard handles uploading a short audio file (<=15s) to S3, storing metadata in DB
// POST /soundboard
func (h *Handler) PostSoundboard(c echo.Context) error {
	userId, err := util.GetTraqUserID(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error: AuthTraQClient": err.Error(),
		})
	}

	// 1) multipart フォームから取得
	file, err := c.FormFile("audio")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "audio file is required (multipart form field: 'audio')",
		})
	}
	soundName := c.FormValue("soundName")
	if soundName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "soundName is required (multipart form field: 'soundName')",
		})
	}
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

	// 2) ファイルを読み込み
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
	// TODO: ファイルが20秒以下かどうかの検証 (省略)

	// 3) サウンドID を生成
	soundId := uuid.NewString()

	// 4) S3へアップロード
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = h.FileService.UploadFile(ctx, fileBytes, soundId) // ファイル名としてsoundIdを使用
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to upload file: %v", err),
		})
	}

	// 5) DBに (soundId, soundName, stampId=空) などを保存
	err = h.repo.InsertSoundboardItem(soundId, soundName, stampId, userId) // stampId はまだ未指定
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to insert soundboard item: %v", err),
		})
	}

	// 6) SoundboardUploadResponse を返す
	resp := models.SoundboardUploadResponse{
		SoundId: soundId,
	}
	return c.JSON(http.StatusOK, resp)
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

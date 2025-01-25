package handler

import (
	"bytes"
	"context"
	"fmt"
	"github.com/livekit/protocol/livekit"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	awscreds "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/labstack/echo/v4"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

// ------------------------------------------------------------------------------
// ここから FileService の実装
// ------------------------------------------------------------------------------
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// 環境変数から読み込む設定値 (S3 互換ストレージ)
var (
	bucketName      = getEnv("AWS_BUCKET_NAME", "sdk-example")
	accountId       = getEnv("AWS_ACCOUNT_ID", "")
	accessKeyId     = getEnv("AWS_ACCESS_KEY_ID", "")
	accessKeySecret = getEnv("AWS_ACCESS_KEY_SECRET", "")
	s3Endpoint      = getEnv("AWS_ENDPOINT", "http://localhost:9000") // 例: MinIO or R2等
)

// FileService は S3 互換ストレージにファイルをアップロード＆取得等を行う
type FileService struct {
	s3Client        *s3.Client
	s3PresignClient *s3.PresignClient
}

// NewFileService は FileService を初期化して返す
func NewFileService() *FileService {
	s3Client := connectS3()
	return &FileService{
		s3Client:        s3Client,
		s3PresignClient: s3.NewPresignClient(s3Client),
	}
}

// connectS3 で S3クライアントを生成する
func connectS3() *s3.Client {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(
			awscreds.NewStaticCredentialsProvider(accessKeyId, accessKeySecret, ""),
		),
		config.WithRegion("auto"), // R2などRegion不要の場合は適宜
	)
	if err != nil {
		log.Fatal("failed to load AWS config:", err)
	}

	// 例: Cloudflare R2 の場合は .BaseEndpoint を指定
	// MinIO や他のサービスなら Endpoints やシグネチャV4設定など要確認
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true // MinIOなどでパススタイルを使うならtrue
		// R2の場合:
		// o.BaseEndpoint = aws.String(
		//    fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountId),
		// )
		// or
		o.EndpointResolver = s3.EndpointResolverFromURL(s3Endpoint)
	})
	return client
}

// uploadFile は音声バイト列を S3 バケットにアップロード
func (fs *FileService) uploadFile(ctx context.Context, file []byte, fileName string) error {
	objectKeyParts := strings.Split(fileName, ".")
	ext := "." + objectKeyParts[len(objectKeyParts)-1]
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err := fs.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &bucketName,
		Key:         &fileName,
		Body:        bytes.NewReader(file),
		ContentType: &contentType,
	})
	return err
}

// generatePresignedURL はダウンロード用の署名付きURLを生成
func (fs *FileService) generatePresignedURL(ctx context.Context, fileName string) (string, error) {
	res, err := fs.s3PresignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucketName,
		Key:    &fileName,
	}, func(opts *s3.PresignOptions) {
		// 有効期限
		opts.Expires = 3 * time.Hour
	})
	if err != nil {
		return "", err
	}
	return res.URL, nil
}

// ------------------------------------------------------------------------------
// エンドポイント: POST /soundboard/play
// Uploadedファイルを LiveKit Ingressで流す
// ------------------------------------------------------------------------------
type SoundboardPlayRequest struct {
	FileName            string `json:"fileName"`
	RoomName            string `json:"roomName"`
	ParticipantIdentity string `json:"participantIdentity"`
	ParticipantName     string `json:"participantName"`
}

// PostSoundboardPlay triggers playing an uploaded audio file via LiveKit Ingress
func (h *Handler) PostSoundboardPlay(c echo.Context) error {
	// 1) JSONリクエストをパース
	var req SoundboardPlayRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}
	if req.FileName == "" || req.RoomName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "fileName and roomName are required",
		})
	}

	// 2) ファイルの署名付きURLを取得
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	audioURL, err := h.fileService.generatePresignedURL(ctx, req.FileName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to generate presigned URL: %v", err),
		})
	}

	// 3) LiveKitの IngressClient を初期化
	ingressClient := lksdk.NewIngressClient(h.repo.LiveKitHost, h.repo.ApiKey, h.repo.ApiSecret)

	// 4) Ingressリクエストを作成
	ingressRequest := &livekit.CreateIngressRequest{
		InputType:           livekit.IngressInput_URL_INPUT,
		Name:                "soundboard-ingress",
		RoomName:            req.RoomName,
		ParticipantIdentity: req.ParticipantIdentity,
		ParticipantName:     req.ParticipantName,
		Url:                 audioURL, // S3の署名付きURL
	}

	// 5) Ingressの作成APIを呼ぶ
	info, err := ingressClient.CreateIngress(ctx, ingressRequest)
	if err != nil {
		fmt.Printf("failed to create ingress: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to create ingress: %v", err),
		})
	}

	// info には IngressId, RTMP URL などが入る
	return c.JSON(http.StatusOK, info)
}

// ------------------------------------------------------------------------------
// エンドポイント: POST /soundboard
// 15秒程度の短い音声ファイルをアップロード
// ------------------------------------------------------------------------------
func (h *Handler) PostSoundboard(c echo.Context) error {
	// 1) ファイル取得 (multipart/form-data, field: "audio")
	file, err := c.FormFile("audio")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "audio file is required (multipart form field: 'audio')",
		})
	}

	// 2) ファイルを開く
	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to open uploaded file: %v", err),
		})
	}
	defer src.Close()

	// 3) 読み取り
	fileBytes, err := io.ReadAll(src)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to read file bytes: %v", err),
		})
	}

	// (必要なら再生時間などのチェックを実装)

	// 4) S3へアップロード
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = h.fileService.uploadFile(ctx, fileBytes, file.Filename)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to upload file: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":  "uploaded",
		"fileName": file.Filename,
	})
}

package repository

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/pikachu0310/livekit-server/internal/pkg/config"
	"log"
	"mime"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config" // ★別名インポート
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// FileService は自作の S3Config を参照
type FileService struct {
	s3Client        *s3.Client
	s3PresignClient *s3.PresignClient
	cfg             *config.S3Config // ★ 自作のS3Config
}

// NewFileService は FileService を初期化
func NewFileService(cfg *config.S3Config) *FileService {
	s3Client := connectS3(cfg)
	return &FileService{
		cfg:             cfg,
		s3Client:        s3Client,
		s3PresignClient: s3.NewPresignClient(s3Client),
	}
}

func connectS3(cfg *config.S3Config) *s3.Client {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.AccessKeySecret, "")),
		awsconfig.WithRegion("auto"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// KOKOMITE
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", "WAKANNNEEE"))
	})
	// 以下のようにも書ける? wakaranai
	/*
		client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.UsePathStyle = true
			o.EndpointResolver = s3.EndpointResolverFromURL(cfg.S3Endpoint)
		})
	*/
	return client
}

// UploadFile は音声バイト列を S3 バケットにアップロード
func (fs *FileService) UploadFile(ctx context.Context, file []byte, fileName string) error {
	parts := strings.Split(fileName, ".")
	ext := "." + parts[len(parts)-1]
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err := fs.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(fs.cfg.BucketName),
		Key:         aws.String(fileName),
		Body:        bytes.NewReader(file),
		ContentType: aws.String(contentType),
	})
	return err
}

// GeneratePresignedURL はダウンロード用の署名付きURLを生成
func (fs *FileService) GeneratePresignedURL(ctx context.Context, fileName string) (string, error) {
	res, err := fs.s3PresignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(fs.cfg.BucketName),
		Key:    aws.String(fileName),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = 3 * time.Hour
	})
	if err != nil {
		return "", err
	}
	return res.URL, nil
}

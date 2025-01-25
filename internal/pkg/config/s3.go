package config

type S3Config struct {
	BucketName      string
	AccountID       string
	AccessKeyID     string
	AccessKeySecret string
	S3Endpoint      string
}

// NewS3Config は環境変数等から設定を組み立てて返す
func NewS3Config() *S3Config {
	return &S3Config{
		BucketName:      getEnv("AWS_BUCKET_NAME", "sdk-example"),
		AccountID:       getEnv("AWS_ACCOUNT_ID", ""),
		AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
		AccessKeySecret: getEnv("AWS_ACCESS_KEY_SECRET", ""),
		S3Endpoint:      getEnv("AWS_ENDPOINT", "http://localhost:9000"),
	}
}

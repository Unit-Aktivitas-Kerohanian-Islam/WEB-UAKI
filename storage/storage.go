package storage

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type StorageService struct {
	Client   *s3.Client
	Bucket   string
	Endpoint string
}

// NewStorageService inisialisasi MinIO client dari ENV
// NewStorageService inisialisasi MinIO client dari ENV
func NewStorageService() *StorageService {
	endpoint := os.Getenv("MINIO_ENDPOINT")
	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	secretKey := os.Getenv("MINIO_SECRET_KEY")
	bucket := os.Getenv("MINIO_BUCKET")

	if endpoint == "" || accessKey == "" || secretKey == "" || bucket == "" {
		log.Fatal("❌ MinIO credentials/env belum di-set (cek .env atau environment variables)")
	}

	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		),
		config.WithEndpointResolver(aws.EndpointResolverFunc(
			func(service, region string) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:           endpoint,
					SigningRegion: "us-east-1",
				}, nil
			})),
	)
	if err != nil {
		log.Fatalf("❌ gagal load config: %v", err)
	}

	// ✅ pakai path style biar nggak jadi article.localhost
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	return &StorageService{
		Client:   client,
		Bucket:   bucket,
		Endpoint: endpoint,
	}
}

func (s *StorageService) ExtractObjectKey(urlStr, bucket string) string {
	// Cari bagian setelah "<bucket>/"
	parts := strings.Split(urlStr, bucket+"/")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

// Upload file ke MinIO
func (s *StorageService) Upload(ctx context.Context, key string, data []byte) (string, error) {
	_, err := s.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:             &s.Bucket,
		Key:                &key,
		Body:               bytes.NewReader(data),
		ContentType:        aws.String(detectContentType(key)),
		ContentDisposition: aws.String("inline"),
	})
	if err != nil {
		return "", err
	}

	// URL file hasil upload
	u := url.URL{
		Scheme: "http",
		Host:   s.Endpoint[len("http://"):], // buang prefix http://
		Path:   fmt.Sprintf("%s/%s", s.Bucket, key),
	}
	return u.String(), nil
}

func detectContentType(key string) string {
	lower := strings.ToLower(key)
	switch {
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".gif"):
		return "image/gif"
	case strings.HasSuffix(lower, ".webp"):
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

// Delete file dari MinIO berdasarkan key
func (s *StorageService) Delete(ctx context.Context, key string) error {
	_, err := s.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &s.Bucket,
		Key:    &key,
	})
	if err != nil {
		return fmt.Errorf("failed to delete object %s: %v", key, err)
	}
	return nil
}

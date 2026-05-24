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
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type StorageService struct {
	Client   *s3.Client
	Bucket   string
	Endpoint string
}

// NewStorageService inisialisasi Biznet NEO Object Storage client dari ENV
// Kompatibel dengan S3 API — menggunakan AWS SDK v2
func NewStorageService() *StorageService {
	endpoint := os.Getenv("NOS_ENDPOINT")
	accessKey := os.Getenv("NOS_ACCESS_KEY")
	secretKey := os.Getenv("NOS_SECRET_KEY")
	bucket := os.Getenv("NOS_BUCKET")
	region := os.Getenv("NOS_REGION")

	if region == "" {
		region = "jkt-1" // default ke Jakarta region Biznet NEO
	}

	if endpoint == "" || accessKey == "" || secretKey == "" || bucket == "" {
		log.Fatal("❌ Biznet NEO Object Storage credentials belum di-set (cek NOS_ENDPOINT, NOS_ACCESS_KEY, NOS_SECRET_KEY, NOS_BUCKET di .env)")
	}

	signingRegion := region

	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		),
		config.WithEndpointResolver(aws.EndpointResolverFunc(
			func(service, reg string) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:           endpoint,
					SigningRegion: signingRegion,
				}, nil
			})),
	)
	if err != nil {
		log.Fatalf("❌ gagal load config Biznet NEO Object Storage: %v", err)
	}

	// ✅ Pakai path-style URL: https://nos.jkt-1.neo.id/<bucket>/<key>
	// Karena Biznet NEO mendukung path-style access
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	return &StorageService{
		Client:   client,
		Bucket:   bucket,
		Endpoint: endpoint,
	}
}

// ExtractObjectKey mengambil object key dari full URL file
func (s *StorageService) ExtractObjectKey(urlStr, bucket string) string {
	// Cari bagian setelah "<bucket>/"
	parts := strings.Split(urlStr, bucket+"/")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

// Upload file ke Biznet NEO Object Storage
// Mengembalikan public URL file yang bisa langsung diakses (bucket harus public)
func (s *StorageService) Upload(ctx context.Context, key string, data []byte) (string, error) {
	_, err := s.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:             &s.Bucket,
		Key:                &key,
		Body:               bytes.NewReader(data),
		ContentType:        aws.String(detectContentType(key)),
		ContentDisposition: aws.String("inline"),
		ACL:                types.ObjectCannedACLPublicRead, // ✅ file bisa diakses publik via URL
	})
	if err != nil {
		return "", err
	}

	// ✅ Gunakan url.Parse agar scheme/host tidak kacau
	parsed, err := url.Parse(s.Endpoint)
	if err != nil {
		return "", fmt.Errorf("invalid endpoint: %v", err)
	}

	// Pastikan scheme ada
	if parsed.Scheme == "" {
		parsed.Scheme = "https"
	}
	if parsed.Host == "" {
		parsed.Host = parsed.Path
		parsed.Path = ""
	}

	// ✅ Bentuk public URL: https://nos.jkt-1.neo.id/<bucket>/<key>
	parsed.Path = fmt.Sprintf("/%s/%s", s.Bucket, key)

	return parsed.String(), nil
}

// detectContentType mendeteksi MIME type berdasarkan ekstensi file
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
	case strings.HasSuffix(lower, ".pdf"):
		return "application/pdf"
	case strings.HasSuffix(lower, ".svg"):
		return "image/svg+xml"
	default:
		return "application/octet-stream"
	}
}

// Delete file dari Biznet NEO Object Storage berdasarkan key
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

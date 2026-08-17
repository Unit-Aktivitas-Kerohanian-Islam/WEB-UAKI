package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type StorageService struct {
	BaseDir string
	BaseURL string
}

func NewStorageService() *StorageService {
	// Folder penyimpanan lokal di dalam VM
	baseDir := "./public/uploads"
	
	// Otomatis membuat folder jika belum ada
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		fmt.Printf("⚠️ Gagal membuat folder uploads: %v\n", err)
	}

	return &StorageService{
		BaseDir: baseDir,
		BaseURL: "/uploads",
	}
}

func (s *StorageService) ExtractObjectKey(urlStr string) string {
	parts := strings.Split(urlStr, "/uploads/")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

// Upload file secara streaming langsung ke Hardisk/SSD VM
func (s *StorageService) Upload(ctx context.Context, key string, reader io.Reader) (string, error) {
	fullPath := filepath.Join(s.BaseDir, key)
	
	// Pastikan sub-folder (seperti cv/, articles/) sudah ada
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %v", err)
	}

	outFile, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %v", err)
	}
	defer outFile.Close()

	// io.Copy memindahkan data sedikit demi sedikit, RAM server 2GB dijamin aman
	if _, err := io.Copy(outFile, reader); err != nil {
		return "", fmt.Errorf("failed to save file: %v", err)
	}

	return fmt.Sprintf("%s/%s", s.BaseURL, key), nil
}

// Delete file fisik dari VM
func (s *StorageService) Delete(ctx context.Context, key string) error {
	fullPath := filepath.Join(s.BaseDir, key)
	err := os.Remove(fullPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %v", err)
	}
	return nil
}
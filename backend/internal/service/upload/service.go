package uploadservice

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"mini-store-go/backend/internal/apperror"
	"mini-store-go/backend/internal/config"
)

type File struct {
	Name        string
	ContentType string
	Size        int64
	Reader      io.Reader
}

type UploadedFile struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

type Service struct {
	cfg         config.UploadConfig
	allowedMIME map[string]struct{}
}

func NewService(cfg config.UploadConfig) (*Service, error) {
	if strings.TrimSpace(cfg.StorageDir) == "" {
		return nil, fmt.Errorf("upload storage directory is required")
	}

	if err := os.MkdirAll(cfg.StorageDir, 0o755); err != nil {
		return nil, fmt.Errorf("create upload storage directory: %w", err)
	}

	allowed := make(map[string]struct{}, len(cfg.AllowedMimeTypes))
	for _, item := range cfg.AllowedMimeTypes {
		allowed[strings.TrimSpace(item)] = struct{}{}
	}

	return &Service{
		cfg:         cfg,
		allowedMIME: allowed,
	}, nil
}

func FileFromHeader(header *multipart.FileHeader) (File, error) {
	stream, err := header.Open()
	if err != nil {
		return File{}, err
	}

	return File{
		Name:        header.Filename,
		ContentType: header.Header.Get("Content-Type"),
		Size:        header.Size,
		Reader:      stream,
	}, nil
}

func (s *Service) SaveImage(ctx context.Context, file File) (*UploadedFile, error) {
	if closer, ok := file.Reader.(io.Closer); ok {
		defer closer.Close()
	}

	if err := s.validate(file); err != nil {
		return nil, err
	}

	ext := normalizeExtension(file.Name, file.ContentType)
	now := time.Now().UTC()
	relativeDir := filepath.Join(
		fmt.Sprintf("%04d", now.Year()),
		fmt.Sprintf("%02d", int(now.Month())),
		fmt.Sprintf("%02d", now.Day()),
	)
	fileName := uuid.NewString() + ext
	relativePath := filepath.ToSlash(filepath.Join(relativeDir, fileName))
	fullDir := filepath.Join(s.cfg.StorageDir, relativeDir)

	if err := os.MkdirAll(fullDir, 0o755); err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to create upload directory", err)
	}

	fullPath := filepath.Join(fullDir, fileName)
	target, err := os.Create(fullPath)
	if err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to create upload file", err)
	}
	defer target.Close()

	if _, err := io.Copy(target, file.Reader); err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to save upload file", err)
	}

	select {
	case <-ctx.Done():
		return nil, apperror.Wrap(apperror.CodeInternal, "upload canceled", ctx.Err())
	default:
	}

	basePath := strings.TrimRight(s.cfg.PublicBasePath, "/")
	return &UploadedFile{
		Name:        file.Name,
		URL:         basePath + "/" + relativePath,
		ContentType: file.ContentType,
		Size:        file.Size,
	}, nil
}

func (s *Service) validate(file File) error {
	if strings.TrimSpace(file.Name) == "" || file.Reader == nil {
		return apperror.New(apperror.CodeBadRequest, "invalid upload file")
	}

	if file.Size <= 0 {
		return apperror.New(apperror.CodeBadRequest, "empty upload file")
	}

	if s.cfg.MaxFileSize > 0 && file.Size > s.cfg.MaxFileSize {
		return apperror.WithDetails(
			apperror.New(apperror.CodeBadRequest, "file exceeds size limit"),
			map[string]any{"max_file_size": s.cfg.MaxFileSize},
		)
	}

	if len(s.allowedMIME) > 0 {
		if _, ok := s.allowedMIME[file.ContentType]; !ok {
			return apperror.WithDetails(
				apperror.New(apperror.CodeBadRequest, "unsupported file type"),
				map[string]any{"content_type": file.ContentType},
			)
		}
	}

	return nil
}

func normalizeExtension(name, contentType string) string {
	ext := strings.ToLower(filepath.Ext(name))
	if ext != "" {
		return ext
	}

	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ""
	}
}

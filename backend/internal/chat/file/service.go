package file

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
)

// DefaultMaxQuotaBytes is the storage quota a tenant has before its first
// upload. It mirrors the storage_quotas.max_bytes column default (000018) so
// a fresh tenant sees the same limit before and after its row exists.
const DefaultMaxQuotaBytes int64 = 10737418240 // 10 GB

// Allowed MIME types
var allowedMimeTypes = map[string]bool{
	// Documents
	"application/pdf":    true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/vnd.ms-excel": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         true,
	"application/vnd.ms-powerpoint":                                             true,
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
	"text/plain":    true,
	"text/csv":      true,
	"text/markdown": true,
	// Images
	"image/jpeg":    true,
	"image/png":     true,
	"image/gif":     true,
	"image/webp":    true,
	"image/svg+xml": true,
	// Archives
	"application/zip":             true,
	"application/x-7z-compressed": true,
}

// Service handles file business logic
type Service struct {
	repo         Repository
	store        FileStore
	scanner      FileScanner
	thumbnailGen ThumbnailGenerator
	maxSizeBytes int64
	presignExp   time.Duration
}

// NewService creates a new file service
func NewService(repo Repository, store FileStore, scanner FileScanner, thumbnailGen ThumbnailGenerator, maxSizeMB int) *Service {
	return &Service{
		repo:         repo,
		store:        store,
		scanner:      scanner,
		thumbnailGen: thumbnailGen,
		maxSizeBytes: int64(maxSizeMB) * 1024 * 1024,
		presignExp:   15 * time.Minute,
	}
}

// UploadInput contains the data needed to upload a file
type UploadInput struct {
	ChannelID uuid.UUID
	MessageID *uuid.UUID
	Filename  string
	MimeType  string
	FileSize  int64
	Reader    io.Reader
	UserID    uuid.UUID
}

// Upload uploads a file to storage and creates a DB record
func (s *Service) Upload(ctx context.Context, input UploadInput) (*models.ChatFileWithUploader, error) {
	// Check membership
	isMember, err := s.repo.IsChannelMember(ctx, input.ChannelID, input.UserID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotChannelMember
	}

	// Check channel not archived
	archived, err := s.repo.IsChannelArchived(ctx, input.ChannelID)
	if err != nil {
		return nil, err
	}
	if archived {
		return nil, ErrChannelArchived
	}

	// Validate file size
	if input.FileSize > s.maxSizeBytes {
		return nil, ErrFileTooLarge
	}

	// Validate MIME type
	if !allowedMimeTypes[strings.ToLower(input.MimeType)] {
		return nil, ErrInvalidMimeType
	}

	tenantID, tenantErr := middleware.GetTenantID(ctx)
	if tenantErr != nil {
		return nil, fmt.Errorf("upload file: %w", tenantErr)
	}

	// Check quota. A tenant that has never uploaded a file has no row yet —
	// it gets the code default rather than an error, since IncrementUsedBytes
	// creates the row on first write below.
	quota, err := s.repo.GetStorageQuota(ctx, tenantID)
	if err != nil {
		if !errors.Is(err, ErrQuotaNotFound) {
			return nil, err
		}
		quota = &models.StorageQuota{TenantID: tenantID, MaxBytes: DefaultMaxQuotaBytes}
	}
	if quota.UsedBytes+input.FileSize > quota.MaxBytes {
		return nil, ErrQuotaExceeded
	}

	// Scan file for viruses
	if scanErr := s.scanner.Scan(ctx, input.Reader, input.Filename); scanErr != nil {
		return nil, ErrScanFailed
	}

	// Generate storage key
	fileID := uuid.New()
	storageKey := fmt.Sprintf("channels/%s/files/%s/%s", input.ChannelID, fileID, input.Filename)

	// Upload to storage
	if uploadErr := s.store.Upload(ctx, storageKey, input.Reader, input.FileSize, input.MimeType); uploadErr != nil {
		return nil, uploadErr
	}

	// Generate thumbnail for images
	var thumbnailKey *string
	if s.thumbnailGen.CanGenerate(input.MimeType) {
		thumbReader, thumbErr := s.generateThumbnail(ctx, storageKey)
		if thumbErr != nil {
			slog.Warn("failed to generate thumbnail", "file_id", fileID, "error", thumbErr)
		} else if thumbReader != nil {
			tk := storageKey + ".thumb.jpg"
			// Read thumbnail into buffer to get size
			thumbData, readErr := io.ReadAll(thumbReader)
			if readErr == nil {
				thumbReader2 := strings.NewReader(string(thumbData))
				if uploadErr := s.store.Upload(ctx, tk, thumbReader2, int64(len(thumbData)), "image/jpeg"); uploadErr != nil {
					slog.Warn("failed to upload thumbnail", "file_id", fileID, "error", uploadErr)
				} else {
					thumbnailKey = &tk
				}
			}
		}
	}

	// Create DB record
	chatFile := &models.ChatFile{
		ID:           fileID,
		TenantID:     tenantID,
		MessageID:    input.MessageID,
		ChannelID:    input.ChannelID,
		Filename:     input.Filename,
		MimeType:     input.MimeType,
		FileSize:     input.FileSize,
		StorageKey:   storageKey,
		ThumbnailKey: thumbnailKey,
		UploadedBy:   input.UserID,
		IsDeleted:    false,
		CreatedAt:    time.Now(),
	}

	if createErr := s.repo.CreateFile(ctx, chatFile); createErr != nil {
		// Best-effort cleanup of uploaded file
		_ = s.store.Delete(ctx, storageKey)
		return nil, createErr
	}

	// Increment quota
	if quotaErr := s.repo.IncrementUsedBytes(ctx, tenantID, input.FileSize); quotaErr != nil {
		slog.Error("failed to increment storage quota", "error", quotaErr)
	}

	slog.Info("file uploaded",
		"file_id", fileID,
		"channel_id", input.ChannelID,
		"filename", input.Filename,
		"size", input.FileSize,
		"uploaded_by", input.UserID,
	)

	// Load uploader info
	firstName, lastName, _ := s.repo.GetUserInfo(ctx, input.UserID)

	return &models.ChatFileWithUploader{
		ChatFile:          *chatFile,
		UploaderFirstName: firstName,
		UploaderLastName:  lastName,
	}, nil
}

// generateThumbnail downloads the file from storage and generates a thumbnail
func (s *Service) generateThumbnail(ctx context.Context, storageKey string) (io.Reader, error) {
	reader, err := s.store.Download(ctx, storageKey)
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()

	return s.thumbnailGen.Generate(ctx, reader, "")
}

// GetDownloadURL returns a presigned URL for downloading a file
func (s *Service) GetDownloadURL(ctx context.Context, fileID, userID uuid.UUID) (string, error) {
	file, err := s.repo.GetFileByID(ctx, fileID)
	if err != nil {
		return "", err
	}

	if file.IsDeleted {
		return "", ErrFileDeleted
	}

	// Check membership
	isMember, memberErr := s.repo.IsChannelMember(ctx, file.ChannelID, userID)
	if memberErr != nil {
		return "", memberErr
	}
	if !isMember {
		return "", ErrNotChannelMember
	}

	return s.store.GetPresignedURL(ctx, file.StorageKey, s.presignExp)
}

// GetThumbnailURL returns a presigned URL for downloading a file's thumbnail
func (s *Service) GetThumbnailURL(ctx context.Context, fileID, userID uuid.UUID) (string, error) {
	file, err := s.repo.GetFileByID(ctx, fileID)
	if err != nil {
		return "", err
	}

	if file.IsDeleted {
		return "", ErrFileDeleted
	}

	if file.ThumbnailKey == nil {
		return "", ErrFileNotFound
	}

	// Check membership
	isMember, memberErr := s.repo.IsChannelMember(ctx, file.ChannelID, userID)
	if memberErr != nil {
		return "", memberErr
	}
	if !isMember {
		return "", ErrNotChannelMember
	}

	return s.store.GetPresignedURL(ctx, *file.ThumbnailKey, s.presignExp)
}

// ListChannelFiles returns paginated files for a channel
func (s *Service) ListChannelFiles(ctx context.Context, channelID, userID uuid.UUID, page, pageSize int) ([]*models.ChatFileWithUploader, int, error) {
	// Check membership
	isMember, err := s.repo.IsChannelMember(ctx, channelID, userID)
	if err != nil {
		return nil, 0, err
	}
	if !isMember {
		return nil, 0, ErrNotChannelMember
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	return s.repo.ListChannelFiles(ctx, channelID, pageSize, offset)
}

// Delete soft-deletes a file (uploader or channel admin/owner)
func (s *Service) Delete(ctx context.Context, fileID, userID uuid.UUID) error {
	file, err := s.repo.GetFileByID(ctx, fileID)
	if err != nil {
		return err
	}

	if file.IsDeleted {
		return nil // idempotent
	}

	// Authorization: uploader or channel admin/owner
	if file.UploadedBy != userID {
		role, roleErr := s.repo.GetChannelRole(ctx, file.ChannelID, userID)
		if roleErr != nil {
			return ErrNotAuthorized
		}
		if !role.CanModerate() {
			return ErrNotAuthorized
		}
	}

	// Soft delete in DB
	if deleteErr := s.repo.DeleteFile(ctx, fileID); deleteErr != nil {
		return deleteErr
	}

	// Decrement quota. Uses the file's own tenant rather than resolving one
	// from ctx: the row being deleted already carries the tenant that owns
	// its quota, so no caller of Delete needs to thread one through.
	if quotaErr := s.repo.DecrementUsedBytes(ctx, file.TenantID, file.FileSize); quotaErr != nil {
		slog.Error("failed to decrement storage quota", "error", quotaErr)
	}

	// Async delete from storage (best effort)
	go func() {
		bgCtx := context.Background()
		if delErr := s.store.Delete(bgCtx, file.StorageKey); delErr != nil {
			slog.Error("failed to delete file from storage", "key", file.StorageKey, "error", delErr)
		}
		if file.ThumbnailKey != nil {
			if delErr := s.store.Delete(bgCtx, *file.ThumbnailKey); delErr != nil {
				slog.Error("failed to delete thumbnail from storage", "key", *file.ThumbnailKey, "error", delErr)
			}
		}
	}()

	slog.Info("file deleted",
		"file_id", fileID,
		"channel_id", file.ChannelID,
		"deleted_by", userID,
	)

	return nil
}

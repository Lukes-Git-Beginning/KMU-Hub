package file

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	chatfile "github.com/kmuhub/kmuhub/internal/chat/file"
	"github.com/kmuhub/kmuhub/internal/models"
)

// Service handles document file business logic.
type Service struct {
	repo    Repository
	store   chatfile.FileStore
	maxSize int64
}

// NewService creates a new document file service.
func NewService(repo Repository, store chatfile.FileStore, maxSizeBytes int64) *Service {
	return &Service{
		repo:    repo,
		store:   store,
		maxSize: maxSizeBytes,
	}
}

// UploadInput contains the data needed to upload a document file.
type UploadInput struct {
	FolderID    uuid.UUID
	Filename    string
	MimeType    string
	FileSize    int64
	Reader      io.Reader
	SpaceType   string
	SpaceID     uuid.UUID
	OwnerID     uuid.UUID
}

// Upload uploads a file to MinIO and creates a DB record with an initial version.
func (s *Service) Upload(ctx context.Context, input UploadInput) (*models.DocumentFile, error) {
	// Validate filename
	filename := strings.TrimSpace(input.Filename)
	if filename == "" {
		return nil, ErrFilenameRequired
	}
	if len(filename) > 255 {
		return nil, ErrFilenameTooLong
	}

	// Validate file size
	if input.FileSize <= 0 {
		return nil, ErrFileSizeZero
	}
	if input.FileSize > s.maxSize {
		return nil, ErrFileTooLarge
	}

	fileID := uuid.New()
	now := time.Now()

	// Generate storage key
	storageKey := fmt.Sprintf("documents/%s/%s/%s/%s/%s",
		input.SpaceType, input.SpaceID, input.FolderID, fileID, filename)

	// Upload to object storage
	if err := s.store.Upload(ctx, storageKey, input.Reader, input.FileSize, input.MimeType); err != nil {
		return nil, err
	}

	// Create DB record
	file := &models.DocumentFile{
		ID:             fileID,
		FolderID:       input.FolderID,
		Filename:       filename,
		MimeType:       input.MimeType,
		FileSize:       input.FileSize,
		StorageKey:     storageKey,
		CurrentVersion: 1,
		OwnerID:        input.OwnerID,
		IsFavorite:     false,
		IsDeleted:      false,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.Create(ctx, file); err != nil {
		// Best-effort cleanup of uploaded object
		_ = s.store.Delete(ctx, storageKey)
		return nil, err
	}

	// Create initial version record
	version := &models.DocumentFileVersion{
		ID:            uuid.New(),
		FileID:        fileID,
		VersionNumber: 1,
		StorageKey:    storageKey,
		FileSize:      input.FileSize,
		CreatedBy:     input.OwnerID,
		CreatedAt:     now,
	}
	if err := s.repo.CreateVersion(ctx, version); err != nil {
		slog.Error("failed to create initial version record",
			"file_id", fileID,
			"error", err,
		)
	}

	slog.Info("file uploaded",
		"file_id", fileID,
		"filename", filename,
		"size", input.FileSize,
		"folder_id", input.FolderID,
		"owner_id", input.OwnerID,
	)

	return file, nil
}

// GetByID retrieves a file by ID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*models.DocumentFile, error) {
	file, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if file.IsDeleted {
		return nil, ErrFileDeleted
	}
	return file, nil
}

// List retrieves files matching the given filter.
func (s *Service) List(ctx context.Context, filter ListFilter) ([]*models.DocumentFile, int, error) {
	return s.repo.List(ctx, filter)
}

// Update updates file metadata.
func (s *Service) Update(ctx context.Context, id uuid.UUID, input UpdateInput) error {
	// Validate filename if changing
	if input.Filename != nil {
		name := strings.TrimSpace(*input.Filename)
		if name == "" {
			return ErrFilenameRequired
		}
		if len(name) > 255 {
			return ErrFilenameTooLong
		}
		input.Filename = &name
	}

	if err := s.repo.Update(ctx, id, input); err != nil {
		return err
	}

	slog.Info("file updated",
		"file_id", id,
	)
	return nil
}

// Delete soft-deletes a file. Does NOT remove from MinIO (enables recovery).
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	file, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if file.IsDeleted {
		return nil // idempotent
	}

	if err := s.repo.SoftDelete(ctx, id); err != nil {
		return err
	}

	slog.Info("file soft-deleted",
		"file_id", id,
		"filename", file.Filename,
	)
	return nil
}

// Copy copies a file to a target folder by downloading and re-uploading the content.
func (s *Service) Copy(ctx context.Context, fileID, targetFolderID, userID uuid.UUID) (*models.DocumentFile, error) {
	file, err := s.repo.GetByID(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if file.IsDeleted {
		return nil, ErrFileDeleted
	}

	// Download original from MinIO
	reader, err := s.store.Download(ctx, file.StorageKey)
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()

	newFileID := uuid.New()
	now := time.Now()

	// Generate new storage key
	newStorageKey := fmt.Sprintf("documents/copy/%s/%s/%s",
		targetFolderID, newFileID, file.Filename)

	// Upload copy to MinIO
	if err := s.store.Upload(ctx, newStorageKey, reader, file.FileSize, file.MimeType); err != nil {
		return nil, err
	}

	// Create new DB record
	newFile := &models.DocumentFile{
		ID:             newFileID,
		FolderID:       targetFolderID,
		Filename:       file.Filename,
		MimeType:       file.MimeType,
		FileSize:       file.FileSize,
		StorageKey:     newStorageKey,
		CurrentVersion: 1,
		OwnerID:        userID,
		IsFavorite:     false,
		IsDeleted:      false,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.Create(ctx, newFile); err != nil {
		_ = s.store.Delete(ctx, newStorageKey)
		return nil, err
	}

	// Create initial version for copy
	version := &models.DocumentFileVersion{
		ID:            uuid.New(),
		FileID:        newFileID,
		VersionNumber: 1,
		StorageKey:    newStorageKey,
		FileSize:      file.FileSize,
		CreatedBy:     userID,
		CreatedAt:     now,
	}
	if err := s.repo.CreateVersion(ctx, version); err != nil {
		slog.Error("failed to create version for copied file",
			"file_id", newFileID,
			"error", err,
		)
	}

	slog.Info("file copied",
		"source_file_id", fileID,
		"new_file_id", newFileID,
		"target_folder_id", targetFolderID,
		"copied_by", userID,
	)

	return newFile, nil
}

// Move moves a file to a different folder (just updates folder_id, MinIO key stays the same).
func (s *Service) Move(ctx context.Context, fileID, targetFolderID uuid.UUID) error {
	file, err := s.repo.GetByID(ctx, fileID)
	if err != nil {
		return err
	}
	if file.IsDeleted {
		return ErrFileDeleted
	}

	if err := s.repo.Update(ctx, fileID, UpdateInput{FolderID: &targetFolderID}); err != nil {
		return err
	}

	slog.Info("file moved",
		"file_id", fileID,
		"from_folder_id", file.FolderID,
		"to_folder_id", targetFolderID,
	)
	return nil
}

// GetDownloadURL returns a presigned URL for downloading the file (1-hour expiry).
func (s *Service) GetDownloadURL(ctx context.Context, fileID uuid.UUID) (string, error) {
	file, err := s.repo.GetByID(ctx, fileID)
	if err != nil {
		return "", err
	}
	if file.IsDeleted {
		return "", ErrFileDeleted
	}

	return s.store.GetPresignedURL(ctx, file.StorageKey, 1*time.Hour)
}

// VersionInput contains the data needed to create a new file version.
type VersionInput struct {
	Reader   io.Reader
	FileSize int64
	MimeType string
	Label    *string
	UserID   uuid.UUID
}

// CreateVersion uploads new content and creates a new version record.
func (s *Service) CreateVersion(ctx context.Context, fileID uuid.UUID, input VersionInput) (*models.DocumentFileVersion, error) {
	file, err := s.repo.GetByID(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if file.IsDeleted {
		return nil, ErrFileDeleted
	}

	newVersionNumber := file.CurrentVersion + 1
	now := time.Now()

	// Generate version-specific storage key
	storageKey := fmt.Sprintf("documents/versions/%s/%d/%s",
		fileID, newVersionNumber, file.Filename)

	// Upload new content
	if err := s.store.Upload(ctx, storageKey, input.Reader, input.FileSize, input.MimeType); err != nil {
		return nil, err
	}

	version := &models.DocumentFileVersion{
		ID:            uuid.New(),
		FileID:        fileID,
		VersionNumber: newVersionNumber,
		VersionLabel:  input.Label,
		StorageKey:    storageKey,
		FileSize:      input.FileSize,
		CreatedBy:     input.UserID,
		CreatedAt:     now,
	}

	if err := s.repo.CreateVersion(ctx, version); err != nil {
		_ = s.store.Delete(ctx, storageKey)
		return nil, err
	}

	// Update current version on the file
	if err := s.repo.UpdateCurrentVersion(ctx, fileID, newVersionNumber); err != nil {
		slog.Error("failed to update current version",
			"file_id", fileID,
			"version", newVersionNumber,
			"error", err,
		)
	}

	slog.Info("file version created",
		"file_id", fileID,
		"version", newVersionNumber,
		"created_by", input.UserID,
	)

	return version, nil
}

// ListVersions returns all versions for a file, newest first.
func (s *Service) ListVersions(ctx context.Context, fileID uuid.UUID) ([]*models.DocumentFileVersion, error) {
	return s.repo.ListVersions(ctx, fileID)
}

// RevertVersion reverts to a previous version by creating a new version with the old content.
func (s *Service) RevertVersion(ctx context.Context, fileID uuid.UUID, versionNumber int, userID uuid.UUID) (*models.DocumentFileVersion, error) {
	file, err := s.repo.GetByID(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if file.IsDeleted {
		return nil, ErrFileDeleted
	}

	// Get the target version
	targetVersion, err := s.repo.GetVersion(ctx, fileID, versionNumber)
	if err != nil {
		return nil, err
	}

	// Download old version content from MinIO
	reader, err := s.store.Download(ctx, targetVersion.StorageKey)
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()

	newVersionNumber := file.CurrentVersion + 1
	now := time.Now()

	// Upload as new version
	newStorageKey := fmt.Sprintf("documents/versions/%s/%d/%s",
		fileID, newVersionNumber, file.Filename)

	if err := s.store.Upload(ctx, newStorageKey, reader, targetVersion.FileSize, file.MimeType); err != nil {
		return nil, err
	}

	revertLabel := fmt.Sprintf("Reverted from v%d", versionNumber)
	version := &models.DocumentFileVersion{
		ID:            uuid.New(),
		FileID:        fileID,
		VersionNumber: newVersionNumber,
		VersionLabel:  &revertLabel,
		StorageKey:    newStorageKey,
		FileSize:      targetVersion.FileSize,
		CreatedBy:     userID,
		CreatedAt:     now,
	}

	if err := s.repo.CreateVersion(ctx, version); err != nil {
		_ = s.store.Delete(ctx, newStorageKey)
		return nil, err
	}

	if err := s.repo.UpdateCurrentVersion(ctx, fileID, newVersionNumber); err != nil {
		slog.Error("failed to update current version after revert",
			"file_id", fileID,
			"version", newVersionNumber,
			"error", err,
		)
	}

	slog.Info("file version reverted",
		"file_id", fileID,
		"from_version", versionNumber,
		"new_version", newVersionNumber,
		"reverted_by", userID,
	)

	return version, nil
}

// LinkToEntity creates a link between a file and a CRM/PM entity.
func (s *Service) LinkToEntity(ctx context.Context, fileID uuid.UUID, entityType string, entityID, userID uuid.UUID) error {
	if !AllowedEntityTypes[entityType] {
		return ErrInvalidEntityType
	}

	// Verify file exists and not deleted
	file, err := s.repo.GetByID(ctx, fileID)
	if err != nil {
		return err
	}
	if file.IsDeleted {
		return ErrFileDeleted
	}

	link := &models.DocumentEntityLink{
		ID:         uuid.New(),
		FileID:     fileID,
		EntityType: entityType,
		EntityID:   entityID,
		LinkedBy:   userID,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.CreateEntityLink(ctx, link); err != nil {
		return err
	}

	slog.Info("file linked to entity",
		"file_id", fileID,
		"entity_type", entityType,
		"entity_id", entityID,
		"linked_by", userID,
	)
	return nil
}

// UnlinkFromEntity removes a link between a file and an entity.
func (s *Service) UnlinkFromEntity(ctx context.Context, linkID uuid.UUID) error {
	return s.repo.DeleteEntityLink(ctx, linkID)
}

// ListEntityLinks returns all entity links for a file.
func (s *Service) ListEntityLinks(ctx context.Context, fileID uuid.UUID) ([]*models.DocumentEntityLink, error) {
	return s.repo.ListEntityLinks(ctx, fileID)
}

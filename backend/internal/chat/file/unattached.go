package file

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
)

// UnattachedUpload is a stored object that no chat file row refers to yet. Its
// storage key is what a consumer (task attachments) records afterwards.
type UnattachedUpload struct {
	StorageKey string
	Filename   string
	MimeType   string
	FileSize   int64
}

// UnattachedUploadPrefix is where uploads without a channel land. The tenant in
// the key is what lets a consumer refuse a key that belongs to another tenant.
func UnattachedUploadPrefix(tenantID uuid.UUID) string {
	return "tenants/" + tenantID.String() + "/uploads/"
}

// IsTenantUpload reports whether key is an unattached upload of tenantID —
// exactly the prefix followed by one object id, so no path tricks pass.
func IsTenantUpload(tenantID uuid.UUID, key string) bool {
	rest, ok := strings.CutPrefix(key, UnattachedUploadPrefix(tenantID))
	if !ok {
		return false
	}
	_, err := uuid.Parse(rest)
	return err == nil && len(rest) == 36
}

// UploadUnattached stores a file that belongs to no channel, under the
// caller's tenant prefix. It passes the same size, MIME, quota and scan checks
// as a channel upload and is charged against the tenant's quota.
// lean: no row and no thumbnail; an upload that is never attached stays in the
// bucket — add an uploads table with a sweeper when orphaned bytes show up in
// the quota.
func (s *Service) UploadUnattached(ctx context.Context, input UploadInput) (*UnattachedUpload, error) {
	tenantID, err := s.admit(ctx, input)
	if err != nil {
		return nil, err
	}

	storageKey := UnattachedUploadPrefix(tenantID) + uuid.New().String()
	if uploadErr := s.store.Upload(ctx, storageKey, input.Reader, input.FileSize, input.MimeType); uploadErr != nil {
		return nil, fmt.Errorf("upload unattached file: %w", uploadErr)
	}

	if quotaErr := s.repo.IncrementUsedBytes(ctx, tenantID, input.FileSize); quotaErr != nil {
		slog.Error("failed to increment storage quota", "error", quotaErr)
	}

	slog.Info("unattached file uploaded",
		"storage_key", storageKey,
		"size", input.FileSize,
		"uploaded_by", input.UserID,
	)

	return &UnattachedUpload{
		StorageKey: storageKey,
		Filename:   input.Filename,
		MimeType:   input.MimeType,
		FileSize:   input.FileSize,
	}, nil
}

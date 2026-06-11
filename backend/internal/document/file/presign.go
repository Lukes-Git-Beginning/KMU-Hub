package file

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/middleware"
)

const (
	presignUploadExpiry   = 15 * time.Minute
	presignDownloadExpiry = 1 * time.Hour
	maxPresignSizeBytes   = 50 * 1024 * 1024 // 50 MB
)

// allowedPresignScopes defines the scopes permitted for generic presigned uploads.
var allowedPresignScopes = map[string]bool{
	"avatar":    true,
	"chat":      true,
	"rapporte":  true,
	"vermietung": true,
	"vertraege": true,
	"fuhrpark":  true,
	"inventar":  true,
	"kontakte":  true,
	"documents": true,
}

// PresignResult contains the presigned URL and its expiry time.
type PresignResult struct {
	URL       string
	ObjectKey string
	ExpiresAt time.Time
}

// GetPresignedUploadURL generates a presigned PUT URL for browser-direct upload.
//
// Security invariants:
//   - tenant_id is extracted from gRPC context (never trusted from request).
//   - scope is validated against allowedPresignScopes allowlist.
//   - size_bytes is validated against the 50 MB hard limit.
//   - content_type must not be empty.
//   - object_key format: {tenant_id}/{scope}/{uuid}{ext}
func (s *Service) GetPresignedUploadURL(ctx context.Context, scope, fileName, contentType string, sizeBytes int64) (*PresignResult, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	if !allowedPresignScopes[scope] {
		return nil, status.Errorf(codes.InvalidArgument, "invalid scope %q: must be one of avatar, chat, rapporte, vermietung, vertraege, fuhrpark, inventar, kontakte, documents", scope)
	}

	if strings.TrimSpace(contentType) == "" {
		return nil, status.Error(codes.InvalidArgument, "content_type must not be empty")
	}

	if sizeBytes <= 0 {
		return nil, status.Error(codes.InvalidArgument, "size_bytes must be greater than zero")
	}
	if sizeBytes > maxPresignSizeBytes {
		return nil, status.Errorf(codes.InvalidArgument, "size_bytes %d exceeds maximum allowed size of %d bytes (50 MB)", sizeBytes, maxPresignSizeBytes)
	}

	ext := filepath.Ext(fileName)
	objectKey := fmt.Sprintf("%s/%s/%s%s", tenantID.String(), scope, uuid.New().String(), ext)

	expiresAt := time.Now().Add(presignUploadExpiry)
	uploadURL, err := s.store.GetPresignedUploadURL(ctx, objectKey, presignUploadExpiry)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate presigned upload URL: %v", err)
	}

	return &PresignResult{
		URL:       uploadURL,
		ObjectKey: objectKey,
		ExpiresAt: expiresAt,
	}, nil
}

// GetPresignedDownloadURL generates a presigned GET URL for an object.
//
// Security invariant: objectKey must be prefixed with the requesting tenant's ID.
// This prevents cross-tenant object access by constructing keys only the owning
// tenant could have received (from GetPresignedUploadURL).
func (s *Service) GetPresignedDownloadURL(ctx context.Context, objectKey string) (*PresignResult, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	expectedPrefix := tenantID.String() + "/"
	if !strings.HasPrefix(objectKey, expectedPrefix) {
		return nil, status.Error(codes.PermissionDenied, "object_key does not belong to your tenant")
	}

	expiresAt := time.Now().Add(presignDownloadExpiry)
	downloadURL, err := s.store.GetPresignedURL(ctx, objectKey, presignDownloadExpiry)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate presigned download URL: %v", err)
	}

	return &PresignResult{
		URL:       downloadURL,
		ObjectKey: objectKey,
		ExpiresAt: expiresAt,
	}, nil
}

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

	// brandingMaxSizeBytes is a tighter cap than the generic limit: logos/icons
	// render inline in the app chrome (topbar), not stored as documents.
	brandingMaxSizeBytes = 2 * 1024 * 1024 // 2 MB
)

// allowedPresignScopes defines the scopes permitted for generic presigned uploads.
var allowedPresignScopes = map[string]bool{
	"avatar":     true,
	"branding":   true,
	"chat":       true,
	"rapporte":   true,
	"vermietung": true,
	"vertraege":  true,
	"fuhrpark":   true,
	"inventar":   true,
	"kontakte":   true,
	"documents":  true,
}

// brandingAllowedContentTypes restricts logo/icon uploads to raster image
// formats the client can render directly. SVG is deliberately excluded — it
// can carry <script>/onload handlers, and there is no sanitization step
// before a stored object is served back to other users of the tenant.
// lean: hardcoded allowlist; add SVG once a sanitizer (e.g. strip <script>/
// event-handler attributes) sits in front of the upload.
var brandingAllowedContentTypes = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/webp": true,
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
//   - size_bytes is validated against the 50 MB hard limit (2 MB for branding).
//   - content_type must not be empty (branding additionally requires a
//     raster image type — no SVG, see brandingAllowedContentTypes).
//   - object_key format: {tenant_id}/{scope}/{uuid}{ext}
func (s *Service) GetPresignedUploadURL(ctx context.Context, scope, fileName, contentType string, sizeBytes int64) (*PresignResult, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	if !allowedPresignScopes[scope] {
		return nil, status.Errorf(codes.InvalidArgument, "invalid scope %q: must be one of avatar, branding, chat, rapporte, vermietung, vertraege, fuhrpark, inventar, kontakte, documents", scope)
	}

	if strings.TrimSpace(contentType) == "" {
		return nil, status.Error(codes.InvalidArgument, "content_type must not be empty")
	}
	if scope == "branding" && !brandingAllowedContentTypes[contentType] {
		return nil, status.Errorf(codes.InvalidArgument, "content_type %q not allowed for branding uploads: must be image/png, image/jpeg, or image/webp (no SVG — it can carry scripts)", contentType)
	}

	if sizeBytes <= 0 {
		return nil, status.Error(codes.InvalidArgument, "size_bytes must be greater than zero")
	}
	maxSize := int64(maxPresignSizeBytes)
	if scope == "branding" {
		maxSize = brandingMaxSizeBytes
	}
	if sizeBytes > maxSize {
		return nil, status.Errorf(codes.InvalidArgument, "size_bytes %d exceeds maximum allowed size of %d bytes", sizeBytes, maxSize)
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

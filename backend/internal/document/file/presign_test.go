package file

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/middleware"
)

// contextWithTenant injects a tenant ID into a context the same way
// the gRPC TenantInboundUnaryInterceptor does in production.
func contextWithTenant(tenantID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), middleware.TenantIDKey, tenantID.String())
}

func newPresignService() *Service {
	return &Service{store: NewMockFileStore()}
}

// ============================================================================
// GetPresignedUploadURL tests
// ============================================================================

func TestGetPresignedUploadURL_KeyFormat(t *testing.T) {
	svc := newPresignService()
	tenantID := uuid.New()
	ctx := contextWithTenant(tenantID)

	result, err := svc.GetPresignedUploadURL(ctx, "chat", "hello world.pdf", "application/pdf", 1024)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Key must be {tenantID}/{scope}/{uuid}.{ext}
	parts := strings.Split(result.ObjectKey, "/")
	require.Len(t, parts, 3, "expected 3 path segments in object key")
	assert.Equal(t, tenantID.String(), parts[0])
	assert.Equal(t, "chat", parts[1])
	assert.True(t, strings.HasSuffix(parts[2], ".pdf"), "key should end with .pdf, got: %s", parts[2])

	// Validate the UUID segment (strip extension first)
	uuidPart := strings.TrimSuffix(parts[2], ".pdf")
	_, parseErr := uuid.Parse(uuidPart)
	assert.NoError(t, parseErr, "key segment should be a valid UUID")
}

func TestGetPresignedUploadURL_AllowedScopes(t *testing.T) {
	svc := newPresignService()
	tenantID := uuid.New()
	ctx := contextWithTenant(tenantID)

	scopes := []string{"avatar", "chat", "rapporte", "vermietung", "vertraege", "fuhrpark", "inventar", "kontakte", "documents"}
	for _, scope := range scopes {
		result, err := svc.GetPresignedUploadURL(ctx, scope, "file.bin", "application/octet-stream", 512)
		assert.NoError(t, err, "scope %q should be allowed", scope)
		assert.NotNil(t, result)
	}
}

func TestGetPresignedUploadURL_ForbiddenScope(t *testing.T) {
	svc := newPresignService()
	ctx := contextWithTenant(uuid.New())

	_, err := svc.GetPresignedUploadURL(ctx, "../../etc", "evil.sh", "text/plain", 100)
	require.Error(t, err)
	s, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, s.Code())
}

func TestGetPresignedUploadURL_UnknownScope(t *testing.T) {
	svc := newPresignService()
	ctx := contextWithTenant(uuid.New())

	_, err := svc.GetPresignedUploadURL(ctx, "unknown_module", "file.txt", "text/plain", 100)
	require.Error(t, err)
	s, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, s.Code())
}

func TestGetPresignedUploadURL_EmptyContentType(t *testing.T) {
	svc := newPresignService()
	ctx := contextWithTenant(uuid.New())

	_, err := svc.GetPresignedUploadURL(ctx, "chat", "file.txt", "", 1024)
	require.Error(t, err)
	s, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, s.Code())
}

func TestGetPresignedUploadURL_SizeLimit_AtMax(t *testing.T) {
	svc := newPresignService()
	ctx := contextWithTenant(uuid.New())

	// Exactly 50 MB should be allowed
	result, err := svc.GetPresignedUploadURL(ctx, "documents", "big.zip", "application/zip", maxPresignSizeBytes)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPresignedUploadURL_SizeLimit_ExceedsMax(t *testing.T) {
	svc := newPresignService()
	ctx := contextWithTenant(uuid.New())

	// 50 MB + 1 byte should be rejected
	_, err := svc.GetPresignedUploadURL(ctx, "documents", "huge.zip", "application/zip", maxPresignSizeBytes+1)
	require.Error(t, err)
	s, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, s.Code())
}

func TestGetPresignedUploadURL_ZeroSize(t *testing.T) {
	svc := newPresignService()
	ctx := contextWithTenant(uuid.New())

	_, err := svc.GetPresignedUploadURL(ctx, "chat", "empty.txt", "text/plain", 0)
	require.Error(t, err)
	s, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, s.Code())
}

func TestGetPresignedUploadURL_MissingTenant(t *testing.T) {
	svc := newPresignService()
	ctx := context.Background() // no tenant

	_, err := svc.GetPresignedUploadURL(ctx, "chat", "file.txt", "text/plain", 1024)
	require.Error(t, err)
	s, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, s.Code())
}

func TestGetPresignedUploadURL_NoExtension(t *testing.T) {
	svc := newPresignService()
	ctx := contextWithTenant(uuid.New())

	// filename without extension should still work
	result, err := svc.GetPresignedUploadURL(ctx, "chat", "noextension", "text/plain", 512)
	require.NoError(t, err)
	// Key should not have a trailing dot
	assert.False(t, strings.HasSuffix(result.ObjectKey, "."), "key should not end with a dot for extension-less files")
}

// ============================================================================
// GetPresignedDownloadURL tests
// ============================================================================

func TestGetPresignedDownloadURL_OwnPrefix(t *testing.T) {
	svc := newPresignService()
	tenantID := uuid.New()
	ctx := contextWithTenant(tenantID)

	objectKey := tenantID.String() + "/chat/somefile.jpg"
	result, err := svc.GetPresignedDownloadURL(ctx, objectKey)
	require.NoError(t, err)
	assert.NotEmpty(t, result.URL)
	assert.Equal(t, objectKey, result.ObjectKey)
}

func TestGetPresignedDownloadURL_ForeignPrefix_PermissionDenied(t *testing.T) {
	svc := newPresignService()
	tenantID := uuid.New()
	ctx := contextWithTenant(tenantID)

	// Key belonging to a different tenant
	foreignKey := uuid.New().String() + "/chat/evil.jpg"
	_, err := svc.GetPresignedDownloadURL(ctx, foreignKey)
	require.Error(t, err)
	s, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, s.Code())
}

func TestGetPresignedDownloadURL_MissingTenant(t *testing.T) {
	svc := newPresignService()
	ctx := context.Background() // no tenant

	_, err := svc.GetPresignedDownloadURL(ctx, "anykey/file.jpg")
	require.Error(t, err)
	s, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, s.Code())
}

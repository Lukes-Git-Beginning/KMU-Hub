package rapporte

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// SaveSignature — service-layer validation tests
// ============================================================================

func TestSaveSignature_Empty(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	rep := addReport(repo, tenantID, "Bericht", StatusDraft)

	_, err := svc.SaveSignature(context.Background(), tenantID, rep.ID, "", "Max Mustermann")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "signature_data is required")
}

func TestSaveSignature_InvalidPrefix(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	rep := addReport(repo, tenantID, "Bericht", StatusDraft)

	_, err := svc.SaveSignature(context.Background(), tenantID, rep.ID, "data:image/jpeg;base64,abc", "Max Mustermann")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "signature_data must be a valid data URL")
}

func TestSaveSignature_TooLarge(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	rep := addReport(repo, tenantID, "Bericht", StatusDraft)

	// 1 byte over the 1 MB limit
	oversized := "data:image/png;base64," + strings.Repeat("A", 1_048_576)
	_, err := svc.SaveSignature(context.Background(), tenantID, rep.ID, oversized, "Max Mustermann")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum size")
}

func TestSaveSignature_MissingSignedBy(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	rep := addReport(repo, tenantID, "Bericht", StatusDraft)

	_, err := svc.SaveSignature(context.Background(), tenantID, rep.ID, "data:image/png;base64,abc123", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "signed_by is required")
}

func TestSaveSignature_HappyPath_PNG(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	rep := addReport(repo, tenantID, "Bericht", StatusDraft)

	updated, err := svc.SaveSignature(
		context.Background(),
		tenantID,
		rep.ID,
		"data:image/png;base64,abc123",
		"Max Mustermann",
	)

	require.NoError(t, err)
	require.NotNil(t, updated.SignatureData)
	assert.Equal(t, "data:image/png;base64,abc123", *updated.SignatureData)
	require.NotNil(t, updated.SignedBy)
	assert.Equal(t, "Max Mustermann", *updated.SignedBy)
	assert.NotNil(t, updated.SignedAt)
}

func TestSaveSignature_HappyPath_SVG(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	rep := addReport(repo, tenantID, "Bericht", StatusDraft)

	updated, err := svc.SaveSignature(
		context.Background(),
		tenantID,
		rep.ID,
		"data:image/svg+xml;base64,abc123",
		"Erika Musterfrau",
	)

	require.NoError(t, err)
	require.NotNil(t, updated.SignatureData)
	assert.Equal(t, "data:image/svg+xml;base64,abc123", *updated.SignatureData)
	require.NotNil(t, updated.SignedBy)
	assert.Equal(t, "Erika Musterfrau", *updated.SignedBy)
	assert.NotNil(t, updated.SignedAt)
}

func TestSaveSignature_ReportNotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.SaveSignature(
		context.Background(),
		uuid.New(),
		uuid.New(),
		"data:image/png;base64,abc123",
		"Max Mustermann",
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrReportNotFound)
}

func TestSaveSignature_CrossTenantBlocked(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantA := uuid.New()
	tenantB := uuid.New()
	rep := addReport(repo, tenantA, "Geheimbericht", StatusDraft)

	// tenantB tries to sign tenantA's report
	_, err := svc.SaveSignature(
		context.Background(),
		tenantB,
		rep.ID,
		"data:image/png;base64,abc123",
		"Angreifer",
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrReportNotFound, "cross-tenant signature attempt must be rejected")
}

// Exactly at the 1 MB limit — must be accepted.
func TestSaveSignature_AtSizeLimit_Accepted(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	rep := addReport(repo, tenantID, "Bericht", StatusDraft)

	prefix := "data:image/png;base64,"
	// Fill up to exactly 1_048_576 bytes total
	payload := prefix + strings.Repeat("A", 1_048_576-len(prefix))
	require.Len(t, payload, 1_048_576)

	_, err := svc.SaveSignature(context.Background(), tenantID, rep.ID, payload, "Max Mustermann")
	require.NoError(t, err)
}

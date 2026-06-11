package vertraege

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	validContractPNGSignature = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	validContractSVGSignature = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciPjwvc3ZnPg=="
)

// ============================================================================
// SaveContractSignature Tests
// ============================================================================

func TestSaveContractSignature_InvalidPrefix(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	c := addContract(repo, tenantID, "CT-SIG-001", "Signaturtest", ContractStatusActive)

	invalidCases := []struct {
		name string
		data string
	}{
		{"jpeg prefix", "data:image/jpeg;base64,/9j/abc"},
		{"missing prefix", "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJ"},
		{"wrong scheme", "base64:image/png,abc"},
		{"empty data uri", "data:,"},
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.SaveSignature(context.Background(),
				tenantID.String(), c.ID.String(), tc.data, "Max Mustermann")
			assert.ErrorIs(t, err, ErrInvalidInput, "expected ErrInvalidInput for prefix: %q", tc.data[:min(20, len(tc.data))])
		})
	}
}

func TestSaveContractSignature_TooLarge(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	c := addContract(repo, tenantID, "CT-SIG-002", "Signaturtest groß", ContractStatusActive)

	// Build a payload that exceeds 1 MiB: valid prefix + 1 MiB+1 bytes of padding
	prefix := "data:image/png;base64,"
	oversized := prefix + strings.Repeat("A", contractSignatureMaxBytes-len(prefix)+1)

	_, err := svc.SaveSignature(context.Background(),
		tenantID.String(), c.ID.String(), oversized, "Max Mustermann")

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestSaveContractSignature_Empty(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	c := addContract(repo, tenantID, "CT-SIG-003", "Signaturtest leer", ContractStatusActive)

	_, err := svc.SaveSignature(context.Background(),
		tenantID.String(), c.ID.String(), "", "Max Mustermann")

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestSaveContractSignature_HappyPath(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	c := addContract(repo, tenantID, "CT-SIG-004", "Dienstleistungsvertrag", ContractStatusActive)

	signedBy := "Anna Müller"

	result, err := svc.SaveSignature(context.Background(),
		tenantID.String(), c.ID.String(), validContractPNGSignature, signedBy)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, c.ID, result.ID)
	assert.Equal(t, tenantID, result.TenantID)
	require.NotNil(t, result.SignatureData)
	assert.Equal(t, validContractPNGSignature, *result.SignatureData)
	require.NotNil(t, result.SignedBy)
	assert.Equal(t, signedBy, *result.SignedBy)
	require.NotNil(t, result.SignedAt)
	assert.WithinDuration(t, time.Now(), *result.SignedAt, 5*time.Second)
}

func TestSaveContractSignature_HappyPath_SVG(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	c := addContract(repo, tenantID, "CT-SIG-005", "NDA", ContractStatusActive)

	result, err := svc.SaveSignature(context.Background(),
		tenantID.String(), c.ID.String(), validContractSVGSignature, "Klaus Weber")

	require.NoError(t, err)
	require.NotNil(t, result.SignatureData)
	assert.Equal(t, validContractSVGSignature, *result.SignatureData)
}

func TestSaveContractSignature_EmptySignedBy(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	c := addContract(repo, tenantID, "CT-SIG-006", "Vertrag", ContractStatusDraft)

	_, err := svc.SaveSignature(context.Background(),
		tenantID.String(), c.ID.String(), validContractPNGSignature, "   ")

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestSaveContractSignature_ContractNotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()

	_, err := svc.SaveSignature(context.Background(),
		tenantID.String(), uuid.New().String(), validContractPNGSignature, "Max Mustermann")

	assert.ErrorIs(t, err, ErrContractNotFound)
}

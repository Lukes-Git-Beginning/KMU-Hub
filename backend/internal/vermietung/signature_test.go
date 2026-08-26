package vermietung

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
	validPNGSignature = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	validSVGSignature = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciPjwvc3ZnPg=="
)

// ============================================================================
// SaveRentalSignature Tests
// ============================================================================

func TestSaveRentalSignature_InvalidPrefix(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")
	rental := addRental(repo, tenantID, obj.ID, may(2026, 1), may(2026, 5), RentalStatusActive)

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
				tenantID.String(), rental.ID.String(), tc.data, "Max Mustermann")
			assert.ErrorIs(t, err, ErrInvalidInput, "expected ErrInvalidInput for prefix: %q", tc.data[:min(20, len(tc.data))])
		})
	}
}

func TestSaveRentalSignature_TooLarge(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")
	rental := addRental(repo, tenantID, obj.ID, may(2026, 1), may(2026, 5), RentalStatusActive)

	// Build a payload that exceeds 1 MiB: valid prefix + 1 MiB+1 bytes of padding
	prefix := "data:image/png;base64,"
	oversized := prefix + strings.Repeat("A", signatureMaxBytes-len(prefix)+1)

	_, err := svc.SaveSignature(context.Background(),
		tenantID.String(), rental.ID.String(), oversized, "Max Mustermann")

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestSaveRentalSignature_Empty(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")
	rental := addRental(repo, tenantID, obj.ID, may(2026, 1), may(2026, 5), RentalStatusActive)

	_, err := svc.SaveSignature(context.Background(),
		tenantID.String(), rental.ID.String(), "", "Max Mustermann")

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestSaveRentalSignature_HappyPath(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")
	rental := addRental(repo, tenantID, obj.ID, may(2026, 1), may(2026, 5), RentalStatusActive)

	signedBy := "Anna Müller"

	result, err := svc.SaveSignature(context.Background(),
		tenantID.String(), rental.ID.String(), validPNGSignature, signedBy)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, rental.ID, result.ID)
	assert.Equal(t, tenantID, result.TenantID)
	require.NotNil(t, result.SignatureData)
	assert.Equal(t, validPNGSignature, *result.SignatureData)
	require.NotNil(t, result.SignedBy)
	assert.Equal(t, signedBy, *result.SignedBy)
	require.NotNil(t, result.SignedAt)
	assert.WithinDuration(t, time.Now(), *result.SignedAt, 5*time.Second)
}

func TestSaveRentalSignature_HappyPath_SVG(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Anhänger")
	rental := addRental(repo, tenantID, obj.ID, may(2026, 3), may(2026, 7), RentalStatusActive)

	result, err := svc.SaveSignature(context.Background(),
		tenantID.String(), rental.ID.String(), validSVGSignature, "Klaus Weber")

	require.NoError(t, err)
	require.NotNil(t, result.SignatureData)
	assert.Equal(t, validSVGSignature, *result.SignatureData)
}

func TestSaveRentalSignature_EmptySignedBy(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")
	rental := addRental(repo, tenantID, obj.ID, may(2026, 1), may(2026, 5), RentalStatusActive)

	_, err := svc.SaveSignature(context.Background(),
		tenantID.String(), rental.ID.String(), validPNGSignature, "   ")

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestSaveRentalSignature_RentalNotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()

	_, err := svc.SaveSignature(context.Background(),
		tenantID.String(), uuid.New().String(), validPNGSignature, "Max Mustermann")

	assert.ErrorIs(t, err, ErrRentalNotFound)
}

// TestSaveRentalSignature_OverwritesExistingSignatureWithoutGuard documents
// the current (gap) behaviour: neither Service.SaveSignature nor
// PostgresRepository.SaveSignature (postgres_repository.go:504) check whether
// a rental already carries a signature before writing a new one — no
// "AND signature_data IS NULL" guard in the UPDATE, no rental-status check.
// A signature is evidence of a fixed state (who handed over/returned the
// object, and when); as built it stays mutable indefinitely, even after the
// rental is completed. This is the same gap already filed for rapporte's
// HandleSaveReportSignature (see BACKLOG.yml fix-rapporte-signature-overwritable-after-signing)
// — filed here as its own fix-unit, not fixed in this coverage unit.
func TestSaveRentalSignature_OverwritesExistingSignatureWithoutGuard(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")
	rental := addRental(repo, tenantID, obj.ID, may(2026, 1), may(2026, 5), RentalStatusCompleted)

	first, err := svc.SaveSignature(context.Background(),
		tenantID.String(), rental.ID.String(), validPNGSignature, "Max Mustermann")
	require.NoError(t, err)
	require.NotNil(t, first.SignatureData)
	firstSignedAt := *first.SignedAt

	// Second signature on the SAME (already completed and signed) rental
	// currently SUCCEEDs and silently overwrites the first signature instead
	// of being rejected.
	second, err := svc.SaveSignature(context.Background(),
		tenantID.String(), rental.ID.String(), validSVGSignature, "Erika Musterfrau")
	require.NoError(t, err, "gap: a second signature on an already-signed rental is currently accepted")
	require.NotNil(t, second.SignatureData)
	assert.Equal(t, validSVGSignature, *second.SignatureData, "the first signature is silently replaced")
	require.NotNil(t, second.SignedBy)
	assert.Equal(t, "Erika Musterfrau", *second.SignedBy, "the first signer's name is lost")
	assert.True(t, second.SignedAt.After(firstSignedAt) || second.SignedAt.Equal(firstSignedAt))
}

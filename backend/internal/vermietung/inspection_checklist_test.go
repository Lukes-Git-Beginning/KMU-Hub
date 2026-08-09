package vermietung

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Inspection Checklist Tests
// ============================================================================

func TestService_CreateInspection_ChecklistDefaultsToEmptySlice(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")
	rental := addRental(repo, tenantID, obj.ID, may(2026, 1), may(2026, 5), RentalStatusActive)

	ins, err := svc.CreateInspection(context.Background(), CreateInspectionInput{
		TenantID: tenantID,
		RentalID: rental.ID,
		Kind:     InspectionKindHandover,
	})

	require.NoError(t, err)
	require.NotNil(t, ins.Checklist)
	assert.Empty(t, ins.Checklist)
}

func TestService_CreateInspection_ChecklistHappyPath(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")
	rental := addRental(repo, tenantID, obj.ID, may(2026, 1), may(2026, 5), RentalStatusActive)

	ins, err := svc.CreateInspection(context.Background(), CreateInspectionInput{
		TenantID: tenantID,
		RentalID: rental.ID,
		Kind:     InspectionKindHandover,
		Checklist: []ChecklistItem{
			{Label: "Windschutzscheibe", Condition: "intakt"},
			{Label: "Reifen", Condition: "beschaedigt", Remark: "vorne links, leichter Riss"},
		},
	})

	require.NoError(t, err)
	require.Len(t, ins.Checklist, 2)
	assert.Equal(t, "Windschutzscheibe", ins.Checklist[0].Label)
	assert.Equal(t, "vorne links, leichter Riss", ins.Checklist[1].Remark)
}

func TestService_CreateInspection_ChecklistRejectsBlankLabel(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")
	rental := addRental(repo, tenantID, obj.ID, may(2026, 1), may(2026, 5), RentalStatusActive)

	_, err := svc.CreateInspection(context.Background(), CreateInspectionInput{
		TenantID: tenantID,
		RentalID: rental.ID,
		Kind:     InspectionKindHandover,
		Checklist: []ChecklistItem{
			{Label: "   ", Condition: "intakt"},
		},
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_UpdateInspection_ReplaceChecklist_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")
	rental := addRental(repo, tenantID, obj.ID, may(2026, 1), may(2026, 5), RentalStatusActive)

	ins, err := svc.CreateInspection(context.Background(), CreateInspectionInput{
		TenantID: tenantID,
		RentalID: rental.ID,
		Kind:     InspectionKindHandover,
		Checklist: []ChecklistItem{
			{Label: "Reifen", Condition: "intakt"},
		},
	})
	require.NoError(t, err)

	updated, err := svc.UpdateInspection(context.Background(), UpdateInspectionInput{
		TenantID:         tenantID,
		InspectionID:     ins.ID,
		ReplaceChecklist: true,
		Checklist: []ChecklistItem{
			{Label: "Lackierung", Condition: "beschaedigt", Remark: "Kratzer Heck"},
		},
	})

	require.NoError(t, err)
	require.Len(t, updated.Checklist, 1)
	assert.Equal(t, "Lackierung", updated.Checklist[0].Label)
}

func TestService_UpdateInspection_ChecklistIgnoredWithoutReplaceFlag(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")
	rental := addRental(repo, tenantID, obj.ID, may(2026, 1), may(2026, 5), RentalStatusActive)

	ins, err := svc.CreateInspection(context.Background(), CreateInspectionInput{
		TenantID: tenantID,
		RentalID: rental.ID,
		Kind:     InspectionKindHandover,
		Checklist: []ChecklistItem{
			{Label: "Reifen", Condition: "intakt"},
		},
	})
	require.NoError(t, err)

	updated, err := svc.UpdateInspection(context.Background(), UpdateInspectionInput{
		TenantID:     tenantID,
		InspectionID: ins.ID,
		Checklist: []ChecklistItem{
			{Label: "Sollte ignoriert werden"},
		},
	})

	require.NoError(t, err)
	require.Len(t, updated.Checklist, 1)
	assert.Equal(t, "Reifen", updated.Checklist[0].Label)
}

func TestService_UpdateInspection_ChecklistRejectsBlankLabel(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")
	rental := addRental(repo, tenantID, obj.ID, may(2026, 1), may(2026, 5), RentalStatusActive)

	ins, err := svc.CreateInspection(context.Background(), CreateInspectionInput{
		TenantID: tenantID,
		RentalID: rental.ID,
		Kind:     InspectionKindHandover,
	})
	require.NoError(t, err)

	_, err = svc.UpdateInspection(context.Background(), UpdateInspectionInput{
		TenantID:         tenantID,
		InspectionID:     ins.ID,
		ReplaceChecklist: true,
		Checklist:        []ChecklistItem{{Label: ""}},
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

// ============================================================================
// Inspection Signature Tests
// ============================================================================

func TestService_UpdateInspection_SignatureData_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")
	rental := addRental(repo, tenantID, obj.ID, may(2026, 1), may(2026, 5), RentalStatusActive)

	ins, err := svc.CreateInspection(context.Background(), CreateInspectionInput{
		TenantID: tenantID,
		RentalID: rental.ID,
		Kind:     InspectionKindHandover,
	})
	require.NoError(t, err)
	require.Nil(t, ins.SignatureData)

	sig := validPNGSignature
	updated, err := svc.UpdateInspection(context.Background(), UpdateInspectionInput{
		TenantID:      tenantID,
		InspectionID:  ins.ID,
		SignatureData: &sig,
	})

	require.NoError(t, err)
	require.NotNil(t, updated.SignatureData)
	assert.Equal(t, validPNGSignature, *updated.SignatureData)
}

func TestService_UpdateInspection_SignatureData_InvalidPrefix(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	obj := addObject(repo, tenantID, "Fahrzeug")
	rental := addRental(repo, tenantID, obj.ID, may(2026, 1), may(2026, 5), RentalStatusActive)

	ins, err := svc.CreateInspection(context.Background(), CreateInspectionInput{
		TenantID: tenantID,
		RentalID: rental.ID,
		Kind:     InspectionKindHandover,
	})
	require.NoError(t, err)

	bad := "data:image/jpeg;base64,/9j/abc"
	_, err = svc.UpdateInspection(context.Background(), UpdateInspectionInput{
		TenantID:      tenantID,
		InspectionID:  ins.ID,
		SignatureData: &bad,
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

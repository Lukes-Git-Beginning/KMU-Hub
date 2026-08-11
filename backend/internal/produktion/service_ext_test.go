package produktion

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// BOM
// ============================================================================

func TestService_CreateBOM(t *testing.T) {
	tests := []struct {
		name    string
		input   CreateBOMInput
		wantErr error
	}{
		{
			name:    "empty product name is rejected",
			input:   CreateBOMInput{TenantID: uuid.New(), ProductName: "   ", SKU: "SKU-1"},
			wantErr: ErrInvalidInput,
		},
		{
			name:    "empty sku is rejected",
			input:   CreateBOMInput{TenantID: uuid.New(), ProductName: "Widget", SKU: "  "},
			wantErr: ErrInvalidInput,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			svc := NewService(repo)
			_, err := svc.CreateBOM(context.Background(), tt.input)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestService_CreateBOM_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	bom, err := svc.CreateBOM(context.Background(), CreateBOMInput{
		TenantID:    tenantID,
		ProductName: "  Widget  ",
		SKU:         "  SKU-1  ",
		Items: []CreateBomItemInput{
			{MaterialName: "Screw", Quantity: 4, Unit: "pcs"},
		},
	})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, bom.ID)
	assert.Equal(t, "Widget", bom.ProductName)
	assert.Equal(t, "SKU-1", bom.SKU)
	assert.Equal(t, "1.0", bom.Version, "empty version must default to 1.0")
	require.Len(t, bom.Items, 1)
	assert.Equal(t, "Screw", bom.Items[0].MaterialName)
	assert.Equal(t, bom.ID, bom.Items[0].BomID)
}

func TestService_UpdateBOM(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	created, err := svc.CreateBOM(context.Background(), CreateBOMInput{
		TenantID:    tenantID,
		ProductName: "Widget",
		SKU:         "SKU-1",
	})
	require.NoError(t, err)

	newName := "Widget v2"
	updated, err := svc.UpdateBOM(context.Background(), UpdateBOMInput{
		TenantID:    tenantID,
		BOMID:       created.ID,
		ProductName: &newName,
	})
	require.NoError(t, err)
	assert.Equal(t, "Widget v2", updated.ProductName)

	fetched, err := svc.GetBOM(context.Background(), tenantID, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "Widget v2", fetched.ProductName)
}

func TestService_UpdateBOM_Errors(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	created, err := svc.CreateBOM(context.Background(), CreateBOMInput{
		TenantID:    tenantID,
		ProductName: "Widget",
		SKU:         "SKU-1",
	})
	require.NoError(t, err)

	t.Run("unknown bom id", func(t *testing.T) {
		_, err := svc.UpdateBOM(context.Background(), UpdateBOMInput{
			TenantID: tenantID,
			BOMID:    uuid.New(),
		})
		assert.ErrorIs(t, err, ErrBOMNotFound)
	})

	t.Run("empty product name rejected", func(t *testing.T) {
		blank := "   "
		_, err := svc.UpdateBOM(context.Background(), UpdateBOMInput{
			TenantID:    tenantID,
			BOMID:       created.ID,
			ProductName: &blank,
		})
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("empty sku rejected", func(t *testing.T) {
		blank := "   "
		_, err := svc.UpdateBOM(context.Background(), UpdateBOMInput{
			TenantID: tenantID,
			BOMID:    created.ID,
			SKU:      &blank,
		})
		assert.ErrorIs(t, err, ErrInvalidInput)
	})
}

func TestService_DeleteBOM(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	created, err := svc.CreateBOM(context.Background(), CreateBOMInput{
		TenantID:    tenantID,
		ProductName: "Widget",
		SKU:         "SKU-1",
	})
	require.NoError(t, err)

	require.NoError(t, svc.DeleteBOM(context.Background(), tenantID, created.ID))

	_, err = svc.GetBOM(context.Background(), tenantID, created.ID)
	assert.ErrorIs(t, err, ErrBOMNotFound)
}

func TestService_DeleteBOM_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	err := svc.DeleteBOM(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrBOMNotFound)
}

func TestService_ListBOMs_PaginationClamping(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		pageSize   int
		wantOffset int
		wantLimit  int
	}{
		{name: "defaults are used as-is", page: 2, pageSize: 10, wantOffset: 10, wantLimit: 10},
		{name: "page below 1 clamps to 1", page: 0, pageSize: 10, wantOffset: 0, wantLimit: 10},
		{name: "negative page clamps to 1", page: -5, pageSize: 10, wantOffset: 0, wantLimit: 10},
		{name: "page size below 1 clamps to 50", page: 1, pageSize: 0, wantOffset: 0, wantLimit: 50},
		{name: "page size above 100 clamps to 50", page: 1, pageSize: 101, wantOffset: 0, wantLimit: 50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			svc := NewService(repo)

			_, _, err := svc.ListBOMs(context.Background(), ListBOMsInput{
				TenantID: uuid.New(),
				Page:     tt.page,
				PageSize: tt.pageSize,
			})
			require.NoError(t, err)
			assert.Equal(t, tt.wantOffset, repo.lastListBOMsOffset)
			assert.Equal(t, tt.wantLimit, repo.lastListBOMsLimit)
		})
	}
}

// ============================================================================
// Work Steps
// ============================================================================

func TestService_CreateWorkStep_EmptyNameRejected(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateWorkStep(context.Background(), CreateWorkStepInput{
		TenantID: uuid.New(),
		OrderID:  uuid.New(),
		Name:     "   ",
	})
	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_CreateWorkStep_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()
	orderID := uuid.New()

	step, err := svc.CreateWorkStep(context.Background(), CreateWorkStepInput{
		TenantID: tenantID,
		OrderID:  orderID,
		StepNr:   1,
		Name:     "  Cut  ",
	})

	require.NoError(t, err)
	assert.Equal(t, "Cut", step.Name)
	assert.Equal(t, WorkStepStatusPending, step.Status, "new work steps start pending")
}

func TestService_UpdateWorkStep(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	created, err := svc.CreateWorkStep(context.Background(), CreateWorkStepInput{
		TenantID: tenantID,
		OrderID:  uuid.New(),
		Name:     "Cut",
	})
	require.NoError(t, err)

	newStatus := WorkStepStatusCompleted
	updated, err := svc.UpdateWorkStep(context.Background(), UpdateWorkStepInput{
		TenantID: tenantID,
		StepID:   created.ID,
		Status:   &newStatus,
	})
	require.NoError(t, err)
	assert.Equal(t, WorkStepStatusCompleted, updated.Status)

	t.Run("unknown step id", func(t *testing.T) {
		_, err := svc.UpdateWorkStep(context.Background(), UpdateWorkStepInput{
			TenantID: tenantID,
			StepID:   uuid.New(),
		})
		assert.ErrorIs(t, err, ErrWorkStepNotFound)
	})

	t.Run("empty name rejected", func(t *testing.T) {
		blank := "  "
		_, err := svc.UpdateWorkStep(context.Background(), UpdateWorkStepInput{
			TenantID: tenantID,
			StepID:   created.ID,
			Name:     &blank,
		})
		assert.ErrorIs(t, err, ErrInvalidInput)
	})
}

func TestService_DeleteWorkStep(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	created, err := svc.CreateWorkStep(context.Background(), CreateWorkStepInput{
		TenantID: tenantID,
		OrderID:  uuid.New(),
		Name:     "Cut",
	})
	require.NoError(t, err)

	require.NoError(t, svc.DeleteWorkStep(context.Background(), tenantID, created.ID))

	_, err = repo.GetWorkStep(context.Background(), tenantID, created.ID)
	assert.ErrorIs(t, err, ErrWorkStepNotFound)
}

func TestService_DeleteWorkStep_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	err := svc.DeleteWorkStep(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrWorkStepNotFound)
}

// ============================================================================
// Machines
// ============================================================================

func TestService_CreateMachine_EmptyNameRejected(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateMachine(context.Background(), CreateMachineInput{
		TenantID: uuid.New(),
		Name:     "   ",
	})
	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_CreateMachine_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	machine, err := svc.CreateMachine(context.Background(), CreateMachineInput{
		TenantID: tenantID,
		Name:     "  CNC-1  ",
	})

	require.NoError(t, err)
	assert.Equal(t, "CNC-1", machine.Name)
	assert.Equal(t, MachineStatusAvailable, machine.Status, "new machines start available")
}

func TestService_UpdateMachine(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	created, err := svc.CreateMachine(context.Background(), CreateMachineInput{
		TenantID: tenantID,
		Name:     "CNC-1",
	})
	require.NoError(t, err)

	newStatus := MachineStatusMaintenance
	updated, err := svc.UpdateMachine(context.Background(), UpdateMachineInput{
		TenantID:  tenantID,
		MachineID: created.ID,
		Status:    &newStatus,
	})
	require.NoError(t, err)
	assert.Equal(t, MachineStatusMaintenance, updated.Status)

	t.Run("unknown machine id", func(t *testing.T) {
		_, err := svc.UpdateMachine(context.Background(), UpdateMachineInput{
			TenantID:  tenantID,
			MachineID: uuid.New(),
		})
		assert.ErrorIs(t, err, ErrMachineNotFound)
	})

	t.Run("empty name rejected", func(t *testing.T) {
		blank := "  "
		_, err := svc.UpdateMachine(context.Background(), UpdateMachineInput{
			TenantID:  tenantID,
			MachineID: created.ID,
			Name:      &blank,
		})
		assert.ErrorIs(t, err, ErrInvalidInput)
	})
}

func TestService_DeleteMachine(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	created, err := svc.CreateMachine(context.Background(), CreateMachineInput{
		TenantID: tenantID,
		Name:     "CNC-1",
	})
	require.NoError(t, err)

	require.NoError(t, svc.DeleteMachine(context.Background(), tenantID, created.ID))

	_, err = repo.GetMachine(context.Background(), tenantID, created.ID)
	assert.ErrorIs(t, err, ErrMachineNotFound)
}

func TestService_DeleteMachine_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	err := svc.DeleteMachine(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrMachineNotFound)
}

func TestService_ListMachines_PaginationClamping(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		pageSize   int
		wantOffset int
		wantLimit  int
	}{
		{name: "defaults are used as-is", page: 3, pageSize: 20, wantOffset: 40, wantLimit: 20},
		{name: "page below 1 clamps to 1", page: 0, pageSize: 20, wantOffset: 0, wantLimit: 20},
		{name: "page size below 1 clamps to 50", page: 1, pageSize: -1, wantOffset: 0, wantLimit: 50},
		{name: "page size above 100 clamps to 50", page: 1, pageSize: 999, wantOffset: 0, wantLimit: 50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			svc := NewService(repo)

			_, _, err := svc.ListMachines(context.Background(), ListMachinesInput{
				TenantID: uuid.New(),
				Page:     tt.page,
				PageSize: tt.pageSize,
			})
			require.NoError(t, err)
			assert.Equal(t, tt.wantOffset, repo.lastListMachinesOffset)
			assert.Equal(t, tt.wantLimit, repo.lastListMachinesLimit)
		})
	}
}

// ============================================================================
// Quality Checks
// ============================================================================

func TestService_CreateQualityCheck_EmptyInspectorRejected(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateQualityCheck(context.Background(), CreateQualityCheckInput{
		TenantID:  uuid.New(),
		OrderID:   uuid.New(),
		Inspector: "   ",
	})
	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_CreateQualityCheck_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()
	orderID := uuid.New()

	check, err := svc.CreateQualityCheck(context.Background(), CreateQualityCheckInput{
		TenantID:  tenantID,
		OrderID:   orderID,
		Inspector: "  Jane Doe  ",
		Passed:    true,
	})

	require.NoError(t, err)
	assert.Equal(t, "Jane Doe", check.Inspector)
	assert.True(t, check.Passed)

	fetched, err := svc.GetQualityCheck(context.Background(), tenantID, check.ID)
	require.NoError(t, err)
	assert.Equal(t, check.ID, fetched.ID)
}

func TestService_GetQualityCheck_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.GetQualityCheck(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrQualityCheckNotFound)
}

func TestService_ListQualityChecks_PaginationClamping(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		pageSize   int
		wantOffset int
		wantLimit  int
	}{
		{name: "defaults are used as-is", page: 2, pageSize: 5, wantOffset: 5, wantLimit: 5},
		{name: "page below 1 clamps to 1", page: -1, pageSize: 5, wantOffset: 0, wantLimit: 5},
		{name: "page size below 1 clamps to 50", page: 1, pageSize: 0, wantOffset: 0, wantLimit: 50},
		{name: "page size above 100 clamps to 50", page: 1, pageSize: 250, wantOffset: 0, wantLimit: 50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			svc := NewService(repo)

			_, _, err := svc.ListQualityChecks(context.Background(), ListQualityChecksInput{
				TenantID: uuid.New(),
				Page:     tt.page,
				PageSize: tt.pageSize,
			})
			require.NoError(t, err)
			assert.Equal(t, tt.wantOffset, repo.lastListQualityChecksOffset)
			assert.Equal(t, tt.wantLimit, repo.lastListQualityChecksLimit)
		})
	}
}

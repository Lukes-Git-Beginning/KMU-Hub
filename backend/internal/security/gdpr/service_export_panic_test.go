package gdpr

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

type panicExportHandler struct{}

func (panicExportHandler) ModuleName() string { return "panicmodule" }
func (panicExportHandler) ExportUserData(context.Context, uuid.UUID) ([]byte, error) {
	panic("boom in export handler")
}

// A panicking module handler must surface as an error to the caller (the
// detached goroutine of ApproveExport and the sweeper loop), not crash the process.
func TestService_ExecuteExportAsync_HandlerPanicBecomesError(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	exportID := uuid.New()
	repo.exports[exportID] = &models.GDPRExportRequest{
		ID:     exportID,
		UserID: uuid.New(),
		Status: models.ExportStatusApproved,
	}
	svc.RegisterExportHandler(panicExportHandler{})

	err := svc.ExecuteExportAsync(testCtx(), exportID)

	require.ErrorIs(t, err, ErrExportPanicked)
	assert.NotEqual(t, models.ExportStatusReady, repo.exports[exportID].Status)
}

package server

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/biz/dunning"
	"github.com/kmuhub/kmuhub/internal/models"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// stubDunningRepo is a minimal dunning.Repository for exercising the gRPC
// SendDunning handler. List intentionally returns nothing so a regression to the
// old List(Limit:1)-based re-fetch (F12) would yield an empty response and fail
// this test, while the GetByID-based fetch populates it correctly.
type stubDunningRepo struct {
	record *models.DunningRecord
}

func (s *stubDunningRepo) Create(context.Context, *models.DunningRecord) error { return nil }

func (s *stubDunningRepo) GetByID(_ context.Context, tenantID, id uuid.UUID) (*models.DunningRecord, error) {
	if s.record != nil && s.record.ID == id && s.record.TenantID == tenantID {
		return s.record, nil
	}
	return nil, dunning.ErrDunningNotFound
}

func (s *stubDunningRepo) List(context.Context, uuid.UUID, dunning.ListFilter) ([]*models.DunningRecord, int, error) {
	return nil, 0, nil
}

func (s *stubDunningRepo) UpdateStatus(_ context.Context, _, _ uuid.UUID, status string, sentAt *time.Time) error {
	if s.record != nil {
		s.record.Status = status
		s.record.SentAt = sentAt
	}
	return nil
}

func (s *stubDunningRepo) GetByInvoiceID(context.Context, uuid.UUID, uuid.UUID) ([]*models.DunningRecord, error) {
	return nil, nil
}

func (s *stubDunningRepo) GetHighestLevelByInvoiceID(context.Context, uuid.UUID, uuid.UUID) (*models.DunningRecord, error) {
	return nil, nil
}

// TestSendDunning_ResponsePopulated guards F12: SendDunning must return the
// dunning record that was just sent (by ID), not an empty object.
func TestSendDunning_ResponsePopulated(t *testing.T) {
	tenantID := uuid.New()
	dunningID := uuid.New()
	invoiceID := uuid.New()

	repo := &stubDunningRepo{record: &models.DunningRecord{
		ID:        dunningID,
		TenantID:  tenantID,
		InvoiceID: invoiceID,
		Level:     1,
		Status:    models.DunningStatusDraft,
		Fee:       decimal.Zero,
		Interest:  decimal.Zero,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
	}}

	svc := dunning.NewService(repo, nil, nil)
	srv := &BizGRPCServer{dunningService: svc}

	resp, err := srv.SendDunning(context.Background(), &bizv1.SendDunningRequest{
		TenantId: tenantID.String(),
		Id:       dunningID.String(),
	})
	require.NoError(t, err)
	require.NotNil(t, resp.GetDunning(), "SendDunning must populate the response with the sent record")
	assert.Equal(t, dunningID.String(), resp.GetDunning().GetId())
	assert.NotNil(t, resp.GetDunning().GetSentAt(), "sent_at must be set after Send")
}

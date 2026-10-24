package quote

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_Update_SetsCurrency(t *testing.T) {
	repo := NewMockRepository()
	svc, _, _ := newTestService(repo)
	tenantID := uuid.New()
	existing := seedDraftQuote(repo, tenantID)

	chf, empty := "CHF", ""
	got, err := svc.Update(context.Background(), tenantID, existing.ID, UpdateInput{Currency: &chf})
	require.NoError(t, err)
	assert.Equal(t, "CHF", got.Currency)

	got, err = svc.Update(context.Background(), tenantID, existing.ID, UpdateInput{Currency: &empty})
	require.NoError(t, err)
	assert.Equal(t, "CHF", got.Currency, "empty currency leaves the stored one")
}

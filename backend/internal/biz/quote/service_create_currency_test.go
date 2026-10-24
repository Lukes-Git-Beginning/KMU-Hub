package quote

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

func TestService_Create_CurrencyExplicitBeatsTenantDefault(t *testing.T) {
	svc, _, cs := newTestService(NewMockRepository())
	cs.settings = &models.CompanySettings{DefaultCurrency: "EUR"}
	input := sampleCreateInput(uuid.New())

	q, err := svc.Create(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, "EUR", q.Currency, "no explicit currency takes the tenant default")

	input.Currency = "USD"
	q, err = svc.Create(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, "USD", q.Currency)
}

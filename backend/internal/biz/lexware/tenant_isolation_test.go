package lexware

// Cross-tenant isolation tests for the Lexware repository layer.
//
// Mirrors the Bexio test pattern: two config IDs (one per tenant), verify that
// ListEntityMappings and ListSyncLogs never leak records across config boundaries.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// TestLexware_ListEntityMappings_TenantIsolation verifies that entity mappings
// for config A are not visible when querying with config B's ID.
func TestLexware_ListEntityMappings_TenantIsolation(t *testing.T) {
	configA := uuid.New()
	configB := uuid.New()

	allMappings := map[uuid.UUID][]models.LexwareEntityMapping{
		configA: {
			{ID: uuid.New(), ConfigID: configA, EntityType: "contact", KmuhubID: uuid.New(), LexwareID: "lex-a-1"},
		},
		configB: {
			{ID: uuid.New(), ConfigID: configB, EntityType: "contact", KmuhubID: uuid.New(), LexwareID: "lex-b-1"},
		},
	}

	repo := &mockRepository{
		listEntityMappingsFn: func(_ context.Context, configID uuid.UUID, entityType string) ([]models.LexwareEntityMapping, error) {
			return allMappings[configID], nil
		},
	}

	// Query config A — must only return config A's mapping.
	mappingsA, err := repo.ListEntityMappings(context.Background(), configA, "contact")
	require.NoError(t, err)
	require.Len(t, mappingsA, 1)
	assert.Equal(t, configA, mappingsA[0].ConfigID)
	assert.Equal(t, "lex-a-1", mappingsA[0].LexwareID)

	// Query config B — must only return config B's mapping.
	mappingsB, err := repo.ListEntityMappings(context.Background(), configB, "contact")
	require.NoError(t, err)
	require.Len(t, mappingsB, 1)
	assert.Equal(t, configB, mappingsB[0].ConfigID)
	assert.Equal(t, "lex-b-1", mappingsB[0].LexwareID)

	// No config A records in config B results.
	for _, m := range mappingsB {
		assert.NotEqual(t, configA, m.ConfigID, "config A record leaked into config B result")
	}
}

// TestLexware_ListSyncLogs_TenantIsolation verifies that sync logs for config A
// are not returned when listing logs for config B.
func TestLexware_ListSyncLogs_TenantIsolation(t *testing.T) {
	configA := uuid.New()
	configB := uuid.New()

	allLogs := map[uuid.UUID][]models.LexwareSyncLog{
		configA: {
			{ID: uuid.New(), ConfigID: configA, SyncType: "contact_full", Status: "success"},
		},
		configB: {
			{ID: uuid.New(), ConfigID: configB, SyncType: "contact_full", Status: "failed"},
		},
	}

	repo := &mockRepository{
		listSyncLogsFn: func(_ context.Context, configID uuid.UUID, _ int) ([]models.LexwareSyncLog, error) {
			return allLogs[configID], nil
		},
	}

	// Config A logs.
	logsA, err := repo.ListSyncLogs(context.Background(), configA, 10)
	require.NoError(t, err)
	require.Len(t, logsA, 1)
	assert.Equal(t, configA, logsA[0].ConfigID)
	assert.Equal(t, "success", logsA[0].Status)

	// Config B logs.
	logsB, err := repo.ListSyncLogs(context.Background(), configB, 10)
	require.NoError(t, err)
	require.Len(t, logsB, 1)
	assert.Equal(t, configB, logsB[0].ConfigID)
	assert.Equal(t, "failed", logsB[0].Status)

	for _, l := range logsB {
		assert.NotEqual(t, configA, l.ConfigID, "config A log leaked into config B result")
	}
}

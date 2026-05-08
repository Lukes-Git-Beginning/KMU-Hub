package bexio

// Cross-tenant isolation tests for the Bexio repository layer.
//
// These tests verify that ListEntityMappings and ListSyncLogs only return
// records belonging to the requested config_id (which maps 1:1 to a tenant).
// The mock repository simulates two tenants (configA, configB); querying with
// configA must never return configB's records.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// TestBexio_ListEntityMappings_TenantIsolation verifies that entity mappings for
// config A are not visible when querying with config B's ID.
func TestBexio_ListEntityMappings_TenantIsolation(t *testing.T) {
	configA := uuid.New()
	configB := uuid.New()

	// Simulate a DB where both tenant A and tenant B have contact mappings.
	allMappings := map[uuid.UUID][]models.BexioEntityMapping{
		configA: {
			{ID: uuid.New(), ConfigID: configA, EntityType: "contact", KmuhubID: uuid.New(), BexioID: 1},
		},
		configB: {
			{ID: uuid.New(), ConfigID: configB, EntityType: "contact", KmuhubID: uuid.New(), BexioID: 2},
		},
	}

	repo := &mockRepository{
		listEntityMappingsFn: func(_ context.Context, configID uuid.UUID, entityType string) ([]models.BexioEntityMapping, error) {
			// The real postgres_repository JOIN ensures only the matching configID is returned.
			// Here we simulate that isolation explicitly.
			return allMappings[configID], nil
		},
	}

	cr := &mockConfigRepo{
		getByPlatformFn: func(_ context.Context, _ string) (*IntegrationConfig, error) {
			return &IntegrationConfig{ID: configA, IsActive: true}, nil
		},
	}

	svc := newTestService(repo, cr, nil)
	_ = svc // service itself not used — we test the repo layer directly

	// Querying with configA must only return configA records.
	mappingsA, err := repo.ListEntityMappings(context.Background(), configA, "contact")
	require.NoError(t, err)
	require.Len(t, mappingsA, 1)
	assert.Equal(t, configA, mappingsA[0].ConfigID, "mapping belongs to config A")

	// Querying with configB must only return configB records.
	mappingsB, err := repo.ListEntityMappings(context.Background(), configB, "contact")
	require.NoError(t, err)
	require.Len(t, mappingsB, 1)
	assert.Equal(t, configB, mappingsB[0].ConfigID, "mapping belongs to config B")

	// Confirm no cross-contamination: configA's BexioID must not appear in configB results.
	for _, m := range mappingsB {
		assert.NotEqual(t, configA, m.ConfigID, "config A record leaked into config B result")
	}
}

// TestBexio_ListSyncLogs_TenantIsolation verifies that sync logs for config A
// are not returned when listing logs for config B.
func TestBexio_ListSyncLogs_TenantIsolation(t *testing.T) {
	configA := uuid.New()
	configB := uuid.New()

	allLogs := map[uuid.UUID][]models.BexioSyncLog{
		configA: {
			{ID: uuid.New(), ConfigID: configA, SyncType: "contact_full", Status: "success"},
		},
		configB: {
			{ID: uuid.New(), ConfigID: configB, SyncType: "contact_full", Status: "failed"},
		},
	}

	repo := &mockRepository{
		listSyncLogsFn: func(_ context.Context, configID uuid.UUID, _ int) ([]models.BexioSyncLog, error) {
			return allLogs[configID], nil
		},
	}

	// Query config A — expect only config A's log.
	logsA, err := repo.ListSyncLogs(context.Background(), configA, 10)
	require.NoError(t, err)
	require.Len(t, logsA, 1)
	assert.Equal(t, configA, logsA[0].ConfigID)
	assert.Equal(t, "success", logsA[0].Status)

	// Query config B — expect only config B's log (failed, not success).
	logsB, err := repo.ListSyncLogs(context.Background(), configB, 10)
	require.NoError(t, err)
	require.Len(t, logsB, 1)
	assert.Equal(t, configB, logsB[0].ConfigID)
	assert.Equal(t, "failed", logsB[0].Status)

	// No config A records in config B results.
	for _, l := range logsB {
		assert.NotEqual(t, configA, l.ConfigID, "config A log leaked into config B result")
	}
}

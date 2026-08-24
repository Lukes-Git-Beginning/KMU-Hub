package datev

// Covers PostgresUploadRepository against the real schema.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func setupDatevUploadRepo(t *testing.T) (*PostgresUploadRepository, *pgxpool.Pool, context.Context, uuid.UUID, uuid.UUID) {
	t.Helper()
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Datev Upload Repo Test Tenant")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("datev-repo-test-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	configID := testutil.SeedRow(t, pool, "integration_configs", map[string]any{
		"tenant_id":             tenantID,
		"platform":              "datev_api",
		"credentials_vault_key": "datev/" + uuid.New().String(),
		"created_by":            userID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "integration_configs", configID) })

	repo := NewPostgresUploadRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	return repo, pool, ctx, tenantID, configID
}

// ============================================================================
// UpsertUploadConfig / CreateUploadLog — fixed in fix-datev-upload-repo-
// missing-tenant-id (Lauf 9). Both now resolve tenant_id from ctx via
// tenantForWrite instead of leaving the NOT NULL column unset.
// ============================================================================

func TestUpsertUploadConfig_InsertsThenUpdatesOnConflict(t *testing.T) {
	repo, pool, ctx, _, configID := setupDatevUploadRepo(t)

	cfg := &models.DatevUploadConfig{ConfigID: configID, ClientNumber: "12345", AutoUploadEnabled: true}
	if err := repo.UpsertUploadConfig(ctx, cfg); err != nil {
		t.Fatalf("UpsertUploadConfig (insert): %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "datev_upload_configs", cfg.ID) })

	got, err := repo.GetUploadConfig(ctx, configID)
	if err != nil {
		t.Fatalf("GetUploadConfig after insert: %v", err)
	}
	if got.ClientNumber != "12345" || !got.AutoUploadEnabled {
		t.Errorf("got = %+v, want client_number=12345 auto_upload_enabled=true", got)
	}

	// Same ConfigID, same repo-assigned ID — this is the ON CONFLICT(config_id) path.
	cfg.ClientNumber = "67890"
	cfg.AutoUploadEnabled = false
	if err := repo.UpsertUploadConfig(ctx, cfg); err != nil {
		t.Fatalf("UpsertUploadConfig (update): %v", err)
	}

	got, err = repo.GetUploadConfig(ctx, configID)
	if err != nil {
		t.Fatalf("GetUploadConfig after update: %v", err)
	}
	if got.ClientNumber != "67890" || got.AutoUploadEnabled {
		t.Errorf("got = %+v, want client_number=67890 auto_upload_enabled=false", got)
	}
}

func TestCreateUploadLog_InsertsRow(t *testing.T) {
	repo, pool, ctx, _, configID := setupDatevUploadRepo(t)

	log := &models.DatevUploadLog{
		ConfigID: configID, UploadType: "buchungsstapel", Status: "uploading", StartedAt: time.Now().UTC(),
	}
	if err := repo.CreateUploadLog(ctx, log); err != nil {
		t.Fatalf("CreateUploadLog: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "datev_upload_log", log.ID) })

	logs, err := repo.ListUploadLogs(ctx, configID, 10)
	if err != nil {
		t.Fatalf("ListUploadLogs: %v", err)
	}
	if len(logs) != 1 || logs[0].ID != log.ID {
		t.Fatalf("logs = %+v, want exactly the just-created log %s", logs, log.ID)
	}
	testutil.AssertRowCount(t, pool, ctx, "datev_upload_log", log.ID, 1)
}

// ============================================================================
// RLS — a write must land under the calling tenant and stay invisible to any
// other tenant, even though the tenant_id column is never supplied by the
// caller directly (tenantForWrite resolves it from ctx).
// ============================================================================

func TestUpsertUploadConfigAndCreateUploadLog_WriteLandsInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Datev Upload RLS Own Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Datev Upload RLS Other Tenant")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("datev-rls-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	configID := testutil.SeedRow(t, pool, "integration_configs", map[string]any{
		"tenant_id":             tenantOwn,
		"platform":              "datev_api",
		"credentials_vault_key": "datev/" + uuid.New().String(),
		"created_by":            userID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "integration_configs", configID) })

	repo := NewPostgresUploadRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	cfg := &models.DatevUploadConfig{ConfigID: configID, ClientNumber: "55555"}
	if err := repo.UpsertUploadConfig(ctxOwn, cfg); err != nil {
		t.Fatalf("UpsertUploadConfig (own ctx): %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "datev_upload_configs", cfg.ID) })
	testutil.AssertRowCount(t, pool, ctxOwn, "datev_upload_configs", cfg.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "datev_upload_configs", cfg.ID, 0)

	log := &models.DatevUploadLog{
		ConfigID: configID, UploadType: "buchungsstapel", Status: "uploading", StartedAt: time.Now().UTC(),
	}
	if err := repo.CreateUploadLog(ctxOwn, log); err != nil {
		t.Fatalf("CreateUploadLog (own ctx): %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "datev_upload_log", log.ID) })
	testutil.AssertRowCount(t, pool, ctxOwn, "datev_upload_log", log.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "datev_upload_log", log.ID, 0)
}

// TestListUploadLogs_InvisibleToForeignTenant belongs with the RLS write test
// above but exercises the read side through the actual accessor callers use:
// ListUploadLogs filters WHERE config_id = $1 alone — RLS is the only thing
// stopping a foreign tenant who somehow learned another tenant's config_id
// from reading their DATEV transfer history.
func TestListUploadLogs_InvisibleToForeignTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Datev Upload Logs RLS Own Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Datev Upload Logs RLS Other Tenant")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("datev-rls-read-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	configID := testutil.SeedRow(t, pool, "integration_configs", map[string]any{
		"tenant_id":             tenantOwn,
		"platform":              "datev_api",
		"credentials_vault_key": "datev/" + uuid.New().String(),
		"created_by":            userID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "integration_configs", configID) })

	logID := seedUploadLog(t, pool, tenantOwn, configID, "completed", time.Now().UTC())

	repo := NewPostgresUploadRepository(pool)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	logs, err := repo.ListUploadLogs(ctxOther, configID, 10)
	if err != nil {
		t.Fatalf("ListUploadLogs (foreign tenant): %v", err)
	}
	if len(logs) != 0 {
		t.Fatalf("logs = %+v, want none — RLS must hide tenantOwn's upload history from tenantOther "+
			"even though the caller supplied tenantOwn's real config_id (log %s)", logs, logID)
	}

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	logs, err = repo.ListUploadLogs(ctxOwn, configID, 10)
	if err != nil {
		t.Fatalf("ListUploadLogs (own tenant): %v", err)
	}
	if len(logs) != 1 || logs[0].ID != logID {
		t.Fatalf("logs = %+v, want exactly the seeded log %s under its own tenant", logs, logID)
	}
}

func TestUpsertUploadConfig_FailsWithoutTenantContext(t *testing.T) {
	repo, _, _, _, configID := setupDatevUploadRepo(t)

	err := repo.UpsertUploadConfig(context.Background(), &models.DatevUploadConfig{ConfigID: configID, ClientNumber: "12345"})
	if !errors.Is(err, ErrTenantMissing) {
		t.Fatalf("err = %v, want ErrTenantMissing", err)
	}
}

func TestCreateUploadLog_FailsWithoutTenantContext(t *testing.T) {
	repo, _, _, _, configID := setupDatevUploadRepo(t)

	err := repo.CreateUploadLog(context.Background(), &models.DatevUploadLog{
		ConfigID: configID, UploadType: "buchungsstapel", Status: "uploading", StartedAt: time.Now().UTC(),
	})
	if !errors.Is(err, ErrTenantMissing) {
		t.Fatalf("err = %v, want ErrTenantMissing", err)
	}
}

// ============================================================================
// GetUploadConfig
// ============================================================================

func TestGetUploadConfig_ReturnsSeededConfig(t *testing.T) {
	repo, pool, ctx, tenantID, configID := setupDatevUploadRepo(t)

	uploadCfgID := testutil.SeedRow(t, pool, "datev_upload_configs", map[string]any{
		"tenant_id":           tenantID,
		"config_id":           configID,
		"client_number":       "55555",
		"auto_upload_enabled": true,
		"upload_after_export": false,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "datev_upload_configs", uploadCfgID) })

	got, err := repo.GetUploadConfig(ctx, configID)
	if err != nil {
		t.Fatalf("GetUploadConfig: %v", err)
	}
	if got.ClientNumber != "55555" {
		t.Errorf("ClientNumber = %q, want 55555", got.ClientNumber)
	}
	if !got.AutoUploadEnabled {
		t.Error("AutoUploadEnabled = false, want true")
	}
	if got.UploadAfterExport {
		t.Error("UploadAfterExport = true, want false")
	}
}

func TestGetUploadConfig_NotFound(t *testing.T) {
	repo, _, ctx, _, _ := setupDatevUploadRepo(t)

	_, err := repo.GetUploadConfig(ctx, uuid.New())
	if err == nil || err.Error() != "datev: upload config not found" {
		t.Fatalf("err = %v, want the not-found sentinel message", err)
	}
}

// ============================================================================
// UpdateUploadLog / ListUploadLogs — seeded directly since Create is broken.
// ============================================================================

func seedUploadLog(t *testing.T, pool *pgxpool.Pool, tenantID, configID uuid.UUID, status string, startedAt time.Time) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, pool, "datev_upload_log", map[string]any{
		"tenant_id":      tenantID,
		"config_id":      configID,
		"upload_type":    "buchungsstapel",
		"status":         status,
		"file_size":      1024,
		"document_count": 3,
		"started_at":     startedAt,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "datev_upload_log", id) })
	return id
}

func TestUpdateUploadLog_UpdatesStatusAndFields(t *testing.T) {
	repo, pool, ctx, tenantID, configID := setupDatevUploadRepo(t)
	logID := seedUploadLog(t, pool, tenantID, configID, "uploading", time.Now().UTC())

	completed := time.Now().UTC()
	errMsg := "502 from DATEV"
	err := repo.UpdateUploadLog(ctx, &models.DatevUploadLog{
		ID: logID, Status: "failed", FileSize: 2048, DocumentCount: 5,
		ErrorMessage: &errMsg, CompletedAt: &completed, Metadata: map[string]any{"attempt": "final"},
	})
	if err != nil {
		t.Fatalf("UpdateUploadLog: %v", err)
	}

	logs, err := repo.ListUploadLogs(ctx, configID, 10)
	if err != nil {
		t.Fatalf("ListUploadLogs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("logs = %d, want 1", len(logs))
	}
	got := logs[0]
	if got.Status != "failed" || got.FileSize != 2048 || got.DocumentCount != 5 {
		t.Errorf("log = %+v, want status=failed file_size=2048 document_count=5", got)
	}
	if got.ErrorMessage == nil || *got.ErrorMessage != errMsg {
		t.Errorf("ErrorMessage = %v, want %q", got.ErrorMessage, errMsg)
	}
	if got.CompletedAt == nil {
		t.Error("CompletedAt was not persisted")
	}
	if got.Metadata["attempt"] != "final" {
		t.Errorf("Metadata = %v, want attempt=final", got.Metadata)
	}
}

func TestUpdateUploadLog_RejectsInvalidStatus(t *testing.T) {
	repo, pool, ctx, tenantID, configID := setupDatevUploadRepo(t)
	logID := seedUploadLog(t, pool, tenantID, configID, "uploading", time.Now().UTC())

	err := repo.UpdateUploadLog(ctx, &models.DatevUploadLog{ID: logID, Status: "not-a-real-status"})
	if err == nil {
		t.Fatal("expected the status CHECK constraint to reject an unknown status")
	}
}

func TestListUploadLogs_OrdersDescByStartedAtAndRespectsLimit(t *testing.T) {
	repo, pool, ctx, tenantID, configID := setupDatevUploadRepo(t)

	base := time.Now().UTC().Truncate(time.Second)
	seedUploadLog(t, pool, tenantID, configID, "completed", base.Add(-2*time.Hour))
	seedUploadLog(t, pool, tenantID, configID, "completed", base.Add(-1*time.Hour))
	newest := seedUploadLog(t, pool, tenantID, configID, "completed", base)

	logs, err := repo.ListUploadLogs(ctx, configID, 2)
	if err != nil {
		t.Fatalf("ListUploadLogs: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("logs = %d, want 2 (limit respected)", len(logs))
	}
	if logs[0].ID != newest {
		t.Errorf("logs[0].ID = %s, want the newest log %s (DESC by started_at)", logs[0].ID, newest)
	}
	if !logs[0].StartedAt.After(logs[1].StartedAt) {
		t.Errorf("logs not ordered DESC by started_at: %s then %s", logs[0].StartedAt, logs[1].StartedAt)
	}
}

func TestListUploadLogs_DefaultLimitWhenNonPositive(t *testing.T) {
	repo, pool, ctx, tenantID, configID := setupDatevUploadRepo(t)
	seedUploadLog(t, pool, tenantID, configID, "completed", time.Now().UTC())

	logs, err := repo.ListUploadLogs(ctx, configID, 0)
	if err != nil {
		t.Fatalf("ListUploadLogs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("logs = %d, want 1 (limit<=0 falls back to a default, not zero rows)", len(logs))
	}
}

func TestListUploadLogs_EmptyForUnknownConfig(t *testing.T) {
	repo, _, ctx, _, _ := setupDatevUploadRepo(t)

	logs, err := repo.ListUploadLogs(ctx, uuid.New(), 10)
	if err != nil {
		t.Fatalf("ListUploadLogs: %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("logs = %d, want 0", len(logs))
	}
}

package gobdarchive

// The GoBD archive is append-only by law (§ 147 AO, "Unveraenderbarkeit").
// Until migration 000315 that was a property of the Go code alone: the
// Repository interface exposes no update and no delete, but kmuhub_app --
// the role every service connects as -- still held UPDATE and DELETE on both
// archive tables through the schema-wide grant in 000121. These tests prove
// the database now refuses the mutation itself, so a future query or an
// injection cannot rewrite an archived document.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/kmuhub/kmuhub/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// insufficientPrivilege is SQLSTATE 42501.
const insufficientPrivilege = "42501"

func requirePrivilegeError(t *testing.T, err error, op string) {
	t.Helper()
	require.Error(t, err, "%s on the archive must be refused by the database", op)
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr, "%s must fail with a PgError, got %v", op, err)
	assert.Equal(t, insufficientPrivilege, pgErr.Code,
		"%s must fail with insufficient_privilege, got SQLSTATE %s: %s", op, pgErr.Code, pgErr.Message)
}

// TestGobdArchive_WormPrivileges_AppRoleCannotMutate archives a document as
// the application role and then tries to change and to erase it. INSERT and
// SELECT must keep working -- revoking too much would break archiving itself.
//
// Everything runs inside one transaction that is always rolled back: with
// DELETE revoked there is no way to clean up a committed fixture row.
func TestGobdArchive_WormPrivileges_AppRoleCannotMutate(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	ctx := testutil.WithSystemCtx(context.Background())

	var role string
	require.NoError(t, pool.QueryRow(ctx, "SELECT current_user").Scan(&role))
	require.Equal(t, "kmuhub_app", role,
		"DATABASE_URL must point at kmuhub_app; as a table owner or superuser this test proves nothing")

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A GoBD WORM")

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	docID := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO gobd_documents (
			id, tenant_id, doc_type, storage_key, sha256, original_filename,
			mime_type, file_size_bytes, archived_by, retention_until
		) VALUES ($1, $2, 'invoice', $3, $4, 'worm.pdf', 'application/pdf', 2048, $5, $6)`,
		docID, testutil.TenantA, "gobd/test/worm/"+docID.String()+".pdf",
		fakeSHA256("worm-"+docID.String()), uuid.New(),
		time.Date(2036, 12, 31, 0, 0, 0, 0, time.UTC),
	)
	require.NoError(t, err, "archiving must stay possible -- INSERT is not revoked")

	var count int
	require.NoError(t, tx.QueryRow(ctx,
		"SELECT count(*) FROM gobd_documents WHERE id = $1", docID).Scan(&count))
	require.Equal(t, 1, count, "reading the archive must stay possible -- SELECT is not revoked")

	// Each failing statement aborts its own savepoint, not the outer tx.
	sp, err := tx.Begin(ctx)
	require.NoError(t, err)
	_, err = sp.Exec(ctx,
		"UPDATE gobd_documents SET original_filename = 'tampered.pdf' WHERE id = $1", docID)
	requirePrivilegeError(t, err, "UPDATE gobd_documents")
	require.NoError(t, sp.Rollback(ctx))

	sp, err = tx.Begin(ctx)
	require.NoError(t, err)
	_, err = sp.Exec(ctx, "DELETE FROM gobd_documents WHERE id = $1", docID)
	requirePrivilegeError(t, err, "DELETE FROM gobd_documents")
	require.NoError(t, sp.Rollback(ctx))

	eventID := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO gobd_document_events (id, tenant_id, document_id, event_type, created_by, note)
		VALUES ($1, $2, $3, 'archived', $4, 'worm probe')`,
		eventID, testutil.TenantA, docID, uuid.New())
	require.NoError(t, err, "appending to the trail must stay possible")

	sp, err = tx.Begin(ctx)
	require.NoError(t, err)
	_, err = sp.Exec(ctx,
		"UPDATE gobd_document_events SET note = 'rewritten' WHERE id = $1", eventID)
	requirePrivilegeError(t, err, "UPDATE gobd_document_events")
	require.NoError(t, sp.Rollback(ctx))

	sp, err = tx.Begin(ctx)
	require.NoError(t, err)
	_, err = sp.Exec(ctx, "DELETE FROM gobd_document_events WHERE id = $1", eventID)
	requirePrivilegeError(t, err, "DELETE FROM gobd_document_events")
	require.NoError(t, sp.Rollback(ctx))
}

// TestGobdArchive_WormPrivileges_AllArchiveTables is the guard for tables that
// do not exist yet: ALTER DEFAULT PRIVILEGES in 000121 hands kmuhub_app UPDATE
// and DELETE on every future table automatically, so a later migration adding
// a gobd_* table would silently reopen the archive. This scans the catalog
// instead of a hardcoded list and turns that omission into a red test.
func TestGobdArchive_WormPrivileges_AllArchiveTables(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	ctx := testutil.WithSystemCtx(context.Background())
	rows, err := pool.Query(ctx, `
		SELECT tablename,
		       has_table_privilege('kmuhub_app', quote_ident(tablename), 'SELECT'),
		       has_table_privilege('kmuhub_app', quote_ident(tablename), 'INSERT'),
		       has_table_privilege('kmuhub_app', quote_ident(tablename), 'UPDATE'),
		       has_table_privilege('kmuhub_app', quote_ident(tablename), 'DELETE')
		FROM pg_tables
		WHERE schemaname = 'public' AND tablename LIKE 'gobd\_%'
		ORDER BY tablename`)
	require.NoError(t, err)
	defer rows.Close()

	seen := 0
	for rows.Next() {
		var table string
		var canSelect, canInsert, canUpdate, canDelete bool
		require.NoError(t, rows.Scan(&table, &canSelect, &canInsert, &canUpdate, &canDelete))
		seen++

		assert.True(t, canSelect, "%s: kmuhub_app must keep SELECT", table)
		assert.True(t, canInsert, "%s: kmuhub_app must keep INSERT -- archiving depends on it", table)
		assert.False(t, canUpdate,
			"%s: kmuhub_app holds UPDATE. A new archive table needs the REVOKE from migration 000315.", table)
		assert.False(t, canDelete,
			"%s: kmuhub_app holds DELETE. A new archive table needs the REVOKE from migration 000315.", table)
	}
	require.NoError(t, rows.Err())
	assert.GreaterOrEqual(t, seen, 2, "expected at least gobd_documents and gobd_document_events")
}

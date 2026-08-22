package server

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
)

// seedEmailExportContact imports one contact through the already-proven import path
// and returns its DB id, so the export tests exercise real persisted data rather than
// fixtures the export path never has to look up.
func seedEmailExportContact(t *testing.T, pool *pgxpool.Pool, srv *EmailGRPCServer, ctx context.Context, tenantID uuid.UUID, email string) uuid.UUID {
	t.Helper()
	csv := "E-Mail,Vorname,Nachname,Firma\r\n" + email + ",Max,Mustermann,Acme GmbH\r\n"
	resp, err := srv.ImportContactsCSV(ctx, &emailv1.ImportContactsCSVRequest{
		FileContent: []byte(csv),
		FieldMapping: map[string]string{
			"E-Mail":   "email",
			"Vorname":  "first_name",
			"Nachname": "last_name",
			"Firma":    "company",
		},
		Visibility: "shared",
	})
	require.NoError(t, err)
	require.Equal(t, int32(1), resp.ImportedCount)

	var contactID uuid.UUID
	var companyID *uuid.UUID
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT id, company_id FROM contacts WHERE tenant_id = $1 AND email = $2`,
		tenantID, email,
	).Scan(&contactID, &companyID))
	if companyID != nil {
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "companies", *companyID) })
	}
	return contactID
}

// TestEmailExportContactsCSV_DB proves ExportContactsCSV persists-then-reads a
// tenant-scoped contact through the real CRM services instead of panicking on the
// nil-provider exportService singleton from cmd/email/main.go.
func TestEmailExportContactsCSV_DB(t *testing.T) {
	pool, srv, tenantID, actorID := emailImportSetup(t)
	ctx := emailImportCtx(tenantID, actorID)

	contactID := seedEmailExportContact(t, pool, srv, ctx, tenantID, "csv-export@email-export.test.local")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "contacts", contactID) })

	resp, err := srv.ExportContactsCSV(ctx, &emailv1.ExportContactsRequest{
		ContactIds: []string{contactID.String()},
	})
	require.NoError(t, err, "must not panic on the nil-provider singleton")
	require.Equal(t, "contacts.csv", resp.Filename)
	require.Contains(t, string(resp.FileContent), "csv-export@email-export.test.local")
	require.Contains(t, string(resp.FileContent), "Acme GmbH")
}

// TestEmailExportContactsVCard_DB is the vCard sibling of the CSV test above.
func TestEmailExportContactsVCard_DB(t *testing.T) {
	pool, srv, tenantID, actorID := emailImportSetup(t)
	ctx := emailImportCtx(tenantID, actorID)

	contactID := seedEmailExportContact(t, pool, srv, ctx, tenantID, "vcard-export@email-export.test.local")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "contacts", contactID) })

	resp, err := srv.ExportContactsVCard(ctx, &emailv1.ExportContactsRequest{
		ContactIds: []string{contactID.String()},
	})
	require.NoError(t, err, "must not panic on the nil-provider singleton")
	require.Equal(t, "contacts.vcf", resp.Filename)
	require.True(t, strings.Contains(string(resp.FileContent), "vcard-export@email-export.test.local"))
}

// TestEmailExportContactsCSV_MissingTenant proves the RPC fails closed with an auth
// error rather than reaching the nil-provider path when tenant context is absent.
func TestEmailExportContactsCSV_MissingTenant(t *testing.T) {
	_, srv, _, _ := emailImportSetup(t)

	_, err := srv.ExportContactsCSV(context.Background(), &emailv1.ExportContactsRequest{
		ContactIds: []string{uuid.New().String()},
	})
	require.Error(t, err)
}

// TestEmailExportContactsCSV_ExcludesOtherTenantContact proves that requesting a
// foreign tenant's contact ID does not leak that tenant's row into the export. The
// tenant-scoped adapter's ListByIDs filters by the caller's tenant_id (see
// internal/crm/contact/postgres_repository.go ListByIDs: "WHERE id = ANY($1) AND
// tenant_id = $2"), so a foreign ID is simply dropped rather than erroring — this
// pins that no-error/no-leak behavior against a second, independently seeded tenant
// instead of assuming it.
func TestEmailExportContactsCSV_ExcludesOtherTenantContact(t *testing.T) {
	pool, srv, tenantOwn, actorOwn := emailImportSetup(t)
	ctxOwn := emailImportCtx(tenantOwn, actorOwn)

	tenantOther := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOther, "EmailExportOtherTenant")
	actorOther := uuid.New()
	_, err := pool.Exec(testutil.WithSystemCtx(context.Background()),
		`INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name)
		 VALUES ($1, $2, $3, 'x', 'Email', 'OtherImporter')`,
		actorOther, tenantOther, actorOther.String()+"@email-export-other.test.local")
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", actorOther) })
	ctxOther := emailImportCtx(tenantOther, actorOther)

	ownContactID := seedEmailExportContact(t, pool, srv, ctxOwn, tenantOwn, "own-tenant@email-export.test.local")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "contacts", ownContactID) })

	otherContactID := seedEmailExportContact(t, pool, srv, ctxOther, tenantOther, "other-tenant@email-export.test.local")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "contacts", otherContactID) })

	resp, err := srv.ExportContactsCSV(ctxOwn, &emailv1.ExportContactsRequest{
		ContactIds: []string{ownContactID.String(), otherContactID.String()},
	})
	require.NoError(t, err)

	out := string(resp.FileContent)
	require.Contains(t, out, "own-tenant@email-export.test.local")
	require.NotContains(t, out, "other-tenant@email-export.test.local",
		"cross-tenant export leaked a foreign tenant's contact row")
}

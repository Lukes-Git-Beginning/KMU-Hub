package server

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	crmcompany "github.com/kmuhub/kmuhub/internal/crm/company"
	crmcontact "github.com/kmuhub/kmuhub/internal/crm/contact"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/testutil"
	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
)

// emailImportSetup wires an EmailGRPCServer against the real DB-backed CRM contact/
// company services. Before fix-email-import-nil-provider-panic, ImportContactsCSV and
// ImportContactsVCard called the shared importService singleton from cmd/email/main.go,
// which was constructed with a nil ContactProvider — every real call panicked. This
// proves the per-request tenant-scoped adapter (mirroring CRMGRPCServer's import path)
// actually persists rows instead.
func emailImportSetup(t *testing.T) (*pgxpool.Pool, *EmailGRPCServer, uuid.UUID, uuid.UUID) {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "EmailImportTenant")

	actorID := uuid.New()
	sysCtx := testutil.WithSystemCtx(context.Background())
	_, err := pool.Exec(sysCtx,
		`INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name)
		 VALUES ($1, $2, $3, 'x', 'Email', 'Importer')`,
		actorID, tenantID, actorID.String()+"@email-import.test.local")
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", actorID) })

	contactService := crmcontact.NewService(crmcontact.NewPostgresRepository(pool))
	companyService := crmcompany.NewService(crmcompany.NewPostgresRepository(pool))

	srv := NewEmailGRPCServer(nil, nil, nil, nil, nil, nil, nil, contactService, companyService, nil, nil)

	return pool, srv, tenantID, actorID
}

func emailImportCtx(tenantID, actorID uuid.UUID) context.Context {
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	return context.WithValue(ctx, middleware.UserIDKey, actorID.String())
}

// TestEmailImportContactsCSV_DB proves ImportContactsCSV persists a tenant-scoped
// contact and its company relation through the real CRM services, the same round
// trip fix-crm-import-company (iteration 28) proved for the CRM gRPC path.
func TestEmailImportContactsCSV_DB(t *testing.T) {
	pool, srv, tenantID, actorID := emailImportSetup(t)
	ctx := emailImportCtx(tenantID, actorID)

	csv := "E-Mail,Vorname,Nachname,Firma\r\nmax@email-import.test.local,Max,Mustermann,Acme GmbH\r\n"
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
	require.NoError(t, err, "must not panic on the nil-provider singleton")
	require.Equal(t, int32(1), resp.ImportedCount)
	require.Empty(t, resp.Errors)

	var contactID uuid.UUID
	var companyID *uuid.UUID
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT id, company_id FROM contacts WHERE tenant_id = $1 AND email = $2`,
		tenantID, "max@email-import.test.local",
	).Scan(&contactID, &companyID))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "contacts", contactID) })

	require.NotNil(t, companyID, "expected the company relation to be persisted")

	var companyName string
	var companyTenant uuid.UUID
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT name, tenant_id FROM companies WHERE id = $1`, *companyID,
	).Scan(&companyName, &companyTenant))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "companies", *companyID) })

	require.Equal(t, "Acme GmbH", companyName)
	require.Equal(t, tenantID, companyTenant)
}

// TestEmailImportContactsVCard_DB is the vCard sibling of the CSV test above — both
// RPCs were fixed independently (each builds its own adapter), so both need their own
// proof.
func TestEmailImportContactsVCard_DB(t *testing.T) {
	pool, srv, tenantID, actorID := emailImportSetup(t)
	ctx := emailImportCtx(tenantID, actorID)

	vcard := "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Erika Musterfrau\r\nEMAIL:erika@email-import.test.local\r\nEND:VCARD\r\n"
	resp, err := srv.ImportContactsVCard(ctx, &emailv1.ImportContactsVCardRequest{
		FileContent: []byte(vcard),
		Visibility:  "shared",
	})
	require.NoError(t, err, "must not panic on the nil-provider singleton")
	require.Equal(t, int32(1), resp.ImportedCount)

	var contactID uuid.UUID
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT id FROM contacts WHERE tenant_id = $1 AND email = $2`,
		tenantID, "erika@email-import.test.local",
	).Scan(&contactID))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "contacts", contactID) })
}

// TestEmailImportContactsCSV_MissingTenant proves the RPC fails closed with an auth
// error rather than reaching the nil-provider path when tenant context is absent.
func TestEmailImportContactsCSV_MissingTenant(t *testing.T) {
	_, srv, _, _ := emailImportSetup(t)

	_, err := srv.ImportContactsCSV(context.Background(), &emailv1.ImportContactsCSVRequest{
		FileContent: []byte("E-Mail,Vorname,Nachname\r\nx@example.com,X,Y\r\n"),
		FieldMapping: map[string]string{
			"E-Mail":   "email",
			"Vorname":  "first_name",
			"Nachname": "last_name",
		},
		Visibility: "shared",
	})
	require.Error(t, err)
}

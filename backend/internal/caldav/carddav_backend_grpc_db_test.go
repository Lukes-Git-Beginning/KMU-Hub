package caldav

// DB-backed coverage for CardDAVBackend against a REAL CRM gRPC server
// (loopback TCP, middleware.TenantInboundUnaryInterceptor -- the exact
// interceptor production wires in cmd/gateway/registry.go) backed by a real
// contact.Service + PostgresRepository. Template: the loopback-TCP fixture
// pattern from internal/gateway/route_hr_manual_entry_idempotency_db_test.go.
//
// CONFIRMED PRODUCTION BUG (same root cause as
// fix-caldav-write-and-exceptions-blocked-by-missing-tenant-ctx, but a wider
// blast radius): CardDAVRoutes.basicAuthMiddleware (route_caldav.go) injects
// only the userID via CtxWithUser -- never middleware.TenantIDKey. Every
// CardDAV RPC call (ListContacts, GetContact, CreateContact, UpdateContact,
// DeleteContact, UpdateContactVisibility in crm_grpc.go) starts with
// `middleware.GetTenantID(ctx)` and fails closed with
// `codes.Unauthenticated, "tenant_id missing from context"` when it's
// missing. registry.GetConnection's TenantOutboundUnaryInterceptor only
// attaches the x-tenant-id gRPC header when GetTenantID(ctx) succeeds on the
// CLIENT side too -- so under the real CardDAV request path it never even
// reaches the wire. Net effect: EVERY CardDAV operation (address book
// listing, single-contact fetch, create/update/delete) 401s for every real
// client (Apple Contacts, DAVx5, Thunderbird), including the account owner.
// This is broader than the CalDAV-side finding (which was scoped to two
// direct-DB-query helpers) because it hits the entire gRPC-mediated
// read/write surface, not just two functions.
//
// TestListAddressObjects_RealCardDAVContext_NoTenant_Returns401 and
// TestGetAddressObject_RealCardDAVContext_NoTenant_Returns401 prove this
// with the exact context CardDAV requests actually carry. The remaining
// tests inject middleware.TenantIDKey manually (as if the root-cause fix in
// basicAuthMiddleware already existed) to document the intended behaviour
// once that's fixed -- and, along the way, found a second bug: ListContacts'
// PageSize:1000 (comment: "Reasonable limit for CalDAV sync") is silently
// clamped to 20 server-side (contact.Service.ListWithVisibility,
// service.go:722), and CardDAV's ListAddressObjects never paginates, so any
// address book past 20 contacts syncs incomplete with no error.

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/emersion/go-vcard"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/crm/consent"
	"github.com/kmuhub/kmuhub/internal/crm/contact"
	"github.com/kmuhub/kmuhub/internal/gateway"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server"
	"github.com/kmuhub/kmuhub/internal/testutil"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// cardDAVCRMFixture wires a real CRM gRPC server (loopback TCP, same
// TenantInboundUnaryInterceptor production uses) behind a CardDAVBackend,
// exactly the path a real CardDAV request travels in production.
type cardDAVCRMFixture struct {
	pool       *pgxpool.Pool
	tenantID   uuid.UUID
	backend    *CardDAVBackend
	grpcServer *grpc.Server
}

func newCardDAVCRMFixture(t *testing.T) *cardDAVCRMFixture {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CardDAV gRPC Fixture Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })

	contactRepo := contact.NewPostgresRepository(pool)
	contactSvc := contact.NewService(contactRepo)
	crmServer := server.NewCRMGRPCServer(nil, nil, contactSvc, nil, nil, nil, nil, nil, nil, nil, nil)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(middleware.TenantInboundUnaryInterceptor()),
	)
	crmv1.RegisterCRMServiceServer(grpcServer, crmServer)
	go func() { _ = grpcServer.Serve(lis) }()
	t.Cleanup(grpcServer.GracefulStop)

	registry := gateway.NewServiceRegistry(nil)
	registry.Register("crm", lis.Addr().String())

	return &cardDAVCRMFixture{
		pool:       pool,
		tenantID:   tenantID,
		backend:    NewCardDAVBackend(registry, nil, pool),
		grpcServer: grpcServer,
	}
}

// seedUser creates a tenant user.
func (f *cardDAVCRMFixture) seedUser(t *testing.T) uuid.UUID {
	t.Helper()
	userID := testutil.SeedRow(t, f.pool, "users", map[string]any{
		"tenant_id":     f.tenantID,
		"email":         fmt.Sprintf("carddav-grpc-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "users", userID) })
	return userID
}

// seedContact creates a contact row directly (no created_by dependency on the
// CRM service layer), tenant-scoped, with the given owner/visibility.
func (f *cardDAVCRMFixture) seedContact(t *testing.T, ownerID uuid.UUID, visibility, firstName, lastName string) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, f.pool, "contacts", map[string]any{
		"tenant_id":  f.tenantID,
		"first_name": firstName,
		"last_name":  lastName,
		"owner_id":   ownerID,
		"visibility": visibility,
		"created_by": ownerID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "contacts", id) })
	return id
}

// realCardDAVCtx mirrors exactly what basicAuthMiddleware -> CtxWithUser
// produces for a real request: the user is known, no tenant anywhere.
func realCardDAVCtx(userID uuid.UUID) context.Context {
	return CtxWithUser(context.Background(), userID)
}

// asIfTenantMiddlewareFixed injects middleware.TenantIDKey the way a fixed
// basicAuthMiddleware would (via resolveTenantID), so the tests below the two
// "MissingTenant" ones can exercise the actual listing/filter logic instead
// of being blocked by the root-cause bug they document.
func asIfTenantMiddlewareFixed(userID, tenantID uuid.UUID) context.Context {
	ctx := context.WithValue(context.Background(), middleware.TenantIDKey, tenantID.String())
	return CtxWithUser(ctx, userID)
}

func personalPath(userID uuid.UUID) string {
	return "/carddav/principals/" + userID.String() + "/addressbooks/personal/"
}

func companyPath(userID uuid.UUID) string {
	return "/carddav/principals/" + userID.String() + "/addressbooks/company/"
}

// ============================================================================
// Confirmed bug: missing tenant context under the real request path
// ============================================================================

func TestListAddressObjects_RealCardDAVContext_NoTenant_Returns401(t *testing.T) {
	f := newCardDAVCRMFixture(t)
	userID := f.seedUser(t)
	f.seedContact(t, userID, "personal", "Anna", "Mueller")

	_, err := f.backend.ListAddressObjects(realCardDAVCtx(userID), personalPath(userID), nil)

	assert.Equal(t, 401, webdavStatusCode(t, err))
}

func TestGetAddressObject_RealCardDAVContext_NoTenant_Returns401(t *testing.T) {
	f := newCardDAVCRMFixture(t)
	userID := f.seedUser(t)
	contactID := f.seedContact(t, userID, "personal", "Anna", "Mueller")

	path := personalPath(userID) + contactID.String() + ".vcf"
	_, err := f.backend.GetAddressObject(realCardDAVCtx(userID), path, nil)

	assert.Equal(t, 401, webdavStatusCode(t, err))
}

// ============================================================================
// Listing scope (done_when #1): personal vs. company address book
// ============================================================================

func TestListAddressObjects_PersonalBook_OnlyOwnPersonalContacts(t *testing.T) {
	f := newCardDAVCRMFixture(t)
	me := f.seedUser(t)
	colleague := f.seedUser(t)
	f.seedContact(t, me, "personal", "Mine", "Contact")
	f.seedContact(t, colleague, "personal", "Colleagues", "Contact")

	objs, err := f.backend.ListAddressObjects(asIfTenantMiddlewareFixed(me, f.tenantID), personalPath(me), nil)

	require.NoError(t, err)
	require.Len(t, objs, 1, "personal book must show only the requesting user's own personal contacts, not every tenant user's")
	assert.Contains(t, objs[0].Card.Value(vcard.FieldFormattedName), "Mine")
}

func TestListAddressObjects_CompanyBook_ReturnsAllSharedRegardlessOfOwner(t *testing.T) {
	f := newCardDAVCRMFixture(t)
	me := f.seedUser(t)
	colleague := f.seedUser(t)
	f.seedContact(t, me, "shared", "My", "SharedContact")
	f.seedContact(t, colleague, "shared", "Colleague", "SharedContact")
	f.seedContact(t, me, "personal", "My", "PrivateContact")

	objs, err := f.backend.ListAddressObjects(asIfTenantMiddlewareFixed(me, f.tenantID), companyPath(me), nil)

	require.NoError(t, err)
	assert.Len(t, objs, 2, "company book must show every shared contact of the tenant regardless of owner, and exclude personal ones")
}

// TestListAddressObjects_PersonalBook_SilentlyTruncatesPast20 documents a
// second, independent bug found while answering the scope question above:
// ListAddressObjects requests PageSize:1000 ("Reasonable limit for CalDAV
// sync", carddav_backend.go) but contact.Service.ListWithVisibility clamps
// any PageSize > 100 down to the default of 20 (service.go:722). CardDAV
// never reads the response's Total or paginates, so a user with more than 20
// personal contacts gets a silently incomplete address book sync -- no
// error, no indication anything is missing.
func TestListAddressObjects_PersonalBook_SilentlyTruncatesPast20(t *testing.T) {
	f := newCardDAVCRMFixture(t)
	me := f.seedUser(t)
	const seeded = 25
	for i := range seeded {
		f.seedContact(t, me, "personal", "Bulk", fmt.Sprintf("Contact%02d", i))
	}

	objs, err := f.backend.ListAddressObjects(asIfTenantMiddlewareFixed(me, f.tenantID), personalPath(me), nil)

	require.NoError(t, err)
	assert.Len(t, objs, 20, "BUG: server-side PageSize clamp (service.go:722) silently truncates CardDAV's requested 1000 down to 20; %d contacts were seeded but ListAddressObjects has no pagination to recover the rest", seeded)
}

// ============================================================================
// Anonymized contact (done_when #3): no real name/PII survives into the vCard
// ============================================================================

func TestGetAddressObject_AnonymizedContact_NoRealNameOrPIIInVCard(t *testing.T) {
	f := newCardDAVCRMFixture(t)
	me := f.seedUser(t)
	contactID := f.seedContact(t, me, "personal", "Secret", "Person")

	tenantCtx := testutil.WithTenantCtx(context.Background(), f.tenantID)
	_, err := f.pool.Exec(tenantCtx, `UPDATE contacts SET email = $2, phone = $3, notes = $4 WHERE id = $1`,
		contactID, "secret@example.com", "+49 30 000000", "sensitive note")
	require.NoError(t, err)

	consentRepo := consent.NewPostgresRepository(f.pool)
	require.NoError(t, consentRepo.AnonymizeContact(tenantCtx, contactID, f.tenantID))

	path := personalPath(me) + contactID.String() + ".vcf"
	obj, err := f.backend.GetAddressObject(asIfTenantMiddlewareFixed(me, f.tenantID), path, nil)

	require.NoError(t, err)
	fn := obj.Card.Value(vcard.FieldFormattedName)
	assert.NotContains(t, fn, "Secret")
	assert.Equal(t, consent.AnonymizedFirstName+" "+consent.AnonymizedLastName, fn)
	assert.Empty(t, obj.Card.Value(vcard.FieldEmail), "anonymized contact must not leak email via vCard")
	assert.Empty(t, obj.Card.Value(vcard.FieldTelephone), "anonymized contact must not leak phone via vCard")
	assert.Empty(t, obj.Card.Value(vcard.FieldNote), "anonymized contact must not leak notes via vCard")
}

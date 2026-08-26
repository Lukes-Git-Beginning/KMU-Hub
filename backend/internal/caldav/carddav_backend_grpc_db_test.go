package caldav

// DB-backed coverage for CardDAVBackend against a REAL CRM gRPC server
// (loopback TCP, middleware.TenantInboundUnaryInterceptor -- the exact
// interceptor production wires in cmd/gateway/registry.go) backed by a real
// contact.Service + PostgresRepository. Template: the loopback-TCP fixture
// pattern from internal/gateway/route_hr_manual_entry_idempotency_db_test.go.
//
// REGRESSION GUARD for a fixed production bug: CardDAVRoutes.basicAuthMiddleware
// (route_caldav.go) used to inject only the userID via CtxWithUser -- never
// middleware.TenantIDKey. Every CardDAV RPC call (ListContacts, GetContact,
// CreateContact, UpdateContact, DeleteContact, UpdateContactVisibility in
// crm_grpc.go) starts with `middleware.GetTenantID(ctx)` and fails closed with
// `codes.Unauthenticated, "tenant_id missing from context"`;
// registry.GetConnection's TenantOutboundUnaryInterceptor only attaches the
// x-tenant-id gRPC header when GetTenantID(ctx) succeeds on the CLIENT side
// too, so under the real CardDAV request path it never even reached the wire.
// Net effect was that EVERY CardDAV operation (address book listing,
// single-contact fetch, create/update/delete) returned 401 for every real
// client (Apple Contacts, DAVx5, Thunderbird), including the account owner.
//
// The fix is NewCtxInjector (caldav_backend.go): the Basic-Auth middleware now
// resolves the owning tenant and stamps middleware.TenantIDKey/UserIDKey the
// way the JWT middleware does. realCardDAVCtx below builds its context through
// that exact production injector, so the tests here exercise the real request
// path rather than a hand-rolled approximation -- and, along the way, found a
// second bug: ListContacts'
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

// seedTenant creates a SECOND tenant next to the fixture's own, so the tenant
// boundary can be exercised with two real users instead of a synthetic UUID.
func (f *cardDAVCRMFixture) seedTenant(t *testing.T) uuid.UUID {
	t.Helper()
	tenantID := uuid.New()
	testutil.EnsureTenant(t, f.pool, tenantID, "CardDAV gRPC Foreign Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "tenants", tenantID) })
	return tenantID
}

// seedUser creates a user in the fixture's own tenant.
func (f *cardDAVCRMFixture) seedUser(t *testing.T) uuid.UUID {
	t.Helper()
	return f.seedUserIn(t, f.tenantID)
}

// seedUserIn creates a user in an arbitrary tenant.
func (f *cardDAVCRMFixture) seedUserIn(t *testing.T, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	userID := testutil.SeedRow(t, f.pool, "users", map[string]any{
		"tenant_id":     tenantID,
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

// realCardDAVCtx builds the context a real CardDAV request carries: exactly
// what basicAuthMiddleware produces from the production injector once an
// app-specific password has been validated. Nothing is hand-injected -- the
// tenant comes out of the same users lookup production performs.
func (f *cardDAVCRMFixture) realCardDAVCtx(t *testing.T, userID uuid.UUID) context.Context {
	t.Helper()
	ctx, err := NewCtxInjector(f.pool)(context.Background(), userID)
	require.NoError(t, err)
	return ctx
}

func personalPath(userID uuid.UUID) string {
	return "/carddav/principals/" + userID.String() + "/addressbooks/personal/"
}

func companyPath(userID uuid.UUID) string {
	return "/carddav/principals/" + userID.String() + "/addressbooks/company/"
}

// ============================================================================
// Root cause (fixed): tenant context under the real request path
// ============================================================================

func TestListAddressObjects_RealCardDAVContext_ResolvesTenant(t *testing.T) {
	f := newCardDAVCRMFixture(t)
	userID := f.seedUser(t)
	f.seedContact(t, userID, "personal", "Anna", "Mueller")

	objs, err := f.backend.ListAddressObjects(f.realCardDAVCtx(t, userID), personalPath(userID), nil)

	require.NoError(t, err, "the real CardDAV request context must carry a tenant")
	assert.Len(t, objs, 1)
}

func TestGetAddressObject_RealCardDAVContext_ResolvesTenant(t *testing.T) {
	f := newCardDAVCRMFixture(t)
	userID := f.seedUser(t)
	contactID := f.seedContact(t, userID, "personal", "Anna", "Mueller")

	path := personalPath(userID) + contactID.String() + ".vcf"
	obj, err := f.backend.GetAddressObject(f.realCardDAVCtx(t, userID), path, nil)

	require.NoError(t, err, "the real CardDAV request context must carry a tenant")
	require.NotNil(t, obj)
	assert.Contains(t, obj.Path, contactID.String())
}

// TestCardDAVCtxInjector_ResolvesOnlyTheOwnTenant is the boundary half of the
// fix: the injector must hand out the tenant the authenticated user actually
// belongs to, never a wider one. An app password issued in tenant B must not
// reach a single contact of tenant A -- neither by listing the shared company
// address book nor by fetching a known contact ID directly.
func TestCardDAVCtxInjector_ResolvesOnlyTheOwnTenant(t *testing.T) {
	f := newCardDAVCRMFixture(t)
	owner := f.seedUser(t)
	foreignContact := f.seedContact(t, owner, "shared", "Anna", "Mueller")

	otherTenant := f.seedTenant(t)
	intruder := f.seedUserIn(t, otherTenant)

	ctx := f.realCardDAVCtx(t, intruder)

	objs, err := f.backend.ListAddressObjects(ctx, companyPath(intruder), nil)
	require.NoError(t, err)
	assert.Empty(t, objs, "company address book must not leak another tenant's shared contacts")

	path := companyPath(intruder) + foreignContact.String() + ".vcf"
	_, err = f.backend.GetAddressObject(ctx, path, nil)
	assert.Equal(t, 404, webdavStatusCode(t, err),
		"a known contact ID from another tenant must stay invisible")
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

	objs, err := f.backend.ListAddressObjects(f.realCardDAVCtx(t, me), personalPath(me), nil)

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

	objs, err := f.backend.ListAddressObjects(f.realCardDAVCtx(t, me), companyPath(me), nil)

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

	objs, err := f.backend.ListAddressObjects(f.realCardDAVCtx(t, me), personalPath(me), nil)

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
	obj, err := f.backend.GetAddressObject(f.realCardDAVCtx(t, me), path, nil)

	require.NoError(t, err)
	fn := obj.Card.Value(vcard.FieldFormattedName)
	assert.NotContains(t, fn, "Secret")
	assert.Equal(t, consent.AnonymizedFirstName+" "+consent.AnonymizedLastName, fn)
	assert.Empty(t, obj.Card.Value(vcard.FieldEmail), "anonymized contact must not leak email via vCard")
	assert.Empty(t, obj.Card.Value(vcard.FieldTelephone), "anonymized contact must not leak phone via vCard")
	assert.Empty(t, obj.Card.Value(vcard.FieldNote), "anonymized contact must not leak notes via vCard")
}

-- Migration 000142: RLS activation for three tables added in Chain PILOT
-- (Migrations 000134 + 000135).
--
-- Tables:
--   password_reset_tokens  (Migration 000134)
--   booking_pages          (Migration 000135)
--   public_bookings        (Migration 000135)
--
-- All three have tenant_id NOT NULL and were written with Option-B/RLS in mind.
--
-- ============================================================================
-- Pre-RLS Audit summary (file-level, checked before writing this migration):
--
-- [password_reset_tokens]
--   INSERT: CreatePasswordResetToken (postgres_repository.go:455) stamps token.TenantID
--           (set from user.TenantID in service.go:547). ✓ WIRED.
--   SELECT: GetPasswordResetToken / MarkPasswordResetTokenUsed — BOTH called via
--           sysCtx = sysctx.With(ctx) in service.go:586+615. ✓ SYSTEM-BYPASS.
--   Risk: The reset flow is entirely unauthenticated (user clicks mail link, no JWT).
--         The service wraps ALL password-reset repo calls in sysctx.With, so the
--         PrepareConn hook sets app.role='system' → is_system_context() = true →
--         the standard enable_tenant_rls policy admits the operation.
--         Standard CALL enable_tenant_rls is SAFE here.
--
-- [booking_pages]
--   INSERT/UPDATE/DELETE: Admin paths (CreateBookingPage, UpdateBookingPage,
--         DeleteBookingPage). Called from CalendarGRPCServer with an authenticated
--         context that carries app.tenant_id via TenantInboundUnaryInterceptor.
--         Repo methods supply tenantID explicitly in WHERE tenant_id=$N.
--         The PrepareConn hook will have set app.tenant_id from the gRPC metadata. ✓
--   SELECT (admin): GetBookingPageByID / ListBookingPages — tenant_id passed as arg
--         AND stamped in GUC. ✓
--   SELECT (public): GetBookingPageBySlug — called by GetPublicBookingPage and
--         GetAvailability/CreatePublicBooking. These arrive via unauthenticated HTTP
--         routes. The gRPC call carries NO x-tenant-id metadata header
--         (TenantOutboundUnaryInterceptor only copies from ctx, which is empty for
--         public routes). PrepareConn falls through to the no-tenant branch → GUCs
--         left at '' defaults → current_tenant_id() returns NULL.
--         The enable_tenant_rls policy would filter EVERY row away (tenant_id = NULL
--         is never true), making slug-based lookup ALWAYS return ErrBookingPageNotFound.
--   FIX: Custom policy that adds an OR branch for slug-based public reads.
--         Use is_system_context() as the public bypass — the CalendarGRPCServer's
--         public RPCs must wrap the context with sysctx.With before the DB call.
--         HOWEVER: The existing code does NOT do this — it passes ctx unchanged.
--         This means two paths are available:
--         (A) Add is_system_context() to the USING clause now and document that a
--             follow-up code change is REQUIRED in calendar_grpc.go to call
--             sysctx.With for the three public RPCs.
--         (B) Skip RLS for booking_pages and public_bookings until the code is fixed.
--         We choose (A): the policy is written now; the code follow-up is a hard
--         requirement (without it, public booking is broken). This is documented
--         explicitly below.
--
-- [public_bookings]
--   INSERT: CreatePublicBooking derives tenant_id from page.TenantID (service.go:388).
--           Same unauthenticated context problem as booking_pages SELECT.
--           WITHOUT sysctx.With, the INSERT would be rejected by RLS too.
--   SELECT: GetBookedSlotsForPage — same context path. Same problem.
--   Same fix as booking_pages: is_system_context() branch + code-fix requirement.
--
-- ============================================================================
-- REQUIRED CODE CHANGE (not in scope of this migration):
--   backend/internal/server/calendar_grpc.go — three public RPC handlers must
--   wrap the incoming context with sysctx.With before calling the booking service:
--     GetPublicBookingPage  → ctx = sysctx.With(ctx)
--     GetAvailability       → ctx = sysctx.With(ctx)
--     CreatePublicBooking   → ctx = sysctx.With(ctx)
--   Without this, all public booking endpoints will return "not found" errors once
--   this migration runs. This is the same pattern used for all auth pre-JWT paths
--   (service.go Login/Register/RefreshToken/RequestPasswordReset/ResetPassword).
-- ============================================================================

SET LOCAL row_security = off;

BEGIN;

-- ============================================================================
-- password_reset_tokens — Standard enable_tenant_rls.
-- All repo operations are wrapped in sysctx.With (sysCtx) in service.go,
-- so is_system_context() = true for every password-reset DB call.
-- ============================================================================
CALL enable_tenant_rls('password_reset_tokens');

-- ============================================================================
-- booking_pages — Custom policy with is_system_context() bypass.
-- Admin paths: standard tenant isolation (app.tenant_id stamped via gRPC metadata).
-- Public paths: must run under sysctx.With (code change required in calendar_grpc.go).
-- The is_system_context() branch covers both migration workers and the public
-- RPC handlers once the required code change is applied.
-- ============================================================================
ALTER TABLE booking_pages ENABLE ROW LEVEL SECURITY;
ALTER TABLE booking_pages FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON booking_pages
    USING  (tenant_id = current_tenant_id() OR is_system_context())
    WITH CHECK (tenant_id = current_tenant_id() OR is_system_context());

-- ============================================================================
-- public_bookings — Same custom policy as booking_pages.
-- The INSERT tenant_id is derived from booking_pages.tenant_id (not from the
-- caller's JWT), so the code path also requires sysctx.With in calendar_grpc.go.
-- ============================================================================
ALTER TABLE public_bookings ENABLE ROW LEVEL SECURITY;
ALTER TABLE public_bookings FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public_bookings
    USING  (tenant_id = current_tenant_id() OR is_system_context())
    WITH CHECK (tenant_id = current_tenant_id() OR is_system_context());

COMMIT;

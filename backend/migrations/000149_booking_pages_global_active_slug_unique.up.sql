-- Migration 000149: global-unique active booking-page slug
-- The public booking lookup (GetBookingPageBySlug) resolves a slug WITHOUT a
-- tenant filter (unauth path, no JWT → no tenant in ctx). Slugs are only
-- UNIQUE(tenant_id, slug), so under multi-tenant two tenants could register the
-- same active slug and the lookup would silently pick the oldest — cross-tenant
-- booking misrouting (WP-F). This partial unique index makes a duplicate ACTIVE
-- slug impossible across all tenants, so the unauth lookup is unambiguous.
-- Interim guard until the Option-B retrofit adds a real tenant discriminator
-- (e.g. subdomain) to the public URL. Coexists with booking_pages_tenant_slug_unique.
--
-- PRE-FLIGHT (single-tenant today → guaranteed empty, verify anyway):
--   SELECT slug, count(*) FROM booking_pages WHERE active GROUP BY slug HAVING count(*) > 1;

BEGIN;

CREATE UNIQUE INDEX idx_booking_pages_active_slug
    ON booking_pages (slug) WHERE active = true;

COMMIT;

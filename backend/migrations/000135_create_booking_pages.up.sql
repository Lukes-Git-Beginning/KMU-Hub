-- Migration 000135 — Public Booking Pages + Public Bookings (Greenfield)
-- Kalender Public-Booking feature: tenant-owned booking pages with slug-based
-- public access, service catalogue (JSONB), availability rules (JSONB), and
-- a public_bookings table that records each external appointment.
--
-- tenant_id NOT NULL on both tables (Option-B, pre-RLS-ready).
-- UNIQUE(tenant_id, slug) prevents cross-tenant slug collisions.
-- UNIQUE(booking_page_id, customer_email, date, time_slot) is the double-submit backstop.

BEGIN;

-- ============================================================================
-- booking_pages
-- ============================================================================

CREATE TABLE IF NOT EXISTS booking_pages (
    id               UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id        UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    calendar_id      UUID        NOT NULL REFERENCES calendars(id) ON DELETE CASCADE,
    slug             TEXT        NOT NULL,
    company_name     TEXT        NOT NULL,
    logo_url         TEXT,
    -- services: array of {id, name, duration_min, price, description?}
    services         JSONB       NOT NULL DEFAULT '[]',
    -- availability_rules: {weekdays, slots_per_weekday, slot_duration_min, buffer_min, lead_time_hours, breaks}
    availability_rules JSONB     NOT NULL DEFAULT '{}',
    active           BOOLEAN     NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT booking_pages_tenant_slug_unique UNIQUE (tenant_id, slug),
    CONSTRAINT booking_pages_slug_format CHECK (slug ~ '^[a-z0-9][a-z0-9\-]{1,61}[a-z0-9]$')
);

CREATE INDEX IF NOT EXISTS idx_booking_pages_tenant_id  ON booking_pages (tenant_id);
CREATE INDEX IF NOT EXISTS idx_booking_pages_calendar_id ON booking_pages (calendar_id);
CREATE INDEX IF NOT EXISTS idx_booking_pages_slug       ON booking_pages (slug);
CREATE INDEX IF NOT EXISTS idx_booking_pages_active     ON booking_pages (tenant_id, active) WHERE active = true;

-- ============================================================================
-- public_bookings
-- ============================================================================

CREATE TABLE IF NOT EXISTS public_bookings (
    id               UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id        UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    booking_page_id  UUID        NOT NULL REFERENCES booking_pages(id) ON DELETE CASCADE,
    service_id       TEXT        NOT NULL,   -- references services[].id in booking_pages.services JSONB
    customer_name    TEXT        NOT NULL,
    customer_email   TEXT        NOT NULL,
    customer_phone   TEXT,
    notes            TEXT,
    date             DATE        NOT NULL,
    time_slot        TEXT        NOT NULL,   -- "HH:MM" start time, e.g. "09:00"
    staff_user_id    UUID,                   -- assigned staff member (may be NULL until assigned)
    status           TEXT        NOT NULL DEFAULT 'confirmed'
                                 CHECK (status IN ('pending','confirmed','cancelled','completed')),
    calendar_event_id UUID,                  -- FK to calendar_events.id (set after event creation)
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Double-submit backstop: same customer cannot book the same slot on the same page+date
    CONSTRAINT public_bookings_unique_slot UNIQUE (booking_page_id, customer_email, date, time_slot)
);

CREATE INDEX IF NOT EXISTS idx_public_bookings_tenant_id      ON public_bookings (tenant_id);
CREATE INDEX IF NOT EXISTS idx_public_bookings_booking_page   ON public_bookings (booking_page_id);
CREATE INDEX IF NOT EXISTS idx_public_bookings_date           ON public_bookings (booking_page_id, date);
CREATE INDEX IF NOT EXISTS idx_public_bookings_status         ON public_bookings (tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_public_bookings_calendar_event ON public_bookings (calendar_event_id) WHERE calendar_event_id IS NOT NULL;

COMMIT;

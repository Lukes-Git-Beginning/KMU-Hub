-- Migration 000046: Create HR tables for Phase 13 (HR & Zeiterfassung)
-- Tables: company settings, employee profiles, leave types, leave requests,
--         leave balances, work time entries, break entries, document categories,
--         employee documents

-- HR company settings (per-tenant configuration)
CREATE TABLE hr_company_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    au_threshold_days INT NOT NULL DEFAULT 3,
    show_absence_reason BOOLEAN NOT NULL DEFAULT TRUE,
    default_annual_leave_days INT NOT NULL DEFAULT 20,
    timezone VARCHAR(50) NOT NULL DEFAULT 'Europe/Berlin',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_hr_company_settings_tenant UNIQUE (tenant_id)
);

-- Employee profiles (extends users table with HR data)
CREATE TABLE hr_employee_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    department VARCHAR(100),
    position_title VARCHAR(200),
    contract_type VARCHAR(20) NOT NULL DEFAULT 'full_time',
    work_days_per_week INT NOT NULL DEFAULT 5,
    annual_leave_days INT NOT NULL DEFAULT 20,
    manager_user_id UUID REFERENCES users(id),
    start_date DATE NOT NULL,
    emergency_contact_name VARCHAR(200),
    emergency_contact_phone VARCHAR(50),
    address_street VARCHAR(255),
    address_city VARCHAR(100),
    address_postal_code VARCHAR(10),
    address_country VARCHAR(2) NOT NULL DEFAULT 'DE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_hr_employee_user UNIQUE (user_id),
    CONSTRAINT chk_hr_contract_type CHECK (contract_type IN ('full_time', 'part_time', 'mini_job', 'intern', 'temporary')),
    CONSTRAINT chk_hr_work_days CHECK (work_days_per_week BETWEEN 1 AND 7)
);

CREATE INDEX idx_hr_employee_profiles_user ON hr_employee_profiles(user_id);
CREATE INDEX idx_hr_employee_profiles_manager ON hr_employee_profiles(manager_user_id);
CREATE INDEX idx_hr_employee_profiles_department ON hr_employee_profiles(department);

-- Leave types (predefined + admin-configurable)
CREATE TABLE hr_leave_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    key VARCHAR(50) NOT NULL,
    color VARCHAR(20) NOT NULL DEFAULT '#3d8abf',
    deducts_from_balance BOOLEAN NOT NULL DEFAULT TRUE,
    requires_approval BOOLEAN NOT NULL DEFAULT TRUE,
    requires_au_document BOOLEAN NOT NULL DEFAULT FALSE,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_hr_leave_type_key UNIQUE (tenant_id, key)
);

CREATE INDEX idx_hr_leave_types_tenant ON hr_leave_types(tenant_id);

-- Leave requests
CREATE TABLE hr_leave_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    employee_id UUID NOT NULL REFERENCES users(id),
    leave_type_id UUID NOT NULL REFERENCES hr_leave_types(id),
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    is_half_day_start BOOLEAN NOT NULL DEFAULT FALSE,
    half_day_period_start VARCHAR(10),
    is_half_day_end BOOLEAN NOT NULL DEFAULT FALSE,
    half_day_period_end VARCHAR(10),
    total_days NUMERIC(5,1) NOT NULL,
    reason TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    approved_by UUID REFERENCES users(id),
    approval_comment TEXT,
    approved_at TIMESTAMPTZ,
    au_document_required BOOLEAN NOT NULL DEFAULT FALSE,
    au_document_file_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_leave_dates CHECK (end_date >= start_date),
    CONSTRAINT chk_leave_status CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled')),
    CONSTRAINT chk_half_day_period_start CHECK (half_day_period_start IS NULL OR half_day_period_start IN ('morning', 'afternoon')),
    CONSTRAINT chk_half_day_period_end CHECK (half_day_period_end IS NULL OR half_day_period_end IN ('morning', 'afternoon'))
);

CREATE INDEX idx_hr_leave_requests_employee ON hr_leave_requests(employee_id);
CREATE INDEX idx_hr_leave_requests_tenant ON hr_leave_requests(tenant_id);
CREATE INDEX idx_hr_leave_requests_dates ON hr_leave_requests(start_date, end_date);
CREATE INDEX idx_hr_leave_requests_status ON hr_leave_requests(status);
CREATE INDEX idx_hr_leave_requests_leave_type ON hr_leave_requests(leave_type_id);

-- Leave balance tracking (per employee per year)
CREATE TABLE hr_leave_balances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    employee_id UUID NOT NULL REFERENCES users(id),
    year INT NOT NULL,
    entitlement NUMERIC(5,1) NOT NULL,
    carried_over NUMERIC(5,1) NOT NULL DEFAULT 0,
    used NUMERIC(5,1) NOT NULL DEFAULT 0,
    remaining NUMERIC(5,1) NOT NULL,
    carryover_expires_at DATE,
    carryover_notified BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_hr_leave_balance UNIQUE (tenant_id, employee_id, year)
);

CREATE INDEX idx_hr_leave_balances_employee ON hr_leave_balances(employee_id);
CREATE INDEX idx_hr_leave_balances_year ON hr_leave_balances(year);

-- Work time entries (clock in/out for ArbZG compliance)
CREATE TABLE hr_work_time_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    employee_id UUID NOT NULL REFERENCES users(id),
    clock_in TIMESTAMPTZ NOT NULL,
    clock_out TIMESTAMPTZ,
    break_minutes INT NOT NULL DEFAULT 0,
    auto_break_deducted INT NOT NULL DEFAULT 0,
    net_work_minutes INT,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    is_correction BOOLEAN NOT NULL DEFAULT FALSE,
    original_entry_id UUID REFERENCES hr_work_time_entries(id),
    correction_reason TEXT,
    correction_approved_by UUID REFERENCES users(id),
    correction_approved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_work_time_status CHECK (status IN ('active', 'completed', 'correction_pending', 'correction_approved'))
);

CREATE INDEX idx_hr_work_time_entries_employee ON hr_work_time_entries(employee_id);
CREATE INDEX idx_hr_work_time_entries_clock_in ON hr_work_time_entries(clock_in);
CREATE INDEX idx_hr_work_time_entries_tenant ON hr_work_time_entries(tenant_id);
-- Partial index for quickly finding active shifts (at most one per employee)
CREATE UNIQUE INDEX idx_hr_work_time_entries_active ON hr_work_time_entries(employee_id) WHERE status = 'active';

-- Break entries (separate break tracking linked to work time entries)
CREATE TABLE hr_break_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    work_time_entry_id UUID NOT NULL REFERENCES hr_work_time_entries(id) ON DELETE CASCADE,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ,
    duration_minutes INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_hr_break_entries_work_time ON hr_break_entries(work_time_entry_id);

-- HR document categories (predefined + admin-configurable)
CREATE TABLE hr_document_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    key VARCHAR(50) NOT NULL,
    visibility VARCHAR(20) NOT NULL DEFAULT 'hr_only',
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_hr_doc_category_key UNIQUE (tenant_id, key),
    CONSTRAINT chk_doc_visibility CHECK (visibility IN ('hr_only', 'manager', 'employee'))
);

CREATE INDEX idx_hr_document_categories_tenant ON hr_document_categories(tenant_id);

-- HR document links (links document service files to HR context)
CREATE TABLE hr_employee_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    employee_id UUID NOT NULL REFERENCES users(id),
    category_id UUID NOT NULL REFERENCES hr_document_categories(id),
    file_id UUID NOT NULL,
    uploaded_by UUID NOT NULL REFERENCES users(id),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_hr_doc_file UNIQUE (employee_id, file_id)
);

CREATE INDEX idx_hr_employee_documents_employee ON hr_employee_documents(employee_id);
CREATE INDEX idx_hr_employee_documents_category ON hr_employee_documents(category_id);

-- ============================================================================
-- Seed system leave types (tenant_id will be populated per-tenant at runtime,
-- using a placeholder UUID for the system default set)
-- ============================================================================

-- Note: These system leave types use a zero UUID as tenant_id placeholder.
-- The application layer copies these to each tenant on first access.
INSERT INTO hr_leave_types (tenant_id, key, name, color, deducts_from_balance, requires_approval, requires_au_document, is_system, sort_order) VALUES
    ('00000000-0000-0000-0000-000000000000', 'urlaub', 'Urlaub', '#3d8abf', TRUE, TRUE, FALSE, TRUE, 1),
    ('00000000-0000-0000-0000-000000000000', 'krank', 'Krankheit', '#bf3d3d', FALSE, FALSE, FALSE, TRUE, 2),
    ('00000000-0000-0000-0000-000000000000', 'sonderurlaub_hochzeit', 'Sonderurlaub (Hochzeit)', '#7c5a8a', FALSE, TRUE, FALSE, TRUE, 3),
    ('00000000-0000-0000-0000-000000000000', 'sonderurlaub_geburt', 'Sonderurlaub (Geburt)', '#7c5a8a', FALSE, TRUE, FALSE, TRUE, 4),
    ('00000000-0000-0000-0000-000000000000', 'sonderurlaub_todesfall', 'Sonderurlaub (Todesfall)', '#7c5a8a', FALSE, TRUE, FALSE, TRUE, 5),
    ('00000000-0000-0000-0000-000000000000', 'sonderurlaub_umzug', 'Sonderurlaub (Umzug)', '#7c5a8a', FALSE, TRUE, FALSE, TRUE, 6),
    ('00000000-0000-0000-0000-000000000000', 'elternzeit', 'Elternzeit', '#4a7c6a', FALSE, TRUE, FALSE, TRUE, 7),
    ('00000000-0000-0000-0000-000000000000', 'unbezahlter_urlaub', 'Unbezahlter Urlaub', '#999999', FALSE, TRUE, FALSE, TRUE, 8),
    ('00000000-0000-0000-0000-000000000000', 'homeoffice', 'Homeoffice', '#4a7c6a', FALSE, TRUE, FALSE, TRUE, 9),
    ('00000000-0000-0000-0000-000000000000', 'weiterbildung', 'Weiterbildung', '#bf8a3d', FALSE, TRUE, FALSE, TRUE, 10);

-- Seed system document categories
INSERT INTO hr_document_categories (tenant_id, key, name, visibility, is_system, sort_order) VALUES
    ('00000000-0000-0000-0000-000000000000', 'arbeitsvertrag', 'Arbeitsvertrag', 'hr_only', TRUE, 1),
    ('00000000-0000-0000-0000-000000000000', 'zeugnisse', 'Zeugnisse', 'manager', TRUE, 2),
    ('00000000-0000-0000-0000-000000000000', 'abmahnungen', 'Abmahnungen', 'hr_only', TRUE, 3),
    ('00000000-0000-0000-0000-000000000000', 'sonstiges', 'Sonstiges', 'employee', TRUE, 4);

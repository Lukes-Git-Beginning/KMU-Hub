-- Finance tables for the Biz service (Rechnungen & Finanzen)

-- Company settings: one row per tenant with tax/bank/display information
CREATE TABLE company_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL DEFAULT '',
    street VARCHAR(255) NOT NULL DEFAULT '',
    plz VARCHAR(10) NOT NULL DEFAULT '',
    city VARCHAR(100) NOT NULL DEFAULT '',
    country VARCHAR(2) NOT NULL DEFAULT 'DE',
    steuernummer VARCHAR(30) NOT NULL DEFAULT '',
    ust_id_nr VARCHAR(20) NOT NULL DEFAULT '',
    handelsregister VARCHAR(50) NOT NULL DEFAULT '',
    bank_name VARCHAR(100) NOT NULL DEFAULT '',
    iban VARCHAR(34) NOT NULL DEFAULT '',
    bic VARCHAR(11) NOT NULL DEFAULT '',
    logo_url TEXT NOT NULL DEFAULT '',
    accent_color VARCHAR(20) NOT NULL DEFAULT '',
    is_kleinunternehmer BOOLEAN NOT NULL DEFAULT FALSE,
    default_payment_terms_days INT NOT NULL DEFAULT 30,
    default_quote_validity_days INT NOT NULL DEFAULT 30,
    basiszinssatz NUMERIC(5,2) NOT NULL DEFAULT 1.27,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_company_settings_tenant UNIQUE (tenant_id)
);

-- Number sequences for generating document numbers (RE-2026-001, AN-2026-001, etc.)
CREATE TABLE finance_number_sequences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    document_type VARCHAR(20) NOT NULL,
    prefix VARCHAR(10) NOT NULL DEFAULT '',
    current_number INT NOT NULL DEFAULT 0,
    fiscal_year INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_finance_number_seq UNIQUE (tenant_id, document_type, fiscal_year)
);

-- Quotes (Angebote)
CREATE TABLE finance_quotes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    quote_number VARCHAR(30) NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    customer_name VARCHAR(255) NOT NULL DEFAULT '',
    customer_address TEXT NOT NULL DEFAULT '',
    customer_email VARCHAR(255) NOT NULL DEFAULT '',
    customer_ust_id_nr VARCHAR(20) NOT NULL DEFAULT '',
    tax_mode VARCHAR(30) NOT NULL DEFAULT 'standard',
    line_items JSONB NOT NULL DEFAULT '[]',
    tax_breakdown JSONB,
    subtotal NUMERIC(15,2) NOT NULL DEFAULT 0,
    total_tax NUMERIC(15,2) NOT NULL DEFAULT 0,
    gross_total NUMERIC(15,2) NOT NULL DEFAULT 0,
    valid_until DATE,
    notes TEXT NOT NULL DEFAULT '',
    deal_id UUID,
    source_quote_id UUID,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_finance_quotes_status CHECK (status IN ('draft','sent','accepted','rejected','expired')),
    CONSTRAINT chk_finance_quotes_tax_mode CHECK (tax_mode IN ('standard','reverse_charge','kleinunternehmer')),
    CONSTRAINT fk_finance_quotes_deal FOREIGN KEY (deal_id) REFERENCES deals(id) ON DELETE SET NULL,
    CONSTRAINT fk_finance_quotes_source FOREIGN KEY (source_quote_id) REFERENCES finance_quotes(id) ON DELETE SET NULL
);

-- Invoices (Rechnungen)
CREATE TABLE finance_invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    invoice_number VARCHAR(30) NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    customer_name VARCHAR(255) NOT NULL DEFAULT '',
    customer_address TEXT NOT NULL DEFAULT '',
    customer_email VARCHAR(255) NOT NULL DEFAULT '',
    customer_ust_id_nr VARCHAR(20) NOT NULL DEFAULT '',
    company_snapshot JSONB,
    tax_mode VARCHAR(30) NOT NULL DEFAULT 'standard',
    line_items JSONB NOT NULL DEFAULT '[]',
    tax_breakdown JSONB,
    subtotal NUMERIC(15,2) NOT NULL DEFAULT 0,
    total_tax NUMERIC(15,2) NOT NULL DEFAULT 0,
    gross_total NUMERIC(15,2) NOT NULL DEFAULT 0,
    invoice_date DATE NOT NULL,
    delivery_date DATE,
    due_date DATE NOT NULL,
    payment_terms TEXT NOT NULL DEFAULT '',
    snapshot_data JSONB,
    source_quote_id UUID,
    notes TEXT NOT NULL DEFAULT '',
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_finance_invoices_status CHECK (status IN ('draft','sent','paid','overdue','cancelled')),
    CONSTRAINT chk_finance_invoices_tax_mode CHECK (tax_mode IN ('standard','reverse_charge','kleinunternehmer'))
);

-- Credit Notes (Gutschriften)
CREATE TABLE finance_credit_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    credit_note_number VARCHAR(30) NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    original_invoice_id UUID NOT NULL,
    customer_name VARCHAR(255) NOT NULL DEFAULT '',
    customer_address TEXT NOT NULL DEFAULT '',
    customer_email VARCHAR(255) NOT NULL DEFAULT '',
    customer_ust_id_nr VARCHAR(20) NOT NULL DEFAULT '',
    tax_mode VARCHAR(30) NOT NULL DEFAULT 'standard',
    line_items JSONB NOT NULL DEFAULT '[]',
    tax_breakdown JSONB,
    subtotal NUMERIC(15,2) NOT NULL DEFAULT 0,
    total_tax NUMERIC(15,2) NOT NULL DEFAULT 0,
    gross_total NUMERIC(15,2) NOT NULL DEFAULT 0,
    reason TEXT NOT NULL DEFAULT '',
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_finance_credit_notes_status CHECK (status IN ('draft','sent')),
    CONSTRAINT chk_finance_credit_notes_tax_mode CHECK (tax_mode IN ('standard','reverse_charge','kleinunternehmer')),
    CONSTRAINT fk_finance_credit_notes_invoice FOREIGN KEY (original_invoice_id) REFERENCES finance_invoices(id) ON DELETE RESTRICT
);

-- Payments (Zahlungen)
CREATE TABLE finance_payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    invoice_id UUID NOT NULL,
    amount NUMERIC(15,2) NOT NULL,
    payment_date DATE NOT NULL,
    method VARCHAR(20) NOT NULL DEFAULT 'bank_transfer',
    reference VARCHAR(100) NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_finance_payments_method CHECK (method IN ('bank_transfer','cash','credit_card','other')),
    CONSTRAINT fk_finance_payments_invoice FOREIGN KEY (invoice_id) REFERENCES finance_invoices(id) ON DELETE RESTRICT
);

-- Dunning Records (Mahnungen)
CREATE TABLE finance_dunning_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    invoice_id UUID NOT NULL,
    level INT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    fee NUMERIC(10,2) NOT NULL DEFAULT 0,
    interest NUMERIC(10,2) NOT NULL DEFAULT 0,
    sent_at TIMESTAMPTZ,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_finance_dunning_level CHECK (level BETWEEN 1 AND 3),
    CONSTRAINT chk_finance_dunning_status CHECK (status IN ('draft','sent','paid')),
    CONSTRAINT fk_finance_dunning_invoice FOREIGN KEY (invoice_id) REFERENCES finance_invoices(id) ON DELETE RESTRICT
);

-- Dunning Configuration (Mahneinstellungen)
CREATE TABLE finance_dunning_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    level1_days_after_due INT NOT NULL,
    level2_days_after_level1 INT NOT NULL,
    level3_days_after_level2 INT NOT NULL,
    level1_fee NUMERIC(10,2) NOT NULL DEFAULT 0,
    level2_fee NUMERIC(10,2) NOT NULL DEFAULT 5,
    level3_fee NUMERIC(10,2) NOT NULL DEFAULT 10,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_finance_dunning_config_tenant UNIQUE (tenant_id)
);

-- ============================================================================
-- Indexes
-- ============================================================================

CREATE INDEX idx_finance_quotes_tenant_status ON finance_quotes(tenant_id, status);
CREATE INDEX idx_finance_quotes_deal ON finance_quotes(deal_id) WHERE deal_id IS NOT NULL;

CREATE INDEX idx_finance_invoices_tenant_status ON finance_invoices(tenant_id, status);
CREATE INDEX idx_finance_invoices_due_date ON finance_invoices(due_date) WHERE status = 'sent';
CREATE UNIQUE INDEX idx_finance_invoices_number ON finance_invoices(tenant_id, invoice_number);

CREATE INDEX idx_finance_credit_notes_invoice ON finance_credit_notes(original_invoice_id);

CREATE INDEX idx_finance_payments_invoice ON finance_payments(invoice_id);

CREATE INDEX idx_finance_dunning_invoice ON finance_dunning_records(invoice_id);

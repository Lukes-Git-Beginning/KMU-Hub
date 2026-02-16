-- Security tables: audit log, recovery codes, 2FA policy, sessions, vault, GDPR, passwords, IP rules

-- Audit log with hash chaining for tamper evidence
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sequence_num BIGSERIAL NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id UUID REFERENCES users(id),
    action VARCHAR(100) NOT NULL,
    target VARCHAR(255),
    target_type VARCHAR(50),
    details JSONB,
    ip_address INET,
    user_agent TEXT,
    result VARCHAR(20) NOT NULL DEFAULT 'success',
    previous_hash VARCHAR(64) NOT NULL DEFAULT '',
    entry_hash VARCHAR(64) NOT NULL
);

CREATE INDEX idx_audit_log_timestamp ON audit_log (timestamp DESC);
CREATE INDEX idx_audit_log_user_id ON audit_log (user_id);
CREATE INDEX idx_audit_log_action ON audit_log (action);
CREATE INDEX idx_audit_log_result ON audit_log (result);
CREATE INDEX idx_audit_log_sequence ON audit_log (sequence_num DESC);
CREATE INDEX idx_audit_log_timestamp_action ON audit_log (timestamp DESC, action);

-- Recovery codes for 2FA (hashed, single-use)
CREATE TABLE recovery_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash VARCHAR(64) NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_recovery_codes_user_id ON recovery_codes (user_id);

-- Per-role 2FA enforcement policy
CREATE TABLE two_factor_policy (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_name VARCHAR(50) NOT NULL,
    enforced BOOLEAN NOT NULL DEFAULT false,
    grace_period_days INT NOT NULL DEFAULT 14,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID REFERENCES users(id)
);

CREATE UNIQUE INDEX idx_two_factor_policy_role ON two_factor_policy (role_name);

-- User sessions with device and location metadata
CREATE TABLE user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_id UUID REFERENCES refresh_tokens(id) ON DELETE SET NULL,
    device_name VARCHAR(255),
    device_type VARCHAR(50),
    ip_address INET,
    location VARCHAR(255),
    user_agent TEXT,
    is_current BOOLEAN NOT NULL DEFAULT false,
    last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_sessions_user_id ON user_sessions (user_id);
CREATE INDEX idx_user_sessions_last_active ON user_sessions (last_active_at DESC);

-- Vault for encrypted secret storage (API keys, SMTP passwords, integration credentials)
CREATE TABLE vault_secrets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_name VARCHAR(255) NOT NULL,
    encrypted_value TEXT NOT NULL,
    key_version INT NOT NULL DEFAULT 1,
    description TEXT,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_vault_secrets_key_name ON vault_secrets (key_name);

-- GDPR data export requests
CREATE TABLE gdpr_export_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_by UUID REFERENCES users(id),
    reviewed_at TIMESTAMPTZ,
    review_note TEXT,
    export_data BYTEA,
    download_token VARCHAR(64),
    download_expires_at TIMESTAMPTZ,
    downloaded_at TIMESTAMPTZ
);

CREATE INDEX idx_gdpr_exports_user_id ON gdpr_export_requests (user_id);
CREATE INDEX idx_gdpr_exports_status ON gdpr_export_requests (status);

-- GDPR erasure log (separate from audit for retention requirements)
CREATE TABLE gdpr_erasure_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    original_user_id UUID NOT NULL,
    anonymized_label VARCHAR(100) NOT NULL,
    executed_by UUID NOT NULL REFERENCES users(id),
    executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    modules_affected JSONB NOT NULL,
    confirmation_hash VARCHAR(64) NOT NULL
);

-- Password policies
CREATE TABLE password_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    min_length INT NOT NULL DEFAULT 12,
    require_uppercase BOOLEAN NOT NULL DEFAULT false,
    require_lowercase BOOLEAN NOT NULL DEFAULT false,
    require_digit BOOLEAN NOT NULL DEFAULT false,
    require_special BOOLEAN NOT NULL DEFAULT false,
    min_entropy FLOAT NOT NULL DEFAULT 50.0,
    max_age_days INT,
    prevent_reuse_count INT NOT NULL DEFAULT 5,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID REFERENCES users(id)
);

-- Password history for reuse prevention
CREATE TABLE password_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_password_history_user_id ON password_history (user_id);

-- IP access rules (allow/block lists)
CREATE TABLE ip_access_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ip_cidr CIDR NOT NULL,
    rule_type VARCHAR(10) NOT NULL CHECK (rule_type IN ('allow', 'block')),
    description TEXT,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ip_access_rules_type ON ip_access_rules (rule_type);

-- Seed default password policy
INSERT INTO password_policies (min_length, min_entropy, prevent_reuse_count) VALUES (12, 50.0, 5);

-- Platform integration configurations (one per platform per org)
CREATE TABLE integration_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    platform VARCHAR(20) NOT NULL CHECK (platform IN ('teams', 'slack')),
    is_active BOOLEAN NOT NULL DEFAULT false,
    -- Teams: app_id, app_password (encrypted via vault key reference)
    -- Slack: bot_token, signing_secret (encrypted via vault key reference)
    credentials_vault_key VARCHAR(200) NOT NULL,
    -- Platform-specific metadata (e.g., team_id for Slack, tenant_id for Teams)
    metadata JSONB NOT NULL DEFAULT '{}',
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(platform)
);

-- Channel mappings: which modules forward to which channels
CREATE TABLE integration_channel_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id UUID NOT NULL REFERENCES integration_configs(id) ON DELETE CASCADE,
    channel_id VARCHAR(200) NOT NULL,
    channel_name VARCHAR(200) NOT NULL,
    -- Module toggles (which modules forward to this channel)
    modules JSONB NOT NULL DEFAULT '[]',
    is_active BOOLEAN NOT NULL DEFAULT true,
    -- Teams: conversation reference for proactive messaging
    -- Slack: n/a (channel_id sufficient)
    platform_data JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_integration_mappings_config ON integration_channel_mappings(config_id);
CREATE INDEX idx_integration_mappings_active ON integration_channel_mappings(config_id)
    WHERE is_active = true;

-- Account links: external user <-> KMU Hub user
CREATE TABLE integration_account_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    platform VARCHAR(20) NOT NULL CHECK (platform IN ('teams', 'slack')),
    external_user_id VARCHAR(200) NOT NULL,
    kmuhub_user_id UUID NOT NULL REFERENCES users(id),
    external_display_name VARCHAR(200),
    linked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(platform, external_user_id),
    UNIQUE(platform, kmuhub_user_id)
);

CREATE INDEX idx_account_links_lookup ON integration_account_links(platform, external_user_id);

-- Delivery log: track forwarded notifications for debugging
CREATE TABLE integration_delivery_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id UUID NOT NULL,
    mapping_id UUID NOT NULL REFERENCES integration_channel_mappings(id),
    platform VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('sent', 'failed', 'rate_limited')),
    -- Platform response (message_ts for Slack, activity_id for Teams)
    platform_message_id VARCHAR(200),
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_delivery_log_notification ON integration_delivery_log(notification_id);
CREATE INDEX idx_delivery_log_cleanup ON integration_delivery_log(created_at);

-- Link tokens for account linking flow
CREATE TABLE integration_link_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    platform VARCHAR(20) NOT NULL,
    external_user_id VARCHAR(200) NOT NULL,
    token_hash VARCHAR(64) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_link_tokens_hash ON integration_link_tokens(token_hash) WHERE NOT used;

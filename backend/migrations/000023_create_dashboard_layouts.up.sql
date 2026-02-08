-- Role-based default dashboard layouts
CREATE TABLE dashboard_defaults (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role VARCHAR(50) NOT NULL UNIQUE,
    layout JSONB NOT NULL DEFAULT '[]'::jsonb,
    active_widgets JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dashboard_defaults_role ON dashboard_defaults(role);

-- Per-user dashboard layout overrides
CREATE TABLE user_dashboard_layouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    layout JSONB NOT NULL DEFAULT '[]'::jsonb,
    active_widgets JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_user_dashboard_layouts_user_id UNIQUE (user_id)
);

CREATE INDEX idx_user_dashboard_layouts_user_id ON user_dashboard_layouts(user_id);

-- Insert default layouts for standard roles
INSERT INTO dashboard_defaults (role, layout, active_widgets) VALUES
('admin', '[{"i":"deal-pipeline","x":0,"y":0,"w":6,"h":4},{"i":"activity-feed","x":6,"y":0,"w":6,"h":4},{"i":"recent-contacts","x":0,"y":4,"w":4,"h":3},{"i":"notification-summary","x":4,"y":4,"w":4,"h":3},{"i":"quick-actions","x":8,"y":4,"w":4,"h":2}]', '["deal-pipeline","activity-feed","recent-contacts","notification-summary","quick-actions"]'),
('manager', '[{"i":"deal-pipeline","x":0,"y":0,"w":6,"h":4},{"i":"unread-messages","x":6,"y":0,"w":6,"h":3},{"i":"recent-contacts","x":0,"y":4,"w":4,"h":3},{"i":"activity-feed","x":4,"y":4,"w":4,"h":3},{"i":"quick-actions","x":8,"y":4,"w":4,"h":2}]', '["deal-pipeline","unread-messages","recent-contacts","activity-feed","quick-actions"]'),
('member', '[{"i":"unread-messages","x":0,"y":0,"w":6,"h":3},{"i":"activity-feed","x":6,"y":0,"w":6,"h":4},{"i":"recent-contacts","x":0,"y":3,"w":4,"h":3},{"i":"quick-actions","x":4,"y":3,"w":4,"h":2},{"i":"notification-summary","x":8,"y":3,"w":4,"h":3}]', '["unread-messages","activity-feed","recent-contacts","quick-actions","notification-summary"]');

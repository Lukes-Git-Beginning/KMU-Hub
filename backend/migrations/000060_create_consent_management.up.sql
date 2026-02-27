CREATE TYPE consent_type AS ENUM (
  'marketing_email',
  'marketing_phone',
  'profiling',
  'newsletter',
  'data_processing',
  'data_sharing'
);

CREATE TYPE consent_legal_basis AS ENUM (
  'consent',
  'legitimate_interest',
  'contract',
  'legal_obligation'
);

CREATE TABLE consent_records (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  contact_id      UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
  consent_type    consent_type NOT NULL,
  granted         BOOLEAN NOT NULL,
  legal_basis     consent_legal_basis NOT NULL DEFAULT 'consent',
  source          VARCHAR(50),
  ip_address      INET,
  notes           TEXT,
  granted_at      TIMESTAMPTZ,
  revoked_at      TIMESTAMPTZ,
  created_by      UUID REFERENCES users(id),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_consent_contact ON consent_records (contact_id);
CREATE INDEX idx_consent_type ON consent_records (contact_id, consent_type);
CREATE INDEX idx_consent_created_at ON consent_records (created_at DESC);

CREATE TABLE gdpr_deletion_requests (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  contact_id      UUID NOT NULL REFERENCES contacts(id),
  requested_by    UUID REFERENCES users(id),
  reason          TEXT,
  status          VARCHAR(20) NOT NULL DEFAULT 'pending',
  completed_at    TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_gdpr_contact ON gdpr_deletion_requests (contact_id);
CREATE INDEX idx_gdpr_status ON gdpr_deletion_requests (status);

DROP INDEX IF EXISTS idx_gdpr_status;
DROP INDEX IF EXISTS idx_gdpr_contact;
DROP TABLE IF EXISTS gdpr_deletion_requests;

DROP INDEX IF EXISTS idx_consent_created_at;
DROP INDEX IF EXISTS idx_consent_type;
DROP INDEX IF EXISTS idx_consent_contact;
DROP TABLE IF EXISTS consent_records;

DROP TYPE IF EXISTS consent_legal_basis;
DROP TYPE IF EXISTS consent_type;

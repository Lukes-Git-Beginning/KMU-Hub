BEGIN;
SET LOCAL row_security = off;

-- rapporte: Signatur-Blob auf work_reports
ALTER TABLE work_reports
    ADD COLUMN signature_data TEXT NULL,
    ADD COLUMN signed_at TIMESTAMPTZ NULL,
    ADD COLUMN signed_by TEXT NULL;

-- vermietung: Signatur-Blob auf rentals
ALTER TABLE rentals
    ADD COLUMN signature_data TEXT NULL,
    ADD COLUMN signed_at TIMESTAMPTZ NULL,
    ADD COLUMN signed_by TEXT NULL;

-- vertraege: Signatur-Blob auf contracts (signature_provider + signed_on existieren bereits, NUR Blob fehlt)
ALTER TABLE contracts
    ADD COLUMN signature_data TEXT NULL,
    ADD COLUMN signed_at TIMESTAMPTZ NULL,
    ADD COLUMN signed_by TEXT NULL;

COMMIT;

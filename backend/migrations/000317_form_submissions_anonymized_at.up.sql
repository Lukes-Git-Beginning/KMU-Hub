-- feat-retention-worker-handler-helpdesk-formulare: FormSubmissionRetentionHandler
-- (internal/security/gdpr/retention_helpdesk_formulare.go) needs a marker to
-- tell an already-anonymized submission apart from one still due, so a second
-- retention run does not rewrite the same rows forever. form_submissions has
-- no updated_at column to bump (unlike tickets, dialer_call_sessions,
-- messages), so a dedicated column is the smallest correct fix instead of
-- widening submitted_at's meaning or adding an updated_at that nothing else
-- needs.
ALTER TABLE form_submissions
    ADD COLUMN anonymized_at TIMESTAMPTZ NULL;

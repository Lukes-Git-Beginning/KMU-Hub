-- ============================================================================
-- Intake target binding for form schemas (BACKLOG B6 "intake-form-target").
--
-- A form can feed a module record (first consumer: a Helpdesk ticket, see
-- desktop/src/renderer/src/modules/helpdesk/intake/helpdesk-ticket-target.ts).
-- `intake_target_id` names which one. It is a text identifier into the
-- shared intake registry (e.g. "helpdesk_ticket"), not a foreign key: the
-- target is a code-level registration, not a database row.
--
-- Field-level role assignment ("this field is the ticket subject") needs no
-- schema change: `fields` is already a JSONB array of FormField objects, and
-- `role` is simply a new optional key inside each element, validated in
-- Go (formulare.validIntakeRoles) against the fixed role list the FE roles
-- dropdown offers today.
-- ============================================================================

ALTER TABLE form_schemas
    ADD COLUMN intake_target_id TEXT NULL;

-- Migration 000228: AI-generated meeting summary (Wave 7C).
--
-- Persists an LLM-generated summary of a meeting's public notes plus the
-- timestamp it was produced. Both columns are nullable — meetings without a
-- generated summary are unaffected. meetings already has RLS enabled; a
-- column-only ALTER requires no new policy.

ALTER TABLE meetings
    ADD COLUMN ai_summary TEXT;

ALTER TABLE meetings
    ADD COLUMN ai_summary_at TIMESTAMPTZ;

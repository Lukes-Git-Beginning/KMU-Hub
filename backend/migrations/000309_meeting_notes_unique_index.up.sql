-- 000309 Partial unique indexes for meeting_notes matching SaveNotes' ON
-- CONFLICT target. Without these, EVERY SaveNotes call fails with "no unique
-- or exclusion constraint matching the ON CONFLICT specification" — the whole
-- notes-writing feature (public and private) has never persisted a single row.
-- Model implied by the existing code (SaveNotes' `WHERE is_private = $5`
-- predicate, GetNotes' unfiltered single-row lookup): one public note and one
-- private note per author per meeting.

CREATE UNIQUE INDEX meeting_notes_meeting_author_public_unique
    ON meeting_notes (meeting_id, author_id)
    WHERE is_private = false;

CREATE UNIQUE INDEX meeting_notes_meeting_author_private_unique
    ON meeting_notes (meeting_id, author_id)
    WHERE is_private = true;

-- Disable pgvector. Will fail if any column of type vector still exists.
-- Run the 000073 down migration first.
DROP EXTENSION IF EXISTS vector;

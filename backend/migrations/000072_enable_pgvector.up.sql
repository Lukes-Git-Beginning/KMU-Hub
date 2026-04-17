-- Enable pgvector extension for embedding storage and vector similarity search.
-- Used by 000073 for contact embeddings. Safe to enable even without consumers.
CREATE EXTENSION IF NOT EXISTS vector;

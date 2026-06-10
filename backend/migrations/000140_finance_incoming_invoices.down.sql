-- Rollback Migration 000140 — Incoming Invoices (E-Rechnung Eingang)

BEGIN;

SET LOCAL row_security = off;

DROP TABLE IF EXISTS finance_incoming_invoices;

COMMIT;

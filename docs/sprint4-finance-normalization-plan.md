# Sprint 4 — Finance Line Items Normalization Plan

> Detailplan zu ADR-0007. Standalone-Dokument für Sprint-4-Execution.
> ADR (Architektur-Entscheidung + Schema-DDL + Backfill-SQL): `docs/adr/0007-finance-line-items-normalization.md`

## ✅ STATUS: IMPLEMENTIERT (2026-06-08)

Relationaler Cutover umgesetzt. **Reale Migrations-Nummern: `000132` (Schema+RLS+Lock-Spalten) + `000133` (Backfill)** — NICHT die unten genannten 114–117 (waren Platzhalter, inzwischen von Option-B-Phase-2 verbraucht; Head war 000131). Wichtigste Abweichungen vom Plan unten (Details in der ADR-Status-Sektion):
- **Phase 1+2 (Schema + Backfill):** ✅ done, end-to-end verifiziert (up/down/up + Backfill-Idempotenz + Lock-Migration aus snapshot_data).
- **Phase 3 (Code):** ✅ invoice/quote/creditnote-Repos relational (atomare Tx, Bulk-Read). **Sauberer Cutover OHNE Dual-Write/Feature-Flag** (keine Prod-Daten) — `modules.finance_lines_relational` wurde NICHT angelegt. `line_items` JSONB bleibt synchron befüllt → **gRPC/pdf/datev/dashboard unverändert** (Proto war schon `repeated LineItem`, kein API-Bruch; kein Frontend-Change). Dashboard-Task gegenstandslos (nutzt kein `jsonb_array_elements`).
- **`tax_rate`-CHECK DACH-sicher** `>= 0 AND <= 100` (nicht DE-only).
- **Phase 4 (JSONB-Drop):** ⏭ deferred auf Sprint 5 (nach Konfidenz-Fenster).
- **Tests:** testcontainers-go statt nur Service-Unit-Tests. Coverage invoice 69.6% / quote 63.7% / creditnote 51.3%. ⚠ `//go:build integration`-gated → CI braucht `-tags=integration` (Follow-up).

## Ziel

`line_items JSONB` in `finance_invoices`, `finance_quotes`, `finance_credit_notes`
durch drei relationale Tabellen ersetzen. GoBD-Audit-Trail auf Positionsebene,
DB-Constraints, indexierbare Aggregationen, ZUGFeRD-Mapping.

## Scope-Abgrenzung

**IN:** `finance_invoice_lines`, `finance_quote_lines`, `finance_credit_note_lines`,
`locked_at`/`locked_by`-Columns (Ablösung `snapshot_data`-Lock-Hack)

**OUT:** `tax_breakdown JSONB` (bleibt als berechnetes Cache), `company_snapshot JSONB`
(echtes Snapshot-Pattern, absichtlich), `snapshot_data JSONB` für alles außer Lock
(Sprint 5 Review)

## Phasen

### Phase 0 — Vorbereitung (Sprint 3, bereits erledigt)

- [x] ADR-0007 verfasst
- [x] Test-Skeletons in `backend/internal/biz/invoice/jsonb_test.go`
- [ ] Bench-Baseline `BenchmarkJSONBArrayElementsSum` ausführen und Wert dokumentieren
- [ ] Open Questions aus ADR klären (tax_rate-Constraint-Scope, Frontend-Adapter-Entscheidung)

### Phase 1 — Schema-Migrationen (Sprint 4, Woche 1)

**Migration `000114_add_finance_line_tables`** (non-breaking, additive)

Tasks:
- [ ] `make migrate-create name=add_finance_line_tables`
- [ ] DDL für `finance_invoice_lines`, `finance_quote_lines`, `finance_credit_note_lines` (vgl. ADR Schema-Abschnitt)
- [ ] Indexes anlegen
- [ ] Migration `000117_invoice_locked_columns` parallel: `locked_at TIMESTAMPTZ`, `locked_by UUID`
- [ ] `go test ./...` grün

### Phase 2 — Backfill-Migration (Sprint 4, Woche 1)

**Migration `000115_backfill_finance_line_tables`** (idempotent)

Tasks:
- [ ] `make migrate-create name=backfill_finance_line_tables`
- [ ] SQL für idempotenten Backfill (vgl. ADR Backfill-Strategie)
- [ ] Down-Migration: `DELETE FROM finance_invoice_lines WHERE invoice_id IN (SELECT id FROM finance_invoices)`
- [ ] Auf Testdaten-Dump testen: Backfill zweimal ausführen → identische Row-Counts

### Phase 3 — Code-Umstellung (Sprint 4, Woche 2)

**Backend (6 Pakete):**

- [ ] `backend/internal/biz/invoice/postgres_repository.go`
  - `Create`: INSERT in `finance_invoice_lines` statt `line_items`-JSONB
  - `GetByID`, `List`: JOIN auf `finance_invoice_lines`, in `[]LineItem` assemblen
  - `Update`: DELETE+INSERT für Lines (atomarer Ersatz im selben Tx)
  - Parallel-Write (Feature-Flag `modules.finance_lines_relational`): JSONB weiterhin schreiben für 2 Wochen

- [ ] `backend/internal/biz/invoice/service_gobd.go`
  - `LockInvoice`: auf `locked_at`/`locked_by`-Columns umstellen (Sprint-4-TODO Zeile 105)
  - `isInvoiceLocked`: `locked_at IS NOT NULL` statt `snapshot_data`-Parse

- [ ] `backend/internal/biz/quote/postgres_repository.go` — analog Invoice

- [ ] `backend/internal/biz/creditnote/postgres_repository.go` — analog Invoice

- [ ] `backend/internal/biz/dashboard/postgres_repository.go`
  - SUM-Queries auf `finance_invoice_lines.line_total` umstellen
  - `jsonb_array_elements`-Aufrufe entfernen

- [ ] `backend/internal/biz/pdf/generator.go`
  - `[]LineItem` kommt jetzt aus Service (schon typisiert), kein `json.Unmarshal` mehr

- [ ] `backend/internal/biz/datev/exporter.go`
  - DATEV-Positionen aus `finance_invoice_lines`-Join statt JSONB-Unmarshal

**Frontend (5 Files):**

- [ ] `desktop/src/renderer/src/types/finance-types.ts`
  - `LineItem`-Interface bleibt identisch (keine Breaking-Change für Frontend)
  - API-Response liefert `lines: LineItem[]` (neuer Key) + `line_items` deprecated (Adapter)

- [ ] `desktop/src/renderer/src/api/hooks/useFinance.ts`
  - `line_items` → `lines` Key-Migration (Feature-Flag gesteuert oder direkt wenn Sprint-Fenster)

- [ ] `desktop/src/renderer/src/components/InvoiceFormDialog.tsx`
- [ ] `desktop/src/renderer/src/components/InvoiceDetailPanel.tsx`
- [ ] `desktop/src/renderer/src/components/QuoteFormDialog.tsx`

### Phase 4 — JSONB-Drop (Sprint 4, Woche 3 / Hardening)

Erst nach:
- [ ] Test-Coverage Finance ≥ 60%
- [ ] Parallel-Write-Phase > 2 Wochen produktiv (Pilot-1-Daten bestätigt)
- [ ] Bench-Post-Migration dokumentiert

**Migration `000116_drop_finance_line_items_jsonb`:**
- [ ] `ALTER TABLE finance_invoices DROP COLUMN line_items`
- [ ] `ALTER TABLE finance_quotes DROP COLUMN line_items`
- [ ] `ALTER TABLE finance_credit_notes DROP COLUMN line_items`
- [ ] Down-Migration: Re-Adds Columns + Backfill-Reverse
- [ ] Feature-Flag `modules.finance_lines_relational` entfernen (kein Parallelbetrieb mehr)

## Test-Coverage-Ziele (Sprint 4)

| Test | Datei | Status |
|------|-------|--------|
| `TestLineItemsJSONBRoundtrip` | `invoice/jsonb_test.go` | Skeleton ✓ |
| `TestTaxBreakdownRoundtrip` | `invoice/jsonb_test.go` | Skeleton ✓ |
| `TestCorruptLineItemsHandling` | `invoice/jsonb_test.go` | Skeleton ✓ |
| `TestBackfillIdempotency` | `invoice/migration_test.go` | TODO Sprint 4 |
| `TestLineItemsRelationalRoundtrip` | `invoice/postgres_repository_test.go` | TODO Sprint 4 |
| `TestTenantIsolationInvoiceLines` | `invoice/tenant_test.go` | TODO Sprint 4 |
| `BenchmarkJSONBArrayElementsSum` | `invoice/jsonb_test.go` | TODO Sprint 3 Baseline |

## Risiken & Mitigationen

| Risiko | Wahrscheinlichkeit | Mitigation |
|--------|-------------------|------------|
| Backfill-Timeout bei großen Datenmengen | Mittel | Batch-Backfill mit `LIMIT 1000`, Bench-Baseline vor Phase 2 |
| API-Breaking-Change bricht Pilot-1-Clients | Hoch | Adapter-Layer im HTTP-Handler, parallele Keys (`lines` + `line_items`), 2-Wochen-Deprecated-Fenster |
| Decimal-Präzisionsverlust bei Roundtrip | Niedrig | `TestLineItemsJSONBRoundtrip` als Regression-Guard vor Umstieg |
| `snapshot_data`-Migration vergessen | Mittel | Explicit Phase-1-Task, `service_gobd.go:105`-TODO als Tracker |
| `tax_rate`-Constraint blockiert EU-Mandanten | Niedrig (Sprint 5+) | Constraint auf `>= 0 AND <= 100` statt hardcoded DE-Sätze — Open Question ADR |

## Abhängigkeiten

- Sprint 3: Open Questions klären + Bench-Baseline
- Sprint 4: Pilot-1 läuft parallel (Parallel-Write-Fenster für Pilot-Daten)
- Sprint 5: Rigorosum Runde 3 — Finance Coverage ≥ 60% als Gate

## Referenzen

- `docs/adr/0007-finance-line-items-normalization.md` — Vollständiger ADR
- `docs/ROADMAP.md` Sprint 4 — Finance-Normalisierung
- `backend/migrations/000045_create_finance_tables.up.sql`
- `backend/internal/biz/invoice/service_gobd.go:105` — TODO Sprint 4

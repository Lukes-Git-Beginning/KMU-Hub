# ADR-0007: Normalize finance line items from JSONB to relational tables

## Status

Proposed (2026-05-08)

## Context

### Problem

Migration 000045 (`CREATE TABLE finance_invoices`) speichert Rechnungspositionen als
`line_items JSONB NOT NULL DEFAULT '[]'`. Dasselbe gilt für `finance_quotes` und
`finance_credit_notes`. Das ist das typische "schnell ausgeliefert, aber technisch
verschuldet"-Muster, das vor Launch korrigiert werden muss.

Konkrete Risiken:

**GoBD-Compliance (§§ 146, 147 AO + GoBDv2):**
GoBD schreibt einen unveränderlichen, lückenlosen Buchungsbeleg-Audit-Trail auf
Positionsebene vor. Mit JSONB gibt es keinen nativen `updated_at`-Timestamp pro
Zeile — ein nachträglich veränderter `line_items`-Blob hinterlässt keinen Fingerabdruck.
Das Prüfer-Nachweis-Problem ist real: "Zeigen Sie mir, wann Position 3 dieser
Rechnung geändert wurde" ist mit der aktuellen Struktur nicht beantwortbar.

**ZUGFeRD-/XRechnung-Compliance:**
ZUGFeRD-2.3 (EN 16931) und XRechnung verlangen maschinenlesbare XML-Positionen mit
normiertem Mapping (BT-126 bis BT-146). Das Mapping aus einem beliebigen JSONB-Blob
ist fragil und fehleranfällig. Aus relationalen Zeilen ist das Mapping deterministisch.

**Query-Performance & Aggregierbarkeit:**
`SELECT SUM(...) FROM finance_invoices` erfordert derzeit `jsonb_array_elements`,
was einen Full-Table-Scan ohne Index erzwingt. Dashboard-Queries (Top-Produkte,
Umsatz nach Artikel, DATEV-Export) skalieren nicht.

**Kein DB-Level-Constraint:**
`quantity > 0`, `tax_rate IN (0, 7, 19)`, `unit_price >= 0` können auf JSONB
nicht als CHECK-Constraints enforced werden. Korrupte Daten (z.B. negative Quantity
aus einem UI-Bug) landen lautlos in der DB.

**Betroffene JSONB-Spalten (Migration 000045):**

| Tabelle | Spalte | Zeile im SQL |
|---------|--------|--------------|
| `finance_invoices` | `line_items JSONB NOT NULL DEFAULT '[]'` | Zeile 83 |
| `finance_invoices` | `tax_breakdown JSONB` | Zeile 84 |
| `finance_invoices` | `snapshot_data JSONB` | Zeile 92 (zweckentfremdet für GoBD-Lock) |
| `finance_invoices` | `company_snapshot JSONB` | Zeile 81 |
| `finance_quotes` | `line_items JSONB NOT NULL DEFAULT '[]'` | Zeile 53 |
| `finance_quotes` | `tax_breakdown JSONB` | Zeile 54 |
| `finance_credit_notes` | `line_items JSONB NOT NULL DEFAULT '[]'` | Zeile 113 |
| `finance_credit_notes` | `tax_breakdown JSONB` | Zeile 114 |

**Betroffene Code-Pfade:**

Backend:
- `backend/internal/biz/invoice/service.go` — JSONB-Marshal/-Unmarshal in Create/Update
- `backend/internal/biz/invoice/postgres_repository.go` — `line_items` und `tax_breakdown` als `json.RawMessage`
- `backend/internal/biz/invoice/service_gobd.go` — `snapshot_data` als temporärer Lock-Träger (TODO Sprint 4 Zeile 105)
- `backend/internal/biz/quote/service.go` und `postgres_repository.go`
- `backend/internal/biz/creditnote/service.go` und `postgres_repository.go`
- `backend/internal/biz/dashboard/postgres_repository.go` — `jsonb_array_elements`-Aggregations
- `backend/internal/biz/pdf/generator.go` — PDF-Rendering aus `[]LineItem`
- `backend/internal/biz/datev/exporter.go` — DATEV-Export, marshallt JSONB

Frontend:
- `desktop/src/renderer/src/types/finance-types.ts` — `LineItem`-Interface
- `desktop/src/renderer/src/api/hooks/useFinance.ts` — API-Calls mit `line_items`-Array
- `desktop/src/renderer/src/components/InvoiceFormDialog.tsx`
- `desktop/src/renderer/src/components/InvoiceDetailPanel.tsx`
- `desktop/src/renderer/src/components/QuoteFormDialog.tsx`

### Aktuelles Go-Modell (zur Referenz)

```go
// models/finance.go
type LineItem struct {
    ID          string          `json:"id"`
    Position    int             `json:"position"`
    Description string          `json:"description"`
    Quantity    decimal.Decimal `json:"quantity"`
    UnitPrice   decimal.Decimal `json:"unit_price"`
    TaxRate     decimal.Decimal `json:"tax_rate"`
    LineTotal   decimal.Decimal `json:"line_total"`
}
```

Das `id`-Feld ist aktuell ein client-seitiger UUID-String, kein DB-Primary-Key mit
`REFERENCES`. Das ist die Kernschwachstelle.

## Decision

**Vor Launch normalisieren.** Sprint 4 ersetzt `line_items JSONB` durch drei neue
relationale Tabellen. Die JSONB-Spalte bleibt während der Backfill-Phase erhalten
und wird erst in Phase 4 (nach vollständiger Migration) gedroppt.

Zusätzlich: `snapshot_data JSONB` wird auf dedizierte `locked_at`/`locked_by`-Spalten
migriert (separates Sub-Task, koordiniert mit dieser Migration).

## Schema (Ziel-DDL)

```sql
-- Positionen für Rechnungen
CREATE TABLE finance_invoice_lines (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id   UUID NOT NULL REFERENCES finance_invoices(id) ON DELETE CASCADE,
    tenant_id    UUID NOT NULL,  -- Denormalisiert für Tenant-Isolation-Queries ohne JOIN
    position     INT  NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    quantity     NUMERIC(15,4) NOT NULL,
    unit_price   NUMERIC(15,4) NOT NULL,
    tax_rate     NUMERIC(5,2)  NOT NULL DEFAULT 0,
    line_total   NUMERIC(15,4) NOT NULL,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_invoice_lines_quantity  CHECK (quantity > 0),
    CONSTRAINT chk_invoice_lines_unit_price CHECK (unit_price >= 0),
    CONSTRAINT chk_invoice_lines_tax_rate   CHECK (tax_rate IN (0, 7, 19)),
    CONSTRAINT chk_invoice_lines_position   CHECK (position >= 1)
);
CREATE INDEX idx_invoice_lines_invoice ON finance_invoice_lines(invoice_id);
CREATE INDEX idx_invoice_lines_tenant  ON finance_invoice_lines(tenant_id);

-- Positionen für Angebote
CREATE TABLE finance_quote_lines (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quote_id     UUID NOT NULL REFERENCES finance_quotes(id) ON DELETE CASCADE,
    tenant_id    UUID NOT NULL,
    position     INT  NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    quantity     NUMERIC(15,4) NOT NULL,
    unit_price   NUMERIC(15,4) NOT NULL,
    tax_rate     NUMERIC(5,2)  NOT NULL DEFAULT 0,
    line_total   NUMERIC(15,4) NOT NULL,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_quote_lines_quantity   CHECK (quantity > 0),
    CONSTRAINT chk_quote_lines_unit_price  CHECK (unit_price >= 0),
    CONSTRAINT chk_quote_lines_tax_rate    CHECK (tax_rate IN (0, 7, 19)),
    CONSTRAINT chk_quote_lines_position    CHECK (position >= 1)
);
CREATE INDEX idx_quote_lines_quote  ON finance_quote_lines(quote_id);
CREATE INDEX idx_quote_lines_tenant ON finance_quote_lines(tenant_id);

-- Positionen für Gutschriften
CREATE TABLE finance_credit_note_lines (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    credit_note_id   UUID NOT NULL REFERENCES finance_credit_notes(id) ON DELETE CASCADE,
    tenant_id        UUID NOT NULL,
    position         INT  NOT NULL,
    description      TEXT NOT NULL DEFAULT '',
    quantity         NUMERIC(15,4) NOT NULL,
    unit_price       NUMERIC(15,4) NOT NULL,
    tax_rate         NUMERIC(5,2)  NOT NULL DEFAULT 0,
    line_total       NUMERIC(15,4) NOT NULL,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_credit_note_lines_quantity   CHECK (quantity > 0),
    CONSTRAINT chk_credit_note_lines_unit_price  CHECK (unit_price >= 0),
    CONSTRAINT chk_credit_note_lines_tax_rate    CHECK (tax_rate IN (0, 7, 19)),
    CONSTRAINT chk_credit_note_lines_position    CHECK (position >= 1)
);
CREATE INDEX idx_credit_note_lines_credit_note ON finance_credit_note_lines(credit_note_id);
CREATE INDEX idx_credit_note_lines_tenant      ON finance_credit_note_lines(tenant_id);
```

**Hinweis zu `tax_rate CHECK`:** Aktuell nur DE-Sätze (0/7/19%). Wenn EU-Mandantenfähigkeit
in Phase E kommt, muss der Constraint auf `>= 0 AND <= 100` relaxt werden. Das ist
bewusste Short-Term-Rigidität für jetzt.

**`tax_breakdown`-Spalten** bleiben als berechnetes Cache-JSONB erhalten — sie sind
nicht normalisierungsrelevant (abgeleitete Daten, keine Source-of-Truth). Sie können
bei Bedarf in Sprint 5+ durch eine View ersetzt werden.

**`company_snapshot`** bleibt als JSONB — das ist ein echtes Snapshot-Pattern
(einmalig eingefroren, nie geupdated), kein Normalisierungsproblem.

## Backfill-Strategie

```sql
-- Idempotenter Backfill für finance_invoice_lines
-- Löscht vorhandene Zeilen für die Invoice und fügt neu ein (safe für Re-Run)
INSERT INTO finance_invoice_lines (
    id, invoice_id, tenant_id, position,
    description, quantity, unit_price, tax_rate, line_total,
    created_at, updated_at
)
SELECT
    COALESCE((elem->>'id')::uuid, gen_random_uuid()),
    fi.id,
    fi.tenant_id,
    (elem->>'position')::int,
    COALESCE(elem->>'description', ''),
    (elem->>'quantity')::numeric,
    (elem->>'unit_price')::numeric,
    (elem->>'tax_rate')::numeric,
    (elem->>'line_total')::numeric,
    fi.created_at,
    fi.updated_at
FROM finance_invoices fi,
     jsonb_array_elements(fi.line_items) WITH ORDINALITY AS t(elem, ord)
WHERE fi.line_items != '[]'::jsonb
  AND fi.line_items IS NOT NULL
ON CONFLICT (id) DO NOTHING;
```

Analoges SQL für `finance_quote_lines` und `finance_credit_note_lines`.

Die Backfill-Migration läuft als eigenständige Migration (Phase 2) nach der
Schema-Migration (Phase 1). Sie ist idempotent durch `ON CONFLICT (id) DO NOTHING`.
Bei > 10.000 Dokumenten: Backfill in Batches von 1.000 mit `LIMIT`/`OFFSET`.

## Migration-Plan (Sprint 4)

| Phase | Migration | Inhalt | Rollback-Risiko |
|-------|-----------|--------|-----------------|
| Phase 1 | `000114_add_finance_line_tables.up.sql` | Neue Tabellen anlegen (non-breaking, additive) | Gering — nur DROP TABLE |
| Phase 2 | `000115_backfill_finance_line_tables.up.sql` | Idempotenter Backfill aus JSONB | Gering — `ON CONFLICT DO NOTHING` |
| Phase 3 | Code-Umstellung | Repos, Services, gRPC-Handler, Frontend auf neue Tabellen. JSONB-Felder bleiben schema-seitig vorhanden (parallel-write via Service-Layer für 2 Wochen) | Mittel — erfordert Feature-Flag |
| Phase 4 | `000116_drop_finance_line_items_jsonb.up.sql` | `ALTER TABLE ... DROP COLUMN line_items` | Hoch — Down-Migration muss Backfill-Richtung umkehren |
| Parallel | `000117_invoice_locked_columns.up.sql` | `ALTER TABLE finance_invoices ADD COLUMN locked_at TIMESTAMPTZ, ADD COLUMN locked_by UUID` — löst `snapshot_data`-Zweckentfremdung | Gering |

**Wichtig für Phase 3:** Während der Parallel-Write-Phase schreiben Service-Layer
`line_items` JSONB **und** `finance_invoice_lines`-Rows gleichzeitig. Das ist ein
2-Wochen-Backward-Compat-Fenster für den Fall eines Hot-Rollbacks auf Phase-2-Code.
Ein Feature-Flag `modules.finance_lines_relational` (default OFF, in Phase 3 ON)
steuert, ob Reader-Pfade aus JSONB oder aus der neuen Tabelle lesen.

## Consequences

### Positiv

- **Zeile-Level-Audit-Trail:** `created_at`/`updated_at` pro Position — GoBD-konform
- **Fremdschlüssel-Integrität:** `ON DELETE CASCADE` verhindert verwaiste Positionen
- **DB-CHECK-Constraints:** `quantity > 0`, `tax_rate IN (0, 7, 19)`, `unit_price >= 0` — DB-enforced
- **SUM-Queries ohne `jsonb_array_elements`:** Dashboard, DATEV-Export, Umsatz-Reports werden indexierbar
- **ZUGFeRD/XRechnung:** deterministisches XML-Mapping aus relationalen Zeilen (BT-126 bis BT-146)
- **GoBD-Lock-Cleanup:** `snapshot_data` verliert seinen Zweck — dedizierte `locked_at`/`locked_by`-Spalten machen den Code verständlicher (vgl. `service_gobd.go:105`)
- **Testbarkeit:** Einzel-Assertions auf `finance_invoice_lines`-Rows vs. JSON-Roundtrip

### Negativ / Cost

- **6 Backend-Pakete** müssen angepasst werden (invoice, quote, creditnote, dashboard, pdf, datev)
- **5 Frontend-Files** (finance-types.ts, useFinance.ts, InvoiceFormDialog, InvoiceDetailPanel, QuoteFormDialog)
- **API-Änderung:** gRPC-Proto `InvoiceResponse` muss `repeated LineItem lines` als Sub-Message liefern. Bestehende Clients erwarten `line_items` als JSON-String. Adapter-Schicht nötig oder Breaking-Change mit Version-Bump.
- **Backfill-Performance:** Unklar bei größeren Datensätzen (Bench-Baseline vor Migration nötig)
- **Test-Coverage-Ausbau nötig:** Finance ist aktuell ~30% — vor Phase 4 mindestens 60% kritische Pfade (ADR-Regel: Test-Coverage ≥ 60% für Finance, Security, Auth)
- **Paralleles Schreiben in Phase 3** erhöht Service-Komplexität temporär — klar abzugrenzen und zu befristen

## Open Questions

1. **Frontend-Adapter-Schicht:** Soll der gRPC-Handler `line_items` als eingebettetes
   Array weiterhin zurückgeben (Backward-Compat für bestehende API-Clients), oder
   Breaking-Change mit neuem `lines`-Feld und Version-Bump auf `/api/v2/`? Empfehlung:
   Adapter im HTTP-Handler für die Parallel-Write-Phase, dann Breaking-Change in Sprint 5.

2. **Backfill-Bench-Baseline:** Wie lange dauert der Backfill bei 10.000 Invoices mit
   je 10 Positionen? Muss vor Phase 2 gemessen werden, um Downtime-Fenster zu kalkulieren.
   Ziel: < 60 Sekunden bei Pilot-Datenmenge.

3. **`tax_rate`-CHECK-Constraint Scope:** Nur DE-Sätze (0/7/19) oder generischer
   `>= 0 AND <= 100`? Bei EU-Mandantenfähigkeit (Phase E) muss der Constraint offen
   sein. Entscheidung vor Migration 000114 notwendig.

4. **`snapshot_data`-Lock-Migration Timing:** Soll `000117_invoice_locked_columns`
   vor oder nach den Line-Items-Migrationen laufen? Empfehlung: vor Phase 3 (Phase 1
   parallel), da `service_gobd.go` dann die neuen Spalten nutzen kann statt des
   JSONB-Hacks.

5. **`company_snapshot` JSONB:** Bleibt JSONB (Snapshot-Pattern, absichtlich). Kein
   Normalisierungsbedarf. Dokumentiert, damit der Begriff "JSONB normalisieren" nicht
   irrtümlich auch dieses Feld umfasst.

## Test-Coverage-Ausbau (Sprint 4 Vorbereitung)

Die folgenden Tests sind in `backend/internal/biz/invoice/jsonb_test.go` als
Skeletons vorbereitet (Sprint 3) und werden in Sprint 4 implementiert:

- `TestLineItemsJSONBRoundtrip` — Marshal/Unmarshal-Roundtrip mit `decimal.Decimal`-Gleichheit
- `TestTaxBreakdownRoundtrip` — `tax_by_rate`-Map Konsistenz unter Serialisierung
- `TestCorruptLineItemsHandling` — `unmarshalLineItems` mit ungültigem JSON → kein Panic
- `BenchmarkJSONBArrayElementsSum` — Baseline-Bench für `jsonb_array_elements`-SUM
  vor der Migration (messbarer Vergleichswert post-Migration)

Außerdem Sprint-4-Tests nach Implementierung:
- `TestBackfillIdempotency` — Backfill-Migration zweimal ausführen → identische Zeilen
- `TestLineItemsRelationalRoundtrip` — Create Invoice → INSERT Lines → Read Lines → verify Decimal equality
- `TestTenantIsolationInvoiceLines` — Tenant A kann Lines von Tenant B nicht lesen

## References

- Rigorosum Runde 2 R2-P1.12 (functional-seahorse, 2026-04-18)
- Migration `backend/migrations/000045_create_finance_tables.up.sql` — Zeilen 53, 83, 113 (JSONB)
- `backend/internal/biz/invoice/service_gobd.go` Zeile 105 — `TODO Sprint 4: replace snapshot_data lock`
- `backend/internal/models/finance.go` — `LineItem`, `TaxBreakdown`, `Invoice`, `Quote`, `CreditNote`
- GoBD-Grundsätze ordnungsmäßiger Buchführung (BMF, 2019), §§ 146, 147 AO
- ZUGFeRD 2.3 / EN 16931 — BT-126 bis BT-146 (Positionsdaten)
- `docs/ROADMAP.md` Sprint 4 — Finance-Normalisierung als dedizierter Sprint-Schwerpunkt

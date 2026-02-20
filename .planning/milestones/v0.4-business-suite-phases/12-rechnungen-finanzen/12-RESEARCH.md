# Phase 12: Rechnungen & Finanzen - Research

**Researched:** 2026-02-18
**Domain:** GoBD-compliant invoicing, German tax calculation, PDF generation, DATEV export
**Confidence:** HIGH (domain well-understood, codebase patterns established, legal requirements verified from official sources)

## Summary

Phase 12 adds GoBD-compliant quote/invoice management, German MwSt calculation, PDF generation, 3-level dunning, payment tracking with a dashboard, and DATEV Buchungsstapel CSV export. The backend will follow the established microservice pattern as a new "biz" service (gRPC on :50058, health on :9098), while the frontend replaces the existing Zustand mock store in the FinanzenPage with TanStack Query hooks backed by real API endpoints.

The legal domain is well-defined: GoBD immutability requirements are clear (no modification after "sent" status), German Pflichtangaben for invoices are codified in UStG section 14, MwSt rates are straightforward (19%/7%/0%), Kleinunternehmerregelung exempts VAT display with a mandatory note, and Reverse Charge requires specific invoice wording. DATEV Buchungsstapel export follows the EXTF format specification with a fixed header line and semicolon-delimited booking records.

The critical challenge is GoBD compliance: once an invoice transitions from "draft" to "sent", the document data must become immutable. This means the data model needs to snapshot all line items, tax calculations, and customer details at send time into a frozen JSONB column, and the sequential invoice number must be gap-free. Credit notes (Gutschriften) must reference the original invoice and use their own number sequence.

**Primary recommendation:** Implement the biz service following the exact same pattern as the document/email services (separate binary, gRPC proto, RouteRegistrar). Use maroto/v2 for PDF generation (pure Go, no external deps). Store immutable invoice snapshots as JSONB for GoBD compliance. Implement DATEV export as server-side CSV generation with the EXTF header format.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Document Design & PDF
- Clean minimal PDF template style (Stripe/Lexoffice aesthetic: white background, thin lines, modern sans-serif)
- Company branding: logo in header + accent color from company settings on headings/lines
- Simple line item table: Position, Bezeichnung, Menge, Einzelpreis, Gesamtpreis
- No grouped sections or multi-line descriptions per line item in v1

#### Deal-to-Invoice Flow
- Both paths supported: standalone invoice creation AND deal -> quote -> invoice conversion
- Quotes auto-expire with configurable default validity period (e.g., 30 days), status changes to "Abgelaufen"
- Deal value auto-syncs when quote is created or modified (single source of truth for pipeline revenue)

#### Dunning Behavior
- Semi-automatic: system detects overdue invoices and creates draft Mahnungen, user reviews and sends manually
- Admin-configurable intervals between dunning levels (no hardcoded defaults)
- Standard 3-level German dunning:
  - Level 1: Zahlungserinnerung (friendly tone)
  - Level 2: 1. Mahnung (formal tone)
  - Level 3: 2. Mahnung / Letzte Mahnung (urgent, threatens Inkasso)
- Configurable Mahngebuehren per level (e.g., 0/5/10 EUR) + Verzugszinsen per BGB section 288 (5% above Basiszinssatz)

#### Finance Dashboard
- Combined revenue + pipeline + status metrics:
  - Revenue: total invoiced, total paid, total outstanding, overdue amount
  - Pipeline: quotes pending, conversion rate (quote->invoice), average deal size, revenue forecast
  - Status: draft/sent/overdue/paid counts as breakdown
- Custom date range picker (from/to) plus predefined shortcuts (this month, quarter, year, etc.)
- Subtle badge indicators on status cards for actionable items (overdue invoices, expiring quotes, pending Mahnungen) -- no separate action section

### Claude's Discretion
- Pflichtangaben layout on PDF (standard German footer block vs split layout)
- Quote-to-invoice conversion mechanics (copy-and-link vs status-transition)
- Deal value sync direction (quote -> deal only, or bidirectional)
- Revenue chart visualization style (bar vs area/line)
- Exact spacing, typography, and color application on PDFs
- DATEV export field mapping details

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| FIN-01 | User can create quotes (Angebote) with line items, tax calculation, and PDF generation | Proto RPC design, maroto/v2 for PDF, line item model with per-item tax rate, quote status lifecycle |
| FIN-02 | User can create invoices (Rechnungen) compliant with GoBD (immutable once sent, sequential numbering, all Pflichtangaben) | JSONB snapshot immutability pattern, gap-free sequence via SELECT FOR UPDATE, Pflichtangaben field requirements from UStG section 14 |
| FIN-03 | System calculates MwSt/USt correctly (19% standard, 7% reduced, 0% Reverse Charge for EU B2B, Kleinunternehmerregelung) | Tax calculation service with rate enum, Reverse Charge detection via buyer USt-IdNr, Kleinunternehmer flag on company settings |
| FIN-04 | User can track payment status per invoice (draft, sent, overdue, paid, cancelled) | Status state machine, payment records table, dashboard aggregation queries |
| FIN-05 | User can convert a CRM deal to a quote and then to an invoice in a seamless flow | Biz service reads deal data via CRM gRPC, copy-and-link pattern for conversion, deal value sync via CRM UpdateDeal RPC |
| FIN-06 | User can export Buchungsstapel in DATEV-compatible CSV format for Steuerberater | EXTF header format, SKR03 account mapping, semicolon-delimited CSV with UTF-8 encoding |
| FIN-07 | User can create credit notes (Gutschriften) referencing original invoices | Credit note as document type with mandatory original_invoice_id reference, own number sequence (GS-prefix) |
</phase_requirements>

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/johnfercher/maroto/v2 | v2.3+ | PDF generation (quotes, invoices, credit notes, dunning letters) | Pure Go, Bootstrap-inspired layout, no external deps (no Chromium/wkhtmltopdf), actively maintained, supports custom fonts |
| github.com/shopspring/decimal | v1.4.0 | Precise decimal arithmetic for monetary values | Already in go.mod, used by deal.Value, prevents floating-point rounding errors |
| github.com/jackc/pgx/v5 | v5.8.0 | PostgreSQL driver, advisory locks, LISTEN/NOTIFY | Already in go.mod, established project pattern |
| google.golang.org/grpc | v1.78.0 | gRPC service communication | Already in go.mod, established project pattern |
| google.golang.org/protobuf | v1.36.11 | Protobuf code generation | Already in go.mod |
| github.com/google/uuid | v1.6.0 | UUID generation for all entities | Already in go.mod |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| encoding/csv (stdlib) | Go 1.25 | DATEV Buchungsstapel CSV export | Server-side CSV generation with semicolon delimiter |
| golang.org/x/text/encoding/charmap | (stdlib) | Windows-1252 encoding for DATEV CSV | DATEV expects Windows-1252 or UTF-8 with BOM |
| time (stdlib) | Go 1.25 | Date arithmetic for due dates, overdue detection, Verzugszinsen calculation | All date-based business logic |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| maroto/v2 | chromedp/wkhtmltopdf | HTML->PDF gives more styling control but requires external Chromium/wkhtmltopdf binary; maroto is pure Go with zero external deps, matching the single-binary deployment model |
| maroto/v2 | jung-kurt/gofpdf | gofpdf is archived since 2021; maroto/v2 wraps it with a better API and is actively maintained |
| maroto/v2 | unidoc/unipdf | Commercial license required, too expensive for this project |
| shopspring/decimal | float64 | Float64 causes rounding errors on financial calculations (e.g., 19% of 123.45 EUR); decimal is already in the project |

**Installation:**
```bash
cd backend
go get github.com/johnfercher/maroto/v2@latest
```

## Architecture Patterns

### Recommended Backend Structure

```
backend/
├── cmd/biz/                      # Biz service binary (new)
│   └── main.go                   # gRPC :50058, health :9098
├── proto/biz/v1/                 # Protobuf definitions (new)
│   └── biz.proto                 # FinanceService with ~35-40 RPCs
├── internal/biz/                 # Business logic packages (new)
│   ├── quote/                    # Quote CRUD, expiry, PDF
│   │   ├── service.go
│   │   ├── repository.go
│   │   ├── postgres_repository.go
│   │   └── errors.go
│   ├── invoice/                  # Invoice CRUD, immutability, numbering, PDF
│   │   ├── service.go
│   │   ├── repository.go
│   │   ├── postgres_repository.go
│   │   └── errors.go
│   ├── creditnote/               # Credit note CRUD, reference validation
│   │   ├── service.go
│   │   ├── repository.go
│   │   └── postgres_repository.go
│   ├── payment/                  # Payment recording, status transitions
│   │   ├── service.go
│   │   ├── repository.go
│   │   └── postgres_repository.go
│   ├── dunning/                  # 3-level Mahnwesen, overdue detection
│   │   ├── service.go
│   │   ├── repository.go
│   │   └── postgres_repository.go
│   ├── tax/                      # MwSt calculation engine
│   │   ├── calculator.go         # Pure function: inputs -> tax breakdown
│   │   └── calculator_test.go
│   ├── pdf/                      # PDF generation for all document types
│   │   ├── generator.go
│   │   ├── templates.go          # Layout definitions
│   │   └── fonts.go              # Custom font embedding
│   ├── datev/                    # DATEV Buchungsstapel CSV export
│   │   ├── exporter.go
│   │   ├── mapping.go            # SKR03 account mapping
│   │   └── exporter_test.go
│   └── dashboard/                # Aggregation queries for finance dashboard
│       ├── service.go
│       └── repository.go
├── internal/gateway/
│   └── route_biz.go              # BizRoutes RouteRegistrar (new)
├── internal/models/
│   └── finance.go                # Finance domain models (new)
├── migrations/
│   ├── 000045_create_finance_tables.up.sql    # Core tables
│   └── 000045_create_finance_tables.down.sql
└── tools/
    └── biz_deps.go               # Retain maroto in go.mod
```

### Recommended Frontend Structure

```
desktop/src/renderer/src/
├── modules/finanzen/             # Already exists (6 files from design)
│   ├── FinanzenPage.tsx          # Rewrite: replace Zustand mock with TanStack Query
│   ├── InvoiceFormDialog.tsx     # Rewrite: EUR/DE locale, German MwSt rates
│   ├── InvoiceDetailPanel.tsx    # Rewrite: real data, PDF download
│   ├── PaymentRecordDialog.tsx   # Rewrite: wire to API
│   ├── ExportDialog.tsx          # Rewrite: DATEV export via API
│   ├── ExpenseFormDialog.tsx     # Remove or stub (expenses not in Phase 12 scope)
│   ├── QuoteFormDialog.tsx       # New: separate form for Angebote
│   ├── CreditNoteDialog.tsx      # New: Gutschrift creation
│   ├── DunningPanel.tsx          # New: Mahnwesen management
│   └── FinanceDashboard.tsx      # New: metrics + charts
├── api/hooks/
│   └── useFinance.ts             # New: TanStack Query hooks for all finance endpoints
├── stores/
│   └── finance.ts                # Rewrite: remove mock data, use as local UI state only
└── types/
    └── finance-types.ts          # New: TypeScript types matching proto messages
```

### Pattern 1: GoBD Immutability via JSONB Snapshot

**What:** When an invoice transitions from `draft` to `sent`, a complete snapshot of the document (line items, tax calculations, customer data, company data) is frozen into a `snapshot_data JSONB` column. After this point, the row becomes immutable (enforced by DB trigger or service-layer guard).

**When to use:** Every time an invoice or credit note is finalized/sent.

**Example:**
```go
// Service layer immutability guard
func (s *Service) SendInvoice(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
    inv, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return err
    }
    if inv.Status != StatusDraft {
        return ErrInvoiceNotDraft
    }

    // Build immutable snapshot
    snapshot := InvoiceSnapshot{
        LineItems:    inv.LineItems,
        TaxBreakdown: s.taxCalc.Calculate(inv.LineItems, inv.TaxMode),
        Customer:     inv.CustomerSnapshot, // frozen copy of customer data
        Company:      inv.CompanySnapshot,   // frozen copy of company data
        Totals:       s.taxCalc.Totals(inv.LineItems, inv.TaxMode),
    }

    snapshotJSON, err := json.Marshal(snapshot)
    if err != nil {
        return fmt.Errorf("marshal snapshot: %w", err)
    }

    // Assign sequential number (gap-free)
    nextNum, err := s.repo.NextInvoiceNumber(ctx) // SELECT ... FOR UPDATE
    if err != nil {
        return err
    }

    return s.repo.SendInvoice(ctx, id, nextNum, snapshotJSON, time.Now())
}
```

### Pattern 2: Gap-Free Sequential Numbering

**What:** Invoice numbers must be sequential without gaps per GoBD. Use a dedicated `number_sequences` table with row-level locking.

**When to use:** Every time an invoice or credit note is finalized.

**Example:**
```sql
-- Number sequence table
CREATE TABLE finance_number_sequences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL, -- future multi-tenancy
    document_type VARCHAR(20) NOT NULL, -- 'invoice', 'credit_note', 'quote'
    prefix VARCHAR(20) NOT NULL,       -- 'RE', 'GS', 'AN'
    current_number INTEGER NOT NULL DEFAULT 0,
    fiscal_year INTEGER NOT NULL,
    UNIQUE(tenant_id, document_type, fiscal_year)
);

-- Acquire next number atomically
-- In Go: within a transaction
-- SELECT current_number FROM finance_number_sequences
--   WHERE document_type = 'invoice' AND fiscal_year = 2026
--   FOR UPDATE;
-- UPDATE ... SET current_number = current_number + 1;
-- Returns: RE-2026-0001, RE-2026-0002, etc.
```

### Pattern 3: Tax Calculation as Pure Function

**What:** Tax calculation is a stateless pure function that takes line items and a tax mode, returning a complete tax breakdown. No side effects, fully testable.

**When to use:** Every quote/invoice create/update, PDF generation, dashboard aggregation.

**Example:**
```go
type TaxMode string
const (
    TaxModeStandard        TaxMode = "standard"         // 19% or 7%
    TaxModeReverseCharge   TaxMode = "reverse_charge"    // 0%, note on invoice
    TaxModeKleinunternehmer TaxMode = "kleinunternehmer" // 0%, note on invoice
)

type TaxBreakdown struct {
    Subtotal    decimal.Decimal            // Sum of line totals (net)
    TaxByRate   map[string]decimal.Decimal  // e.g., "19.00" -> 45.60
    TotalTax    decimal.Decimal
    GrossTotal  decimal.Decimal
}

func Calculate(items []LineItem, mode TaxMode) TaxBreakdown {
    // Pure function, no DB, no side effects
    var breakdown TaxBreakdown
    breakdown.TaxByRate = make(map[string]decimal.Decimal)

    for _, item := range items {
        lineNet := item.Quantity.Mul(item.UnitPrice)
        breakdown.Subtotal = breakdown.Subtotal.Add(lineNet)

        if mode == TaxModeStandard {
            tax := lineNet.Mul(item.TaxRate).Div(decimal.NewFromInt(100))
            rateKey := item.TaxRate.String()
            breakdown.TaxByRate[rateKey] = breakdown.TaxByRate[rateKey].Add(tax)
            breakdown.TotalTax = breakdown.TotalTax.Add(tax)
        }
        // ReverseCharge and Kleinunternehmer: tax = 0
    }
    breakdown.GrossTotal = breakdown.Subtotal.Add(breakdown.TotalTax)
    return breakdown
}
```

### Pattern 4: Copy-and-Link for Quote-to-Invoice Conversion

**What:** When converting a quote to an invoice, create a new invoice document with data copied from the quote and a `source_quote_id` FK linking them. The quote's status transitions to "Angenommen" (accepted). The original quote remains as an immutable historical record.

**When to use:** FIN-05 deal-to-quote-to-invoice flow.

**Recommendation:** Copy-and-link is better than status-transition because:
1. Quotes and invoices have different Pflichtangaben (quote doesn't need Leistungsdatum)
2. Quote may have different validity terms vs. payment terms
3. User may want to modify line items when converting
4. Audit trail is clearer: "Quote AN-2026-003 led to Invoice RE-2026-007"

### Anti-Patterns to Avoid

- **Mutable sent invoices:** Once status = sent, the invoice data MUST NOT change. Enforce at service layer AND consider a DB trigger as safety net. Any correction requires a credit note + new invoice.
- **Float64 for money:** Always use `shopspring/decimal`. The project already does this for deals. Apply the same pattern to all finance calculations.
- **Hardcoded tax rates:** Tax rates should be per-line-item, not per-invoice. A single invoice may have items at 19% and 7%.
- **Manual sequence numbers:** Never let the client supply the invoice number. The server must generate it atomically within a transaction using FOR UPDATE locking.
- **Storing calculated totals without breakdown:** Always store both the inputs (line items with per-item tax rates) AND the calculated outputs (tax breakdown) in the snapshot. This enables audit verification.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| PDF generation | HTML-to-PDF pipeline (needs Chromium) | maroto/v2 | Pure Go, single binary, no external deps, good table support |
| Decimal arithmetic | float64 math for money | shopspring/decimal | Rounding errors on percentage calculations, already in go.mod |
| CSV generation | Manual string concatenation | encoding/csv (stdlib) | Handles quoting, escaping, delimiters correctly |
| Sequential numbering | Application-level counter | PostgreSQL SELECT FOR UPDATE | Database-level atomicity prevents duplicates under concurrency |
| Date calculations | Manual day math | time.AddDate() + time package | Handles month boundaries, leap years correctly |

**Key insight:** Financial software has zero tolerance for rounding errors and data integrity issues. Every calculation must use decimal arithmetic, every mutation must be transactional, and every finalized document must be immutable.

## Common Pitfalls

### Pitfall 1: Invoice Number Gaps Under Concurrent Requests
**What goes wrong:** Two users send invoices simultaneously, race condition causes duplicate or skipped numbers.
**Why it happens:** SELECT max(number) + 1 without proper locking.
**How to avoid:** Use a dedicated sequence table with SELECT FOR UPDATE within the same transaction that creates the invoice. Advisory locks (pg_advisory_xact_lock) are an alternative.
**Warning signs:** Duplicate invoice numbers in the DB, numbers like RE-2026-003 missing between RE-2026-002 and RE-2026-004.

### Pitfall 2: Floating-Point Tax Calculation Errors
**What goes wrong:** 19% of 123.45 EUR = 23.4555... rounds differently than expected.
**Why it happens:** IEEE 754 floating-point cannot represent 0.19 exactly.
**How to avoid:** Use shopspring/decimal for ALL financial calculations. Round only at the final display step. Apply rounding per line item (not on totals).
**Warning signs:** Invoice totals off by 1 cent, tax amounts don't add up.

### Pitfall 3: Modifying Sent Invoices
**What goes wrong:** User changes a line item on a sent invoice, violating GoBD.
**Why it happens:** No immutability enforcement at service layer.
**How to avoid:** Service layer rejects all mutations when status != draft. The JSONB snapshot is the frozen record. Any correction requires: credit note for original -> new corrected invoice.
**Warning signs:** UpdateInvoice succeeds for non-draft invoices, snapshot_data differs from line items.

### Pitfall 4: DATEV Import Fails Silently
**What goes wrong:** Steuerberater imports CSV, DATEV shows 0 bookings or errors.
**Why it happens:** Wrong encoding (UTF-8 without BOM vs Windows-1252), wrong delimiter, missing header fields, wrong date format.
**How to avoid:** Use the exact EXTF header format. Semicolon delimiter. Dates as DDMM (4-digit, no separator). Amounts without thousand separator, comma as decimal. Test with DATEV Belegvorgabe or a Steuerberater.
**Warning signs:** Empty import in DATEV, encoding errors showing special characters (umlauts).

### Pitfall 5: Overdue Detection Timezone Issues
**What goes wrong:** Invoice due 2026-02-28 shows as overdue on 2026-02-28 at 23:00 UTC (which is 2026-03-01 in CET).
**Why it happens:** Comparing timestamps without considering timezone.
**How to avoid:** Store due dates as DATE (not TIMESTAMPTZ). Compare with `CURRENT_DATE` in PostgreSQL (which uses the server's timezone). Or explicitly: `WHERE due_date < CURRENT_DATE AT TIME ZONE 'Europe/Berlin'`.
**Warning signs:** Invoices become overdue a day early or late depending on server timezone.

### Pitfall 6: Verzugszinsen Calculation with Wrong Basiszinssatz
**What goes wrong:** Dunning letter shows incorrect interest amount.
**Why it happens:** Using a hardcoded base rate instead of the current Bundesbank rate.
**How to avoid:** Store the Basiszinssatz as a configurable setting (admin can update semi-annually). Current value: 1.27% as of 2026-01-01. Default interest: B2C = base + 5% = 6.27%, B2B = base + 9% = 10.27%.
**Warning signs:** Interest amounts don't match Bundesbank published rates.

## Code Examples

### DATEV Buchungsstapel EXTF Header

```go
// Source: DATEV Developer Portal + community documentation
func writeEXTFHeader(w *csv.Writer, fiscalYearStart time.Time, generatedAt time.Time) {
    // Line 1: File header
    header := []string{
        `"EXTF"`,           // Format identifier (External Transfer Format)
        "700",              // Version number
        "21",               // Data category (21 = Buchungsstapel)
        `"Buchungsstapel"`, // Format name
        "13",               // Format version
        generatedAt.Format("20060102150405") + "000", // Timestamp
        "",                 // Reserved
        `"KMU Hub"`,        // Source application
        `""`,               // Reserved
        `""`,               // Reserved
        "",                 // Consultant number (Beraternummer) - optional
        "",                 // Client number (Mandantennummer) - optional
        fiscalYearStart.Format("20060102"), // Fiscal year start
        "4",                // Account length (SKR03 standard)
        fiscalYearStart.Format("20060102"), // Booking period start
        "",                 // Booking period end
        `"Buchungsstapel"`, // Label
        `""`,               // Dictation shortcut
        "1",                // Booking type (1 = Finanzbuchfuehrung)
        "0",                // Locking (0 = not locked)
        "0",                // Reserved
        `"EUR"`,            // Currency
        "",                 // Reserved
        `""`,               // Reserved
        "",                 // Reserved
        "",                 // Reserved
        `""`,               // Reserved
    }
    w.Write(header)
}
```

### Pflichtangaben Fields for German Invoices (UStG section 14)

```go
// Source: UStG section 14 Abs. 4, verified via multiple legal sources
type InvoicePflichtangaben struct {
    // 1. Vollstaendiger Name und Anschrift des leistenden Unternehmers
    SellerName    string
    SellerAddress string // Street, PLZ, City

    // 2. Vollstaendiger Name und Anschrift des Leistungsempfaengers
    BuyerName    string
    BuyerAddress string

    // 3. Steuernummer oder USt-IdNr des leistenden Unternehmers
    SellerTaxNumber string // Steuernummer
    SellerVATID     string // USt-IdNr (for EU transactions)

    // 4. Ausstellungsdatum (Rechnungsdatum)
    InvoiceDate time.Time

    // 5. Fortlaufende Rechnungsnummer
    InvoiceNumber string // Gap-free sequential

    // 6. Menge und Art der gelieferten Gegenstaende / Umfang der Leistung
    LineItems []LineItem

    // 7. Zeitpunkt der Lieferung/Leistung
    DeliveryDate *time.Time // "Leistungsdatum" - can equal invoice date

    // 8. Entgelt (aufgeschluesselt nach Steuersaetzen)
    NetAmountByRate map[string]decimal.Decimal

    // 9. Anzuwendender Steuersatz + Steuerbetrag
    TaxAmountByRate map[string]decimal.Decimal

    // 10. Ggf. Hinweis auf Steuerbefreiung
    TaxExemptionNote string // For Reverse Charge or Kleinunternehmer

    // Additional: payment terms, bank details, Handelsregistereintrag
    PaymentTerms     string
    BankDetails      BankInfo
    CommercialRegister string // e.g., "HRB 12345, AG Muenchen"
}

// Required notes for special tax modes:
// Reverse Charge: "Steuerschuldnerschaft des Leistungsempfaengers (section 13b UStG)"
// Kleinunternehmer: "Kein Ausweis von Umsatzsteuer, da Kleinunternehmer gemaess section 19 UStG"
```

### MwSt Calculation with Decimal Precision

```go
// Source: Project pattern (shopspring/decimal already in use)
func calculateLineTax(quantity, unitPrice, taxRatePercent decimal.Decimal) (net, tax, gross decimal.Decimal) {
    net = quantity.Mul(unitPrice)
    tax = net.Mul(taxRatePercent).Div(decimal.NewFromInt(100)).Round(2)
    gross = net.Add(tax)
    return
}

// German MwSt rates
var (
    RateStandard = decimal.NewFromFloat(19.0)  // Regelsteuersatz
    RateReduced  = decimal.NewFromFloat(7.0)   // Ermaessigter Satz
    RateZero     = decimal.NewFromFloat(0.0)   // Reverse Charge / Kleinunternehmer
)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| PDF invoices as legal format | XRechnung/ZUGFeRD e-invoices mandatory for B2B | 2025-01-01 (receipt), 2027/2028 (sending) | Phase 12 generates PDF for v1; XRechnung/ZUGFeRD is v2 (FIN-12). Transitional period allows PDF until 2027. |
| Kleinunternehmer threshold 22,000 EUR | New threshold 25,000 EUR (prior year) / 100,000 EUR (current year) | 2025-01-01 | Implementation must use updated thresholds |
| GoBD 2019 rules | GoBD 2025 amendment (e-invoice archiving) | 2025-07-14 | Our PDF approach is fine during transition; structured data (XML) archiving not required until e-invoicing mandate |
| gofpdf (archived 2021) | maroto/v2 (active, wraps gofpdf) | 2023 | Use maroto/v2 exclusively |

**Deprecated/outdated:**
- gofpdf: Archived since 2021, maroto/v2 wraps it with a better API
- Swiss CHF/8.1% in existing frontend mock: Must be replaced with EUR/19%/7% for Deutschland-First

## Discretion Recommendations

### Pflichtangaben Layout on PDF
**Recommendation:** Standard German footer block. Place Handelsregistereintrag, Steuernummer, USt-IdNr, bank details (IBAN/BIC) in a 3-column footer at the bottom of every page. This is the most common layout in German business invoices and what Steuerberater expect to see.

### Quote-to-Invoice Conversion Mechanics
**Recommendation:** Copy-and-link. Create a new invoice entity with data copied from the quote, store `source_quote_id` FK. Quote transitions to status "Angenommen". Rationale: different document types have different fields, user may adjust line items during conversion, clear audit trail.

### Deal Value Sync Direction
**Recommendation:** Unidirectional: quote -> deal only. When a quote is created from a deal or modified, update the deal's `value` field via CRM gRPC `UpdateDeal`. Do NOT sync deal changes back to quotes (quote data may be sent to customer and should not change retroactively). This keeps the quote as the single source of truth for customer-facing amounts.

### Revenue Chart Visualization
**Recommendation:** Area chart with filled gradient for revenue over time. Area charts convey cumulative trends better than bars for financial data and look cleaner. Use a subtle secondary line for comparison period (e.g., previous year/quarter).

### PDF Spacing and Typography
**Recommendation:** A4 portrait, 15mm margins. Header: company logo (left) + company details (right). Sans-serif font (Liberation Sans or Noto Sans for umlaut support). Line item table with thin 0.5pt borders, alternating row shading at 5% opacity. Font sizes: title 14pt, body 9pt, table 8pt, footer 7pt. Accent color from company settings for horizontal rules and headings only.

### DATEV Export Field Mapping
**Recommendation:** Map to standard SKR03 accounts:
- Revenue accounts: 8400 (Erloese 19%), 8300 (Erloese 7%), 8125 (steuerfreie innergemeinschaftliche Lieferungen)
- Debitor accounts: 10000-69999 range (auto-created per customer)
- Tax: automatic via BU-Schluessel (Berichtigungsschluessel) codes in DATEV

## Open Questions

1. **Company Settings Model**
   - What we know: Company model exists (models/company.go) with basic fields (name, address, domain, industry)
   - What's unclear: No fields for Steuernummer, USt-IdNr, Handelsregistereintrag, bank details (IBAN/BIC), logo URL, accent color. These are required for Pflichtangaben on invoices.
   - Recommendation: Add a `company_settings` or `tenant_settings` table in the migration (or extend the companies table with new columns). The biz service needs to read these for every PDF generation.

2. **Multi-Tenancy Readiness**
   - What we know: Current system has no explicit tenant_id; all data is implicitly single-tenant
   - What's unclear: Whether to add tenant_id columns now for future readiness
   - Recommendation: Add tenant_id to finance tables with a default UUID. Low cost now, prevents painful migration later. The sequence table already needs it for per-tenant numbering.

3. **DATEV Beraternummer / Mandantennummer**
   - What we know: DATEV header requires consultant number and client number
   - What's unclear: Whether users will know these values at setup time
   - Recommendation: Make them optional admin settings. DATEV import can work without them (some fields are optional), but Steuerberater will appreciate having them pre-filled.

4. **Existing Frontend Mock Data Uses CHF**
   - What we know: The Zustand finance store uses CHF formatting and 8.1% Swiss VAT rates
   - What's unclear: N/A -- this is clearly wrong per Deutschland-First decision
   - Recommendation: The frontend rewrite must replace all CHF references with EUR, all 8.1%/7.7% rates with 19%/7%, and all de-CH locale formatting with de-DE.

## Sources

### Primary (HIGH confidence)
- Codebase exploration: gateway registration pattern, service structure, migration numbering (000044 is latest), deal model, config, proto patterns
- [Bundesbank Basiszinssatz](https://www.bundesbank.de/de/presse/pressenotizen/bekanntgabe-des-basiszinssatzes-zum-1-januar-2026-basiszinssatz-bleibt-unveraendert-bei-1-27--973974) - Current base rate 1.27% as of 2026-01-01
- [UStG section 14 via Stripe](https://stripe.com/resources/more/14-ustg) - Pflichtangaben requirements
- [BGB section 288 via buzer.de](https://www.buzer.de/288_BGB.htm) - Verzugszinsen calculation rules
- [Invoicing in Germany (Eurofiscalis)](https://www.eurofiscalis.com/en/invoicing-in-germany/) - Complete mandatory field list
- [Reverse Charge via Stripe](https://stripe.com/resources/more/reverse-charge-vat-germany) - section 13b UStG requirements
- [DATEV Developer Portal](https://developer.datev.de/en/file-format/details/datev-format/getting-started) - EXTF format specification
- [DATEV EXTF example (GitHub)](https://github.com/ledermann/datev/blob/master/examples/EXTF_Buchungsstapel.csv) - Reference CSV structure

### Secondary (MEDIUM confidence)
- [GoBD 2025 Amendment (AODocs)](https://www.aodocs.com/blog/gobd-explained-requirements-for-audit-ready-digital-bookkeeping-in-germany-and-beyond/) - GoBD compliance overview
- [GoBD 2025 (KMLZ)](https://www.kmlz.de/en/gobd-2025-whats-new-procedural-documentation-again-focus-german-tax-authorities) - Updated GoBD rules
- [Kleinunternehmerregelung 2026 (sevdesk)](https://sevdesk.de/ratgeber/buchhaltung-finanzen/kleinunternehmer/) - Updated thresholds
- [Maroto v2 (GitHub)](https://github.com/johnfercher/maroto) - PDF generation library
- [ConAktiv DATEV Handbook](https://handbuch.conaktiv.de/wiki/version-15/buchhaltungsmodule/buchhaltung-in-conaktiv/nutzung-der-datev-schnittstelle-2017/datev-buchungsstapel-datei-extf-buchungsstapel-csv/) - DATEV field mapping details
- [Settle in Berlin Invoice Guide](https://www.settle-in-berlin.com/how-to-become-a-freelancer-in-germany-self-employed/invoice-format-germany/) - Practical Pflichtangaben checklist

### Tertiary (LOW confidence)
- DATEV header field positions: Partially reconstructed from community documentation and example CSV. Official DATEV specification requires developer portal registration for full download. The header structure shown in Code Examples is based on multiple community sources but should be validated against a Steuerberater's DATEV import.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - maroto/v2 is established, shopspring/decimal already in use, patterns directly from codebase
- Architecture: HIGH - follows exact same patterns as document/email services, well-established in this project
- Legal/compliance (GoBD, UStG, BGB): HIGH - verified from official German legal sources (gesetze-im-internet.de, Bundesbank, IHK)
- DATEV format: MEDIUM - community documentation cross-referenced, but official spec requires DATEV developer portal access
- Tax calculation: HIGH - MwSt rates are well-defined statutory rates, Kleinunternehmer thresholds updated from official IHK/sevdesk sources
- Pitfalls: HIGH - common financial software issues, verified from domain experience and multiple sources

**Research date:** 2026-02-18
**Valid until:** 2026-07-01 (Basiszinssatz updates semi-annually; e-invoicing mandate may affect requirements by 2027)

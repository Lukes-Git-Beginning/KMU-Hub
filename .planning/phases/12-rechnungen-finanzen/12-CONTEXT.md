# Phase 12: Rechnungen & Finanzen - Context

**Gathered:** 2026-02-18
**Status:** Ready for planning

<domain>
## Phase Boundary

GoBD-compliant quotes and invoices, tax calculation (MwSt 19%/7%/0%), DATEV export, credit notes, 3-level dunning, payment tracking dashboard. Deutschland-First: EUR, de-DE locale.

**NOT in scope:** Full accounting (FiBu/Buchhaltung), double-entry bookkeeping, payroll (integration-only via Bexio/Abacus/RmA in Phases 18-19).

</domain>

<decisions>
## Implementation Decisions

### Document Design & PDF
- Clean minimal PDF template style (Stripe/Lexoffice aesthetic: white background, thin lines, modern sans-serif)
- Company branding: logo in header + accent color from company settings on headings/lines
- Simple line item table: Position, Bezeichnung, Menge, Einzelpreis, Gesamtpreis
- No grouped sections or multi-line descriptions per line item in v1

### Deal-to-Invoice Flow
- Both paths supported: standalone invoice creation AND deal → quote → invoice conversion
- Quotes auto-expire with configurable default validity period (e.g., 30 days), status changes to "Abgelaufen"
- Deal value auto-syncs when quote is created or modified (single source of truth for pipeline revenue)

### Dunning Behavior
- Semi-automatic: system detects overdue invoices and creates draft Mahnungen, user reviews and sends manually
- Admin-configurable intervals between dunning levels (no hardcoded defaults)
- Standard 3-level German dunning:
  - Level 1: Zahlungserinnerung (friendly tone)
  - Level 2: 1. Mahnung (formal tone)
  - Level 3: 2. Mahnung / Letzte Mahnung (urgent, threatens Inkasso)
- Configurable Mahngebuehren per level (e.g., 0/5/10 EUR) + Verzugszinsen per BGB §288 (5% above Basiszinssatz)

### Finance Dashboard
- Combined revenue + pipeline + status metrics:
  - Revenue: total invoiced, total paid, total outstanding, overdue amount
  - Pipeline: quotes pending, conversion rate (quote→invoice), average deal size, revenue forecast
  - Status: draft/sent/overdue/paid counts as breakdown
- Custom date range picker (from/to) plus predefined shortcuts (this month, quarter, year, etc.)
- Subtle badge indicators on status cards for actionable items (overdue invoices, expiring quotes, pending Mahnungen) — no separate action section

### Claude's Discretion
- Pflichtangaben layout on PDF (standard German footer block vs split layout)
- Quote-to-invoice conversion mechanics (copy-and-link vs status-transition)
- Deal value sync direction (quote → deal only, or bidirectional)
- Revenue chart visualization style (bar vs area/line)
- Exact spacing, typography, and color application on PDFs
- DATEV export field mapping details

</decisions>

<specifics>
## Specific Ideas

- PDF style reference: clean and minimal like Stripe invoices or Lexoffice — not cluttered, not overly designed
- Dunning follows standard German business practice: Zahlungserinnerung is the polite first nudge, Letzte Mahnung explicitly references potential Inkasso
- Verzugszinsen calculation must follow BGB §288 (5 percentage points above Basiszinssatz for B2C, 9 for B2B)
- Dashboard should give a full picture without being overwhelming — metrics + status at a glance, charts for trend

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 12-rechnungen-finanzen*
*Context gathered: 2026-02-18*

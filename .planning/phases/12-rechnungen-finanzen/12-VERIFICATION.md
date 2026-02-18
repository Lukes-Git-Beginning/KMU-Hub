---
phase: 12-rechnungen-finanzen
verified: 2026-02-18T15:00:00Z
status: passed
score: 12/12 must-haves verified
re_verification:
  previous_status: gaps_found
  previous_score: 10/12
  gaps_closed:
    - "User can download PDFs for quotes, invoices, credit notes, and dunning letters (Gap 1 -- PDF 501 stubs removed, real gRPC-to-HTTP streaming wired)"
    - "User can convert a CRM deal to a quote in a seamless flow -- FIN-05 (Gap 2 -- deal-to-quote gateway route and DealDetailPage button added)"
  gaps_remaining: []
  regressions: []
human_verification:
  - test: "Open a quote PDF from the finance module"
    expected: "PDF downloads with all Pflichtangaben in footer (Steuernummer, IBAN, Handelsregister) per section 14 UStG"
    why_human: "PDF binary content can only be verified by reading the downloaded document visually"
  - test: "Create a quote from the CRM deals view via Angebot erstellen button"
    expected: "Quote is created with customer name/address/email pre-filled from the deal; user is navigated to /finanzen"
    why_human: "End-to-end UI flow requires browser interaction and visual confirmation of pre-populated fields"
  - test: "Dunning letter tone escalation"
    expected: "Level 1 is friendly (Zahlungserinnerung), Level 2 formal (1. Mahnung), Level 3 urgent with Inkasso threat"
    why_human: "PDF content verification requires reading the document"
  - test: "DATEV export file validity"
    expected: "File imports without errors in DATEV-compatible tooling; SKR03 accounts (8400, 8300) correct; BU-Schluessel present"
    why_human: "DATEV tooling validation cannot be done programmatically"
---

# Phase 12: Rechnungen & Finanzen Verification Report

**Phase Goal:** Users can create legally compliant quotes and invoices, track payments, manage dunning, and export to their Steuerberater -- replacing standalone invoicing tools. This is NOT full accounting (Buchhaltung/FiBu) -- no double-entry bookkeeping, no payroll.
**Verified:** 2026-02-18T15:00:00Z
**Status:** passed
**Re-verification:** Yes -- after gap closure plans 12-06 (PDF streaming) and 12-07 (deal-to-quote route)

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Proto defines FinanceService with ~34 RPCs for all document types | VERIFIED | `backend/proto/biz/v1/biz.proto` lines 9-64: now 38 RPCs (original 34 + 4 PDF RPCs added in 12-06) |
| 2 | Database tables exist for all finance entities | VERIFIED (no change) | `backend/migrations/000045_create_finance_tables.up.sql`: 8 tables unchanged |
| 3 | Tax calculator correctly computes 19%/7%/0% RC/0% KU with decimal precision | VERIFIED (no change) | `backend/internal/biz/tax/calculator.go` with 15 tests; no modifications in gap closure plans |
| 4 | Invoice immutability and gap-free sequential numbering are enforced | VERIFIED (no change) | `ErrInvoiceImmutable` in invoice/errors.go; `SELECT FOR UPDATE` in postgres_repository.go; unchanged |
| 5 | Quote auto-syncs deal value to CRM when deal_id is set | VERIFIED (no change) | `quote/service.go` calls `s.dealUpdater.UpdateDealValue()` in Create and Update; unchanged |
| 6 | Credit notes, payments, and dunning lifecycle are fully implemented | VERIFIED (no change) | creditnote/service.go, payment/service.go, dunning/service.go all intact; dunning/service.go gained `GetByID` in 12-06 |
| 7 | Finance dashboard aggregates revenue, pipeline, and status metrics | VERIFIED (no change) | `dashboard/service.go` and `FinanceDashboard.tsx` with `useFinanceDashboard` hook; unchanged |
| 8 | DATEV Buchungsstapel CSV export produces valid EXTF format | VERIFIED (no change) | `datev/exporter.go` with UTF-8 BOM, EXTF header, SKR03 accounts; unchanged |
| 9 | Gateway exposes ~30 HTTP routes under /api/v1/finance/* | VERIFIED (no change, +1 route) | `route_biz.go` retains all original routes; gained `POST /api/v1/finance/deals/{dealId}/quote` in 12-07 |
| 10 | Frontend uses TanStack Query (not Zustand mock) for all finance data | VERIFIED (no change) | `FinanzenPage.tsx` still imports `useInvoices`, `useQuotes` from hooks; no Zustand mock references |
| 11 | Users can download PDFs for quotes, invoices, credit notes, and dunning letters | VERIFIED (was FAILED) | `respondPDFNotImplemented` is completely removed (0 occurrences); all 4 gateway handlers now call gRPC PDF RPCs and return `application/pdf` binary via `respondPDF` helper; 4 gRPC server implementations fetch doc + company settings + generate via `pdf.NewGenerator` + return bytes |
| 12 | User can convert a CRM deal to a quote in a seamless flow (FIN-05) | VERIFIED (was PARTIAL) | `POST /api/v1/finance/deals/{dealId}/quote` registered at gateway line 137-140; `HandleCreateQuoteFromDeal` fetches deal/contact/company from CRM gRPC, builds customer snapshot, calls `bizClient.CreateQuote` with `deal_id` set; `DealDetailPage.tsx` line 70 calls `useCreateQuoteFromDeal()`, line 172-177 renders "Angebot erstellen" button with `onClick={handleCreateQuoteFromDeal}` and `disabled={isPending}` |

**Score:** 12/12 truths verified

---

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `backend/proto/biz/v1/biz.proto` | FinanceService gRPC definition (now 38 RPCs) | VERIFIED | 4 PDF RPCs added at lines 61-64 with 8 new message types at lines 711-750 |
| `backend/proto/biz/v1/biz.pb.go` | Generated protobuf Go code | VERIFIED | Regenerated in 12-06; includes `GenerateQuotePDFRequest/Response` etc. |
| `backend/proto/biz/v1/biz_grpc.pb.go` | Generated gRPC Go code | VERIFIED | Regenerated; PDF client/server interfaces present |
| `backend/internal/server/biz_grpc.go` | gRPC server with 4 PDF generation methods | VERIFIED | `GenerateQuotePDF` at line 1131, `GenerateInvoicePDF` at 1161, `GenerateCreditNotePDF` at 1191, `GenerateDunningPDF` at 1221; each fetches doc, fetches company settings, creates fresh `pdf.NewGenerator`, returns bytes |
| `backend/internal/biz/dunning/service.go` | Service with GetByID method (added in 12-06) | VERIFIED | `func (s *Service) GetByID(...)` at line 274; passthrough to `s.repo.GetByID` |
| `backend/internal/gateway/route_biz.go` | PDF handlers streaming gRPC bytes; deal-to-quote route | VERIFIED | `respondPDF` helper at line 1459-1466; `HandleCreateQuoteFromDeal` at line 1282; route registered at line 137 |
| `desktop/src/renderer/src/modules/crm/deals/DealDetailPage.tsx` | Angebot erstellen button consuming useCreateQuoteFromDeal | VERIFIED | `useCreateQuoteFromDeal()` at line 70; `handleCreateQuoteFromDeal` at line 87-97; button at lines 169-177 |

All artifacts from the original verification (plans 12-01 through 12-05) remain unchanged and verified. Only the gap-closure artifacts above required re-verification.

---

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `gateway/route_biz.go` (PDF handlers) | `proto/biz/v1/biz_grpc.pb.go` | `client.GenerateQuotePDF` etc. | WIRED | Lines 510, 749, 887, 1132 call gRPC PDF RPCs with `{Id: id, TenantId: tenantID}` |
| `gateway/route_biz.go` (respondPDF) | HTTP client | `Content-Type: application/pdf` | WIRED | `respondPDF` helper at line 1460 sets `application/pdf`, `Content-Disposition`, `Content-Length` |
| `server/biz_grpc.go` (GeneratePDF methods) | `biz/pdf/generator.go` | `pdf.NewGenerator(*settings).Generate*PDF()` | WIRED | All 4 methods: `gen := pdf.NewGenerator(*settings)` then `gen.GenerateQuotePDF(*q)` etc. |
| `server/biz_grpc.go` (GenerateDunningPDF) | `biz/dunning/service.go GetByID` | `s.dunningService.GetByID` | WIRED | Line 1227: `dr, err := s.dunningService.GetByID(ctx, tenantID, id)` |
| `gateway/route_biz.go` (HandleCreateQuoteFromDeal) | CRM gRPC via `getCRMClient()` | `crmClient.GetDeal`, `GetContact`, `GetCompany` | WIRED | Lines 1300, 1321, 1335: fetches deal, then conditionally fetches contact and company for customer snapshot |
| `gateway/route_biz.go` (HandleCreateQuoteFromDeal) | Biz gRPC `CreateQuote` | `bizClient.CreateQuote` with `DealId` set | WIRED | Line 1359-1371: `bizClient.CreateQuote` with `CustomerSnapshot`, `DealId: dealID`, `TaxMode: TAX_MODE_STANDARD` |
| `DealDetailPage.tsx` | `useFinance.ts` | `useCreateQuoteFromDeal` import | WIRED | Line 25: `import { useCreateQuoteFromDeal } from '@/api/hooks/useFinance'`; line 70: called as hook |
| `useCreateQuoteFromDeal` | `finance-client.ts` | `financeDealApi.createQuoteFromDeal` | WIRED | Confirmed from initial verification; unchanged in gap closure plans |
| `useDownloadQuotePDF` | `/api/v1/finance/quotes/{id}/pdf` | `financeQuoteApi.getPDF` | WIRED | Line 474: `mutationFn: (id: string) => financeQuoteApi.getPDF(id)` |

All previously NOT_WIRED links are now WIRED. No previously WIRED links have regressed.

---

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| FIN-01 | 12-01, 12-02, 12-05, 12-06 | User can create quotes with line items, tax calculation, and PDF generation | SATISFIED | Quote CRUD and tax calculation verified (initial); PDF download now functional via real gRPC PDF RPCs (12-06); `respondPDFNotImplemented` removed |
| FIN-02 | 12-01, 12-02, 12-05, 12-06 | User can create GoBD-compliant invoices (immutable, sequential numbering, Pflichtangaben) | SATISFIED | Immutability and sequential numbering verified (initial); Pflichtangaben in PDF footer code; PDF download now functional (12-06); gap closed |
| FIN-03 | 12-01, 12-05 | System calculates MwSt/USt correctly (19%/7%/0% RC/0% KU) | SATISFIED (no change) | Tax calculator with 15 tests; unchanged |
| FIN-04 | 12-03, 12-05 | User can track payment status per invoice (draft/sent/overdue/paid/cancelled) | SATISFIED (no change) | Payment recording with auto-transition; dunning 3-level management; unchanged |
| FIN-05 | 12-04, 12-05, 12-07 | User can convert a CRM deal to a quote and then to an invoice in a seamless flow | SATISFIED | Deal-to-quote gateway route added (12-07); DealDetailPage "Angebot erstellen" button wired; quote-to-invoice via `CreateFromQuote` already worked; `useCreateQuoteFromDeal` no longer orphaned |
| FIN-06 | 12-04, 12-05 | User can export Buchungsstapel in DATEV-compatible CSV format | SATISFIED (no change) | DATEV EXTF exporter with SKR03 mapping; unchanged |
| FIN-07 | 12-03, 12-05 | User can create credit notes referencing original invoices | SATISFIED (no change) | Credit note service, GS-prefix numbering, CreditNoteDialog; unchanged |

All 7 requirements (FIN-01 through FIN-07) are SATISFIED.

---

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `backend/internal/biz/invoice/postgres_repository.go` | 238-245 | `NextInvoiceNumber` comment says "delegated to shared repo" with no actual implementation in this file | Info (unchanged) | Could mislead future developers; actual `SELECT FOR UPDATE` is in quote package. Non-blocking. |

No new anti-patterns introduced by gap closure plans. The previous blocker (`respondPDFNotImplemented`) has been eliminated. The previous warning (`useCreateQuoteFromDeal` orphaned hook) has been resolved.

---

### Human Verification Required

#### 1. PDF Download from Invoice Detail Panel

**Test:** Open the finance module, create a test invoice, send it, and click the PDF download button.
**Expected:** PDF downloads with all Pflichtangaben in footer (Steuernummer, USt-IdNr, IBAN/BIC, Handelsregister) per section 14 UStG; German filename such as `Rechnung_RE-2024-001.pdf`.
**Why human:** PDF binary content and layout can only be verified by reading the downloaded document visually.

#### 2. Quote PDF with Reverse Charge Tax Note

**Test:** Create a Reverse Charge quote (EU B2B customer) and download its PDF.
**Expected:** PDF contains "Steuerschuldnerschaft des Leistungsempfaengers (section 13b UStG)" note below the totals.
**Why human:** PDF content can only be verified by reading the document visually.

#### 3. CRM Deal to Quote End-to-End Flow

**Test:** Navigate to the CRM module, open a deal, click "Angebot erstellen". Verify the quote appears in the finance module with pre-filled customer data.
**Expected:** Quote created with customer name, address, and email from the deal's linked contact/company; user navigated to /finanzen; the new quote is visible in the Angebote tab.
**Why human:** End-to-end UI flow and CRM-to-finance data pre-population requires browser interaction.

#### 4. Dunning Letter Tone by Level

**Test:** Create Level 1, Level 2, and Level 3 dunning records; generate PDFs for each.
**Expected:** Level 1 Zahlungserinnerung is friendly in tone; Level 2 (1. Mahnung) is formal; Level 3 (2. Mahnung) is urgent with Inkasso threat. Filenames match: `Zahlungserinnerung_RE-*.pdf`, `1_Mahnung_RE-*.pdf`, `2_Mahnung_RE-*.pdf`.
**Why human:** PDF content and tone verification requires reading the document.

#### 5. DATEV Export File Validity

**Test:** Export DATEV Buchungsstapel for a date range with test invoices; open the CSV in a DATEV-compatible tool.
**Expected:** File imports without errors; accounts map to SKR03 (8400 for 19%, 8300 for 7%); UTF-8 BOM present; semicolon delimiters; EXTF header correct.
**Why human:** DATEV tooling validation cannot be performed programmatically.

---

## Gaps Summary

No gaps remain. Both gaps identified in the initial verification have been closed:

**Gap 1 -- PDF Download (CLOSED in plan 12-06):**
All 4 gateway PDF endpoints (`GET /api/v1/finance/quotes/{id}/pdf`, `/invoices/{id}/pdf`, `/credit-notes/{id}/pdf`, `/dunning/{id}/pdf`) now return real `application/pdf` binary data. The implementation is complete end-to-end:
- Proto: 4 PDF RPCs with `bytes pdf_data` response fields
- gRPC server: 4 implementations each fetching the document + company settings + calling `pdf.NewGenerator(*settings).Generate*PDF()` + returning bytes
- Gateway: 4 handlers replaced with real gRPC calls + `respondPDF` helper (Content-Type, Content-Disposition, Content-Length)
- `respondPDFNotImplemented` deleted completely (0 occurrences in entire codebase)
- Dunning service gained `GetByID` method required by the PDF gRPC implementation

**Gap 2 -- Deal-to-Quote Route (CLOSED in plan 12-07):**
The FIN-05 "seamless deal-to-quote flow" is now complete:
- Gateway: `POST /api/v1/finance/deals/{dealId}/quote` registered; `HandleCreateQuoteFromDeal` fetches CRM deal/contact/company via gRPC, builds customer snapshot (company name preferred over contact name, B2B DACH norm), calls `bizClient.CreateQuote` with `deal_id` and `CustomerSnapshot`
- Frontend: `DealDetailPage.tsx` imports `useCreateQuoteFromDeal`, renders "Angebot erstellen" button with loading state ("Erstelle..."), navigates to `/finanzen` on success
- Previously orphaned `useCreateQuoteFromDeal` hook is now consumed

Phase 12 (Rechnungen & Finanzen) achieves its goal: users can create legally compliant quotes and invoices, track payments, manage dunning, download PDFs, convert CRM deals to quotes, and export to DATEV for their Steuerberater.

---

*Verified: 2026-02-18T15:00:00Z*
*Verifier: Claude (gsd-verifier)*
*Re-verification after gap closure plans 12-06 and 12-07*

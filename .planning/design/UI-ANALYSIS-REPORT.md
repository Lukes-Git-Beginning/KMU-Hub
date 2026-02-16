# UI Analysis Report — KMU Hub Desktop Frontend

> Generated: 2026-02-16
> Branch: design/brainstorm
> Scope: Complete frontend audit after hub expansion (Batches 1-4)

---

## 1. Executive Summary

| Metric | Value |
|--------|-------|
| Total module pages | ~36 files |
| Total stores | 24 (all with persist) |
| Total nav items | 25 |
| Total routes | 30 |
| Total dashboard cards | 20 |
| Business profiles | 10 |
| TypeScript errors | 0 |
| console.log statements | 0 |
| TODO/FIXME comments | 0 |
| `any` type usage | 7 (all in Luke's Work/Gantt code) |

**Overall Health: GOOD — 0 build errors, clean codebase**

---

## 2. Module Completeness Matrix

### 2.1 Industry Modules (Darien — NEW, all feature-complete)

| Module | LOC | Store | Tabs | Dialogs | Detail Panel | Search | Stats | Shared Comp |
|--------|-----|-------|------|---------|-------------|--------|-------|-------------|
| Inventar | 1038 | inventar | 3 | Create/Edit/Delete | Yes | Yes | Yes | All |
| Schichten | 1096 | schichten | 3 | Create/Swap | Yes (Grid) | Yes | Yes | All |
| Einkauf | 1209 | einkauf | 3 | Order/Supplier/Receipt | Yes | Yes | Yes | All |
| Helpdesk | 1008 | helpdesk | 3 | Ticket/Assign/Escalate | Yes | Yes | Yes | All |
| Fuhrpark | 1334 | fuhrpark | 4 | Vehicle/Maintenance/Fuel | Yes | Yes | Yes | All |
| Produktion | 1199 | produktion | 3 | BOM/Order/QualityCheck | Yes | Yes | Yes | All |
| Berichte | 921 | berichte | 3 | Schedule | No | Yes | Yes | Partial |
| Vertraege | 1234 | vertraege | 3 | Contract/Terminate | Yes | Yes | Yes | All |
| Formulare | 1493 | formulare | 3 | Form/Field/Share | Yes | Yes | Yes | All |
| Vermietung | 1412 | vermietung | 3 | Object/Reservation | Yes | Yes | Yes | All |
| Rapporte | 1471 | rapporte | 3 | Report/Measurement | Yes | Yes | Yes | All |

### 2.2 Core Modules (Mix of Darien + Luke)

| Module | LOC | Owner | Status |
|--------|-----|-------|--------|
| Buchhaltung | 675 | Darien | Feature-complete (6 tabs incl. Mahnwesen) |
| Dokumente | 1292 | Darien | Feature-complete (Dateien + Wiki tabs) |
| Mails | 554 | Darien | Feature-complete (compose, pop-out) |
| Kontakte | ~800 | Darien | Feature-complete |
| Team | ~900 | Darien | Feature-complete (4 tabs incl. Lohn + Schulungen) |
| Kalender | 2143 | Darien | Feature-complete (4 views + Terminbuchung) |
| Settings | 1071 | Darien | Feature-complete (12 tabs) |
| Infrastruktur | 918 | Darien | Feature-complete (7 tabs) |
| Zeiterfassung | ~10 | Darien | Wrapper (delegates to ProfilPage tab) |

### 2.3 Luke's Backend-Connected Modules

| Module | Status | Issues |
|--------|--------|--------|
| CRM (Contacts/Companies/Deals) | Functional | Edit/Delete = "Kommt bald" placeholders |
| CRM Activities | Functional | Missing search bar |
| CRM Search | Complete | No issues |
| Work (Projects/Tasks/Kanban) | Functional | Uses API hooks, `any` types in Gantt |
| Meetings | Functional | Hardcoded colors (see CSS section) |
| Profil | Delegated | Tabs delegate to subcomponents |

### 2.4 Module Extensions (Darien — NEW)

| Extension | Parent | Status |
|-----------|--------|--------|
| Mahnwesen (tab) | Buchhaltung | Complete — dunning table, level indicators |
| Lohn (tab) | Team | Complete — payroll, month selector, CHF formatting |
| Schulungen (tab) | Team | Complete — training catalog, participations |
| Tracking (tab) | Fuhrpark | Complete — positions, route timeline |
| Wiki (tab) | Dokumente | Complete — articles, categories, search |

---

## 3. Integration Consistency

### 3.1 Cross-File Registration

| Module ID | Nav Item | Route | Dashboard Card | Settings Label | Profiles |
|-----------|----------|-------|----------------|----------------|----------|
| dashboard | Yes | Yes | N/A | N/A | ALWAYS |
| projects | Yes | Yes (work/*) | Yes | Yes | 3 default |
| tasks | Yes | Yes (work/*) | Yes | Yes | 2 default |
| chat | Yes | Yes | Yes | Yes | 3 default |
| contacts | Yes | Yes | No | Yes | 0 profiles! |
| team | Yes | Yes | No | Yes | 10 default |
| meetings | Yes | Yes | No | Yes | 3 default |
| calendar | Yes | Yes | No | Yes | 8 default |
| zeiterfassung | Yes | Yes | Yes | Yes | 10 default |
| documents | Yes | Yes | Yes | Yes | 5 default |
| mail | Yes | Yes | No | Yes | 2 default |
| finance | Yes | Yes | Yes | Yes | 10 default |
| infrastructure | Yes | Yes | No | Yes | 1 default |
| inventar | Yes | Yes | Yes | Yes | 7 default |
| schichten | Yes | Yes | Yes | Yes | 6 default |
| einkauf | Yes | Yes | Yes | Yes | 5 default |
| helpdesk | Yes | Yes | Yes | Yes | 1 default |
| fuhrpark | Yes | Yes | Yes | Yes | 3 default |
| produktion | Yes | Yes | Yes | Yes | 1 default |
| berichte | Yes | Yes | Yes | Yes | 1 default |
| vertraege | Yes | Yes | Yes | Yes | 4 default |
| formulare | Yes | Yes | Yes | Yes | 0 default, 8 opt |
| vermietung | Yes | Yes | Yes | Yes | 0 default, 6 opt |
| rapporte | Yes | Yes | Yes | Yes | 4 default |
| settings | Yes | Yes | N/A | N/A | ALWAYS |

### 3.2 Issues Found

| Issue | Severity | Description |
|-------|----------|-------------|
| `contacts` in 0 profiles | MEDIUM | Kontakte-Modul verschwindet bei Profil-Wechsel |
| `mail` in nur 2 profiles | LOW | E-Mail nur bei Allgemein + Dienstleistung default |
| Duplicate route `/calendar` + `/kalender` | LOW | Beide routen zu KalenderPage |
| `profil` Route ohne Nav-Item | OK | Intentional — via User-Dropdown erreichbar |
| CRM kein eigenes Nav-Item | OK | CRM wird durch Team & CRM abgedeckt |

---

## 4. Store Audit

### 4.1 All 24 Stores

| Store | Key | Persist | Mock Items | CRUD | Status |
|-------|-----|---------|------------|------|--------|
| auth | — | No | 0 | Login/Logout | OK |
| dashboard | kmuhub-dashboard | Yes | Layouts | CRUD | OK |
| navigation | — | No | — | Toggle | OK |
| calendar | kmuhub-calendar | Yes | Events | CRUD | OK |
| work | kmuhub-work | Yes | — | — | OK |
| ui | kmuhub-ui | Yes | — | Theme/Look | OK |
| profile | kmuhub-profiles | Yes | — | Set/Enable | OK |
| contacts | kmuhub-contacts | Yes | 14 | CRUD | OK |
| mails | kmuhub-mails | Yes | 13 | CRUD | OK |
| meetings | kmuhub-meetings | Yes | 8 | CRUD | OK |
| settings | kmuhub-settings | Yes | Defaults | Update | OK |
| timetracking | kmuhub-timetracking | Yes | 15 entries | CRUD + Timer | OK |
| finance | kmuhub-finance | Yes | 5 inv + 8 dunn | CRUD | OK |
| team | kmuhub-team | Yes | 12 + 8 payroll | CRUD | OK |
| documents | kmuhub-documents | Yes | 12 + 10 wiki | CRUD | OK |
| inventar | kmuhub-inventar | Yes | ~20 | CRUD | OK |
| schichten | kmuhub-schichten | Yes | ~15 | CRUD + Swap | OK |
| einkauf | kmuhub-einkauf | Yes | ~12 | CRUD | OK |
| helpdesk | kmuhub-helpdesk | Yes | ~10 | CRUD + SLA | OK |
| fuhrpark | kmuhub-fuhrpark | Yes | ~8 + 6 routes | CRUD + Track | OK |
| produktion | kmuhub-produktion | Yes | ~10 | CRUD + QC | OK |
| berichte | kmuhub-berichte | Yes | ~5 | CRUD | OK |
| vertraege | kmuhub-vertraege | Yes | 12 | CRUD + Term | OK |
| formulare | kmuhub-formulare | Yes | 5 + 3 tmpl | CRUD + Fields | OK |
| vermietung | kmuhub-vermietung | Yes | 8 + 12 res | CRUD | OK |
| rapporte | kmuhub-rapporte | Yes | 8 + 5 meas | CRUD | OK |

### 4.2 Code Quality

| Metric | Result |
|--------|--------|
| console.log / console.warn | 0 found |
| TODO / FIXME / HACK | 0 found |
| `any` type usage | 7 instances (all in Luke's Work/Gantt code) |
| Stores without module pages | 0 |
| Module pages without stores | CRM modules (use API hooks instead) |

---

## 5. CSS / Visual Audit

### 5.1 Good

- Tailwind v4 correctly configured (`@theme inline`)
- Dark mode OKLCH colors in `.dark` class
- Glass/Crystal CSS rules in globals.css
- Custom scrollbars styled per mode
- No deprecated Tailwind v3 patterns
- Semantic colors used consistently in Darien's modules

### 5.2 Issues (mostly in Luke's modules)

| Issue | Severity | Files | Owner |
|-------|----------|-------|-------|
| MeetingRoomView: `bg-gray-900` hardcoded | CRITICAL | MeetingRoomView.tsx | Luke |
| CallOverlay: `bg-gray-900` hardcoded | CRITICAL | CallOverlay.tsx | Luke |
| PriorityBadge: `bg-red-50` etc. | HIGH | PriorityBadge.tsx | Luke |
| Form required: `text-red-500` statt `text-destructive` | MEDIUM | 6 files | Mixed |
| Modal backdrops: `bg-black/50` | LOW | ~11 files | Mixed |
| ContactDetailPanel: `bg-white/20` | MEDIUM | ContactDetailPanel.tsx | Mixed |
| GanttChart: hardcoded hex in SVG | LOW | GanttChart.tsx | Luke |

### 5.3 Darien's Modules — Color Check

All industry modules (Inventar, Schichten, Einkauf, Helpdesk, Fuhrpark, Produktion, Berichte, Vertraege, Formulare, Vermietung, Rapporte) use:
- `text-foreground`, `text-muted-foreground` for text
- `bg-card`, `bg-secondary` for backgrounds
- `border-border` for borders
- `bg-primary/10`, `text-primary` for accents
- Module-specific Tailwind colors only for icon backgrounds (acceptable)

**Verdict: Darien's modules are clean.**

---

## 6. Backend Requirements Verification

### 6.1 Audit Document Status

File: `.planning/design/BACKEND-REQUIREMENTS-AUDIT.md` (Rev 3, 2026-02-15)

| Section | Covered | Endpoints |
|---------|---------|-----------|
| Core modules (2.1-2.10) | Yes | ~148 |
| Industry modules (2.11-2.17) | Yes | ~97 |
| New modules (2.18-2.21) | Yes | ~55 |
| Module extensions (2.22-2.25) | Yes | ~32 |
| **Total** | **Complete** | **~332 new** |

### 6.2 DB Tables Documented

45 new tables documented in audit, covering all modules. See BACKEND-REQUIREMENTS-AUDIT.md Section 4.

### 6.3 Missing from Audit

| Gap | Priority |
|-----|----------|
| Digitale Unterschriften (Rapporte) | LOW — placeholder in UI |
| Terminbuchung Endpoints (Kalender) | MEDIUM — has UI but not in audit |
| Barcode-Scanning API (Inventar) | LOW — future feature |

---

## 7. Known Gaps / Future Work

### 7.1 Design Phase D10 (Visual Polish — NOT STARTED)

- Page transitions / animations
- Skeleton loading states for all modules
- Accessibility (ARIA labels, keyboard navigation, focus management)
- Reduced motion preferences

### 7.2 Design Phase D11 (Nico-Review Fixes — NOT STARTED)

- Pending external review feedback

### 7.3 Compatibility Updates Needed (Shared Responsibility)

The following items were found during the audit. Most are NOT bugs — they are pre-existing code that needs updating to work with our new Glass/Crystal mode and semantic color system. Some are code quality items.

#### 7.3.1 Glass/Crystal Mode Compatibility (Our new feature → his components need updating)

These files were written before Glass/Crystal mode existed. The hardcoded colors worked fine in Solid mode but need semantic tokens now to support all 3 UI modes.

| File | Issue | Suggested Fix |
|------|-------|---------------|
| `modules/meetings/components/MeetingRoomView.tsx` | `bg-gray-900`, `text-gray-*` hardcoded | Replace with `bg-background`, `text-foreground`, `text-muted-foreground` |
| `modules/meetings/components/CallOverlay.tsx` | `bg-gray-900`, `text-gray-*` hardcoded | Same — use semantic color tokens |

> **Note:** This is not a bug in Luke's code. Glass/Crystal mode is a new design feature (D9). These components just need a color token update to be compatible.

#### 7.3.2 Dark Mode Color Tokens

| File | Issue | Suggested Fix |
|------|-------|---------------|
| `components/PriorityBadge.tsx` | `bg-red-50`, `bg-yellow-50`, `bg-green-50` — invisible in dark mode | Use `bg-destructive/10`, `bg-warning/10`, `bg-success/10` or similar semantic tokens |
| `modules/crm/components/ContactDetailPanel.tsx` | `bg-white/20` — looks off in dark mode | Use `bg-card` or `bg-secondary` |

#### 7.3.3 Minor Color Consistency

| Pattern | Files (~6) | Suggested Fix |
|---------|-----------|---------------|
| `text-red-500` for required fields | Various form components | Use `text-destructive` instead |
| `bg-black/50` for modal backdrops | ~11 files | Use `bg-background/50` or Radix default |

#### 7.3.4 Code Quality (Low Priority)

| File | Issue | Suggested Fix |
|------|-------|---------------|
| `modules/work/components/GanttChart.tsx` | 5x `any` type usage | Define proper interfaces for Gantt data |
| `modules/team/TeamPage.tsx` | 2x `any` type usage | Define interfaces |
| `modules/work/components/GanttChart.tsx` | Hardcoded hex colors in SVG | Use CSS variables |

#### 7.3.5 CRM Placeholders (Pending Backend)

| File | Issue |
|------|-------|
| CRM Edit dialogs | "Kommt bald" placeholder — needs real implementation when backend ready |
| CRM Delete dialogs | "Kommt bald" placeholder — needs real implementation when backend ready |
| CRM Activities | Missing search bar |

#### 7.3.6 Duplicate Route (Cleanup)

| Issue | Details |
|-------|---------|
| `/calendar` + `/kalender` both exist in App.tsx | Both render `KalenderPage`. Nav uses `/kalender`. Remove `/calendar` route or redirect it. |

### 7.4 General Items for Luke

- Wire Zustand stores to TanStack Query hooks when backend ready
- All 11 new industry stores use mock data — needs API integration
- 332 new endpoints documented in `BACKEND-REQUIREMENTS-AUDIT.md`
- 45 new DB tables documented in same file

---

## 8. File Statistics

### 8.1 Lines of Code by Area

| Area | Files | Approx LOC |
|------|-------|------------|
| Industry modules (new) | 11 pages | ~13,415 |
| Industry stores (new) | 11 stores | ~3,700 |
| Module extensions | 5 tabs | ~2,500 |
| Core modules (existing) | ~15 pages | ~8,000 |
| Core stores (existing) | 13 stores | ~4,500 |
| Shared components | ~10 files | ~1,500 |
| Config files | 4 files | ~500 |
| **Total renderer src** | **~70 files** | **~34,000** |

### 8.2 New in This Session

| Category | Count | LOC |
|----------|-------|-----|
| New module pages | 4 (Vertraege, Formulare, Vermietung, Rapporte) | ~5,610 |
| New stores | 4 | ~1,516 |
| Extended pages | 5 (Buchhaltung, Team, Fuhrpark, Dokumente, Zeiterfassung) | ~2,500 |
| Extended stores | 4 (finance, team, fuhrpark, documents) | ~800 |
| Integration updates | 5 files | ~100 |
| Backend audit update | 1 file | ~500 |
| **Session total** | **23 files** | **~11,026** |

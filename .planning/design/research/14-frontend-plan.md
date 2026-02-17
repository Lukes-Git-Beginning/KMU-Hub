# 14 — Frontend Implementation Plan

> KMU Hub Desktop App — Comprehensive build plan for all remaining UI work.
> Author: Darien | Date: 2026-02-17 | Branch: `design/brainstorm`

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Current State Inventory](#2-current-state-inventory)
3. [New Modules](#3-new-modules)
   - 3.1 Kommunikation (Unified External Inbox)
   - 3.2 Wiki (Knowledge Base)
4. [Module Rebuilds and Extensions](#4-module-rebuilds-and-extensions)
   - 4.1 Buchhaltung -> Rechnungen & Finanzen
   - 4.2 Team/HR (Lohn removal + integration panel)
   - 4.3 CRM/Kontakte extensions
   - 4.4 Dokumente extensions
   - 4.5 Chat extensions
   - 4.6 Helpdesk extensions
   - 4.7 Kalender extensions
   - 4.8 Vertraege extension (E-Signatur)
5. [New Shared UI Components](#5-new-shared-ui-components)
   - 5.1 Video/Meeting Room UI
   - 5.2 Status/Presence System
   - 5.3 Global Search (Cmd+K)
   - 5.4 Notification Center (rebuild)
   - 5.5 Integration Settings Panels
   - 5.6 TipTap Rich Text Editor
6. [New Zustand Stores](#6-new-zustand-stores)
7. [Routing Changes](#7-routing-changes)
8. [Navigation Updates](#8-navigation-updates)
9. [Sprint Plan (Buildable Now vs Backend-Dependent)](#9-sprint-plan)
10. [Priority Order and Dependency Graph](#10-priority-order-and-dependency-graph)
11. [LOC Estimates Summary](#11-loc-estimates-summary)
12. [File Structure Overview](#12-file-structure-overview)

---

## 1. Executive Summary

KMU Hub's desktop frontend currently has **28 modules** with full mock data, totaling
roughly 25,000+ lines of TSX/TS across modules, stores, and components. All modules
run on Zustand stores with `localStorage` persistence — no real backend connection yet.

This plan covers the **next wave of frontend work**: 2 brand-new modules, 8 module
rebuilds/extensions, 6 new shared components, 7 new stores, and routing/navigation
updates. The total estimated new code is **~18,000-22,000 LOC**.

### Key decisions

| Decision | Rationale |
|----------|-----------|
| Kommunikation is a NEW module, separate from Mails | Mails = internal email client (IMAP). Kommunikation = unified external inbox (Teams, WhatsApp, Widget, Portal). Different data models, different UX patterns. |
| Wiki is a standalone module, not a Dokumente tab | Wiki needs tree navigation, versioning, templates, and TipTap — fundamentally different from file management. |
| Buchhaltung gets renamed, not replaced | Existing invoice UI is solid. We add tabs/panels, not rebuild from scratch. |
| TipTap editor is a shared component | Reused in Wiki, Helpdesk (canned responses), Chat (rich messages), and Formulare. Build once. |
| All new stores follow existing pattern | `create<State>()(persist((set, get) => ({...}), { name: 'kmuhub-xxx' }))` |
| Mock data in stores, not in components | Same pattern as `mails.ts`, `meetings.ts` — mock arrays at top, store actions below. |

### What Darien can build immediately (no backend needed)

Everything in this document can be built with mock data except the items explicitly
marked as `BACKEND-DEPENDENT` in Section 9. The mock-first approach means Luke can
wire up real API calls later by replacing store actions with `react-query` mutations.

---

## 2. Current State Inventory

### Existing modules (in `desktop/src/renderer/src/modules/`)

| Module | Route | Main file LOC | Store | Sub-components |
|--------|-------|---------------|-------|----------------|
| Dashboard | `/` | ~400 | `dashboard.ts` | widgets/ |
| CRM | `/crm/*` | Layout: 81 | `contacts.ts` | sub-routes |
| Chat | `/chat/*` | Layout: 81 | (inline) | channels/, messages/, threads/ (1091 total) |
| Work (Projects+Tasks) | `/work/*` | Layout | `work.ts` | sub-routes |
| Kalender | `/kalender` | 2143 | `calendar.ts` | Terminbuchung tab |
| Kontakte | `/kontakte` | 500 | `contacts.ts` | DetailPanel, FormDialog, GroupManager, Import |
| Mails | `/mails` | 553 | `mails.ts` (337) | ComposeInline, ComposeModal, ComposeWindow |
| Meetings | `/meetings` | 617 | `meetings.ts` (421) | — |
| Team | `/team` | 1212 | `team.ts` | AbsenceCalendar, EditMember, HRApproval, etc. |
| Buchhaltung | `/buchhaltung` | 675 | `finance.ts` | InvoiceDetail, InvoiceForm, ExpenseForm, Export |
| Dokumente | `/dokumente` | 1406 | `documents.ts` | — |
| Zeiterfassung | `/zeiterfassung` | ~800 | `timetracking.ts` | — |
| Berichte | `/berichte` | 921 | `berichte.ts` | — |
| Inventar | `/inventar` | 1038 | `inventar.ts` | — |
| Schichten | `/schichten` | 1096 | `schichten.ts` | — |
| Einkauf | `/einkauf` | 1209 | `einkauf.ts` | — |
| Helpdesk | `/helpdesk` | 1008 | `helpdesk.ts` | — |
| Fuhrpark | `/fuhrpark` | 1334 | `fuhrpark.ts` | — |
| Produktion | `/produktion` | 1199 | `produktion.ts` | — |
| Vertraege | `/vertraege` | 1234 | `vertraege.ts` | — |
| Formulare | `/formulare` | 1493 | `formulare.ts` | — |
| Vermietung | `/vermietung` | 1412 | `vermietung.ts` | — |
| Rapporte | `/rapporte` | 1471 | `rapporte.ts` | — |
| Notifications | `/notifications` | 391 | (inline) | — |
| Settings | `/settings` | ~900 | `settings.ts` | tabs |
| Profil | `/profil` | ~400 | `auth.ts` | — |
| Admin/Infrastruktur | `/infrastruktur` | ~600 | — | — |

### Existing stores (in `stores/`)

28 stores total. All use Zustand with `persist` middleware and `localStorage`.
Key pattern: mock data defined as const arrays, store wraps CRUD actions.

### Existing shared components (in `components/`)

- **UI primitives** (21): button, card, dialog, dropdown-menu, input, select, tabs, table, badge, avatar, checkbox, label, popover, scroll-area, separator, sheet, skeleton, switch, textarea, tooltip, alert-dialog
- **Shared** (5): ConfirmDialog, DetailPanel, EmptyState, FormField, ItemActions
- **Layout** (7): AppShell, DeskEnvironment, DeskFrame, Header, ModuleShell, OfflineBanner, sidebar/

### Existing config

- `business-profiles.ts` — 10 profiles with module visibility rules
- `desk-themes.ts` — 6 desk themes
- `desk-asset-urls.ts` — Decoration assets
- `roles.ts` — Dev profiles for role switching

---

## 3. New Modules

### 3.1 Kommunikation (Unified External Inbox)

**Purpose:** Single inbox for ALL external customer communication channels. This is
separate from Chat (internal team messaging) and Mails (email client). Kommunikation
is the unified view where support/sales staff see conversations from E-Mail, Microsoft
Teams bridge, WhatsApp Business, the website widget, and the customer portal.

**Route:** `/kommunikation`

**Why separate from Mails?** The Mails module is a traditional email client with
folders (Inbox, Sent, Drafts, Spam, Trash). Kommunikation is a conversation-centric
view where the same customer might reach out via email AND WhatsApp, and the agent
sees a unified thread. Different data model, different UX.

#### Component hierarchy

```
KommunikationPage.tsx
├── ChannelTabs.tsx                    # [E-Mail] [Teams] [WhatsApp] [Widget] [Portal] [Alle]
├── ConversationList.tsx               # Left panel — filterable conversation list
│   ├── ConversationListHeader.tsx     # Search, filter, sort controls
│   ├── ConversationListItem.tsx       # Avatar, name, preview, channel icon, time, unread dot
│   └── ConversationListFilters.tsx    # Status (offen/wartend/geloest), assigned-to, priority
├── ConversationThread.tsx             # Center panel — message thread
│   ├── ConversationThreadHeader.tsx   # Contact name, channel, status, assign, actions
│   ├── MessageTimeline.tsx            # Messages with timestamps, sender indicators
│   │   └── MessageItem.tsx            # Individual message (text, attachments, system note)
│   ├── ReplyComposer.tsx              # Reply input with rich text, attachments, templates
│   │   ├── CannedResponsePicker.tsx   # Quick insert pre-written responses
│   │   └── InsertFromCRMButton.tsx    # Pull documents/offers from CRM
│   └── InternalNoteComposer.tsx       # Internal notes (not visible to customer)
├── ContextPanel.tsx                   # Right panel — CRM context
│   ├── ContactCard.tsx                # Matched CRM contact info
│   ├── OpenDeals.tsx                  # Active deals for this contact
│   ├── OpenTickets.tsx                # Active helpdesk tickets
│   ├── RelatedProjects.tsx            # Projects this contact is linked to
│   └── ActivityTimeline.tsx           # Recent interactions across all channels
├── NewConversationDialog.tsx          # Start new outbound conversation
└── ChannelSettingsDialog.tsx          # Per-channel configuration
```

#### Wireframe (text-based)

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│  [E-Mail] [Teams] [WhatsApp] [Widget] [Portal] [Alle]                         │
├──────────────────┬───────────────────────────────────┬──────────────────────────┤
│ ◉ Search...      │ ← Thomas Weber                   │ CRM-Kontakt             │
│ Filters ▾  Sort ▾│ #E-Mail · Offen · Zugewiesen: DK │ ┌──────────────────────┐│
│──────────────────│───────────────────────────────────│ │ Thomas Weber         ││
│ ● Thomas Weber   │                                   │ │ ABC GmbH             ││
│   RE: Angebot... │ [Thomas Weber]     14:30          │ │ thomas@abc-gmbh.ch   ││
│   vor 2 Std ·  ✉│ Vielen Dank fuer das Angebot.     │ │ +41 44 123 45 67     ││
│──────────────────│ Wir sind grundsaetzlich           │ └──────────────────────┘│
│   Sarah Klein    │ einverstanden. Koennten wir...    │                          │
│   Design Review  │                                   │ Offene Deals             │
│   vor 5 Std ·  ✉│                                   │ ● CRM Phase 2  CHF 48k  │
│──────────────────│ [Du]               14:45          │ ● Support SLA  CHF 12k  │
│   Eva Brunner    │ Sehr geehrter Herr Weber,         │                          │
│   DSGVO Gutach...│ bzgl. der Schulungskosten: wir    │ Offene Tickets           │
│   vor 4 Tage · ✉│ koennen das Paket auf...          │ #1042 Login-Problem      │
│──────────────────│                                   │ #1038 API Timeout        │
│                  │                                   │                          │
│                  │                                   │ Letzte Aktivitaet        │
│                  │                                   │ · Anruf 12.02 (15 min)  │
│                  │                                   │ · Meeting 10.02         │
│                  │───────────────────────────────────│ · E-Mail 09.02          │
│                  │ [Antworten]  [Interne Notiz]      │                          │
│                  │ ┌─────────────────────────────┐   │                          │
│                  │ │ Nachricht eingeben...       │   │                          │
│                  │ │                             │   │                          │
│                  │ │  📎  📋  🔗CRM  Senden ➤   │   │                          │
│                  │ └─────────────────────────────┘   │                          │
└──────────────────┴───────────────────────────────────┴──────────────────────────┘
```

**Layout:** 3-column with resizable panels. Left ~280px (conversation list),
center flex-1 (thread), right ~300px (context, collapsible).

#### Store: `stores/communication.ts`

```typescript
interface CommunicationState {
  // Data
  conversations: Conversation[]
  messages: Record<string, Message[]>        // conversationId -> messages
  cannedResponses: CannedResponse[]

  // UI state
  activeChannel: ChannelType | 'all'
  activeConversationId: string | null
  searchQuery: string
  filterStatus: 'all' | 'open' | 'waiting' | 'resolved'
  filterAssignee: string | null
  contextPanelOpen: boolean

  // Actions
  setActiveChannel: (channel: ChannelType | 'all') => void
  setActiveConversation: (id: string | null) => void
  sendReply: (conversationId: string, body: string, attachments?: Attachment[]) => void
  addInternalNote: (conversationId: string, note: string) => void
  assignConversation: (conversationId: string, userId: string) => void
  changeStatus: (conversationId: string, status: ConversationStatus) => void
  // ... CRUD for canned responses
}
```

**Types:**

```typescript
type ChannelType = 'email' | 'teams' | 'whatsapp' | 'widget' | 'portal'
type ConversationStatus = 'open' | 'waiting' | 'resolved' | 'spam'

interface Conversation {
  id: string
  channel: ChannelType
  contactId: string | null         // matched CRM contact
  contactName: string
  contactEmail: string
  subject: string
  lastMessage: string
  lastMessageAt: string
  status: ConversationStatus
  assignedTo: string | null
  priority: 'low' | 'normal' | 'high' | 'urgent'
  unreadCount: number
  tags: string[]
}

interface Message {
  id: string
  conversationId: string
  type: 'inbound' | 'outbound' | 'internal_note' | 'system'
  senderName: string
  body: string
  attachments: Attachment[]
  timestamp: string
  channel: ChannelType
}
```

**Mock data:** 12-15 conversations across all channels, 3-8 messages each.

**Estimated LOC:**
- `KommunikationPage.tsx` + sub-components: ~1800
- `stores/communication.ts`: ~450
- Total: **~2250**

#### Auto CRM contact matching (mock)

In the mock store, conversations have a `contactId` that maps to contacts in the
contacts store. The ContextPanel reads from `useContactsStore` to display matched
contact info. In production, Luke's backend will do fuzzy matching on email/phone.

#### "Insert from CRM" button

Opens a dialog listing recent Angebote, Rechnungen, and Dokumente linked to the
matched contact. Clicking inserts a formatted link/preview into the reply composer.
Mock version pulls from `useFinanceStore` and `useDocumentsStore`.

---

### 3.2 Wiki (Knowledge Base)

**Purpose:** Internal knowledge base with rich-text articles. Tree-structured
categories, full-text search, version history, and article templates.

**Route:** `/wiki`

#### Component hierarchy

```
WikiPage.tsx
├── WikiSidebar.tsx                    # Left panel — tree navigation
│   ├── WikiTreeNode.tsx               # Expandable tree item (category/article)
│   ├── WikiSearch.tsx                 # Full-text search input + results
│   └── WikiNewButton.tsx              # "Neue Seite" / "Neue Kategorie"
├── WikiArticle.tsx                    # Center panel — article content
│   ├── WikiArticleHeader.tsx          # Title, breadcrumb, last edited, author
│   ├── WikiEditor.tsx                 # TipTap rich text editor (edit mode)
│   ├── WikiRenderer.tsx              # Read-only rendered view
│   └── WikiArticleFooter.tsx          # Tags, related articles
├── WikiVersionHistory.tsx             # Right panel (toggle) — version diffs
│   └── WikiVersionItem.tsx            # Individual version entry
├── WikiTemplateDialog.tsx             # "New from template" dialog
├── WikiCategoryDialog.tsx             # Create/edit category
└── WikiShareDialog.tsx                # Share settings (team-wide, role-based)
```

#### Wireframe (text-based)

```
┌──────────────────────────────────────────────────────────────────────────────┐
│  Wiki    [+ Neue Seite]  [🔍 Suche...]                         [Bearbeiten]│
├─────────────────┬────────────────────────────────────────────────────────────┤
│ 📂 Onboarding   │ Onboarding > Erste Schritte                               │
│  ├ Erste Schritte│─────────────────────────────────────────────────────────── │
│  ├ IT-Setup     │                                                            │
│  └ HR-Prozess   │ # Erste Schritte fuer neue Mitarbeiter                     │
│ 📂 Produkt      │                                                            │
│  ├ Features     │ Willkommen bei KMU Hub! Diese Seite hilft dir beim         │
│  ├ Roadmap      │ Einstieg in dein neues Team.                               │
│  └ API Docs     │                                                            │
│ 📂 Prozesse     │ ## 1. Accounts einrichten                                  │
│  ├ Support-Flow │                                                            │
│  ├ Release-Flow │ - [ ] E-Mail-Konto aktivieren                              │
│  └ Deployment   │ - [ ] KMU Hub Login erhalten                               │
│ 📂 Templates    │ - [ ] VPN-Zugang konfigurieren                             │
│  ├ Meeting Notes│                                                            │
│  └ Post-Mortem  │ ## 2. Erste Woche                                          │
│                 │                                                            │
│                 │ | Tag | Aufgabe            | Kontakt      |                │
│                 │ |-----|--------------------|--------------|                │
│                 │ | Mo  | Buero-Tour         | Anna Mueller |                │
│                 │ | Di  | Tool-Einfuehrung   | Jonas Diaz   |                │
│                 │ | Mi  | Projekt-Briefing   | Michael Berg |                │
│                 │                                                            │
│                 │ > **Tipp:** Nutze Cmd+K fuer die Schnellsuche.             │
│                 │                                                            │
│                 │──────────────────────────────────────────────────────────── │
│                 │ Tags: onboarding, hr, neu    Zuletzt bearbeitet: 12.02.26  │
└─────────────────┴────────────────────────────────────────────────────────────┘
```

**Layout:** 2-column. Left ~240px (tree), center flex-1 (article). Version history
is a slide-in panel from the right (using `DetailPanel` component).

#### Store: `stores/wiki.ts`

```typescript
interface WikiState {
  categories: WikiCategory[]
  articles: WikiArticle[]
  versions: Record<string, WikiVersion[]>   // articleId -> versions
  templates: WikiTemplate[]

  // UI
  activeArticleId: string | null
  expandedCategories: string[]
  searchQuery: string
  searchResults: WikiArticle[]
  editMode: boolean

  // Actions
  setActiveArticle: (id: string | null) => void
  toggleCategory: (id: string) => void
  createArticle: (article: Omit<WikiArticle, 'id' | 'createdAt' | 'updatedAt'>) => void
  updateArticle: (id: string, updates: Partial<WikiArticle>) => void
  deleteArticle: (id: string) => void
  createCategory: (category: Omit<WikiCategory, 'id'>) => void
  reorderArticle: (articleId: string, newCategoryId: string, newIndex: number) => void
  searchArticles: (query: string) => void
  toggleEditMode: () => void
}
```

**Types:**

```typescript
interface WikiCategory {
  id: string
  name: string
  parentId: string | null       // nested categories
  order: number
  icon: string                  // lucide icon name
}

interface WikiArticle {
  id: string
  categoryId: string
  title: string
  content: string               // TipTap JSON or HTML
  tags: string[]
  authorId: string
  authorName: string
  createdAt: string
  updatedAt: string
  pinned: boolean
}

interface WikiVersion {
  id: string
  articleId: string
  content: string
  authorName: string
  timestamp: string
  changeNote: string
}

interface WikiTemplate {
  id: string
  name: string
  description: string
  content: string
}
```

**Mock data:** 4 categories, 12 articles, 3 templates (Meeting Notes, Post-Mortem, How-To).

**Estimated LOC:**
- `WikiPage.tsx` + sub-components: ~1400
- `stores/wiki.ts`: ~350
- Total: **~1750**

---

## 4. Module Rebuilds and Extensions

### 4.1 Buchhaltung -> "Rechnungen & Finanzen"

**Current state:** `BuchhaltungPage.tsx` (675 LOC) + 4 sub-dialogs. Tab-based layout
with Rechnungen, Ausgaben, Uebersicht tabs.

**Changes:**

1. **Rename** — Label in `nav-items.ts` from "Buchhaltung" to "Rechnungen & Finanzen".
   Route stays `/buchhaltung` (no breaking change needed, rename is cosmetic).

2. **Remove FiBu features** — Remove any Finanzbuchhaltung references, Kontenrahmen,
   Bilanz/GuV sections. We never build a full FiBu. We are an invoicing tool.

3. **New tab: Belegkette** — Document chain visualization.
   ```
   Angebot → Auftrag → Lieferschein → Rechnung → Mahnung
   ```
   Visual pipeline with status indicators per document. Click a step to see linked
   documents. "Neues Angebot" starts the chain; each step can "convert to next".

4. **New tab: Exporte & Integrationen** — DATEV export panel + Bexio sync dashboard.
   - DATEV: Date range picker, account mapping table, "Export starten" button, export history
   - Bexio: Connection status indicator, last sync timestamp, sync button, conflict list

5. **QR-Rechnung preview** — In `InvoiceDetailPanel.tsx`, add a mock Swiss QR-bill
   rendering below the invoice. Shows QR code placeholder, payment slip fields
   (IBAN, reference, amount).

6. **ZUGFeRD indicator** — Badge on invoices indicating ZUGFeRD compliance level
   (Basic, Comfort, Extended). Tooltip explains what it means.

7. **MWSt multi-country** — In invoice form, country selector (DE/AT/CH) auto-sets
   tax rates: DE 19%/7%, AT 20%/10%/13%, CH 8.1%/2.6%.

#### New/modified files

| File | Action | Est. LOC change |
|------|--------|-----------------|
| `BuchhaltungPage.tsx` | Add 2 tabs, restructure | +350 |
| `BelegketteTab.tsx` | NEW — pipeline visualization | +500 |
| `ExporteTab.tsx` | NEW — DATEV + Bexio panels | +450 |
| `QRRechnungPreview.tsx` | NEW — Swiss QR-bill mock | +180 |
| `InvoiceFormDialog.tsx` | Add MWSt country selector | +80 |
| `InvoiceDetailPanel.tsx` | Add QR preview + ZUGFeRD badge | +120 |
| `stores/finance.ts` | Add Belegkette types, MWSt config | +150 |
| `nav-items.ts` | Rename label | +1 |

**Total new LOC: ~1830**

#### Belegkette wireframe

```
┌───────────────────────────────────────────────────────────────────────────┐
│  Belegkette                                                    [+ Neu]   │
│─────────────────────────────────────────────────────────────────────────── │
│                                                                          │
│  🟢 Angebot ──→ 🟢 Auftrag ──→ ⚪ Lieferschein ──→ ⚪ Rechnung ──→ ⚪ Mahnung │
│  #A-2024-042    #AU-2024-038                                             │
│  CHF 48'000     CHF 48'000                                               │
│  12.01.2026     15.01.2026     (nicht erstellt)     (nicht erstellt)      │
│                                                                          │
│  [In Auftrag umwandeln]  [Lieferschein erstellen]                        │
│                                                                          │
│─────────────────────────────────────────────────────────────────────────── │
│  Letzte Belegketten                                                      │
│                                                                          │
│  ABC GmbH — CRM Phase 2         🟢🟢🟢🟢⚪  4/5 Schritte                │
│  Meier AG — Wartungsvertrag      🟢🟢🟢⚪⚪  3/5 Schritte                │
│  TechVentures — Demo-Paket      🟢⚪⚪⚪⚪  1/5 Schritte                │
└───────────────────────────────────────────────────────────────────────────┘
```

---

### 4.2 Team/HR (Lohn removal + integration panel)

**Current state:** `TeamPage.tsx` (1212 LOC) with tabs: Mitarbeiter, Urlaub,
Abwesenheiten, Schulungen, Lohn.

**Changes:**

1. **Remove Lohn tab entirely.** We never build payroll. It is handled by DATEV Lohn
   or Abacus HR — external systems.

2. **Add "Integrationen" tab** in place of Lohn:
   - DATEV Lohn card: Connection status, employee sync count, last sync, "Konfigurieren" link
   - Abacus HR card: Same pattern
   - Generic "Weitere HR-Systeme" placeholder with "Kontakt aufnehmen" CTA

3. **Keep:** Mitarbeiter list, Urlaub planner, Abwesenheiten calendar, Schulungen.

#### Modified files

| File | Action | Est. LOC change |
|------|--------|-----------------|
| `TeamPage.tsx` | Remove Lohn tab, add Integrationen tab | -200, +250 |
| `HRIntegrationPanel.tsx` | NEW — DATEV/Abacus connection cards | +280 |
| `stores/team.ts` | Remove Lohn mock data, add integration status | +40 / -80 |

**Net new LOC: ~290**

---

### 4.3 CRM/Kontakte Extensions

**Current state:** `KontaktePage.tsx` (500 LOC) + ContactDetailPanel, ContactFormDialog,
GroupManagerDialog, ImportContactsDialog.

**Changes:**

#### 4.3.1 Custom Fields UI

Admin-configurable custom fields stored as JSONB in the backend (when connected).
For now, mock data. Field types: text, number, date, dropdown, checkbox, URL.

```
CustomFieldsConfig.tsx              # Settings tab — admin manages field definitions
├── CustomFieldRow.tsx              # Single field config (name, type, required, default)
└── CustomFieldPreview.tsx          # Live preview of the field
```

In `ContactDetailPanel.tsx` and `ContactFormDialog.tsx`, render custom fields
dynamically from the field definitions in the store.

- `CustomFieldsConfig.tsx`: ~300 LOC
- Changes to existing components: ~150 LOC
- Store additions: ~100 LOC

#### 4.3.2 Firma (Company) as own entity

Currently contacts are flat (person-level). Companies need their own detail page.

```
FirmaDetailPage.tsx                 # Full-page company detail
├── FirmaHeader.tsx                 # Company name, logo, industry, website
├── FirmaContactList.tsx            # Employees at this company
├── FirmaDeals.tsx                  # Linked deals
├── FirmaTimeline.tsx               # Activity timeline
└── FirmaCustomFields.tsx           # Company-level custom fields
```

Route: either a sub-route of `/kontakte` (e.g., `/kontakte/firma/:id`) or rendered
inline as a wide detail panel. Recommended: detail panel first (consistent with
existing pattern), full page later if needed.

- `FirmaDetailPanel.tsx`: ~500 LOC
- Store additions (`contacts.ts`): ~120 LOC

#### 4.3.3 Duplikaterkennung dialog

When creating or importing contacts, check for duplicates by email, phone, or name
similarity. Show a dialog with potential matches and merge options.

```
DuplicateDetectionDialog.tsx
├── DuplicateMatchCard.tsx          # Side-by-side comparison
└── MergeFieldSelector.tsx          # Pick which field value to keep
```

- `DuplicateDetectionDialog.tsx` + children: ~350 LOC

#### 4.3.4 Newsletter panel

Integration panel for Brevo or CleverReach. Shows:
- Connection status
- Subscriber lists synced from CRM
- Send history
- "Sync contacts to Brevo" button

- `NewsletterPanel.tsx`: ~250 LOC

**Total new LOC for CRM: ~1770**

---

### 4.4 Dokumente Extensions

**Current state:** `DokumentePage.tsx` (1406 LOC). File browser with folders, upload,
preview, tags.

**Changes:**

#### 4.4.1 OnlyOffice webview integration

Placeholder iframe for opening .docx/.xlsx/.pptx files inline. In mock mode, shows
a styled placeholder "OnlyOffice wird geladen..." with a loading animation. The real
WOPI endpoint URL will come from Luke's backend.

- `OnlyOfficeViewer.tsx`: ~200 LOC (iframe wrapper with loading/error states)
- Changes to `DokumentePage.tsx`: ~50 LOC (button to open viewer)

#### 4.4.2 Template gallery

"Neu aus Vorlage" dialog with categorized document templates.

```
TemplateGalleryDialog.tsx
├── TemplateCard.tsx                # Preview card with icon, name, description
└── TemplateCategoryFilter.tsx      # Category tabs (Vertraege, Briefe, Formulare, ...)
```

- `TemplateGalleryDialog.tsx` + children: ~350 LOC
- Store additions: ~80 LOC (template definitions)

#### 4.4.3 External link sharing UI

Dialog to generate a shareable link for a document. Options: expiry date, password
protection, view-only vs download.

- `ShareLinkDialog.tsx`: ~200 LOC

#### 4.4.4 Wiki link

Add a navigation link from Dokumente to the Wiki module. Could be a tab or a
prominent button in the toolbar. Since Wiki is its own module, a simple
"Zum Wiki" button with router navigation is sufficient.

- Changes to `DokumentePage.tsx`: ~15 LOC

**Total new LOC for Dokumente: ~895**

---

### 4.5 Chat Extensions

**Current state:** 1091 LOC total across ChatLayout, channels/, messages/, threads/.
Basic channel list, message list, message input, thread panel.

**Changes:**

#### 4.5.1 Thread replies

Enhance `ThreadPanel.tsx` (currently 88 LOC) to show full threaded conversations.
Add "Reply in thread" button to each message. Thread panel slides in from the right.

- Enhance `ThreadPanel.tsx`: +200 LOC
- Enhance `MessageBubble.tsx`: +40 LOC (thread reply count indicator)

#### 4.5.2 Reactions

Emoji reaction bar below messages. Click to add reaction, click again to remove.
Reaction picker (small emoji grid) on hover/click of "+" button.

- `ReactionBar.tsx`: ~120 LOC
- `ReactionPicker.tsx`: ~150 LOC
- Changes to `MessageBubble.tsx`: +30 LOC

#### 4.5.3 @mentions with autocomplete

In `MessageInput.tsx`, typing `@` triggers an autocomplete dropdown with team
members. Selected mention becomes a styled chip in the input.

- `MentionAutocomplete.tsx`: ~200 LOC
- Changes to `MessageInput.tsx`: +60 LOC

#### 4.5.4 File sharing (drag & drop)

Drag files onto the message area to upload. Show preview thumbnails for images,
file cards for documents.

- `FileDropZone.tsx`: ~100 LOC
- `FileAttachmentCard.tsx`: ~80 LOC
- Changes to `MessageInput.tsx`: +40 LOC
- Changes to `MessageBubble.tsx`: +50 LOC

#### 4.5.5 Status/presence indicators

Online/away/busy/DND dots next to user names in channel member list and message
bubbles. Uses the shared presence store (Section 6).

- Changes across chat components: +80 LOC

**Total new LOC for Chat: ~1150**

---

### 4.6 Helpdesk Extensions

**Current state:** `HelpdeskPage.tsx` (1008 LOC). Ticket list, ticket detail panel,
basic reply functionality.

**Changes:**

#### 4.6.1 Canned responses library

Pre-written response templates for common questions. Managed in a settings-like
panel within Helpdesk. Insertable via dropdown in the reply composer.

```
CannedResponsesPanel.tsx
├── CannedResponseList.tsx          # List with search
├── CannedResponseEditor.tsx        # Create/edit response (uses TipTap)
└── CannedResponsePicker.tsx        # Dropdown in reply composer
```

- Total: ~400 LOC

#### 4.6.2 Private notes

Internal notes on tickets that the customer cannot see. Visually distinct from
public replies (different background color, "Interne Notiz" label).

- Changes to `HelpdeskPage.tsx`: +80 LOC (note input, note display)

#### 4.6.3 Business hours configuration

Settings dialog where admins configure business hours per day, holidays, and
timezone. SLA calculations reference these hours.

- `BusinessHoursDialog.tsx`: ~300 LOC
- Store additions: ~80 LOC

#### 4.6.4 SLA indicators

Visual indicators on tickets showing SLA status: time remaining, overdue warning,
breach alert. Color-coded badges (green/yellow/red).

- `SLABadge.tsx`: ~60 LOC
- Changes to `HelpdeskPage.tsx`: +50 LOC

**Total new LOC for Helpdesk: ~970**

---

### 4.7 Kalender Extensions

**Current state:** `KalenderPage.tsx` (2143 LOC). Full calendar with month/week/day
views, Terminbuchung tab, event creation.

**Changes:**

#### 4.7.1 Video meeting integration

"Start Meeting" button on calendar events that have `isVideoCall: true`. For Starter
tier, generates a Zoom-style link (mock). For Business+ tier, opens LiveKit room
(mock — shell UI only, real integration needs Luke).

- Changes to `KalenderPage.tsx`: +80 LOC (button + meeting link display)
- Integration with `stores/meetings.ts`: +30 LOC

#### 4.7.2 Meeting link generation

When creating a new event, toggle "Video-Meeting" adds a meeting link field.
Mock generates a `https://meet.kmuhub.ch/abc-def-ghi` style link.

- Changes to event creation dialog: +60 LOC

**Total new LOC for Kalender: ~170**

---

### 4.8 Vertraege Extension (E-Signatur)

**Current state:** `VertraegePage.tsx` (1234 LOC). Contract list, detail panel, status
tracking.

**Changes:**

#### E-Signatur dialog (Skribble integration mock)

Dialog that simulates sending a contract for e-signature via Skribble. Shows:
- Signer list (add signers by email)
- Signing order (sequential or parallel)
- Reminder settings
- Status tracking (sent, viewed, signed, declined)

```
ESignaturDialog.tsx
├── SignerList.tsx                   # List of signers with status indicators
├── SigningOrderConfig.tsx           # Sequential vs parallel toggle
└── SignatureStatusTimeline.tsx      # Visual timeline of signature events
```

- `ESignaturDialog.tsx` + children: ~450 LOC
- Changes to `VertraegePage.tsx`: +50 LOC (button to open dialog, status badge)
- Store additions: ~100 LOC

**Total new LOC for Vertraege: ~600**

---

## 5. New Shared UI Components

### 5.1 Video/Meeting Room UI

**Location:** `components/shared/VideoMeeting/`

A meeting room component for LiveKit video calls. In mock mode, shows the UI shell
without actual video streams.

```
VideoMeetingRoom.tsx
├── VideoGrid.tsx                    # Grid of participant video tiles
│   └── VideoTile.tsx                # Single participant (avatar/name when no video)
├── MeetingControls.tsx              # Bottom bar: mute, camera, share, chat, leave
├── MeetingParticipantList.tsx       # Sidebar — participant list with status
├── MeetingSidebar.tsx               # Collapsible sidebar (participants, chat, settings)
│   └── MeetingChat.tsx              # In-meeting text chat
├── ScreenShareOverlay.tsx           # Full-screen shared screen view
└── MeetingEndDialog.tsx             # "Meeting beenden" confirmation
```

#### Wireframe

```
┌──────────────────────────────────────────────────────────────┐
│  Sprint Planning Q1                              [X Verlassen]│
├──────────────────────────────────────────┬───────────────────┤
│                                          │ Teilnehmer (4)    │
│   ┌──────────┐  ┌──────────┐            │                   │
│   │          │  │          │            │ 🟢 Anna Mueller   │
│   │  Anna M. │  │ Michael  │            │ 🟢 Michael Berg   │
│   │  (Video) │  │  (Video) │            │ 🟡 Sarah Klein    │
│   │          │  │          │            │ 🔴 Jonas Diaz     │
│   └──────────┘  └──────────┘            │                   │
│                                          │─────────────────── │
│   ┌──────────┐  ┌──────────┐            │ Meeting-Chat      │
│   │          │  │          │            │                   │
│   │  Sarah   │  │  Jonas   │            │ Anna: Link ist im │
│   │  (Muted) │  │  (Away)  │            │ Chat geteilt.     │
│   │          │  │          │            │                   │
│   └──────────┘  └──────────┘            │ [Nachricht...]    │
│                                          │                   │
├──────────────────────────────────────────┴───────────────────┤
│     [🎤 Mute]  [📷 Kamera]  [🖥 Teilen]  [💬 Chat]  [🔴 Beenden] │
└──────────────────────────────────────────────────────────────┘
```

**Estimated LOC:** ~800

**Dependencies:** Uses `stores/meeting.ts` (exists), extended with participant
video/audio state. In mock mode, tiles show avatars with initials.

---

### 5.2 Status/Presence System

**Location:** `components/shared/Presence/`

User online/offline/away/busy/DND status indicators. Used across Chat, Meetings,
Kontakte, Team.

```
StatusDot.tsx                        # Colored dot component (green/yellow/orange/red/gray)
StatusPicker.tsx                     # Dropdown to set own status
SetStatusDialog.tsx                  # Custom status with emoji + text + duration
UserPresenceProvider.tsx             # Context provider (wraps app, manages heartbeat)
```

#### Status values

| Status | Color | Dot | Meaning |
|--------|-------|-----|---------|
| online | green | 🟢 | Active in last 5 min |
| away | yellow | 🟡 | Idle > 5 min |
| busy | orange | 🟠 | In a meeting or DND time block |
| dnd | red | 🔴 | Do Not Disturb — no notifications |
| offline | gray | ⚫ | Not logged in |

**Estimated LOC:** ~350

**Store:** `stores/presence.ts` (~120 LOC)

---

### 5.3 Global Search (Cmd+K)

**Location:** `components/shared/GlobalSearch/`

Command palette / spotlight search. Opens with Cmd+K (Mac) or Ctrl+K (Windows).
Searches across all modules: contacts, projects, tasks, documents, wiki articles,
calendar events, emails, conversations.

```
GlobalSearchDialog.tsx               # Modal overlay with search input
├── SearchInput.tsx                  # Auto-focus input with icon
├── SearchResultGroup.tsx            # Grouped results by module
│   └── SearchResultItem.tsx         # Single result with icon, title, subtitle, module badge
├── RecentSearches.tsx               # Last 5 searches (stored in localStorage)
└── QuickActions.tsx                 # "Neuer Kontakt", "Neues Projekt", etc.
```

#### Wireframe

```
┌──────────────────────────────────────────────────┐
│  🔍 Suche...                              Esc    │
├──────────────────────────────────────────────────┤
│                                                  │
│  Kontakte                                        │
│  👤 Thomas Weber — ABC GmbH          [Kontakte]  │
│  👤 Thomas Mueller — XYZ AG          [Kontakte]  │
│                                                  │
│  Projekte                                        │
│  📁 Website Relaunch                 [Projekte]  │
│                                                  │
│  Dokumente                                       │
│  📄 Angebot_Phase2.pdf              [Dokumente]  │
│                                                  │
│  Kalender                                        │
│  📅 Kundenpraesentation Meier AG    [Kalender]   │
│                                                  │
│  Schnellaktionen                                 │
│  ➕ Neuer Kontakt    ➕ Neues Projekt              │
│  ➕ Neue Aufgabe     ➕ Neues Dokument             │
└──────────────────────────────────────────────────┘
```

**Keyboard navigation:** Arrow up/down to select, Enter to navigate, Esc to close.

**Estimated LOC:** ~550

**Store:** `stores/search.ts` (~100 LOC — recent searches, result caching)

**Integration:** Searches across all existing stores (contacts, work, documents,
calendar, mails, wiki, etc.) using simple string matching on titles/names.
Backend will provide federated search API later.

---

### 5.4 Notification Center (rebuild)

**Current state:** `NotificationCenter.tsx` (391 LOC). Basic page with list.

**Target:** Bell icon in header with dropdown, plus full page view. OS-native push
notifications for critical alerts.

```
NotificationBell.tsx                 # Header icon with unread count badge
NotificationDropdown.tsx             # Dropdown with last 10 notifications
├── NotificationItem.tsx             # Single notification (icon, text, time, actions)
└── NotificationGroupHeader.tsx      # "Heute", "Gestern", "Aelter"
NotificationCenter.tsx               # Full page — enhanced version
NotificationSettingsPanel.tsx        # Per-category notification preferences
```

**Notification types:**
- Neue Nachricht (Chat)
- Neue E-Mail
- Aufgabe zugewiesen
- Aufgabe faellig
- Meeting in 15 Min
- Neuer Helpdesk-Ticket
- Vertrag laeuft aus
- Rechnung ueberfaellig
- Erwähnung (@mention)
- Systembenachrichtigung

**Estimated LOC:**
- Components: ~500
- Store `stores/notifications.ts`: ~200
- Total: ~700

---

### 5.5 Integration Settings Panels

**Location:** `modules/settings/integrations/`

Per-integration configuration UIs, all accessible from the Settings page under a new
"Integrationen" tab.

```
IntegrationSettingsTab.tsx           # Grid of integration cards
├── IntegrationCard.tsx              # Card with logo, name, status, "Konfigurieren"
├── DATEVConfigPanel.tsx             # DATEV export settings
├── BexioConfigPanel.tsx             # Bexio sync settings
├── BrevoConfigPanel.tsx             # Brevo newsletter settings
├── SkribbleConfigPanel.tsx          # Skribble e-signature settings
├── OnlyOfficeConfigPanel.tsx        # OnlyOffice WOPI settings
├── ZoomConfigPanel.tsx              # Zoom OAuth settings
├── LiveKitConfigPanel.tsx           # LiveKit server settings
├── TeamsConfigPanel.tsx             # Microsoft Teams bridge settings
├── WhatsAppConfigPanel.tsx          # WhatsApp Business API settings
└── GenericIntegrationPanel.tsx      # Reusable panel template
```

Each panel follows the same pattern:
1. Connection status indicator (green connected / gray disconnected)
2. Credentials input (API key, OAuth button, server URL)
3. Sync settings (auto-sync interval, scope)
4. Test connection button
5. Activity log (last 5 sync events)

**Estimated LOC:**
- `IntegrationSettingsTab.tsx` + `IntegrationCard.tsx`: ~250
- `GenericIntegrationPanel.tsx`: ~150
- 10 specific panels x ~120 LOC each: ~1200
- Total: ~1600

**Store:** `stores/integrations.ts` (~250 LOC)

---

### 5.6 TipTap Rich Text Editor

**Location:** `components/shared/RichTextEditor/`

Reusable rich text editor built on TipTap. Used in: Wiki, Helpdesk (canned responses),
Chat (optional rich messages), Kommunikation (reply composer), Formulare (rich
description fields).

```
RichTextEditor.tsx                   # Main editor wrapper
├── EditorToolbar.tsx                # Formatting toolbar
│   ├── FormatGroup.tsx              # Bold, italic, underline, strikethrough
│   ├── HeadingGroup.tsx             # H1, H2, H3
│   ├── ListGroup.tsx                # Bullet list, ordered list, task list
│   ├── InsertGroup.tsx              # Link, image, table, code block, divider
│   └── AlignGroup.tsx               # Left, center, right
├── EditorContent.tsx                # TipTap editor area
├── EditorBubbleMenu.tsx             # Floating toolbar on selection
└── EditorFooter.tsx                 # Word count, character count
```

**Features:**
- Headings (H1-H3)
- Bold, italic, underline, strikethrough
- Bullet and ordered lists
- Task lists (checkboxes)
- Tables
- Code blocks with syntax highlighting
- Links
- Images (paste, drag & drop)
- Horizontal dividers
- Placeholder text
- Read-only mode (for rendering)

**npm dependency:** `@tiptap/react`, `@tiptap/starter-kit`, `@tiptap/extension-*`

**Estimated LOC:** ~600

---

## 6. New Zustand Stores

All stores follow the existing project pattern:

```typescript
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

// Types at top
// Mock data as const array
// Store with persist middleware, localStorage key: 'kmuhub-{name}'
```

| Store | File | Key | Est. LOC | Used by |
|-------|------|-----|----------|---------|
| Communication | `stores/communication.ts` | `kmuhub-communication` | ~450 | Kommunikation module |
| Wiki | `stores/wiki.ts` | `kmuhub-wiki` | ~350 | Wiki module |
| Presence | `stores/presence.ts` | `kmuhub-presence` | ~120 | StatusDot, Chat, Team, Meetings |
| Search | `stores/search.ts` | `kmuhub-search` | ~100 | GlobalSearch |
| Notifications | `stores/notifications.ts` | `kmuhub-notifications` | ~200 | NotificationCenter, Bell |
| Integrations | `stores/integrations.ts` | `kmuhub-integrations` | ~250 | Settings integration panels |
| Custom Fields | (extend `contacts.ts`) | (existing key) | ~100 | CRM custom fields |

**Total new store LOC: ~1570**

Note: `stores/meeting.ts` already exists (421 LOC) and will be extended in-place for
video meeting state (active participants, audio/video toggle, screen share state).
That extension is estimated at ~80 additional LOC within the existing file.

---

## 7. Routing Changes

Changes to `App.tsx` router configuration:

### New routes

```typescript
// In the router children array:

// Kommunikation (Unified External Inbox)
{
  path: 'kommunikation',
  element: (
    <Suspense fallback={<ModuleLoadingFallback />}>
      <KommunikationPage />
    </Suspense>
  ),
},

// Wiki (Knowledge Base)
{
  path: 'wiki',
  element: (
    <Suspense fallback={<ModuleLoadingFallback />}>
      <WikiPage />
    </Suspense>
  ),
},

// Video Meeting Room (standalone view)
{
  path: 'meeting/:roomId',
  element: (
    <Suspense fallback={<ModuleLoadingFallback />}>
      <VideoMeetingRoom />
    </Suspense>
  ),
},
```

### Route renames

No actual route path changes. The `/buchhaltung` route stays — only the display
label in the sidebar changes from "Buchhaltung" to "Rechnungen & Finanzen". This
avoids breaking bookmarks and internal links.

### Lazy imports to add at top of `App.tsx`

```typescript
const KommunikationPage = lazy(() => import('@/modules/kommunikation/KommunikationPage'))
const WikiPage = lazy(() => import('@/modules/wiki/WikiPage'))
const VideoMeetingRoom = lazy(() => import('@/components/shared/VideoMeeting/VideoMeetingRoom'))
```

---

## 8. Navigation Updates

### Changes to `nav-items.ts`

```typescript
// ADD — between 'mail' and 'finance':
{ id: 'kommunikation', to: '/kommunikation', icon: Inbox, label: 'Kommunikation', enabled: true, section: 'main', badge: { type: 'text', value: '4' } },

// ADD — between 'documents' and 'mail':
{ id: 'wiki', to: '/wiki', icon: BookOpen, label: 'Wiki', enabled: true, section: 'main' },

// CHANGE — rename label:
{ id: 'finance', to: '/buchhaltung', icon: Calculator, label: 'Rechnungen & Finanzen', enabled: true, section: 'main' },
```

New Lucide imports needed: `Inbox`, `BookOpen`.

### Changes to `business-profiles.ts`

Add `'kommunikation'` and `'wiki'` to relevant profiles:

- **allgemein:** Add both to `defaultModules`
- **handwerk:** Add `'kommunikation'` to `defaultModules`, `'wiki'` to `optionalModules`
- **gastronomie:** Add `'kommunikation'` to `optionalModules`
- **einzelhandel:** Add `'kommunikation'` to `defaultModules`
- **dienstleistung:** Add both to `defaultModules`
- **it_tech:** Add both to `defaultModules`
- **produktion:** Add `'kommunikation'` to `optionalModules`
- **logistik:** Add `'kommunikation'` to `optionalModules`
- **gesundheit:** Add `'kommunikation'` to `defaultModules`
- **bau:** Add `'kommunikation'` to `optionalModules`

---

## 9. Sprint Plan (Buildable Now vs Backend-Dependent)

### Immediately buildable (Darien, mock data)

| # | Feature | Complexity | Est. days | Dependencies |
|---|---------|------------|-----------|-------------|
| 1 | TipTap Rich Text Editor component | Medium | 2 | npm: @tiptap/* |
| 2 | Kommunikation module (full UI) | High | 4-5 | TipTap (#1) |
| 3 | Wiki module (full UI) | High | 3-4 | TipTap (#1) |
| 4 | Buchhaltung rename + restructure | Medium | 2 | — |
| 5 | Belegkette tab | Medium | 2 | — |
| 6 | Exporte tab (DATEV/Bexio panels) | Medium | 1.5 | — |
| 7 | QR-Rechnung preview | Low | 0.5 | — |
| 8 | MWSt multi-country selector | Low | 0.5 | — |
| 9 | Team/HR Lohn removal + integration panel | Low | 1 | — |
| 10 | Custom Fields UI | Medium | 2 | — |
| 11 | Firma detail panel | Medium | 2 | Custom Fields (#10) |
| 12 | Duplikaterkennung dialog | Medium | 1.5 | — |
| 13 | Newsletter panel | Low | 1 | — |
| 14 | Chat: Thread replies enhancement | Medium | 1 | — |
| 15 | Chat: Reactions | Medium | 1 | — |
| 16 | Chat: @mentions autocomplete | Medium | 1.5 | — |
| 17 | Chat: File sharing | Medium | 1 | — |
| 18 | Status/Presence system | Medium | 1.5 | — |
| 19 | Chat: Presence integration | Low | 0.5 | Presence (#18) |
| 20 | Video meeting room UI shell | High | 3 | — |
| 21 | Global search (Cmd+K) | Medium | 2 | — |
| 22 | Notification center rebuild | Medium | 2 | — |
| 23 | Helpdesk: Canned responses | Medium | 1.5 | TipTap (#1) |
| 24 | Helpdesk: Private notes | Low | 0.5 | — |
| 25 | Helpdesk: Business hours config | Medium | 1 | — |
| 26 | Helpdesk: SLA indicators | Low | 0.5 | — |
| 27 | Kalender: Video meeting button | Low | 0.5 | — |
| 28 | Vertraege: E-Signatur dialog | Medium | 1.5 | — |
| 29 | Dokumente: Template gallery | Medium | 1 | — |
| 30 | Dokumente: Share link dialog | Low | 0.5 | — |
| 31 | Integration settings panels | High | 3 | — |
| 32 | All new stores | Medium | 2 | — |
| 33 | Routing + navigation updates | Low | 0.5 | — |

**Total estimated: ~48-52 dev days**

### Backend-dependent (needs Luke first)

These items need real backend endpoints. Darien builds the UI shell with mock data,
but the actual functionality requires Luke's work.

| Feature | Backend requirement | Luke's phase |
|---------|-------------------|--------------|
| Real email sync in Kommunikation | IMAP connector service | Phase 10 |
| Teams Bridge messages | Microsoft Graph API proxy | Phase 10 |
| WhatsApp messages | Meta Cloud API proxy | Phase 10 |
| OnlyOffice document editing | WOPI endpoint + OnlyOffice Docker | Phase 11 |
| LiveKit video rooms | Token generation service | Phase 9 |
| Zoom meeting links | OAuth2 + Zoom API proxy | Phase 9 |
| Real CRM search + dedup | Fuzzy search endpoint + dedup algorithm | Phase 8 |
| DATEV export | DATEV format generator service | Phase 10 |
| Bexio sync | Bexio API connector | Phase 10 |
| Skribble e-signature | Skribble API integration | Phase 11 |
| Real notifications | WebSocket push service | Phase 8 |
| Real presence/status | WebSocket presence channel | Phase 8 |
| Newsletter sync (Brevo) | Brevo API connector | Phase 11 |
| Custom fields persistence | JSONB column + CRUD endpoints | Phase 8 |
| Wiki versioning | Server-side version storage | Phase 9 |

**Strategy:** Darien builds all UI now with mock stores. When Luke finishes each
backend phase, the store actions get replaced with `react-query` API calls. The
components themselves should not need changes — they consume store state, not
API details directly.

---

## 10. Priority Order and Dependency Graph

### Phase F1: Foundation components (Week 1-2)

Build reusable components that other features depend on.

```
TipTap Rich Text Editor ──→ Wiki, Kommunikation, Helpdesk canned responses
Status/Presence system  ──→ Chat presence, Meeting room, Team page
Global Search (Cmd+K)   ──→ All modules (search integration)
stores (all new)        ──→ All new modules
```

**Deliverables:** TipTap editor, StatusDot/StatusPicker, GlobalSearchDialog,
all 7 new stores with mock data.

### Phase F2: Kommunikation module (Week 3-4)

Biggest new feature, highest product impact. Unified inbox for external communication.

```
stores/communication.ts (from F1)
TipTap editor (from F1)
        │
        ▼
KommunikationPage.tsx
├── ConversationList
├── ConversationThread (uses TipTap for ReplyComposer)
├── ContextPanel (reads from contacts store)
├── CannedResponsePicker (shared with Helpdesk)
└── InsertFromCRMButton (reads from finance + documents stores)
```

**Deliverables:** Full Kommunikation module with 12-15 mock conversations.

### Phase F3: Wiki module (Week 4-5)

Second new module. TipTap editor reuse proves the component works.

```
stores/wiki.ts (from F1)
TipTap editor (from F1)
        │
        ▼
WikiPage.tsx
├── WikiSidebar (tree navigation)
├── WikiArticle (TipTap read + edit modes)
├── WikiVersionHistory
└── WikiTemplateDialog
```

**Deliverables:** Full Wiki module with 12 mock articles, 4 categories, 3 templates.

### Phase F4: Finance + CRM rebuilds (Week 5-7)

Buchhaltung restructuring and CRM extensions.

```
Buchhaltung rename + tabs          CRM Custom Fields
Belegkette tab                     Firma detail panel
Exporte tab (DATEV/Bexio)          Duplikaterkennung
QR-Rechnung preview                Newsletter panel
MWSt multi-country
```

**Deliverables:** Renamed "Rechnungen & Finanzen" with new tabs. CRM with custom
fields, company entity, dedup dialog.

### Phase F5: Chat + Helpdesk extensions (Week 7-8)

Chat gets threads, reactions, mentions, file sharing. Helpdesk gets canned responses,
private notes, business hours, SLA.

```
Chat extensions (parallel)          Helpdesk extensions (parallel)
├── Thread replies                  ├── Canned responses (uses TipTap)
├── Reactions                       ├── Private notes
├── @mentions                       ├── Business hours
├── File sharing                    └── SLA indicators
└── Presence integration
```

### Phase F6: Video + Notifications + Kalender (Week 8-9)

```
Video Meeting Room UI shell
Notification center rebuild
Kalender video meeting button
Team/HR Lohn removal + integration panel
Vertraege E-Signatur dialog
```

### Phase F7: Integration panels + polish (Week 9-10)

```
Integration settings panels (10 panels)
Dokumente extensions (template gallery, share link, OnlyOffice shell)
Routing + navigation final updates
Cross-module testing
```

### Dependency graph (simplified)

```
                    ┌─────────────┐
                    │  TipTap     │
                    │  Editor     │
                    └──┬───┬───┬──┘
                       │   │   │
            ┌──────────┘   │   └──────────┐
            ▼              ▼              ▼
     ┌─────────────┐ ┌──────────┐  ┌──────────────┐
     │ Kommunikation│ │  Wiki    │  │ Helpdesk     │
     │ Module       │ │  Module  │  │ Canned Resp. │
     └─────────────┘ └──────────┘  └──────────────┘

     ┌─────────────┐
     │ Presence     │
     │ System       │
     └──┬───┬───┬──┘
        │   │   │
        ▼   ▼   ▼
     Chat  Team  Meetings

     ┌─────────────┐
     │ New Stores   │──→ All new modules + components
     └─────────────┘

     ┌─────────────┐
     │ Custom Fields│──→ Kontakte + Firma detail
     └─────────────┘
```

---

## 11. LOC Estimates Summary

### New modules

| Module | Components | Store | Total |
|--------|-----------|-------|-------|
| Kommunikation | 1800 | 450 | **2250** |
| Wiki | 1400 | 350 | **1750** |

### Module extensions

| Module | New LOC |
|--------|---------|
| Buchhaltung (Rechnungen & Finanzen) | **1830** |
| Team/HR | **290** |
| CRM/Kontakte | **1770** |
| Dokumente | **895** |
| Chat | **1150** |
| Helpdesk | **970** |
| Kalender | **170** |
| Vertraege | **600** |

### Shared components

| Component | LOC |
|-----------|-----|
| Video Meeting Room | **800** |
| Status/Presence | **350** |
| Global Search (Cmd+K) | **550** |
| Notification Center (rebuild) | **700** |
| Integration Settings Panels | **1600** |
| TipTap Rich Text Editor | **600** |

### Infrastructure

| Item | LOC |
|------|-----|
| New stores (7) | **1570** |
| Routing + nav updates | **~80** |
| Meeting store extension | **~80** |

### Grand total

| Category | LOC |
|----------|-----|
| New modules | 4,000 |
| Module extensions | 7,675 |
| Shared components | 4,600 |
| Infrastructure | 1,730 |
| **TOTAL** | **~18,005** |

With inevitable iteration, test code, types, and polish, realistic total is
**~20,000-22,000 LOC**.

---

## 12. File Structure Overview

New files and directories to create (relative to `desktop/src/renderer/src/`):

```
modules/
├── kommunikation/                        # NEW MODULE
│   ├── KommunikationPage.tsx
│   ├── ChannelTabs.tsx
│   ├── ConversationList.tsx
│   ├── ConversationListHeader.tsx
│   ├── ConversationListItem.tsx
│   ├── ConversationListFilters.tsx
│   ├── ConversationThread.tsx
│   ├── ConversationThreadHeader.tsx
│   ├── MessageTimeline.tsx
│   ├── MessageItem.tsx
│   ├── ReplyComposer.tsx
│   ├── CannedResponsePicker.tsx
│   ├── InsertFromCRMButton.tsx
│   ├── InternalNoteComposer.tsx
│   ├── ContextPanel.tsx
│   ├── ContactCard.tsx
│   ├── OpenDeals.tsx
│   ├── OpenTickets.tsx
│   ├── RelatedProjects.tsx
│   ├── ActivityTimeline.tsx
│   ├── NewConversationDialog.tsx
│   └── ChannelSettingsDialog.tsx
│
├── wiki/                                 # NEW MODULE
│   ├── WikiPage.tsx
│   ├── WikiSidebar.tsx
│   ├── WikiTreeNode.tsx
│   ├── WikiSearch.tsx
│   ├── WikiNewButton.tsx
│   ├── WikiArticle.tsx
│   ├── WikiArticleHeader.tsx
│   ├── WikiEditor.tsx
│   ├── WikiRenderer.tsx
│   ├── WikiArticleFooter.tsx
│   ├── WikiVersionHistory.tsx
│   ├── WikiVersionItem.tsx
│   ├── WikiTemplateDialog.tsx
│   ├── WikiCategoryDialog.tsx
│   └── WikiShareDialog.tsx
│
├── buchhaltung/                          # EXTENDED
│   ├── BelegketteTab.tsx                 # NEW
│   ├── ExporteTab.tsx                    # NEW
│   └── QRRechnungPreview.tsx             # NEW
│
├── kontakte/                             # EXTENDED
│   ├── CustomFieldsConfig.tsx            # NEW
│   ├── CustomFieldRow.tsx                # NEW
│   ├── CustomFieldPreview.tsx            # NEW
│   ├── FirmaDetailPanel.tsx              # NEW
│   ├── DuplicateDetectionDialog.tsx      # NEW
│   ├── DuplicateMatchCard.tsx            # NEW
│   ├── MergeFieldSelector.tsx            # NEW
│   └── NewsletterPanel.tsx               # NEW
│
├── helpdesk/                             # EXTENDED
│   ├── CannedResponsesPanel.tsx          # NEW
│   ├── CannedResponseList.tsx            # NEW
│   ├── CannedResponseEditor.tsx          # NEW
│   ├── BusinessHoursDialog.tsx           # NEW
│   └── SLABadge.tsx                      # NEW
│
├── vertraege/                            # EXTENDED
│   ├── ESignaturDialog.tsx               # NEW
│   ├── SignerList.tsx                     # NEW
│   ├── SigningOrderConfig.tsx            # NEW
│   └── SignatureStatusTimeline.tsx        # NEW
│
├── dokumente/                            # EXTENDED
│   ├── OnlyOfficeViewer.tsx              # NEW
│   ├── TemplateGalleryDialog.tsx         # NEW
│   ├── TemplateCard.tsx                  # NEW
│   ├── TemplateCategoryFilter.tsx        # NEW
│   └── ShareLinkDialog.tsx               # NEW
│
├── team/                                 # EXTENDED
│   └── HRIntegrationPanel.tsx            # NEW
│
├── settings/                             # EXTENDED
│   └── integrations/                     # NEW DIRECTORY
│       ├── IntegrationSettingsTab.tsx
│       ├── IntegrationCard.tsx
│       ├── GenericIntegrationPanel.tsx
│       ├── DATEVConfigPanel.tsx
│       ├── BexioConfigPanel.tsx
│       ├── BrevoConfigPanel.tsx
│       ├── SkribbleConfigPanel.tsx
│       ├── OnlyOfficeConfigPanel.tsx
│       ├── ZoomConfigPanel.tsx
│       ├── LiveKitConfigPanel.tsx
│       ├── TeamsConfigPanel.tsx
│       └── WhatsAppConfigPanel.tsx
│
components/
├── shared/
│   ├── VideoMeeting/                     # NEW DIRECTORY
│   │   ├── VideoMeetingRoom.tsx
│   │   ├── VideoGrid.tsx
│   │   ├── VideoTile.tsx
│   │   ├── MeetingControls.tsx
│   │   ├── MeetingParticipantList.tsx
│   │   ├── MeetingSidebar.tsx
│   │   ├── MeetingChat.tsx
│   │   ├── ScreenShareOverlay.tsx
│   │   └── MeetingEndDialog.tsx
│   │
│   ├── Presence/                         # NEW DIRECTORY
│   │   ├── StatusDot.tsx
│   │   ├── StatusPicker.tsx
│   │   ├── SetStatusDialog.tsx
│   │   └── UserPresenceProvider.tsx
│   │
│   ├── GlobalSearch/                     # NEW DIRECTORY
│   │   ├── GlobalSearchDialog.tsx
│   │   ├── SearchInput.tsx
│   │   ├── SearchResultGroup.tsx
│   │   ├── SearchResultItem.tsx
│   │   ├── RecentSearches.tsx
│   │   └── QuickActions.tsx
│   │
│   └── RichTextEditor/                   # NEW DIRECTORY
│       ├── RichTextEditor.tsx
│       ├── EditorToolbar.tsx
│       ├── FormatGroup.tsx
│       ├── HeadingGroup.tsx
│       ├── ListGroup.tsx
│       ├── InsertGroup.tsx
│       ├── AlignGroup.tsx
│       ├── EditorContent.tsx
│       ├── EditorBubbleMenu.tsx
│       └── EditorFooter.tsx
│
stores/
├── communication.ts                      # NEW
├── wiki.ts                               # NEW
├── presence.ts                           # NEW
├── search.ts                             # NEW
├── notifications.ts                      # NEW (replaces inline)
└── integrations.ts                       # NEW
```

**New files total:** ~95 new `.tsx`/`.ts` files
**Modified files:** ~15 existing files (App.tsx, nav-items.ts, business-profiles.ts,
BuchhaltungPage.tsx, TeamPage.tsx, HelpdeskPage.tsx, KalenderPage.tsx,
VertraegePage.tsx, DokumentePage.tsx, ChatLayout.tsx + chat sub-components,
contacts.ts, meetings.ts, finance.ts)

---

## Appendix A: npm Dependencies to Add

| Package | Version | Purpose | Size |
|---------|---------|---------|------|
| `@tiptap/react` | ^2.x | TipTap React bindings | ~50 KB |
| `@tiptap/starter-kit` | ^2.x | Core extensions bundle | ~80 KB |
| `@tiptap/extension-link` | ^2.x | Link support | ~5 KB |
| `@tiptap/extension-image` | ^2.x | Image embedding | ~5 KB |
| `@tiptap/extension-table` | ^2.x | Table support | ~15 KB |
| `@tiptap/extension-task-list` | ^2.x | Checkbox lists | ~5 KB |
| `@tiptap/extension-task-item` | ^2.x | Task list items | ~3 KB |
| `@tiptap/extension-placeholder` | ^2.x | Placeholder text | ~3 KB |
| `@tiptap/extension-code-block-lowlight` | ^2.x | Syntax highlighting | ~10 KB |
| `@tiptap/extension-mention` | ^2.x | @mentions in editor | ~5 KB |
| `lowlight` | ^3.x | Syntax highlighting engine | ~20 KB |

**Total new bundle size:** ~200 KB (before tree-shaking). TipTap is modular, so
only imported extensions are bundled.

No other major dependencies needed. The video meeting UI shell uses mock data and
does not need LiveKit SDK until Luke's backend is ready.

---

## Appendix B: Type Definitions to Add

New file: `types/communication.ts`
```typescript
export type ChannelType = 'email' | 'teams' | 'whatsapp' | 'widget' | 'portal'
export type ConversationStatus = 'open' | 'waiting' | 'resolved' | 'spam'
export type MessageType = 'inbound' | 'outbound' | 'internal_note' | 'system'

export interface Conversation { /* ... as defined in Section 3.1 */ }
export interface ConversationMessage { /* ... */ }
export interface CannedResponse {
  id: string
  title: string
  content: string
  category: string
  shortcut: string       // e.g. "/gruss" expands to full greeting
}
```

New file: `types/wiki.ts`
```typescript
export interface WikiCategory { /* ... as defined in Section 3.2 */ }
export interface WikiArticle { /* ... */ }
export interface WikiVersion { /* ... */ }
export interface WikiTemplate { /* ... */ }
```

New file: `types/presence.ts`
```typescript
export type UserStatus = 'online' | 'away' | 'busy' | 'dnd' | 'offline'

export interface UserPresence {
  userId: string
  status: UserStatus
  customMessage: string | null
  customEmoji: string | null
  expiresAt: string | null       // custom status expiry
  lastSeen: string
}
```

New file: `types/integrations.ts`
```typescript
export type IntegrationId =
  | 'datev' | 'bexio' | 'brevo' | 'cleverreach'
  | 'skribble' | 'onlyoffice' | 'zoom' | 'livekit'
  | 'teams_bridge' | 'whatsapp'

export type IntegrationStatus = 'connected' | 'disconnected' | 'error' | 'syncing'

export interface IntegrationConfig {
  id: IntegrationId
  name: string
  description: string
  icon: string
  status: IntegrationStatus
  lastSync: string | null
  config: Record<string, string>
}
```

---

## Appendix C: CSS/Styling Considerations

All new components must support:

1. **Light and dark mode** — Use CSS variables (`var(--card)`, `var(--heading)`,
   `var(--muted)`, etc.) from `globals.css`. No hardcoded colors.

2. **Glass/Crystal mode** — Any overlay components (dialogs, panels, dropdowns)
   must work with `.ui-glass` and `.ui-crystal` classes on `<html>`. Use
   `.glass-elevated` class on non-Radix overlays (like DetailPanel).

3. **Desk theme compatibility** — Components render inside the DeskFrame window.
   All background colors must use CSS variables, not Tailwind color utilities
   directly.

4. **Responsive in-frame** — The app renders in a desk "window" that can be
   various sizes. Components must handle container widths from ~800px to ~1600px
   gracefully. Use flex layouts, not fixed widths (except sidebar panels).

5. **Scrollbar styling** — ScrollArea from Radix is already styled in globals.css.
   Use `<ScrollArea>` for all scrollable content areas.

6. **TipTap editor styling** — The editor needs custom CSS for the content area.
   Create `styles/tiptap.css` with ProseMirror overrides that respect the design
   system's CSS variables.

---

## Appendix D: Testing Strategy

While Darien focuses on UI implementation, all new components should be structured
for testability:

1. **Store logic is pure functions** — All Zustand actions are testable without
   React. Test CRUD operations on mock data.

2. **Components accept props** — Avoid directly consuming stores in leaf components.
   Parent components read from stores and pass props down. This makes leaf
   components unit-testable.

3. **Dialog components are standalone** — All dialogs (ESignaturDialog,
   DuplicateDetectionDialog, etc.) accept `open`, `onClose`, and data props.
   Testable in isolation.

4. **Keyboard shortcuts are documented** — Global search (Cmd+K), Escape to close
   dialogs, arrow keys for navigation — all documented and testable.

5. **LOC target for test files** — When test infrastructure is set up, aim for
   1 test file per store (~100-200 LOC each) and 1 test file per complex
   component (~50-100 LOC each).

---

*End of Frontend Implementation Plan*
*Next steps: Begin Phase F1 (TipTap editor + Presence system + stores)*

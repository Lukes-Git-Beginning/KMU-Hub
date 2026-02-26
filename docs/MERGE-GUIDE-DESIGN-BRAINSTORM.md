# Merge Guide: design/brainstorm -> main

> This document is written for Darien's AI agent to follow when preparing
> pull requests from the `design/brainstorm` branch into `main`.

## Branch Overview

- **Branch:** `origin/design/brainstorm`
- **Base:** `main`
- **Scope:** 694 changed files, ~141k insertions, ~21k deletions
- **Content:** Design system overhaul (20 waves), new modules, backend guest-chat removal, GSD tooling, planning docs, reference prototypes

---

## A) Files to EXCLUDE from PRs

These files/directories must **not** be included in any PR. They are internal
tooling, local configuration, or reference-only material.

| Path | Reason |
|------|--------|
| `.claude/agents/` | Internal GSD agent definitions (each dev has their own) |
| `.claude/commands/gsd/` | Internal GSD slash commands |
| `.claude/get-shit-done/` | GSD framework (templates, references, workflows) |
| `.claude/hooks/` | Local GSD hooks |
| `.claude/settings.json` | Local Claude Code settings |
| `.claude/package.json` | Local Claude Code package |
| `.claude/gsd-file-manifest.json` | GSD manifest |
| `.planning/` | Internal planning & strategy documents |
| `desktop/design-reference/` | Reference design prototype (97 files) — not product code |
| `desktop/scripts/asset-gen/` | AI asset generation scripts — dev tooling only |
| `docs/PRODUCT-STRATEGY.md` | Internal strategy doc |
| `docs/STRATEGY.md` | Internal strategy doc |

**Tip:** When creating PRs from the branch, use `git diff` with path filters to only
include the relevant files for each PR.

---

## B) PR Sequence (merge in this order)

Each PR must build cleanly on its own. Merge them in order — each one depends
on the previous.

---

### PR 1: Infrastructure & Dependencies

**~10 files — small, quick review**

Purpose: Update project configuration and add new npm dependencies that all
subsequent PRs depend on.

**Files:**
- `.gitignore`
- `CLAUDE.md`
- `desktop/package.json`
- `desktop/package-lock.json`
- `desktop/src/renderer/index.html`
- `desktop/eslint.config.mjs` (if changed)

**New dependencies to highlight in PR description:**
- Tiptap ecosystem (`@tiptap/react`, `@tiptap/starter-kit`, + 14 extensions) — rich text editing
- `@xyflow/react` — flow/org chart visualization
- `lowlight` — syntax highlighting for code blocks
- `openai` (devDep) — AI feature integration

**PR description template:**
```
## Infrastructure & Dependencies

Add new npm dependencies required by the design system overhaul:
- Tiptap rich text editor ecosystem (15 packages)
- React Flow for org charts and process diagrams
- Updated ESLint configuration

These dependencies are prerequisites for all subsequent design PRs.
```

---

### PR 2: Design System & Styles

**~30 files — CSS, themes, UI component library**

Purpose: Introduce the new visual foundation — animations, glass effects,
theme configuration, and shadcn/ui component enhancements.

**New CSS files:**
- `desktop/src/renderer/src/styles/animations.css`
- `desktop/src/renderer/src/styles/background-patterns.css`
- `desktop/src/renderer/src/styles/glass-effects.css`
- `desktop/src/renderer/src/styles/micro-interactions.css`
- `desktop/src/renderer/src/styles/nav-icons.css`
- `desktop/src/renderer/src/styles/tiptap.css`

**Modified:**
- `desktop/src/renderer/src/styles/globals.css`

**Theme configuration:**
- `desktop/src/renderer/src/config/background-patterns.ts`
- `desktop/src/renderer/src/config/desk-themes.ts`
- `desktop/src/renderer/src/config/desk-asset-urls.ts`
- `desktop/src/renderer/src/config/business-profiles.ts`

**Desk assets:**
- `desktop/src/renderer/assets/desk/dreamy/` (new images)

**UI component library updates (shadcn/ui):**
- `components/ui/alert-dialog.tsx`
- `components/ui/avatar.tsx`
- `components/ui/badge.tsx`
- `components/ui/button.tsx`
- `components/ui/card.tsx`
- `components/ui/checkbox.tsx`
- `components/ui/dialog.tsx`
- `components/ui/dropdown-menu.tsx`
- `components/ui/input.tsx`
- `components/ui/label.tsx`
- `components/ui/popover.tsx`
- `components/ui/scroll-area.tsx`
- `components/ui/select.tsx`
- `components/ui/sheet.tsx`
- `components/ui/skeleton.tsx`
- `components/ui/switch.tsx`
- `components/ui/table.tsx`
- `components/ui/tabs.tsx`
- `components/ui/textarea.tsx`
- `components/ui/tooltip.tsx`

**Include screenshots** of theme variations (light/dark/glass/crystal).

---

### PR 3: Layout System & Navigation

**~35 files — core structural changes**

Purpose: Introduce 4 navigation layout modes (classic sidebar, dock, top-nav,
original), refactor AppShell to DeskEnvironment, and update routing.

> **CRITICAL:** `DEV_BYPASS_AUTH = true` in `App.tsx` must be removed or
> gated behind `import.meta.env.DEV` before this PR is submitted.

**Layout system:**
- `desktop/src/renderer/src/components/layout/DeskEnvironment.tsx`
- `desktop/src/renderer/src/components/layout/AppShell.tsx`
- `desktop/src/renderer/src/components/layout/PageTransitionOutlet.tsx` (new)
- `desktop/src/renderer/src/components/layout/BackgroundPattern.tsx` (new)

**4 layout variants:**
- `components/layout/classic/ClassicLayout.tsx`
- `components/layout/classic/ClassicSidebar.tsx`
- `components/layout/classic/index.ts`
- `components/layout/dock/DockBar.tsx`
- `components/layout/dock/DockLayout.tsx`
- `components/layout/dock/index.ts`
- `components/layout/topnav/TopNavBar.tsx`
- `components/layout/topnav/TopNavLayout.tsx`
- `components/layout/topnav/ModuleOverviewPanel.tsx`
- `components/layout/topnav/index.ts`

**Sidebar updates:**
- `components/layout/sidebar/Sidebar.tsx`
- `components/layout/sidebar/SidebarBranding.tsx`
- `components/layout/sidebar/SidebarModulePanel.tsx` (new)
- `components/layout/sidebar/SidebarNav.tsx`
- `components/layout/sidebar/nav-items.ts`

**Header system:**
- `components/header/SearchBar.tsx`
- `components/header/TimeTrackerWidget.tsx`
- `components/header/HeaderWidgetSlots.tsx` (new)
- `components/header/header-widgets/NextMeetingWidget.tsx` (new)
- `components/header/header-widgets/PomodoroWidget.tsx` (new)
- `components/header/header-widgets/QuickNoteWidget.tsx` (new)
- `components/header/header-widgets/UnreadCountWidget.tsx` (new)
- `components/header/header-widgets/WeatherWidget.tsx` (new)
- `components/header/header-widgets/index.ts` (new)
- `components/header/index.ts`

**Shared layout components:**
- `components/shared/LayoutSwitcher.tsx` (new)
- `components/shared/PaletteSwitcher.tsx` (new)

**Hooks:**
- `hooks/useFilteredNavItems.ts` (new)
- `hooks/usePageTransition.ts` (new)
- `hooks/useTiltEffect.ts` (new)

**Routing (App.tsx):**
- `desktop/src/renderer/src/App.tsx`
- New routes: `/wiki`, `/employee-wizard-window`, `/compose-window`
- Security admin consolidation: 7 individual pages merged into `/admin/security`
- `RouteErrorFallback` component added
- `ProfileSwitcher` for dev mode only

**Config:**
- `desktop/src/renderer/src/config/roles.ts`

**Auth module:**
- `desktop/src/renderer/src/modules/auth/LoginPage.tsx`

**Include GIFs** demonstrating layout switching between all 4 modes.

---

### PR 4: Shared Components, Stores & Data Layer

**~50 files — reusable components, state management, API layer**

Purpose: Add shared building blocks (rich text editor, global search, animated
components), update all Zustand stores, and refactor API clients.

**New shared components:**
- `components/shared/GlobalSearch/` (7 files: dialog, input, results, quick actions, recent)
- `components/shared/RichTextEditor/` (6 files: editor, toolbar, bubble menu, footer)
- `components/shared/AnimatedCheckmark.tsx`
- `components/shared/AnimatedList.tsx`
- `components/shared/ConfettiBurst.tsx`
- `components/shared/LoadingSpinner.tsx`
- `components/shared/PageHeader.tsx`
- `components/shared/StatCard.tsx`
- `components/shared/TextReveal.tsx`
- `components/shared/index.ts`

**Modified shared components:**
- `components/shared/ConfirmDialog.tsx`
- `components/shared/DetailPanel.tsx`
- `components/shared/EmptyState.tsx`
- `components/shared/FormField.tsx`

**New Zustand stores:**
- `stores/ai.ts` — AI features/governance state
- `stores/integrations.ts` — integrations state
- `stores/notifications.ts` — notification state
- `stores/search.ts` — global search state
- `stores/wiki.ts` — wiki state

**Modified stores (18):**
- `stores/berichte.ts`, `stores/contacts.ts`, `stores/dashboard.ts`,
  `stores/einkauf.ts`, `stores/finance.ts`, `stores/formulare.ts`,
  `stores/fuhrpark.ts`, `stores/helpdesk.ts`, `stores/inventar.ts`,
  `stores/kommunikation.ts`, `stores/meetings.ts`, `stores/produktion.ts`,
  `stores/rapporte.ts`, `stores/team.ts`, `stores/timetracking.ts`,
  `stores/ui.ts`, `stores/vermietung.ts`, `stores/vertraege.ts`

**API layer:**
- `api/notification-client.ts` (new)
- `api/hooks/useIntegrations.ts` (new, replaces `useIntegration.ts`)
- `api/hooks/useIntegration.ts` (deleted)
- `api/hooks/hr-hooks.ts`, `api/hooks/useCaldav.ts`, `api/hooks/useNotifications.ts` (modified)
- `api/caldav-client.ts`, `api/calendar-client.ts`, `api/client.ts`, `api/hr-client.ts`, `api/integration-client.ts` (modified)
- `api/hr-types.ts`, `api/integration-types.ts` (modified)

**New types:**
- `types/communication.ts`
- `types/wiki.ts`

**IPC handlers:**
- `desktop/src/main/ipc/compose.ts` (new)
- `desktop/src/main/ipc/employee-wizard.ts` (new)
- `desktop/src/main/ipc/index.ts` (modified)
- `desktop/src/main/menu.ts` (modified)
- `desktop/src/preload/index.ts`, `desktop/src/preload/types.ts` (modified)

**Utilities:**
- `lib/format.ts` (new)
- `lib/index.ts` (modified)

**Mock data:**
- `mocks/mock-db.ts` (new) — realistic German business data for offline dev

**i18n:**
- `i18n/messages/de.json` (modified)
- `i18n/messages/en.json` (modified)

---

### PR 5: New & Enhanced Modules

**~150+ files — this is the largest chunk**

Split this into **sub-PRs** for reviewability. Each sub-PR should build
on the previous.

#### PR 5a: Wiki System (11 new files)
- `modules/wiki/WikiPage.tsx`
- `modules/wiki/WikiArticle.tsx`
- `modules/wiki/WikiArticleHeader.tsx`
- `modules/wiki/WikiCategoryDialog.tsx`
- `modules/wiki/WikiEditor.tsx`
- `modules/wiki/WikiSearch.tsx`
- `modules/wiki/WikiShareDialog.tsx`
- `modules/wiki/WikiSidebar.tsx`
- `modules/wiki/WikiTemplateDialog.tsx`
- `modules/wiki/WikiTreeNode.tsx`
- `modules/wiki/WikiVersionHistory.tsx`
- `modules/wiki/WikiVersionItem.tsx`

#### PR 5b: Kommunikation Refactor (~18 files)
**New:**
- `modules/kommunikation/ActivityTimeline.tsx`
- `modules/kommunikation/CannedResponsePicker.tsx`
- `modules/kommunikation/ChannelSettingsDialog.tsx`
- `modules/kommunikation/ChannelTabs.tsx`
- `modules/kommunikation/ContactCard.tsx`
- `modules/kommunikation/ContextPanel.tsx`
- `modules/kommunikation/ConversationList.tsx`
- `modules/kommunikation/ConversationListFilters.tsx`
- `modules/kommunikation/ConversationListItem.tsx`
- `modules/kommunikation/ConversationThread.tsx`
- `modules/kommunikation/ConversationThreadHeader.tsx`
- `modules/kommunikation/InternalNoteComposer.tsx`
- `modules/kommunikation/MessageItem.tsx`
- `modules/kommunikation/MessageTimeline.tsx`
- `modules/kommunikation/NewConversationDialog.tsx`
- `modules/kommunikation/OpenDeals.tsx`
- `modules/kommunikation/OpenTickets.tsx`
- `modules/kommunikation/ReplyComposer.tsx`

**Modified:** `modules/kommunikation/KommunikationPage.tsx`

**Deleted:**
- `modules/kommunikation/InboxSidebar.tsx`
- `modules/kommunikation/MessageDetail.tsx`
- `modules/kommunikation/MessageList.tsx`

#### PR 5c: CRM, Kontakte & Helpdesk
**CRM:**
- `modules/crm/CRMLayout.tsx` (modified)
- `modules/crm/ImportExportDialog.tsx` (new)
- `modules/crm/activities/ActivitiesListPage.tsx` (modified)
- `modules/crm/companies/CompaniesListPage.tsx`, `CompanyFormDialog.tsx` (modified/new)
- `modules/crm/contacts/ContactsListPage.tsx`, `ContactTimeline.tsx`, `TimelineItem.tsx` (modified/new)
- `modules/crm/deals/DealsListPage.tsx`, `DealFormDialog.tsx`, `DealPipelineView.tsx` (modified/new)

**Kontakte:**
- `modules/kontakte/KontaktePage.tsx` (modified)
- `modules/kontakte/ContactFormDialog.tsx` (modified)
- New: `ConsentPanel.tsx`, `CustomFieldPreview.tsx`, `CustomFieldRow.tsx`,
  `CustomFieldsConfig.tsx`, `DuplicateDetectionDialog.tsx`, `DuplicateMatchCard.tsx`,
  `FirmaDetailPanel.tsx`, `MergeFieldSelector.tsx`, `NewsletterPanel.tsx`

**Helpdesk:**
- `modules/helpdesk/HelpdeskPage.tsx` (modified)
- New: `BusinessHoursDialog.tsx`, `CSATWidget.tsx`, `CannedResponsePicker.tsx`,
  `CannedResponsesPanel.tsx`, `SLABadge.tsx`, `TicketRoutingConfig.tsx`

#### PR 5d: Finanzen, Buchhaltung & Mails
**Finanzen:**
- `modules/finanzen/FinanzenPage.tsx` (modified)
- New: `BankingWidget.tsx`, `BelegketteTab.tsx`, `EInvoiceIndicator.tsx`,
  `HoursToInvoiceDialog.tsx`, `PDFPreviewPanel.tsx`, `QRRechnungPreview.tsx`
- Modified: `InvoiceDetailPanel.tsx`, `InvoiceFormDialog.tsx`

**Buchhaltung:**
- `modules/buchhaltung/BuchhaltungPage.tsx` (new)
- New: `ExpenseFormDialog.tsx`, `ExportDialog.tsx`, `InvoiceDetailPanel.tsx`,
  `InvoiceFormDialog.tsx`, `PaymentRecordDialog.tsx`

**Mails:**
- `modules/mails/MailsPage.tsx` (modified)
- `modules/mails/ComposeInline.tsx`, `ComposeModal.tsx`, `ComposeWindowPage.tsx`,
  `compose-shared.tsx` (modified)
- New: `EmailTemplateDialog.tsx`

#### PR 5e: Team/HR & Dashboard Widgets
**Team/HR:**
- `modules/team/TeamPage.tsx` (modified)
- `modules/team/MemberDetailPanel.tsx` (modified)
- New: `CreateEmployeeWizard.tsx`, `EmployeeWizardWindowPage.tsx`,
  `HRIntegrationPanel.tsx`, `OnboardingChecklist.tsx`, `OrgChart.tsx`,
  `PersonnelDocuments.tsx`, `SelfServiceView.tsx`, `TimeCorrectionPanel.tsx`

**Dashboard widgets:**
- `modules/dashboard/DashboardPage.tsx` (modified)
- `modules/dashboard/widgets/ActivityFeed.tsx`, `QuickActions.tsx` (modified)
- New widgets: `Absences.tsx`, `Birthdays.tsx`, `CalendarUpcoming.tsx`,
  `KpiDeals.tsx`, `KpiRevenue.tsx`, `KpiTasks.tsx`, `MiniChart.tsx`,
  `MyCalendar.tsx`, `MyTasks.tsx`, `NotificationFeedWidget.tsx`,
  `TeamChat.tsx`, `TeamStatus.tsx`, `TimeClockWidget.tsx`

#### PR 5f: All Other Modules
- `modules/automatisierung/` (3 files)
- `modules/berichte/BerichtePage.tsx`
- `modules/chat/` (channels, messages with file/mention/reaction support, threads)
- `modules/dokumente/DokumentePage.tsx` + dialogs
- `modules/einkauf/EinkaufPage.tsx`
- `modules/formulare/FormularePage.tsx`
- `modules/fuhrpark/FuhrparkPage.tsx` + `SchadensmeldungDialog.tsx`
- `modules/inventar/InventarPage.tsx`
- `modules/kalender/CalendarLayout.tsx`
- `modules/meetings/` (5 files)
- `modules/notifications/NotificationCenter.tsx`, `NotificationToast.tsx`
- `modules/produktion/ProduktionPage.tsx` + `MaschinenbelegungChart.tsx`
- `modules/profil/ProfilPage.tsx` + zeiterfassung
- `modules/rapporte/RapportePage.tsx` + `SignatureCanvas.tsx`, `SketchCanvas.tsx`
- `modules/schichten/SchichtenPage.tsx`
- `modules/security/SecurityAdminPage.tsx` + `DSARSearchPage.tsx`, `RetentionPolicyPage.tsx` (+ deleted individual pages)
- `modules/settings/` (major refactor: new tabs, integration panels, deleted old)
- `modules/vertraege/VertraegePage.tsx` + `ESignaturDialog.tsx`
- `modules/vermietung/VermietungPage.tsx` + `ZustandsprotokollDialog.tsx`
- `modules/video/VideoPage.tsx`
- `modules/work/` (components & projects subfolders)
- `modules/admin/CalDAVAdminPage.tsx`, `InfrastrukturPage.tsx`

---

### PR 6: Backend & Guest Chat Removal

**~40 files — backend-only changes**

Purpose: Remove the guest chat service (frontend SPA + backend service + proto
definitions + migration) and update core chat infrastructure.

**Deleted — Guest Chat SPA (`guest-chat/` directory):**
- `index.html`, `package.json`, `package-lock.json`, `tsconfig.json`, `vite.config.ts`
- `src/App.tsx`, `src/main.tsx`, `src/styles/globals.css`
- `src/api/client.ts`, `src/api/types.ts`
- `src/components/` (8 components)
- `src/hooks/useGuestChat.ts`, `src/hooks/useWebSocket.ts`

**Deleted — Backend guest service:**
- `backend/internal/chat/guest/errors.go`
- `backend/internal/chat/guest/postgres_repository.go`
- `backend/internal/chat/guest/rate_limiter.go`
- `backend/internal/chat/guest/repository.go`
- `backend/internal/chat/guest/service.go`
- `backend/internal/chat/guest/types.go`
- `backend/internal/gateway/route_guest.go`
- `backend/internal/inbox/adapter/guest_adapter.go`

**Deleted — Migration:**
- `backend/migrations/000054_add_guest_chat_support.down.sql`
- `backend/migrations/000054_add_guest_chat_support.up.sql`

**Modified — Core services:**
- `backend/cmd/gateway/main.go`
- `backend/internal/chat/channel/service.go`
- `backend/internal/chat/message/postgres_repository.go`
- `backend/internal/chat/message/repository.go`
- `backend/internal/chat/message/service.go`
- `backend/internal/chat/message/service_test.go`
- `backend/internal/models/channel.go`
- `backend/internal/models/chat_message.go`
- `backend/internal/notification/event/types.go`
- `backend/internal/server/chat_grpc.go`
- `backend/internal/server/websocket.go`

**Modified — Proto definitions:**
- `backend/proto/chat/v1/chat.proto` + `.pb.go`
- `backend/proto/inbox/v1/inbox.proto` + `.pb.go`

> **Note:** Run `make test` after this PR to verify no regressions in
> chat/channel/message services.

---

## C) Security Review

### GitHub Action Setup

Add the Claude Code Security Review action to the repository:

```yaml
# .github/workflows/security-review.yml
name: Security Review

on:
  pull_request:
    types: [opened, synchronize]

permissions:
  contents: read
  pull-requests: write

jobs:
  security-review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: anthropics/claude-code-security-review@v1
        with:
          anthropic_api_key: ${{ secrets.ANTHROPIC_API_KEY }}
```

Repository: https://github.com/anthropics/claude-code-security-review

### Pre-PR Local Check

Before creating each PR, run locally in Claude Code:

```
/security-review
```

### Security-Critical PRs

Pay extra attention to security findings on:
- **PR 3** — `DEV_BYPASS_AUTH` removal, routing changes, auth flow
- **PR 4** — API clients, data layer, IPC handlers
- **PR 6** — Backend service changes, proto updates, gateway routes

---

## D) Critical Reminders

1. **`DEV_BYPASS_AUTH = true`** in `App.tsx` — MUST be removed or gated behind
   `import.meta.env.DEV` before PR 3 is created. This bypasses all
   authentication and must never reach production.

2. **Security admin consolidation** — 7 individual pages
   (`AuditLogPage`, `SessionsPage`, `VaultPage`, `PasswordPolicyPage`,
   `IPAccessPage`, `GDPRExportPage`, `GDPRErasurePage`) are merged into
   `SecurityAdminPage`. Verify no functionality is lost.

3. **Deleted files** — Always list deleted files explicitly in the PR
   description so reviewers can verify the removal is intentional.

4. **`design-reference/`** — Do NOT include this directory in any PR. It is a
   standalone reference prototype and not part of the product.

5. **Dependencies in PR 1** — All new npm packages (Tiptap, xyflow, openai,
   lowlight) must land in PR 1 so subsequent PRs can import them.

6. **Migration 000054** — The guest chat migration is being deleted. If this
   migration was already applied in any environment, a new migration should
   be created to cleanly remove the tables instead of just deleting the
   migration files.

7. **Build verification** — Each PR must pass `npm run build` in the desktop
   directory before submission. For PR 6, also run `make build` and `make test`
   in the backend directory.

---

## E) Quick Reference: Creating PRs from a Branch

To create a PR with only specific files from the branch:

```bash
# Create a new branch for the PR
git checkout main
git checkout -b pr/01-infrastructure

# Cherry-pick specific changes (or use checkout for individual files)
git checkout design/brainstorm -- .gitignore CLAUDE.md desktop/package.json desktop/package-lock.json desktop/src/renderer/index.html

# Commit and push
git add -A
git commit -m "feat: update infrastructure and add design system dependencies"
git push -u origin pr/01-infrastructure

# Create PR
gh pr create --title "feat: infrastructure & dependencies for design system" --body "..."
```

Repeat for each PR in sequence, always branching from the latest merged state
of `main`.

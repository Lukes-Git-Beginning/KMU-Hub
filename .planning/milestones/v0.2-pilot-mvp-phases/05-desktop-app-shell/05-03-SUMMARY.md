---
phase: 05
plan: 03
subsystem: desktop-crm
tags: [react, tanstack-query, crm, pipeline, search, shadcn-ui]
depends_on:
  requires: ["05-02"]
  provides: ["CRM module UI with list/detail/pipeline/search views", "TanStack Query hooks for all CRM entities"]
  affects: ["05-04", "05-05", "05-06"]
tech-stack:
  added: []
  patterns: ["TanStack Query hooks per entity", "Debounced search pattern", "List/detail page pattern", "Pipeline Kanban column layout"]
key-files:
  created:
    - desktop/src/renderer/src/api/hooks/useContacts.ts
    - desktop/src/renderer/src/api/hooks/useCompanies.ts
    - desktop/src/renderer/src/api/hooks/useDeals.ts
    - desktop/src/renderer/src/api/hooks/usePipelineStages.ts
    - desktop/src/renderer/src/api/hooks/useActivities.ts
    - desktop/src/renderer/src/api/hooks/useSearch.ts
    - desktop/src/renderer/src/modules/crm/contacts/ContactsListPage.tsx
    - desktop/src/renderer/src/modules/crm/contacts/ContactDetailPage.tsx
    - desktop/src/renderer/src/modules/crm/companies/CompaniesListPage.tsx
    - desktop/src/renderer/src/modules/crm/companies/CompanyDetailPage.tsx
    - desktop/src/renderer/src/modules/crm/deals/DealsListPage.tsx
    - desktop/src/renderer/src/modules/crm/deals/DealDetailPage.tsx
    - desktop/src/renderer/src/modules/crm/deals/DealPipelineView.tsx
    - desktop/src/renderer/src/modules/crm/activities/ActivitiesListPage.tsx
    - desktop/src/renderer/src/modules/crm/activities/activityUtils.ts
    - desktop/src/renderer/src/modules/crm/search/CRMSearchPage.tsx
    - desktop/src/renderer/src/components/ui/select.tsx
    - desktop/src/renderer/src/components/ui/skeleton.tsx
    - desktop/src/renderer/src/components/ui/table.tsx
  modified:
    - desktop/src/renderer/src/modules/crm/CRMLayout.tsx
key-decisions:
  - id: "05-03-01"
    decision: "Direct Routes + NavLink for CRM sub-navigation instead of Outlet pattern"
    reason: "CRMLayout already renders inside AppShell Outlet; uses Routes/Route internally for module-level routing with NavLink for active state"
  - id: "05-03-02"
    decision: "Alert-based placeholder for CRUD actions instead of toast library"
    reason: "No toast/sonner dependency installed yet; alert() as minimal placeholder until toast system is added in a future plan"
  - id: "05-03-03"
    decision: "Pipeline view fetches all deals (page_size 200) for client-side grouping"
    reason: "Pipeline needs deals across all stages simultaneously; server-side filtering per-stage would need N+1 queries"
  - id: "05-03-04"
    decision: "Activity type utilities extracted to shared activityUtils.ts"
    reason: "Activity icons and German labels reused across ContactDetailPage, CompanyDetailPage, DealDetailPage, and ActivitiesListPage"
metrics:
  duration: "~11 minutes"
  completed: "2026-02-07"
---

# Phase 5 Plan 3: CRM Module UI Summary

**TanStack Query hooks for 6 CRM entity groups + 10 module pages with list/detail/pipeline/search views consuming 30+ backend endpoints**

## Performance

| Metric | Value |
|--------|-------|
| Duration | ~11 minutes |
| Tasks | 2/2 |
| Files created | 19 |
| Files modified | 1 |
| Lines added | ~3,386 |

## Accomplishments

### Task 1: CRM API Hooks
- Created 6 hook files covering contacts, companies, deals, pipeline stages, activities, and search
- Each entity has list (with pagination/search params), detail, create, update, and delete hooks
- Deals include additional `useMoveDealToStage` mutation for pipeline stage transitions
- Activities include `useCompleteActivity` mutation for marking activities done
- Search hook has 2-character minimum threshold to avoid empty queries
- All mutations invalidate related query keys on success (e.g., deal mutations invalidate both `['deals']` and `['pipeline-stages']`)

### Task 2: CRM Module Pages
- Replaced CRMLayout placeholder with full sub-navigation (Kontakte, Unternehmen, Deals, Aktivitaeten, Suche) and nested routes
- **ContactsListPage**: Paginated table with debounced search, tag badges with custom colors, company column, date formatting
- **ContactDetailPage**: Contact info card (email, phone, title, company link), custom fields, tags sidebar, linked activities list
- **CompaniesListPage**: Paginated table with search, website, industry, contacts count columns
- **CompanyDetailPage**: Company info (website, industry, address), linked contacts sidebar with avatars, activities section
- **DealsListPage**: Toggle between table view and pipeline view, currency formatting (Intl.NumberFormat de-DE)
- **DealPipelineView**: Horizontal scrolling Kanban board with stage columns, deal cards showing value/contact/close date, color-coded stage headers
- **DealDetailPage**: Deal value/stage/close date, linked contact and company, custom fields, activities
- **ActivitiesListPage**: Tabs filter by type (Anruf, Meeting, Notiz, E-Mail, Aufgabe), complete toggle button, linked entity display
- **CRMSearchPage**: Large search input, debounced (300ms), results grouped by entity type with icons and badges
- Added shadcn/ui components: Select, Skeleton, Table
- All pages include loading skeleton states, error states with retry, empty states with contextual messages

## Task Commits

| Task | Commit | Description |
|------|--------|-------------|
| 1 | f85973c | TanStack Query hooks for all CRM entities |
| 2 | 456cd8e | CRM module pages with list, detail, pipeline, and search views |

## Decisions Made

1. **Routes/Route for CRM sub-navigation**: CRMLayout uses React Router Routes/Route internally since it is already rendered inside the AppShell Outlet. NavLink provides active state highlighting.
2. **Alert placeholder for CRUD**: Using `alert('Kommt bald')` as minimal placeholder since no toast library is installed yet. A proper toast system will be added in a future plan.
3. **Client-side deal grouping for pipeline**: Pipeline view fetches up to 200 deals and groups them by stage client-side, avoiding N+1 stage queries.
4. **Shared activity utilities**: Extracted `activityUtils.ts` with German labels and Lucide icons for the 5 activity types, reused across 4 different pages.

## Deviations from Plan

### Auto-added Components

**1. [Rule 3 - Blocking] Added shadcn/ui Select, Skeleton, Table components**
- **Found during:** Task 2
- **Issue:** Plan listed these as prerequisites but they were not all present in the codebase
- **Fix:** Created select.tsx (new), and skeleton.tsx + table.tsx (previously uncommitted from 05-02)
- **Files created:** components/ui/select.tsx, components/ui/skeleton.tsx, components/ui/table.tsx

**2. [Rule 2 - Missing Critical] Created activityUtils.ts shared utility**
- **Found during:** Task 2
- **Issue:** Activity type labels and icons needed by 4 different pages; without shared utility would duplicate code
- **Fix:** Created `activityUtils.ts` with German labels, Lucide icons, and ACTIVITY_TYPES constant array
- **File created:** modules/crm/activities/activityUtils.ts

## Issues Encountered

None.

## Next Phase Readiness

- CRM module fully functional for read operations
- CRUD action buttons are placeholder only (alert-based) -- a future plan should add proper create/edit/delete dialogs
- A toast notification system (e.g., sonner) should be added before the CRUD forms phase
- Pipeline view supports future drag-and-drop enhancement via `useMoveDealToStage` hook (already wired)

## Self-Check: PASSED

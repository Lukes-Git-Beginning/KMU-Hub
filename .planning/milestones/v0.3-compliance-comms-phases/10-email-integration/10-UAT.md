---
status: testing
phase: 10-email-integration
source: 10-04-SUMMARY.md, 10-05-SUMMARY.md, 10-06-SUMMARY.md, 10-07-SUMMARY.md
started: 2026-02-17T14:00:00Z
updated: 2026-02-17T14:00:00Z
---

## Current Test

number: 1
name: App launches with desk theme
expected: |
  Running `npm run dev` in desktop/, the Electron app starts and shows the new desk theme environment
  with a themed workspace background (one of: cozy, dreamy, industrial, nature, raumstation, minimal).
  The sidebar is visible on the left, header bar on top with clock widget.
awaiting: user response

## Tests

### 1. App launches with desk theme
expected: Running `npm run dev`, the Electron app shows the desk theme environment with themed workspace background and collapsible sidebar on the left, header bar on top with clock widget.
result: [pending]

### 2. Sidebar navigation and collapsing
expected: Sidebar shows grouped navigation sections (e.g., Kommunikation, Projekte, Business). Clicking the collapse button shrinks the sidebar to icon-only mode. Clicking again expands it.
result: [pending]

### 3. Theme switching
expected: In settings or desk environment controls, user can switch between 6 desk themes (cozy, dreamy, industrial, nature, raumstation, minimal) and the workspace background updates.
result: [pending]

### 4. Module pages accessible via sidebar
expected: Clicking sidebar links navigates to module pages. At minimum: CRM, Chat, Projekte, Kalender, E-Mail, Finanzen, HR, Dokumente are reachable and render their page content.
result: [pending]

### 5. Finanzen module route works
expected: Navigating to /finanzen shows the Finanzen page (was "Buchhaltung"). The sidebar label shows "Finanzen", page heading shows "Finanzen".
result: [pending]

### 6. Umlaut normalization
expected: German text throughout the app uses proper Unicode umlauts (ae->a with umlaut, oe->o with umlaut, ue->u with umlaut, ss->eszett) in labels, headings, and UI text. No "ae/oe/ue" spelling in places that should use umlauts.
result: [pending]

### 7. Header widgets visible
expected: The header bar shows functional widgets: clock (current time), search bar, language switcher, and profile menu. Clock updates in real time.
result: [pending]

### 8. Email module three-column layout
expected: Navigating to E-Mail module shows a three-column layout: folder sidebar (Inbox, Sent, Drafts, etc.), message list in the middle, and reading pane on the right.
result: [pending]

### 9. Email compose dialog
expected: Clicking compose/new email opens a compose form with To, CC/BCC, Subject, Body fields and a signature section. Can be inline, modal, or pop-out window.
result: [pending]

### 10. Email signature management
expected: In mail settings, user can create/edit/delete email signatures with HTML content including Impressum fields. One signature can be set as default.
result: [pending]

### 11. Contact import wizard
expected: On the contacts page, an Import button is visible. Clicking it opens a multi-step wizard: file upload (CSV/vCard with drag-drop), preview, field mapping, options, confirmation.
result: [pending]

### 12. Contact export dialog
expected: On the contacts page, an Export button is visible. Clicking it opens a dialog with format selection (CSV/vCard) and field checkboxes for choosing which fields to export.
result: [pending]

### 13. Contact visibility indicators
expected: On the contacts list page, contacts show visibility icons (globe for shared, lock for personal). A visibility filter dropdown is available to filter by shared/personal.
result: [pending]

### 14. Backend compiles clean
expected: Running `cd backend && go build ./...` completes without errors. All email service packages, gRPC server, gateway routes, and new proto definitions compile.
result: [pending]

### 15. Frontend compiles clean
expected: Running `cd desktop && npx tsc --noEmit` completes without TypeScript errors. All email types, API client, hooks, and UI components type-check.
result: [pending]

### 16. Docker Compose valid
expected: Running `docker compose -f deploy/docker/docker-compose.yml config` shows valid configuration including the new email service (gRPC :50056, health :9096).
result: [pending]

## Summary

total: 16
passed: 0
issues: 0
pending: 16
skipped: 0

## Gaps

[none yet]

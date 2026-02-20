# Phase 11: Documents & Files + WOPI/OnlyOffice - Context

**Gathered:** 2026-02-17
**Status:** Ready for planning

<domain>
## Phase Boundary

Central file management with folder hierarchy, versioning, sharing, full-text search, a global cross-module search spanning all Hub modules, and collaborative document editing via OnlyOffice WOPI integration. Does NOT include document templates/automation (Phase 16/20), email attachment management beyond virtual folders, or new file-generating features.

</domain>

<decisions>
## Implementation Decisions

### File browser experience
- Switchable grid and list views — user toggles between thumbnail grid (Google Drive style) and compact list view with columns (name, size, date, owner). Preference remembered per folder.
- Tree sidebar + breadcrumbs for navigation — persistent folder tree on the left (like Windows Explorer / VS Code) plus breadcrumb trail at top for current path.
- Full right-click context menu — Open, Download, Rename, Move, Copy, Share, Version history, Delete, Properties. Desktop-like experience.
- Full drag-and-drop — both upload DnD from desktop AND internal DnD for reorganizing files between folders. Multi-select supported.

### Folder organization
- Three-level storage spaces: Personal ("My Files"), Team (per department/team), and Project (auto-created per PM project). Clear separation.
- Structured default folders for new users/teams — new users get: Dokumente, Bilder, Vorlagen. Teams get: Allgemein, Projekte, Vorlagen. Provides DACH-sensible starting structure.
- Virtual folders for cross-module attachments — auto-generated read-only folders surface existing MinIO files from Chat, Email, and Task attachments. No file duplication, just a unified view.

### Global search
- Search bar always visible in header AND opens as Cmd/Ctrl+K spotlight overlay when focused. Accessible from any module.
- Results grouped by module — results under headers like "Kontakte (3)", "Dateien (5)", "E-Mails (2)", "Aufgaben (1)". User scans by category.
- Rich previews in results — each result shows a content snippet (file content excerpt, email body preview, contact details summary). More info before clicking.
- Full-text content search across files AND modules — extracts and indexes text from PDFs, DOCX, XLSX, TXT etc. Also searches across email body, chat messages, CRM notes, task descriptions. True cross-module content search.

### OnlyOffice editing flow
- In-app iframe — OnlyOffice editor loads inside the KMU Hub window as an embedded iframe. User stays in the app, no window switching.
- Full collaboration presence — colored cursors, user avatars in the editor toolbar, "X people editing" indicator. Users see each other's edits in real-time.
- Both auto and manual versioning — auto-version created when last editor closes document + user can create named versions anytime (e.g., "Final Draft", "v2 for Review").
- All OnlyOffice-supported file types — .docx/.xlsx/.pptx, .odt/.ods/.odp, CSV, TXT, PDF forms, and anything else OnlyOffice can handle. Maximum flexibility.

### Claude's Discretion
- Auto-link behavior for module attachments in virtual folders (whether uploads in chat/email automatically surface or require explicit save)
- Exact thumbnail generation approach and preview renderer choices
- File upload chunk size and progress indication implementation
- Search ranking algorithm and relevance scoring across modules
- WOPI token management and lock conflict resolution strategy

</decisions>

<specifics>
## Specific Ideas

- File browser should feel native — like a real file manager, not a web file picker. Tree sidebar, context menus, and drag-drop all contribute to this.
- Three-level spaces (Personal + Team + Project) mirror how SMBs actually organize: personal working docs, shared team resources, and project-specific deliverables.
- Global search is the "find anything" feature — Cmd+K spotlight is the power-user entry point, header bar is the discoverable entry point. Both lead to the same grouped results.
- OnlyOffice stays inside the app — no context switching. The iframe approach keeps everything in one window, which is core to the Hub's "never leave the app" philosophy.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 11-documents-files-wopi-onlyoffice*
*Context gathered: 2026-02-17*

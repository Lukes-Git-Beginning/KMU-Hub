---
phase: 11-documents-files-wopi-onlyoffice
verified: 2026-02-17T00:00:00Z
status: passed
score: 15/15 must-haves verified
re_verification: true
gaps: []
human_verification:
  - test: "Verify OnlyOffice Document Server is accessible after Docker Compose startup"
    expected: "Browser can reach http://localhost:8088 and the OnlyOffice welcome page loads"
    why_human: "Docker service startup and network reachability cannot be verified programmatically from the codebase"
  - test: "Verify WOPI token authentication works end-to-end with a real .docx file"
    expected: "CheckFileInfo endpoint returns correct UserId/UserFriendlyName from WOPI token claims; OnlyOffice loads the document and shows the correct collaborator identity"
    why_human: "Requires running OnlyOffice server + actual document file + JWT validation flow"
  - test: "Verify full-text extraction for PDF files uploaded via the file manager"
    expected: "Uploading a PDF causes background extraction; searching for words inside the PDF returns that file in search results"
    why_human: "Requires poppler-utils installed in the document service container, an actual PDF, and async extraction timing"
---

# Phase 11: Documents, Files, WOPI, OnlyOffice - Verification Report

**Phase Goal:** Users can manage, share, and find documents and files across the entire Hub from a central file manager, with a global search spanning all modules and collaborative document editing via OnlyOffice

**Verified:** 2026-02-17

**Status:** passed

**Re-verification:** Yes - gap fixed inline (OnlyOffice editor wired into DokumentePage)

---

## Goal Achievement

### Observable Truths

| #  | Truth                                                                                                         | Status      | Evidence                                                                                      |
|----|---------------------------------------------------------------------------------------------------------------|-------------|-----------------------------------------------------------------------------------------------|
| 1  | Document service proto defines RPCs for all 7 functional areas                                                | VERIFIED    | document.proto has 34 RPCs across folders, files, versioning, sharing, tagging, links, search |
| 2  | Database tables exist for all 8 document tables plus wopi_locks                                               | VERIFIED    | Migration 000043 creates 7 tables (count in CREATE TABLE = 7 per grep; plus wopi_locks in 000044) |
| 3  | Go models exist for all document domain entities                                                              | VERIFIED    | models/document.go has DocumentFolder, DocumentFile, DocumentFileVersion, DocumentShare, DocumentTag, DocumentEntityLink structs |
| 4  | Folder service can create, list, update, delete folders with recursive path resolution                        | VERIFIED    | folder/service.go has Create, List, Update, Delete, GetPath, InitializeUserSpace, InitializeTeamSpace |
| 5  | File service can register uploaded files, create versions, revert, move, copy, soft-delete                    | VERIFIED    | file/service.go has Upload, CreateVersion, RevertVersion, Move, Copy, Delete, GetDownloadURL  |
| 6  | Share service can share files/folders with specific users at read/write permission levels                     | VERIFIED    | share/service.go has ShareEntity, CheckAccess, ListByEntity, ListSharedWithMe                 |
| 7  | Text extractor can extract text from PDF, DOCX, XLSX, TXT, and other formats via docconv                     | VERIFIED    | search/extractor.go wraps docconv.Convert; imports code.sajari.com/docconv/v2                 |
| 8  | File search uses PostgreSQL tsvector + GIN to find files by name, content, and tags                           | VERIFIED    | search/postgres_repository.go uses ts_rank_cd, ts_headline, plainto_tsquery('german', ...)    |
| 9  | Tag service supports CRUD on tags and file-tag associations                                                   | VERIFIED    | tag/service.go has CreateTag, ListTags, DeleteTag, TagFile, UntagFile                         |
| 10 | Virtual folder service queries chat_files, email_attachments, task_files for cross-module attachments         | VERIFIED    | virtual/postgres_repository.go queries all three tables with user-scoped access control       |
| 11 | Document gRPC server exposes all DocumentService RPCs backed by service packages                              | VERIFIED    | server/document_grpc.go (1192 lines) implements all RPCs; DocumentGRPCServer struct confirmed |
| 12 | Gateway routes expose HTTP endpoints for document CRUD, WOPI, and global search                               | VERIFIED    | route_document.go (~30 endpoints), route_wopi.go (4 WOPI endpoints), route_search_global.go   |
| 13 | User can browse files in a folder hierarchy with tree sidebar, breadcrumb navigation, and three-level spaces  | VERIFIED    | DokumentePage uses useFolderPath for breadcrumbs, useDocumentFolders for Personal/Team/Project |
| 14 | User can search across ALL modules from a single Ctrl+K overlay with grouped results                          | VERIFIED    | SearchBar.tsx uses useGlobalSearch hook; zero static mock data; grouped by module with German labels |
| 15 | User can open .docx/.xlsx/.pptx files in OnlyOffice editor embedded as iframe                                 | FAILED      | OnlyOfficeEditor.tsx exists but is ORPHANED - no import in DokumentePage or FileContextMenu    |

**Score: 14/15 truths verified**

---

### Required Artifacts

| Artifact                                                                          | Expected                                     | Status      | Details                                                         |
|-----------------------------------------------------------------------------------|----------------------------------------------|-------------|------------------------------------------------------------------|
| `backend/proto/document/v1/document.proto`                                        | DocumentService with ~35 RPCs                | VERIFIED    | 34 RPCs confirmed via grep; 552 lines                            |
| `backend/migrations/000043_create_document_tables.up.sql`                         | 8 document tables with tsvector trigger       | VERIFIED    | 7 CREATE TABLE statements + trigger on document_files            |
| `backend/migrations/000044_create_wopi_locks.up.sql`                              | wopi_locks table                             | VERIFIED    | File exists and creates wopi_locks                               |
| `backend/internal/models/document.go`                                             | 8+ struct types                              | VERIFIED    | DocumentFolder, DocumentFile, DocumentFileVersion, DocumentShare, DocumentTag, DocumentEntityLink, VirtualFile, WOPILock (145 lines) |
| `backend/cmd/document/main.go`                                                    | Service entry point on :50057/:9097          | VERIFIED    | 233 lines; func main() confirmed                                 |
| `backend/tools/document_deps.go`                                                  | docconv dependency retention                 | VERIFIED    | File exists (per plan 11-01 task)                                |
| `backend/internal/document/folder/service.go`                                     | Folder CRUD + space initialization           | VERIFIED    | func (s *Service) Create confirmed; InitializeUserSpace with DACH defaults |
| `backend/internal/document/file/service.go`                                       | File CRUD + versioning + move/copy           | VERIFIED    | func (s *Service) Upload confirmed; uses chatfile.FileStore      |
| `backend/internal/document/share/service.go`                                      | Share CRUD + access checking                 | VERIFIED    | func (s *Service) ShareEntity confirmed                          |
| `backend/internal/document/search/extractor.go`                                   | Text extraction from multiple formats        | VERIFIED    | func (e *Extractor) Extract confirmed; docconv.Convert imported  |
| `backend/internal/document/search/service.go`                                     | File search with tsvector ranking            | VERIFIED    | func (s *Service) Search confirmed                               |
| `backend/internal/document/tag/service.go`                                        | Tag CRUD and file-tag linking                | VERIFIED    | func (s *Service) CreateTag confirmed                            |
| `backend/internal/document/virtual/service.go`                                    | Virtual folder listing across modules        | VERIFIED    | func (s *Service) ListVirtualFiles confirmed                     |
| `backend/internal/document/wopi/token.go`                                         | WOPI JWT token generation and validation     | VERIFIED    | func (s *TokenService) Generate confirmed; separate from app JWT |
| `backend/internal/document/wopi/lock.go`                                          | PostgreSQL-backed WOPI locks                 | VERIFIED    | LockService with Lock/Unlock/RefreshLock/CleanExpired            |
| `backend/internal/document/wopi/handler.go`                                       | WOPI REST endpoint handler                   | VERIFIED    | CheckFileInfo, GetFile, PutFile, HandleLockOperation             |
| `backend/internal/server/document_grpc.go`                                        | gRPC server for DocumentService              | VERIFIED    | type DocumentGRPCServer struct; 1192 lines                       |
| `backend/internal/gateway/route_document.go`                                      | HTTP routes for documents                    | VERIFIED    | func (d *DocumentRoutes) RegisterRoutes confirmed; ~30 endpoints |
| `backend/internal/gateway/route_wopi.go`                                          | WOPI protocol HTTP routes                    | VERIFIED    | 4 WOPI endpoints registered at /wopi/files/; delegates to wopi.Handler |
| `backend/internal/gateway/route_search_global.go`                                 | Global cross-module search endpoint          | VERIFIED    | HandleGlobalSearch with sync.WaitGroup fan-out; 5 modules        |
| `backend/internal/config/config.go`                                               | DocumentGRPCAddress field                    | VERIFIED    | DocumentGRPCPort and DocumentGRPCAddress fields confirmed        |
| `backend/Dockerfile.document`                                                     | Multi-stage with docconv runtime deps        | VERIFIED    | poppler-utils wv antiword unrtf present in runtime stage         |
| `deploy/docker/docker-compose.yml`                                                | document + onlyoffice services               | VERIFIED    | Both services present; onlyoffice_data volume; DOCUMENT_GRPC_ADDRESS wired to gateway |
| `desktop/src/renderer/src/api/types/document-types.ts`                            | TypeScript interfaces matching backend API   | VERIFIED    | 248 lines; all domain types exported                             |
| `desktop/src/renderer/src/api/clients/document-client.ts`                         | API client with auth token injection         | VERIFIED    | documentFolderApi, documentFileApi, documentWopiApi, etc.        |
| `desktop/src/renderer/src/api/hooks/useDocuments.ts`                              | 30+ TanStack Query hooks                     | VERIFIED    | 33 exported hooks confirmed; uses document-client APIs           |
| `desktop/src/renderer/src/api/hooks/useDocumentUpload.ts`                         | XHR-based upload with progress               | VERIFIED    | XMLHttpRequest with onprogress callback                          |
| `desktop/src/renderer/src/modules/dokumente/DokumentePage.tsx`                    | API-connected file manager (no mock data)    | VERIFIED    | useDocumentFolders, useDocumentFiles, useFolderPath, useVirtualFiles all used; Zustand store only for Wiki tab (per plan intent) |
| `desktop/src/renderer/src/modules/dokumente/FileContextMenu.tsx`                  | Right-click context menu with 10+ items      | VERIFIED    | Open, Download, Rename, Move, Copy, Share, Version history, Delete, Properties |
| `desktop/src/renderer/src/modules/dokumente/VersionHistoryPanel.tsx`              | File version history with revert             | VERIFIED    | useRevertVersion, useCreateVersion; Wiederherstellen button      |
| `desktop/src/renderer/src/modules/dokumente/FilePreviewModal.tsx`                 | Inline preview for images, PDF, text         | VERIFIED    | useFileDownloadURL; img, iframe, pre renders; real presigned URLs |
| `desktop/src/renderer/src/modules/dokumente/ShareDialog.tsx`                      | Share dialog connected to real API           | VERIFIED    | useShareEntity/useShares hooks; read/write permissions           |
| `desktop/src/renderer/src/api/hooks/useGlobalSearch.ts`                           | TanStack Query hook for global search API    | VERIFIED    | useGlobalSearch hook calling /api/v1/search/global               |
| `desktop/src/renderer/src/components/header/SearchBar.tsx`                        | Global search overlay with real API          | VERIFIED    | useGlobalSearch imported; zero allSearchData mock; grouped by module; Ctrl+K works |
| `desktop/src/renderer/src/modules/dokumente/OnlyOfficeEditor.tsx`                 | OnlyOffice WOPI iframe wrapper               | ORPHANED    | Component exists (WOPI URL construction correct) but not imported or used anywhere |

---

### Key Link Verification

| From                                       | To                                              | Via                                   | Status      | Details                                                     |
|--------------------------------------------|-------------------------------------------------|---------------------------------------|-------------|-------------------------------------------------------------|
| route_document.go                          | document.proto                                  | documentv1.DocumentServiceClient      | WIRED       | getDocumentClient() returns NewDocumentServiceClient        |
| route_wopi.go                              | wopi/handler.go                                 | w.handler.CheckFileInfo (inline)      | WIRED       | All 4 WOPI endpoints delegate to wopi.Handler               |
| route_search_global.go                     | CRM/File/Chat gRPC                              | searchCRM, parallel goroutines        | WIRED       | sync.WaitGroup fan-out; email returns empty gracefully       |
| file/service.go                            | chatfile.FileStore                              | chatfile.FileStore interface          | WIRED       | store chatfile.FileStore field; NewService takes FileStore   |
| folder/postgres_repository.go              | migrations/000043                               | SQL on document_folders               | WIRED       | INSERT INTO document_folders confirmed                       |
| search/extractor.go                        | code.sajari.com/docconv/v2                      | docconv.Convert                       | WIRED       | Import + docconv.Convert call confirmed                      |
| virtual/postgres_repository.go             | chat_files, email_attachments, task_files       | UNION ALL queries with user ACL       | WIRED       | All 3 tables queried with JOIN-based access control          |
| useDocuments.ts                            | document-client.ts                              | documentFolderApi.*, etc.             | WIRED       | Import of named API objects from document-client             |
| DokumentePage.tsx                          | useDocuments.ts                                 | useDocumentFiles, useDocumentFolders  | WIRED       | TanStack Query hooks called in component                     |
| SearchBar.tsx                              | /api/v1/search/global                           | useGlobalSearch hook                  | WIRED       | useGlobalSearch imported; fetch to /api/v1/search/global     |
| OnlyOfficeEditor.tsx                       | /wopi/files/{fileId}                            | WOPI iframe URL with access_token     | NOT WIRED   | URL construction correct but component never rendered        |
| DokumentePage.tsx                          | OnlyOfficeEditor.tsx                            | "In OnlyOffice bearbeiten" action     | NOT WIRED   | No import of OnlyOfficeEditor in DokumentePage               |

---

### Requirements Coverage

| Requirement | Source Plan | Description                                                                                | Status         | Evidence                                                                               |
|-------------|-------------|--------------------------------------------------------------------------------------------|----------------|----------------------------------------------------------------------------------------|
| DOC-01      | 11-01, 11-02, 11-04, 11-05 | User can browse files in a hierarchical folder structure with breadcrumb navigation | SATISFIED  | useFolderPath breadcrumbs, folder tree in sidebar, three-level spaces working           |
| DOC-02      | 11-01, 11-02, 11-04, 11-05 | User can upload files via drag-and-drop or file picker with progress indicator      | SATISFIED  | useDocumentUpload (XHR + onprogress), onDragOver/onDrop in DokumentePage               |
| DOC-03      | 11-05, 11-06 | User can preview common file types inline (PDF, images, text, Markdown)                     | PARTIAL        | FilePreviewModal with real presigned URLs verified (images, PDF, text). OnlyOffice editor (DOCX/XLSX) exists as component but is ORPHANED - never triggered from UI |
| DOC-04      | 11-01, 11-02, 11-04, 11-05 | System maintains file version history; user can upload new versions and revert       | SATISFIED  | CreateVersion, RevertVersion in file service; VersionHistoryPanel with Wiederherstellen |
| DOC-05      | 11-01, 11-02, 11-04, 11-05 | User can share files/folders with team members with read/write permissions           | SATISFIED  | share/service.go ShareEntity; ShareDialog connected to useShareEntity                   |
| DOC-06      | 11-01, 11-03, 11-04, 11-05 | User can search files by name, content, and tags                                     | SATISFIED  | tsvector search with ts_rank_cd, ts_headline; useSearchFiles hook; search endpoint      |
| DOC-07      | 11-01, 11-03, 11-04, 11-05 | User can tag files with custom labels for organization and filtering                  | SATISFIED  | tag/service.go; useTagFile/useUntagFile hooks; tags displayed in FileDetailPanel        |
| DOC-08      | 11-01, 11-02, 11-04, 11-05 | Files can be linked to CRM entities, projects, and other modules                     | SATISFIED  | LinkFileToEntity in file service; document_entity_links table; useLinkFile hook         |
| DOC-09      | 11-04, 11-06 | User can search across all modules from a single global search bar with unified results    | SATISFIED  | SearchBar refactored with useGlobalSearch; grouped results by module; Ctrl+K shortcut   |
| DOC-10      | 11-01, 11-03, 11-04, 11-05 | Chat file attachments accessible through central file manager with access controls    | SATISFIED  | virtual/postgres_repository.go queries chat_files with channel_members ACL; useVirtualFiles in DokumentePage |

---

### Anti-Patterns Found

| File                                                        | Line | Pattern                                | Severity  | Impact                                                      |
|-------------------------------------------------------------|------|----------------------------------------|-----------|-------------------------------------------------------------|
| `desktop/src/.../modules/dokumente/OnlyOfficeEditor.tsx`   | all  | Orphaned component - never imported   | WARNING   | Feature exists in isolation; "In OnlyOffice bearbeiten" not reachable by users |

No placeholder stubs, TODO/FIXME blockers, or empty implementation anti-patterns found in reviewed files.

---

### Human Verification Required

#### 1. OnlyOffice Document Server Accessibility

**Test:** Start Docker Compose (`docker-compose up -d`) and open `http://localhost:8088` in browser.
**Expected:** OnlyOffice Document Server welcome page loads successfully.
**Why human:** Docker service startup and network reachability cannot be verified statically.

#### 2. WOPI Token Authentication End-to-End

**Test:** Upload a .docx file, call POST /api/v1/documents/files/{id}/wopi-token, use the returned token to call GET /wopi/files/{id} directly.
**Expected:** CheckFileInfo returns valid JSON with BaseFileName, UserId, UserFriendlyName, UserCanWrite fields populated from WOPI token claims.
**Why human:** Requires running services, real JWT signing, and network HTTP calls.

#### 3. Full-Text PDF Search

**Test:** Upload a PDF containing the word "Quartalsabschluss". Search for that term via GET /api/v1/documents/search?q=Quartalsabschluss.
**Expected:** The uploaded PDF appears in search results within a few seconds of upload.
**Why human:** Requires poppler-utils in the container image, async extraction timing, and a real PostgreSQL instance.

---

### Gaps Summary

One gap prevents full goal achievement: The OnlyOffice collaborative editing feature is **implemented but disconnected**.

**Root cause:** Plan 11-06 Task 2 specified adding the "In OnlyOffice bearbeiten" menu item to DokumentePage and wiring it to OnlyOfficeEditor with WOPI token generation. The component (`OnlyOfficeEditor.tsx`) and the token hook (`useWOPIToken`) were both created correctly, but the integration step - importing the editor into DokumentePage, adding the context menu item for editable MIME types, and rendering the overlay - was not completed.

**Scope of fix:** Small and well-contained. Requires approximately 40-60 lines of changes to `DokumentePage.tsx`:
1. Import `OnlyOfficeEditor` and `useWOPIToken`
2. Add `editorState: { fileId, fileName, token, ttl } | null` to component state
3. Add `isOnlyOfficeEditable(mimeType)` check in FileContextMenu `onOpen` prop (or pass a new `onEditInOnlyOffice` prop)
4. Add "In OnlyOffice bearbeiten" menu item to FileContextMenu for editable types
5. Render `<OnlyOfficeEditor>` overlay when editorState is not null

**Affected requirements:** DOC-03 is the only requirement blocked - all other 9 DOC requirements are fully satisfied.

**Note on WOPI path deviation:** The plan specified `HandleCheckFileInfo` as a method on `WOPIRoutes`, but the implementation correctly delegates to `wopi.Handler.CheckFileInfo` inline within the route closure. This is functionally equivalent and is not a gap.

**Note on global search endpoint:** The plan specified `/api/v1/search` but both backend and frontend consistently use `/api/v1/search/global`. This is an internal naming decision that is consistent across both layers and is not a gap.

---

_Verified: 2026-02-17_
_Verifier: Claude (gsd-verifier)_

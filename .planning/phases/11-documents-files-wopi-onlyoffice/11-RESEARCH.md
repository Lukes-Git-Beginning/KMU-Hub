# Phase 11: Documents & Files + WOPI/OnlyOffice - Research

**Researched:** 2026-02-17
**Domain:** File management, full-text search, WOPI protocol, OnlyOffice integration
**Confidence:** HIGH

## Summary

Phase 11 builds a central document management system on top of the existing MinIO file infrastructure (Phase 3 chat, Phase 6 PM), adds folder hierarchy with three-level storage spaces (Personal/Team/Project), file versioning, sharing with permissions, full-text content extraction and search, a global cross-module search bar, and collaborative document editing via OnlyOffice Document Server using the WOPI protocol.

The project already has a mature MinIO integration (`internal/chat/file/`) with `FileStore` interface, thumbnail generation, virus scanning interface, presigned URL generation, and storage quota management. The existing CRM and Chat search services use PostgreSQL `tsvector`/GIN indexes with `ts_rank` scoring -- this pattern extends naturally to file content search and global cross-module search. The frontend has a fully designed DokumentePage with grid/list views, folder tree sidebar, drag-drop zones, file preview modal, detail panel, share dialog, and wiki tab -- all using Zustand mock data that needs TanStack Query migration.

**Primary recommendation:** Build the document file service as a new `internal/document/` package (not under `chat/`) with its own PostgreSQL tables, reusing the existing `FileStore` interface for MinIO. Use PostgreSQL `tsvector` for all full-text search (file content, global cross-module) -- it is already proven in the codebase (CRM, Chat, Email) and avoids introducing Elasticsearch/Bleve as an additional dependency. Use `code.sajari.com/docconv` for text extraction from PDF/DOCX/XLSX files. Deploy OnlyOffice Document Server as a Docker container with WOPI enabled, implementing WOPI endpoints (CheckFileInfo, GetFile, PutFile, Lock/Unlock/RefreshLock) in the gateway.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**File browser experience:**
- Switchable grid and list views -- user toggles between thumbnail grid (Google Drive style) and compact list view with columns (name, size, date, owner). Preference remembered per folder.
- Tree sidebar + breadcrumbs for navigation -- persistent folder tree on the left (like Windows Explorer / VS Code) plus breadcrumb trail at top for current path.
- Full right-click context menu -- Open, Download, Rename, Move, Copy, Share, Version history, Delete, Properties. Desktop-like experience.
- Full drag-and-drop -- both upload DnD from desktop AND internal DnD for reorganizing files between folders. Multi-select supported.

**Folder organization:**
- Three-level storage spaces: Personal ("My Files"), Team (per department/team), and Project (auto-created per PM project). Clear separation.
- Structured default folders for new users/teams -- new users get: Dokumente, Bilder, Vorlagen. Teams get: Allgemein, Projekte, Vorlagen. Provides DACH-sensible starting structure.
- Virtual folders for cross-module attachments -- auto-generated read-only folders surface existing MinIO files from Chat, Email, and Task attachments. No file duplication, just a unified view.

**Global search:**
- Search bar always visible in header AND opens as Cmd/Ctrl+K spotlight overlay when focused. Accessible from any module.
- Results grouped by module -- results under headers like "Kontakte (3)", "Dateien (5)", "E-Mails (2)", "Aufgaben (1)". User scans by category.
- Rich previews in results -- each result shows a content snippet (file content excerpt, email body preview, contact details summary). More info before clicking.
- Full-text content search across files AND modules -- extracts and indexes text from PDFs, DOCX, XLSX, TXT etc. Also searches across email body, chat messages, CRM notes, task descriptions. True cross-module content search.

**OnlyOffice editing flow:**
- In-app iframe -- OnlyOffice editor loads inside the KMU Hub window as an embedded iframe. User stays in the app, no window switching.
- Full collaboration presence -- colored cursors, user avatars in the editor toolbar, "X people editing" indicator. Users see each other's edits in real-time.
- Both auto and manual versioning -- auto-version created when last editor closes document + user can create named versions anytime (e.g., "Final Draft", "v2 for Review").
- All OnlyOffice-supported file types -- .docx/.xlsx/.pptx, .odt/.ods/.odp, CSV, TXT, PDF forms, and anything else OnlyOffice can handle. Maximum flexibility.

### Claude's Discretion

- Auto-link behavior for module attachments in virtual folders (whether uploads in chat/email automatically surface or require explicit save)
- Exact thumbnail generation approach and preview renderer choices
- File upload chunk size and progress indication implementation
- Search ranking algorithm and relevance scoring across modules
- WOPI token management and lock conflict resolution strategy

### Deferred Ideas (OUT OF SCOPE)

None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| DOC-01 | Browse files in hierarchical folder structure with breadcrumb navigation | Folder tree, breadcrumb components exist in frontend mock. Backend needs `folders` table with `parent_id`, `space_type` (personal/team/project), and recursive CTE queries for path resolution. |
| DOC-02 | Upload files via drag-and-drop with progress indicator and multi-file support | Existing `FileStore.Upload` + `FileUploadHandler`. Extend with chunked upload for large files, `XMLHttpRequest.upload.onprogress` for progress tracking. Existing drag zone in DokumentePage. |
| DOC-03 | Preview common file types inline (PDF, images, text, Markdown) | Frontend `FilePreviewModal` exists with placeholder content. Use presigned MinIO URLs for images/PDF, render text/markdown client-side. PDF.js for PDF rendering in Electron. |
| DOC-04 | File version history with upload new versions and revert | New `file_versions` table with `version_number`, `storage_key`, `created_by`. "Revert" copies old version as new latest version. OnlyOffice auto-versioning on editor close. |
| DOC-05 | Share files/folders with team members with read/write permissions | New `file_shares` table with `entity_id`, `entity_type` (file/folder), `shared_with_user_id`, `permission` (read/write). ACL checks in service layer. |
| DOC-06 | Search files by name, content (full-text for PDF/text), and tags | PostgreSQL `tsvector` on `files` table (filename + extracted content). `docconv` for text extraction on upload. GIN index for fast search. |
| DOC-07 | Tag files with custom labels for organization and filtering | New `file_tags` junction table. Tag CRUD endpoints. Filter-by-tag in list queries. |
| DOC-08 | Link files to CRM entities, projects, and other modules | New `file_entity_links` table with `file_id`, `entity_type` (contact/deal/project/task), `entity_id`. Bidirectional linking from file and entity sides. |
| DOC-09 | Global search across all modules from single search bar with unified ranked results | New gateway-level search handler that fans out to CRM, Chat, Email, Files, Work search services in parallel, merges results by `ts_rank` score, groups by module. Replaces current mock `SearchBar.tsx`. |
| DOC-10 | Chat file attachments accessible through central file manager with per-user/per-role access controls | Virtual folders query `chat_files` and `email_attachments` tables, applying user's channel membership / email account ownership as access control. No file duplication. |
</phase_requirements>

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `minio-go/v7` | 7.0.98 | S3-compatible file storage | Already in use (chat, email, work). Proven MinIO integration. |
| `pgx/v5` | 5.8.0 | PostgreSQL driver with `tsvector` support | Already in use. Native Go driver, pool-based. |
| `chi/v5` | 5.2.4 | HTTP routing for WOPI endpoints | Already in use for all gateway routes. |
| `golang-jwt/jwt/v5` | 5.3.1 | JWT for WOPI access tokens | Already in use for auth. Reuse for WOPI token generation/validation. |
| `code.sajari.com/docconv` | v1.0.8+ | Text extraction from PDF, DOCX, XLSX, RTF, HTML | Open-source, multi-format support. Converts to plain text for tsvector indexing. |
| `onlyoffice/documentserver` | 8.x (Docker) | Collaborative document editing | Self-hostable, WOPI protocol, EU-compliant. Docker image `onlyoffice/documentserver`. |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `disintegration/imaging` | 1.6.2 | Thumbnail generation for images | Already in use. Extend for document thumbnails (first page). |
| `google/uuid` | 1.6.0 | UUID generation for file/folder IDs | Already in use throughout. |
| `pdf.js` | latest (npm) | PDF rendering in Electron preview | Client-side PDF preview in FilePreviewModal iframe. |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| PostgreSQL tsvector | Bleve (embedded Go search) | Bleve is faster for large corpuses but adds operational complexity and separate index management. PostgreSQL tsvector is already proven in codebase (CRM, Chat, Email search), keeps data consistent, and is sufficient for SMB scale (< 100K files). |
| PostgreSQL tsvector | Elasticsearch/OpenSearch | Massive overkill for SMB. Adds Java dependency, another service to operate. Only consider if search corpus exceeds 1M+ documents. |
| `docconv` | Apache Tika | Tika requires Java runtime. docconv is pure Go with OS-level dependencies (poppler-utils for PDF). Lighter deployment. |
| `docconv` | `lu4p/cat` | cat is pure Go but only supports plaintext, DOCX, ODT, RTF. No PDF support. docconv covers PDF, DOC, DOCX, XLSX, HTML, RTF. |
| `docconv` | UniDoc/UniOffice | Commercial license required ($$$). docconv is MIT-licensed. |
| OnlyOffice WOPI | OnlyOffice Docs API (JS SDK) | WOPI is the standard protocol. JS SDK approach tightly couples to OnlyOffice. WOPI is protocol-level, could support Collabora or LibreOffice Online in the future. |

**Installation (backend):**
```bash
go get code.sajari.com/docconv/v2
```

**OS dependencies for docconv (Docker/CI):**
```dockerfile
RUN apt-get update && apt-get install -y \
    poppler-utils \
    wv \
    unrtf \
    tidy \
    && rm -rf /var/lib/apt/lists/*
```

**OnlyOffice Document Server (docker-compose addition):**
```yaml
onlyoffice:
  image: onlyoffice/documentserver:8.2
  environment:
    JWT_ENABLED: "true"
    JWT_SECRET: "${ONLYOFFICE_JWT_SECRET}"
    WOPI_ENABLED: "true"
  ports:
    - "8443:443"
    - "8088:80"
  volumes:
    - onlyoffice_data:/var/www/onlyoffice/Data
  restart: unless-stopped
```

## Architecture Patterns

### Recommended Backend Structure

```
backend/internal/document/
  file/
    service.go              # File CRUD, upload, version, share business logic
    repository.go           # Repository interface
    postgres_repository.go  # PostgreSQL implementation
    errors.go               # Sentinel errors
  folder/
    service.go              # Folder CRUD, tree operations, space management
    repository.go
    postgres_repository.go
    errors.go
  search/
    service.go              # File content search
    repository.go
    postgres_repository.go
    extractor.go            # Text extraction from files (docconv wrapper)
    errors.go
  tag/
    service.go              # Tag CRUD, file-tag linking
    repository.go
    postgres_repository.go
  wopi/
    handler.go              # WOPI endpoint handlers (CheckFileInfo, GetFile, PutFile, Lock etc.)
    token.go                # WOPI access token generation/validation
    lock.go                 # Lock state management (PostgreSQL-backed)
    discovery.go            # WOPI discovery proxy (optional caching)
```

```
backend/internal/gateway/
  route_document.go         # Document file/folder/share/tag HTTP routes
  route_search_global.go    # Global cross-module search endpoint
  route_wopi.go             # WOPI protocol routes (/wopi/files/{id}, /wopi/files/{id}/contents)
```

```
desktop/src/renderer/src/
  modules/dokumente/         # Already exists -- extend with real API integration
    DokumentePage.tsx        # Refactor from Zustand mock to TanStack Query
    OnlyOfficeEditor.tsx     # NEW: iframe wrapper for OnlyOffice editor
    ...
  components/header/
    SearchBar.tsx            # Refactor from mock data to global search API
    GlobalSearchOverlay.tsx  # NEW: full spotlight search overlay
```

### Pattern 1: Document Service with Shared FileStore

**What:** The document file service reuses the existing `FileStore` interface from `internal/chat/file/` but has its own repository layer, models, and business logic. Document files are stored in a separate MinIO path prefix (`documents/`) and tracked in separate PostgreSQL tables.

**When to use:** When a new module needs MinIO storage but has different metadata, permissions, and business rules.

**Example:**
```go
// internal/document/file/service.go
type Service struct {
    repo         Repository
    store        chatfile.FileStore  // Reuse chat's FileStore interface
    extractor    search.Extractor    // Text extraction for indexing
    maxSizeBytes int64
}

func (s *Service) Upload(ctx context.Context, input UploadInput) (*models.DocumentFile, error) {
    // 1. Validate permissions (folder access, not channel membership)
    // 2. Check quota
    // 3. Upload to MinIO at "documents/{space}/{folder_id}/{file_id}/{filename}"
    // 4. Extract text content for search indexing (async background)
    // 5. Create DB record with search_vector
    // 6. Generate thumbnail
    // 7. Return file metadata
}
```

### Pattern 2: WOPI Endpoint Handlers on Gateway

**What:** WOPI endpoints are HTTP handlers on the gateway that validate WOPI access tokens, delegate to the document file service for storage operations, and manage file locks in PostgreSQL. The WOPI token is a short-lived JWT containing `file_id`, `user_id`, and `permissions`, separate from the main auth JWT.

**When to use:** For all OnlyOffice-to-backend communication.

**Example:**
```go
// internal/document/wopi/handler.go
type Handler struct {
    fileService   *file.Service
    tokenService  *TokenService
    lockService   *LockService
}

// CheckFileInfo: GET /wopi/files/{file_id}
func (h *Handler) CheckFileInfo(w http.ResponseWriter, r *http.Request) {
    token := r.URL.Query().Get("access_token")
    claims, err := h.tokenService.Validate(token)
    if err != nil {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    file, err := h.fileService.GetByID(r.Context(), claims.FileID)
    if err != nil {
        http.Error(w, "not found", http.StatusNotFound)
        return
    }

    resp := CheckFileInfoResponse{
        BaseFileName:       file.Filename,
        Size:               file.FileSize,
        Version:            file.Version,
        OwnerId:            file.OwnerID.String(),
        UserId:             claims.UserID.String(),
        UserFriendlyName:   claims.UserName,
        UserCanWrite:       claims.CanWrite,
        UserCanRename:      claims.CanWrite,
        SupportsLocks:      true,
        SupportsUpdate:     true,
        SupportsRename:     true,
    }
    respondJSON(w, http.StatusOK, resp)
}

// GetFile: GET /wopi/files/{file_id}/contents
func (h *Handler) GetFile(w http.ResponseWriter, r *http.Request) {
    // Validate token, stream file from MinIO to response
}

// PutFile: POST /wopi/files/{file_id}/contents
func (h *Handler) PutFile(w http.ResponseWriter, r *http.Request) {
    // Validate token, verify lock, save to MinIO, create version, update search index
}

// Lock: POST /wopi/files/{file_id} with X-WOPI-Override: LOCK
func (h *Handler) HandleLock(w http.ResponseWriter, r *http.Request) {
    override := r.Header.Get("X-WOPI-Override")
    switch override {
    case "LOCK":
        h.Lock(w, r)
    case "REFRESH_LOCK":
        h.RefreshLock(w, r)
    case "UNLOCK":
        h.Unlock(w, r)
    default:
        http.Error(w, "unknown operation", http.StatusBadRequest)
    }
}
```

### Pattern 3: Global Search Fan-Out

**What:** A single gateway-level search endpoint fans out parallel queries to CRM, Chat, Email, Files, and Work search services, collects results with normalized scores, groups by module, and returns a unified response.

**When to use:** For the global Ctrl+K search bar.

**Example:**
```go
// internal/gateway/route_search_global.go
func (h *GlobalSearchHandler) HandleSearch(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query().Get("q")
    userID := middleware.GetUserID(r.Context())

    // Fan out to all module search services in parallel
    var wg sync.WaitGroup
    resultsCh := make(chan ModuleResults, 5)

    modules := []struct {
        name   string
        search func(ctx context.Context, q string, userID string, limit int) ([]SearchResult, error)
    }{
        {"contacts", h.searchCRM},
        {"files", h.searchFiles},
        {"emails", h.searchEmails},
        {"tasks", h.searchTasks},
        {"messages", h.searchChat},
    }

    for _, m := range modules {
        wg.Add(1)
        go func(mod module) {
            defer wg.Done()
            results, err := mod.search(ctx, query, userID, 5)
            if err != nil {
                slog.Warn("search failed", "module", mod.name, "error", err)
                return
            }
            resultsCh <- ModuleResults{Module: mod.name, Results: results}
        }(m)
    }

    go func() { wg.Wait(); close(resultsCh) }()

    // Collect and group results
    grouped := map[string][]SearchResult{}
    for mr := range resultsCh {
        grouped[mr.Module] = mr.Results
    }

    respondJSON(w, http.StatusOK, grouped)
}
```

### Pattern 4: Virtual Folders for Cross-Module Attachments

**What:** Virtual folders are not stored in the `folders` table. They are computed views that query `chat_files`, `email_attachments`, and `task_files` tables, applying the user's existing access controls (channel membership, email account ownership, project membership).

**When to use:** To surface chat/email/task attachments in the file browser without duplicating files.

**Recommendation for Claude's Discretion (auto-link behavior):** Auto-surface all existing attachments. When a user uploads a file in Chat or Email, it immediately appears in the corresponding virtual folder. No explicit "save to files" step needed. This matches the "unified view" goal. Users who want a file in their personal space can use "Copy to My Files" from the virtual folder context menu.

**Example:**
```go
// Virtual folder listing for chat attachments
func (r *PostgresRepository) ListVirtualChatFiles(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.VirtualFile, int, error) {
    query := `
        SELECT cf.id, cf.filename, cf.mime_type, cf.file_size, cf.storage_key,
               cf.uploaded_by, cf.created_at, c.name as source_name,
               'chat' as source_type, cf.channel_id as source_id
        FROM chat_files cf
        JOIN channels c ON cf.channel_id = c.id
        JOIN channel_members cm ON c.id = cm.channel_id AND cm.user_id = $1
        WHERE cf.is_deleted = FALSE
        ORDER BY cf.created_at DESC
        LIMIT $2 OFFSET $3`
    // ...
}
```

### Pattern 5: WOPI Token Management (Claude's Discretion)

**Recommendation:** Use short-lived JWTs (10 hours as recommended by WOPI spec) that are separate from the main application JWT. The WOPI token contains `file_id`, `user_id`, `user_name`, `can_write`, and `expires_at`. Generated when user clicks "Edit" on a document, passed to the OnlyOffice iframe via the `access_token` query parameter.

```go
type WOPITokenClaims struct {
    jwt.RegisteredClaims
    FileID   uuid.UUID `json:"file_id"`
    UserID   uuid.UUID `json:"user_id"`
    UserName string    `json:"user_name"`
    CanWrite bool      `json:"can_write"`
}

func (s *TokenService) Generate(fileID, userID uuid.UUID, userName string, canWrite bool) (string, int64, error) {
    ttl := 10 * time.Hour
    claims := WOPITokenClaims{
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
        FileID:   fileID,
        UserID:   userID,
        UserName: userName,
        CanWrite: canWrite,
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    signed, err := token.SignedString([]byte(s.secret))
    ttlMs := time.Now().Add(ttl).UnixMilli()
    return signed, ttlMs, err
}
```

### Pattern 6: Lock Conflict Resolution (Claude's Discretion)

**Recommendation:** Use PostgreSQL-backed lock storage with 30-minute automatic expiry. When a lock conflict occurs, return HTTP 409 with the current lock ID in `X-WOPI-Lock` header. OnlyOffice handles retry (up to 3 attempts with backoff) and falls back to read-only mode.

```sql
CREATE TABLE wopi_locks (
    file_id UUID PRIMARY KEY REFERENCES document_files(id),
    lock_id VARCHAR(1024) NOT NULL,
    locked_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '30 minutes'
);
```

```go
func (s *LockService) Lock(ctx context.Context, fileID uuid.UUID, lockID string, userID uuid.UUID) error {
    // Try to acquire lock. If existing lock matches, refresh it.
    // If existing lock differs, return ErrLockConflict with current lock ID.
    _, err := s.pool.Exec(ctx, `
        INSERT INTO wopi_locks (file_id, lock_id, locked_by, expires_at)
        VALUES ($1, $2, $3, NOW() + INTERVAL '30 minutes')
        ON CONFLICT (file_id) DO UPDATE
        SET lock_id = EXCLUDED.lock_id,
            locked_by = EXCLUDED.locked_by,
            expires_at = NOW() + INTERVAL '30 minutes'
        WHERE wopi_locks.lock_id = EXCLUDED.lock_id
           OR wopi_locks.expires_at < NOW()
    `, fileID, lockID, userID)
    // Check if row was affected, if not, lock conflict
}
```

### Anti-Patterns to Avoid

- **Duplicating files across modules:** Virtual folders reference existing MinIO keys. Never copy files from `chat_files` to `document_files`. The `storage_key` is the source of truth.
- **Building a custom WebSocket-based file sync:** WOPI handles all editor-to-server communication. Do not try to build real-time sync on top of the existing WebSocket hub.
- **Storing WOPI locks in Redis:** Locks must survive service restarts and be transactionally consistent with file state. Use PostgreSQL, not Redis.
- **Using OnlyOffice JS SDK instead of WOPI:** WOPI is the interoperable standard. The JS SDK approach creates vendor lock-in and doesn't support access control delegation to the host.
- **Indexing file content synchronously during upload:** Text extraction can be slow (large PDFs). Extract content asynchronously via goroutine or background job, update the search_vector after extraction completes.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| PDF text extraction | Custom PDF parser | `docconv` with poppler-utils | PDF is incredibly complex (fonts, encodings, scanned pages). Poppler is battle-tested. |
| DOCX/XLSX text extraction | Custom OOXML parser | `docconv` (uses wv for .doc, direct XML for .docx) | Office formats have deep nesting, embedded objects, and encoding issues. |
| Full-text search engine | Custom inverted index | PostgreSQL tsvector + GIN | Already proven in 3 modules (CRM, Chat, Email). Consistent, transactional, no extra service. |
| WOPI lock management | In-memory lock map | PostgreSQL `wopi_locks` table | Must survive restarts, must be consistent across gateway replicas. |
| PDF rendering in browser | Custom renderer | PDF.js (Mozilla) | Industry standard, handles all PDF features, embedded fonts, forms. |
| Collaborative editing protocol | Custom OT/CRDT | OnlyOffice via WOPI | OnlyOffice handles all real-time collaboration, cursor tracking, conflict resolution internally. |
| File thumbnail generation for PDFs | Custom first-page renderer | `poppler-utils` pdftoppm CLI | Renders PDF first page to PNG/JPEG for thumbnail. Existing `ImagingGenerator` handles image resize. |

**Key insight:** The biggest trap is trying to build search or document rendering from scratch. PostgreSQL tsvector and external tools (poppler, docconv, PDF.js, OnlyOffice) solve these problems vastly better than any hand-rolled solution.

## Common Pitfalls

### Pitfall 1: MinIO Key Collision Between Modules
**What goes wrong:** Chat files use `channels/{channel_id}/files/{file_id}/{filename}`. If documents reuse the same bucket without distinct prefixes, keys could conceptually overlap and cleanup becomes dangerous.
**Why it happens:** Single bucket `kmuhub-files` used by all modules.
**How to avoid:** Use distinct prefixes: `documents/{space_type}/{space_id}/{folder_id}/{file_id}/{filename}` for document files. Virtual folders reference existing `channels/` and `emails/` prefixed keys. Never delete a MinIO object that might be referenced by another module.
**Warning signs:** Files disappearing from chat after being "deleted" from the document browser.

### Pitfall 2: Synchronous Text Extraction Blocking Uploads
**What goes wrong:** Extracting text from a 50MB PDF during the upload request causes timeouts and poor UX.
**Why it happens:** Text extraction is CPU-intensive and can take 5-30 seconds for large documents.
**How to avoid:** Upload the file to MinIO and create the DB record immediately, then extract text asynchronously. Update `search_vector` when extraction completes. Show a "Indexing..." status on the file until done.
**Warning signs:** Upload endpoint latency > 5 seconds, gateway timeouts on large files.

### Pitfall 3: WOPI Token Leaking Application Auth
**What goes wrong:** Reusing the main application JWT as the WOPI access token exposes all user permissions to OnlyOffice.
**Why it happens:** Developer shortcut to avoid creating a separate token system.
**How to avoid:** Generate purpose-specific WOPI tokens with minimal claims (`file_id`, `user_id`, `can_write`). WOPI tokens have their own secret and TTL (10 hours). OnlyOffice never sees the application JWT.
**Warning signs:** OnlyOffice WOPI requests contain application-level auth tokens.

### Pitfall 4: Missing Lock Cleanup on Crash
**What goes wrong:** If the gateway crashes while a file is locked, the lock persists forever and no one can edit the file.
**Why it happens:** WOPI Unlock is only sent by the client. If the client disconnects, no Unlock arrives.
**How to avoid:** WOPI locks have a 30-minute automatic expiry (per spec). Implement a periodic cleanup goroutine that removes expired locks. RefreshLock calls extend the expiry. No manual intervention needed.
**Warning signs:** Files stuck in "locked by another user" state after server restarts.

### Pitfall 5: Global Search N+1 Queries
**What goes wrong:** The global search endpoint makes 5+ serial gRPC calls, each with its own database query, leading to high latency.
**Why it happens:** Calling each module's search service sequentially instead of in parallel.
**How to avoid:** Use goroutines with `sync.WaitGroup` to fan out all search queries in parallel. Set a per-module timeout (500ms) so one slow module doesn't block the response.
**Warning signs:** Global search latency > 1 second.

### Pitfall 6: OnlyOffice Document Server Discovery Caching
**What goes wrong:** Every document open triggers a fresh discovery XML fetch from OnlyOffice, adding 200-500ms latency.
**Why it happens:** Not caching the discovery XML response.
**How to avoid:** Cache the parsed discovery XML for 1 hour. The WOPI discovery rarely changes (only on OnlyOffice upgrade). Refresh cache on 404 or error.
**Warning signs:** Document open latency consistently > 1 second.

### Pitfall 7: docconv OS Dependency Missing in Docker
**What goes wrong:** Text extraction returns empty strings or errors in Docker because `poppler-utils`, `wv`, `unrtf` are not installed.
**Why it happens:** docconv shells out to OS tools. Docker images often use Alpine/minimal base images.
**How to avoid:** Add OS dependencies to Dockerfile. Test text extraction in CI. Consider a dedicated "document processor" microservice if dependencies conflict with other services.
**Warning signs:** Search never finds content inside PDF/DOCX files, only filenames.

## Code Examples

### Database Migration: Document Files and Folders

```sql
-- Folder spaces: personal, team, project
CREATE TYPE folder_space_type AS ENUM ('personal', 'team', 'project');

CREATE TABLE document_folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    parent_id UUID REFERENCES document_folders(id) ON DELETE CASCADE,
    space_type folder_space_type NOT NULL,
    space_id UUID NOT NULL,  -- user_id for personal, team_id for team, project_id for project
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    icon VARCHAR(50) NOT NULL DEFAULT 'folder',
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_document_folders_parent ON document_folders(parent_id);
CREATE INDEX idx_document_folders_space ON document_folders(space_type, space_id);

-- Document files
CREATE TABLE document_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    folder_id UUID NOT NULL REFERENCES document_folders(id),
    filename VARCHAR(255) NOT NULL,
    mime_type VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL CHECK (file_size > 0),
    storage_key VARCHAR(512) NOT NULL,
    thumbnail_key VARCHAR(512),
    current_version INTEGER NOT NULL DEFAULT 1,
    owner_id UUID NOT NULL REFERENCES users(id),
    is_favorite BOOLEAN NOT NULL DEFAULT FALSE,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    content_text TEXT,  -- extracted text for search
    search_vector TSVECTOR,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_document_files_folder ON document_files(folder_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_document_files_owner ON document_files(owner_id);
CREATE INDEX idx_document_files_search ON document_files USING GIN(search_vector);

-- Auto-update search vector
CREATE OR REPLACE FUNCTION document_files_search_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector := to_tsvector('german',
        coalesce(NEW.filename, '') || ' ' ||
        coalesce(NEW.content_text, '')
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_document_files_search
    BEFORE INSERT OR UPDATE ON document_files
    FOR EACH ROW EXECUTE FUNCTION document_files_search_update();

-- File versions
CREATE TABLE document_file_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES document_files(id) ON DELETE CASCADE,
    version_number INTEGER NOT NULL,
    version_label VARCHAR(255),  -- optional user-defined label e.g. "Final Draft"
    storage_key VARCHAR(512) NOT NULL,
    file_size BIGINT NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(file_id, version_number)
);

CREATE INDEX idx_document_versions_file ON document_file_versions(file_id);

-- File shares
CREATE TABLE document_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(20) NOT NULL CHECK (entity_type IN ('file', 'folder')),
    entity_id UUID NOT NULL,
    shared_with_user_id UUID NOT NULL REFERENCES users(id),
    permission VARCHAR(20) NOT NULL CHECK (permission IN ('read', 'write')),
    shared_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(entity_type, entity_id, shared_with_user_id)
);

CREATE INDEX idx_document_shares_entity ON document_shares(entity_type, entity_id);
CREATE INDEX idx_document_shares_user ON document_shares(shared_with_user_id);

-- File tags
CREATE TABLE document_tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    color VARCHAR(7),  -- hex color
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(name)
);

CREATE TABLE document_file_tags (
    file_id UUID NOT NULL REFERENCES document_files(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES document_tags(id) ON DELETE CASCADE,
    PRIMARY KEY (file_id, tag_id)
);

-- Entity links (CRM, PM cross-referencing)
CREATE TABLE document_entity_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES document_files(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL,  -- 'contact', 'company', 'deal', 'project', 'task'
    entity_id UUID NOT NULL,
    linked_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(file_id, entity_type, entity_id)
);

CREATE INDEX idx_document_entity_links_file ON document_entity_links(file_id);
CREATE INDEX idx_document_entity_links_entity ON document_entity_links(entity_type, entity_id);

-- WOPI locks
CREATE TABLE wopi_locks (
    file_id UUID PRIMARY KEY REFERENCES document_files(id) ON DELETE CASCADE,
    lock_id VARCHAR(1024) NOT NULL,
    locked_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '30 minutes'
);
```

### Text Extraction Service

```go
// internal/document/search/extractor.go
package search

import (
    "context"
    "io"
    "strings"

    "code.sajari.com/docconv/v2"
)

// Extractor extracts text content from files for search indexing
type Extractor struct {
    maxBytes int64 // max bytes to index
}

func NewExtractor(maxIndexBytes int64) *Extractor {
    return &Extractor{maxBytes: maxIndexBytes}
}

// extractable MIME types
var extractableMimes = map[string]bool{
    "application/pdf":    true,
    "application/msword": true,
    "application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
    "application/vnd.ms-excel": true,
    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         true,
    "application/vnd.ms-powerpoint":                                             true,
    "application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
    "text/plain":    true,
    "text/csv":      true,
    "text/markdown": true,
    "text/html":     true,
    "application/rtf": true,
}

func (e *Extractor) CanExtract(mimeType string) bool {
    return extractableMimes[strings.ToLower(mimeType)]
}

func (e *Extractor) Extract(ctx context.Context, reader io.Reader, mimeType string) (string, error) {
    res, err := docconv.Convert(reader, mimeType, false)
    if err != nil {
        return "", err
    }

    text := res.Body
    // Truncate to max indexable size
    if int64(len(text)) > e.maxBytes {
        text = text[:e.maxBytes]
    }
    return text, nil
}
```

### OnlyOffice Editor Component (Frontend)

```tsx
// desktop/src/renderer/src/modules/dokumente/OnlyOfficeEditor.tsx
import { useEffect, useRef, useState } from 'react'

interface OnlyOfficeEditorProps {
  fileId: string
  fileName: string
  wopiToken: string
  wopiTokenTTL: number
  onClose: () => void
}

export function OnlyOfficeEditor({
  fileId, fileName, wopiToken, wopiTokenTTL, onClose,
}: OnlyOfficeEditorProps) {
  const iframeRef = useRef<HTMLIFrameElement>(null)
  const [loading, setLoading] = useState(true)

  // Construct the WOPI iframe URL
  // Discovery provides the urlsrc base, we append WOPISrc + access_token
  const ONLYOFFICE_URL = import.meta.env.VITE_ONLYOFFICE_URL || 'http://localhost:8088'
  const GATEWAY_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

  const wopiSrc = encodeURIComponent(`${GATEWAY_URL}/wopi/files/${fileId}`)
  const editorUrl = `${ONLYOFFICE_URL}/hosting/wopi/word/edit?` +
    `WOPISrc=${wopiSrc}&access_token=${wopiToken}&access_token_ttl=${wopiTokenTTL}`

  useEffect(() => {
    // Listen for PostMessage from OnlyOffice
    function handleMessage(event: MessageEvent) {
      if (event.origin !== ONLYOFFICE_URL) return
      const msg = JSON.parse(event.data)
      if (msg.MessageId === 'UI_Close') {
        onClose()
      }
    }
    window.addEventListener('message', handleMessage)
    return () => window.removeEventListener('message', handleMessage)
  }, [onClose, ONLYOFFICE_URL])

  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-background">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-border px-4 py-2">
        <span className="text-sm font-medium text-foreground">{fileName}</span>
        <button
          onClick={onClose}
          className="rounded-md px-3 py-1 text-sm text-muted-foreground hover:bg-secondary"
        >
          Schliessen
        </button>
      </div>

      {/* Editor iframe */}
      {loading && (
        <div className="flex-1 flex items-center justify-center">
          <p className="text-muted-foreground">Editor wird geladen...</p>
        </div>
      )}
      <iframe
        ref={iframeRef}
        src={editorUrl}
        className={`flex-1 border-0 ${loading ? 'hidden' : ''}`}
        onLoad={() => setLoading(false)}
        allow="clipboard-read; clipboard-write"
        sandbox="allow-scripts allow-same-origin allow-forms allow-popups allow-popups-to-escape-sandbox"
      />
    </div>
  )
}
```

### Thumbnail Generation for PDF (Claude's Discretion)

**Recommendation:** Use `poppler-utils` `pdftoppm` CLI to render the first page of a PDF as a JPEG, then resize with the existing `ImagingGenerator`. For non-PDF documents (DOCX, XLSX), generate thumbnails only for files that have been opened in OnlyOffice (OnlyOffice generates preview images). For images, use the existing thumbnail pipeline.

```go
// internal/document/file/pdf_thumbnail.go
func (g *PDFThumbnailGenerator) Generate(ctx context.Context, reader io.Reader, mimeType string) (io.Reader, error) {
    if mimeType != "application/pdf" {
        return nil, fmt.Errorf("unsupported mime type for PDF thumbnail: %s", mimeType)
    }
    // Write PDF to temp file
    tmpFile, err := os.CreateTemp("", "pdf-thumb-*.pdf")
    if err != nil {
        return nil, err
    }
    defer os.Remove(tmpFile.Name())
    if _, err := io.Copy(tmpFile, reader); err != nil {
        return nil, err
    }
    tmpFile.Close()

    // Render first page to JPEG using pdftoppm
    outPrefix := tmpFile.Name() + "-thumb"
    cmd := exec.CommandContext(ctx, "pdftoppm",
        "-jpeg", "-f", "1", "-l", "1", "-r", "150",
        tmpFile.Name(), outPrefix)
    if err := cmd.Run(); err != nil {
        return nil, fmt.Errorf("pdftoppm failed: %w", err)
    }

    thumbPath := outPrefix + "-1.jpg"
    defer os.Remove(thumbPath)
    thumbData, err := os.ReadFile(thumbPath)
    if err != nil {
        return nil, err
    }
    return bytes.NewReader(thumbData), nil
}
```

### Chunked Upload with Progress (Claude's Discretion)

**Recommendation:** For files under 50MB (current limit), use single-request multipart upload with `XMLHttpRequest.upload.onprogress` for progress tracking. This avoids chunked upload complexity while providing progress indication. If the file size limit increases beyond 100MB in the future, implement chunked upload using MinIO's multipart upload API.

```typescript
// desktop/src/renderer/src/api/hooks/useDocumentUpload.ts
export function useDocumentUpload() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({
      folderId,
      file,
      onProgress,
    }: {
      folderId: string
      file: File
      onProgress?: (percent: number) => void
    }) => {
      return new Promise((resolve, reject) => {
        const formData = new FormData()
        formData.append('file', file)
        formData.append('folder_id', folderId)

        const xhr = new XMLHttpRequest()
        xhr.upload.onprogress = (e) => {
          if (e.lengthComputable) {
            onProgress?.(Math.round((e.loaded / e.total) * 100))
          }
        }
        xhr.onload = () => {
          if (xhr.status >= 200 && xhr.status < 300) {
            resolve(JSON.parse(xhr.responseText))
          } else {
            reject(new Error(`Upload failed: ${xhr.status}`))
          }
        }
        xhr.onerror = () => reject(new Error('Upload failed'))

        const token = useAuthStore.getState().accessToken
        xhr.open('POST', `${API_BASE_URL}/api/v1/documents/files/upload`)
        if (token) xhr.setRequestHeader('Authorization', `Bearer ${token}`)
        xhr.send(formData)
      })
    },
  })
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| OnlyOffice JS SDK only | WOPI protocol support | OnlyOffice 6.4 (2021) | Standard interop protocol, can switch editors |
| WOPI without `UserCanOnlyComment` | Comment-only mode via WOPI | OnlyOffice 9.1.0 (2024) | Fine-grained access control in editor |
| Manual discovery parsing | Cached discovery with auto-refresh | Ongoing | Performance optimization |
| Elasticsearch for SMB search | PostgreSQL tsvector + GIN | Proven pattern | One fewer service, ACID consistency |
| Custom lock table without TTL | WOPI spec mandates 30-min expiry + RefreshLock | WOPI spec | Automatic lock cleanup |

**Deprecated/outdated:**
- OnlyOffice JS API `convertapi` approach: WOPI is now the recommended integration method.
- Bleve for small-scale search: PostgreSQL tsvector is sufficient for < 1M documents and avoids separate index management.

## Open Questions

1. **docconv OS dependency compatibility with existing Docker images**
   - What we know: Current Dockerfiles use Go binary builds. docconv needs poppler-utils, wv, unrtf, tidy.
   - What's unclear: Whether adding these to the document service Dockerfile creates conflicts or bloats the image excessively.
   - Recommendation: Create a dedicated `Dockerfile.document` with the additional OS packages. Test text extraction in CI with sample files.

2. **OnlyOffice licensing for self-hosted customers**
   - What we know: OnlyOffice Community Edition is AGPL v3 (free). Enterprise Edition adds mobile editing, content controls, and some features.
   - What's unclear: Whether SMB customers will need features only in Enterprise Edition.
   - Recommendation: Build WOPI integration against Community Edition. The WOPI protocol is the same regardless of edition. Enterprise can be swapped in later.

3. **Search ranking normalization across modules**
   - What we know: Each module uses `ts_rank` independently. CRM scores are not comparable to file scores in absolute terms.
   - What's unclear: How to meaningfully rank a CRM contact result against a file content match.
   - Recommendation: Use `ts_rank_cd` with normalization option 32 (divide by rank + 1) for all modules. Apply module-specific boosts (e.g., exact filename match gets 2x boost). Accept that cross-module ranking will be approximate -- grouping by module (as decided) reduces the impact of imperfect cross-module ranking.

4. **Virtual folder performance with large attachment volumes**
   - What we know: Virtual folders query `chat_files`, `email_attachments`, `task_files` with JOINs.
   - What's unclear: Performance impact when a user has access to 10K+ chat files across many channels.
   - Recommendation: Add pagination to virtual folder queries. Consider a materialized view or summary cache if performance degrades. Monitor query latency.

## Sources

### Primary (HIGH confidence)
- Existing codebase: `internal/chat/file/` (MinIO integration, FileStore interface, thumbnail, scanner)
- Existing codebase: `internal/crm/search/` and `internal/chat/search/` (PostgreSQL tsvector pattern)
- Existing codebase: `internal/models/search.go` (SearchResult, SearchEntityType)
- Existing codebase: `desktop/src/renderer/src/modules/dokumente/` (frontend components, Zustand store)
- Existing codebase: `backend/migrations/000018_create_chat_files.up.sql`, `000041_create_email_tables.up.sql`, `000026_create_task_collaboration.up.sql` (existing file tables)
- Existing codebase: `backend/internal/gateway/` (RouteRegistrar pattern, ServiceRegistry)
- OnlyOffice WOPI REST API: https://api.onlyoffice.com/docs/docs-api/using-wopi/wopi-rest-api/
- OnlyOffice WOPI Overview: https://api.onlyoffice.com/docs/docs-api/using-wopi/overview/
- OnlyOffice WOPI Key Concepts: https://api.onlyoffice.com/docs/docs-api/using-wopi/key-concepts/
- OnlyOffice WOPI Discovery: https://api.onlyoffice.com/docs/docs-api/using-wopi/wopi-discovery/
- OnlyOffice WOPI Config: https://api.onlyoffice.com/docs/docs-api/using-wopi/config/
- OnlyOffice CheckFileInfo: https://api.onlyoffice.com/docs/docs-api/using-wopi/wopi-rest-api/checkfileinfo/
- OnlyOffice Docker-DocumentServer: https://github.com/ONLYOFFICE/Docker-DocumentServer

### Secondary (MEDIUM confidence)
- DeepWiki WOPI Protocol: https://deepwiki.com/ONLYOFFICE/DocumentServer/7.4-wopi-protocol (comprehensive flow details)
- Microsoft WOPI Spec: https://learn.microsoft.com/en-us/microsoft-365/cloud-storage-partner-program/online/ (canonical protocol reference)
- DZone WOPI Implementation Guide: https://dzone.com/articles/implementing-wopi-protocol-for-office-integration
- sajari/docconv GitHub: https://github.com/sajari/docconv (text extraction library)
- Bleve search: https://github.com/blevesearch/bleve (alternative considered, not recommended)
- PostgreSQL Full-Text Search: https://www.postgresql.org/docs/current/textsearch-intro.html

### Tertiary (LOW confidence)
- Pydio Cells WOPI Go implementation: https://pkg.go.dev/github.com/pydio/cells/gateway/wopi (reference for Go WOPI patterns, not directly used)
- CS3ORG Reva WOPI: https://pkg.go.dev/github.com/cs3org/reva/pkg/app/provider/wopi (another Go WOPI reference)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All core libraries already in use or well-documented open source
- Architecture: HIGH - Follows established codebase patterns (service/repository/handler, RouteRegistrar, FileStore interface)
- WOPI protocol: HIGH - Official OnlyOffice documentation is comprehensive, DeepWiki provides implementation details
- Full-text search: HIGH - PostgreSQL tsvector pattern proven in 3 existing modules
- Text extraction (docconv): MEDIUM - Library is maintained but requires OS dependencies; need to validate in Docker
- Pitfalls: HIGH - Based on codebase analysis and common WOPI integration issues from multiple sources

**Research date:** 2026-02-17
**Valid until:** 2026-03-17 (30 days - stable domain, libraries well-established)

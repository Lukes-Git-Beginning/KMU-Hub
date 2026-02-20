# Phase 10: Email Integration - Research

**Researched:** 2026-02-16
**Domain:** IMAP/SMTP email integration, CRM email linking, contact import/export
**Confidence:** HIGH

## Summary

Phase 10 adds a full email client within KMU Hub: IMAP sync, SMTP sending, threaded conversations, CRM auto-linking, HTML signatures, and contact import/export with shared/personal visibility. The email service follows the established microservice pattern (new `email` gRPC service at port `:50056`) with domain packages under `internal/email/`. Frontend stubs already exist (MailsPage.tsx, mails.ts Zustand store, MailSettingsTab.tsx, ComposeInline.tsx) from the design integration -- these need to be rewired from mock data to TanStack Query hooks backed by the real API.

The Go ecosystem provides a complete, cohesive library set from emersion (go-imap v2, go-smtp, go-message, go-vcard) that covers IMAP sync, SMTP sending, MIME parsing, and vCard import/export. For the frontend rich text editor, TipTap v3 is the recommended choice -- it has the best balance of feature completeness (tables, images, links, font formatting), developer experience, and React integration. Email threading uses the JWZ algorithm (RFC 5256 REFERENCES), with a Go implementation available at gatherstars-com/jwz. Contact import/export combines go-vcard (RFC 6350) with Go's standard encoding/csv package. Attachments reuse the existing MinIO infrastructure already established for chat file sharing.

**Primary recommendation:** Build a standalone `email` microservice following the exact same patterns as the `crm` and `work` services. Use emersion's library suite for all email protocol handling. Use TipTap v3 for the compose editor. Reuse existing MinIO file store for attachments. The IMAP sync engine is the most architecturally complex part -- use IDLE with polling fallback, local DB cache with UIDVALIDITY tracking, and per-user background sync goroutines.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Inbox experience:**
- Three-column layout: folder sidebar + message list + reading pane side-by-side (Outlook-style)
- Threaded conversations: related emails collapsed into one row showing latest message, expand to see full thread (Gmail-style)
- User-switchable density: both comfortable (sender, subject, 1-line preview, date, attachment icon) and compact (sender + subject + date on one row) modes, togglable in settings or via UI toggle
- Smart folder defaults: pin Inbox/Sent/Drafts/Trash at top, remaining IMAP folders in collapsible section below

**Compose & signatures:**
- Full rich text editor: bold, italic, lists, links, inline images, tables, font size/color (Outlook-level capability)
- Visual signature builder: WYSIWYG editor with structured fields (name, title, phone, logo, Impressum block) for non-technical users
- Local-first drafts: draft stored locally while composing, saved to IMAP Drafts folder only when compose window is closed
- Compose mode: default inline (opens in reading pane area), with pop-out button to expand to full modal overlay

**CRM auto-linking:**
- Automatic + indicator: emails matching a CRM contact's email address are auto-linked silently, with a subtle CRM badge shown on linked emails
- Unknown senders: show a subtle "Add to CRM?" prompt on emails from addresses not matching any contact
- Email history on contact profiles: emails appear in the existing activity timeline alongside calls, notes, and deal changes (no separate tab)
- Deal linking: auto-link to contacts by email match, and if that contact has open deals, show a "Link to deal?" option for manual deal association

**Contact import/export:**
- Step-by-step import wizard: upload -> preview data -> map fields -> handle duplicates -> confirm -> import (guided multi-step flow)
- Duplicate handling: auto-merge by email match -- if email matches existing contact, auto-merge new fields into existing record (no duplicates created)
- Visibility model: import wizard asks shared vs personal for the batch; manually created contacts default to shared; users can mark specific contacts as personal (only they see them); admin can override visibility
- Export: user picks which fields to export (name, email, phone, company, etc.) and chooses CSV or vCard format

### Claude's Discretion
- IMAP sync frequency and strategy (polling interval, IDLE push support, initial sync depth)
- Rich text editor library choice (TipTap, Slate, etc.)
- Thread reconstruction algorithm details (References/In-Reply-To header parsing)
- Attachment storage strategy in MinIO
- Email search implementation approach
- Read/unread bidirectional sync mechanism

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

## Standard Stack

### Core (Backend - Go)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| emersion/go-imap/v2 | v2.0.0-beta.8 | IMAP client (sync, IDLE, fetch, store flags) | Only maintained IMAP4rev2 Go library; IDLE built-in; same author as go-smtp/go-message/go-vcard |
| emersion/go-smtp | v0.24.0 | SMTP client (send email via STARTTLS) | Standard Go SMTP client; supports AUTH, STARTTLS, PIPELINING, DSN |
| emersion/go-message | latest | MIME parsing/creation (multipart, attachments, HTML) | Companion to go-imap; RFC 2045/2046/2047 compliant; handles charset/encoding |
| emersion/go-vcard | latest | vCard parsing and generation (contact import/export) | RFC 6350 compliant; same ecosystem as go-imap |
| gatherstars-com/jwz | v1.4.0 | JWZ email threading algorithm | Go implementation of proven algorithm; tested against thousands of emails |
| gocarina/gocsv | latest | CSV parsing/generation (contact import/export) | Struct-tag based CSV serialization; cleaner than encoding/csv for field mapping |
| minio/minio-go/v7 | v7.0.98 (already in go.mod) | Attachment storage in MinIO | Already used for chat file storage; reuse existing MinIO infrastructure |

### Core (Frontend - React/TypeScript)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| @tiptap/react | v3.19.0 | Rich text email compose editor | Best DX for React; 100+ extensions; tables, images, links, font formatting; headless (style-agnostic, works with Tailwind) |
| @tiptap/pm | v3.19.0 | ProseMirror peer dependency | Required by TipTap |
| @tiptap/starter-kit | v3.19.0 | Basic extensions bundle | Bold, italic, lists, code, headings, etc. |
| @tiptap/extension-table | v3.x | Table support in compose | Tables extension for email formatting |
| @tiptap/extension-image | v3.x | Inline images in compose | Image insertion/resizing |
| @tiptap/extension-link | v3.x | Link insertion/editing | Hyperlink support |
| @tiptap/extension-color | v3.x | Text color | Font color for email formatting |
| @tiptap/extension-text-style | v3.x | Font size/family | Text styling for Outlook-level formatting |
| @tiptap/extension-underline | v3.x | Underline support | Not in starter-kit, needed for email |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| encoding/csv (stdlib) | Go stdlib | CSV parsing fallback | If gocsv is overkill for simple CSV generation |
| net/mail (stdlib) | Go stdlib | Email address parsing/validation | Already used in contact service; reuse for email address validation |
| golang.org/x/text | already in go.mod | Character encoding normalization | MIME charset handling (UTF-8 normalization) |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| TipTap v3 | Lexical (Meta) | More performant at massive scale but no v1.0 yet, steeper learning curve, less mature extension ecosystem |
| TipTap v3 | Slate | Maximum flexibility but more boilerplate, less batteries-included |
| go-imap v2 beta | go-imap v1.2.1 (stable) | v1 is stable but IMAP4rev1 only, no built-in IDLE, requires separate go-imap-idle extension; v2 has IDLE built-in and modern API |
| gatherstars-com/jwz | Custom threading | JWZ algorithm is complex (6 steps); hand-rolling is error-prone; this library is battle-tested |
| gocsv | encoding/csv (stdlib) | stdlib is lower-level; gocsv provides struct tag marshaling which maps cleanly to CRM contact fields |

### Installation

**Backend:**
```bash
cd backend
go get github.com/emersion/go-imap/v2@v2.0.0-beta.8
go get github.com/emersion/go-smtp@v0.24.0
go get github.com/emersion/go-message@latest
go get github.com/emersion/go-vcard@latest
go get github.com/gatherstars-com/jwz@v1.4.0
go get github.com/gocarina/gocsv@latest
```

**Frontend:**
```bash
cd desktop
npm install @tiptap/react @tiptap/pm @tiptap/starter-kit @tiptap/extension-table @tiptap/extension-image @tiptap/extension-link @tiptap/extension-color @tiptap/extension-text-style @tiptap/extension-underline
```

## Architecture Patterns

### Recommended Project Structure

```
backend/
  cmd/
    email/
      main.go                    # Email service entry point (new binary)
  internal/
    email/
      account/                   # IMAP/SMTP account configuration per user
        errors.go
        repository.go
        postgres_repository.go
        service.go
        service_test.go
      sync/                      # IMAP sync engine (IDLE + polling)
        engine.go                # Per-user sync goroutine management
        imap_client.go           # go-imap v2 wrapper
        errors.go
      message/                   # Email message CRUD + thread assembly
        repository.go
        postgres_repository.go
        service.go
        service_test.go
        thread.go                # JWZ threading integration
      send/                      # SMTP sending + draft management
        service.go
        service_test.go
      signature/                 # HTML signature builder + storage
        repository.go
        postgres_repository.go
        service.go
      attachment/                # Email attachment storage (MinIO)
        service.go
        store.go                 # Reuse chat/file MinIOStore pattern
      contact/                   # Contact import/export + visibility
        import_service.go        # CSV/vCard import with field mapping
        export_service.go        # CSV/vCard export with field selection
        visibility.go            # Shared/personal visibility logic
  proto/
    email/
      v1/
        email.proto              # EmailService gRPC definition

desktop/
  src/renderer/src/
    api/
      email-client.ts            # Fetch wrapper (same pattern as calendar-client.ts)
      hooks/
        email.ts                 # TanStack Query hooks for email operations
        contacts-import.ts       # Import/export hooks
    modules/
      mails/
        MailsPage.tsx            # REWIRE from Zustand mock to TanStack Query
        ComposeInline.tsx        # REWIRE + add TipTap editor
        ComposeModal.tsx         # REWIRE
        ThreadView.tsx           # NEW: thread expansion component
        SignatureBuilder.tsx     # NEW: WYSIWYG signature editor
        ImportWizard.tsx         # NEW: step-by-step contact import
        ExportDialog.tsx         # NEW: field selection + format picker
    stores/
      mails.ts                   # REPLACE: ephemeral UI state only (no mock data)
```

### Pattern 1: Email Service as Standalone Microservice

**What:** New `email` service binary at gRPC port `:50056`, same pattern as `crm` and `work`.
**When to use:** Always for this phase. The email sync engine maintains long-lived IMAP connections per user -- this workload profile justifies a separate process.

```go
// backend/cmd/email/main.go (follows cmd/crm/main.go pattern exactly)
func main() {
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    slog.SetDefault(logger)

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    cfg, err := config.Load(ctx)
    // ... same init pattern as crm/work ...

    // Initialize email-specific services
    accountRepo := account.NewPostgresRepository(pool)
    accountService := account.NewService(accountRepo, vaultEncryptor)
    messageRepo := message.NewPostgresRepository(pool)
    messageService := message.NewService(messageRepo)
    // ... etc ...

    // Sync engine manages per-user IMAP connections
    syncEngine := sync.NewEngine(accountService, messageService, attachmentService)
    go syncEngine.Start(ctx) // Background goroutine

    // gRPC server
    grpcServer := grpc.NewServer(/* interceptors */)
    emailGRPC := server.NewEmailGRPCServer(accountService, messageService, sendService, signatureService, importService, exportService)
    emailv1.RegisterEmailServiceServer(grpcServer, emailGRPC)
    // ... health, metrics, shutdown (same pattern) ...
}
```

### Pattern 2: IMAP Sync Engine with IDLE + Polling Fallback

**What:** A per-user background sync mechanism that maintains IMAP connections and syncs email to local PostgreSQL cache.
**When to use:** Core of the email reading experience. Without this, every email view would require a direct IMAP fetch.

**Sync strategy (Claude's discretion recommendation):**
- **Initial sync:** Fetch last 30 days of headers + body snippets on first account setup. Full bodies fetched on-demand when user opens an email.
- **IDLE (real-time):** Use go-imap v2's built-in `Idle()` on INBOX. When server notifies of new messages, fetch new message headers immediately.
- **Polling fallback:** If IMAP server doesn't support IDLE, poll every 60 seconds via `Noop()`.
- **Non-INBOX folders:** Poll every 5 minutes (IDLE only supported on one folder at a time).
- **UIDVALIDITY tracking:** Store UIDVALIDITY per folder. If it changes, invalidate local cache and re-sync from scratch.
- **Delta sync:** Use highest known UID to fetch only new messages (`UID FETCH <last_uid+1>:*`).

```go
// Sync engine pseudocode
type Engine struct {
    accounts  *account.Service
    messages  *message.Service
    workers   map[uuid.UUID]*userWorker // userID -> worker
    mu        sync.Mutex
}

func (e *Engine) Start(ctx context.Context) {
    // Load all active email accounts, start a worker per user
    accounts, _ := e.accounts.ListActive(ctx)
    for _, acc := range accounts {
        e.startWorker(ctx, acc)
    }
}

type userWorker struct {
    client  *imapclient.Client
    account *models.EmailAccount
}

func (w *userWorker) run(ctx context.Context) {
    // 1. Connect + Login
    // 2. Check UIDVALIDITY for each folder
    // 3. Delta sync new messages
    // 4. Enter IDLE loop on INBOX (or polling fallback)
    // 5. Periodically sync other folders
}
```

### Pattern 3: IMAP Credentials Encryption via Vault

**What:** IMAP/SMTP passwords stored encrypted at rest using the existing vault encryption infrastructure from Phase 9.
**When to use:** Always. Email passwords are the most sensitive user credential we store.

```go
// Account service encrypts IMAP/SMTP password before storage
type Service struct {
    repo      Repository
    encryptor auth.VaultEncryptor // Reuse Phase 9 vault encryption
}

func (s *Service) Create(ctx context.Context, input CreateAccountInput) (*models.EmailAccount, error) {
    // Encrypt password before storing
    encrypted, err := s.encryptor.Encrypt(ctx, []byte(input.Password))
    if err != nil {
        return nil, fmt.Errorf("encrypt password: %w", err)
    }
    // Store encrypted password in DB
    acc := &models.EmailAccount{
        IMAPHost:          input.IMAPHost,
        IMAPPort:          input.IMAPPort,
        PasswordEncrypted: encrypted,
        // ...
    }
    return acc, s.repo.Create(ctx, acc)
}
```

### Pattern 4: CRM Auto-Linking via Email Address Match

**What:** When emails are synced, the sender/recipient addresses are matched against CRM contact email fields. Matches create email-contact links in a junction table.
**When to use:** On every email sync (new message ingestion).

```go
// During sync, after storing a new email message:
func (s *MessageService) LinkToCRM(ctx context.Context, msgID uuid.UUID, addresses []string) error {
    for _, addr := range addresses {
        contact, err := s.crmRepo.GetByEmail(ctx, addr)
        if err != nil || contact == nil {
            continue // No CRM match, skip (frontend shows "Add to CRM?" prompt)
        }
        // Create link: email_id <-> contact_id
        s.repo.CreateEmailContactLink(ctx, msgID, contact.ID)
    }
    return nil
}
```

### Pattern 5: Thread Reconstruction

**What:** Group related emails into conversation threads using Message-ID, References, and In-Reply-To headers.
**When to use:** When displaying email list and when syncing new messages.

**Thread reconstruction approach (Claude's discretion recommendation):**
- Store `message_id`, `in_reply_to`, and `references` (text array) columns in the emails table
- Use the gatherstars-com/jwz library to build thread trees from stored headers
- Thread ID assigned at sync time: compute a `thread_id` (UUID) for each conversation group
- New emails joining existing threads get the same `thread_id`
- Thread root = earliest message in the conversation

```go
// Thread assignment during sync
func (s *MessageService) AssignThread(ctx context.Context, msg *models.EmailMessage) error {
    // 1. Check if any message in references/in_reply_to already has a thread_id
    threadID, err := s.repo.FindThreadByReferences(ctx, msg.References, msg.InReplyTo)
    if err != nil {
        return err
    }
    if threadID == uuid.Nil {
        // New thread: generate fresh thread_id
        threadID = uuid.New()
    }
    msg.ThreadID = threadID
    return s.repo.UpdateThreadID(ctx, msg.ID, threadID)
}
```

### Pattern 6: Read/Unread Bidirectional Sync

**What:** When user marks email as read/unread in Hub, push flag change to IMAP server. When IMAP server reports flag changes (via IDLE/sync), update local state.
**When to use:** Every read/unread state change.

**Mechanism (Claude's discretion recommendation):**
- **Hub -> IMAP:** When user opens email (marks read), call `client.Store(seqSet, &imap.StoreFlags{Op: imap.StoreFlagsAdd, Flags: []imap.Flag{imap.FlagSeen}}, nil)` via gRPC to sync engine
- **IMAP -> Hub:** During sync/IDLE, detect flag changes and update local DB
- **Conflict resolution:** IMAP server is source of truth. If conflict, IMAP flag wins.

### Anti-Patterns to Avoid

- **Direct IMAP fetch on every UI action:** Never fetch directly from IMAP for list/read operations. Always serve from local PostgreSQL cache. Only the sync engine talks to IMAP.
- **Storing email passwords in plaintext:** Always use vault encryption (Phase 9 infrastructure).
- **Single IMAP connection for all users:** Each user needs their own IMAP connection. Use a goroutine-per-user pattern with proper lifecycle management.
- **Blocking the gRPC handler during IMAP operations:** IMAP operations can be slow/network-bound. The sync engine runs asynchronously; gRPC handlers serve from local cache.
- **Syncing entire mailbox on first setup:** Sync headers for last 30 days initially. Full bodies on-demand. Prevents multi-hour initial sync.
- **Dual-write to IMAP and DB:** DB is the read cache, IMAP is the source of truth. Write to IMAP first, then update DB from IMAP state.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| IMAP protocol | Custom IMAP parser | emersion/go-imap v2 | IMAP4rev2 is enormously complex (RFC 9051, 100+ pages); go-imap handles all edge cases, extensions, encoding |
| SMTP sending | net/smtp (frozen stdlib) | emersion/go-smtp | stdlib net/smtp is frozen, no STARTTLS, no AUTH extensions, no PIPELINING |
| MIME parsing | Custom multipart parser | emersion/go-message | MIME is deceptively complex: nested multiparts, charset encoding, transfer encoding, RFC 2047 encoded words |
| Email threading | Custom threading code | gatherstars-com/jwz | JWZ algorithm is 6 complex steps with many edge cases; this library is battle-tested against thousands of emails |
| vCard parsing | Custom .vcf parser | emersion/go-vcard | vCard 4.0 (RFC 6350) has structured types, encoding rules, parameter handling; library handles all of it |
| Rich text editor | Custom contenteditable | TipTap v3 | Rich text editing is the hardest problem in web development; TipTap wraps ProseMirror which has years of battle-testing |
| Email HTML rendering | dangerouslySetInnerHTML | iframe sandbox or DOMPurify | Email HTML is untrusted input; must sanitize XSS, strip scripts, sandbox CSS |

**Key insight:** Email is one of the most complex protocols in computing. Every component (IMAP sync, MIME parsing, threading, HTML rendering, SMTP sending) has decades of edge cases encoded in RFCs. Using battle-tested libraries for all protocol-level operations is not optional -- it's the only viable approach.

## Common Pitfalls

### Pitfall 1: UIDVALIDITY Changes Invalidate Local Cache
**What goes wrong:** IMAP server resets UIDVALIDITY (e.g., after mailbox rebuild), making all locally cached UIDs invalid. If not handled, emails appear duplicated or missing.
**Why it happens:** UIDVALIDITY is the server's contract that UIDs are stable. When it changes, all cached UIDs are invalid.
**How to avoid:** Store UIDVALIDITY per folder per account. On every sync, compare with server's value. If different: wipe local cache for that folder and re-sync from scratch. Log a warning (this shouldn't happen frequently).
**Warning signs:** Duplicate emails appearing in a folder, or "phantom" emails that no longer exist on server.

### Pitfall 2: IMAP Connection Lifecycle Management
**What goes wrong:** IMAP connections are long-lived TCP connections. They can silently die (network change, server timeout, firewall drops). If not detected, the sync engine thinks it's connected but receives no updates.
**Why it happens:** TCP keepalive may not detect all connection failures. IMAP IDLE has a 30-minute recommended restart interval (RFC 2177).
**How to avoid:** Restart IDLE every 25 minutes (go-imap v2 does this automatically). Implement reconnection with exponential backoff. Monitor `client.Closed()` channel. Health check IMAP connections periodically.
**Warning signs:** New emails not appearing despite being visible in Outlook/Gmail.

### Pitfall 3: Email HTML Rendering Security (XSS)
**What goes wrong:** Emails contain arbitrary HTML. Rendering directly in the app allows XSS attacks, CSS injection, or tracking pixel execution.
**Why it happens:** Email HTML is untrusted content from external senders.
**How to avoid:** Render email HTML in a sandboxed iframe with `sandbox="allow-same-origin"` (no scripts). Alternatively, use DOMPurify to sanitize HTML before rendering. Strip `<script>`, `<style>` external refs, `onload` handlers, etc. Block external image loading by default (privacy).
**Warning signs:** JavaScript executing in email view, external resources loading without user consent.

### Pitfall 4: SMTP Credential Exposure
**What goes wrong:** SMTP passwords stored in plaintext in the database are exposed in a breach.
**Why it happens:** Developer shortcuts -- storing credentials as-is instead of encrypting.
**How to avoid:** Use the Phase 9 vault encryption service (`VaultEncryptor` interface) to encrypt IMAP/SMTP passwords at rest. Decrypt only when establishing a connection, never log decrypted values.
**Warning signs:** Plaintext password columns in the database, passwords appearing in logs.

### Pitfall 5: Blocking UI During IMAP Operations
**What goes wrong:** User clicks "Refresh" and the UI freezes for 5+ seconds while the IMAP fetch completes.
**Why it happens:** Direct IMAP fetch in the request path instead of serving from cache.
**How to avoid:** Architecture rule: UI always reads from local PostgreSQL cache. Sync engine writes to cache asynchronously. "Refresh" triggers a sync request (async), and the UI polls/subscribes for updates.
**Warning signs:** Slow email list loading, UI hangs when switching folders.

### Pitfall 6: Large Attachment Memory Pressure
**What goes wrong:** Syncing an email with a 25MB attachment loads the entire attachment into memory during parsing.
**Why it happens:** go-message's Reader can buffer large MIME parts.
**How to avoid:** Stream attachments directly to MinIO during sync. Use go-message's `Part.Body` as an `io.Reader` and pipe directly to `minio.PutObject`. Never buffer entire attachments in memory.
**Warning signs:** High memory usage during sync, OOM kills on large mailboxes.

### Pitfall 7: Contact Import Duplicate Explosion
**What goes wrong:** Importing a CSV with 500 contacts creates 500 new contacts even though 400 already exist.
**Why it happens:** No duplicate detection before import.
**How to avoid:** User decision specifies auto-merge by email match. During import: for each row, check if email matches existing contact. If yes, merge new fields into existing record (don't overwrite non-empty fields with empty). Show a preview of merges/new contacts before executing.
**Warning signs:** Contact count exploding after imports, duplicate entries with slight variations.

### Pitfall 8: Thread Reconstruction Missing Messages
**What goes wrong:** Emails appear as separate threads even though they're part of the same conversation.
**Why it happens:** Some email clients don't set References/In-Reply-To correctly. Outlook uses proprietary Thread-Index header.
**How to avoid:** Fall back to subject-based grouping as a secondary strategy: normalize subject (strip "Re:", "Fwd:", "AW:", "WG:" prefixes), group by normalized subject + overlapping participants within a time window. JWZ algorithm handles most cases; subject fallback catches the rest.
**Warning signs:** "Re: Meeting" appearing as a new thread separate from "Meeting".

## Code Examples

### IMAP Connect + Fetch Latest Messages (go-imap v2)

```go
// Source: https://pkg.go.dev/github.com/emersion/go-imap/v2/imapclient
import (
    "github.com/emersion/go-imap/v2"
    "github.com/emersion/go-imap/v2/imapclient"
)

func fetchLatestEmails(host string, port int, username, password string, limit int) error {
    addr := fmt.Sprintf("%s:%d", host, port)

    options := &imapclient.Options{
        UnilateralDataHandler: &imapclient.UnilateralDataHandler{
            Mailbox: func(data *imapclient.UnilateralDataMailbox) {
                if data.NumMessages != nil {
                    slog.Info("new message notification", "count", *data.NumMessages)
                }
            },
        },
    }

    c, err := imapclient.DialTLS(addr, options)
    if err != nil {
        return fmt.Errorf("dial TLS: %w", err)
    }
    defer c.Close()

    if err := c.Login(username, password).Wait(); err != nil {
        return fmt.Errorf("login: %w", err)
    }

    selectData, err := c.Select("INBOX", nil).Wait()
    if err != nil {
        return fmt.Errorf("select INBOX: %w", err)
    }

    if selectData.NumMessages == 0 {
        return nil // Empty mailbox
    }

    // Fetch last N messages
    from := uint32(1)
    if selectData.NumMessages > uint32(limit) {
        from = selectData.NumMessages - uint32(limit) + 1
    }
    seqSet := imap.SeqSetNum()
    seqSet.AddRange(from, selectData.NumMessages)

    fetchOptions := &imap.FetchOptions{
        Envelope: true,
        Flags:    true,
        UID:      true,
        BodySection: []*imap.FetchItemBodySection{
            {Specifier: imap.PartSpecifierHeader},
        },
    }

    fetchCmd := c.Fetch(seqSet, fetchOptions)
    defer fetchCmd.Close()

    for {
        msg := fetchCmd.Next()
        if msg == nil {
            break
        }
        // Process envelope, flags, UID, headers...
    }

    return fetchCmd.Close()
}
```

### IMAP IDLE with Fallback

```go
// Source: https://pkg.go.dev/github.com/emersion/go-imap/v2/imapclient
func idleWithFallback(c *imapclient.Client, ctx context.Context) error {
    if c.Caps().Has(imap.CapIdle) {
        return idleLoop(c, ctx)
    }
    return pollLoop(c, ctx)
}

func idleLoop(c *imapclient.Client, ctx context.Context) error {
    for {
        idleCmd, err := c.Idle()
        if err != nil {
            return err
        }

        done := make(chan error, 1)
        go func() { done <- idleCmd.Wait() }()

        // Restart IDLE every 25 minutes (RFC recommends 29 min max)
        timer := time.NewTimer(25 * time.Minute)
        select {
        case <-ctx.Done():
            timer.Stop()
            idleCmd.Close()
            return ctx.Err()
        case <-timer.C:
            idleCmd.Close()
            <-done
            // Re-enter IDLE
        case err := <-done:
            timer.Stop()
            if err != nil {
                return err
            }
            // Server sent an update, process it then re-IDLE
        }
    }
}

func pollLoop(c *imapclient.Client, ctx context.Context) error {
    ticker := time.NewTicker(60 * time.Second)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            if err := c.Noop().Wait(); err != nil {
                return err
            }
            // Check for new messages via mailbox state
        }
    }
}
```

### SMTP Send Email (go-smtp)

```go
// Source: https://pkg.go.dev/github.com/emersion/go-smtp
import (
    "github.com/emersion/go-smtp"
    "github.com/emersion/go-sasl"
)

func sendEmail(host string, port int, username, password, from string, to []string, body io.Reader) error {
    addr := fmt.Sprintf("%s:%d", host, port)

    auth := sasl.NewPlainClient("", username, password)

    // Use STARTTLS (port 587) or direct TLS (port 465)
    if port == 465 {
        return smtp.SendMailTLS(addr, auth, from, to, body)
    }
    return smtp.SendMail(addr, auth, from, to, body)
}
```

### TipTap Email Compose Editor (React)

```tsx
// Source: https://tiptap.dev/docs/editor/getting-started/install/react
import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Table from '@tiptap/extension-table'
import TableRow from '@tiptap/extension-table-row'
import TableCell from '@tiptap/extension-table-cell'
import TableHeader from '@tiptap/extension-table-header'
import Image from '@tiptap/extension-image'
import Link from '@tiptap/extension-link'
import Underline from '@tiptap/extension-underline'
import Color from '@tiptap/extension-color'
import TextStyle from '@tiptap/extension-text-style'

function EmailCompose() {
    const editor = useEditor({
        extensions: [
            StarterKit,
            Table.configure({ resizable: true }),
            TableRow, TableCell, TableHeader,
            Image, Link, Underline,
            Color, TextStyle,
        ],
        content: '',
    })

    const getHTML = () => editor?.getHTML() ?? ''

    return (
        <div className="border border-border rounded-lg">
            {/* Toolbar with formatting buttons */}
            <ComposeToolbar editor={editor} />
            <EditorContent editor={editor} className="prose max-w-none px-4 py-3" />
        </div>
    )
}
```

### Database Schema (Key Tables)

```sql
-- Email accounts (one per user, encrypted IMAP/SMTP credentials)
CREATE TABLE email_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    email_address VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    imap_host VARCHAR(255) NOT NULL,
    imap_port INTEGER NOT NULL DEFAULT 993,
    smtp_host VARCHAR(255) NOT NULL,
    smtp_port INTEGER NOT NULL DEFAULT 587,
    username VARCHAR(255) NOT NULL,
    password_encrypted TEXT NOT NULL,  -- Vault-encrypted
    use_ssl BOOLEAN NOT NULL DEFAULT true,
    last_sync_at TIMESTAMPTZ,
    sync_enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id)  -- One account per user in v1
);

-- Email folders (mapped from IMAP)
CREATE TABLE email_folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES email_accounts(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    imap_name VARCHAR(255) NOT NULL,  -- Original IMAP folder name
    folder_type VARCHAR(50) NOT NULL DEFAULT 'custom',  -- inbox, sent, drafts, trash, spam, archive, custom
    uid_validity BIGINT NOT NULL DEFAULT 0,
    highest_uid BIGINT NOT NULL DEFAULT 0,
    message_count INTEGER NOT NULL DEFAULT 0,
    unread_count INTEGER NOT NULL DEFAULT 0,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(account_id, imap_name)
);
CREATE INDEX idx_email_folders_account ON email_folders(account_id);

-- Email messages (cached from IMAP)
CREATE TABLE email_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES email_accounts(id) ON DELETE CASCADE,
    folder_id UUID NOT NULL REFERENCES email_folders(id) ON DELETE CASCADE,
    uid BIGINT NOT NULL,               -- IMAP UID
    message_id VARCHAR(998),           -- RFC 5322 Message-ID
    in_reply_to VARCHAR(998),          -- RFC 5322 In-Reply-To
    references TEXT[],                 -- RFC 5322 References (array)
    thread_id UUID,                    -- Computed thread group ID
    from_name VARCHAR(255),
    from_email VARCHAR(255) NOT NULL,
    to_addresses JSONB NOT NULL DEFAULT '[]',   -- [{name, email}]
    cc_addresses JSONB NOT NULL DEFAULT '[]',
    bcc_addresses JSONB NOT NULL DEFAULT '[]',
    subject VARCHAR(998),
    preview TEXT,                       -- First ~200 chars of body text
    body_text TEXT,                     -- Plain text body
    body_html TEXT,                     -- HTML body
    is_read BOOLEAN NOT NULL DEFAULT false,
    is_starred BOOLEAN NOT NULL DEFAULT false,
    is_draft BOOLEAN NOT NULL DEFAULT false,
    has_attachments BOOLEAN NOT NULL DEFAULT false,
    date TIMESTAMPTZ NOT NULL,         -- RFC 5322 Date header
    size_bytes BIGINT NOT NULL DEFAULT 0,
    raw_headers TEXT,                  -- Full headers for threading/debugging
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(folder_id, uid)
);
CREATE INDEX idx_email_messages_account ON email_messages(account_id);
CREATE INDEX idx_email_messages_folder ON email_messages(folder_id);
CREATE INDEX idx_email_messages_thread ON email_messages(thread_id);
CREATE INDEX idx_email_messages_date ON email_messages(date DESC);
CREATE INDEX idx_email_messages_message_id ON email_messages(message_id);
CREATE INDEX idx_email_messages_from ON email_messages(from_email);

-- Email-CRM contact links
CREATE TABLE email_contact_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES email_messages(id) ON DELETE CASCADE,
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    link_type VARCHAR(20) NOT NULL DEFAULT 'auto',  -- auto, manual
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(message_id, contact_id)
);
CREATE INDEX idx_email_contact_links_contact ON email_contact_links(contact_id);

-- Email attachments (metadata; files in MinIO)
CREATE TABLE email_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES email_messages(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    content_type VARCHAR(255) NOT NULL,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    minio_key VARCHAR(512) NOT NULL,   -- MinIO object key
    content_id VARCHAR(255),           -- For inline images (CID)
    is_inline BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_email_attachments_message ON email_attachments(message_id);

-- Email signatures (one active per user)
CREATE TABLE email_signatures (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    name VARCHAR(255) NOT NULL DEFAULT 'Standard',
    html_content TEXT NOT NULL DEFAULT '',
    is_default BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_email_signatures_user ON email_signatures(user_id);

-- Contact visibility extension (add to existing contacts table)
ALTER TABLE contacts ADD COLUMN visibility VARCHAR(20) NOT NULL DEFAULT 'shared';  -- shared, personal
ALTER TABLE contacts ADD COLUMN owner_id UUID REFERENCES users(id);
CREATE INDEX idx_contacts_visibility ON contacts(visibility);
CREATE INDEX idx_contacts_owner ON contacts(owner_id);
```

### Attachment Storage in MinIO

**Strategy (Claude's discretion recommendation):**
- Bucket: `kmuhub-files` (reuse existing)
- Key pattern: `email-attachments/{account_id}/{message_uid}/{filename}`
- During sync: stream attachment part directly from IMAP to MinIO (no memory buffering)
- On download: generate presigned URL (same as chat file downloads)
- Inline images: store in MinIO, replace CID references with presigned URLs when rendering HTML

### Email Search Implementation

**Approach (Claude's discretion recommendation):**
- Use PostgreSQL full-text search (same approach as CRM contacts and chat messages)
- Create a tsvector column on email_messages: `tsvector_col = to_tsvector('german', coalesce(subject,'') || ' ' || coalesce(body_text,'') || ' ' || coalesce(from_name,'') || ' ' || coalesce(from_email,''))`
- Add a GIN index on the tsvector column
- Search query uses `plainto_tsquery('german', ?)` with ranking via `ts_rank`
- This matches the existing pattern used for CRM search (PostgreSQL tsvector, German config)

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| go-imap v1 (IMAP4rev1) | go-imap v2 (IMAP4rev2, RFC 9051) | 2024-2025 | Built-in IDLE, modern API, no separate idle extension needed |
| net/smtp (Go stdlib, frozen) | emersion/go-smtp | Ongoing | Active development, STARTTLS, AUTH extensions, DSN |
| Custom MIME parsing | emersion/go-message | Stable | RFC 2045/2046/2047 compliant, handles charset/encoding edge cases |
| Quill/Draft.js editors | TipTap v3 / Lexical | 2024-2025 | TipTap v3 released; Draft.js deprecated by Meta; Quill 2.0 released but less extensible |

**Deprecated/outdated:**
- **Draft.js**: Deprecated by Meta in favor of Lexical. Do not use.
- **net/smtp (Go stdlib)**: Frozen, no new features. Use emersion/go-smtp.
- **go-imap v1**: Still works but IMAP4rev1 only. v2 recommended for new projects despite beta status.
- **go-imap-idle (separate package)**: Only needed for go-imap v1. v2 has IDLE built-in.

## Open Questions

1. **go-imap v2 beta stability in production**
   - What we know: v2.0.0-beta.8 (Dec 2025), actively maintained, modern API, IDLE built-in
   - What's unclear: Beta status means API may change. No v2 stable release yet.
   - Recommendation: Use v2 anyway. The API is mature enough for our use case. Pin to specific beta version. The alternative (v1 + separate idle extension) is worse. Monitor for v2 stable release and upgrade when available.

2. **IMAP OAuth2 support (Gmail, Office 365)**
   - What we know: Gmail/Office 365 are deprecating password-based IMAP access in favor of OAuth2. go-imap v2 supports AUTHENTICATE with SASL, which can carry OAuth2 tokens.
   - What's unclear: Whether DACH KMU customers predominantly use Gmail/O365 or self-hosted mail (Postfix, Exchange on-prem, Hetzner mail).
   - Recommendation: Implement password-based auth first (covers self-hosted mail, which is the primary target). Add OAuth2 flow as a v2 feature. Document this limitation.

3. **Email body storage size**
   - What we know: HTML emails can be large (especially with inline images). Storing full body_html in PostgreSQL works but adds DB size.
   - What's unclear: Average email size for DACH SMB workload.
   - Recommendation: Store full body (text + HTML) in PostgreSQL. For attachments, only store metadata in DB and files in MinIO. Monitor DB growth. If it becomes an issue, move body_html to MinIO and store only preview in DB. This is an optimization that can be done later without schema changes (just add a minio_key column).

4. **TipTap v3 vs v2 migration**
   - What we know: TipTap v3.19.0 is current. Some extensions may still reference v2 patterns.
   - What's unclear: Whether all needed extensions (table, image, color) are fully v3-compatible.
   - Recommendation: Install v3 and test. TipTap v3 migration is straightforward. All listed extensions have v3 releases.

## Sources

### Primary (HIGH confidence)
- [pkg.go.dev/github.com/emersion/go-imap/v2](https://pkg.go.dev/github.com/emersion/go-imap/v2) - v2.0.0-beta.8, full API reference, IDLE example
- [pkg.go.dev/github.com/emersion/go-imap/v2/imapclient](https://pkg.go.dev/github.com/emersion/go-imap/v2/imapclient) - Client methods, DialTLS, Fetch, Store, Idle, Search, Thread
- [pkg.go.dev/github.com/emersion/go-smtp](https://pkg.go.dev/github.com/emersion/go-smtp) - v0.24.0, SendMail, DialTLS, AUTH, STARTTLS
- [pkg.go.dev/github.com/emersion/go-vcard](https://pkg.go.dev/github.com/emersion/go-vcard) - RFC 6350, Decoder/Encoder API
- [npmjs.com/@tiptap/react](https://www.npmjs.com/package/@tiptap/react) - v3.19.0
- [tiptap.dev/docs/editor/getting-started/install/react](https://tiptap.dev/docs/editor/getting-started/install/react) - Installation, setup
- [RFC 5322 - Internet Message Format](https://www.rfc-editor.org/rfc/rfc5322) - Message-ID, In-Reply-To, References
- [RFC 5256 - IMAP SORT and THREAD](https://www.rfc-editor.org/rfc/rfc5256.html) - REFERENCES threading algorithm
- [RFC 4549 - Synchronization Operations for Disconnected IMAP4 Clients](https://datatracker.ietf.org/doc/html/rfc4549) - UIDVALIDITY, sync strategy

### Secondary (MEDIUM confidence)
- [github.com/gatherstars-com/jwz](https://github.com/gatherstars-com/jwz) - JWZ threading Go implementation, v1.4.0, Apache-2.0
- [github.com/emersion/go-imap](https://github.com/emersion/go-imap) - Repository README, v2 development status
- [github.com/emersion/go-smtp](https://github.com/emersion/go-smtp) - v0.24.0, RFC compliance
- [github.com/gocarina/gocsv](https://github.com/gocarina/gocsv) - CSV serialization for Go
- [liveblocks.io/blog/which-rich-text-editor-framework-should-you-choose-in-2025](https://liveblocks.io/blog/which-rich-text-editor-framework-should-you-choose-in-2025) - TipTap vs Lexical vs Slate comparison

### Tertiary (LOW confidence)
- [nylas.com/blog/guide-to-imap-send-and-sync-mail](https://www.nylas.com/blog/guide-to-imap-send-and-sync-mail/) - IMAP sync architecture patterns (commercial, may be biased)
- [en.wikipedia.org/wiki/Conversation_threading](https://en.wikipedia.org/wiki/Conversation_threading) - Threading overview

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All libraries verified via pkg.go.dev and npm with exact version numbers
- Architecture: HIGH - Follows exact same patterns as existing services (crm, work, chat); no novel architectural decisions
- Email protocol handling: HIGH - Libraries from same author (emersion) designed to work together
- Threading algorithm: HIGH - JWZ algorithm is well-documented (RFC 5256, jwz.org); Go implementation exists
- Frontend editor: MEDIUM - TipTap v3 is current but email compose is a specific use case; may need custom extensions
- IMAP sync engine: MEDIUM - Architecture is sound but implementation complexity is high (connection lifecycle, error recovery, multi-folder sync)
- Pitfalls: HIGH - Based on established email client engineering knowledge (UIDVALIDITY, connection management, HTML security)

**Research date:** 2026-02-16
**Valid until:** 2026-03-16 (30 days - libraries are stable, email protocols don't change)

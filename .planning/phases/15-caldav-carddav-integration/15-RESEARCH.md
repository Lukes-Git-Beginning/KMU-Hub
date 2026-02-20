# Phase 15: CalDAV/CardDAV Integration - Research

**Researched:** 2026-02-20
**Domain:** CalDAV (RFC 4791) / CardDAV (RFC 6352) protocol servers, iCalendar (RFC 5545), vCard (RFC 6350), auto-discovery (RFC 6764), sync tokens (RFC 6578)
**Confidence:** HIGH

## Summary

CalDAV and CardDAV are mature, well-standardized WebDAV extensions for calendar and contact synchronization. The Go ecosystem has a clear winner: `emersion/go-webdav` (v0.7.0) provides production-grade CalDAV and CardDAV server handlers that implement `http.Handler` and delegate to a Backend interface. KMU Hub already uses `emersion/go-vcard` and `teambition/rrule-go` -- both are dependencies of `go-webdav/go-ical`, so the integration is natural.

The primary challenge is not protocol implementation (go-webdav handles that) but building the Backend adapters that translate between go-webdav's path-based CalDAV/CardDAV model and KMU Hub's UUID-based PostgreSQL data model. Key concerns: (1) CalDAV uses HTTP Basic Auth, not JWT -- requiring app-specific passwords with bcrypt hashing, (2) ETag/CTag/sync-token generation for efficient incremental sync, (3) correct iCalendar serialization of recurring events with EXDATE/RECURRENCE-ID exceptions, (4) per-user context injection since go-webdav's Backend interface is stateless.

**Primary recommendation:** Use `emersion/go-webdav` v0.7.0 with custom CalDAV/CardDAV Backend implementations that call existing calendar and CRM services via gRPC. Run CalDAV/CardDAV handlers in the gateway process (like WOPI) with custom Basic Auth middleware for app-specific passwords. Use `emersion/go-ical` for iCalendar data construction.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Expose personal calendar AND team calendars the user belongs to
- Each calendar appears as a separate CalDAV collection in the client
- Events created from external clients get full KMU Hub treatment -- attendees resolved to KMU Hub users, reminders synced, categories mapped
- Full recurring event exception support -- single-occurrence edits from external clients create proper EXDATE/exception records
- Two separate CardDAV address books per user: "My Contacts" (personal) and "Company Contacts" (shared)
- Contacts created via CardDAV land as basic contacts (name/email/phone/address) -- no CRM-specific features until manually enriched in KMU Hub
- Default visibility for CardDAV-created contacts: determined by which address book they're added to
- Last-write-wins conflict resolution using ETags
- Implement RFC 6578 sync tokens from day one for efficient incremental sync
- Push notifications where clients support it (RFC 7230 / Apple push), polling as fallback
- Full auto-discovery via .well-known/caldav and .well-known/carddav (RFC 6764)
- App-specific passwords for CalDAV/CardDAV auth -- user generates dedicated passwords in KMU Hub settings, individually revocable
- In-app setup wizard with step-by-step instructions per client, password generation, URL display, and connection testing
- Target clients for testing and setup instructions: Microsoft Outlook (desktop), Mozilla Thunderbird, macOS Calendar/Contacts
- Role-based write access for team calendars via CalDAV: admins/managers can write, members get read-only
- Personal calendars: full read/write for the owning user
- Company Contacts address book: admins/managers can create/edit, members get read-only
- Personal address book: full read/write for the owning user
- Full admin visibility: admins can see which users have CalDAV/CardDAV enabled, revoke app passwords, disable sync per user
- Global organization toggle: admin can enable/disable CalDAV/CardDAV organization-wide (off by default)

### Claude's Discretion
- Go library choice for CalDAV/CardDAV server implementation (go-webdav or alternatives)
- ETag generation strategy and storage
- Push notification implementation details
- CalDAV/CardDAV URL path structure
- App-specific password hashing and storage approach

### Deferred Ideas (OUT OF SCOPE)
- Mobile client support (iOS Calendar/Contacts, DAVx5 for Android) -- future enhancement
- Linux client support (GNOME Calendar, Evolution) -- future enhancement
- WebDAV for file access -- Phase 15 originally separate, now covered by download/upload flow
- SRV DNS records for enterprise auto-discovery -- nice-to-have after .well-known works
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| INT-01 | System exposes a CalDAV endpoint for bidirectional calendar sync with Outlook, Thunderbird, and macOS Calendar | go-webdav CalDAV Handler + custom Backend adapter translating to existing calendar gRPC service |
| INT-02 | System exposes a CardDAV endpoint for bidirectional contact sync with external clients | go-webdav CardDAV Handler + custom Backend adapter translating to existing CRM contact gRPC service |
| INT-03 | CalDAV/CardDAV supports per-user authenticated access with proper ACL | App-specific passwords with HTTP Basic Auth, role-based permission checking in Backend methods |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| emersion/go-webdav | v0.7.0 | CalDAV/CardDAV server handlers (implements http.Handler) | Only mature Go library with both CalDAV + CardDAV server support. Used by tokidoki and other production servers. Active maintenance (last release Oct 2025). |
| emersion/go-ical | v0.0.0 (latest) | iCalendar (RFC 5545) data construction and parsing | Companion to go-webdav; provides Calendar/Event/Component types that go-webdav's CalDAV handler uses internally. MIT license. |
| emersion/go-vcard | v0.0.0-20241024213814 | vCard 4.0 construction and parsing | **Already in go.mod.** Used by existing CRM contact export (Phase 10). go-webdav's CardDAV handler uses it internally. |
| teambition/rrule-go | v1.8.2 | RRULE recurrence rule parsing and expansion | **Already in go.mod.** Used by existing calendar service (Phase 7). go-ical uses it for RecurrenceSet. |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| golang.org/x/crypto/bcrypt | (in go.mod) | App-specific password hashing | Hash app-specific passwords at creation, verify on each CalDAV/CardDAV request |
| crypto/rand | stdlib | Generating secure random app-specific passwords | 128-bit random passwords for app-specific tokens |
| crypto/subtle | stdlib | Constant-time password comparison | Prevent timing attacks on Basic Auth validation |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| emersion/go-webdav | samedi/caldav-go | caldav-go is CalDAV only (no CardDAV), less maintained, no server Handler |
| emersion/go-ical | arran4/golang-ical | golang-ical has more GitHub stars but is not the companion to go-webdav; using both would mean type conversion overhead |
| App-specific passwords | OAuth2 for CalDAV | No major CalDAV client supports OAuth2 reliably; Basic Auth is the universal standard |

**Installation:**
```bash
cd backend
go get github.com/emersion/go-webdav@v0.7.0
go get github.com/emersion/go-ical@latest
# go-vcard and rrule-go already in go.mod
```

## Architecture Patterns

### Recommended Project Structure
```
backend/
├── internal/
│   ├── caldav/                    # CalDAV/CardDAV domain
│   │   ├── caldav_backend.go      # caldav.Backend implementation
│   │   ├── carddav_backend.go     # carddav.Backend implementation
│   │   ├── ical_converter.go      # CalendarEvent <-> ical.Calendar conversion
│   │   ├── vcard_converter.go     # Contact <-> vcard.Card conversion
│   │   ├── app_password.go        # App-specific password service
│   │   ├── app_password_repo.go   # Repository interface
│   │   ├── postgres_app_password.go # PostgreSQL repository
│   │   ├── sync_token.go          # Sync token generation and validation
│   │   └── errors.go              # Domain errors
│   ├── gateway/
│   │   ├── route_caldav.go        # CalDAV/CardDAV HTTP routing + Basic Auth middleware
│   │   └── ...
│   └── ...
├── migrations/
│   ├── 000049_create_app_passwords.up.sql    # App-specific passwords table
│   ├── 000049_create_app_passwords.down.sql
│   ├── 000050_create_caldav_sync.up.sql      # Sync tokens + change tracking
│   └── 000050_create_caldav_sync.down.sql
└── ...
```

### Pattern 1: Gateway-Hosted CalDAV/CardDAV Handlers (like WOPI)
**What:** CalDAV/CardDAV handlers run inside the gateway process, not as separate microservices. They use gRPC to communicate with backend services (work/calendar, crm) for actual data operations.
**When to use:** Always -- CalDAV/CardDAV is a protocol translation layer, not a business logic service.
**Why:** Same pattern as WOPI (route_wopi.go): protocol-specific HTTP handler in gateway, gRPC for data. CalDAV/CardDAV have their own auth (Basic Auth, not JWT), same as WOPI (access_token, not JWT).
**Example:**
```go
// In cmd/gateway/main.go
caldavBackend := caldavpkg.NewCalDAVBackend(registry, pool)
carddavBackend := caldavpkg.NewCardDAVBackend(registry, pool)
appPasswordService := caldavpkg.NewAppPasswordService(pool)

caldavHandler := &caldav.Handler{
    Backend: caldavBackend,
    Prefix:  "/caldav",
}
carddavHandler := &carddav.Handler{
    Backend: carddavBackend,
    Prefix:  "/carddav",
}

caldavRoutes := gateway.NewCalDAVRoutes(caldavHandler, carddavHandler, appPasswordService)
caldavRoutes.RegisterRoutes(r, nil) // no standard auth -- uses Basic Auth internally
```

### Pattern 2: Per-User Backend via Context Injection
**What:** go-webdav's Backend interface methods receive `context.Context` but have no explicit user parameter. Inject the authenticated user ID into the context via middleware, then extract it in the Backend implementation.
**When to use:** Every Backend method that needs to know "who is the current user."
**Example:**
```go
// Context key for CalDAV user
type caldavUserKey struct{}

// Basic Auth middleware sets user in context
func basicAuthMiddleware(appPasswordService *AppPasswordService) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            username, password, ok := r.BasicAuth()
            if !ok {
                w.Header().Set("WWW-Authenticate", `Basic realm="KMU Hub CalDAV"`)
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }

            userID, err := appPasswordService.Validate(r.Context(), username, password)
            if err != nil {
                w.Header().Set("WWW-Authenticate", `Basic realm="KMU Hub CalDAV"`)
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }

            ctx := context.WithValue(r.Context(), caldavUserKey{}, userID)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// Backend extracts user from context
func (b *CalDAVBackend) CurrentUserPrincipal(ctx context.Context) (string, error) {
    userID := ctx.Value(caldavUserKey{}).(uuid.UUID)
    return fmt.Sprintf("/caldav/principals/%s/", userID), nil
}

func (b *CalDAVBackend) CalendarHomeSetPath(ctx context.Context) (string, error) {
    userID := ctx.Value(caldavUserKey{}).(uuid.UUID)
    return fmt.Sprintf("/caldav/principals/%s/calendars/", userID), nil
}
```

### Pattern 3: ETag from UpdatedAt + UUID Hash
**What:** Generate deterministic ETags from the combination of entity UUID and updated_at timestamp. Store nothing extra -- derive from existing data.
**When to use:** For individual calendar objects and address objects.
**Example:**
```go
func generateETag(id uuid.UUID, updatedAt time.Time) string {
    h := sha256.New()
    h.Write(id[:])
    binary.Write(h, binary.BigEndian, updatedAt.UnixNano())
    return fmt.Sprintf(`"%x"`, h.Sum(nil)[:16])
}
```

### Pattern 4: CTag/Sync-Token from Sequence Counter
**What:** Each calendar/address book has a monotonically increasing `sync_version` counter in the database. Incremented on every change (insert/update/delete of objects). The sync-token is this counter value, and CTag derives from it.
**When to use:** For collection-level change detection (RFC 6578).
**Example:**
```sql
-- sync_versions table tracks per-collection change sequence
CREATE TABLE caldav_sync_versions (
    collection_type VARCHAR(20) NOT NULL,  -- 'calendar', 'addressbook'
    collection_id UUID NOT NULL,
    sync_version BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (collection_type, collection_id)
);

-- Change log for sync-collection REPORT
CREATE TABLE caldav_change_log (
    id BIGSERIAL PRIMARY KEY,
    collection_type VARCHAR(20) NOT NULL,
    collection_id UUID NOT NULL,
    object_path TEXT NOT NULL,
    change_type VARCHAR(10) NOT NULL,  -- 'created', 'modified', 'deleted'
    sync_version BIGINT NOT NULL,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Pattern 5: URL Path Structure
**What:** Hierarchical paths that map to CalDAV/CardDAV collections and objects.
**When to use:** Defines the full URL scheme for client configuration.
**Structure:**
```
CalDAV:
  /.well-known/caldav                                    -> 301 to /caldav/
  /caldav/principals/{user_id}/                          -> user principal
  /caldav/principals/{user_id}/calendars/                -> calendar home set
  /caldav/principals/{user_id}/calendars/{calendar_id}/  -> calendar collection
  /caldav/principals/{user_id}/calendars/{calendar_id}/{event_uid}.ics  -> calendar object

CardDAV:
  /.well-known/carddav                                   -> 301 to /carddav/
  /carddav/principals/{user_id}/                         -> user principal
  /carddav/principals/{user_id}/addressbooks/            -> address book home set
  /carddav/principals/{user_id}/addressbooks/personal/   -> personal address book
  /carddav/principals/{user_id}/addressbooks/company/    -> company address book
  /carddav/principals/{user_id}/addressbooks/{book}/{contact_id}.vcf  -> address object
```

### Anti-Patterns to Avoid
- **Running CalDAV/CardDAV as separate microservice:** Adds unnecessary complexity. It is a protocol adapter, not a business service. Run in gateway (like WOPI).
- **Storing iCalendar/vCard blobs in the database:** KMU Hub already has structured calendar_events and contacts tables. Serialize to iCal/vCard on read, parse on write. Never store raw protocol data.
- **Using JWT for CalDAV/CardDAV auth:** No CalDAV/CardDAV client supports Bearer tokens. HTTP Basic Auth is the universal standard. Must use app-specific passwords (never the user's main password).
- **Implementing CalDAV REPORT methods from scratch:** go-webdav handles all PROPFIND/REPORT/MKCALENDAR methods. Only implement the Backend interface.
- **Trying to extend existing calendar gRPC service for CalDAV:** CalDAV needs path-based lookups and raw iCal data. Keep the Backend adapter in the gateway; call gRPC for data, but do iCal conversion locally.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| CalDAV/CardDAV protocol handling | Custom WebDAV XML parser, PROPFIND handler | emersion/go-webdav Handler | XML multiget, depth handling, conditional requests, DAV headers are extremely complex. go-webdav handles all of this. |
| iCalendar serialization | Custom VCALENDAR string builder | emersion/go-ical Encoder/Decoder | Line folding (75 chars), property escaping, VTIMEZONE generation, RRULE serialization all have edge cases. |
| vCard serialization | Custom vCard builder | emersion/go-vcard Encoder/Decoder | Already used in codebase. Handles vCard 3.0 and 4.0 differences that clients expect. |
| RRULE expansion | Manual recurrence calculation | teambition/rrule-go | Already used in codebase. Handles BYDAY, BYMONTH, INTERVAL, COUNT, UNTIL. |
| HTTP Basic Auth parsing | Manual header parsing | net/http r.BasicAuth() | stdlib, constant-time safe. |
| Password hashing for app-specific passwords | Custom hashing | golang.org/x/crypto/bcrypt | Already used in auth service. bcrypt with cost 12 is the standard. |
| .well-known redirect | Custom redirect handler | Simple chi.Get route returning 301 | Trivial but must be correct (RFC 6764 requires redirect, not proxy). |

**Key insight:** CalDAV/CardDAV is an interoperability protocol with decades of client quirks. The value is in correct protocol handling (go-webdav) and correct data mapping (Backend adapters). Building protocol handling from scratch would take months and still break with certain clients.

## Common Pitfalls

### Pitfall 1: Recurring Event Exception Mapping
**What goes wrong:** External client edits a single occurrence of a recurring event. The CalDAV REPORT returns a VEVENT with RECURRENCE-ID, but the server fails to create the corresponding EventException record or handles EXDATE incorrectly.
**Why it happens:** KMU Hub stores exceptions in an `event_exceptions` table with structured fields, but CalDAV clients send/expect raw iCalendar data with EXDATE and RECURRENCE-ID properties. The mapping between these models is non-trivial.
**How to avoid:** When converting KMU Hub events to iCalendar, produce one master VEVENT with RRULE plus one VEVENT per exception (with RECURRENCE-ID). When parsing incoming iCalendar, detect RECURRENCE-ID VEVENTs and create/update EventException records. Handle cancelled exceptions as EXDATE on the master event (some clients use STATUS:CANCELLED instead of EXDATE -- support both).
**Warning signs:** Thunderbird shows duplicate events; Outlook drops exceptions; macOS Calendar shows "phantom" cancelled occurrences.

### Pitfall 2: ETag/If-Match Mismatch
**What goes wrong:** Client sends PUT with If-Match containing an ETag, but the server generates a different ETag for the same resource, causing 412 Precondition Failed errors and sync failures.
**Why it happens:** ETags must be byte-identical between GET responses and subsequent If-Match headers. If the ETag generation changes or is inconsistent, clients cannot sync.
**How to avoid:** Use a deterministic ETag formula (hash of UUID + updated_at). Never include whitespace or formatting differences. Test that the ETag returned from PutCalendarObject matches the one clients send back in subsequent If-Match.
**Warning signs:** Clients report "conflict" on every save; sync loops where the same event is synced repeatedly.

### Pitfall 3: Outlook Does Not Support CalDAV Natively
**What goes wrong:** Team assumes Outlook supports CalDAV out of the box and tests against it directly.
**Why it happens:** Microsoft Outlook for Windows/Mac does NOT have native CalDAV support. It requires the third-party "CalDav Synchronizer" plugin (free, open source).
**How to avoid:** Document in the setup wizard that Outlook requires "CalDav Synchronizer" plugin. Test primarily against Thunderbird (native CalDAV) and macOS Calendar (native CalDAV). Test Outlook only with CalDav Synchronizer installed.
**Warning signs:** Users report "can't add CalDAV account" in Outlook; setup instructions reference native Outlook CalDAV settings that don't exist.

### Pitfall 4: Address Book Visibility and CardDAV Write Permissions
**What goes wrong:** A member-role user creates a contact via CardDAV in the "Company Contacts" address book, bypassing the KMU Hub role check.
**Why it happens:** CardDAV Backend.PutAddressObject receives the path but doesn't check whether the user has write permission for that address book.
**How to avoid:** In PutAddressObject, extract the address book type from the path. If it's "company", check that the user has admin or manager role. If it's "personal", allow. Return 403 Forbidden via `webdav.NewHTTPError(http.StatusForbidden, ...)` for unauthorized writes.
**Warning signs:** Members can create company-wide contacts from Thunderbird; shared contacts appear that shouldn't.

### Pitfall 5: Sync Token Not Updated on Cascade Deletes
**What goes wrong:** A calendar is deleted (CASCADE deletes all events), but the sync_version for that calendar is not updated, causing clients to never learn about the deletions.
**Why it happens:** PostgreSQL ON DELETE CASCADE happens at the DB level; the application code that increments sync_version is never called.
**How to avoid:** Use a PostgreSQL trigger on `calendar_events` and `contacts` tables that auto-increments the corresponding sync_version entry and logs changes to the change_log table. OR handle deletes explicitly at the service layer before calling the repository delete.
**Warning signs:** Clients keep showing deleted events/contacts; manual full-sync required.

### Pitfall 6: Client-Specific VTIMEZONE Handling
**What goes wrong:** Events show at wrong times in certain clients because VTIMEZONE components are missing or incorrect.
**Why it happens:** Some clients (especially older Outlook + CalDav Synchronizer) require explicit VTIMEZONE components. Others (macOS Calendar) handle timezone names without VTIMEZONE. If the server omits VTIMEZONE, some clients assume UTC.
**How to avoid:** Always include VTIMEZONE components for all referenced timezone IDs in VEVENT objects. go-ical does not auto-generate VTIMEZONE -- build a helper that creates VTIMEZONE from Go's time.Location IANA data.
**Warning signs:** Events appear 1-2 hours off in certain clients; all-day events shift dates across timezone boundary.

## Code Examples

Verified patterns from official sources:

### CalDAV Backend - ListCalendars
```go
// Source: go-webdav caldav.Backend interface
func (b *CalDAVBackend) ListCalendars(ctx context.Context) ([]caldav.Calendar, error) {
    userID := b.userFromCtx(ctx)

    // Call existing calendar gRPC service
    conn, err := b.registry.GetConnection("work")
    if err != nil {
        return nil, fmt.Errorf("calendar service unavailable: %w", err)
    }
    client := calendarv1.NewCalendarServiceClient(conn)

    resp, err := client.ListCalendars(ctx, &calendarv1.ListCalendarsRequest{
        UserId:        userID.String(),
        IncludeHidden: false,
    })
    if err != nil {
        return nil, err
    }

    var calendars []caldav.Calendar
    for _, cal := range resp.Calendars {
        // Determine supported components
        components := []string{"VEVENT"}

        calendars = append(calendars, caldav.Calendar{
            Path:                  fmt.Sprintf("/caldav/principals/%s/calendars/%s/", userID, cal.Calendar.Id),
            Name:                  cal.Calendar.Name,
            Description:           stringOrEmpty(cal.Calendar.Description),
            SupportedComponentSet: components,
        })
    }

    return calendars, nil
}
```

### iCalendar Event Construction
```go
// Source: go-ical encoder example + RFC 5545
func eventToICal(event *models.CalendarEvent, exceptions []models.EventException, attendees []models.EventAttendee) *ical.Calendar {
    cal := ical.NewCalendar()
    cal.Props.SetText(ical.PropVersion, "2.0")
    cal.Props.SetText(ical.PropProductID, "-//KMU Hub//CalDAV//DE")

    // Master event
    vevent := ical.NewEvent()
    vevent.Props.SetText(ical.PropUID, event.ID.String())
    vevent.Props.SetDateTime(ical.PropDateTimeStamp, event.UpdatedAt)
    vevent.Props.SetText(ical.PropSummary, event.Title)

    if event.IsAllDay {
        vevent.Props.SetDate(ical.PropDateTimeStart, event.StartTime)
        vevent.Props.SetDate(ical.PropDateTimeEnd, event.EndTime)
    } else {
        vevent.Props.SetDateTime(ical.PropDateTimeStart, event.StartTime)
        vevent.Props.SetDateTime(ical.PropDateTimeEnd, event.EndTime)
    }

    if event.Description != nil {
        vevent.Props.SetText(ical.PropDescription, *event.Description)
    }
    if event.Location != nil {
        vevent.Props.SetText(ical.PropLocation, *event.Location)
    }
    if event.RRule != nil {
        vevent.Props.SetText(ical.PropRecurrenceRule, *event.RRule)
    }

    // Add EXDATE for cancelled exceptions
    for _, exc := range exceptions {
        if exc.IsCancelled {
            prop := ical.NewProp(ical.PropExceptionDates)
            prop.SetDate(exc.OriginalDate)
            vevent.Props.Add(prop)
        }
    }

    // Add attendees
    for _, att := range attendees {
        prop := ical.NewProp(ical.PropAttendee)
        prop.Value = fmt.Sprintf("urn:uuid:%s", att.UserID)
        prop.Params.Set(ical.ParamCommonName, att.FirstName+" "+att.LastName)
        prop.Params.Set(ical.ParamParticipationStatus, rsvpToPartStat(att.RSVPStatus))
        vevent.Props.Add(prop)
    }

    cal.Children = append(cal.Children, vevent.Component)

    // Exception events (modified occurrences)
    for _, exc := range exceptions {
        if !exc.IsCancelled && hasOverrides(exc) {
            excEvent := ical.NewEvent()
            excEvent.Props.SetText(ical.PropUID, event.ID.String())
            excEvent.Props.SetDate(ical.PropRecurrenceID, exc.OriginalDate)

            title := event.Title
            if exc.Title != nil {
                title = *exc.Title
            }
            excEvent.Props.SetText(ical.PropSummary, title)

            if exc.StartTime != nil {
                excEvent.Props.SetDateTime(ical.PropDateTimeStart, *exc.StartTime)
            }
            if exc.EndTime != nil {
                excEvent.Props.SetDateTime(ical.PropDateTimeEnd, *exc.EndTime)
            }

            excEvent.Props.SetDateTime(ical.PropDateTimeStamp, exc.UpdatedAt)
            cal.Children = append(cal.Children, excEvent.Component)
        }
    }

    return cal
}
```

### CardDAV Backend - PutAddressObject with ACL
```go
// Source: go-webdav carddav.Backend interface
func (b *CardDAVBackend) PutAddressObject(ctx context.Context, path string, card vcard.Card, opts *carddav.PutAddressObjectOptions) (*carddav.AddressObject, error) {
    userID := b.userFromCtx(ctx)
    bookType := b.bookTypeFromPath(path) // "personal" or "company"

    // ACL check: company address book requires admin/manager role
    if bookType == "company" {
        roles := b.getUserRoles(ctx, userID)
        if !containsAny(roles, "admin", "manager") {
            return nil, webdav.NewHTTPError(http.StatusForbidden,
                fmt.Errorf("insufficient permissions for company address book"))
        }
    }

    // Parse vCard fields
    fn := card.Get(vcard.FieldFormattedName)
    names := card.Get(vcard.FieldName)
    email := card.Get(vcard.FieldEmail)
    phone := card.Get(vcard.FieldTelephone)

    // Map to CRM contact fields and create/update via gRPC
    // ...

    return &carddav.AddressObject{
        Path:    path,
        ModTime: time.Now(),
        ETag:    generateETag(contactID, time.Now()),
        Card:    card,
    }, nil
}
```

### App-Specific Password Table
```sql
-- Migration 000049: App-specific passwords for CalDAV/CardDAV
CREATE TABLE app_specific_passwords (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label VARCHAR(100) NOT NULL,           -- user-chosen name, e.g. "Thunderbird Buero"
    password_hash VARCHAR(255) NOT NULL,   -- bcrypt hash
    password_prefix VARCHAR(4) NOT NULL,   -- first 4 chars for identification in UI
    scope VARCHAR(20) NOT NULL DEFAULT 'caldav_carddav',  -- scope of access
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ                 -- NULL = active, set = revoked
);

CREATE INDEX idx_app_passwords_user ON app_specific_passwords (user_id);
CREATE INDEX idx_app_passwords_active ON app_specific_passwords (user_id) WHERE revoked_at IS NULL;
```

### .well-known Auto-Discovery
```go
// Source: RFC 6764
func registerWellKnown(r chi.Router) {
    // CalDAV auto-discovery
    r.Get("/.well-known/caldav", func(w http.ResponseWriter, r *http.Request) {
        http.Redirect(w, r, "/caldav/", http.StatusMovedPermanently)
    })

    // CardDAV auto-discovery
    r.Get("/.well-known/carddav", func(w http.ResponseWriter, r *http.Request) {
        http.Redirect(w, r, "/carddav/", http.StatusMovedPermanently)
    })
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| CTag (Apple proprietary) for collection change tracking | RFC 6578 sync-token (WebDAV Sync REPORT) | CTag deprecated 2015 | Sync-token is the standard; CTag can be derived from sync-token for backward compat |
| Full collection re-download on each sync | Incremental sync via sync-collection REPORT | RFC 6578 (2012) | Massively reduces bandwidth; essential for mobile clients |
| Custom CalDAV auth per server | App-specific passwords (industry standard) | ~2015 (Google, Apple, Fastmail) | Users never expose main password to CalDAV clients; individually revocable |
| Manual CalDAV URL entry | .well-known auto-discovery (RFC 6764) | 2013 | User enters email/hostname only; client discovers CalDAV/CardDAV endpoints automatically |

**Deprecated/outdated:**
- CTag (getctag property): Deprecated in favor of DAV:sync-token. Still supported by servers for backward compatibility with old clients, but sync-token is the primary mechanism.
- CalDAV without HTTPS: All modern clients require or strongly prefer HTTPS for CalDAV/CardDAV. The gateway must run behind TLS (handled by reverse proxy in production).

## Open Questions

1. **Push Notifications (RFC 7230 / Apple Push)**
   - What we know: Apple push for CalDAV is Apple-proprietary, requires registering with Apple. RFC 7230 is an unrelated HTTP spec (the correct reference may be draft-thomson-webpush). Thunderbird and Outlook CalDav Synchronizer do not support push -- they poll.
   - What's unclear: Whether implementing push is worth the effort for the 3 target clients (Outlook, Thunderbird, macOS). macOS Calendar supports Apple push, but only from Apple-registered servers.
   - Recommendation: Defer push notification implementation. All three target clients poll at configurable intervals (15min for macOS, configurable for Thunderbird/Outlook). The sync-token ensures polling is efficient (returns empty changeset when nothing changed). Add push as a future enhancement when mobile clients (DAVx5, iOS) are in scope.

2. **VTIMEZONE Component Generation**
   - What we know: go-ical provides the component types but does not auto-generate VTIMEZONE definitions from IANA timezone names. Some clients require VTIMEZONE, others work with timezone name alone.
   - What's unclear: Whether a minimal approach (include VTIMEZONE for Europe/Berlin only since Deutschland-first) is sufficient, or if all referenced timezones need full VTIMEZONE.
   - Recommendation: Build a small VTIMEZONE generator using Go's time.LoadLocation + tzdata. Cache generated VTIMEZONE components. Include for all referenced timezones. This prevents timezone-related bugs across all clients.

3. **Attendee Mapping in CalDAV**
   - What we know: CalDAV represents attendees as ATTENDEE properties with mailto: or urn:uuid: URIs. KMU Hub stores attendees as user_id UUIDs with RSVP status.
   - What's unclear: How to resolve an ATTENDEE mailto:hans@example.com from an external client to a KMU Hub user_id, and what to do if the email doesn't match any user.
   - Recommendation: Use urn:uuid:{user_id} as ATTENDEE value for outgoing iCalendar. For incoming, try to resolve by email first (GetByEmail in CRM), fall back to user lookup. If no match, ignore the attendee (external attendees not supported in v1 calendar).

## Sources

### Primary (HIGH confidence)
- [pkg.go.dev/github.com/emersion/go-webdav/caldav](https://pkg.go.dev/github.com/emersion/go-webdav/caldav) - CalDAV Backend interface, Handler type, Calendar/CalendarObject types
- [pkg.go.dev/github.com/emersion/go-webdav/carddav](https://pkg.go.dev/github.com/emersion/go-webdav/carddav) - CardDAV Backend interface, Handler type, AddressBook/AddressObject types
- [pkg.go.dev/github.com/emersion/go-webdav](https://pkg.go.dev/github.com/emersion/go-webdav) - UserPrincipalBackend interface, ConditionalMatch, NewHTTPError
- [pkg.go.dev/github.com/emersion/go-ical](https://pkg.go.dev/github.com/emersion/go-ical) - Complete iCalendar type system, all property/component constants, Encoder/Decoder
- [github.com/emersion/go-webdav/releases](https://github.com/emersion/go-webdav/releases) - Latest version v0.7.0 (Oct 2025)
- [RFC 6764](https://datatracker.ietf.org/doc/html/rfc6764) - .well-known/caldav and .well-known/carddav auto-discovery
- [RFC 6578](https://datatracker.ietf.org/doc/html/rfc6578) - WebDAV sync-token and sync-collection REPORT
- Existing codebase: calendar.proto (40 RPCs), contact.go model, calendar_event.go model, export_service.go (vCard export), route_wopi.go (non-JWT auth pattern)

### Secondary (MEDIUM confidence)
- [sabre.io/dav/building-a-caldav-client](https://sabre.io/dav/building-a-caldav-client/) - Client-side sync protocol flow (CTag vs sync-token, ETag verification)
- [devguide.calconnect.org/CardDAV/building-a-carddav-client](https://devguide.calconnect.org/CardDAV/building-a-carddav-client/) - CardDAV discovery and sync flow
- [kb.mailbox.org CalDAV clients](https://kb.mailbox.org/de/privat/adressbuch-und-kalender/caldav-clients-outlook-thunderbird-macos-evolution-kontact/) - Client compatibility notes (Outlook needs plugin, Thunderbird native, macOS native)
- [DAVx5 technical documentation](https://manual.davx5.com/technical_information.html) - Sync token and ETag handling patterns

### Tertiary (LOW confidence)
- Push notification for CalDAV: The CONTEXT.md references "RFC 7230 / Apple push" but RFC 7230 is HTTP/1.1 Message Syntax, not push. The actual Apple CalDAV push spec is Apple-proprietary. Deferred per open questions.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - go-webdav is the clear Go ecosystem choice, already uses emersion/go-vcard and teambition/rrule-go from existing codebase
- Architecture: HIGH - Gateway-hosted pattern proven by WOPI implementation, Backend interface well-documented
- Pitfalls: HIGH - CalDAV/CardDAV is mature (~20 years), pitfalls well-documented by community
- Sync token/ETag strategy: MEDIUM - Approach is sound but specific details (trigger-based vs application-level sync_version increment) need validation during implementation
- Push notifications: LOW - Deferred; client support is inconsistent across target clients

**Research date:** 2026-02-20
**Valid until:** 2026-03-20 (stable domain, go-webdav release cycle ~4 months)

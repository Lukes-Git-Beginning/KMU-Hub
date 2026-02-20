# Phase 15: CalDAV/CardDAV Integration - Context

**Gathered:** 2026-02-20
**Status:** Ready for planning

<domain>
## Phase Boundary

Expose KMU Hub calendar events and CRM contacts via standard CalDAV/CardDAV protocols for bidirectional sync with external clients (Outlook, Thunderbird, macOS). Includes auto-discovery, app-specific passwords, and role-based access control. Does NOT include new calendar/contact features — only protocol-level access to existing data.

</domain>

<decisions>
## Implementation Decisions

### Calendar scope
- Expose personal calendar AND team calendars the user belongs to
- Each calendar appears as a separate CalDAV collection in the client
- Events created from external clients get full KMU Hub treatment — attendees resolved to KMU Hub users, reminders synced, categories mapped
- Full recurring event exception support — single-occurrence edits from external clients create proper EXDATE/exception records

### Address book structure
- Two separate CardDAV address books per user: "My Contacts" (personal) and "Company Contacts" (shared)
- Contacts created via CardDAV land as basic contacts (name/email/phone/address) — no CRM-specific features until manually enriched in KMU Hub
- Default visibility for CardDAV-created contacts: determined by which address book they're added to

### Sync behavior
- Last-write-wins conflict resolution using ETags
- Implement RFC 6578 sync tokens from day one for efficient incremental sync
- Push notifications where clients support it (RFC 7230 / Apple push), polling as fallback

### User discovery & onboarding
- Full auto-discovery via .well-known/caldav and .well-known/carddav (RFC 6764)
- App-specific passwords for CalDAV/CardDAV auth — user generates dedicated passwords in KMU Hub settings, individually revocable
- In-app setup wizard with step-by-step instructions per client, password generation, URL display, and connection testing
- Target clients for testing and setup instructions: Microsoft Outlook (desktop), Mozilla Thunderbird, macOS Calendar/Contacts

### Access control
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

</decisions>

<specifics>
## Specific Ideas

- VCard import/export already exists from Phase 10 — leverage for CardDAV data conversion
- Events already use standard RRULE fields — minimal mapping work for CalDAV
- EventException model already supports single-occurrence modifications
- Deutschland-first: setup wizard instructions in German, client names localized where applicable

</specifics>

<deferred>
## Deferred Ideas

- Mobile client support (iOS Calendar/Contacts, DAVx5 for Android) — future enhancement
- Linux client support (GNOME Calendar, Evolution) — future enhancement
- WebDAV for file access — Phase 15 originally separate, now covered by download/upload flow
- SRV DNS records for enterprise auto-discovery — nice-to-have after .well-known works

</deferred>

---

*Phase: 15-caldav-carddav-integration*
*Context gathered: 2026-02-20*

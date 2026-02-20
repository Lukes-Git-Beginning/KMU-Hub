# Phase 10: Email Integration - Context

**Gathered:** 2026-02-16
**Status:** Ready for planning

<domain>
## Phase Boundary

Full email client within the Hub — IMAP sync, SMTP send, threaded conversations, CRM auto-linking, HTML signatures with Impressum, and contact import/export with shared/personal visibility. Design integration (Plans 10-01 to 10-03) is already complete; this context covers the email backend and UI (Plans 10-04 to 10-07).

</domain>

<decisions>
## Implementation Decisions

### Inbox experience
- Three-column layout: folder sidebar + message list + reading pane side-by-side (Outlook-style)
- Threaded conversations: related emails collapsed into one row showing latest message, expand to see full thread (Gmail-style)
- User-switchable density: both comfortable (sender, subject, 1-line preview, date, attachment icon) and compact (sender + subject + date on one row) modes, togglable in settings or via UI toggle
- Smart folder defaults: pin Inbox/Sent/Drafts/Trash at top, remaining IMAP folders in collapsible section below

### Compose & signatures
- Full rich text editor: bold, italic, lists, links, inline images, tables, font size/color (Outlook-level capability)
- Visual signature builder: WYSIWYG editor with structured fields (name, title, phone, logo, Impressum block) for non-technical users
- Local-first drafts: draft stored locally while composing, saved to IMAP Drafts folder only when compose window is closed
- Compose mode: default inline (opens in reading pane area), with pop-out button to expand to full modal overlay

### CRM auto-linking
- Automatic + indicator: emails matching a CRM contact's email address are auto-linked silently, with a subtle CRM badge shown on linked emails
- Unknown senders: show a subtle "Add to CRM?" prompt on emails from addresses not matching any contact
- Email history on contact profiles: emails appear in the existing activity timeline alongside calls, notes, and deal changes (no separate tab)
- Deal linking: auto-link to contacts by email match, and if that contact has open deals, show a "Link to deal?" option for manual deal association

### Contact import/export
- Step-by-step import wizard: upload → preview data → map fields → handle duplicates → confirm → import (guided multi-step flow)
- Duplicate handling: auto-merge by email match — if email matches existing contact, auto-merge new fields into existing record (no duplicates created)
- Visibility model: import wizard asks shared vs personal for the batch; manually created contacts default to shared; users can mark specific contacts as personal (only they see them); admin can override visibility
- Export: user picks which fields to export (name, email, phone, company, etc.) and chooses CSV or vCard format

### Claude's Discretion
- IMAP sync frequency and strategy (polling interval, IDLE push support, initial sync depth)
- Rich text editor library choice (TipTap, Slate, etc.)
- Thread reconstruction algorithm details (References/In-Reply-To header parsing)
- Attachment storage strategy in MinIO
- Email search implementation approach
- Read/unread bidirectional sync mechanism

</decisions>

<specifics>
## Specific Ideas

- Three-column layout should feel like Outlook desktop — familiar to business users
- Thread grouping like Gmail — collapsed conversations in list, expanded in reading pane
- Signature builder should be accessible to non-technical office workers — structured fields, not raw HTML
- CRM badge on emails should be subtle, not intrusive — a small icon or colored dot, not a banner
- "Add to CRM?" prompt for unknown senders should be dismissable and not block email reading
- Deal linking is a secondary action — contact linking happens automatically, deal linking is offered as suggestion

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 10-email-integration*
*Context gathered: 2026-02-16*

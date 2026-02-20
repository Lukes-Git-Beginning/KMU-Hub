# Phase 17: Integration - Teams & Slack - Context

**Gathered:** 2026-02-20
**Status:** Ready for planning

<domain>
## Phase Boundary

KMU Hub notifications flow outbound to Microsoft Teams and Slack channels via rich cards, and users can perform context-dependent actions (acknowledge, reply, approve/reject) from within Teams/Slack back into KMU Hub. Admin configures channel mappings with module-level filtering. Both platforms can be active simultaneously.

</domain>

<decisions>
## Implementation Decisions

### Notification scope & formatting
- All notification types are forwardable — admin controls which modules forward via module-level toggles per channel
- Rich cards using native platform formats: Teams Adaptive Cards, Slack Block Kit
- Cards include title, body, colored sidebar, action buttons, and module icon
- Module-level toggle per channel mapping (CRM on/off, HR on/off, Finance on/off, etc.) — not individual notification categories

### Channel routing & mapping
- Multi-channel mapping: admin maps module groups to different Teams/Slack channels (e.g., CRM → #sales, HR → #hr)
- Both Teams AND Slack can be configured simultaneously for the same org
- "Send test notification" button when setting up a channel mapping to verify webhook connectivity
- Most-specific-wins routing: if a notification matches multiple channel mappings, only the most specific one fires (avoids duplicates)

### Inbound interactions
- Full context-dependent action set on notification cards:
  - **Acknowledge** — marks notification as read in KMU Hub
  - **Reply** — opens thread, text sent back as comment/response in KMU Hub
  - **Approve/Reject** — for workflow notifications (leave requests, invoice approvals, etc.)
- Buttons shown are context-dependent based on notification type
- Explicit account linking: user runs a `/link` command in Teams/Slack once to connect their KMU Hub account (more reliable than email matching)
- Card updates in-place after action (e.g., "Approved by Max" with checkmark) — no reply clutter
- Unlinked users get an ephemeral (private) prompt to link their account when they try to interact

### Setup & configuration flow
- **Teams:** Full Teams App via Bot Framework (Azure AD app registration) — handles both outbound posting and inbound interactions in one setup
- **Slack:** Slack App with OAuth install flow — bot posting, interactive messages, slash commands
- Configuration lives in **Admin Settings > Integrations** — dedicated section with cards for each platform (scales for future Bexio/Abacus integrations)
- Step-by-step wizard for setup: 1) Choose platform, 2) Authenticate/install app, 3) Select channels + module mappings, 4) Send test notification

### Claude's Discretion
- Whether forwarded notifications include original KMU Hub user identity (name/avatar) vs bot-only identity — decide based on DSGVO considerations and platform conventions
- Card color scheme and icon mapping per module
- Exact Adaptive Card / Block Kit template structure
- Error handling for failed webhook deliveries (retry strategy, admin alerts)
- Rate limiting on outbound notifications to avoid Teams/Slack throttling

</decisions>

<specifics>
## Specific Ideas

- Wizard flow mirrors the pattern from Phase 16 automation wizard (4-step guided setup)
- Admin Settings > Integrations page should be a reusable pattern for Phases 18-19 (Bexio, Abacus) — card-based layout with platform logos, connection status, and "Configure" button
- In-place card updates after interactions keep the channel clean (no reply noise)

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 17-integration-teams-slack*
*Context gathered: 2026-02-20*

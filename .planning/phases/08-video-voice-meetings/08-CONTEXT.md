# Phase 8: Video, Voice & Meetings - Context

**Gathered:** 2026-02-10
**Status:** Ready for planning

<domain>
## Phase Boundary

LiveKit-powered video/voice calls, screen sharing, recording with DSGVO consent, meeting management with agenda and action items, emoji reactions on chat messages and during calls, and presence/online status system. Replaces Zoom/Teams within the Hub.

Darien's `design/brainstorm` branch contains 5 complete UI components (MeetingsPage, MeetingFormDialog, MeetingDetailPanel, MeetingRoomView, CallOverlay) with a Zustand mock store. Backend builds to match his data model; frontend plans wire his components to real APIs via TanStack Query hooks.

</domain>

<decisions>
## Implementation Decisions

### Recording & DSGVO Consent

- **Who can record:** Role-based permission -- Meeting Host (organizer) and roles with recording rights (admin/manager). Controlled via existing RBAC system, configurable per role.
- **Consent flow:** When someone presses "Record", every participant gets a consent popup. Each participant must actively accept or decline.
- **Declined participants:** Person stays in the meeting normally, but their audio/video stream is replaced in the recording with a placeholder (avatar + "Aufnahme abgelehnt" label). Recording continues for consenting participants.
- **Storage:** Recordings stored in MinIO with automatic deletion after configurable retention period (default: 30 days).
- **Retention policy:** Admin-configurable globally (e.g. 30-365 days). System runs cleanup job to delete expired recordings.
- **Access control:** Configurable per recording. Organizer can set who has access: meeting participants, project team members, or specific individual users.

### Presence System

- **Status types:** Full set (8+): Online, Away, Busy, DND (Do Not Disturb), In a Call, In a Meeting, Presenting, Offline + Custom Status with emoji + free text.
- **Custom status:** User chooses an emoji + types free text (e.g. "Im Urlaub bis 15.3"). Optional expiration time so status auto-clears.
- **Auto-away:** Enabled by default. After configurable inactivity timeout (admin sets default, e.g. 10min), status switches to Away. User can override manually (manual takes priority).
- **Auto-away timeout:** Admin-configurable globally with sensible default.
- **Auto-call status:** When user joins a LiveKit call/meeting, presence automatically switches to "In a Call" / "In a Meeting". On leave, reverts to previous status.
- **DND behavior:** Real suppression -- desktop push notifications are suppressed when DND is active. Badge counters still update. Urgent/priority notifications can optionally break through DND.
- **Visibility:** Communication contexts only -- Chat member lists, Meeting participants, Team page. Not in CRM contacts or general sidebar.
- **Persistence:** Redis-based with heartbeat. Client sends heartbeat every 30s. No heartbeat for 60s = Offline. Tolerates short network interruptions.

### Emoji Reactions

- **Emoji set:** Twemoji (Twitter open-source) for consistent cross-platform rendering. ~3MB bundle.
- **Scope:** Chat messages (channels + DMs) + Meeting-internal chat + Live video call reactions (floating emojis).
- **Self-reactions:** Allowed -- user can react to their own messages.
- **UI pattern:** 6-8 Quick-Reaction buttons (thumbs up, heart, laugh, party, surprised, thumbs down) + full emoji picker for the rest.
- **Toggle behavior:** Clicking an already-set reaction removes it (Slack/Discord pattern).
- **Call reactions:** Fly-up animation -- emoji flies from bottom of screen and fades after 2-3 seconds (Zoom/Instagram Live style).
- **Data model:** Toggle semantics -- reaction table with (message_id, user_id, emoji) unique constraint. Batch summary query for reaction counts per message.

### Meeting Lifecycle

- **Meeting notes:** Only organizer/host can write notes during meeting. Other participants see notes as read-only in real-time.
- **Agenda:** Structured checklist created by organizer before the meeting. Each agenda item has title + optional time allocation (e.g. "Statusupdate - 10min"). During meeting, items can be checked off. Timer shows if meeting is on track.
- **Action items:** Collected during meeting (by host). After meeting ends, all action items are presented as a batch list. User selects which ones to convert to PM tasks (with meeting link).
- **Action item reminders:** Post-meeting page shown immediately when last participant leaves. If not processed, reminder notification sent 24h later.
- **Post-meeting summary:** For now, structured template auto-generated (participants, duration, checked agenda items, notes, action items, next meeting). Data model prepared for future AI transcription + summary generation.
- **Follow-up meetings:** Host can set a follow-up date after meeting. System creates a follow-up meeting with open (unconverted) action items carried over. Separate from recurring meetings.
- **Meeting states:** scheduled -> live -> completed (or cancelled). Live state set when first participant joins via LiveKit.

### Call Signaling (from Darien's UI)

- **Incoming 1:1 calls:** Overlay with pulsing avatar animation. User can accept or decline. Call can be minimized to floating bubble while working.
- **Group calls:** Users join via "Beitreten" button on meeting card (no ringing for group calls).
- **Video room:** Dark theme (gray-900), responsive grid (1-4 columns based on participant count), active speaker highlight.
- **Toolbar:** Mic, Video, Screen Share, Hand Raise, Whiteboard, Chat toggle, Participants toggle, Leave (red).
- **Pre-Join:** Darien's UI has no pre-join screen. Decision deferred to Claude's discretion based on UX best practices.

### Claude's Discretion

- Pre-Join screen implementation (whether to add for scheduled meetings vs. spontaneous calls)
- Exact heartbeat/timeout intervals for presence
- Quick-reaction emoji selection (which 6-8 emojis as defaults)
- Meeting notes editor component choice (rich text vs. markdown)
- Call ringing timeout duration and missed call notification
- Screen sharing UI during active share
- Whiteboard integration approach (embedded vs. external launch)

</decisions>

<specifics>
## Specific Ideas

- Darien's design uses a "desk metaphor" with warm beige/teal color palette (OKLCH dark mode). Meeting room is dark (gray-900) for reduced eye strain during calls.
- Call overlay minimizes to a small bubble (bottom-right) showing initials + name + elapsed time -- user continues working while on call.
- Live meetings show pulsing red dot in meeting list with "LIVE JETZT" header.
- Past meetings appear with opacity-70 for visual de-emphasis.
- Meeting form has expandable "Weitere Optionen" section (recurrence, reminder, description, files) -- keeps simple meetings simple.
- Darien's Backend Requirements Audit identifies ~9 endpoints needed for meetings, ~159 total endpoints across all modules. Pattern: replace Zustand mocks with TanStack Query hooks.

</specifics>

<deferred>
## Deferred Ideas

- **AI-generated meeting summaries** -- Data model prepared now, but AI transcription + summary generation deferred to future enhancement (requires speech-to-text integration).
- **Collaborative real-time notes** (CRDT/OT) -- Too complex for v1. Host-only notes with read-only sharing is sufficient. Could be upgraded in future phase.
- **Virtual drop-in rooms** -- Darien noted "Braucht Presence-System" in brainstorm. Concept of persistent team rooms users can drop into. Consider for future phase.
- **Whiteboard canvas** -- Button exists in Darien's UI but actual whiteboard implementation (drawing, shapes, collaboration) is a significant feature. External tool launch for v1, embedded whiteboard for future.

</deferred>

---

*Phase: 08-video-voice-meetings*
*Context gathered: 2026-02-10*

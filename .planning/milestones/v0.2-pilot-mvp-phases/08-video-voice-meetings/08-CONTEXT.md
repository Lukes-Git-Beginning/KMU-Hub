# Phase 8: Video, Voice & Meetings - Context

**Gathered:** 2026-02-11
**Status:** Ready for planning

<domain>
## Phase Boundary

LiveKit-powered video/voice calls, meeting management with lifecycle (schedule, lobby, notes, action items), DSGVO-compliant recording, emoji reactions on chat messages, and presence/online-status indicators. Replaces Zoom/Teams for the Hub's target users.

**Design integration:** Darien has built 5 UI components + Zustand store on `design/brainstorm` (~1,700 lines total):
- MeetingsPage.tsx (446 lines) - Meeting overview
- MeetingFormDialog.tsx (335 lines) - Schedule/edit meetings
- MeetingRoomView.tsx (275 lines) - Virtual meeting rooms
- MeetingDetailPanel.tsx (220 lines) - Meeting details
- CallOverlay.tsx (157 lines) - Video/audio call UI
- meetings.ts store (288 lines)

Strategy: Cherry-pick from design/brainstorm + wire to backend API hooks (same as Phase 7 calendar integration).

</domain>

<decisions>
## Implementation Decisions

### Call Experience & Controls
- **Video layout:** Gallery + Speaker view, umschaltbar. Default Gallery-Grid (alle gleich gross), Klick auf Teilnehmer oder aktiver Sprecher wechselt zu Speaker-View (1 gross + Rest als Thumbnails)
- **Floating call bar:** Kompakte schwebende Mini-Bar (oben oder unten) mit Mute/Hangup/Kamera-Toggle + Dauer-Timer wenn User wegnavigiert. Call laeuft im Hintergrund weiter
- **Incoming call:** Fullscreen-Overlay ueber dem ganzen Screen (wie Telefon-App) mit Avatar, Name, Annehmen/Ablehnen Buttons
- **Screen sharing:** Geteilter Screen ersetzt den Speaker-Bereich (Hauptbereich), Video-Thumbnails der Teilnehmer wandern in eine Seitenleiste

### Meeting Lifecycle
- **Pre-meeting lobby:** Kamera/Mikrofon-Preview + Check, geteilte Meeting-Dokumente und Agenda zum Durchlesen vor dem Start
- **Meeting notes:** Waehrend des Meetings gibt es ein Notiz-Panel. Nach Ende wird daraus automatisch ein Summary-Draft erstellt, Organisator reviewed und finalisiert
- **Action items -> Tasks:** Batch-Konvertierung nach dem Meeting: "Alle Action Items als Tasks erstellen" Button, erstellt Tasks in einem Rutsch im gewaehlten Projekt
- **Recurring meetings:** Beim Oeffnen eines wiederkehrenden Meetings zeigt ein "Letzte Notizen" Panel das Summary vom Vorgaenger-Meeting an

### DSGVO Recording Consent
- **Consent flow:** Alle Teilnehmer werden gefragt (Zustimmen/Ablehnen Popup). Teilnehmer die zustimmen werden voll aufgenommen (Video + Audio). Teilnehmer die ablehnen werden in der Aufnahme geblurrt (Video) und gemutet (Audio) -- selektiver Consent
- **Speicherung:** Aufnahme erscheint im Meeting-Detail-Panel UND im zentralen Datei-Manager (Phase 11) unter einem Meeting-Ordner. Nur Meeting-Teilnehmer haben Zugriff
- **Aufbewahrung:** 30 Tage Retention, danach automatische Loeschung. DSGVO-konform als Prioritaet

### Presence & Online-Status
- **Erkennung:** Automatisch basierend auf Aktivitaet + manuell ueberschreibbar. Away-Timeout ist admin-konfigurierbar (nicht hardcoded) -- moeglichst flexibel fuer den Kunden
- **Status-Stufen:** 5 Stufen: Online (gruen), Abwesend (gelb), Nicht stoeren (rot), Im Anruf (lila/blau, automatisch gesetzt), Offline (grau)
- **"Im Anruf" Status:** Wird automatisch gesetzt wenn User in einem aktiven Call ist
- **Sichtbarkeit:** Presence-Dots nur in Chat-Teilnehmerlisten/DMs und Team-Uebersicht -- nicht in CRM oder Kalender

### Claude's Discretion
- LiveKit SDK integration patterns (room management, token generation, track handling)
- Exact floating bar positioning and animation
- Summary auto-draft algorithm from meeting notes
- Presence heartbeat interval and Redis data structure
- Recording blur/mute technical approach (LiveKit Egress capabilities vs post-processing)
- Emoji reaction picker component choice and animation style

</decisions>

<specifics>
## Specific Ideas

- Recording consent with selective blur/mute is a premium DSGVO feature -- if LiveKit Egress doesn't support per-participant blur natively, research composite recording or post-processing approaches
- Admin-configurable away timeout means a system settings entry (not per-user) -- admin sets the rule for the entire organization
- Darien's existing CallOverlay.tsx (157 lines) likely covers the basic call controls -- check if it includes gallery/speaker toggle before building from scratch
- Meeting notes panel during call should be non-intrusive -- possibly a side panel that doesn't reduce video area

</specifics>

<deferred>
## Deferred Ideas

- Whiteboard during calls (D8 in Darien's design roadmap, large scope) -- future phase
- Meeting transcription (AI-powered) -- future phase or v2
- Virtual background / background blur for user's own camera -- future enhancement
- Meeting room booking from meeting form (already exists in Calendar Phase 7, just needs linking)

</deferred>

---

*Phase: 08-video-voice-meetings*
*Context gathered: 2026-02-11*

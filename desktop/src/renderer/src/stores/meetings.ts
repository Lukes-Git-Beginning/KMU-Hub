import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { DEMO_MODE } from '@/mocks/demo-mode'

export interface MeetingParticipant {
  id: string
  name: string
  initials: string
}

export interface MeetingFile {
  id: string
  name: string
  size: string
}

export interface AgendaItem {
  id: string
  text: string
  done: boolean
}

export interface Meeting {
  id: string
  title: string
  status: 'live' | 'scheduled' | 'past' | 'cancelled'
  project: string
  color: string
  date: string
  startTime: string
  duration: number
  room: string
  isVideoCall: boolean
  recurrence: 'none' | 'daily' | 'weekly' | 'monthly'
  reminder: '15min' | '30min' | '1h' | 'none'
  description: string
  participants: MeetingParticipant[]
  organizerId: string
  agenda: AgendaItem[]
  notes: string
  files: MeetingFile[]
  whiteboardLink: string
  projectLink: string
  calendarEventId?: string
  invitationsSent?: boolean
  /** CRM contact linked to this meeting (nullable). */
  contact_id?: string | null
  /** CRM deal linked to this meeting (nullable). */
  deal_id?: string | null
}

// ---------------------------------------------------------------------------
// Video meeting state (for in-meeting UI)
// ---------------------------------------------------------------------------

export type MeetingLayout = 'grid' | 'speaker' | 'sidebar'

export interface VideoMeetingState {
  isInMeeting: boolean
  audioEnabled: boolean
  videoEnabled: boolean
  screenSharing: boolean
  activeSpeakerId: string | null
  layout: MeetingLayout
  handRaised: boolean
}

// ---------------------------------------------------------------------------
// Call history
// ---------------------------------------------------------------------------

export interface CallHistoryEntry {
  id: string
  contactName: string
  contactInitials: string
  type: 'video' | 'audio'
  direction: 'incoming' | 'outgoing' | 'missed'
  date: string
  startTime: string
  duration: number // minutes, 0 for missed
  /** Optional enrichment for the call-history detail view (all nullable). */
  contactCompany?: string
  contactPhone?: string
  /** Short subject/topic of the call. */
  topic?: string
  /** Free-text note captured after the call. */
  notes?: string
  /** Whether a recording exists for this call. */
  hasRecording?: boolean
  /** Playback length of the recording in seconds. */
  recordingDuration?: number
  /** Additional participants for group calls (1:1 calls leave this empty). */
  participants?: MeetingParticipant[]
}

const mockCallHistory: CallHistoryEntry[] = [
  {
    id: 'ch1', contactName: 'Anna Müller', contactInitials: 'AM', type: 'video', direction: 'outgoing',
    date: '2026-02-22', startTime: '09:15', duration: 23,
    contactCompany: 'Müller Consulting', contactPhone: '+49 151 23456789',
    topic: 'Angebotsbesprechung Website-Relaunch',
    notes: 'Kundin möchte zusätzlich ein SEO-Paket. Angebot bis Freitag nachreichen, Follow-up nächste Woche.',
    hasRecording: true, recordingDuration: 1380,
  },
  {
    id: 'ch2', contactName: 'Weber GmbH', contactInitials: 'WG', type: 'audio', direction: 'incoming',
    date: '2026-02-22', startTime: '08:42', duration: 8,
    contactCompany: 'Weber GmbH', contactPhone: '+49 30 9876543',
    topic: 'Rückfrage zur Rechnung 2026-0042',
    notes: 'Zahlungsziel einvernehmlich auf 30 Tage verlängert.',
  },
  {
    id: 'ch3', contactName: 'Peter Koch', contactInitials: 'PK', type: 'video', direction: 'missed',
    date: '2026-02-21', startTime: '17:30', duration: 0,
    contactPhone: '+49 160 1112223',
    topic: 'Verpasster Videoanruf',
  },
  {
    id: 'ch4', contactName: 'Sarah Klein', contactInitials: 'SK', type: 'video', direction: 'incoming',
    date: '2026-02-21', startTime: '14:00', duration: 45,
    contactCompany: 'Zentria UG', contactPhone: '+49 172 4455667',
    topic: 'Design Review Mobile App',
    notes: 'Onboarding-Flow abgenommen. Dashboard-Layout wird bis nächste Woche überarbeitet.',
    hasRecording: true, recordingDuration: 2700,
    participants: [
      { id: 'p3', name: 'Sarah Klein', initials: 'SK' },
      { id: 'p4', name: 'Lisa Schmidt', initials: 'LS' },
      { id: 'p2', name: 'Michael Berg', initials: 'MB' },
    ],
  },
  {
    id: 'ch5', contactName: 'Lisa Schmidt', contactInitials: 'LS', type: 'audio', direction: 'outgoing',
    date: '2026-02-21', startTime: '11:20', duration: 12,
    contactPhone: '+49 151 7788990',
    topic: 'Abstimmung Icon-Set',
  },
  {
    id: 'ch6', contactName: 'Thomas Weber', contactInitials: 'TW', type: 'video', direction: 'outgoing',
    date: '2026-02-20', startTime: '16:00', duration: 31,
    contactCompany: 'Weber GmbH', contactPhone: '+49 30 9876540',
    topic: 'Budget-Planung Q2',
    notes: 'Rahmenbudget bestätigt. Detailplanung folgt im nächsten Termin.',
    hasRecording: true, recordingDuration: 1860,
  },
  {
    id: 'ch7', contactName: 'Jonas Diaz', contactInitials: 'JD', type: 'audio', direction: 'missed',
    date: '2026-02-20', startTime: '10:05', duration: 0,
    contactPhone: '+49 176 3322110',
    topic: 'Verpasster Anruf',
  },
  {
    id: 'ch8', contactName: 'Meier AG', contactInitials: 'MA', type: 'video', direction: 'incoming',
    date: '2026-02-19', startTime: '13:30', duration: 58,
    contactCompany: 'Meier AG', contactPhone: '+49 89 5566778',
    topic: 'Kundenpräsentation CRM-Integration',
    notes: 'Live-Demo lief sehr gut. Vertragsverlängerung um 12 Monate in Aussicht — Angebot vorbereiten.',
    hasRecording: true, recordingDuration: 3480,
    participants: [
      { id: 'p1', name: 'Anna Müller', initials: 'AM' },
      { id: 'p5', name: 'Peter Koch', initials: 'PK' },
      { id: 'p2', name: 'Michael Berg', initials: 'MB' },
    ],
  },
]

// ---------------------------------------------------------------------------
// Store interface
// ---------------------------------------------------------------------------

interface MeetingsState {
  meetings: Meeting[]
  callHistory: CallHistoryEntry[]
  activeMeetingId: string | null
  activeCallContactId: string | null
  activeCallContactName: string | null

  // Video meeting state
  videoMeeting: VideoMeetingState

  addMeeting: (meeting: Omit<Meeting, 'id'>) => void
  updateMeeting: (id: string, updates: Partial<Meeting>) => void
  deleteMeeting: (id: string) => void
  cancelMeeting: (id: string) => void
  duplicateMeeting: (id: string) => void
  toggleAgendaItem: (meetingId: string, agendaItemId: string) => void
  addAgendaItem: (meetingId: string, text: string) => void
  removeAgendaItem: (meetingId: string, agendaItemId: string) => void
  reorderAgendaItem: (meetingId: string, agendaItemId: string, direction: 'up' | 'down') => void
  updateNotes: (meetingId: string, notes: string) => void
  setActiveMeeting: (id: string | null) => void
  startCall: (contactId: string, contactName: string) => void
  endCall: () => void

  // Video meeting actions
  joinMeeting: (meetingId: string) => void
  leaveMeeting: () => void
  toggleAudio: () => void
  toggleVideo: () => void
  toggleScreenShare: () => void
  toggleHandRaise: () => void
  setLayout: (layout: MeetingLayout) => void
  setActiveSpeaker: (userId: string | null) => void
}

const mockMeetings: Meeting[] = [
  {
    id: 'm1',
    title: 'Sprint Planning Q1',
    status: 'live',
    project: 'Website Relaunch',
    color: '#3B82F6',
    date: '2026-02-09',
    startTime: '10:00',
    duration: 45,
    room: 'Konferenzraum A',
    isVideoCall: true,
    recurrence: 'weekly',
    reminder: '15min',
    description: 'Wöchentliches Sprint Planning für das Relaunch-Projekt. Besprechung der Tasks für die kommende Woche.',
    participants: [
      { id: 'p1', name: 'Anna Müller', initials: 'AM' },
      { id: 'p2', name: 'Michael Berg', initials: 'MB' },
      { id: 'p3', name: 'Sarah Klein', initials: 'SK' },
    ],
    organizerId: 'p1',
    agenda: [
      { id: 'a1', text: 'Review offener Tasks aus letztem Sprint', done: true },
      { id: 'a2', text: 'Neue User Stories priorisieren', done: false },
      { id: 'a3', text: 'Kapazitätsplanung Team', done: false },
      { id: 'a4', text: 'Blocker und Risiken besprechen', done: false },
    ],
    notes: 'Sprint 12 war erfolgreich — 18/20 Story Points abgeschlossen.\nFokus diese Woche: Landing Page Redesign + Performance-Optimierung.',
    files: [{ id: 'f1', name: 'Sprint-Backlog.xlsx', size: '245 KB' }],
    whiteboardLink: '',
    projectLink: 'website-relaunch',
    calendarEventId: 'cal-m1',
    invitationsSent: true,
  },
  {
    id: 'm2',
    title: 'Design Review',
    status: 'live',
    project: 'Mobile App',
    color: '#8B5CF6',
    date: '2026-02-09',
    startTime: '10:30',
    duration: 30,
    room: 'Huddle Space',
    isVideoCall: true,
    recurrence: 'none',
    reminder: '15min',
    description: 'Review der neuen Design-Mockups für die Mobile App. Feedback und Freigabe.',
    participants: [
      { id: 'p3', name: 'Sarah Klein', initials: 'SK' },
      { id: 'p4', name: 'Lisa Schmidt', initials: 'LS' },
    ],
    organizerId: 'p3',
    agenda: [
      { id: 'a5', text: 'Onboarding-Flow Mockups präsentieren', done: true },
      { id: 'a6', text: 'Feedback zum Dashboard-Layout', done: false },
      { id: 'a7', text: 'Farb- und Icon-Konsistenz prüfen', done: false },
    ],
    notes: '',
    files: [{ id: 'f2', name: 'Mockups-v3.fig', size: '12 MB' }],
    whiteboardLink: '',
    projectLink: 'mobile-app',
    invitationsSent: false,
  },
  {
    id: 'm3',
    title: 'Kundenpräsentation Meier AG',
    status: 'scheduled',
    project: 'CRM Integration',
    color: '#10B981',
    date: '2026-02-09',
    startTime: '14:00',
    duration: 60,
    room: 'Konferenzraum B',
    isVideoCall: true,
    recurrence: 'none',
    reminder: '30min',
    description: 'Fortschrittspräsentation für Meier AG. Aktueller Stand der CRM-Integration und nächste Schritte.',
    participants: [
      { id: 'p1', name: 'Anna Müller', initials: 'AM' },
      { id: 'p5', name: 'Peter Koch', initials: 'PK' },
      { id: 'p2', name: 'Michael Berg', initials: 'MB' },
    ],
    organizerId: 'p1',
    agenda: [
      { id: 'a8', text: 'Aktuelle Meilensteine vorstellen', done: false },
      { id: 'a9', text: 'Live-Demo der Integration', done: false },
      { id: 'a10', text: 'Zeitplan Q2 besprechen', done: false },
      { id: 'a11', text: 'Offene Fragen klären', done: false },
    ],
    notes: '',
    files: [
      { id: 'f3', name: 'Präsentation-CRM.pptx', size: '8.4 MB' },
      { id: 'f4', name: 'Zeitplan-Q1.pdf', size: '156 KB' },
    ],
    whiteboardLink: '',
    projectLink: 'crm-integration',
    calendarEventId: 'cal-m3',
    invitationsSent: true,
  },
  {
    id: 'm4',
    title: 'Team Standup',
    status: 'scheduled',
    project: 'Allgemein',
    color: '#6B7280',
    date: '2026-02-10',
    startTime: '09:00',
    duration: 15,
    room: 'Remote',
    isVideoCall: true,
    recurrence: 'daily',
    reminder: '15min',
    description: 'Tägliches Team Standup. Kurzer Austausch über aktuelle Aufgaben und Blocker.',
    participants: [
      { id: 'p1', name: 'Anna Müller', initials: 'AM' },
      { id: 'p2', name: 'Michael Berg', initials: 'MB' },
      { id: 'p3', name: 'Sarah Klein', initials: 'SK' },
      { id: 'p6', name: 'Jonas Diaz', initials: 'JD' },
      { id: 'p4', name: 'Lisa Schmidt', initials: 'LS' },
    ],
    organizerId: 'p1',
    agenda: [
      { id: 'a12', text: 'Was habe ich gestern gemacht?', done: false },
      { id: 'a13', text: 'Was mache ich heute?', done: false },
      { id: 'a14', text: 'Gibt es Blocker?', done: false },
    ],
    notes: '',
    files: [],
    whiteboardLink: '',
    projectLink: '',
  },
  {
    id: 'm5',
    title: 'Retrospektive Sprint 12',
    status: 'scheduled',
    project: 'Website Relaunch',
    color: '#3B82F6',
    date: '2026-02-11',
    startTime: '15:00',
    duration: 45,
    room: 'Konferenzraum A',
    isVideoCall: false,
    recurrence: 'none',
    reminder: '30min',
    description: 'Rückblick auf Sprint 12. Was lief gut, was können wir verbessern?',
    participants: [
      { id: 'p1', name: 'Anna Müller', initials: 'AM' },
      { id: 'p2', name: 'Michael Berg', initials: 'MB' },
    ],
    organizerId: 'p2',
    agenda: [
      { id: 'a15', text: 'Was lief gut?', done: false },
      { id: 'a16', text: 'Was können wir verbessern?', done: false },
      { id: 'a17', text: 'Maßnahmen für nächsten Sprint', done: false },
    ],
    notes: '',
    files: [],
    whiteboardLink: '',
    projectLink: 'website-relaunch',
  },
  {
    id: 'm6',
    title: 'Budget-Planung 2026',
    status: 'scheduled',
    project: 'Finanzen',
    color: '#F59E0B',
    date: '2026-02-12',
    startTime: '10:00',
    duration: 90,
    room: 'Konferenzraum B',
    isVideoCall: false,
    recurrence: 'none',
    reminder: '1h',
    description: 'Jahresbudget-Planung für 2026. Kosten, Investitionen, Personalplanung.',
    participants: [
      { id: 'p1', name: 'Anna Müller', initials: 'AM' },
      { id: 'p5', name: 'Peter Koch', initials: 'PK' },
      { id: 'p7', name: 'Thomas Weber', initials: 'TW' },
    ],
    organizerId: 'p5',
    agenda: [
      { id: 'a18', text: 'Jahresübersicht 2025 präsentieren', done: false },
      { id: 'a19', text: 'Investitionspläne 2026', done: false },
      { id: 'a20', text: 'Personalkosten und Neueinstellungen', done: false },
      { id: 'a21', text: 'IT-Infrastruktur Budget', done: false },
      { id: 'a22', text: 'Freigabe und nächste Schritte', done: false },
    ],
    notes: '',
    files: [{ id: 'f5', name: 'Budget-Vorlage-2026.xlsx', size: '1.2 MB' }],
    whiteboardLink: '',
    projectLink: '',
  },
  {
    id: 'm7',
    title: 'API Review',
    status: 'past',
    project: 'Mobile App',
    color: '#8B5CF6',
    date: '2026-02-07',
    startTime: '11:00',
    duration: 30,
    room: 'Remote',
    isVideoCall: true,
    recurrence: 'none',
    reminder: '15min',
    description: 'Review der API-Endpoints für die Mobile App. Authentifizierung und Datenmodell.',
    participants: [
      { id: 'p6', name: 'Jonas Diaz', initials: 'JD' },
      { id: 'p5', name: 'Peter Koch', initials: 'PK' },
    ],
    organizerId: 'p6',
    agenda: [
      { id: 'a23', text: 'Auth-Endpoints durchgehen', done: true },
      { id: 'a24', text: 'Datenmodell validieren', done: true },
      { id: 'a25', text: 'Rate Limiting besprechen', done: true },
    ],
    notes: 'Alle Endpoints abgenommen. Rate Limiting wird auf 100 req/min gesetzt.\nJonas übernimmt die Dokumentation bis Freitag.',
    files: [{ id: 'f6', name: 'API-Spezifikation.yaml', size: '45 KB' }],
    whiteboardLink: '',
    projectLink: 'mobile-app',
  },
  {
    id: 'm8',
    title: 'Security Briefing',
    status: 'past',
    project: 'Security Audit',
    color: '#EF4444',
    date: '2026-02-06',
    startTime: '14:00',
    duration: 60,
    room: 'Konferenzraum A',
    isVideoCall: false,
    recurrence: 'none',
    reminder: '30min',
    description: 'Besprechung der Sicherheitsaudit-Ergebnisse. Maßnahmenplan und Priorisierung.',
    participants: [
      { id: 'p5', name: 'Peter Koch', initials: 'PK' },
      { id: 'p6', name: 'Jonas Diaz', initials: 'JD' },
      { id: 'p1', name: 'Anna Müller', initials: 'AM' },
    ],
    organizerId: 'p5',
    agenda: [
      { id: 'a26', text: 'Audit-Ergebnisse präsentieren', done: true },
      { id: 'a27', text: 'Kritische Findings priorisieren', done: true },
      { id: 'a28', text: 'Maßnahmenplan erstellen', done: true },
      { id: 'a29', text: 'Verantwortlichkeiten zuteilen', done: true },
    ],
    notes: '3 kritische Findings identifiziert:\n1. SQL Injection in Legacy-Modul → Patch bis 15.02.\n2. Fehlende 2FA für Admin-Accounts → Sofort umsetzen\n3. Veraltete Dependencies → Sprint 13 einplanen',
    files: [{ id: 'f7', name: 'Audit-Report-2026.pdf', size: '2.3 MB' }],
    whiteboardLink: '',
    projectLink: '',
  },
]

let nextId = 9

export const useMeetingsStore = create<MeetingsState>()(
  persist(
    (set, get) => ({
      // Seed sample data only in demo mode; production starts empty so the
      // real backend meetings are the single source of truth.
      meetings: DEMO_MODE ? mockMeetings : [],
      callHistory: DEMO_MODE ? mockCallHistory : [],
      activeMeetingId: null,
      activeCallContactId: null,
      activeCallContactName: null,

      videoMeeting: {
        isInMeeting: false,
        audioEnabled: true,
        videoEnabled: true,
        screenSharing: false,
        activeSpeakerId: null,
        layout: 'grid' as MeetingLayout,
        handRaised: false,
      },

      addMeeting: (meeting) =>
        set((state) => ({
          meetings: [{ ...meeting, id: `m${nextId++}` }, ...state.meetings],
        })),

      updateMeeting: (id, updates) =>
        set((state) => ({
          meetings: state.meetings.map((m) =>
            m.id === id ? { ...m, ...updates } : m
          ),
        })),

      deleteMeeting: (id) =>
        set((state) => ({
          meetings: state.meetings.filter((m) => m.id !== id),
          activeMeetingId: state.activeMeetingId === id ? null : state.activeMeetingId,
        })),

      cancelMeeting: (id) =>
        set((state) => ({
          meetings: state.meetings.map((m) =>
            m.id === id ? { ...m, status: 'cancelled' as const } : m
          ),
        })),

      duplicateMeeting: (id) => {
        const original = get().meetings.find((m) => m.id === id)
        if (!original) return
        const duplicate: Meeting = {
          ...original,
          id: `m${nextId++}`,
          title: `${original.title} (Kopie)`,
          status: 'scheduled',
        }
        set((state) => ({ meetings: [duplicate, ...state.meetings] }))
      },

      toggleAgendaItem: (meetingId, agendaItemId) =>
        set((state) => ({
          meetings: state.meetings.map((m) =>
            m.id === meetingId
              ? { ...m, agenda: m.agenda.map((a) => a.id === agendaItemId ? { ...a, done: !a.done } : a) }
              : m
          ),
        })),

      addAgendaItem: (meetingId, text) =>
        set((state) => ({
          meetings: state.meetings.map((m) =>
            m.id === meetingId
              ? { ...m, agenda: [...m.agenda, { id: `a${Date.now()}`, text, done: false }] }
              : m
          ),
        })),

      removeAgendaItem: (meetingId, agendaItemId) =>
        set((state) => ({
          meetings: state.meetings.map((m) =>
            m.id === meetingId
              ? { ...m, agenda: m.agenda.filter((a) => a.id !== agendaItemId) }
              : m
          ),
        })),

      reorderAgendaItem: (meetingId, agendaItemId, direction) =>
        set((state) => ({
          meetings: state.meetings.map((m) => {
            if (m.id !== meetingId) return m
            const idx = m.agenda.findIndex((a) => a.id === agendaItemId)
            if (idx === -1) return m
            const swapIdx = direction === 'up' ? idx - 1 : idx + 1
            if (swapIdx < 0 || swapIdx >= m.agenda.length) return m
            const newAgenda = [...m.agenda]
            ;[newAgenda[idx], newAgenda[swapIdx]] = [newAgenda[swapIdx], newAgenda[idx]]
            return { ...m, agenda: newAgenda }
          }),
        })),

      updateNotes: (meetingId, notes) =>
        set((state) => ({
          meetings: state.meetings.map((m) =>
            m.id === meetingId ? { ...m, notes } : m
          ),
        })),

      setActiveMeeting: (id) => set({ activeMeetingId: id }),

      startCall: (contactId, contactName) =>
        set({ activeCallContactId: contactId, activeCallContactName: contactName }),

      endCall: () =>
        set({ activeCallContactId: null, activeCallContactName: null }),

      // -- Video meeting actions --

      joinMeeting: (meetingId) =>
        set({
          activeMeetingId: meetingId,
          videoMeeting: {
            isInMeeting: true,
            audioEnabled: true,
            videoEnabled: true,
            screenSharing: false,
            activeSpeakerId: null,
            layout: 'grid',
            handRaised: false,
          },
        }),

      leaveMeeting: () =>
        set((state) => ({
          activeMeetingId: null,
          videoMeeting: {
            ...state.videoMeeting,
            isInMeeting: false,
            screenSharing: false,
            handRaised: false,
          },
        })),

      toggleAudio: () =>
        set((state) => ({
          videoMeeting: { ...state.videoMeeting, audioEnabled: !state.videoMeeting.audioEnabled },
        })),

      toggleVideo: () =>
        set((state) => ({
          videoMeeting: { ...state.videoMeeting, videoEnabled: !state.videoMeeting.videoEnabled },
        })),

      toggleScreenShare: () =>
        set((state) => ({
          videoMeeting: { ...state.videoMeeting, screenSharing: !state.videoMeeting.screenSharing },
        })),

      toggleHandRaise: () =>
        set((state) => ({
          videoMeeting: { ...state.videoMeeting, handRaised: !state.videoMeeting.handRaised },
        })),

      setLayout: (layout) =>
        set((state) => ({
          videoMeeting: { ...state.videoMeeting, layout },
        })),

      setActiveSpeaker: (userId) =>
        set((state) => ({
          videoMeeting: { ...state.videoMeeting, activeSpeakerId: userId },
        })),
    }),
    {
      name: 'cosmi-meetings',
      version: 1,
      // v1 drops the seeded sample meetings/call-history that older builds
      // persisted to localStorage, so existing installs no longer show mock
      // data alongside the real backend meetings. User-created local meetings
      // (m9+) and call entries are preserved.
      migrate: (persisted) => {
        const state = (persisted ?? {}) as Partial<MeetingsState>
        if (DEMO_MODE) return state as MeetingsState
        const MOCK_MEETING_IDS = new Set(['m1', 'm2', 'm3', 'm4', 'm5', 'm6', 'm7', 'm8'])
        const MOCK_CALL_IDS = new Set(['ch1', 'ch2', 'ch3', 'ch4', 'ch5', 'ch6', 'ch7', 'ch8'])
        return {
          ...state,
          meetings: (state.meetings ?? []).filter((m) => !MOCK_MEETING_IDS.has(m.id)),
          callHistory: (state.callHistory ?? []).filter((c) => !MOCK_CALL_IDS.has(c.id)),
        } as MeetingsState
      },
    }
  )
)

import { create } from 'zustand'
import { persist } from 'zustand/middleware'

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
}

interface MeetingsState {
  meetings: Meeting[]
  activeMeetingId: string | null
  activeCallContactId: string | null
  activeCallContactName: string | null
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
      { id: 'p1', name: 'Anna Mueller', initials: 'AM' },
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
      { id: 'a7', text: 'Farb- und Icon-Konsistenz pruefen', done: false },
    ],
    notes: '',
    files: [{ id: 'f2', name: 'Mockups-v3.fig', size: '12 MB' }],
    whiteboardLink: '',
    projectLink: 'mobile-app',
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
      { id: 'p1', name: 'Anna Mueller', initials: 'AM' },
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
      { id: 'p1', name: 'Anna Mueller', initials: 'AM' },
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
      { id: 'p1', name: 'Anna Mueller', initials: 'AM' },
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
      { id: 'p1', name: 'Anna Mueller', initials: 'AM' },
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
      { id: 'p1', name: 'Anna Mueller', initials: 'AM' },
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
      meetings: mockMeetings,
      activeMeetingId: null,
      activeCallContactId: null,
      activeCallContactName: null,

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
    }),
    { name: 'kmuhub-meetings' }
  )
)

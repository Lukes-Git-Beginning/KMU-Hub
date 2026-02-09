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

export interface Meeting {
  id: string
  title: string
  status: 'live' | 'scheduled' | 'past' | 'cancelled'
  project: string
  date: string
  startTime: string
  duration: number
  room: string
  isVideoCall: boolean
  recurrence: 'none' | 'daily' | 'weekly' | 'monthly'
  reminder: '15min' | '30min' | '1h' | 'none'
  description: string
  participants: MeetingParticipant[]
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
    date: '2026-02-09',
    startTime: '10:00',
    duration: 45,
    room: 'Konferenzraum A',
    isVideoCall: true,
    recurrence: 'weekly',
    reminder: '15min',
    description: 'Woechentliches Sprint Planning fuer das Relaunch-Projekt. Besprechung der Tasks fuer die kommende Woche.',
    participants: [
      { id: 'p1', name: 'Anna Mueller', initials: 'AM' },
      { id: 'p2', name: 'Michael Berg', initials: 'MB' },
      { id: 'p3', name: 'Sarah Klein', initials: 'SK' },
    ],
    files: [{ id: 'f1', name: 'Sprint-Backlog.xlsx', size: '245 KB' }],
    whiteboardLink: '',
    projectLink: 'website-relaunch',
  },
  {
    id: 'm2',
    title: 'Design Review',
    status: 'live',
    project: 'Mobile App',
    date: '2026-02-09',
    startTime: '10:30',
    duration: 30,
    room: 'Huddle Space',
    isVideoCall: true,
    recurrence: 'none',
    reminder: '15min',
    description: 'Review der neuen Design-Mockups fuer die Mobile App. Feedback und Freigabe.',
    participants: [
      { id: 'p3', name: 'Sarah Klein', initials: 'SK' },
      { id: 'p4', name: 'Lisa Schmidt', initials: 'LS' },
    ],
    files: [{ id: 'f2', name: 'Mockups-v3.fig', size: '12 MB' }],
    whiteboardLink: '',
    projectLink: 'mobile-app',
  },
  {
    id: 'm3',
    title: 'Kundenpraesentation Meier AG',
    status: 'scheduled',
    project: 'CRM Integration',
    date: '2026-02-09',
    startTime: '14:00',
    duration: 60,
    room: 'Konferenzraum B',
    isVideoCall: true,
    recurrence: 'none',
    reminder: '30min',
    description: 'Fortschrittspraesentation fuer Meier AG. Aktueller Stand der CRM-Integration und naechste Schritte.',
    participants: [
      { id: 'p1', name: 'Anna Mueller', initials: 'AM' },
      { id: 'p5', name: 'Peter Koch', initials: 'PK' },
      { id: 'p2', name: 'Michael Berg', initials: 'MB' },
    ],
    files: [
      { id: 'f3', name: 'Praesentation-CRM.pptx', size: '8.4 MB' },
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
    date: '2026-02-10',
    startTime: '09:00',
    duration: 15,
    room: 'Remote',
    isVideoCall: true,
    recurrence: 'daily',
    reminder: '15min',
    description: 'Taegliches Team Standup. Kurzer Austausch ueber aktuelle Aufgaben und Blocker.',
    participants: [
      { id: 'p1', name: 'Anna Mueller', initials: 'AM' },
      { id: 'p2', name: 'Michael Berg', initials: 'MB' },
      { id: 'p3', name: 'Sarah Klein', initials: 'SK' },
      { id: 'p6', name: 'Jonas Diaz', initials: 'JD' },
      { id: 'p4', name: 'Lisa Schmidt', initials: 'LS' },
    ],
    files: [],
    whiteboardLink: '',
    projectLink: '',
  },
  {
    id: 'm5',
    title: 'Retrospektive Sprint 12',
    status: 'scheduled',
    project: 'Website Relaunch',
    date: '2026-02-11',
    startTime: '15:00',
    duration: 45,
    room: 'Konferenzraum A',
    isVideoCall: false,
    recurrence: 'none',
    reminder: '30min',
    description: 'Rueckblick auf Sprint 12. Was lief gut, was koennen wir verbessern?',
    participants: [
      { id: 'p1', name: 'Anna Mueller', initials: 'AM' },
      { id: 'p2', name: 'Michael Berg', initials: 'MB' },
    ],
    files: [],
    whiteboardLink: '',
    projectLink: 'website-relaunch',
  },
  {
    id: 'm6',
    title: 'Budget-Planung 2026',
    status: 'scheduled',
    project: 'Finanzen',
    date: '2026-02-12',
    startTime: '10:00',
    duration: 90,
    room: 'Konferenzraum B',
    isVideoCall: false,
    recurrence: 'none',
    reminder: '1h',
    description: 'Jahresbudget-Planung fuer 2026. Kosten, Investitionen, Personalplanung.',
    participants: [
      { id: 'p1', name: 'Anna Mueller', initials: 'AM' },
      { id: 'p5', name: 'Peter Koch', initials: 'PK' },
      { id: 'p7', name: 'Thomas Weber', initials: 'TW' },
    ],
    files: [{ id: 'f5', name: 'Budget-Vorlage-2026.xlsx', size: '1.2 MB' }],
    whiteboardLink: '',
    projectLink: '',
  },
  {
    id: 'm7',
    title: 'API Review',
    status: 'past',
    project: 'Mobile App',
    date: '2026-02-07',
    startTime: '11:00',
    duration: 30,
    room: 'Remote',
    isVideoCall: true,
    recurrence: 'none',
    reminder: '15min',
    description: 'Review der API-Endpoints fuer die Mobile App. Authentifizierung und Datenmodell.',
    participants: [
      { id: 'p6', name: 'Jonas Diaz', initials: 'JD' },
      { id: 'p5', name: 'Peter Koch', initials: 'PK' },
    ],
    files: [{ id: 'f6', name: 'API-Spezifikation.yaml', size: '45 KB' }],
    whiteboardLink: '',
    projectLink: 'mobile-app',
  },
  {
    id: 'm8',
    title: 'Security Briefing',
    status: 'past',
    project: 'Security Audit',
    date: '2026-02-06',
    startTime: '14:00',
    duration: 60,
    room: 'Konferenzraum A',
    isVideoCall: false,
    recurrence: 'none',
    reminder: '30min',
    description: 'Besprechung der Sicherheitsaudit-Ergebnisse. Massnahmenplan und Priorisierung.',
    participants: [
      { id: 'p5', name: 'Peter Koch', initials: 'PK' },
      { id: 'p6', name: 'Jonas Diaz', initials: 'JD' },
      { id: 'p1', name: 'Anna Mueller', initials: 'AM' },
    ],
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

      setActiveMeeting: (id) => set({ activeMeetingId: id }),

      startCall: (contactId, contactName) =>
        set({ activeCallContactId: contactId, activeCallContactName: contactName }),

      endCall: () =>
        set({ activeCallContactId: null, activeCallContactName: null }),
    }),
    { name: 'kmuhub-meetings' }
  )
)

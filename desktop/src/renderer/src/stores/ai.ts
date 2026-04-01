import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type ClassificationLevel = 'öffentlich' | 'intern' | 'vertraulich'

export interface AIActivityEntry {
  id: string
  module: string
  action: string
  timestamp: string
  inputPreview: string
  outputPreview: string
}

interface AIStoreState {
  // Governance (persisted)
  aiEnabled: boolean
  moduleOptOuts: Record<string, boolean>
  noTrainingOnData: boolean
  logAIActivity: boolean

  // Classification cache (persisted)
  fileClassifications: Record<string, ClassificationLevel>

  // Activity log (persisted, capped at 50)
  activityLog: AIActivityEntry[]

  // Actions
  setAIEnabled: (v: boolean) => void
  setModuleOptOut: (module: string, optedOut: boolean) => void
  setLogAIActivity: (v: boolean) => void
  addActivityLog: (entry: Omit<AIActivityEntry, 'id' | 'timestamp'>) => void
  clearActivityLog: () => void
  setFileClassification: (fileId: string, level: ClassificationLevel) => void
  isModuleEnabled: (module: string) => boolean
}

const MOCK_ACTIVITY_LOG: AIActivityEntry[] = [
  {
    id: 'log-1',
    module: 'E-Mail',
    action: 'Entwurf generiert',
    timestamp: '2026-02-22T10:30:00',
    inputPreview: 'Antwort auf Angebot #2847...',
    outputPreview: 'Sehr geehrter Herr Müller, vielen Dank...',
  },
  {
    id: 'log-2',
    module: 'Helpdesk',
    action: 'Antwort vorgeschlagen',
    timestamp: '2026-02-22T09:15:00',
    inputPreview: 'Ticket TK-1247: VPN-Problem...',
    outputPreview: 'Bitte prüfen Sie die Netzwerkeinstellungen...',
  },
  {
    id: 'log-3',
    module: 'Dokumente',
    action: 'Klassifizierung',
    timestamp: '2026-02-22T08:45:00',
    inputPreview: 'Vertrag_Müller_2026.pdf',
    outputPreview: 'vertraulich',
  },
  {
    id: 'log-4',
    module: 'Meetings',
    action: 'Zusammenfassung erstellt',
    timestamp: '2026-02-21T16:20:00',
    inputPreview: 'Sprint Review Meeting (45 Min)...',
    outputPreview: '3 Themen besprochen, 4 Action Items...',
  },
  {
    id: 'log-5',
    module: 'Suche',
    action: 'Semantische Suche',
    timestamp: '2026-02-21T14:05:00',
    inputPreview: 'Wie geht der Onboarding-Prozess?',
    outputPreview: '2 relevante Ergebnisse (87%, 72%)',
  },
  {
    id: 'log-6',
    module: 'E-Mail',
    action: 'Entwurf generiert',
    timestamp: '2026-02-21T11:40:00',
    inputPreview: 'Neue E-Mail an Lieferant...',
    outputPreview: 'Sehr geehrte Damen und Herren...',
  },
  {
    id: 'log-7',
    module: 'Helpdesk',
    action: 'Antwort vorgeschlagen',
    timestamp: '2026-02-21T10:00:00',
    inputPreview: 'Ticket TK-1198: Druckerproblem...',
    outputPreview: 'Der Drucker wurde neu konfiguriert...',
  },
  {
    id: 'log-8',
    module: 'Dokumente',
    action: 'Klassifizierung',
    timestamp: '2026-02-20T17:30:00',
    inputPreview: 'Jahresbericht_2025.pdf',
    outputPreview: 'intern',
  },
  {
    id: 'log-9',
    module: 'Meetings',
    action: 'Zusammenfassung erstellt',
    timestamp: '2026-02-20T15:10:00',
    inputPreview: 'Kundentermin Müller GmbH...',
    outputPreview: 'Angebot wird bis Freitag erstellt...',
  },
  {
    id: 'log-10',
    module: 'Suche',
    action: 'Semantische Suche',
    timestamp: '2026-02-20T09:25:00',
    inputPreview: 'DATEV Export konfigurieren',
    outputPreview: '3 relevante Ergebnisse (91%, 78%, 65%)',
  },
]

let logCounter = 100

export const useAIStore = create<AIStoreState>()(
  persist(
    (set, get) => ({
      aiEnabled: true,
      moduleOptOuts: {
        email: false,
        meetings: false,
        helpdesk: false,
        search: false,
        docs: false,
      },
      noTrainingOnData: true,
      logAIActivity: true,
      fileClassifications: {},
      activityLog: MOCK_ACTIVITY_LOG,

      setAIEnabled: (v) => set({ aiEnabled: v }),

      setModuleOptOut: (module, optedOut) =>
        set((state) => ({
          moduleOptOuts: { ...state.moduleOptOuts, [module]: optedOut },
        })),

      setLogAIActivity: (v) => set({ logAIActivity: v }),

      addActivityLog: (entry) =>
        set((state) => {
          const newEntry: AIActivityEntry = {
            ...entry,
            id: `log-${++logCounter}`,
            timestamp: new Date().toISOString(),
          }
          const log = [newEntry, ...state.activityLog].slice(0, 50)
          return { activityLog: log }
        }),

      clearActivityLog: () => set({ activityLog: [] }),

      setFileClassification: (fileId, level) =>
        set((state) => ({
          fileClassifications: { ...state.fileClassifications, [fileId]: level },
        })),

      isModuleEnabled: (module) => {
        const s = get()
        return s.aiEnabled && !s.moduleOptOuts[module]
      },
    }),
    { name: 'kmuhub-ai' },
  ),
)

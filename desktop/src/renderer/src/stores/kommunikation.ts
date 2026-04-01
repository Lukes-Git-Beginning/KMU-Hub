/**
 * Kommunikation (Unified Inbox) store.
 *
 * Server data comes from TanStack Query hooks (useInboxMessages, etc.).
 * This store holds only UI state and canned responses (until canned response API exists).
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type {
  CannedResponse,
  CommunicationChannel,
} from '@/types/communication'

// ---------------------------------------------------------------------------
// State interface
// ---------------------------------------------------------------------------

interface KommunikationState {
  // Data — canned responses (TODO: migrate to API when canned response CRUD endpoint exists)
  cannedResponses: CannedResponse[]

  // UI state
  activeView: string
  activeChannel: CommunicationChannel | 'all'
  selectedConversationId: string | null
  sidebarCollapsed: boolean
  detailPaneOpen: boolean
  searchQuery: string

  // Actions — UI
  setActiveView: (view: string) => void
  setActiveChannel: (channel: CommunicationChannel | 'all') => void
  setSelectedConversation: (id: string | null) => void
  toggleSidebar: () => void
  toggleDetailPane: () => void
  setSearchQuery: (query: string) => void

  // Actions — canned responses
  addCannedResponse: (response: Omit<CannedResponse, 'id' | 'usageCount'>) => void
  deleteCannedResponse: (id: string) => void
  incrementCannedResponseUsage: (id: string) => void
}

// ---------------------------------------------------------------------------
// Mock data — Canned responses
// TODO: Replace with canned response CRUD API
// ---------------------------------------------------------------------------

const mockCannedResponses: CannedResponse[] = [
  {
    id: 'cr1',
    title: 'Begrüssung',
    content: 'Guten Tag! Vielen Dank für Ihre Nachricht. Wie kann ich Ihnen helfen?',
    category: 'allgemein',
    shortcut: '/hi',
    createdBy: 'Anna Müller',
    usageCount: 47,
  },
  {
    id: 'cr2',
    title: 'Weiterleitung intern',
    content: 'Ich leite Ihre Anfrage an die zuständige Abteilung weiter. Sie erhalten in Kürze eine Rückmeldung.',
    category: 'allgemein',
    shortcut: '/weiter',
    createdBy: 'Sarah Meier',
    usageCount: 32,
  },
  {
    id: 'cr3',
    title: 'Terminbestätigung',
    content: 'Hiermit bestätigen wir Ihren Termin am {{datum}} um {{uhrzeit}}. Bitte bringen Sie alle relevanten Unterlagen mit.',
    category: 'termine',
    shortcut: '/termin',
    createdBy: 'Lisa Braun',
    usageCount: 28,
  },
  {
    id: 'cr4',
    title: 'Angebot nachfassen',
    content: 'Bezugnehmend auf unser Angebot vom {{datum}} wollte ich mich erkundigen, ob Sie noch Fragen haben oder weitere Informationen benötigen.',
    category: 'vertrieb',
    shortcut: '/followup',
    createdBy: 'Peter Schmidt',
    usageCount: 19,
  },
  {
    id: 'cr5',
    title: 'Ticket erstellt',
    content: 'Wir haben ein Support-Ticket (#{{ticketNr}}) für Ihre Anfrage erstellt. Unser Team kümmert sich darum und meldet sich schnellstmöglich.',
    category: 'support',
    shortcut: '/ticket',
    createdBy: 'Sarah Meier',
    usageCount: 55,
  },
]

// ---------------------------------------------------------------------------
// ID counter for canned responses
// ---------------------------------------------------------------------------

let nextCrId = 6

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useKommunikationStore = create<KommunikationState>()(
  persist(
    (set) => ({
      // Data
      cannedResponses: mockCannedResponses,

      // UI state
      activeView: 'all',
      activeChannel: 'all',
      selectedConversationId: null,
      sidebarCollapsed: false,
      detailPaneOpen: true,
      searchQuery: '',

      // -- UI actions --

      setActiveView: (view) =>
        set({ activeView: view, selectedConversationId: null }),

      setActiveChannel: (channel) =>
        set({ activeChannel: channel, selectedConversationId: null }),

      setSelectedConversation: (id) =>
        set({ selectedConversationId: id, detailPaneOpen: id !== null }),

      toggleSidebar: () =>
        set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),

      toggleDetailPane: () =>
        set((state) => ({ detailPaneOpen: !state.detailPaneOpen })),

      setSearchQuery: (query) => set({ searchQuery: query }),

      // -- Canned response actions --

      addCannedResponse: (response) => {
        const id = `cr${nextCrId++}`
        set((state) => ({
          cannedResponses: [...state.cannedResponses, { ...response, id, usageCount: 0 }],
        }))
      },

      deleteCannedResponse: (id) =>
        set((state) => ({
          cannedResponses: state.cannedResponses.filter((r) => r.id !== id),
        })),

      incrementCannedResponseUsage: (id) =>
        set((state) => ({
          cannedResponses: state.cannedResponses.map((r) =>
            r.id === id ? { ...r, usageCount: r.usageCount + 1 } : r,
          ),
        })),
    }),
    {
      name: 'kmuhub-kommunikation',
      partialize: (state) => ({
        activeView: state.activeView,
        activeChannel: state.activeChannel,
        sidebarCollapsed: state.sidebarCollapsed,
      }),
    },
  ),
)

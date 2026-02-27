import { create } from 'zustand'
import { persist } from 'zustand/middleware'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type SearchResultType =
  | 'contact'
  | 'project'
  | 'task'
  | 'document'
  | 'wiki'
  | 'event'
  | 'email'
  | 'conversation'
  | 'invoice'
  | 'ticket'

export interface SearchResult {
  id: string
  type: SearchResultType
  title: string
  subtitle: string
  icon: string
  route: string
}

export interface QuickAction {
  id: string
  label: string
  icon: string
  shortcut?: string
  route?: string
  action?: string
}

export interface RecentSearch {
  query: string
  timestamp: string
}

// ---------------------------------------------------------------------------
// State interface
// ---------------------------------------------------------------------------

interface SearchState {
  isOpen: boolean
  query: string
  results: SearchResult[]
  recentSearches: RecentSearch[]
  isLoading: boolean

  // Actions
  open: () => void
  close: () => void
  setQuery: (query: string) => void
  setResults: (results: SearchResult[]) => void
  setLoading: (loading: boolean) => void
  addRecentSearch: (query: string) => void
  clearRecentSearches: () => void
  removeRecentSearch: (query: string) => void
}

// ---------------------------------------------------------------------------
// Quick actions (static)
// ---------------------------------------------------------------------------

export const QUICK_ACTIONS: QuickAction[] = [
  { id: 'qa1', label: 'Neuer Kontakt', icon: 'UserPlus', shortcut: 'N', route: '/kontakte' },
  { id: 'qa2', label: 'Neues Projekt', icon: 'FolderPlus', route: '/work' },
  { id: 'qa3', label: 'Neue Rechnung', icon: 'Receipt', route: '/buchhaltung' },
  { id: 'qa4', label: 'Neues Meeting', icon: 'Video', route: '/meetings' },
  { id: 'qa5', label: 'Timer starten', icon: 'Clock', route: '/zeiterfassung' },
  { id: 'qa6', label: 'Neue E-Mail', icon: 'Mail', action: 'compose-email' },
  { id: 'qa7', label: 'Neues Ticket', icon: 'LifeBuoy', route: '/helpdesk' },
  { id: 'qa8', label: 'Wiki-Artikel erstellen', icon: 'BookOpen', route: '/wiki' },
]

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useSearchStore = create<SearchState>()(
  persist(
    (set) => ({
      isOpen: false,
      query: '',
      results: [],
      recentSearches: [],
      isLoading: false,

      open: () => set({ isOpen: true, query: '', results: [] }),
      close: () => set({ isOpen: false, query: '', results: [] }),

      setQuery: (query) => set({ query }),
      setResults: (results) => set({ results, isLoading: false }),
      setLoading: (loading) => set({ isLoading: loading }),

      addRecentSearch: (query) => {
        if (!query.trim()) return
        set((state) => {
          const filtered = state.recentSearches.filter((r) => r.query !== query)
          return {
            recentSearches: [
              { query, timestamp: new Date().toISOString() },
              ...filtered,
            ].slice(0, 5),
          }
        })
      },

      clearRecentSearches: () => set({ recentSearches: [] }),

      removeRecentSearch: (query) =>
        set((state) => ({
          recentSearches: state.recentSearches.filter((r) => r.query !== query),
        })),
    }),
    {
      name: 'kmuhub-search',
      partialize: (state) => ({
        recentSearches: state.recentSearches,
      }),
    },
  ),
)

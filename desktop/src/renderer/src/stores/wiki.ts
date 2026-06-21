/**
 * Wiki UI Store — replaces the former mock-data store.
 *
 * Server state (articles, categories, versions, templates) is now managed by
 * TanStack Query hooks in api/hooks/useWiki.ts.
 *
 * This store only handles:
 *   - UI navigation state (selected article/category)
 *   - Editor state (isEditing)
 *   - Search query (client-side filter)
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

// ---------------------------------------------------------------------------
// Article meta (mock-first — pin state has no backend field yet; tags are now
// a real article field served by MSW + the adapter)
// ---------------------------------------------------------------------------

export interface WikiArticleMeta {
  pinned: boolean
}

/** Demo seed so the list shows pins before the backend exposes them. */
const SEED_ARTICLE_META: Record<string, WikiArticleMeta> = {
  'wart-001': { pinned: true },
  'wart-002': { pinned: true },
  'wart-003': { pinned: false },
  'wart-004': { pinned: false },
  'wart-005': { pinned: false },
  'wart-006': { pinned: false },
  'wart-007': { pinned: false },
}

// ---------------------------------------------------------------------------
// UI state interface
// ---------------------------------------------------------------------------

interface WikiUIState {
  selectedArticleId: string | null
  selectedCategoryId: string | null
  isEditing: boolean
  searchQuery: string
  /** Per-article pin + tag overrides (mock-first). */
  articleMeta: Record<string, WikiArticleMeta>

  setSelectedArticle: (id: string | null) => void
  setSelectedCategory: (id: string | null) => void
  setEditing: (editing: boolean) => void
  setSearchQuery: (query: string) => void
  togglePin: (id: string) => void
}

export const useWikiStore = create<WikiUIState>()(
  persist(
    (set) => ({
      selectedArticleId: null,
      selectedCategoryId: null,
      isEditing: false,
      searchQuery: '',
      articleMeta: SEED_ARTICLE_META,
      setSelectedArticle: (selectedArticleId) => set({ selectedArticleId }),
      setSelectedCategory: (selectedCategoryId) => set({ selectedCategoryId }),
      setEditing: (isEditing) => set({ isEditing }),
      setSearchQuery: (searchQuery) => set({ searchQuery }),
      togglePin: (id) =>
        set((state) => {
          const current = state.articleMeta[id] ?? { pinned: false }
          return {
            articleMeta: {
              ...state.articleMeta,
              [id]: { ...current, pinned: !current.pinned },
            },
          }
        }),
    }),
    {
      name: 'cosmi-wiki',
      version: 5,
      partialize: (state) => ({
        selectedCategoryId: state.selectedCategoryId,
        articleMeta: state.articleMeta,
      }),
    },
  ),
)

/**
 * Wiki UI Store — replaces the former mock-data store.
 *
 * Server state (articles, categories, versions) is now managed by
 * TanStack Query hooks in api/hooks/useWiki.ts.
 *
 * This store only handles:
 *   - UI navigation state (selected article/category)
 *   - Editor state (isEditing)
 *   - Search query (client-side filter)
 *   - Static template definitions
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { WikiTemplate } from '@/types/wiki'

// ---------------------------------------------------------------------------
// Static templates (content scaffolds — no backend needed)
// ---------------------------------------------------------------------------

export const WIKI_TEMPLATES: WikiTemplate[] = [
  {
    id: 'wt1',
    name: 'Meeting-Protokoll',
    description: 'Standardvorlage für Meeting-Notizen',
    content: '<h2>Meeting: [Titel]</h2><p><strong>Datum:</strong> [Datum]<br><strong>Teilnehmer:</strong> [Namen]</p><h3>Agenda</h3><ol><li>[Punkt 1]</li><li>[Punkt 2]</li></ol><h3>Beschlüsse</h3><ul><li>[Beschluss]</li></ul><h3>Nächste Schritte</h3><ul><li>[Aktion] — [Verantwortlich] — [Frist]</li></ul>',
    icon: 'FileText',
    category: 'meetings',
  },
  {
    id: 'wt2',
    name: 'Post-Mortem',
    description: 'Analyse nach einem Vorfall oder Problem',
    content: '<h2>Post-Mortem: [Vorfall]</h2><p><strong>Datum:</strong> [Datum]<br><strong>Schweregrad:</strong> [Kritisch/Hoch/Mittel]</p><h3>Zusammenfassung</h3><p>[Was ist passiert?]</p><h3>Timeline</h3><ul><li>[HH:MM] — [Ereignis]</li></ul><h3>Root Cause</h3><p>[Ursache]</p><h3>Massnahmen</h3><ul><li>[Massnahme] — [Verantwortlich] — [Frist]</li></ul><h3>Lessons Learned</h3><ul><li>[Erkenntnis]</li></ul>',
    icon: 'AlertTriangle',
    category: 'incidents',
  },
  {
    id: 'wt3',
    name: 'How-To Anleitung',
    description: 'Schritt-für-Schritt-Anleitung für Prozesse',
    content: '<h2>How-To: [Titel]</h2><p>[Kurzbeschreibung — was wird erreicht?]</p><h3>Voraussetzungen</h3><ul><li>[Voraussetzung 1]</li></ul><h3>Schritte</h3><ol><li>[Schritt 1]</li><li>[Schritt 2]</li><li>[Schritt 3]</li></ol><h3>Häufige Probleme</h3><ul><li><strong>Problem:</strong> [Beschreibung]<br><strong>Lösung:</strong> [Lösung]</li></ul>',
    icon: 'BookOpen',
    category: 'howto',
  },
]

// ---------------------------------------------------------------------------
// UI state interface
// ---------------------------------------------------------------------------

interface WikiUIState {
  selectedArticleId: string | null
  selectedCategoryId: string | null
  isEditing: boolean
  searchQuery: string

  setSelectedArticle: (id: string | null) => void
  setSelectedCategory: (id: string | null) => void
  setEditing: (editing: boolean) => void
  setSearchQuery: (query: string) => void
}

export const useWikiStore = create<WikiUIState>()(
  persist(
    (set) => ({
      selectedArticleId: null,
      selectedCategoryId: null,
      isEditing: false,
      searchQuery: '',
      setSelectedArticle: (selectedArticleId) => set({ selectedArticleId }),
      setSelectedCategory: (selectedCategoryId) => set({ selectedCategoryId }),
      setEditing: (isEditing) => set({ isEditing }),
      setSearchQuery: (searchQuery) => set({ searchQuery }),
    }),
    {
      name: 'cosmi-wiki',
      version: 3,
      partialize: (state) => ({ selectedCategoryId: state.selectedCategoryId }),
    },
  ),
)

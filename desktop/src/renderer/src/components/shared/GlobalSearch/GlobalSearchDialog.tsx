import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  User,
  FolderKanban,
  CheckSquare,
  FileText,
  BookOpen,
  Calendar,
  Mail,
  MessageSquare,
  Receipt,
  LifeBuoy,
  Search,
  Sparkles,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from '@/components/ui/dialog'
import { useSearchStore, QUICK_ACTIONS } from '@/stores/search'
import type { SearchResultType, QuickAction } from '@/stores/search'
import { useContacts } from '@/api/hooks/useContacts'
import { backendContactToUI } from '@/modules/kontakte/adapters'
import { SearchInput } from './SearchInput'
import { SearchResultGroup } from './SearchResultGroup'
import type { GroupedResult } from './SearchResultGroup'
import { RecentSearches } from './RecentSearches'
import { QuickActions } from './QuickActions'
import { useAIStore } from '@/stores/ai'

// ---------------------------------------------------------------------------
// Icon mapping for search result types
// ---------------------------------------------------------------------------

const typeIcons: Record<SearchResultType, LucideIcon> = {
  contact: User,
  project: FolderKanban,
  task: CheckSquare,
  document: FileText,
  wiki: BookOpen,
  event: Calendar,
  email: Mail,
  conversation: MessageSquare,
  invoice: Receipt,
  ticket: LifeBuoy,
}

const typeLabels: Record<SearchResultType, string> = {
  contact: 'Kontakte',
  project: 'Projekte',
  task: 'Aufgaben',
  document: 'Dokumente',
  wiki: 'Wiki',
  event: 'Termine',
  email: 'E-Mails',
  conversation: 'Konversationen',
  invoice: 'Rechnungen',
  ticket: 'Tickets',
}

// ---------------------------------------------------------------------------
// Mock search — searches through existing store data
// ---------------------------------------------------------------------------

function useMockSearch(query: string): GroupedResult[] {
  const { data } = useContacts()
  const contacts = (data?.contacts ?? []).map(backendContactToUI)

  return useMemo(() => {
    if (!query.trim()) return []
    const q = query.toLowerCase()
    const groups: GroupedResult[] = []

    // Search contacts
    const matchedContacts = contacts
      .filter(
        (c) =>
          `${c.firstName} ${c.lastName}`.toLowerCase().includes(q) ||
          c.email.toLowerCase().includes(q) ||
          c.company.toLowerCase().includes(q),
      )
      .slice(0, 5)
      .map((c) => ({
        id: c.id,
        icon: typeIcons.contact,
        title: `${c.firstName} ${c.lastName}`.trim(),
        subtitle: [c.company, c.email].filter(Boolean).join(' · '),
        route: '/kontakte',
      }))

    if (matchedContacts.length > 0) {
      groups.push({ label: typeLabels.contact, items: matchedContacts })
    }

    // Mock project results
    const projectKeywords = ['projekt', 'website', 'app', 'redesign', 'migration', 'crm']
    if (projectKeywords.some((k) => k.includes(q) || q.includes(k))) {
      groups.push({
        label: typeLabels.project,
        items: [
          { id: 'p1', icon: typeIcons.project, title: 'Website Redesign', subtitle: 'In Bearbeitung · 68%', route: '/work' },
          { id: 'p2', icon: typeIcons.project, title: 'CRM Migration', subtitle: 'Geplant · Q2 2026', route: '/work' },
        ],
      })
    }

    // Mock wiki results
    const wikiKeywords = ['wiki', 'artikel', 'anleitung', 'howto', 'how-to', 'doku']
    if (wikiKeywords.some((k) => k.includes(q) || q.includes(k))) {
      groups.push({
        label: typeLabels.wiki,
        items: [
          { id: 'w1', icon: typeIcons.wiki, title: 'Onboarding-Prozess', subtitle: 'Wiki · Zuletzt bearbeitet vor 3 Tagen', route: '/wiki' },
          { id: 'w2', icon: typeIcons.wiki, title: 'API-Dokumentation', subtitle: 'Wiki · Entwurf', route: '/wiki' },
        ],
      })
    }

    // Mock ticket results
    const ticketKeywords = ['ticket', 'bug', 'fehler', 'support', 'helpdesk']
    if (ticketKeywords.some((k) => k.includes(q) || q.includes(k))) {
      groups.push({
        label: typeLabels.ticket,
        items: [
          { id: 't1', icon: typeIcons.ticket, title: 'Login funktioniert nicht', subtitle: '#1042 · Offen · Hoch', route: '/helpdesk' },
          { id: 't2', icon: typeIcons.ticket, title: 'PDF-Export fehlerhaft', subtitle: '#1038 · In Bearbeitung', route: '/helpdesk' },
        ],
      })
    }

    return groups
  }, [query, contacts])
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function GlobalSearchDialog() {
  const isOpen = useSearchStore((s) => s.isOpen)
  const open = useSearchStore((s) => s.open)
  const close = useSearchStore((s) => s.close)
  const query = useSearchStore((s) => s.query)
  const setQuery = useSearchStore((s) => s.setQuery)
  const recentSearches = useSearchStore((s) => s.recentSearches)
  const addRecentSearch = useSearchStore((s) => s.addRecentSearch)
  const removeRecentSearch = useSearchStore((s) => s.removeRecentSearch)
  const clearRecentSearches = useSearchStore((s) => s.clearRecentSearches)

  const navigate = useNavigate()
  const inputRef = useRef<HTMLInputElement>(null)
  const [activeIndex, setActiveIndex] = useState(0)

  const searchResults = useMockSearch(query)
  const hasQuery = query.trim().length > 0

  // Semantic search (AI)
  const aiSearchEnabled = useAIStore((s) => s.isModuleEnabled('search'))
  const [semanticResults, setSemanticResults] = useState<GroupedResult[]>([])
  const [semanticLoading, setSemanticLoading] = useState(false)

   
  useEffect(() => {
    if (!hasQuery || query.length < 3 || !aiSearchEnabled) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- reset state on dependency change
      setSemanticResults([])
      setSemanticLoading(false)
      return
    }
    setSemanticLoading(true)
    const timer = setTimeout(() => {
      const mockSemanticMap: Record<string, GroupedResult> = {
        default: {
          label: 'KI-Ergebnisse',
          items: [
            { id: 'ai-1', icon: Sparkles, title: 'Ähnliches Thema im Wiki', subtitle: 'Onboarding-Prozess — 87% Relevanz', route: '/wiki' },
            { id: 'ai-2', icon: Sparkles, title: 'Verwandtes Ticket', subtitle: '#1038 PDF-Export — 72% Relevanz', route: '/helpdesk' },
          ],
        },
      }
      setSemanticResults([mockSemanticMap.default])
      setSemanticLoading(false)
      useAIStore.getState().addActivityLog({
        module: 'Suche',
        action: 'Semantische Suche',
        inputPreview: query.slice(0, 50),
        outputPreview: '2 relevante Ergebnisse',
      })
    }, 1500)
    return () => clearTimeout(timer)
  }, [query, hasQuery, aiSearchEnabled])

  const allResults = useMemo(() => {
    if (semanticResults.length > 0) return [...searchResults, ...semanticResults]
    return searchResults
  }, [searchResults, semanticResults])

  // Total navigable items for keyboard nav
  const totalItems = useMemo(() => {
    if (hasQuery) {
      return allResults.reduce((sum, g) => sum + g.items.length, 0)
    }
    return QUICK_ACTIONS.length
  }, [hasQuery, allResults])

   
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- reset state on dependency change
    setActiveIndex(0)
  }, [query])

  // Global Cmd+K / Ctrl+K shortcut
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        if (isOpen) {
          close()
        } else {
          open()
        }
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [isOpen, open, close])

  // Focus input on open
  useEffect(() => {
    if (isOpen) {
      setTimeout(() => inputRef.current?.focus(), 50)
    }
  }, [isOpen])

  const handleSelect = useCallback(
    (route: string) => {
      if (hasQuery) addRecentSearch(query)
      close()
      navigate(route)
    },
    [hasQuery, query, addRecentSearch, close, navigate],
  )

  const handleQuickAction = useCallback(
    (action: QuickAction) => {
      close()
      if (action.route) {
        navigate(action.route)
      }
    },
    [close, navigate],
  )

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      switch (e.key) {
        case 'ArrowDown':
          e.preventDefault()
          setActiveIndex((prev) => (prev + 1) % Math.max(totalItems, 1))
          break
        case 'ArrowUp':
          e.preventDefault()
          setActiveIndex((prev) =>
            prev <= 0 ? Math.max(totalItems - 1, 0) : prev - 1,
          )
          break
        case 'Enter': {
          e.preventDefault()
          if (hasQuery) {
            // Find the item at activeIndex across groups
            let idx = 0
            for (const group of searchResults) {
              for (const item of group.items) {
                if (idx === activeIndex) {
                  handleSelect(item.route)
                  return
                }
                idx++
              }
            }
          } else {
            const action = QUICK_ACTIONS[activeIndex]
            if (action) handleQuickAction(action)
          }
          break
        }
      }
    },
    [totalItems, hasQuery, activeIndex, searchResults, handleSelect, handleQuickAction],
  )

  return (
    <Dialog
      open={isOpen}
      onOpenChange={(o) => {
        if (!o) close()
      }}
    >
      <DialogContent
        className="top-[20%] max-w-lg translate-y-0 gap-0 overflow-hidden p-0 sm:top-[20%] sm:translate-y-0"
        onKeyDown={handleKeyDown}
      >
        <DialogTitle className="sr-only">Globale Suche</DialogTitle>

        <SearchInput
          ref={inputRef}
          value={query}
          onChange={setQuery}
          onClear={() => setQuery('')}
        />

        <div className="max-h-[360px] overflow-y-auto">
          {hasQuery ? (
            allResults.length > 0 || semanticLoading ? (
              <>
                {(() => {
                  let offset = 0
                  return allResults.map((group) => {
                    const startIndex = offset
                    offset += group.items.length
                    return (
                      <SearchResultGroup
                        key={group.label}
                        group={group}
                        activeIndex={activeIndex}
                        startIndex={startIndex}
                        onSelect={handleSelect}
                        onHover={setActiveIndex}
                      />
                    )
                  })
                })()}
                {semanticLoading && (
                  <div className="flex items-center gap-2 px-4 py-3 border-t border-border">
                    <span className="h-3 w-3 animate-spin rounded-full border-2 border-primary border-t-transparent" />
                    <span className="text-xs text-primary">KI sucht semantisch...</span>
                  </div>
                )}
              </>
            ) : (
              <div className="flex flex-col items-center gap-2 py-10 text-muted-foreground">
                <Search className="h-8 w-8 opacity-40" />
                <span className="text-sm">Keine Ergebnisse für &quot;{query}&quot;</span>
              </div>
            )
          ) : (
            <>
              <RecentSearches
                searches={recentSearches}
                onSelect={(q) => setQuery(q)}
                onRemove={removeRecentSearch}
                onClearAll={clearRecentSearches}
              />
              <QuickActions
                actions={QUICK_ACTIONS}
                activeIndex={activeIndex}
                startIndex={0}
                onSelect={handleQuickAction}
                onHover={setActiveIndex}
              />
            </>
          )}
        </div>

        <div className="flex items-center gap-3 border-t border-border px-3 py-2 text-[11px] text-muted-foreground">
          <span className="flex items-center gap-1">
            <kbd className="rounded border border-border bg-muted px-1 py-0.5 font-mono text-[10px]">↑↓</kbd>
            Navigieren
          </span>
          <span className="flex items-center gap-1">
            <kbd className="rounded border border-border bg-muted px-1 py-0.5 font-mono text-[10px]">↵</kbd>
            Öffnen
          </span>
          <span className="flex items-center gap-1">
            <kbd className="rounded border border-border bg-muted px-1 py-0.5 font-mono text-[10px]">esc</kbd>
            Schliessen
          </span>
        </div>
      </DialogContent>
    </Dialog>
  )
}

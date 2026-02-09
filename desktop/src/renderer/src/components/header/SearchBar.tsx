import { useState, useRef, useEffect } from 'react'
import {
  Search,
  FolderKanban,
  CheckSquare,
  FileText,
  MessageSquare,
  Users,
  Calculator,
  ArrowRight,
  X,
  Filter,
} from 'lucide-react'
import { useNavigate } from 'react-router-dom'

interface SearchResult {
  id: string
  title: string
  category: string
  categoryIcon: typeof FolderKanban
  path: string
  description?: string
  iconColor: string
  bgColor: string
}

const allSearchData: SearchResult[] = [
  { id: 'p1', title: 'Website Relaunch', category: 'Projekte', categoryIcon: FolderKanban, path: '/work/projects', description: 'In Progress', iconColor: 'text-blue-600', bgColor: 'bg-blue-500/10' },
  { id: 'p2', title: 'Mobile App Development', category: 'Projekte', categoryIcon: FolderKanban, path: '/work/projects', description: 'Planning', iconColor: 'text-blue-600', bgColor: 'bg-blue-500/10' },
  { id: 'p3', title: 'Brand Refresh', category: 'Projekte', categoryIcon: FolderKanban, path: '/work/projects', description: 'Review', iconColor: 'text-blue-600', bgColor: 'bg-blue-500/10' },
  { id: 'p4', title: 'API Integration', category: 'Projekte', categoryIcon: FolderKanban, path: '/work/projects', description: 'Testing', iconColor: 'text-blue-600', bgColor: 'bg-blue-500/10' },
  { id: 't1', title: 'API Dokumentation schreiben', category: 'Aufgaben', categoryIcon: CheckSquare, path: '/work/my-tasks', description: 'Faellig morgen', iconColor: 'text-emerald-600', bgColor: 'bg-emerald-500/10' },
  { id: 't2', title: 'Design Review vorbereiten', category: 'Aufgaben', categoryIcon: CheckSquare, path: '/work/my-tasks', description: 'Hohe Prioritaet', iconColor: 'text-emerald-600', bgColor: 'bg-emerald-500/10' },
  { id: 't3', title: 'Logo Redesign', category: 'Aufgaben', categoryIcon: CheckSquare, path: '/work/my-tasks', description: 'In Bearbeitung', iconColor: 'text-emerald-600', bgColor: 'bg-emerald-500/10' },
  { id: 't4', title: 'Testing Sprint vorbereiten', category: 'Aufgaben', categoryIcon: CheckSquare, path: '/work/my-tasks', description: 'Diese Woche', iconColor: 'text-emerald-600', bgColor: 'bg-emerald-500/10' },
  { id: 'd1', title: 'Q1 Budget.xlsx', category: 'Dokumente', categoryIcon: FileText, path: '/dokumente', description: 'Finanzen', iconColor: 'text-purple-600', bgColor: 'bg-purple-500/10' },
  { id: 'd2', title: 'Vertrag_Kunde_A.pdf', category: 'Dokumente', categoryIcon: FileText, path: '/dokumente', description: 'Legal', iconColor: 'text-purple-600', bgColor: 'bg-purple-500/10' },
  { id: 'd3', title: 'Projektplan_2025.docx', category: 'Dokumente', categoryIcon: FileText, path: '/dokumente', description: 'Planung', iconColor: 'text-purple-600', bgColor: 'bg-purple-500/10' },
  { id: 'a1', title: 'Rechnung #2024-156', category: 'Buchhaltung', categoryIcon: Calculator, path: '/buchhaltung', description: 'CHF 12,500', iconColor: 'text-orange-600', bgColor: 'bg-orange-500/10' },
  { id: 'a2', title: 'Bexio Integration', category: 'Buchhaltung', categoryIcon: Calculator, path: '/buchhaltung', description: 'Verbunden', iconColor: 'text-orange-600', bgColor: 'bg-orange-500/10' },
  { id: 'c1', title: 'Michael Berg', category: 'Kontakte', categoryIcon: MessageSquare, path: '/kontakte', description: 'Senior Designer', iconColor: 'text-pink-600', bgColor: 'bg-pink-500/10' },
  { id: 'c2', title: 'Sarah Klein', category: 'Kontakte', categoryIcon: MessageSquare, path: '/kontakte', description: 'Lead Developer', iconColor: 'text-pink-600', bgColor: 'bg-pink-500/10' },
  { id: 'c3', title: 'Anna Mueller', category: 'Kontakte', categoryIcon: MessageSquare, path: '/kontakte', description: 'Project Manager', iconColor: 'text-pink-600', bgColor: 'bg-pink-500/10' },
  { id: 'tm1', title: 'Team-Mitglieder verwalten', category: 'Team', categoryIcon: Users, path: '/team', description: '12 Mitglieder', iconColor: 'text-cyan-600', bgColor: 'bg-cyan-500/10' },
  { id: 'tm2', title: 'Entwickler Team', category: 'Team', categoryIcon: Users, path: '/team', description: '5 Mitglieder', iconColor: 'text-cyan-600', bgColor: 'bg-cyan-500/10' },
]

const categoryFilters = ['Alle', 'Projekte', 'Aufgaben', 'Dokumente', 'Kontakte', 'Buchhaltung', 'Team']

/**
 * Compact search trigger button that opens a modal search overlay.
 * Keyboard shortcut: Ctrl+K / Cmd+K.
 */
export function SearchBar() {
  const [isOpen, setIsOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<SearchResult[]>([])
  const [activeFilter, setActiveFilter] = useState('Alle')
  const inputRef = useRef<HTMLInputElement>(null)
  const navigate = useNavigate()

  // Ctrl+K shortcut
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        setIsOpen(true)
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [])

  // Focus input when modal opens
  useEffect(() => {
    if (isOpen) {
      setTimeout(() => inputRef.current?.focus(), 50)
    } else {
      setQuery('')
      setResults([])
      setActiveFilter('Alle')
    }
  }, [isOpen])

  // Search logic
  useEffect(() => {
    if (!query.trim()) {
      setResults([])
      return
    }
    const q = query.toLowerCase()
    const filtered = allSearchData.filter((item) => {
      if (activeFilter !== 'Alle' && item.category !== activeFilter) return false
      return (
        item.title.toLowerCase().includes(q) ||
        item.category.toLowerCase().includes(q) ||
        item.description?.toLowerCase().includes(q)
      )
    })
    setResults(filtered.slice(0, 8))
  }, [query, activeFilter])

  const handleResultClick = (path: string) => {
    navigate(path)
    setIsOpen(false)
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && results.length > 0) {
      handleResultClick(results[0].path)
    }
    if (e.key === 'Escape') {
      setIsOpen(false)
    }
  }

  return (
    <>
      {/* Compact trigger */}
      <button
        onClick={() => setIsOpen(true)}
        className="flex w-56 md:w-72 items-center gap-2 rounded-lg border border-border bg-secondary/50 px-3 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
      >
        <Search className="h-3.5 w-3.5" />
        <span className="hidden sm:inline">Suchen...</span>
        <kbd className="hidden md:inline-flex h-5 items-center rounded border border-border bg-card px-1.5 text-[10px] font-medium text-text-disabled">
          Ctrl+K
        </kbd>
      </button>

      {/* Modal overlay */}
      {isOpen && (
        <div
          className="fixed inset-0 z-50 flex items-start justify-center pt-[15vh]"
          onClick={(e) => {
            if (e.target === e.currentTarget) setIsOpen(false)
          }}
        >
          {/* Backdrop */}
          <div className="absolute inset-0 bg-black/40" />

          {/* Search modal */}
          <div className="relative w-full max-w-xl rounded-xl border border-border bg-card shadow-[var(--shadow-large)] overflow-hidden">
            {/* Search input */}
            <div className="flex items-center gap-3 border-b border-border px-4 py-3">
              <Search className="h-5 w-5 shrink-0 text-muted-foreground" />
              <input
                ref={inputRef}
                type="text"
                placeholder="Projekte, Aufgaben, Kontakte, Dokumente..."
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                onKeyDown={handleKeyDown}
                className="flex-1 bg-transparent text-sm text-foreground placeholder:text-muted-foreground outline-none"
              />
              <button
                onClick={() => setIsOpen(false)}
                className="rounded-md p-1 text-muted-foreground hover:bg-secondary"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            {/* Category filters */}
            <div className="flex items-center gap-1 border-b border-border-muted px-4 py-2 overflow-x-auto scrollbar-hide">
              <Filter className="h-3.5 w-3.5 shrink-0 text-muted-foreground mr-1" />
              {categoryFilters.map((cat) => (
                <button
                  key={cat}
                  onClick={() => setActiveFilter(cat)}
                  className={`shrink-0 rounded-full px-2.5 py-1 text-xs transition-colors ${
                    activeFilter === cat
                      ? 'bg-primary text-primary-foreground'
                      : 'text-muted-foreground hover:bg-secondary hover:text-foreground'
                  }`}
                >
                  {cat}
                </button>
              ))}
            </div>

            {/* Results */}
            <div className="max-h-80 overflow-y-auto">
              {query.trim() && results.length > 0 && (
                <div className="p-2">
                  {results.map((result) => {
                    const Icon = result.categoryIcon
                    return (
                      <button
                        key={result.id}
                        onClick={() => handleResultClick(result.path)}
                        className="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg hover:bg-secondary transition-colors text-left group"
                      >
                        <div className={`h-9 w-9 rounded-lg ${result.bgColor} flex items-center justify-center shrink-0`}>
                          <Icon className={`h-4 w-4 ${result.iconColor}`} />
                        </div>
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-medium text-foreground truncate">{result.title}</p>
                          <div className="flex items-center gap-2 mt-0.5">
                            <span className="text-xs text-muted-foreground">{result.category}</span>
                            {result.description && (
                              <>
                                <span className="text-xs text-border">&bull;</span>
                                <span className="text-xs text-muted-foreground">{result.description}</span>
                              </>
                            )}
                          </div>
                        </div>
                        <ArrowRight className="h-4 w-4 text-muted-foreground opacity-0 group-hover:opacity-100 group-hover:translate-x-0.5 transition-all shrink-0" />
                      </button>
                    )
                  })}
                </div>
              )}

              {query.trim() && results.length === 0 && (
                <div className="py-10 text-center">
                  <Search className="h-8 w-8 mx-auto mb-2 text-muted-foreground/30" />
                  <p className="text-sm text-muted-foreground">Keine Ergebnisse fuer &ldquo;{query}&rdquo;</p>
                </div>
              )}

              {!query.trim() && (
                <div className="py-10 text-center">
                  <p className="text-sm text-muted-foreground">Tippe, um zu suchen</p>
                </div>
              )}
            </div>

            {/* Footer hints */}
            <div className="flex items-center gap-4 border-t border-border px-4 py-2 text-[10px] text-text-disabled">
              <span>
                <kbd className="px-1 py-0.5 bg-secondary border border-border rounded">↵</kbd> Oeffnen
              </span>
              <span>
                <kbd className="px-1 py-0.5 bg-secondary border border-border rounded">Esc</kbd> Schliessen
              </span>
              <span>
                <kbd className="px-1 py-0.5 bg-secondary border border-border rounded">Tab</kbd> Filter wechseln
              </span>
            </div>
          </div>
        </div>
      )}
    </>
  )
}

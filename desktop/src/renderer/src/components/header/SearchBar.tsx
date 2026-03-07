import { useState, useRef, useEffect, useCallback, useMemo } from 'react'
import {
  Search,
  CheckSquare,
  FileText,
  MessageSquare,
  Users,
  Mail,
  ArrowRight,
  X,
  Filter,
  Loader2,
  AlertCircle,
  ChevronDown,
  ChevronRight,
} from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import {
  useGlobalSearch,
  type GlobalSearchModule,
  type GlobalSearchResultItem,
} from '@/api/hooks/useGlobalSearch'
import { moduleHsl, moduleHslBg } from '@/components/layout/sidebar/nav-items'

// ---------------------------------------------------------------------------
// Module config — colors derived from nav-items.ts (single source of truth)
// ---------------------------------------------------------------------------

type ModuleKey = 'contacts' | 'files' | 'emails' | 'tasks' | 'messages'

interface ModuleConfig {
  label: string
  icon: typeof Users
  /** nav-items module ID for color lookup */
  colorId: string
  basePath: string
}

const MODULE_CONFIG: Record<string, ModuleConfig> = {
  crm: { label: 'Kontakte', icon: Users, colorId: 'contacts', basePath: '/kontakte' },
  documents: { label: 'Dateien', icon: FileText, colorId: 'documents', basePath: '/dokumente' },
  email: { label: 'E-Mails', icon: Mail, colorId: 'mail', basePath: '/email' },
  tasks: { label: 'Aufgaben', icon: CheckSquare, colorId: 'tasks', basePath: '/work/my-tasks' },
  messages: { label: 'Nachrichten', icon: MessageSquare, colorId: 'chat', basePath: '/chat' },
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function truncate(text: string, maxLength: number): string {
  if (text.length <= maxLength) return text
  return text.slice(0, maxLength).trimEnd() + '...'
}

function getResultDescription(module: string, item: GlobalSearchResultItem): string {
  switch (module) {
    case 'crm':
      return [item.company, item.email].filter(Boolean).join(' - ')
    case 'documents':
      return item.content_text
        ? truncate(item.content_text, 100)
        : item.description || ''
    case 'email':
      return item.subject
        ? truncate(item.subject, 100)
        : item.description || ''
    case 'tasks':
      return [item.project_name, item.status].filter(Boolean).join(' - ')
    case 'messages':
      return item.channel
        ? `#${item.channel} - ${truncate(item.description || '', 80)}`
        : item.description || ''
    default:
      return item.description || ''
  }
}

function getResultUrl(module: string, item: GlobalSearchResultItem): string {
  if (item.url) return item.url
  const config = MODULE_CONFIG[module]
  if (!config) return '/'
  return `${config.basePath}/${item.id}`
}

// ---------------------------------------------------------------------------
// Debounce hook
// ---------------------------------------------------------------------------

function useDebouncedValue<T>(value: T, delayMs: number): T {
  const [debouncedValue, setDebouncedValue] = useState(value)

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedValue(value), delayMs)
    return () => clearTimeout(timer)
  }, [value, delayMs])

  return debouncedValue
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

/**
 * Compact search trigger button that opens a modal search overlay.
 * Keyboard shortcut: Ctrl+K / Cmd+K.
 * Searches across all modules (CRM, Documents, Email, Tasks, Messages)
 * via the global search API with grouped results.
 */
export function SearchBar() {
  const [isOpen, setIsOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [activeFilter, setActiveFilter] = useState<string>('Alle')
  const [highlightedIndex, setHighlightedIndex] = useState(0)
  const [expandedModules, setExpandedModules] = useState<Set<string>>(new Set())
  const inputRef = useRef<HTMLInputElement>(null)
  const resultsRef = useRef<HTMLDivElement>(null)
  const navigate = useNavigate()

  const debouncedQuery = useDebouncedValue(query, 300)

  // Determine which modules to search based on active filter
  const searchModules = useMemo(() => {
    if (activeFilter === 'Alle') return undefined
    // Map German filter label back to module key
    const entry = Object.entries(MODULE_CONFIG).find(
      ([, config]) => config.label === activeFilter,
    )
    return entry ? [entry[0]] : undefined
  }, [activeFilter])

  const { data, isLoading, isError } = useGlobalSearch({
    query: debouncedQuery,
    modules: searchModules,
    limit: 5,
    enabled: isOpen,
  })

  // Build flat list of results for keyboard navigation
  const flatResults = useMemo(() => {
    if (!data?.modules) return []
    const items: Array<{ module: string; item: GlobalSearchResultItem }> = []
    for (const mod of data.modules) {
      if (!mod.results?.length) continue
      const maxVisible = expandedModules.has(mod.module) ? mod.results.length : Math.min(mod.results.length, 5)
      for (let i = 0; i < maxVisible; i++) {
        items.push({ module: mod.module, item: mod.results[i] })
      }
    }
    return items
  }, [data, expandedModules])

  // Build filter list from available modules
  const availableFilters = useMemo(() => {
    const filters: Array<{ key: string; label: string; count: number }> = [
      { key: 'Alle', label: 'Alle', count: 0 },
    ]
    let totalCount = 0

    if (data?.modules) {
      for (const mod of data.modules) {
        const config = MODULE_CONFIG[mod.module]
        if (config && mod.total > 0) {
          filters.push({
            key: config.label,
            label: `${config.label} (${mod.total})`,
            count: mod.total,
          })
          totalCount += mod.total
        }
      }
    }

    filters[0].count = totalCount
    if (totalCount > 0) {
      filters[0].label = `Alle (${totalCount})`
    }

    return filters
  }, [data])

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

  // Focus input when modal opens; reset state when closed
  useEffect(() => {
    if (isOpen) {
      setTimeout(() => inputRef.current?.focus(), 50)
    } else {
      setQuery('')
      setActiveFilter('Alle')
      setHighlightedIndex(0)
      setExpandedModules(new Set())
    }
  }, [isOpen])

  // Reset highlighted index when results change
  useEffect(() => {
    setHighlightedIndex(0)
  }, [flatResults.length])

  const handleResultClick = useCallback(
    (module: string, item: GlobalSearchResultItem) => {
      navigate(getResultUrl(module, item))
      setIsOpen(false)
    },
    [navigate],
  )

  const toggleModuleExpanded = useCallback((module: string) => {
    setExpandedModules((prev) => {
      const next = new Set(prev)
      if (next.has(module)) {
        next.delete(module)
      } else {
        next.add(module)
      }
      return next
    })
  }, [])

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setHighlightedIndex((prev) =>
        prev < flatResults.length - 1 ? prev + 1 : 0,
      )
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setHighlightedIndex((prev) =>
        prev > 0 ? prev - 1 : flatResults.length - 1,
      )
    } else if (e.key === 'Enter' && flatResults.length > 0) {
      e.preventDefault()
      const selected = flatResults[highlightedIndex]
      if (selected) {
        handleResultClick(selected.module, selected.item)
      }
    } else if (e.key === 'Escape') {
      setIsOpen(false)
    }
  }

  // Scroll highlighted item into view
  useEffect(() => {
    if (!resultsRef.current) return
    const highlighted = resultsRef.current.querySelector(
      `[data-result-index="${highlightedIndex}"]`,
    )
    if (highlighted) {
      highlighted.scrollIntoView({ block: 'nearest' })
    }
  }, [highlightedIndex])

  return (
    <div className="relative">
      {/* Compact trigger — uses w-full so parent controls width (Main's layout) */}
      <button
        onClick={() => setIsOpen(true)}
        className="flex w-full items-center gap-2 rounded-lg border border-border bg-secondary/50 px-3 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
      >
        <Search className="h-3.5 w-3.5" />
        <span className="hidden sm:inline">Suchen...</span>
        <kbd className="hidden md:inline-flex h-5 items-center rounded border border-border bg-card px-1.5 text-[10px] font-medium text-text-disabled">
          Ctrl+K
        </kbd>
      </button>

      {/* Dropdown overlay */}
      {isOpen && (
        <>
          {/* Invisible backdrop to catch outside clicks */}
          <div
            className="fixed inset-0 z-40"
            onClick={() => setIsOpen(false)}
          />

          {/* Search dropdown */}
          <div className="absolute top-full left-0 z-50 mt-1 w-full min-w-[24rem] rounded-xl border border-border bg-card shadow-[var(--shadow-large)] overflow-hidden">
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
              {isLoading && debouncedQuery.length >= 2 && (
                <Loader2 className="h-4 w-4 shrink-0 animate-spin text-muted-foreground" />
              )}
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
              {availableFilters.map((filter) => (
                <button
                  key={filter.key}
                  onClick={() => setActiveFilter(filter.key)}
                  className={`shrink-0 rounded-full px-2.5 py-1 text-xs transition-colors ${
                    activeFilter === filter.key
                      ? 'bg-primary text-primary-foreground'
                      : 'text-muted-foreground hover:bg-secondary hover:text-foreground'
                  }`}
                >
                  {filter.label}
                </button>
              ))}
            </div>

            {/* Results */}
            <div ref={resultsRef} className="max-h-80 overflow-y-auto">
              {/* Grouped results */}
              {debouncedQuery.length >= 2 &&
                !isLoading &&
                !isError &&
                data?.modules &&
                data.modules.some((m) => m.results?.length > 0) && (
                  <div className="p-2">
                    {data.modules.map((mod) => (
                      <ModuleGroup
                        key={mod.module}
                        module={mod}
                        expandedModules={expandedModules}
                        onToggleExpand={toggleModuleExpanded}
                        highlightedIndex={highlightedIndex}
                        flatResults={flatResults}
                        onResultClick={handleResultClick}
                      />
                    ))}
                  </div>
                )}

              {/* Loading skeleton */}
              {isLoading && debouncedQuery.length >= 2 && (
                <div className="p-4 space-y-3">
                  {[1, 2, 3].map((i) => (
                    <div key={i} className="flex items-center gap-3 animate-pulse">
                      <div className="h-9 w-9 rounded-lg bg-secondary" />
                      <div className="flex-1 space-y-1.5">
                        <div className="h-3.5 w-2/3 rounded bg-secondary" />
                        <div className="h-3 w-1/3 rounded bg-secondary" />
                      </div>
                    </div>
                  ))}
                </div>
              )}

              {/* Error state */}
              {isError && debouncedQuery.length >= 2 && (
                <div className="py-10 text-center">
                  <AlertCircle className="h-8 w-8 mx-auto mb-2 text-destructive/50" />
                  <p className="text-sm text-muted-foreground">
                    Suche fehlgeschlagen. Versuche es erneut.
                  </p>
                </div>
              )}

              {/* No results */}
              {debouncedQuery.length >= 2 &&
                !isLoading &&
                !isError &&
                data?.modules &&
                !data.modules.some((m) => m.results?.length > 0) && (
                  <div className="py-10 text-center">
                    <Search className="h-8 w-8 mx-auto mb-2 text-muted-foreground/30" />
                    <p className="text-sm text-muted-foreground">
                      Keine Ergebnisse fuer &ldquo;{debouncedQuery}&rdquo;
                    </p>
                  </div>
                )}

              {/* Empty state */}
              {debouncedQuery.length < 2 && (
                <div className="py-10 text-center">
                  <p className="text-sm text-muted-foreground">
                    Tippe, um zu suchen
                  </p>
                </div>
              )}
            </div>

            {/* Footer hints */}
            <div className="flex items-center gap-4 border-t border-border px-4 py-2 text-[10px] text-text-disabled">
              <span>
                <kbd className="px-1 py-0.5 bg-secondary border border-border rounded">
                  ↑↓
                </kbd>{' '}
                Navigieren
              </span>
              <span>
                <kbd className="px-1 py-0.5 bg-secondary border border-border rounded">
                  ↵
                </kbd>{' '}
                Oeffnen
              </span>
              <span>
                <kbd className="px-1 py-0.5 bg-secondary border border-border rounded">
                  Esc
                </kbd>{' '}
                Schliessen
              </span>
            </div>
          </div>
        </>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Module group sub-component
// ---------------------------------------------------------------------------

function ModuleGroup({
  module,
  expandedModules,
  onToggleExpand,
  highlightedIndex,
  flatResults,
  onResultClick,
}: {
  module: GlobalSearchModule
  expandedModules: Set<string>
  onToggleExpand: (module: string) => void
  highlightedIndex: number
  flatResults: Array<{ module: string; item: GlobalSearchResultItem }>
  onResultClick: (module: string, item: GlobalSearchResultItem) => void
}) {
  if (!module.results?.length) return null

  const config = MODULE_CONFIG[module.module]
  if (!config) return null

  const isExpanded = expandedModules.has(module.module)
  const visibleResults = isExpanded
    ? module.results
    : module.results.slice(0, 5)
  const hasMore = module.results.length > 5 && !isExpanded

  const ModuleIcon = config.icon

  return (
    <div className="mb-2 last:mb-0">
      {/* Module header */}
      <button
        onClick={() => onToggleExpand(module.module)}
        className="flex w-full items-center gap-2 px-3 py-1.5 text-xs font-medium text-muted-foreground hover:text-foreground transition-colors rounded-md hover:bg-secondary/50"
      >
        {isExpanded ? (
          <ChevronDown className="h-3 w-3" />
        ) : (
          <ChevronRight className="h-3 w-3" />
        )}
        <ModuleIcon className="h-3.5 w-3.5" style={{ color: moduleHsl(config.colorId) }} />
        <span>
          {config.label} ({module.total})
        </span>
      </button>

      {/* Results */}
      {visibleResults.map((item) => {
        // Find this item's flat index for highlight tracking
        const flatIndex = flatResults.findIndex(
          (fr) => fr.module === module.module && fr.item.id === item.id,
        )
        const isHighlighted = flatIndex === highlightedIndex

        return (
          <button
            key={item.id}
            data-result-index={flatIndex}
            onClick={() => onResultClick(module.module, item)}
            className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg transition-colors text-left group ${
              isHighlighted
                ? 'bg-secondary'
                : 'hover:bg-secondary'
            }`}
          >
            <div
              className="h-9 w-9 rounded-lg flex items-center justify-center shrink-0"
              style={{ backgroundColor: moduleHslBg(config.colorId) }}
            >
              <ModuleIcon className="h-4 w-4" style={{ color: moduleHsl(config.colorId) }} />
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium text-foreground truncate">
                {item.title || item.file_name || item.subject || item.id}
              </p>
              <div className="flex items-center gap-2 mt-0.5">
                <span className="text-xs text-muted-foreground truncate">
                  {truncate(getResultDescription(module.module, item), 100)}
                </span>
              </div>
            </div>
            <ArrowRight className="h-4 w-4 text-muted-foreground opacity-0 group-hover:opacity-100 group-hover:translate-x-0.5 transition-all shrink-0" />
          </button>
        )
      })}

      {/* Show more */}
      {hasMore && (
        <button
          onClick={() => onToggleExpand(module.module)}
          className="w-full text-center py-1.5 text-xs text-primary hover:text-primary/80 transition-colors"
        >
          Mehr anzeigen ({module.results.length - 5} weitere)
        </button>
      )}
    </div>
  )
}

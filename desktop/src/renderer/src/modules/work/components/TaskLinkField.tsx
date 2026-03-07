/**
 * Search-and-link component for CRM entity linking on tasks.
 *
 * Features:
 * - "Verknuepfung hinzufügen" button opens search popover
 * - Entity type tabs: Alle, Kontakte, Unternehmen, Deals
 * - CRM search via /api/v1/search with type filtering
 * - Context-aware auto-suggest based on task title/description
 * - Existing links displayed as clickable chips with remove button
 */
import { useState, useEffect, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Plus,
  X,
  Search,
  User,
  Building2,
  TrendingUp,
  ExternalLink,
} from 'lucide-react'
import { cn } from '@/lib'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { useCRMSearch } from '@/api/hooks/useSearch'
import {
  useTaskEntityLinks,
  useLinkEntity,
  useUnlinkEntity,
} from '@/api/hooks/useTasks'
import type { components } from '@/api/types'

type TaskEntityLink = components['schemas']['TaskEntityLinkResponse']

const ENTITY_TYPE_CONFIG: Record<
  string,
  {
    label: string
    icon: React.ComponentType<{ className?: string }>
    route: (id: string) => string
  }
> = {
  contact: {
    label: 'Kontakt',
    icon: User,
    route: (id) => `/crm/contacts/${id}`,
  },
  company: {
    label: 'Unternehmen',
    icon: Building2,
    route: (id) => `/crm/companies/${id}`,
  },
  deal: {
    label: 'Deal',
    icon: TrendingUp,
    route: (id) => `/crm/deals/${id}`,
  },
}

const ENTITY_TABS = [
  { key: '', label: 'Alle' },
  { key: 'contact', label: 'Kontakte' },
  { key: 'company', label: 'Unternehmen' },
  { key: 'deal', label: 'Deals' },
] as const

interface TaskLinkFieldProps {
  taskId: string
  taskTitle?: string
  taskDescription?: string
}

export default function TaskLinkField({
  taskId,
  taskTitle,
  taskDescription,
}: TaskLinkFieldProps) {
  const navigate = useNavigate()
  const [searchOpen, setSearchOpen] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [debouncedQuery, setDebouncedQuery] = useState('')
  const [activeTab, setActiveTab] = useState('')
  const [dismissedSuggestions, setDismissedSuggestions] = useState<Set<string>>(
    new Set()
  )

  const { data: linksData } = useTaskEntityLinks(taskId)
  const links: TaskEntityLink[] = useMemo(() => linksData?.links ?? [], [linksData?.links])

  const linkEntity = useLinkEntity()
  const unlinkEntity = useUnlinkEntity()

  // Debounce search query
   
  useEffect(() => {
    const timer = setTimeout(() => setDebouncedQuery(searchQuery), 300)
    return () => clearTimeout(timer)
  }, [searchQuery])

  // CRM search
  const searchTypes = activeTab || 'contact,company,deal'
  const { data: searchData, isLoading: searchLoading } = useCRMSearch(
    debouncedQuery,
    { types: searchTypes, limit: 10 }
  )
  const searchResults = searchData?.results ?? []

  // Filter out already linked entities
  const linkedEntityIds = useMemo(
    () => new Set(links.map((l) => l.entity_id)),
    [links]
  )
  const filteredResults = searchResults.filter(
    (r) => r.id && !linkedEntityIds.has(r.id)
  )

  // Context-aware auto-suggest: match task title/description against CRM entities
  const contextText = `${taskTitle ?? ''} ${taskDescription ?? ''}`.trim()
  const { data: suggestData } = useCRMSearch(
    contextText.length >= 3 ? contextText.substring(0, 50) : '',
    { types: 'contact,company,deal', limit: 5 }
  )
  // eslint-disable-next-line react-hooks/preserve-manual-memoization -- deps are correct
  const suggestions = useMemo(() => {
    if (!suggestData?.results) return []
    return suggestData.results.filter(
      (r) =>
        r.id &&
        !linkedEntityIds.has(r.id) &&
        !dismissedSuggestions.has(r.id)
    )
  }, [suggestData?.results, linkedEntityIds, dismissedSuggestions])

  function handleLink(
    entityType: string,
    entityId: string
  ) {
    if (!entityType || !entityId) return
    linkEntity.mutate({
      taskId,
      entity_type: entityType as TaskEntityLink['entity_type'] & string,
      entity_id: entityId,
    })
    setSearchQuery('')
    setSearchOpen(false)
  }

  function handleUnlink(linkId: string) {
    unlinkEntity.mutate({ linkId, taskId })
  }

  function handleDismissSuggestion(entityId: string) {
    setDismissedSuggestions((prev) => new Set(prev).add(entityId))
  }

  function navigateToEntity(entityType: string, entityId: string) {
    const config = ENTITY_TYPE_CONFIG[entityType]
    if (config) {
      navigate(config.route(entityId))
    }
  }

  return (
    <div className="space-y-2">
      {/* Context-aware auto-suggestions */}
      {suggestions.length > 0 && (
        <div className="rounded-md border border-blue-200 bg-blue-50/50 px-3 py-2 space-y-1.5">
          <p className="text-xs text-blue-700 font-medium">
            Meinten Sie:
          </p>
          {suggestions.slice(0, 3).map((suggestion) => {
            const config =
              ENTITY_TYPE_CONFIG[suggestion.entity_type ?? '']
            const Icon = config?.icon ?? ExternalLink
            return (
              <div
                key={suggestion.id}
                className="flex items-center justify-between gap-2"
              >
                <button
                  type="button"
                  className="flex items-center gap-1.5 text-xs text-blue-700 hover:text-blue-900 transition-colors"
                  onClick={() =>
                    handleLink(
                      suggestion.entity_type ?? '',
                      suggestion.id ?? ''
                    )
                  }
                >
                  <Icon className="h-3 w-3" />
                  <span className="truncate max-w-[160px]">
                    {suggestion.title}
                  </span>
                  <span className="text-blue-500">
                    ({config?.label ?? suggestion.entity_type})
                  </span>
                </button>
                <button
                  type="button"
                  className="text-blue-400 hover:text-blue-600 transition-colors"
                  onClick={() =>
                    handleDismissSuggestion(suggestion.id ?? '')
                  }
                  title="Vorschlag ausblenden"
                >
                  <X className="h-3 w-3" />
                </button>
              </div>
            )
          })}
        </div>
      )}

      {/* Existing links as chips */}
      {links.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {links.map((link) => {
            const config =
              ENTITY_TYPE_CONFIG[link.entity_type ?? '']
            const Icon = config?.icon ?? ExternalLink
            return (
              <span
                key={link.id}
                className="inline-flex items-center gap-1 rounded-md border border-border bg-background px-2 py-0.5 text-xs group"
              >
                <button
                  type="button"
                  className="flex items-center gap-1 hover:text-primary transition-colors"
                  onClick={() =>
                    navigateToEntity(
                      link.entity_type ?? '',
                      link.entity_id ?? ''
                    )
                  }
                  title={`${config?.label ?? link.entity_type} öffnen`}
                >
                  <Icon className="h-3 w-3 text-muted-foreground" />
                  <span className="truncate max-w-[120px]">
                    {link.entity_display_name ?? 'Verknuepfung'}
                  </span>
                </button>
                <button
                  type="button"
                  className="ml-0.5 text-muted-foreground hover:text-destructive opacity-0 group-hover:opacity-100 transition-all"
                  onClick={() => link.id && handleUnlink(link.id)}
                  title="Verknuepfung entfernen"
                >
                  <X className="h-3 w-3" />
                </button>
              </span>
            )
          })}
        </div>
      )}

      {/* Add link button + search popover */}
      <Popover open={searchOpen} onOpenChange={setSearchOpen}>
        <PopoverTrigger asChild>
          <Button
            variant="ghost"
            size="sm"
            className="h-7 gap-1 text-xs text-muted-foreground"
          >
            <Plus className="h-3.5 w-3.5" />
            Verknuepfung hinzufügen
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-72 p-0" align="start">
          {/* Search input */}
          <div className="p-2 border-b border-border">
            <div className="relative">
              <Search className="absolute left-2 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder="Kontakt, Unternehmen oder Deal suchen..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-7 h-8 text-xs"
                autoFocus
              />
            </div>
          </div>

          {/* Entity type tabs */}
          <div className="flex items-center gap-0.5 border-b border-border px-2 py-1">
            {ENTITY_TABS.map((tab) => (
              <button
                key={tab.key}
                type="button"
                className={cn(
                  'rounded px-2 py-0.5 text-xs font-medium transition-colors',
                  activeTab === tab.key
                    ? 'bg-secondary text-secondary-foreground'
                    : 'text-muted-foreground hover:text-foreground'
                )}
                onClick={() => setActiveTab(tab.key)}
              >
                {tab.label}
              </button>
            ))}
          </div>

          {/* Search results */}
          <div className="max-h-48 overflow-y-auto py-1">
            {searchLoading && debouncedQuery.length >= 2 ? (
              <p className="px-3 py-2 text-xs text-muted-foreground">
                Suche...
              </p>
            ) : debouncedQuery.length < 2 ? (
              <p className="px-3 py-2 text-xs text-muted-foreground">
                Mindestens 2 Zeichen eingeben...
              </p>
            ) : filteredResults.length === 0 ? (
              <p className="px-3 py-2 text-xs text-muted-foreground">
                Keine Ergebnisse
              </p>
            ) : (
              filteredResults.map((result) => {
                const config =
                  ENTITY_TYPE_CONFIG[result.entity_type ?? '']
                const Icon = config?.icon ?? ExternalLink
                return (
                  <button
                    key={result.id}
                    type="button"
                    className="flex w-full items-center gap-2 px-3 py-1.5 text-xs hover:bg-accent transition-colors"
                    onClick={() =>
                      handleLink(
                        result.entity_type ?? '',
                        result.id ?? ''
                      )
                    }
                  >
                    <Icon className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                    <div className="flex-1 min-w-0 text-left">
                      <p className="truncate font-medium">
                        {result.title}
                      </p>
                      {result.subtitle && (
                        <p className="truncate text-muted-foreground">
                          {result.subtitle}
                        </p>
                      )}
                    </div>
                    <span className="shrink-0 text-muted-foreground">
                      {config?.label ?? result.entity_type}
                    </span>
                  </button>
                )
              })
            )}
          </div>
        </PopoverContent>
      </Popover>
    </div>
  )
}

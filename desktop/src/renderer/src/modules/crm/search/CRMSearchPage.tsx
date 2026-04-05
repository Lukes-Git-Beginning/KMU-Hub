/**
 * CRM search page for searching across contacts, companies, and deals.
 *
 * Large search input with debounced query (300ms, min 2 chars).
 * Results grouped by entity type with navigation to detail pages.
 */
import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Search, Users, Building2, TrendingUp, Activity } from 'lucide-react'
import { useCRMSearch } from '@/api/hooks/useSearch'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Button } from '@/components/ui/button'

const entityTypeConfig: Record<
  string,
  { labelKey: string; icon: typeof Users; route: string; color: string }
> = {
  contact: {
    labelKey: 'crm.search.entityContact',
    icon: Users,
    route: '/crm/contacts',
    color: 'text-blue-600',
  },
  company: {
    labelKey: 'crm.search.entityCompany',
    icon: Building2,
    route: '/crm/companies',
    color: 'text-purple-600',
  },
  deal: {
    labelKey: 'crm.search.entityDeal',
    icon: TrendingUp,
    route: '/crm/deals',
    color: 'text-green-600',
  },
  activity: {
    labelKey: 'crm.search.entityActivity',
    icon: Activity,
    route: '/crm/activities',
    color: 'text-orange-600',
  },
}

export default function CRMSearchPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [query, setQuery] = useState('')
  const [debouncedQuery, setDebouncedQuery] = useState('')

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedQuery(query)
    }, 300)
    return () => clearTimeout(timer)
  }, [query])

  const { data, isLoading, error, refetch } = useCRMSearch(debouncedQuery, {
    limit: 50,
  })

  const results = data?.results ?? []

  // Group results by entity type
  const grouped = new Map<string, typeof results>()
  for (const result of results) {
    const type = result.entity_type ?? 'unknown'
    if (!grouped.has(type)) {
      grouped.set(type, [])
    }
    grouped.get(type)!.push(result)
  }

  function navigateToResult(result: (typeof results)[0]) {
    const config = entityTypeConfig[result.entity_type ?? '']
    if (config && result.id) {
      navigate(`${config.route}/${result.id}`)
    }
  }

  return (
    <div className="p-6 space-y-6">
      <h1 className="text-2xl font-semibold text-foreground">{t('crm.search.title')}</h1>

      {/* Search input */}
      <div className="relative max-w-lg">
        <Search className="absolute left-4 top-1/2 h-5 w-5 -translate-y-1/2 text-muted-foreground" />
        <Input
          placeholder={t('crm.search.placeholder')}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          className="h-12 pl-12 text-base"
          autoFocus
        />
      </div>

      {/* Results */}
      {error ? (
        <div className="text-center py-8">
          <p className="text-lg font-semibold text-foreground">
            {t('crm.search.error')}
          </p>
          <p className="mt-2 text-sm text-muted-foreground">
            {error instanceof Error
              ? error.message
              : t('crm.error.unexpected')}
          </p>
          <Button variant="outline" className="mt-4" onClick={() => refetch()}>
            {t('common.retry')}
          </Button>
        </div>
      ) : debouncedQuery.length < 2 ? (
        <div className="flex flex-col items-center justify-center py-16">
          <Search className="h-12 w-12 text-muted-foreground" />
          <p className="mt-4 text-lg font-medium text-foreground">
            {t('crm.search.heading')}
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            {t('crm.search.hint')}
          </p>
        </div>
      ) : isLoading ? (
        <div className="space-y-4">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-16 w-full" />
          ))}
        </div>
      ) : results.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16">
          <Search className="h-12 w-12 text-muted-foreground" />
          <p className="mt-4 text-lg font-medium text-foreground">
            {t('common.noResults')}
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            {t('crm.search.noResults', { query: debouncedQuery })}
          </p>
        </div>
      ) : (
        <div className="space-y-6">
          {Array.from(grouped.entries()).map(([type, typeResults]) => {
            const config = entityTypeConfig[type]
            if (!config) return null
            const Icon = config.icon

            return (
              <div key={type}>
                <div className="flex items-center gap-2 mb-3">
                  <Icon className={`h-4 w-4 ${config.color}`} />
                  <h2 className="text-sm font-semibold text-foreground">
                    {t(config.labelKey)} ({typeResults.length})
                  </h2>
                </div>
                <div className="space-y-2">
                  {typeResults.map((result) => (
                    <div
                      key={`${result.entity_type}-${result.id}`}
                      className="flex items-center gap-4 rounded-lg border border-border p-3 cursor-pointer transition-colors hover:bg-accent/50"
                      onClick={() => navigateToResult(result)}
                    >
                      <Icon
                        className={`h-5 w-5 shrink-0 ${config.color}`}
                      />
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium truncate">
                          {result.title}
                        </p>
                        {result.subtitle && (
                          <p className="text-xs text-muted-foreground truncate">
                            {result.subtitle}
                          </p>
                        )}
                      </div>
                      <Badge variant="outline" className="text-xs shrink-0">
                        {t(config.labelKey)}
                      </Badge>
                    </div>
                  ))}
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

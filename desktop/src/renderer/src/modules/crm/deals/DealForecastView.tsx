/**
 * Weighted pipeline forecast.
 *
 * Aggregates open-stage deals into a weighted revenue forecast (value ×
 * stage probability), broken down per stage and per expected-close month.
 * Read-only companion to the list and pipeline views.
 */
import { useTranslation } from 'react-i18next'
import { usePipelineStages } from '@/api/hooks/usePipelineStages'
import { useDeals } from '@/api/hooks/useDeals'
import { Skeleton } from '@/components/ui/skeleton'
import { Button } from '@/components/ui/button'

type Deal = NonNullable<ReturnType<typeof useDeals>['data']>['deals'][number]
type Stage = NonNullable<ReturnType<typeof usePipelineStages>['data']>['stages'][number]

function formatCurrency(value: number, currency = 'EUR'): string {
  return new Intl.NumberFormat('de-DE', {
    style: 'currency',
    currency,
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(value)
}

function stageIsWon(s: Stage): boolean {
  const r = s as Record<string, unknown>
  return Boolean(r.isWon ?? r.is_won)
}
function stageIsLost(s: Stage): boolean {
  const r = s as Record<string, unknown>
  return Boolean(r.isLost ?? r.is_lost)
}

export default function DealForecastView() {
  const { t } = useTranslation()
  const { data: stagesData, isLoading: sl, error: se, refetch: rs } = usePipelineStages()
  const { data: dealsData, isLoading: dl, error: de, refetch: rd } = useDeals({ page_size: 200 })

  const stages = stagesData?.stages ?? []
  const deals = dealsData?.deals ?? []
  const isLoading = sl || dl
  const error = se || de

  if (error) {
    return (
      <div className="flex items-center justify-center py-16">
        <div className="text-center">
          <p className="text-lg font-semibold text-foreground">{t('crm.deals.pipelineLoadError')}</p>
          <Button variant="outline" className="mt-4" onClick={() => { rs(); rd() }}>
            {t('common.retry')}
          </Button>
        </div>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          {Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="h-24 w-full" />)}
        </div>
        <Skeleton className="h-64 w-full" />
      </div>
    )
  }

  const openStages = stages.filter((s) => !stageIsWon(s) && !stageIsLost(s))
  const wonStage = stages.find(stageIsWon)

  const dealsByStage = new Map<string, Deal[]>()
  for (const s of stages) dealsByStage.set(s.id ?? '', [])
  for (const d of deals) {
    const list = dealsByStage.get(d.stageId ?? '')
    if (list) list.push(d)
  }

  // Per-open-stage aggregation
  const stageRows = openStages.map((s) => {
    const list = dealsByStage.get(s.id ?? '') ?? []
    const value = list.reduce((sum, d) => sum + (d.value ?? 0), 0)
    const prob = s.probability ?? 0
    return { stage: s, count: list.length, value, weighted: (value * prob) / 100, prob }
  })

  const openValue = stageRows.reduce((sum, r) => sum + r.value, 0)
  const weightedForecast = stageRows.reduce((sum, r) => sum + r.weighted, 0)
  const wonValue = (dealsByStage.get(wonStage?.id ?? '') ?? []).reduce((sum, d) => sum + (d.value ?? 0), 0)
  const maxWeighted = Math.max(1, ...stageRows.map((r) => r.weighted))

  // By expected-close month (open deals only, weighted by stage probability)
  const probByStage = new Map(stages.map((s) => [s.id ?? '', s.probability ?? 0]))
  const monthMap = new Map<string, number>()
  for (const r of stageRows) {
    for (const d of dealsByStage.get(r.stage.id ?? '') ?? []) {
      if (!d.expectedCloseDate) continue
      const key = d.expectedCloseDate.slice(0, 7) // YYYY-MM
      const w = ((d.value ?? 0) * (probByStage.get(d.stageId ?? '') ?? 0)) / 100
      monthMap.set(key, (monthMap.get(key) ?? 0) + w)
    }
  }
  const monthRows = Array.from(monthMap.entries())
    .sort(([a], [b]) => a.localeCompare(b))
    .slice(0, 6)
    .map(([key, weighted]) => {
      const [y, m] = key.split('-')
      const label = new Date(Number(y), Number(m) - 1, 1).toLocaleDateString('de-DE', { month: 'short', year: 'numeric' })
      return { key, label, weighted }
    })
  const maxMonth = Math.max(1, ...monthRows.map((r) => r.weighted))

  if (openValue === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-16">
        <p className="text-sm text-muted-foreground">{t('crm.deals.forecast.empty')}</p>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Stat cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <div className="rounded-xl border border-border bg-card p-4">
          <p className="text-xs text-muted-foreground">{t('crm.deals.forecast.openVolume')}</p>
          <p className="mt-1 text-2xl font-semibold text-foreground">{formatCurrency(openValue)}</p>
        </div>
        <div className="rounded-xl border border-primary/30 bg-primary/5 p-4">
          <p className="text-xs text-muted-foreground">{t('crm.deals.forecast.weighted')}</p>
          <p className="mt-1 text-2xl font-semibold text-primary">{formatCurrency(weightedForecast)}</p>
        </div>
        <div className="rounded-xl border border-border bg-card p-4">
          <p className="text-xs text-muted-foreground">{t('crm.deals.forecast.wonValue')}</p>
          <p className="mt-1 text-2xl font-semibold text-success">{formatCurrency(wonValue)}</p>
        </div>
      </div>

      {/* By stage */}
      <div className="rounded-xl border border-border bg-card p-4">
        <h3 className="mb-4 text-sm font-semibold text-foreground">{t('crm.deals.forecast.byStage')}</h3>
        <div className="space-y-3">
          {stageRows.map((r) => (
            <div key={r.stage.id}>
              <div className="mb-1 flex items-center justify-between text-sm">
                <span className="flex items-center gap-2">
                  <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: r.stage.color || '#94a3b8' }} />
                  <span className="font-medium text-foreground">{r.stage.name}</span>
                  <span className="text-xs text-muted-foreground">
                    {t('crm.deals.dealCount', { count: r.count })} · {r.prob}%
                  </span>
                </span>
                <span className="font-semibold text-foreground">{formatCurrency(r.weighted)}</span>
              </div>
              <div className="h-2 w-full overflow-hidden rounded-full bg-secondary">
                <div
                  className="h-full rounded-full transition-all"
                  style={{ width: `${(r.weighted / maxWeighted) * 100}%`, backgroundColor: r.stage.color || '#94a3b8' }}
                />
              </div>
              <p className="mt-0.5 text-[11px] text-muted-foreground">
                {t('crm.deals.value')}: {formatCurrency(r.value)}
              </p>
            </div>
          ))}
        </div>
      </div>

      {/* By month */}
      {monthRows.length > 0 && (
        <div className="rounded-xl border border-border bg-card p-4">
          <h3 className="mb-4 text-sm font-semibold text-foreground">{t('crm.deals.forecast.byMonth')}</h3>
          <div className="space-y-3">
            {monthRows.map((r) => (
              <div key={r.key}>
                <div className="mb-1 flex items-center justify-between text-sm">
                  <span className="font-medium text-foreground">{r.label}</span>
                  <span className="font-semibold text-foreground">{formatCurrency(r.weighted)}</span>
                </div>
                <div className="h-2 w-full overflow-hidden rounded-full bg-secondary">
                  <div
                    className="h-full rounded-full bg-primary transition-all"
                    style={{ width: `${(r.weighted / maxMonth) * 100}%` }}
                  />
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

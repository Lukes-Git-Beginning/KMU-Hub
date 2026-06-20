import { useTranslation } from 'react-i18next'
import { BarChart3, Loader2 } from 'lucide-react'
import type { BuilderQueryConfig, ReportResult } from '@/api/berichte-types'
import type { ReportSource } from '../../report-sources/types'
import { fieldByKey } from '../../report-sources/types'
import { ChartRenderer } from '../charts/ChartRenderer'

interface LivePreviewProps {
  query: BuilderQueryConfig | null
  result: ReportResult | undefined
  source: ReportSource | undefined
  isLoading: boolean
  isError: boolean
}

const AGG_LABEL: Record<string, string> = {
  count: 'Anzahl',
  sum: 'Summe',
  avg: 'Durchschnitt',
  min: 'Minimum',
  max: 'Maximum',
}

/** Human-readable "Summe von Umsatz nach Monat" sentence (aggregation transparency). */
function describe(query: BuilderQueryConfig, source: ReportSource | undefined): string {
  const measure = query.measures[0]
  const dim = query.dimensions[0]
  const measureLabel = measure
    ? measure.agg === 'count'
      ? AGG_LABEL.count
      : `${AGG_LABEL[measure.agg]} ${fieldByKey(source ?? ({} as ReportSource), measure.field)?.label ?? ''}`
    : 'Anzahl'
  const dimLabel = dim ? fieldByKey(source ?? ({} as ReportSource), dim)?.label : undefined
  return dimLabel ? `${measureLabel} nach ${dimLabel}` : measureLabel
}

export function LivePreview({ query, result, source, isLoading, isError }: LivePreviewProps) {
  const { t } = useTranslation()

  return (
    <div className="flex h-full flex-col rounded-xl border border-border bg-card">
      <div className="flex items-center justify-between border-b border-border px-5 py-3">
        <div className="min-w-0">
          <p className="text-sm font-medium text-foreground">
            {t('berichte.builder.preview.title', { defaultValue: 'Live-Vorschau' })}
          </p>
          {query && (
            <p className="truncate text-xs text-muted-foreground">{describe(query, source)}</p>
          )}
        </div>
        {isLoading && <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />}
      </div>

      <div className="flex-1 p-5">
        {!query ? (
          <div className="flex h-full min-h-[300px] flex-col items-center justify-center gap-3 text-center">
            <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-secondary">
              <BarChart3 className="h-6 w-6 text-muted-foreground" />
            </div>
            <p className="max-w-xs text-sm text-muted-foreground">
              {t('berichte.builder.preview.empty', {
                defaultValue: 'Wähle eine Datenquelle und mindestens ein Feld, um die Vorschau zu sehen.',
              })}
            </p>
          </div>
        ) : isError ? (
          <div className="flex h-full min-h-[300px] items-center justify-center">
            <p className="text-sm text-destructive">
              {t('berichte.builder.preview.error', { defaultValue: 'Vorschau konnte nicht geladen werden.' })}
            </p>
          </div>
        ) : result ? (
          <ChartRenderer result={result} viz={query.viz} options={query.options} height={360} />
        ) : (
          <div className="flex h-full min-h-[300px] items-center justify-center">
            <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          </div>
        )}
      </div>
    </div>
  )
}

import { useTranslation } from 'react-i18next'
import { Sigma } from 'lucide-react'
import type { AggregationFn, ReportMeasure } from '@/api/berichte-types'
import type { ReportSource } from '../../report-sources/types'
import { fieldByKey } from '../../report-sources/types'

interface SummarizeBlockProps {
  source: ReportSource
  measures: ReportMeasure[]
  dimensions: string[]
  onChangeAgg: (field: string, agg: AggregationFn) => void
}

const AGG_OPTIONS: { value: AggregationFn; label: string }[] = [
  { value: 'sum', label: 'Summe' },
  { value: 'avg', label: 'Durchschnitt' },
  { value: 'count', label: 'Anzahl' },
  { value: 'min', label: 'Minimum' },
  { value: 'max', label: 'Maximum' },
]

export function SummarizeBlock({ source, measures, dimensions, onChangeAgg }: SummarizeBlockProps) {
  const { t } = useTranslation()

  return (
    <div>
      <div className="mb-1.5 flex items-center gap-1.5">
        <Sigma className="h-3.5 w-3.5 text-primary" />
        <label className="text-xs font-medium text-muted-foreground">
          {t('berichte.builder.summarize.label', { defaultValue: 'Zusammenfassen' })}
        </label>
      </div>

      {measures.length === 0 ? (
        <p className="text-[11px] text-muted-foreground/70">
          {dimensions.length > 0
            ? t('berichte.builder.summarize.countHint', {
                defaultValue: 'Ohne Kennzahl wird die Anzahl der Datensätze gezählt.',
              })
            : t('berichte.builder.summarize.empty', {
                defaultValue: 'Wähle eine Kennzahl, um sie zusammenzufassen.',
              })}
        </p>
      ) : (
        <div className="space-y-1.5">
          {measures.map((m) => {
            const field = fieldByKey(source, m.field)
            return (
              <div
                key={m.field}
                className="flex items-center gap-2 rounded-lg border border-border-muted bg-primary-light/40 px-2 py-1.5"
              >
                <select
                  value={m.agg}
                  onChange={(e) => onChangeAgg(m.field, e.target.value as AggregationFn)}
                  className="shrink-0 rounded-md border border-border bg-card px-2 py-1 text-xs font-medium text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                >
                  {AGG_OPTIONS.map((o) => (
                    <option key={o.value} value={o.value}>
                      {o.label}
                    </option>
                  ))}
                </select>
                <span className="truncate text-xs text-muted-foreground">
                  {t('berichte.builder.summarize.of', { defaultValue: 'von' })}{' '}
                  <span className="text-foreground">{field?.label ?? m.field}</span>
                </span>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

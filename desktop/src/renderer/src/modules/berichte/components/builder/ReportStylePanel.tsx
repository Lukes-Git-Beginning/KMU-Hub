import { useTranslation } from 'react-i18next'
import { Palette } from 'lucide-react'
import type { ReportMeasure, ReportViewOptions, VisualizationType } from '@/api/berichte-types'
import type { ReportSource } from '../../report-sources/types'
import { fieldByKey } from '../../report-sources/types'
import { PALETTE_OPTIONS } from '../charts/palettes'

interface ReportStylePanelProps {
  viz: VisualizationType
  source: ReportSource
  dimensions: string[]
  measures: ReportMeasure[]
  options: ReportViewOptions
  onChangeOptions: (patch: Partial<ReportViewOptions>) => void
  sort: { field: string; dir: 'asc' | 'desc' } | undefined
  limit: number | undefined
  onChangeSort: (sort: { field: string; dir: 'asc' | 'desc' } | undefined) => void
  onChangeLimit: (limit: number | undefined) => void
}

const SELECT = 'rounded-md border border-border bg-card px-2 py-1 text-xs text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring'
const CARTESIAN: VisualizationType[] = ['bar', 'line', 'area', 'combo']

function Toggle({ checked, onChange, label }: { checked: boolean; onChange: (v: boolean) => void; label: string }) {
  return (
    <label className="flex cursor-pointer items-center justify-between gap-2">
      <span className="text-xs text-muted-foreground">{label}</span>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        onClick={() => onChange(!checked)}
        className={`relative h-5 w-9 shrink-0 rounded-full transition-colors ${checked ? 'bg-primary' : 'bg-secondary'}`}
      >
        <span
          className={`absolute top-0.5 h-4 w-4 rounded-full bg-white transition-transform ${checked ? 'translate-x-4' : 'translate-x-0.5'}`}
        />
      </button>
    </label>
  )
}

export function ReportStylePanel({
  viz,
  source,
  dimensions,
  measures,
  options,
  onChangeOptions,
  sort,
  limit,
  onChangeSort,
  onChangeLimit,
}: ReportStylePanelProps) {
  const { t } = useTranslation()
  const isCartesian = CARTESIAN.includes(viz)
  const canStack = (viz === 'bar' || viz === 'area') && dimensions.length >= 2
  const canLegend = viz !== 'kpi' && viz !== 'gauge'
  const canLimit = viz !== 'kpi' && viz !== 'gauge'
  const canDataLabels = viz === 'bar' || viz === 'donut'

  // Sortable fields: the grouping dimension + each measure.
  const sortFields = [
    ...(dimensions[0] ? [{ value: dimensions[0], label: fieldByKey(source, dimensions[0])?.label ?? dimensions[0] }] : []),
    ...measures.map((m) => ({ value: m.field, label: fieldByKey(source, m.field)?.label ?? m.field })),
  ]

  return (
    <div>
      <div className="mb-2 flex items-center gap-1.5">
        <Palette className="h-3.5 w-3.5 text-muted-foreground" />
        <label className="text-xs font-medium text-muted-foreground">
          {t('berichte.builder.style.label', { defaultValue: 'Darstellung' })}
        </label>
      </div>

      <div className="space-y-2.5">
        {/* Palette */}
        <div className="flex items-center justify-between gap-2">
          <span className="text-xs text-muted-foreground">
            {t('berichte.builder.style.palette', { defaultValue: 'Farbpalette' })}
          </span>
          <select
            value={options.palette ?? 'default'}
            onChange={(e) => onChangeOptions({ palette: e.target.value })}
            className={SELECT}
          >
            {PALETTE_OPTIONS.map((p) => (
              <option key={p.id} value={p.id}>
                {p.label}
              </option>
            ))}
          </select>
        </div>

        {canLegend && (
          <Toggle
            checked={options.showLegend ?? true}
            onChange={(v) => onChangeOptions({ showLegend: v })}
            label={t('berichte.builder.style.legend', { defaultValue: 'Legende' })}
          />
        )}

        {canDataLabels && (
          <Toggle
            checked={options.showDataLabels ?? false}
            onChange={(v) => onChangeOptions({ showDataLabels: v })}
            label={t('berichte.builder.style.dataLabels', { defaultValue: 'Datenbeschriftung' })}
          />
        )}

        {canStack && (
          <Toggle
            checked={options.stacked ?? false}
            onChange={(v) => onChangeOptions({ stacked: v })}
            label={t('berichte.builder.style.stacked', { defaultValue: 'Gestapelt' })}
          />
        )}

        {/* Sort */}
        {sortFields.length > 0 && isCartesian && (
          <div className="flex items-center justify-between gap-2">
            <span className="text-xs text-muted-foreground">
              {t('berichte.builder.style.sort', { defaultValue: 'Sortierung' })}
            </span>
            <div className="flex items-center gap-1">
              <select
                value={sort?.field ?? ''}
                onChange={(e) =>
                  onChangeSort(e.target.value ? { field: e.target.value, dir: sort?.dir ?? 'desc' } : undefined)
                }
                className={SELECT}
              >
                <option value="">{t('berichte.builder.style.sortNone', { defaultValue: 'Keine' })}</option>
                {sortFields.map((f) => (
                  <option key={f.value} value={f.value}>
                    {f.label}
                  </option>
                ))}
              </select>
              {sort?.field && (
                <select
                  value={sort.dir}
                  onChange={(e) => onChangeSort({ field: sort.field, dir: e.target.value as 'asc' | 'desc' })}
                  className={SELECT}
                >
                  <option value="desc">{t('berichte.builder.style.desc', { defaultValue: 'Absteigend' })}</option>
                  <option value="asc">{t('berichte.builder.style.asc', { defaultValue: 'Aufsteigend' })}</option>
                </select>
              )}
            </div>
          </div>
        )}

        {/* Top-N */}
        {canLimit && (
          <div className="flex items-center justify-between gap-2">
            <span className="text-xs text-muted-foreground">
              {t('berichte.builder.style.topN', { defaultValue: 'Top-N begrenzen' })}
            </span>
            <input
              type="number"
              min={0}
              value={limit ?? ''}
              placeholder={t('berichte.builder.style.all', { defaultValue: 'Alle' })}
              onChange={(e) => onChangeLimit(e.target.value ? Number(e.target.value) : undefined)}
              className={`${SELECT} w-20`}
            />
          </div>
        )}
      </div>
    </div>
  )
}

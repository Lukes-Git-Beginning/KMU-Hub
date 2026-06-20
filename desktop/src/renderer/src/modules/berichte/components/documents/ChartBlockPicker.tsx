import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { BarChart3, FileText, Search } from 'lucide-react'
import type { ReportDefinition, VisualizationType } from '@/api/berichte-types'
import { isBuilderQuery } from '@/api/berichte-types'
import { useDefinitions, useReportResult } from '@/api/hooks/useBerichte'
import { DetailModal, Skeleton } from '@/components/shared'
import { resolveSource } from '../../report-sources/registry'
import { VIZ_GALLERY } from '../builder/builder-utils'
import { ChartRenderer } from '../charts/ChartRenderer'

/** Derive a sensible visualization for a definition (its builder viz, or a
 *  type-based default for legacy system definitions). */
export function vizForDefinition(def: ReportDefinition): VisualizationType {
  if (isBuilderQuery(def.query_config)) return def.query_config.viz
  const kind = (def.query_config as { kind?: string }).kind
  if (kind?.startsWith('datev')) return 'table'
  if (kind === 'revenue') return 'line'
  return 'bar'
}

function vizIconFor(def: ReportDefinition) {
  const viz = vizForDefinition(def)
  return VIZ_GALLERY.find((v) => v.type === viz)?.icon ?? FileText
}

function sourceLabelFor(def: ReportDefinition): string {
  if (isBuilderQuery(def.query_config)) {
    return resolveSource(def.query_config.sourceId)?.label ?? def.module
  }
  return def.module
}

interface ChartBlockPickerProps {
  open: boolean
  onClose: () => void
  /** Table blocks force a table visualization; chart blocks keep the def's viz. */
  mode: 'chart' | 'table'
  onApply: (patch: { definitionId: string; viz?: VisualizationType }) => void
}

/** Modal to attach a saved chart/table definition to a block (R-1b, library tab). */
export function ChartBlockPicker({ open, onClose, mode, onApply }: ChartBlockPickerProps) {
  const { t } = useTranslation()
  const { data, isLoading } = useDefinitions()
  const [search, setSearch] = useState('')
  const [selectedId, setSelectedId] = useState<string | null>(null)

  const defs = useMemo(() => data?.definitions ?? [], [data])
  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase()
    if (!q) return defs
    return defs.filter(
      (d) => d.name.toLowerCase().includes(q) || d.description.toLowerCase().includes(q),
    )
  }, [defs, search])

  const selected = defs.find((d) => d.id === selectedId) ?? null
  const previewViz: VisualizationType =
    mode === 'table' ? 'table' : selected ? vizForDefinition(selected) : 'bar'

  function apply() {
    if (!selected) return
    onApply({ definitionId: selected.id, viz: mode === 'table' ? undefined : previewViz })
    onClose()
  }

  return (
    <DetailModal
      open={open}
      onClose={onClose}
      maxWidth="max-w-3xl"
      title={
        mode === 'table'
          ? t('berichte.docs.chartPicker.titleTable')
          : t('berichte.docs.chartPicker.titleChart')
      }
      footer={
        <div className="flex items-center justify-end gap-2">
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-secondary"
          >
            {t('berichte.docs.chartPicker.cancel')}
          </button>
          <button
            type="button"
            onClick={apply}
            disabled={!selected}
            className="rounded-lg bg-primary px-3 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
          >
            {t('berichte.docs.chartPicker.apply')}
          </button>
        </div>
      }
    >
      <div className="grid grid-cols-1 gap-4 p-5 sm:grid-cols-[minmax(0,1fr)_minmax(0,1.2fr)]">
        {/* Definition list */}
        <div className="space-y-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
            <input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t('berichte.docs.chartPicker.search')}
              className="w-full rounded-lg border border-border bg-card py-2 pl-9 pr-3 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>
          <div className="max-h-[340px] space-y-1.5 overflow-y-auto pr-1">
            {isLoading ? (
              Array.from({ length: 4 }).map((_, i) => (
                <Skeleton key={i} className="h-14 w-full rounded-lg" />
              ))
            ) : filtered.length === 0 ? (
              <p className="py-8 text-center text-sm text-muted-foreground">
                {t('berichte.docs.chartPicker.empty')}
              </p>
            ) : (
              filtered.map((def) => {
                const Icon = vizIconFor(def)
                const active = def.id === selectedId
                return (
                  <button
                    key={def.id}
                    type="button"
                    onClick={() => setSelectedId(def.id)}
                    className={`flex w-full items-start gap-2.5 rounded-lg border p-2.5 text-left transition-colors ${
                      active
                        ? 'border-primary/40 bg-primary-light'
                        : 'border-border hover:border-primary/40 hover:bg-secondary'
                    }`}
                  >
                    <div
                      className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-lg ${
                        active ? 'bg-primary/15 text-primary' : 'bg-secondary text-muted-foreground'
                      }`}
                    >
                      <Icon className="h-4 w-4" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium text-foreground">{def.name}</p>
                      <p className="truncate text-[11px] text-muted-foreground">{def.description}</p>
                      <span className="mt-1 inline-block rounded-full bg-secondary px-2 py-0.5 text-[10px] text-muted-foreground">
                        {sourceLabelFor(def)}
                      </span>
                    </div>
                  </button>
                )
              })
            )}
          </div>
        </div>

        {/* Preview */}
        <div className="rounded-xl border border-border bg-card p-4">
          {selected ? (
            <DefinitionPreview definitionId={selected.id} viz={previewViz} name={selected.name} />
          ) : (
            <div className="flex h-full min-h-[240px] flex-col items-center justify-center gap-2 text-center text-sm text-muted-foreground">
              <BarChart3 className="h-8 w-8 text-muted-foreground/40" />
              {t('berichte.docs.chartPicker.previewHint')}
            </div>
          )}
        </div>
      </div>
    </DetailModal>
  )
}

function DefinitionPreview({
  definitionId,
  viz,
  name,
}: {
  definitionId: string
  viz: VisualizationType
  name: string
}) {
  const { t } = useTranslation()
  const { data, isLoading } = useReportResult(definitionId)
  const result = data?.result
  return (
    <div className="space-y-2">
      <p className="text-xs font-medium text-foreground">{name}</p>
      {result ? (
        <ChartRenderer result={result} viz={viz} height={260} />
      ) : isLoading ? (
        <Skeleton className="h-[240px] w-full rounded-lg" />
      ) : (
        <div className="flex h-[200px] items-center justify-center text-sm text-muted-foreground">
          {t('berichte.docs.chartPicker.previewError')}
        </div>
      )}
    </div>
  )
}

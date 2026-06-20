import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { BarChart3, FilePlus2, FolderOpen, FileText, Search } from 'lucide-react'
import type {
  ReportDateRange,
  ReportDefinition,
  ReportFilter,
  VisualizationType,
} from '@/api/berichte-types'
import { isBuilderQuery } from '@/api/berichte-types'
import {
  useCreateDefinition,
  useDefinitions,
  useReportPreview,
  useReportResult,
} from '@/api/hooks/useBerichte'
import { DetailModal, Skeleton } from '@/components/shared'
import { resolveSource } from '../../report-sources/registry'
import { fieldByKey, type ReportSource } from '../../report-sources/types'
import {
  emptyBuilderState,
  suggestViz,
  toQuery,
  MAX_DIMENSIONS,
  VIZ_GALLERY,
  type BuilderState,
} from '../builder/builder-utils'
import { SourcePicker } from '../builder/SourcePicker'
import { FieldPicker } from '../builder/FieldPicker'
import { FilterBuilder } from '../builder/FilterBuilder'
import { DateRangePicker } from '../builder/DateRangePicker'
import { VizSwitcher } from '../builder/VizSwitcher'
import { LivePreview } from '../builder/LivePreview'
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

/** Modal to attach a chart/table definition to a block — pick from the library
 *  or build a new one inline (R-1b). */
export function ChartBlockPicker({ open, onClose, mode, onApply }: ChartBlockPickerProps) {
  const { t } = useTranslation()
  const [tab, setTab] = useState<'library' | 'new'>('library')

  function applyAndClose(patch: { definitionId: string; viz?: VisualizationType }) {
    onApply(patch)
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
    >
      <div className="space-y-4 p-5">
        {/* Tab switcher */}
        <div className="inline-flex rounded-lg border border-border p-0.5">
          <button
            type="button"
            onClick={() => setTab('library')}
            className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm transition-colors ${
              tab === 'library'
                ? 'bg-primary-light font-medium text-primary'
                : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            <FolderOpen className="h-3.5 w-3.5" />
            {t('berichte.docs.chartPicker.tabLibrary')}
          </button>
          <button
            type="button"
            onClick={() => setTab('new')}
            className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm transition-colors ${
              tab === 'new'
                ? 'bg-primary-light font-medium text-primary'
                : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            <FilePlus2 className="h-3.5 w-3.5" />
            {t('berichte.docs.chartPicker.tabNew')}
          </button>
        </div>

        {tab === 'library' ? (
          <LibraryTab mode={mode} onApply={applyAndClose} onCancel={onClose} />
        ) : (
          <NewChartTab mode={mode} onApply={applyAndClose} onCancel={onClose} />
        )}
      </div>
    </DetailModal>
  )
}

/* ------------------------------ Library tab ------------------------------- */

function LibraryTab({
  mode,
  onApply,
  onCancel,
}: {
  mode: 'chart' | 'table'
  onApply: (patch: { definitionId: string; viz?: VisualizationType }) => void
  onCancel: () => void
}) {
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

  return (
    <>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-[minmax(0,1fr)_minmax(0,1.2fr)]">
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
          <div className="max-h-[320px] space-y-1.5 overflow-y-auto pr-1">
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
            <div className="flex h-full min-h-[220px] flex-col items-center justify-center gap-2 text-center text-sm text-muted-foreground">
              <BarChart3 className="h-8 w-8 text-muted-foreground/40" />
              {t('berichte.docs.chartPicker.previewHint')}
            </div>
          )}
        </div>
      </div>

      <PickerActions
        onCancel={onCancel}
        onConfirm={() =>
          selected &&
          onApply({
            definitionId: selected.id,
            viz: mode === 'table' ? undefined : previewViz,
          })
        }
        confirmLabel={t('berichte.docs.chartPicker.apply')}
        disabled={!selected}
      />
    </>
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
        <ChartRenderer result={result} viz={viz} height={240} />
      ) : isLoading ? (
        <Skeleton className="h-[220px] w-full rounded-lg" />
      ) : (
        <div className="flex h-[180px] items-center justify-center text-sm text-muted-foreground">
          {t('berichte.docs.chartPicker.previewError')}
        </div>
      )}
    </div>
  )
}

/* -------------------------------- New tab --------------------------------- */

function applyAutoViz(next: BuilderState): BuilderState {
  if (next.vizManual) return next
  return {
    ...next,
    viz: suggestViz(next, next.sourceId ? resolveSource(next.sourceId) : undefined),
  }
}

function NewChartTab({
  mode,
  onApply,
  onCancel,
}: {
  mode: 'chart' | 'table'
  onApply: (patch: { definitionId: string; viz?: VisualizationType }) => void
  onCancel: () => void
}) {
  const { t } = useTranslation()
  const [state, setState] = useState<BuilderState>(emptyBuilderState)
  const createMutation = useCreateDefinition()

  const source: ReportSource | undefined = state.sourceId
    ? resolveSource(state.sourceId)
    : undefined
  const query = useMemo(() => {
    const q = toQuery(state)
    if (q && mode === 'table') return { ...q, viz: 'table' as VisualizationType }
    return q
  }, [state, mode])

  // Debounce so typing doesn't refetch the preview on every change.
  const [debounced, setDebounced] = useState(query)
  useEffect(() => {
    const id = setTimeout(() => setDebounced(query), 150)
    return () => clearTimeout(id)
  }, [query])
  const preview = useReportPreview(debounced)

  function setSource(id: string) {
    setState(applyAutoViz({ ...emptyBuilderState(), sourceId: id }))
  }
  function toggleDimension(key: string) {
    setState((s) => {
      const has = s.dimensions.includes(key)
      let dimensions = has ? s.dimensions.filter((d) => d !== key) : [...s.dimensions, key]
      if (dimensions.length > MAX_DIMENSIONS) dimensions = dimensions.slice(0, MAX_DIMENSIONS)
      return applyAutoViz({ ...s, dimensions })
    })
  }
  function toggleMeasure(key: string) {
    setState((s) => {
      const has = s.measures.some((m) => m.field === key)
      if (has) return applyAutoViz({ ...s, measures: s.measures.filter((m) => m.field !== key) })
      const field = source ? fieldByKey(source, key) : undefined
      return applyAutoViz({
        ...s,
        measures: [...s.measures, { field: key, agg: field?.defaultAgg ?? 'sum' }],
      })
    })
  }
  function selectViz(viz: VisualizationType) {
    setState((s) => ({ ...s, viz, vizManual: true }))
  }
  function setFilters(filters: ReportFilter[]) {
    setState((s) => ({ ...s, filters }))
  }
  function setDateRange(dateRange: ReportDateRange | undefined) {
    setState((s) => ({ ...s, dateRange }))
  }

  function save() {
    if (!query || !source) return
    const name =
      state.name.trim() ||
      `${source.label} – ${VIZ_GALLERY.find((v) => v.type === query.viz)?.label ?? ''}`.trim()
    createMutation.mutate(
      {
        name,
        module: source.module,
        kind: 'custom',
        query_config: query,
        default_format: 'pdf',
        is_published: false,
      },
      {
        onSuccess: (res) =>
          onApply({
            definitionId: res.definition.id,
            viz: mode === 'table' ? undefined : query.viz,
          }),
      },
    )
  }

  const hasFields = state.dimensions.length > 0 || state.measures.length > 0

  return (
    <>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-[minmax(0,1fr)_minmax(0,1.2fr)]">
        <div className="max-h-[360px] space-y-3 overflow-y-auto pr-1">
          <div className="rounded-xl border border-border bg-card p-3">
            <SourcePicker value={state.sourceId} onChange={setSource} />
          </div>
          {source && (
            <div className="rounded-xl border border-border bg-card p-3">
              <FieldPicker
                source={source}
                dimensions={state.dimensions}
                measures={state.measures}
                onToggleDimension={toggleDimension}
                onToggleMeasure={toggleMeasure}
              />
            </div>
          )}
          {source && (
            <div className="space-y-3 rounded-xl border border-border bg-card p-3">
              <DateRangePicker source={source} value={state.dateRange} onChange={setDateRange} />
              <FilterBuilder source={source} filters={state.filters} onChange={setFilters} />
            </div>
          )}
          {source && hasFields && mode === 'chart' && (
            <div className="rounded-xl border border-border bg-card p-3">
              <VizSwitcher value={state.viz} auto={!state.vizManual} onSelect={selectViz} />
            </div>
          )}
          {source && query && (
            <input
              value={state.name}
              onChange={(e) => setState((s) => ({ ...s, name: e.target.value }))}
              placeholder={t('berichte.docs.chartPicker.namePlaceholder')}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          )}
        </div>

        <div className="min-h-[300px]">
          <LivePreview
            query={query}
            result={preview.data?.result}
            source={source}
            isLoading={preview.isFetching}
            isError={preview.isError}
          />
        </div>
      </div>

      <PickerActions
        onCancel={onCancel}
        onConfirm={save}
        confirmLabel={t('berichte.docs.chartPicker.saveInsert')}
        disabled={!query || createMutation.isPending}
      />
    </>
  )
}

/* ------------------------------- Shared bits ------------------------------ */

function PickerActions({
  onCancel,
  onConfirm,
  confirmLabel,
  disabled,
}: {
  onCancel: () => void
  onConfirm: () => void
  confirmLabel: string
  disabled: boolean
}) {
  const { t } = useTranslation()
  return (
    <div className="flex items-center justify-end gap-2 border-t border-border-muted pt-3">
      <button
        type="button"
        onClick={onCancel}
        className="rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-secondary"
      >
        {t('berichte.docs.chartPicker.cancel')}
      </button>
      <button
        type="button"
        onClick={onConfirm}
        disabled={disabled}
        className="rounded-lg bg-primary px-3 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
      >
        {confirmLabel}
      </button>
    </div>
  )
}

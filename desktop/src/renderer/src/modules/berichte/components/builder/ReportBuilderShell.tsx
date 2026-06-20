import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { FilePlus2, FolderOpen, Plus } from 'lucide-react'
import { toast } from 'sonner'
import type { BuilderState } from './builder-utils'
import { emptyBuilderState, fromQuery, MAX_DIMENSIONS, suggestViz, toQuery } from './builder-utils'
import type {
  AggregationFn,
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
  useUpdateDefinition,
} from '@/api/hooks/useBerichte'
import { resolveSource } from '../../report-sources/registry'
import { fieldByKey } from '../../report-sources/types'
import { SourcePicker } from './SourcePicker'
import { FieldPicker } from './FieldPicker'
import { SummarizeBlock } from './SummarizeBlock'
import { DateRangePicker } from './DateRangePicker'
import { FilterBuilder } from './FilterBuilder'
import { VizSwitcher } from './VizSwitcher'
import { LivePreview } from './LivePreview'
import { SaveReportBar } from './SaveReportBar'
import { MyReportsLibrary } from './MyReportsLibrary'

function applyAutoViz(next: BuilderState): BuilderState {
  if (next.vizManual) return next
  return { ...next, viz: suggestViz(next, next.sourceId ? resolveSource(next.sourceId) : undefined) }
}

export function ReportBuilderShell() {
  const { t } = useTranslation()
  const [view, setView] = useState<'builder' | 'library'>('builder')
  const [editingId, setEditingId] = useState<string | null>(null)
  const [state, setState] = useState<BuilderState>(emptyBuilderState)

  const source = state.sourceId ? resolveSource(state.sourceId) : undefined
  const query = useMemo(() => toQuery(state), [state])
  // Debounce so typing in filter inputs doesn't refetch on every keystroke.
  const [debouncedQuery, setDebouncedQuery] = useState(query)
  useEffect(() => {
    const id = setTimeout(() => setDebouncedQuery(query), 150)
    return () => clearTimeout(id)
  }, [query])
  const preview = useReportPreview(debouncedQuery)

  const createMutation = useCreateDefinition()
  const updateMutation = useUpdateDefinition()
  const customDefs = useDefinitions({ kind: 'custom' })
  const customCount = customDefs.data?.definitions.length ?? 0

  function setSource(id: string) {
    setState(() => applyAutoViz({ ...emptyBuilderState(), sourceId: id }))
    setEditingId(null)
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
      if (has) {
        return applyAutoViz({ ...s, measures: s.measures.filter((m) => m.field !== key) })
      }
      const src = s.sourceId ? resolveSource(s.sourceId) : undefined
      const field = src ? fieldByKey(src, key) : undefined
      return applyAutoViz({
        ...s,
        measures: [...s.measures, { field: key, agg: field?.defaultAgg ?? 'sum' }],
      })
    })
  }

  function selectViz(viz: VisualizationType) {
    setState((s) => ({ ...s, viz, vizManual: true }))
  }

  function setDateRange(dateRange: ReportDateRange | undefined) {
    setState((s) => ({ ...s, dateRange }))
  }

  function setFilters(filters: ReportFilter[]) {
    setState((s) => ({ ...s, filters }))
  }

  function setMeasureAgg(field: string, agg: AggregationFn) {
    setState((s) => ({
      ...s,
      measures: s.measures.map((m) => (m.field === field ? { ...m, agg } : m)),
    }))
  }

  function setName(name: string) {
    setState((s) => ({ ...s, name }))
  }

  function startNew() {
    setState(emptyBuilderState())
    setEditingId(null)
    setView('builder')
  }

  function handleSave() {
    if (!query || !source || !state.name.trim()) return
    if (editingId) {
      updateMutation.mutate(
        { id: editingId, name: state.name.trim(), module: source.module, query_config: query },
        { onSuccess: () => toast.success(t('berichte.builder.save.updated', { defaultValue: 'Bericht aktualisiert.' })) },
      )
    } else {
      createMutation.mutate(
        {
          name: state.name.trim(),
          module: source.module,
          kind: 'custom',
          query_config: query,
          default_format: 'pdf',
          is_published: false,
        },
        {
          onSuccess: (res) => {
            setEditingId(res.definition.id)
            toast.success(t('berichte.builder.save.saved', { defaultValue: 'Bericht gespeichert.' }))
          },
        },
      )
    }
  }

  function handleOpen(def: ReportDefinition) {
    if (!isBuilderQuery(def.query_config)) {
      toast.error(
        t('berichte.builder.open.notBuilder', { defaultValue: 'Dieser System-Bericht lässt sich nicht im Builder öffnen.' }),
      )
      return
    }
    setState(fromQuery(def.query_config, def.name))
    setEditingId(def.id)
    setView('builder')
  }

  const isPending = createMutation.isPending || updateMutation.isPending

  return (
    <div className="space-y-4">
      {/* View toggle */}
      <div className="inline-flex rounded-lg border border-border p-0.5">
        <button
          type="button"
          onClick={() => setView('builder')}
          className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm transition-colors ${
            view === 'builder' ? 'bg-primary-light font-medium text-primary' : 'text-muted-foreground hover:text-foreground'
          }`}
        >
          <FilePlus2 className="h-3.5 w-3.5" />
          {t('berichte.builder.tab.builder', { defaultValue: 'Neuer Bericht' })}
        </button>
        <button
          type="button"
          onClick={() => setView('library')}
          className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm transition-colors ${
            view === 'library' ? 'bg-primary-light font-medium text-primary' : 'text-muted-foreground hover:text-foreground'
          }`}
        >
          <FolderOpen className="h-3.5 w-3.5" />
          {t('berichte.builder.tab.library', { defaultValue: 'Meine Berichte' })}
          {customCount > 0 && (
            <span className="rounded-full bg-secondary px-1.5 text-[10px] text-muted-foreground">{customCount}</span>
          )}
        </button>
      </div>

      {view === 'library' ? (
        <MyReportsLibrary onOpen={handleOpen} />
      ) : (
        <div className="space-y-4">
          {/* Save bar */}
          <div className="flex items-center gap-2">
            <div className="flex-1">
              <SaveReportBar
                name={state.name}
                onNameChange={setName}
                onSave={handleSave}
                isPending={isPending}
                isEditing={!!editingId}
                canSave={!!query}
              />
            </div>
            {editingId && (
              <button
                type="button"
                onClick={startNew}
                className="flex shrink-0 items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
              >
                <Plus className="h-4 w-4" />
                {t('berichte.builder.tab.new', { defaultValue: 'Neu' })}
              </button>
            )}
          </div>

          {/* Builder grid */}
          <div className="grid grid-cols-1 gap-5 lg:grid-cols-[340px_minmax(0,1fr)]">
            <div className="space-y-4">
              <div className="rounded-xl border border-border bg-card p-4">
                <SourcePicker value={state.sourceId} onChange={setSource} />
              </div>

              {source && (
                <div className="space-y-4 rounded-xl border border-border bg-card p-4">
                  <FieldPicker
                    source={source}
                    dimensions={state.dimensions}
                    measures={state.measures}
                    onToggleDimension={toggleDimension}
                    onToggleMeasure={toggleMeasure}
                  />
                  {(state.measures.length > 0 || state.dimensions.length > 0) && (
                    <div className="border-t border-border-muted pt-4">
                      <SummarizeBlock
                        source={source}
                        measures={state.measures}
                        dimensions={state.dimensions}
                        onChangeAgg={setMeasureAgg}
                      />
                    </div>
                  )}
                </div>
              )}

              {source && (
                <div className="space-y-4 rounded-xl border border-border bg-card p-4">
                  <DateRangePicker source={source} value={state.dateRange} onChange={setDateRange} />
                  <FilterBuilder source={source} filters={state.filters} onChange={setFilters} />
                </div>
              )}

              {source && (state.dimensions.length > 0 || state.measures.length > 0) && (
                <div className="rounded-xl border border-border bg-card p-4">
                  <VizSwitcher value={state.viz} auto={!state.vizManual} onSelect={selectViz} />
                </div>
              )}
            </div>

            <div className="min-h-[480px]">
              <LivePreview
                query={query}
                result={preview.data?.result}
                source={source}
                isLoading={preview.isFetching}
                isError={preview.isError}
              />
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

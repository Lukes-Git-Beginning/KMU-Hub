import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { BuilderState } from './builder-utils'
import { emptyBuilderState, MAX_DIMENSIONS, suggestViz, toQuery } from './builder-utils'
import type { VisualizationType } from '@/api/berichte-types'
import { useReportPreview } from '@/api/hooks/useBerichte'
import { resolveSource } from '../../report-sources/registry'
import { fieldByKey } from '../../report-sources/types'
import { SourcePicker } from './SourcePicker'
import { FieldPicker } from './FieldPicker'
import { VizSwitcher } from './VizSwitcher'
import { LivePreview } from './LivePreview'

function applyAutoViz(next: BuilderState): BuilderState {
  if (next.vizManual) return next
  return { ...next, viz: suggestViz(next, next.sourceId ? resolveSource(next.sourceId) : undefined) }
}

export function ReportBuilderShell() {
  const { t } = useTranslation()
  const [state, setState] = useState<BuilderState>(emptyBuilderState)
  const source = state.sourceId ? resolveSource(state.sourceId) : undefined
  const query = useMemo(() => toQuery(state), [state])
  const preview = useReportPreview(query)

  function setSource(id: string) {
    setState(() => applyAutoViz({ ...emptyBuilderState(), sourceId: id }))
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

  return (
    <div className="grid grid-cols-1 gap-5 lg:grid-cols-[340px_minmax(0,1fr)]">
      {/* Config column */}
      <div className="space-y-4">
        <div className="rounded-xl border border-border bg-card p-4">
          <SourcePicker value={state.sourceId} onChange={setSource} />
        </div>

        {source && (
          <div className="rounded-xl border border-border bg-card p-4">
            <FieldPicker
              source={source}
              dimensions={state.dimensions}
              measures={state.measures}
              onToggleDimension={toggleDimension}
              onToggleMeasure={toggleMeasure}
            />
          </div>
        )}

        {source && (state.dimensions.length > 0 || state.measures.length > 0) && (
          <div className="rounded-xl border border-border bg-card p-4">
            <VizSwitcher value={state.viz} auto={!state.vizManual} onSelect={selectViz} />
          </div>
        )}
      </div>

      {/* Preview column */}
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
  )
}

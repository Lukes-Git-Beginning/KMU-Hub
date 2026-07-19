import { useTranslation } from 'react-i18next'
import { Pin } from 'lucide-react'
import type { ReportDefinition } from '@/api/berichte-types'
import { isBuilderQuery } from '@/api/berichte-types'
import { useDefinitions, useReportPreview } from '@/api/hooks/useBerichte'
import { useCapabilitySet } from '@/hooks/useCapability'
import { isReportModuleVisible } from '../report-module-visibility'
import { ChartRenderer } from './charts/ChartRenderer'

function PinnedReportCard({ def }: { def: ReportDefinition }) {
  const query = isBuilderQuery(def.query_config) ? def.query_config : null
  const preview = useReportPreview(query)
  return (
    <div className="rounded-xl border border-border bg-card p-4">
      <h4 className="mb-3 truncate text-sm font-medium text-foreground">{def.name}</h4>
      {query && preview.data?.result ? (
        <ChartRenderer result={preview.data.result} viz={query.viz} options={query.options} height={200} />
      ) : (
        <div className="flex h-[200px] items-center justify-center text-xs text-muted-foreground">
          {preview.isError ? '—' : ''}
        </div>
      )}
    </div>
  )
}

/** Dashboard section: custom builder reports the user pinned to the dashboard. */
export function PinnedReports() {
  const { t } = useTranslation()
  const { data } = useDefinitions({ kind: 'custom', is_published: true })
  // RBAC: pinned builder charts follow their source module's visibility.
  const { has, ready } = useCapabilitySet()
  const pinned = (data?.definitions ?? []).filter(
    (d) => isBuilderQuery(d.query_config) && isReportModuleVisible(has, ready, String(d.module)),
  )
  if (pinned.length === 0) return null

  return (
    <div className="mb-8">
      <div className="mb-3 flex items-center gap-2">
        <Pin className="h-4 w-4 text-primary" />
        <h3 className="text-sm font-medium text-foreground">
          {t('berichte.dashboard.pinned', { defaultValue: 'Meine Berichte' })}
        </h3>
      </div>
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        {pinned.map((def) => (
          <PinnedReportCard key={def.id} def={def} />
        ))}
      </div>
    </div>
  )
}

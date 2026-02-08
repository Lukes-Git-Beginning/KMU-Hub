/**
 * DealPipeline widget -- mini pipeline overview with stage bars.
 *
 * Shows a horizontal bar per pipeline stage with deal count and total EUR value.
 * Uses PipelineStageInfo which already includes dealCount and totalValue.
 * Click navigates to the deals list filtered by stage.
 */
import { memo } from 'react'
import { useNavigate } from 'react-router-dom'
import { TrendingUp } from 'lucide-react'
import { usePipelineStages } from '@/api/hooks/usePipelineStages'
import { Skeleton } from '@/components/ui/skeleton'
import { Button } from '@/components/ui/button'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

function formatCurrency(value?: number): string {
  if (value == null) return '0 EUR'
  return new Intl.NumberFormat('de-DE', {
    style: 'currency',
    currency: 'EUR',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(value)
}

function DealPipeline(_props: WidgetProps) {
  const navigate = useNavigate()
  const { data, isLoading, error, refetch } = usePipelineStages()

  const stages = data?.stages ?? []

  if (error) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <div className="text-center">
          <p className="text-sm text-muted-foreground">Fehler beim Laden</p>
          <Button variant="ghost" size="sm" className="mt-2" onClick={() => refetch()}>
            Erneut versuchen
          </Button>
        </div>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="space-y-3 p-4">
        {Array.from({ length: 5 }).map((_, i) => (
          <div key={i} className="space-y-1">
            <Skeleton className="h-3.5 w-1/3" />
            <Skeleton className="h-6 w-full rounded" />
          </div>
        ))}
      </div>
    )
  }

  if (stages.length === 0) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <div className="text-center">
          <TrendingUp className="mx-auto h-8 w-8 text-muted-foreground/50" />
          <p className="mt-2 text-sm text-muted-foreground">Keine Deals vorhanden</p>
        </div>
      </div>
    )
  }

  // Find max total value for proportional bar widths
  const maxValue = Math.max(...stages.map((s) => s.totalValue ?? 0), 1)

  return (
    <div className="space-y-2.5 p-4">
      {stages.map((stage) => {
        const dealCount = stage.dealCount ?? 0
        const total = stage.totalValue ?? 0
        const barWidth = maxValue > 0 ? Math.max((total / maxValue) * 100, 4) : 4

        return (
          <div
            key={stage.id}
            className="cursor-pointer rounded-md p-2 transition-colors hover:bg-accent/50"
            onClick={() => navigate(`/crm/deals?stage=${stage.id}`)}
          >
            <div className="mb-1 flex items-center justify-between">
              <span className="text-xs font-medium">{stage.name}</span>
              <span className="text-xs text-muted-foreground">
                {dealCount} Deal{dealCount !== 1 ? 's' : ''}
              </span>
            </div>
            <div className="flex items-center gap-2">
              <div className="h-5 flex-1 overflow-hidden rounded bg-muted">
                <div
                  className="flex h-full items-center rounded px-2 text-[10px] font-semibold text-white transition-all"
                  style={{
                    width: `${barWidth}%`,
                    backgroundColor: stage.color || '#3b82f6',
                    minWidth: dealCount > 0 ? '2rem' : undefined,
                  }}
                >
                  {dealCount > 0 && (
                    <span className="truncate">{formatCurrency(total)}</span>
                  )}
                </div>
              </div>
            </div>
          </div>
        )
      })}
    </div>
  )
}

export default memo(DealPipeline)

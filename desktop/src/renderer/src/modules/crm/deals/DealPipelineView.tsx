/**
 * Visual deal pipeline view displaying deals grouped by stage in horizontal columns.
 *
 * Each column represents a pipeline stage. Deals appear as cards within
 * their respective stage column. Horizontal scrolling when stages exceed
 * viewport width. Drag-and-drop is a future enhancement.
 */
import { useNavigate } from 'react-router-dom'
import { usePipelineStages } from '@/api/hooks/usePipelineStages'
import { useDeals } from '@/api/hooks/useDeals'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { ScrollArea, ScrollBar } from '@/components/ui/scroll-area'
import { Button } from '@/components/ui/button'

function formatCurrency(value?: number, currency?: string): string {
  if (value == null) return '-'
  return new Intl.NumberFormat('de-DE', {
    style: 'currency',
    currency: currency || 'EUR',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(value)
}

export default function DealPipelineView() {
  const navigate = useNavigate()
  const {
    data: stagesData,
    isLoading: stagesLoading,
    error: stagesError,
    refetch: refetchStages,
  } = usePipelineStages()
  const {
    data: dealsData,
    isLoading: dealsLoading,
    error: dealsError,
    refetch: refetchDeals,
  } = useDeals({ page_size: 200 })

  const stages = stagesData?.stages ?? []
  const deals = dealsData?.deals ?? []

  const isLoading = stagesLoading || dealsLoading
  const error = stagesError || dealsError

  if (error) {
    return (
      <div className="flex items-center justify-center py-16">
        <div className="text-center">
          <p className="text-lg font-semibold text-foreground">
            Fehler beim Laden der Pipeline
          </p>
          <p className="mt-2 text-sm text-muted-foreground">
            {error instanceof Error ? error.message : 'Ein unerwarteter Fehler ist aufgetreten.'}
          </p>
          <Button
            variant="outline"
            className="mt-4"
            onClick={() => {
              refetchStages()
              refetchDeals()
            }}
          >
            Erneut versuchen
          </Button>
        </div>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="flex gap-4 overflow-x-auto pb-4">
        {Array.from({ length: 5 }).map((_, i) => (
          <div key={i} className="min-w-[280px]">
            <Skeleton className="mb-3 h-8 w-full" />
            {Array.from({ length: 3 }).map((_, j) => (
              <Skeleton key={j} className="mb-2 h-24 w-full" />
            ))}
          </div>
        ))}
      </div>
    )
  }

  if (stages.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-16">
        <p className="text-lg font-medium text-foreground">
          Keine Pipeline-Phasen definiert
        </p>
        <p className="mt-1 text-sm text-muted-foreground">
          Pipeline-Phasen werden im Backend konfiguriert.
        </p>
      </div>
    )
  }

  // Group deals by stage
  const dealsByStage = new Map<string, typeof deals>()
  for (const stage of stages) {
    dealsByStage.set(stage.id ?? '', [])
  }
  for (const deal of deals) {
    const stageDeals = dealsByStage.get(deal.stageId ?? '')
    if (stageDeals) {
      stageDeals.push(deal)
    }
  }

  return (
    <ScrollArea className="w-full">
      <div className="flex gap-4 pb-4" style={{ minWidth: `${stages.length * 296}px` }}>
        {stages.map((stage) => {
          const stageDeals = dealsByStage.get(stage.id ?? '') ?? []
          const stageTotal = stageDeals.reduce(
            (sum, d) => sum + (d.value ?? 0),
            0
          )

          return (
            <div
              key={stage.id}
              className="min-w-[280px] max-w-[280px] flex-shrink-0"
            >
              {/* Stage header */}
              <div
                className="mb-3 flex items-center justify-between rounded-lg px-3 py-2"
                style={{
                  backgroundColor: stage.color
                    ? `${stage.color}15`
                    : undefined,
                  borderLeft: stage.color
                    ? `3px solid ${stage.color}`
                    : '3px solid hsl(var(--border))',
                }}
              >
                <div>
                  <p className="text-sm font-semibold">{stage.name}</p>
                  <p className="text-xs text-muted-foreground">
                    {stageDeals.length} Deal{stageDeals.length !== 1 ? 's' : ''}{' '}
                    &middot; {formatCurrency(stageTotal)}
                  </p>
                </div>
                {stage.probability != null && (
                  <Badge variant="outline" className="text-xs">
                    {stage.probability}%
                  </Badge>
                )}
              </div>

              {/* Deal cards */}
              <div className="space-y-2">
                {stageDeals.length === 0 ? (
                  <div className="rounded-lg border border-dashed border-border p-4 text-center">
                    <p className="text-xs text-muted-foreground">
                      Keine Deals
                    </p>
                  </div>
                ) : (
                  stageDeals.map((deal) => (
                    <div
                      key={deal.id}
                      className="cursor-pointer rounded-lg border border-border bg-card p-3 shadow-sm transition-colors hover:bg-accent/50"
                      onClick={() => navigate(`/crm/deals/${deal.id}`)}
                    >
                      <p className="text-sm font-medium truncate">
                        {deal.name}
                      </p>
                      <p className="mt-1 text-sm font-semibold text-primary">
                        {formatCurrency(deal.value, deal.currency)}
                      </p>
                      {deal.contactName && (
                        <p className="mt-1 text-xs text-muted-foreground truncate">
                          {deal.contactName}
                        </p>
                      )}
                      {deal.expectedCloseDate && (
                        <p className="mt-1 text-xs text-muted-foreground">
                          Abschluss:{' '}
                          {new Date(
                            deal.expectedCloseDate
                          ).toLocaleDateString('de-DE')}
                        </p>
                      )}
                      {deal.tags && deal.tags.length > 0 && (
                        <div className="mt-2 flex flex-wrap gap-1">
                          {deal.tags.slice(0, 3).map((tag) => (
                            <Badge
                              key={tag.id}
                              variant="secondary"
                              className="text-[10px] px-1.5 py-0"
                              style={
                                tag.color
                                  ? {
                                      backgroundColor: `${tag.color}20`,
                                      color: tag.color,
                                    }
                                  : undefined
                              }
                            >
                              {tag.name}
                            </Badge>
                          ))}
                          {deal.tags.length > 3 && (
                            <Badge
                              variant="secondary"
                              className="text-[10px] px-1.5 py-0"
                            >
                              +{deal.tags.length - 3}
                            </Badge>
                          )}
                        </div>
                      )}
                    </div>
                  ))
                )}
              </div>
            </div>
          )
        })}
      </div>
      <ScrollBar orientation="horizontal" />
    </ScrollArea>
  )
}

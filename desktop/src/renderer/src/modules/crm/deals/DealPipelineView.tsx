/**
 * Visual deal pipeline (Kanban) with drag-and-drop between stages.
 *
 * Cards are draggable (@dnd-kit) and drop onto stage columns. Stage moves use
 * useMoveDealToStage with an optimistic cache update for instant feedback and
 * rollback on error. Cards surface a "rotting" indicator (days since last
 * activity) for open stages and hover quick-actions to mark won/lost.
 */
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  DndContext,
  DragOverlay,
  closestCorners,
  PointerSensor,
  KeyboardSensor,
  useSensor,
  useSensors,
  useDraggable,
  useDroppable,
  type DragStartEvent,
  type DragEndEvent,
} from '@dnd-kit/core'
import { useQueryClient } from '@tanstack/react-query'
import { Clock, Trophy, XCircle } from 'lucide-react'
import { toast } from 'sonner'
import { usePipelineStages } from '@/api/hooks/usePipelineStages'
import { useDeals, useMoveDealToStage } from '@/api/hooks/useDeals'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { ScrollArea, ScrollBar } from '@/components/ui/scroll-area'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib'
import { formatDate } from '@/lib/format'

type Deal = NonNullable<ReturnType<typeof useDeals>['data']>['deals'][number]
type Stage = NonNullable<ReturnType<typeof usePipelineStages>['data']>['stages'][number]

// Rotting thresholds (days since last update on an open-stage deal).
const ROT_AMBER = 14
const ROT_RED = 30

function formatCurrency(value?: number, currency?: string): string {
  if (value == null) return '-'
  return new Intl.NumberFormat('de-DE', {
    style: 'currency',
    currency: currency || 'EUR',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(value)
}

// Mock data carries snake_case timestamps while the OpenAPI type is camelCase;
// read both so the indicator works against mock and real backend alike.
function inactiveDays(deal: Deal): number {
  const r = deal as Record<string, unknown>
  const raw = (r.updatedAt ?? r.updated_at ?? r.createdAt ?? r.created_at) as string | undefined
  if (!raw) return 0
  const ms = Date.now() - new Date(raw).getTime()
  return Math.max(0, Math.floor(ms / 86_400_000))
}

function stageIsWon(s: Stage): boolean {
  const r = s as Record<string, unknown>
  return Boolean(r.isWon ?? r.is_won)
}
function stageIsLost(s: Stage): boolean {
  const r = s as Record<string, unknown>
  return Boolean(r.isLost ?? r.is_lost)
}

// ---------------------------------------------------------------------------
// Draggable deal card
// ---------------------------------------------------------------------------

interface DealCardProps {
  deal: Deal
  stageOpen: boolean
  onOpen: (id: string) => void
  onMark: (deal: Deal, target: 'won' | 'lost') => void
  isDragOverlay?: boolean
}

function DealCard({ deal, stageOpen, onOpen, onMark, isDragOverlay }: DealCardProps) {
  const { t } = useTranslation()
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({
    id: deal.id ?? '',
    data: { deal },
  })

  const style = transform
    ? { transform: `translate3d(${transform.x}px, ${transform.y}px, 0)` }
    : undefined

  const days = stageOpen ? inactiveDays(deal) : 0
  const rotting = days >= ROT_RED ? 'red' : days >= ROT_AMBER ? 'amber' : null

  const stopDrag = (e: React.PointerEvent) => e.stopPropagation()

  return (
    <div
      ref={setNodeRef}
      style={style}
      {...attributes}
      {...listeners}
      role="button"
      tabIndex={0}
      className={cn(
        'group relative cursor-grab rounded-lg border border-border bg-card p-3 shadow-sm transition-shadow active:cursor-grabbing',
        'hover:shadow-md hover:border-border/80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50',
        isDragging && 'opacity-30',
        isDragOverlay && 'shadow-lg border-primary/30 rotate-2 scale-105'
      )}
      onClick={() => { if (!isDragging && deal.id) onOpen(deal.id) }}
      onKeyDown={(e) => {
        if ((e.key === 'Enter' || e.key === ' ') && deal.id) { e.preventDefault(); onOpen(deal.id) }
      }}
    >
      <p className="text-sm font-medium truncate pr-1">{deal.name}</p>
      <p className="mt-1 text-sm font-semibold text-primary">
        {formatCurrency(deal.value, deal.currency)}
      </p>
      {deal.contactName && (
        <p className="mt-1 text-xs text-muted-foreground truncate">{deal.contactName}</p>
      )}
      {deal.expectedCloseDate && (
        <p className="mt-1 text-xs text-muted-foreground">
          {t('crm.deals.closeLabel')}: {formatDate(deal.expectedCloseDate)}
        </p>
      )}

      {/* Rotting indicator */}
      {rotting && (
        <div
          className={cn(
            'mt-2 inline-flex items-center gap-1 rounded-full px-1.5 py-0.5 text-[10px] font-medium',
            rotting === 'red'
              ? 'bg-destructive/10 text-destructive'
              : 'bg-warning-light text-warning'
          )}
          title={t('crm.deals.inactiveDays', { days })}
        >
          <Clock className="h-2.5 w-2.5" />
          {t('crm.deals.inactiveDays', { days })}
        </div>
      )}

      {/* Tags */}
      {deal.tags && deal.tags.length > 0 && (
        <div className="mt-2 flex flex-wrap gap-1">
          {deal.tags.slice(0, 3).map((tag) => (
            <Badge
              key={tag.id}
              variant="secondary"
              className="text-[10px] px-1.5 py-0"
              style={tag.color ? { backgroundColor: `${tag.color}20`, color: tag.color } : undefined}
            >
              {tag.name}
            </Badge>
          ))}
          {deal.tags.length > 3 && (
            <Badge variant="secondary" className="text-[10px] px-1.5 py-0">
              +{deal.tags.length - 3}
            </Badge>
          )}
        </div>
      )}

      {/* Hover quick-actions — only for open stages, hidden in the drag overlay */}
      {stageOpen && !isDragOverlay && (
        <div
          className="mt-2 hidden items-center gap-1 group-hover:flex group-focus-within:flex"
          onPointerDown={stopDrag}
        >
          <button
            type="button"
            onClick={(e) => { e.stopPropagation(); onMark(deal, 'won') }}
            className="inline-flex items-center gap-1 rounded-md border border-success/40 px-2 py-0.5 text-[11px] font-medium text-success transition-colors hover:bg-success-light"
          >
            <Trophy className="h-3 w-3" />
            {t('crm.deals.markWon')}
          </button>
          <button
            type="button"
            onClick={(e) => { e.stopPropagation(); onMark(deal, 'lost') }}
            className="inline-flex items-center gap-1 rounded-md border border-destructive/40 px-2 py-0.5 text-[11px] font-medium text-destructive transition-colors hover:bg-destructive/10"
          >
            <XCircle className="h-3 w-3" />
            {t('crm.deals.markLost')}
          </button>
        </div>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Droppable stage column
// ---------------------------------------------------------------------------

function StageColumn({
  stage,
  deals,
  onOpen,
  onMark,
}: {
  stage: Stage
  deals: Deal[]
  onOpen: (id: string) => void
  onMark: (deal: Deal, target: 'won' | 'lost') => void
}) {
  const { t } = useTranslation()
  const { setNodeRef, isOver } = useDroppable({ id: `stage-${stage.id}`, data: { stageId: stage.id } })
  const open = !stageIsWon(stage) && !stageIsLost(stage)
  const total = deals.reduce((sum, d) => sum + (d.value ?? 0), 0)

  return (
    <div className="min-w-[280px] max-w-[280px] flex-shrink-0">
      <div
        className="mb-3 flex items-center justify-between rounded-lg px-3 py-2"
        style={{
          backgroundColor: stage.color ? `${stage.color}15` : undefined,
          borderLeft: stage.color ? `3px solid ${stage.color}` : '3px solid var(--border)',
        }}
      >
        <div>
          <p className="text-sm font-semibold">{stage.name}</p>
          <p className="text-xs text-muted-foreground">
            {t('crm.deals.dealCount', { count: deals.length })} &middot; {formatCurrency(total)}
          </p>
        </div>
        {stage.probability != null && (
          <Badge variant="outline" className="text-xs">{stage.probability}%</Badge>
        )}
      </div>

      <div
        ref={setNodeRef}
        className={cn(
          'space-y-2 rounded-lg p-1 transition-colors min-h-[80px]',
          isOver && 'bg-primary/5 ring-2 ring-primary/40'
        )}
      >
        {deals.length === 0 ? (
          <div className="rounded-lg border border-dashed border-border p-4 text-center">
            <p className="text-xs text-muted-foreground">{t('crm.deals.noDeals')}</p>
          </div>
        ) : (
          deals.map((deal) => (
            <DealCard key={deal.id} deal={deal} stageOpen={open} onOpen={onOpen} onMark={onMark} />
          ))
        )}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Pipeline view
// ---------------------------------------------------------------------------

export default function DealPipelineView() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const moveDeal = useMoveDealToStage()
  const [activeDeal, setActiveDeal] = useState<Deal | null>(null)

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
    useSensor(KeyboardSensor)
  )

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

  const wonStage = stages.find(stageIsWon)
  const lostStage = stages.find(stageIsLost)

  function applyMove(dealId: string, targetStage: Stage) {
    // Optimistic update across all cached deal lists.
    queryClient.setQueriesData<typeof dealsData>({ queryKey: ['deals'] }, (old) => {
      if (!old?.deals) return old
      return {
        ...old,
        deals: old.deals.map((d) =>
          d.id === dealId ? { ...d, stageId: targetStage.id, stageName: targetStage.name } : d
        ),
      }
    })
    moveDeal.mutate(
      { id: dealId, stage_id: targetStage.id ?? '' },
      {
        onSuccess: () => toast.success(t('crm.deals.movedTo', { stage: targetStage.name })),
        onError: () => {
          toast.error(t('crm.deals.moveError'))
          queryClient.invalidateQueries({ queryKey: ['deals'] })
        },
      }
    )
  }

  function handleDragStart(event: DragStartEvent) {
    setActiveDeal((event.active.data.current?.deal as Deal) ?? null)
  }

  function handleDragEnd(event: DragEndEvent) {
    setActiveDeal(null)
    const { active, over } = event
    if (!over) return
    const dealId = active.id as string
    const deal = active.data.current?.deal as Deal | undefined
    if (!deal) return
    const overId = String(over.id)
    const targetStageId = overId.startsWith('stage-') ? overId.slice('stage-'.length) : null
    if (!targetStageId || targetStageId === deal.stageId) return
    const targetStage = stages.find((s) => s.id === targetStageId)
    if (!targetStage) return
    applyMove(dealId, targetStage)
  }

  function handleMark(deal: Deal, target: 'won' | 'lost') {
    const targetStage = target === 'won' ? wonStage : lostStage
    if (!targetStage || !deal.id || targetStage.id === deal.stageId) return
    applyMove(deal.id, targetStage)
  }

  if (error) {
    return (
      <div className="flex items-center justify-center py-16">
        <div className="text-center">
          <p className="text-lg font-semibold text-foreground">{t('crm.deals.pipelineLoadError')}</p>
          <p className="mt-2 text-sm text-muted-foreground">
            {error instanceof Error ? error.message : t('crm.error.unexpected')}
          </p>
          <Button variant="outline" className="mt-4" onClick={() => { refetchStages(); refetchDeals() }}>
            {t('common.retry')}
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
        <p className="text-lg font-medium text-foreground">{t('crm.deals.noStages')}</p>
        <p className="mt-1 text-sm text-muted-foreground">{t('crm.deals.noStagesHint')}</p>
      </div>
    )
  }

  // Group deals by stage
  const dealsByStage = new Map<string, Deal[]>()
  for (const stage of stages) dealsByStage.set(stage.id ?? '', [])
  for (const deal of deals) {
    const list = dealsByStage.get(deal.stageId ?? '')
    if (list) list.push(deal)
  }

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCorners}
      onDragStart={handleDragStart}
      onDragEnd={handleDragEnd}
    >
      <ScrollArea className="w-full">
        <div className="flex gap-4 pb-4" style={{ minWidth: `${stages.length * 296}px` }}>
          {stages.map((stage) => (
            <StageColumn
              key={stage.id}
              stage={stage}
              deals={dealsByStage.get(stage.id ?? '') ?? []}
              onOpen={(id) => navigate(`/kontakte/pipeline/${id}`)}
              onMark={handleMark}
            />
          ))}
        </div>
        <ScrollBar orientation="horizontal" />
      </ScrollArea>

      <DragOverlay>
        {activeDeal ? (
          <DealCard deal={activeDeal} stageOpen={false} onOpen={() => {}} onMark={() => {}} isDragOverlay />
        ) : null}
      </DragOverlay>
    </DndContext>
  )
}

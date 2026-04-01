/**
 * KPI Deals widget — open deals count, pipeline value, and win rate.
 */
import { memo, useMemo } from 'react'
import { Handshake, TrendingUp, Target } from 'lucide-react'
import { useDeals } from '@/api/hooks/useDeals'
import { usePipelineStages } from '@/api/hooks/usePipelineStages'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

function fmt(n: number) {
  return new Intl.NumberFormat('de-DE', { style: 'currency', currency: 'EUR', maximumFractionDigits: 0 }).format(n)
}

function KpiDeals(_props: WidgetProps) {
  const { data, isLoading } = useDeals()
  const { data: stagesData } = usePipelineStages()

  const stats = useMemo(() => {
    const deals = (data as { deals?: Array<Record<string, unknown>>; total?: number })?.deals ?? []
    const stages = (stagesData as { stages?: Array<Record<string, unknown>> })?.stages ?? []

    const wonStageIds = new Set(stages.filter((s) => s.is_won).map((s) => s.id))
    const lostStageIds = new Set(stages.filter((s) => s.is_lost).map((s) => s.id))

    const now = new Date()
    const monthStart = new Date(now.getFullYear(), now.getMonth(), 1)

    const activeDeals = deals.filter(
      (d) => !wonStageIds.has(d.stageId as string) && !lostStageIds.has(d.stageId as string),
    )
    const pipelineValue = activeDeals.reduce((sum, d) => sum + (Number(d.value) || 0), 0)

    const closedThisMonth = deals.filter((d) => {
      const updated = new Date(d.updated_at as string)
      return updated >= monthStart && (wonStageIds.has(d.stageId as string) || lostStageIds.has(d.stageId as string))
    })

    const wonThisMonth = closedThisMonth.filter((d) => wonStageIds.has(d.stageId as string)).length
    const lostThisMonth = closedThisMonth.filter((d) => lostStageIds.has(d.stageId as string)).length
    const total = wonThisMonth + lostThisMonth
    const winRate = total > 0 ? Math.round((wonThisMonth / total) * 100) : 0

    return { pipelineValue, activeDeals: activeDeals.length, wonThisMonth, lostThisMonth, winRate }
  }, [data, stagesData])

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <div className="h-5 w-5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    )
  }

  const { pipelineValue, activeDeals, wonThisMonth, lostThisMonth, winRate } = stats

  return (
    <div className="flex h-full flex-col p-4">
      {/* Header */}
      <div className="flex items-center gap-2 mb-3">
        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-violet-500/10">
          <Handshake className="h-4 w-4 text-violet-600" />
        </div>
        <span className="text-xs font-medium text-muted-foreground">Deal-Überblick</span>
      </div>

      {/* Main KPI */}
      <p className="text-2xl font-bold text-foreground">{fmt(pipelineValue)}</p>
      <p className="text-xs text-muted-foreground mt-0.5">Pipeline-Wert ({activeDeals} offene Deals)</p>

      {/* Stats grid */}
      <div className="mt-auto grid grid-cols-3 gap-2 pt-3">
        <div className="rounded-lg bg-success/10 p-2 text-center">
          <TrendingUp className="mx-auto h-3.5 w-3.5 text-success mb-0.5" />
          <p className="text-sm font-bold text-success">{wonThisMonth}</p>
          <p className="text-[9px] text-muted-foreground">Gewonnen</p>
        </div>
        <div className="rounded-lg bg-destructive/10 p-2 text-center">
          <Target className="mx-auto h-3.5 w-3.5 text-destructive mb-0.5" />
          <p className="text-sm font-bold text-destructive">{lostThisMonth}</p>
          <p className="text-[9px] text-muted-foreground">Verloren</p>
        </div>
        <div className="rounded-lg bg-violet-500/10 p-2 text-center">
          <Handshake className="mx-auto h-3.5 w-3.5 text-violet-600 mb-0.5" />
          <p className="text-sm font-bold text-violet-600">{winRate}%</p>
          <p className="text-[9px] text-muted-foreground">Win-Rate</p>
        </div>
      </div>
    </div>
  )
}

export default memo(KpiDeals)

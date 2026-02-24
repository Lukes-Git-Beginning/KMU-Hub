/**
 * KPI Deals widget — open deals count, pipeline value, and win rate.
 */
import { memo } from 'react'
import { Handshake, TrendingUp, Target } from 'lucide-react'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

const MOCK = {
  openDeals: 12,
  pipelineValue: 237_500,
  wonThisMonth: 4,
  lostThisMonth: 1,
  winRate: 80,
  avgDealSize: 19_800,
}

function fmt(n: number) {
  return new Intl.NumberFormat('de-DE', { style: 'currency', currency: 'EUR', maximumFractionDigits: 0 }).format(n)
}

function KpiDeals(_props: WidgetProps) {
  return (
    <div className="flex h-full flex-col p-4">
      {/* Header */}
      <div className="flex items-center gap-2 mb-3">
        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-violet-500/10">
          <Handshake className="h-4 w-4 text-violet-600" />
        </div>
        <span className="text-xs font-medium text-muted-foreground">Deal-Ueberblick</span>
      </div>

      {/* Main KPI */}
      <p className="text-2xl font-bold text-foreground">{fmt(MOCK.pipelineValue)}</p>
      <p className="text-xs text-muted-foreground mt-0.5">Pipeline-Wert ({MOCK.openDeals} offene Deals)</p>

      {/* Stats grid */}
      <div className="mt-auto grid grid-cols-3 gap-2 pt-3">
        <div className="rounded-lg bg-emerald-500/10 p-2 text-center">
          <TrendingUp className="mx-auto h-3.5 w-3.5 text-emerald-600 mb-0.5" />
          <p className="text-sm font-bold text-emerald-600">{MOCK.wonThisMonth}</p>
          <p className="text-[9px] text-muted-foreground">Gewonnen</p>
        </div>
        <div className="rounded-lg bg-red-500/10 p-2 text-center">
          <Target className="mx-auto h-3.5 w-3.5 text-red-500 mb-0.5" />
          <p className="text-sm font-bold text-red-500">{MOCK.lostThisMonth}</p>
          <p className="text-[9px] text-muted-foreground">Verloren</p>
        </div>
        <div className="rounded-lg bg-violet-500/10 p-2 text-center">
          <Handshake className="mx-auto h-3.5 w-3.5 text-violet-600 mb-0.5" />
          <p className="text-sm font-bold text-violet-600">{MOCK.winRate}%</p>
          <p className="text-[9px] text-muted-foreground">Win-Rate</p>
        </div>
      </div>
    </div>
  )
}

export default memo(KpiDeals)

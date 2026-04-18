import { ArrowDownRight, ArrowUpRight, ChevronRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { DashboardKPI } from '@/api/berichte-types'

interface KPICardProps {
  kpi: DashboardKPI
  active?: boolean
  onClick?: () => void
  /**
   * For metrics where a decrease is good (response time, defect rate),
   * the caller can invert the goodness interpretation.
   */
  invertGoodness?: boolean
}

export function KPICard({ kpi, active = false, onClick, invertGoodness = false }: KPICardProps) {
  const { t } = useTranslation()
  const change = kpi.change_percent ?? 0
  const isPositive = change >= 0
  const isGood = invertGoodness ? !isPositive : isPositive

  const content = (
    <>
      <div className="mb-1 flex items-center justify-between">
        <p className="text-xs text-muted-foreground">{kpi.label}</p>
        <div className="flex items-center gap-1.5">
          {kpi.change_percent !== null && (
            <div
              className={`flex items-center gap-0.5 rounded-full px-1.5 py-0.5 text-[10px] font-medium ${
                isGood ? 'bg-success-light text-success' : 'bg-error-light text-error'
              }`}
            >
              {isPositive ? (
                <ArrowUpRight className="h-3 w-3" />
              ) : (
                <ArrowDownRight className="h-3 w-3" />
              )}
              {isPositive ? '+' : ''}
              {change}%
            </div>
          )}
          {onClick && (
            <ChevronRight
              className={`h-3.5 w-3.5 text-muted-foreground transition-transform ${
                active ? 'rotate-90' : ''
              }`}
            />
          )}
        </div>
      </div>
      <div className="mb-1 flex items-baseline gap-2">
        <span className="text-2xl font-semibold text-foreground">{kpi.value}</span>
        {kpi.unit && <span className="text-sm text-muted-foreground">{kpi.unit}</span>}
      </div>
      <p className="text-[10px] text-muted-foreground">{t('berichte.dashboard.vorMonat')}</p>
    </>
  )

  const base =
    'rounded-xl border bg-card p-4 text-left transition-all duration-200 hover:border-primary/30 hover:-translate-y-0.5 hover:shadow-md'
  const border = active ? 'border-primary ring-1 ring-primary/20' : 'border-border'

  if (onClick) {
    return (
      <button type="button" onClick={onClick} className={`group ${base} ${border}`}>
        {content}
      </button>
    )
  }
  return <div className={`${base} ${border}`}>{content}</div>
}

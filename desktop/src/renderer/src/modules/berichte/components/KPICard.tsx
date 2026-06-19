import { ArrowDownRight, ArrowUpRight, ChevronRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Line, LineChart, ResponsiveContainer, Tooltip, XAxis } from 'recharts'
import type { DashboardKPI } from '@/api/berichte-types'
import { useChartTheme } from '../utils/chartTheme'
import { usePrefersReducedMotion } from '../utils/chartMotion'

interface KPICardProps {
  kpi: DashboardKPI
  active?: boolean
  onClick?: () => void
  /**
   * For metrics where a decrease is good (response time, defect rate),
   * the caller can invert the goodness interpretation.
   */
  invertGoodness?: boolean
  /**
   * Optional mini trend series rendered as a sparkline at the bottom of the
   * card. When omitted or empty the card renders exactly as before. The `label`
   * (period) drives the hover tooltip so values are readable without a click.
   */
  sparklineData?: { label?: string; value: number }[]
}

export function KPICard({
  kpi,
  active = false,
  onClick,
  invertGoodness = false,
  sparklineData,
}: KPICardProps) {
  const { t } = useTranslation()
  const theme = useChartTheme()
  const prefersReducedMotion = usePrefersReducedMotion()
  const change = kpi.change_percent ?? 0
  const isPositive = change >= 0
  const isGood = invertGoodness ? !isPositive : isPositive
  const hasSparkline = !!sparklineData && sparklineData.length > 0

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
      {hasSparkline && (
        <div className="mt-2 h-10">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={sparklineData} margin={{ top: 4, right: 2, bottom: 2, left: 2 }}>
              <XAxis dataKey="label" hide />
              <Tooltip
                cursor={{ stroke: theme.grid, strokeWidth: 1 }}
                content={({ active, payload, label }) => {
                  if (!active || !payload?.length) return null
                  const v = Number(payload[0].value)
                  return (
                    <div className="rounded-md border border-border bg-card px-2 py-1 text-[11px] shadow-sm">
                      {label && <span className="text-muted-foreground">{label}: </span>}
                      <span className="font-medium text-foreground">
                        {v.toLocaleString('de-DE')}
                        {kpi.unit ? ` ${kpi.unit}` : ''}
                      </span>
                    </div>
                  )
                }}
              />
              <Line
                type="monotone"
                dataKey="value"
                stroke={theme.primary}
                strokeWidth={2}
                dot={false}
                activeDot={{ r: 3, fill: theme.primary }}
                isAnimationActive={!prefersReducedMotion}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      )}
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

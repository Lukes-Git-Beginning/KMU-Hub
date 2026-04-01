import { useState } from 'react'
import {
  TrendingUp,
  TrendingDown,
  AlertCircle,
  DollarSign,
  FileText,
  Clock,
  AlertTriangle,
  BarChart3,
} from 'lucide-react'
import { useFinanceDashboard } from '@/api/hooks/useFinance'
import { useFinanceUIStore, formatEUR } from '@/stores/finance'
import type { InvoiceStatus } from '@/types/finance-types'

const STATUS_COLORS: Record<InvoiceStatus, string> = {
  draft: 'bg-secondary',
  sent: 'bg-info',
  paid: 'bg-success',
  overdue: 'bg-error',
  cancelled: 'bg-muted-foreground',
}

const STATUS_LABELS: Record<InvoiceStatus, string> = {
  draft: 'Entwurf',
  sent: 'Gesendet',
  paid: 'Bezahlt',
  overdue: 'Ueberfällig',
  cancelled: 'Storniert',
}

type DatePreset = 'month' | 'quarter' | 'year' | 'last_year' | 'custom'

function getPresetRange(preset: DatePreset): { from: string; to: string } {
  const now = new Date()
  const year = now.getFullYear()
  const month = now.getMonth()

  switch (preset) {
    case 'month': {
      const start = new Date(year, month, 1)
      const end = new Date(year, month + 1, 0)
      return {
        from: start.toISOString().split('T')[0],
        to: end.toISOString().split('T')[0],
      }
    }
    case 'quarter': {
      const qStart = Math.floor(month / 3) * 3
      const start = new Date(year, qStart, 1)
      const end = new Date(year, qStart + 3, 0)
      return {
        from: start.toISOString().split('T')[0],
        to: end.toISOString().split('T')[0],
      }
    }
    case 'year':
      return { from: `${year}-01-01`, to: `${year}-12-31` }
    case 'last_year':
      return { from: `${year - 1}-01-01`, to: `${year - 1}-12-31` }
    default:
      return { from: `${year}-01-01`, to: `${year}-12-31` }
  }
}

export function FinanceDashboard() {
  const { dateRange, setDateRange } = useFinanceUIStore()
  const [preset, setPreset] = useState<DatePreset>('year')
  const { data: dashboard, isLoading } = useFinanceDashboard(
    dateRange.from,
    dateRange.to,
  )

  const handlePreset = (p: DatePreset) => {
    setPreset(p)
    if (p !== 'custom') {
      setDateRange(getPresetRange(p))
    }
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-16 text-sm text-muted-foreground">
        Lade Dashboard...
      </div>
    )
  }

  const totalInvoiced = Number(dashboard?.total_invoiced ?? 0)
  const totalPaid = Number(dashboard?.total_paid ?? 0)
  const totalOutstanding = Number(dashboard?.total_outstanding ?? 0)
  const overdueAmount = Number(dashboard?.overdue_amount ?? 0)
  const quotesPending = dashboard?.quotes_pending ?? 0
  const conversionRate = dashboard?.conversion_rate ?? 0
  const avgDealSize = Number(dashboard?.average_deal_size ?? 0)
  const revenueForecast = Number(dashboard?.revenue_forecast ?? 0)
  const statusBreakdown = dashboard?.status_breakdown ?? {} as Record<InvoiceStatus, number>
  const recentInvoices = dashboard?.recent_invoices ?? []
  const expiringQuotes = dashboard?.expiring_quotes ?? []
  const pendingDunnings = dashboard?.pending_dunnings ?? []

  // Calculate status bar widths
  const totalInvoices = Object.values(statusBreakdown).reduce(
    (sum, count) => sum + count,
    0,
  )

  return (
    <div className="space-y-6">
      {/* Date range picker */}
      <div className="flex items-center gap-3 flex-wrap">
        <div className="flex items-center gap-1.5">
          {(
            [
              { key: 'month', label: 'Dieser Monat' },
              { key: 'quarter', label: 'Dieses Quartal' },
              { key: 'year', label: 'Dieses Jahr' },
              { key: 'last_year', label: 'Letztes Jahr' },
              { key: 'custom', label: 'Benutzerdefiniert' },
            ] as const
          ).map((p) => (
            <button
              key={p.key}
              onClick={() => handlePreset(p.key)}
              className={`rounded-md px-3 py-1.5 text-xs transition-colors ${
                preset === p.key
                  ? 'bg-primary text-primary-foreground'
                  : 'bg-secondary text-muted-foreground hover:text-foreground'
              }`}
            >
              {p.label}
            </button>
          ))}
        </div>
        {preset === 'custom' && (
          <div className="flex items-center gap-2">
            <input
              type="date"
              value={dateRange.from}
              onChange={(e) =>
                setDateRange({ ...dateRange, from: e.target.value })
              }
              className="rounded-md border border-border bg-card px-2 py-1 text-xs text-foreground"
            />
            <span className="text-xs text-muted-foreground">bis</span>
            <input
              type="date"
              value={dateRange.to}
              onChange={(e) =>
                setDateRange({ ...dateRange, to: e.target.value })
              }
              className="rounded-md border border-border bg-card px-2 py-1 text-xs text-foreground"
            />
          </div>
        )}
      </div>

      {/* Revenue metric cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <MetricCard
          label="Gesamt fakturiert"
          value={formatEUR(totalInvoiced)}
          icon={TrendingUp}
          color="text-primary"
          bg="bg-primary-light"
        />
        <MetricCard
          label="Bezahlt"
          value={formatEUR(totalPaid)}
          icon={DollarSign}
          color="text-success"
          bg="bg-success-light"
        />
        <MetricCard
          label="Ausstehend"
          value={formatEUR(totalOutstanding)}
          icon={Clock}
          color="text-warning"
          bg="bg-warning-light"
        />
        <MetricCard
          label="Ueberfällig"
          value={formatEUR(overdueAmount)}
          icon={AlertCircle}
          color="text-error"
          bg="bg-error-light"
          badge={
            pendingDunnings.length > 0
              ? `${pendingDunnings.length} Mahnung${pendingDunnings.length > 1 ? 'en' : ''}`
              : undefined
          }
        />
      </div>

      {/* Pipeline metrics */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <MetricCard
          label="Offene Angebote"
          value={String(quotesPending)}
          icon={FileText}
          color="text-info"
          bg="bg-info-light"
          badge={
            expiringQuotes.length > 0
              ? `${expiringQuotes.length} bald ablaufend`
              : undefined
          }
        />
        <MetricCard
          label="Konversionsrate"
          value={`${conversionRate.toFixed(1)}%`}
          icon={TrendingUp}
          color="text-primary"
          bg="bg-primary-light"
        />
        <MetricCard
          label="Durchschn. Auftragsgröße"
          value={formatEUR(avgDealSize)}
          icon={BarChart3}
          color="text-primary"
          bg="bg-primary-light"
        />
        <MetricCard
          label="Umsatzprognose"
          value={formatEUR(revenueForecast)}
          icon={TrendingDown}
          color="text-success"
          bg="bg-success-light"
        />
      </div>

      {/* Status breakdown bar */}
      {totalInvoices > 0 && (
        <div className="rounded-lg border border-border bg-card p-4">
          <h3 className="text-sm font-medium text-foreground mb-3">
            Rechnungsstatus
          </h3>
          <div className="flex h-6 rounded-full overflow-hidden bg-secondary">
            {(
              ['paid', 'sent', 'draft', 'overdue', 'cancelled'] as InvoiceStatus[]
            ).map((status) => {
              const count = statusBreakdown[status] ?? 0
              if (count === 0) return null
              const pct = (count / totalInvoices) * 100
              return (
                <div
                  key={status}
                  className={`${STATUS_COLORS[status]} transition-all`}
                  style={{ width: `${pct}%` }}
                  title={`${STATUS_LABELS[status]}: ${count}`}
                />
              )
            })}
          </div>
          <div className="flex flex-wrap items-center gap-4 mt-2">
            {(
              ['paid', 'sent', 'draft', 'overdue', 'cancelled'] as InvoiceStatus[]
            ).map((status) => {
              const count = statusBreakdown[status] ?? 0
              if (count === 0) return null
              return (
                <span
                  key={status}
                  className="flex items-center gap-1.5 text-xs text-muted-foreground"
                >
                  <span
                    className={`h-2 w-2 rounded-full ${STATUS_COLORS[status]}`}
                  />
                  {STATUS_LABELS[status]} ({count})
                </span>
              )
            })}
          </div>
        </div>
      )}

      {/* Compact action lists */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {/* Recent invoices */}
        {recentInvoices.length > 0 && (
          <div className="rounded-lg border border-border bg-card p-4">
            <h4 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-2">
              Letzte Rechnungen
            </h4>
            <div className="space-y-2">
              {recentInvoices.slice(0, 5).map((inv) => (
                <div
                  key={inv.id}
                  className="flex items-center justify-between text-xs"
                >
                  <div className="min-w-0">
                    <p className="text-foreground font-mono truncate">
                      {inv.invoice_number}
                    </p>
                    <p className="text-[10px] text-muted-foreground truncate">
                      {inv.customer.name}
                    </p>
                  </div>
                  <span className="text-foreground font-medium ml-2 shrink-0">
                    {formatEUR(inv.tax_breakdown.gross_total)}
                  </span>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Expiring quotes */}
        {expiringQuotes.length > 0 && (
          <div className="rounded-lg border border-border bg-card p-4">
            <h4 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-2">
              Ablaufende Angebote
            </h4>
            <div className="space-y-2">
              {expiringQuotes.slice(0, 5).map((q) => (
                <div
                  key={q.id}
                  className="flex items-center justify-between text-xs"
                >
                  <div className="min-w-0">
                    <p className="text-foreground font-mono truncate">
                      {q.quote_number}
                    </p>
                    <p className="text-[10px] text-muted-foreground truncate">
                      {q.customer.name}
                    </p>
                  </div>
                  <span className="text-warning text-[10px] ml-2 shrink-0">
                    bis {new Date(q.valid_until).toLocaleDateString('de-DE')}
                  </span>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Pending dunnings */}
        {pendingDunnings.length > 0 && (
          <div className="rounded-lg border border-border bg-card p-4">
            <h4 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-2">
              Offene Mahnungen
            </h4>
            <div className="space-y-2">
              {pendingDunnings.slice(0, 5).map((d) => (
                <div
                  key={d.id}
                  className="flex items-center justify-between text-xs"
                >
                  <div className="flex items-center gap-1.5">
                    <AlertTriangle
                      className={`h-3 w-3 ${
                        d.level === 3
                          ? 'text-error'
                          : d.level === 2
                            ? 'text-orange-500'
                            : 'text-warning'
                      }`}
                    />
                    <span className="text-foreground">Stufe {d.level}</span>
                  </div>
                  <span className="text-foreground font-medium ml-2">
                    {formatEUR(d.fee)}
                  </span>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Metric card sub-component
// ---------------------------------------------------------------------------

function MetricCard({
  label,
  value,
  icon: Icon,
  color,
  bg,
  badge,
}: {
  label: string
  value: string
  icon: typeof TrendingUp
  color: string
  bg: string
  badge?: string
}) {
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <div className="flex items-center justify-between mb-2">
        <p className="text-xs text-muted-foreground">{label}</p>
        <div
          className={`flex h-8 w-8 items-center justify-center rounded-lg ${bg}`}
        >
          <Icon className={`h-4 w-4 ${color}`} />
        </div>
      </div>
      <p className={`text-xl font-semibold ${color}`}>{value}</p>
      {badge && (
        <span className="inline-block mt-1 rounded-full bg-secondary px-2 py-0.5 text-[10px] text-muted-foreground">
          {badge}
        </span>
      )}
    </div>
  )
}

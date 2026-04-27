/**
 * Billing-Tab — Cosmi-Subscription-Uebersicht fuer den Firmen-Admin.
 *
 * Sprint 1 Frontend-only mit Mock-Daten. Backend-Anbindung in Sprint 2
 * (siehe ~/.claude/plans/als-n-chstes-brauch-ich-peppy-lighthouse.md).
 *
 * Apple-Disziplin (klare Hierarchie, keine Spielereien) + Discord-Waerme
 * an einem Punkt: Hero-KPI mit NumberTicker fuer Gesamtkosten.
 */
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  CreditCard,
  Users,
  Package,
  TrendingDown,
  Receipt,
  ExternalLink,
  CheckCircle2,
  AlertCircle,
  Sparkles,
} from 'lucide-react'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import {
  MODULE_PRICES,
  type ModuleId,
  type ModuleAssignment,
  type SupportTier,
  calculateMonthly,
  formatEUR,
} from '@/lib/pricing'

// ───────────────────────── Mock data (Sprint 1) ─────────────────────────

interface MockTenant {
  planType: 'cosmi' | 'orbit'
  supportTier: SupportTier
  billingPeriodStart: string
  billingPeriodEnd: string
  totalSeats: number
  status: 'active' | 'past_due' | 'paused'
}

const MOCK_TENANT: MockTenant = {
  planType: 'cosmi',
  supportTier: 'priority',
  billingPeriodStart: '2026-04-01',
  billingPeriodEnd: '2026-04-30',
  totalSeats: 14,
  status: 'active',
}

interface MockInvoice {
  id: string
  number: string
  periodStart: string
  periodEnd: string
  amount: number
  status: 'paid' | 'open' | 'overdue'
  paidAt: string | null
}

const MOCK_INVOICES: MockInvoice[] = [
  { id: 'inv-1', number: 'COSMI-2026-04', periodStart: '2026-04-01', periodEnd: '2026-04-30', amount: 287.5, status: 'open', paidAt: null },
  { id: 'inv-2', number: 'COSMI-2026-03', periodStart: '2026-03-01', periodEnd: '2026-03-31', amount: 287.5, status: 'paid', paidAt: '2026-04-02' },
  { id: 'inv-3', number: 'COSMI-2026-02', periodStart: '2026-02-01', periodEnd: '2026-02-28', amount: 264.2, status: 'paid', paidAt: '2026-03-02' },
  { id: 'inv-4', number: 'COSMI-2026-01', periodStart: '2026-01-01', periodEnd: '2026-01-31', amount: 264.2, status: 'paid', paidAt: '2026-02-02' },
]

const INITIAL_ASSIGNMENTS: ModuleAssignment[] = [
  { moduleId: 'crm', assignedSeats: 14 },
  { moduleId: 'tasks', assignedSeats: 14 },
  { moduleId: 'finance', assignedSeats: 4 },
  { moduleId: 'calendar', assignedSeats: 14 },
  { moduleId: 'documents', assignedSeats: 14 },
  { moduleId: 'chat', assignedSeats: 14 },
  { moduleId: 'meetings', assignedSeats: 8 },
  { moduleId: 'mail', assignedSeats: 14 },
  { moduleId: 'team', assignedSeats: 14 },
  { moduleId: 'zeiterfassung', assignedSeats: 14 },
  { moduleId: 'projects', assignedSeats: 12 },
  { moduleId: 'vertraege', assignedSeats: 6 },
]

// ───────────────────────── Component ─────────────────────────

export function BillingSettingsTab() {
  const { t } = useTranslation()
  const [assignments, setAssignments] = useState<ModuleAssignment[]>(INITIAL_ASSIGNMENTS)
  const [supportTier] = useState<SupportTier>(MOCK_TENANT.supportTier)

  const summary = useMemo(
    () => calculateMonthly(assignments, MOCK_TENANT.totalSeats, supportTier),
    [assignments, supportTier],
  )

  const activeModuleCount = summary.perModule.length
  const totalModuleSeats = summary.perModule.reduce((sum, m) => sum + m.seats, 0)

  const toggleModule = (moduleId: ModuleId) => {
    setAssignments((prev) => {
      const existing = prev.find((a) => a.moduleId === moduleId)
      if (existing && existing.assignedSeats > 0) {
        return prev.map((a) => (a.moduleId === moduleId ? { ...a, assignedSeats: 0 } : a))
      }
      return prev.find((a) => a.moduleId === moduleId)
        ? prev.map((a) => (a.moduleId === moduleId ? { ...a, assignedSeats: MOCK_TENANT.totalSeats } : a))
        : [...prev, { moduleId, assignedSeats: MOCK_TENANT.totalSeats }]
    })
  }

  // Module-IDs in stable order for the table (alle aus MODULE_PRICES)
  const allModuleIds = Object.keys(MODULE_PRICES) as ModuleId[]

  return (
    <div className="max-w-4xl">
      {/* ── Header ───────────────────────────── */}
      <div className="mb-1 flex items-center gap-2">
        <h2 className="text-foreground">{t('settings.billing.title')}</h2>
        <Badge variant="secondary" className="font-mono text-[10px] uppercase">
          {MOCK_TENANT.planType}
        </Badge>
      </div>
      <p className="mb-8 text-sm text-muted-foreground">{t('settings.billing.subtitle')}</p>

      {/* ── Hero KPI: Gesamtkosten/Monat ─────────────────────── */}
      <div className="mb-6 rounded-2xl border border-border bg-gradient-to-br from-primary-light/40 to-card p-6">
        <p className="text-xs uppercase tracking-wider text-muted-foreground">
          {t('settings.billing.monthlyTotal')}
        </p>
        <div className="mt-2 flex items-end gap-3">
          <p className="text-4xl font-semibold text-foreground tabular-nums">
            {formatEUR(summary.monthlyTotal)}
          </p>
          <p className="pb-1 text-sm text-muted-foreground">{t('settings.billing.perMonth')}</p>
        </div>
        <div className="mt-3 flex items-center gap-2 text-xs text-muted-foreground">
          <CreditCard className="h-3.5 w-3.5" />
          <span>
            {t('settings.billing.nextBilling', {
              date: new Date(MOCK_TENANT.billingPeriodEnd).toLocaleDateString('de-DE', {
                day: '2-digit',
                month: 'long',
                year: 'numeric',
              }),
            })}
          </span>
        </div>
      </div>

      {/* ── Stat-Cards ───────────────────────── */}
      <div className="mb-8 grid grid-cols-1 gap-4 sm:grid-cols-3">
        <StatCard
          icon={Users}
          label={t('settings.billing.userSeats')}
          value={String(MOCK_TENANT.totalSeats)}
          hint={t('settings.billing.activeUsers')}
        />
        <StatCard
          icon={Package}
          label={t('settings.billing.activeModules')}
          value={String(activeModuleCount)}
          hint={t('settings.billing.totalAssignments', { count: totalModuleSeats })}
        />
        <StatCard
          icon={TrendingDown}
          label={t('settings.billing.volumeDiscount')}
          value={`-${Math.round(summary.tier.discount * 100)}%`}
          hint={summary.tier.label}
          accent={summary.tier.discount > 0}
        />
      </div>

      {/* ── Modul-Tabelle mit Toggles ────────────── */}
      <section className="mb-8">
        <div className="mb-3 flex items-center justify-between">
          <h3 className="text-foreground">{t('settings.billing.modules.title')}</h3>
          <p className="text-xs text-muted-foreground">{t('settings.billing.modules.hint')}</p>
        </div>
        <div className="overflow-hidden rounded-xl border border-border bg-card">
          <div className="grid grid-cols-[1fr_120px_120px_140px_60px] gap-3 border-b border-border bg-secondary/40 px-4 py-2.5 text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
            <span>{t('settings.billing.modules.col.name')}</span>
            <span className="text-right tabular-nums">{t('settings.billing.modules.col.pricePerSeat')}</span>
            <span className="text-right tabular-nums">{t('settings.billing.modules.col.seats')}</span>
            <span className="text-right tabular-nums">{t('settings.billing.modules.col.subtotal')}</span>
            <span className="text-right">{t('settings.billing.modules.col.active')}</span>
          </div>

          {allModuleIds.map((moduleId) => {
            const assignment = assignments.find((a) => a.moduleId === moduleId)
            const seats = assignment?.assignedSeats ?? 0
            const isActive = seats > 0
            const pricePerSeat = MODULE_PRICES[moduleId]
            const subtotal = pricePerSeat * seats

            return (
              <div
                key={moduleId}
                className={`grid grid-cols-[1fr_120px_120px_140px_60px] items-center gap-3 border-b border-border-muted px-4 py-2.5 transition-colors hover:bg-secondary/20 last:border-b-0 ${
                  !isActive ? 'opacity-50' : ''
                }`}
              >
                <span className="text-sm text-foreground">
                  {t(`dashboard.modules.${moduleId}.name`, { defaultValue: moduleId })}
                </span>
                <span className="text-right text-sm tabular-nums text-muted-foreground">
                  {formatEUR(pricePerSeat)}
                </span>
                <span className="text-right text-sm tabular-nums text-muted-foreground">
                  {isActive ? `${seats} × ` : '—'}
                </span>
                <span className="text-right text-sm font-medium tabular-nums text-foreground">
                  {isActive ? formatEUR(subtotal) : '—'}
                </span>
                <div className="flex justify-end">
                  <Switch checked={isActive} onCheckedChange={() => toggleModule(moduleId)} />
                </div>
              </div>
            )
          })}

          {/* ── Total breakdown ─────── */}
          <div className="border-t-2 border-border bg-secondary/30 px-4 py-3 text-sm">
            <div className="flex items-center justify-between">
              <span className="text-muted-foreground">{t('settings.billing.total.modulesGross')}</span>
              <span className="tabular-nums text-foreground">{formatEUR(summary.modulesGross)}</span>
            </div>
            {summary.tier.discount > 0 && (
              <div className="mt-1 flex items-center justify-between">
                <span className="text-muted-foreground">
                  {t('settings.billing.total.volumeDiscount', {
                    percent: Math.round(summary.tier.discount * 100),
                  })}
                </span>
                <span className="tabular-nums text-success">
                  {formatEUR(summary.volumeDiscountAmount)}
                </span>
              </div>
            )}
            <div className="mt-1 flex items-center justify-between">
              <span className="text-muted-foreground">
                {t('settings.billing.total.support', { tier: t(`settings.billing.support.${supportTier}`) })}
              </span>
              <span className="tabular-nums text-foreground">{formatEUR(summary.supportFee)}</span>
            </div>
            <Separator className="my-2" />
            <div className="flex items-center justify-between text-base font-semibold">
              <span className="text-foreground">{t('settings.billing.total.monthlyTotal')}</span>
              <span className="tabular-nums text-foreground">{formatEUR(summary.monthlyTotal)}</span>
            </div>
          </div>
        </div>
      </section>

      {/* ── Support-Tier ─────────────── */}
      <section className="mb-8">
        <div className="mb-3 flex items-center justify-between">
          <h3 className="text-foreground">{t('settings.billing.support.title')}</h3>
          <Button variant="outline" size="sm" disabled>
            <Sparkles className="mr-1.5 h-3.5 w-3.5" />
            {t('settings.billing.support.upgrade')}
          </Button>
        </div>
        <div className="rounded-xl border border-border bg-card p-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-light">
              <CheckCircle2 className="h-5 w-5 text-primary" />
            </div>
            <div className="flex-1">
              <p className="text-sm font-medium text-foreground">
                {t(`settings.billing.support.${supportTier}`)}
              </p>
              <p className="text-xs text-muted-foreground">
                {t(`settings.billing.support.${supportTier}.description`)}
              </p>
            </div>
            <span className="text-sm font-medium tabular-nums text-foreground">
              {formatEUR(summary.supportFee)}
              <span className="text-xs text-muted-foreground"> / {t('settings.billing.month')}</span>
            </span>
          </div>
        </div>
      </section>

      {/* ── Cosmi-Rechnungs-Historie ─────────────── */}
      <section>
        <div className="mb-3 flex items-center justify-between">
          <h3 className="text-foreground">{t('settings.billing.history.title')}</h3>
          <p className="text-xs text-muted-foreground">{t('settings.billing.history.subtitle')}</p>
        </div>
        <div className="overflow-hidden rounded-xl border border-border bg-card">
          <div className="grid grid-cols-[120px_1fr_140px_120px_40px] gap-3 border-b border-border bg-secondary/40 px-4 py-2.5 text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
            <span>{t('settings.billing.history.col.number')}</span>
            <span>{t('settings.billing.history.col.period')}</span>
            <span className="text-right tabular-nums">{t('settings.billing.history.col.amount')}</span>
            <span>{t('settings.billing.history.col.status')}</span>
            <span />
          </div>
          {MOCK_INVOICES.map((inv) => (
            <div
              key={inv.id}
              className="grid grid-cols-[120px_1fr_140px_120px_40px] items-center gap-3 border-b border-border-muted px-4 py-3 transition-colors hover:bg-secondary/20 last:border-b-0"
            >
              <span className="font-mono text-xs text-foreground">{inv.number}</span>
              <span className="text-sm text-muted-foreground">
                {new Date(inv.periodStart).toLocaleDateString('de-DE', { day: '2-digit', month: 'short' })}
                {' – '}
                {new Date(inv.periodEnd).toLocaleDateString('de-DE', { day: '2-digit', month: 'short', year: 'numeric' })}
              </span>
              <span className="text-right text-sm font-medium tabular-nums text-foreground">
                {formatEUR(inv.amount)}
              </span>
              <InvoiceStatus status={inv.status} />
              <button
                type="button"
                className="rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
                aria-label={t('settings.billing.history.openPdf')}
                disabled
              >
                <ExternalLink className="h-3.5 w-3.5" />
              </button>
            </div>
          ))}
        </div>
      </section>
    </div>
  )
}

// ───────────────────────── Subcomponents ─────────────────────────

function StatCard({
  icon: Icon,
  label,
  value,
  hint,
  accent,
}: {
  icon: typeof Users
  label: string
  value: string
  hint: string
  accent?: boolean
}) {
  return (
    <div className="rounded-xl border border-border bg-card p-4">
      <div className="mb-2 flex items-center justify-between">
        <p className="text-[11px] uppercase tracking-wider text-muted-foreground">{label}</p>
        <div className={`flex h-7 w-7 items-center justify-center rounded-lg ${accent ? 'bg-success-light' : 'bg-secondary'}`}>
          <Icon className={`h-3.5 w-3.5 ${accent ? 'text-success' : 'text-muted-foreground'}`} />
        </div>
      </div>
      <p className={`text-2xl font-semibold tabular-nums ${accent ? 'text-success' : 'text-foreground'}`}>
        {value}
      </p>
      <p className="mt-0.5 text-xs text-muted-foreground">{hint}</p>
    </div>
  )
}

function InvoiceStatus({ status }: { status: 'paid' | 'open' | 'overdue' }) {
  const { t } = useTranslation()
  const map = {
    paid: { icon: CheckCircle2, color: 'bg-success-light text-success', labelKey: 'settings.billing.history.status.paid' },
    open: { icon: Receipt, color: 'bg-info-light text-info', labelKey: 'settings.billing.history.status.open' },
    overdue: { icon: AlertCircle, color: 'bg-error-light text-error', labelKey: 'settings.billing.history.status.overdue' },
  }
  const cfg = map[status]
  const Icon = cfg.icon
  return (
    <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${cfg.color}`}>
      <Icon className="h-3 w-3" />
      {t(cfg.labelKey)}
    </span>
  )
}

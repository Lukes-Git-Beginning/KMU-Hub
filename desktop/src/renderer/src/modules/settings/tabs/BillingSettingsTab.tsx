/**
 * BillingSettingsTab — Cosmi-Abrechnung Read-only-Insight-Dashboard.
 *
 * stop-the-slop preflight:
 * purpose      — Billing-Übersicht für Firmen-Admin, täglicher Gebrauch
 * tone         — industrial / utilitarian (Apple-Disziplin, kein Joy im Daily-Flow)
 * constraints  — Kein Backend-Call, kein NumberTicker, keine Card-in-Card, keine Hover-Lift-Schatten
 * memorable    — Klare Kosten-Hierarchie + handlungsrelevante Inaktivitäts-Hinweise
 *
 * Joy-Moment NUR beim Verwerfen-Toast (Phase 2: Apply-Toast nach echter Modul-Zuteilung).
 */
import { useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import {
  CreditCard,
  Receipt,
  ExternalLink,
  CheckCircle2,
  AlertCircle,
  ArrowUpRight,
  Lightbulb,
  AlertTriangle,
  ChevronRight,
  X,
} from 'lucide-react'
import { toast } from 'sonner'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import {
  calculateMonthly,
  formatEUR,
  DEFAULT_INSIGHT_SETTINGS,
  type InsightSettings,
  type ModuleAssignment,
} from '@/lib/pricing'
import {
  useTenant,
  useModuleUsage,
  useInsightSettings,
  useUpdateInsightSettings,
  useRecommendations,
  useDismissRecommendation,
  useDismissedRecommendations,
  useCostSavingsTracker,
  useBillingRecommendationNotifier,
  useInvoiceHistory,
} from '@/api/hooks/useBilling'
import { useNavigationStore } from '@/stores/navigation'
import { InlineStat } from '@/components/shared'

// ───────────────────────── Statische Basis-Assignments (für Kosten-Berechnung) ─────────────────────────
// Entspricht den aktuell zugeteilten Modul-Seats — read-only, kein Toggle mehr.

const BILLING_ASSIGNMENTS: ModuleAssignment[] = [
  { moduleId: 'crm', assignedSeats: 14 },
  { moduleId: 'tasks', assignedSeats: 14 },
  { moduleId: 'finance', assignedSeats: 4 },
  { moduleId: 'calendar', assignedSeats: 14 },
  { moduleId: 'documents', assignedSeats: 14 },
  { moduleId: 'chat', assignedSeats: 14 },
  { moduleId: 'meetings', assignedSeats: 8 },
  { moduleId: 'mail', assignedSeats: 14 },
  { moduleId: 'dialer', assignedSeats: 8 },
  { moduleId: 'team', assignedSeats: 14 },
  { moduleId: 'zeiterfassung', assignedSeats: 14 },
  { moduleId: 'projects', assignedSeats: 12 },
  { moduleId: 'vertraege', assignedSeats: 6 },
]

// ───────────────────────── Helpers ─────────────────────────

function formatDate(iso: string, locale: string, monthStyle: 'short' | 'long' = 'short'): string {
  return new Intl.DateTimeFormat(locale || 'de-DE', {
    day: '2-digit',
    month: monthStyle,
    year: 'numeric',
  }).format(new Date(iso))
}

// ───────────────────────── Component ─────────────────────────

export function BillingSettingsTab() {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const navigationStore = useNavigationStore()

  // Notification-Layer: feuert Bell-Notification wenn neue Recommendations erscheinen.
  useBillingRecommendationNotifier()

  // Data hooks
  const tenant = useTenant()
  const moduleUsage = useModuleUsage()
  const insightSettings = useInsightSettings()
  const updateSettings = useUpdateInsightSettings()
  const recommendations = useRecommendations()
  const dismissRecommendation = useDismissRecommendation()
  const dismissedRecommendations = useDismissedRecommendations()
  const savings = useCostSavingsTracker()
  const invoiceHistory = useInvoiceHistory()

  const tenantData = tenant.data ?? {
    planType: 'cosmi' as const,
    supportTier: 'priority' as const,
    billingPeriodEnd: '2026-04-30',
    totalSeats: 14,
    status: 'active' as const,
  }

  const settings = insightSettings.data ?? DEFAULT_INSIGHT_SETTINGS

  // Billing summary (berechnet aus fixen Assignments für totals — read-only)
  const summary = calculateMonthly(BILLING_ASSIGNMENTS, tenantData.totalSeats, tenantData.supportTier)
  const activeModuleCount = summary.perModule.length
  const totalModuleSeats = summary.perModule.reduce((sum, m) => sum + m.seats, 0)

  const handleSettingsChange = useCallback(
    (patch: Partial<InsightSettings>) => {
      updateSettings.mutate({ ...settings, ...patch })
    },
    [settings, updateSettings],
  )

  const handleDismiss = useCallback(
    (hash: string) => {
      dismissRecommendation.mutate(hash)
      toast.success(t('settings.billing.recommendations.dismissed'))
    },
    [dismissRecommendation, t],
  )

  const handleAdjust = useCallback(
    (moduleId: string, userIds: string[]) => {
      navigationStore.setIntent({
        type: 'open-team-modulzuteilung',
        data: { moduleId, userIds },
      })
      void navigate('/team')
    },
    [navigationStore, navigate],
  )

  const handleTeamLink = useCallback(() => {
    navigationStore.setIntent({
      type: 'open-team-modulzuteilung',
      data: {},
    })
    void navigate('/team')
  }, [navigationStore, navigate])

  return (
    <div className="max-w-4xl">
      {/* ── Header ───────────────────────────── */}
      <div className="mb-1 flex items-center gap-2">
        <h2 className="text-foreground" style={{ textWrap: 'balance' }}>
          {t('settings.billing.title')}
        </h2>
        <Badge variant="secondary" className="font-mono text-[10px] uppercase" translate="no">
          {tenantData.planType}
        </Badge>
      </div>
      <p className="mb-8 text-sm text-muted-foreground" style={{ textWrap: 'pretty' }}>
        {t('settings.billing.subtitle')}
      </p>

      {/* ── Total-Block (typografisch, kein Card) ── */}
      <div className="mb-8 border-b border-border pb-6">
        <p className="text-[11px] uppercase tracking-wider text-muted-foreground">
          {t('settings.billing.monthlyTotal')}
        </p>
        <div className="mt-1 flex items-baseline gap-2">
          {/* KEIN NumberTicker — Apple-Disziplin */}
          <p className="text-4xl font-semibold tabular-nums text-foreground" style={{ textWrap: 'balance' }}>
            {formatEUR(summary.monthlyTotal)}
          </p>
          <p className="text-sm text-muted-foreground">{t('settings.billing.perMonth')}</p>
        </div>
        <p className="mt-1.5 flex items-center gap-1.5 text-xs text-muted-foreground">
          <CreditCard className="h-3 w-3" aria-hidden="true" />
          <span>
            {t('settings.billing.nextBilling', {
              date: formatDate(tenantData.billingPeriodEnd, i18n.language, 'long'),
            })}
          </span>
        </p>

        <dl className="mt-5 flex flex-wrap gap-x-10 gap-y-3 text-sm">
          <InlineStat
            label={t('settings.billing.userSeats')}
            value={String(tenantData.totalSeats)}
            hint={t('settings.billing.activeUsers')}
          />
          <InlineStat
            label={t('settings.billing.activeModules')}
            value={String(activeModuleCount)}
            hint={t('settings.billing.totalAssignments', { count: totalModuleSeats })}
          />
          <InlineStat
            label={t('settings.billing.volumeDiscount')}
            value={summary.tier.discount > 0 ? `−${Math.round(summary.tier.discount * 100)}%` : '—'}
            hint={summary.tier.label}
            accent={summary.tier.discount > 0 ? 'success' : 'default'}
          />
        </dl>
      </div>

      {/* ── Modul-Auslastung ─────────────────── */}
      <section className="mb-8" aria-labelledby="section-module-usage">
        <div className="mb-3 flex items-center justify-between">
          <h3 id="section-module-usage" className="text-foreground" style={{ textWrap: 'balance' }}>
            {t('settings.billing.modules.title')}
          </h3>
        </div>
        <div className="overflow-hidden rounded-xl border border-border bg-card">
          {/* Spalten: Modul | Pro Seat | Zugeteilt | Aktiv (days)T | Auslastung % | Status */}
          <div className="grid grid-cols-[1fr_100px_100px_120px_120px_80px] gap-3 border-b border-border bg-secondary/40 px-4 py-2.5 text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
            <span>{t('settings.billing.modules.col.name')}</span>
            <span className="text-right tabular-nums">{t('settings.billing.modules.col.pricePerSeat')}</span>
            <span className="text-right tabular-nums">{t('settings.billing.modules.col.seats')}</span>
            <span className="text-right tabular-nums">
              {t('settings.billing.modules.usage.col.activeNDays', { days: settings.observationDays })}
            </span>
            <span className="text-right tabular-nums">
              {t('settings.billing.modules.usage.col.utilization')}
            </span>
            <span className="text-right">{t('settings.billing.modules.col.active')}</span>
          </div>

          {(moduleUsage.data ?? []).map((stat) => {
            const isWarning = stat.utilizationPercent < settings.utilizationThreshold
            const pricePerSeat = summary.perModule.find((m) => m.moduleId === stat.moduleId)?.pricePerSeat
            const isAssigned = stat.assignedUserCount > 0

            return (
              <div
                key={stat.moduleId}
                className={`grid grid-cols-[1fr_100px_100px_120px_120px_80px] items-center gap-3 border-b border-border-muted px-4 py-2.5 last:border-b-0 ${
                  isWarning && isAssigned ? 'bg-warning/5' : ''
                }`}
              >
                <span className="text-sm text-foreground">
                  {t(`dashboard.modules.${stat.moduleId}.name`, { defaultValue: stat.moduleId })}
                </span>
                <span className="text-right text-sm tabular-nums text-muted-foreground">
                  {pricePerSeat !== undefined ? formatEUR(pricePerSeat) : '—'}
                </span>
                <span className="text-right text-sm tabular-nums text-muted-foreground">
                  {stat.assignedUserCount > 0 ? stat.assignedUserCount : '—'}
                </span>
                <span className="text-right text-sm tabular-nums text-muted-foreground">
                  {isAssigned ? stat.activeUserCount : '—'}
                </span>
                <span className="flex items-center justify-end gap-1 text-sm tabular-nums">
                  {isWarning && isAssigned && (
                    <AlertTriangle
                      className="h-3 w-3 shrink-0 text-warning"
                      aria-hidden="true"
                    />
                  )}
                  <span className={isWarning && isAssigned ? 'text-warning' : 'text-muted-foreground'}>
                    {isAssigned ? `${stat.utilizationPercent} %` : '—'}
                  </span>
                </span>
                <div className="flex justify-end">
                  {isAssigned ? (
                    <span className="inline-flex items-center rounded-full bg-success-light px-2 py-0.5 text-[10px] font-medium text-success">
                      {t('settings.billing.modules.statusActive')}
                    </span>
                  ) : (
                    <span className="text-sm text-muted-foreground" aria-label="Nicht zugeteilt">
                      {t('settings.billing.modules.statusInactive')}
                    </span>
                  )}
                </div>
              </div>
            )
          })}

          {/* Footer-Link zum Team-Bereich */}
          <div className="border-t border-border bg-secondary/20 px-4 py-2.5">
            <button
              type="button"
              onClick={handleTeamLink}
              className="flex items-center gap-1 text-xs text-muted-foreground transition-colors hover:text-foreground focus-visible:rounded-sm focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
            >
              {t('settings.billing.modules.tableFooter')}
              <ChevronRight className="h-3 w-3" aria-hidden="true" />
            </button>
          </div>
        </div>
      </section>

      {/* ── Optimierungs-Vorschläge ─────────────── */}
      <section className="mb-8" aria-labelledby="section-recommendations">
        <div className="mb-3 flex items-center justify-between">
          <h3 id="section-recommendations" className="text-foreground" style={{ textWrap: 'balance' }}>
            {t('settings.billing.recommendations.title')}
          </h3>
        </div>

        {/* Cost-Saving-Counter — nur wenn >0 gespart wurde */}
        {(savings.data?.totalSaved ?? 0) > 0 && (
          <p className="mb-3 text-sm text-muted-foreground tabular-nums">
            {t('settings.billing.recommendations.savings.applied', {
              monthly: formatEUR(savings.data!.totalSaved),
              count: savings.data!.applyCount,
            })}
          </p>
        )}

        <div className="overflow-hidden rounded-xl border border-border bg-card">
          {(recommendations.data ?? []).length === 0 ? (
            /* Empty State — drei Zustände */
            <RecommendationsEmptyState
              recommendationsEnabled={settings.recommendationsEnabled}
              dismissedCount={dismissedRecommendations.data?.size ?? 0}
            />
          ) : (
            (recommendations.data ?? []).slice(0, 3).map((rec, idx, arr) => (
              <article
                key={rec.hash}
                className={`grid grid-cols-[40px_1fr_auto] items-start gap-4 px-4 py-4 ${
                  idx < arr.length - 1 ? 'border-b border-border-muted' : ''
                }`}
                aria-labelledby={`rec-title-${rec.hash}`}
              >
                {/* Icon-Spalte */}
                <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-warning/10">
                  <Lightbulb className="h-4 w-4 text-warning" aria-hidden="true" />
                </div>

                {/* Text-Block */}
                <div className="min-w-0">
                  <h4
                    id={`rec-title-${rec.hash}`}
                    className="text-sm font-medium text-foreground"
                    style={{ textWrap: 'balance' }}
                  >
                    {t('settings.billing.recommendations.itemTitle', {
                      module: rec.moduleName,
                      count: rec.inactiveUsers.length,
                      assigned: rec.inactiveUsers.length + (moduleUsage.data?.find(s => s.moduleId === rec.moduleId)?.activeUserCount ?? 0),
                    })}
                  </h4>
                  <p className="mt-0.5 text-xs text-muted-foreground">
                    {rec.inactiveUsers.map((u) => u.userName).join(', ')}
                    {' · '}
                    {t('settings.billing.recommendations.timeframe', { days: rec.observationDays })}
                  </p>
                  <p className="mt-1 text-xs font-medium text-success">
                    {t('settings.billing.recommendations.savings', {
                      monthly: formatEUR(rec.monthlySavings),
                      yearly: formatEUR(rec.yearlySavings),
                    })}
                  </p>
                </div>

                {/* Action-Buttons — rechts */}
                <div className="flex shrink-0 items-center gap-1">
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-8 gap-1 px-2 text-xs"
                    onClick={() => handleAdjust(rec.moduleId, rec.inactiveUsers.map((u) => u.userId))}
                  >
                    {t('settings.billing.recommendations.adjust')}
                    <ChevronRight className="h-3 w-3" aria-hidden="true" />
                  </Button>
                  <button
                    type="button"
                    onClick={() => handleDismiss(rec.hash)}
                    aria-label={t('settings.billing.recommendations.dismissAria')}
                    className="flex h-8 w-8 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
                  >
                    <X className="h-3.5 w-3.5" aria-hidden="true" />
                  </button>
                </div>
              </article>
            ))
          )}
        </div>
      </section>

      {/* ── Support-Tier ─────────────── */}
      <section className="mb-8">
        <div className="mb-3 flex items-center justify-between">
          <h3 className="text-foreground" style={{ textWrap: 'balance' }}>
            {t('settings.billing.support.title')}
          </h3>
          <Button variant="outline" size="sm" disabled>
            {t('settings.billing.support.upgrade')}
            <ArrowUpRight className="ml-1.5 h-3.5 w-3.5" aria-hidden="true" />
          </Button>
        </div>
        <div className="flex items-baseline justify-between gap-4 border-y border-border py-4">
          <div className="min-w-0 flex-1">
            <p className="text-sm font-medium text-foreground">
              {t(`settings.billing.support.${tenantData.supportTier}`)}
            </p>
            <p className="mt-0.5 text-xs text-muted-foreground" style={{ textWrap: 'pretty' }}>
              {t(`settings.billing.support.${tenantData.supportTier}.description`)}
            </p>
          </div>
          <span className="shrink-0 text-sm font-medium tabular-nums text-foreground">
            {formatEUR(summary.supportFee)}
            <span className="text-xs font-normal text-muted-foreground"> / {t('settings.billing.month')}</span>
          </span>
        </div>
      </section>

      {/* ── Cosmi-Rechnungs-Historie ─────────────── */}
      <section className="mb-8" aria-labelledby="section-invoice-history">
        <div className="mb-3 flex items-center justify-between">
          <h3 id="section-invoice-history" className="text-foreground" style={{ textWrap: 'balance' }}>
            {t('settings.billing.history.title')}
          </h3>
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
          {(invoiceHistory.data ?? []).map((inv) => (
            <div
              key={inv.id}
              className="grid grid-cols-[120px_1fr_140px_120px_40px] items-center gap-3 border-b border-border-muted px-4 py-3 transition-colors hover:bg-secondary/20 last:border-b-0"
            >
              <span className="font-mono text-xs text-foreground">{inv.number}</span>
              <span className="text-sm text-muted-foreground">
                {new Intl.DateTimeFormat(i18n.language, { day: '2-digit', month: 'short' }).format(
                  new Date(inv.periodStart),
                )}
                {' – '}
                {formatDate(inv.periodEnd, i18n.language, 'short')}
              </span>
              <span className="text-right text-sm font-medium tabular-nums text-foreground">
                {formatEUR(inv.amount)}
              </span>
              <InvoiceStatus status={inv.status} />
              <button
                type="button"
                className="rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
                aria-label={t('settings.billing.history.openPdf')}
                disabled
              >
                <ExternalLink className="h-3.5 w-3.5" />
              </button>
            </div>
          ))}
        </div>
      </section>

      {/* ── Insight-Einstellungen ─────────────── */}
      <section aria-labelledby="section-insight-settings">
        <div className="mb-3">
          <h3 id="section-insight-settings" className="text-foreground" style={{ textWrap: 'balance' }}>
            {t('settings.billing.insightSettings.title')}
          </h3>
          <p className="mt-0.5 text-xs text-muted-foreground" style={{ textWrap: 'pretty' }}>
            {t('settings.billing.insightSettings.subtitle')}
          </p>
        </div>

        <div className="space-y-5 rounded-xl border border-border bg-card px-4 py-5">
          {/* Beobachtungs-Zeitraum */}
          <div className="grid grid-cols-[200px_1fr] items-start gap-4">
            <div>
              <p className="text-sm font-medium text-foreground">
                {t('settings.billing.insightSettings.observationDays')}
              </p>
              <p className="mt-0.5 text-xs text-muted-foreground" style={{ textWrap: 'pretty' }}>
                {t('settings.billing.insightSettings.observationDaysHelp')}
              </p>
            </div>
            <fieldset className="flex items-center gap-4">
              <legend className="sr-only">{t('settings.billing.insightSettings.observationDays')}</legend>
              {([30, 90, 180] as const).map((days) => (
                <label
                  key={days}
                  className="flex cursor-pointer items-center gap-1.5 text-sm"
                >
                  <input
                    type="radio"
                    name="observationDays"
                    value={days}
                    checked={settings.observationDays === days}
                    onChange={() => handleSettingsChange({ observationDays: days })}
                    className="accent-foreground"
                  />
                  <span className={settings.observationDays === days ? 'font-medium text-foreground' : 'text-muted-foreground'}>
                    {days}
                  </span>
                </label>
              ))}
            </fieldset>
          </div>

          <Separator />

          {/* Auslastungs-Schwelle */}
          <div className="grid grid-cols-[200px_1fr] items-start gap-4">
            <div>
              <p className="text-sm font-medium text-foreground">
                {t('settings.billing.insightSettings.threshold')}
              </p>
              <p className="mt-0.5 text-xs text-muted-foreground" style={{ textWrap: 'pretty' }}>
                {t('settings.billing.insightSettings.thresholdHelp')}
              </p>
            </div>
            <div className="flex items-center gap-3">
              <input
                type="range"
                min={0}
                max={100}
                step={5}
                value={settings.utilizationThreshold}
                onChange={(e) => handleSettingsChange({ utilizationThreshold: Number(e.target.value) })}
                className="w-full max-w-[200px] accent-foreground"
                aria-valuemin={0}
                aria-valuemax={100}
                aria-valuenow={settings.utilizationThreshold}
                aria-valuetext={`${settings.utilizationThreshold}%`}
                aria-label={t('settings.billing.insightSettings.threshold')}
              />
              <span className="min-w-[44px] rounded-md border border-border bg-secondary/40 px-2 py-0.5 text-center text-sm tabular-nums text-foreground">
                {settings.utilizationThreshold} %
              </span>
            </div>
          </div>

          <Separator />

          {/* Mindest-Inaktivität */}
          <div className="grid grid-cols-[200px_1fr] items-start gap-4">
            <div>
              <p className="text-sm font-medium text-foreground">
                {t('settings.billing.insightSettings.minInactivity')}
              </p>
              <p className="mt-0.5 text-xs text-muted-foreground" style={{ textWrap: 'pretty' }}>
                {t('settings.billing.insightSettings.minInactivityHelp')}
              </p>
            </div>
            <div className="flex items-center gap-2">
              <input
                type="number"
                min={1}
                max={365}
                value={settings.minInactivityDays}
                onChange={(e) => {
                  const v = Number(e.target.value)
                  if (v >= 1 && v <= 365) handleSettingsChange({ minInactivityDays: v })
                }}
                className="max-w-[6ch] rounded-md border border-border bg-background px-2 py-1 text-sm tabular-nums text-foreground focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
                aria-label={t('settings.billing.insightSettings.minInactivity')}
              />
              <span className="text-sm text-muted-foreground">
                {t('settings.billing.insightSettings.minInactivityUnit')}
              </span>
            </div>
          </div>

          <Separator />

          {/* Empfehlungen aktiv */}
          <div className="grid grid-cols-[200px_1fr] items-start gap-4">
            <div>
              <p className="text-sm font-medium text-foreground">
                {t('settings.billing.insightSettings.recommendationsEnabled')}
              </p>
              <p className="mt-0.5 text-xs text-muted-foreground" style={{ textWrap: 'pretty' }}>
                {t('settings.billing.insightSettings.recommendationsEnabledHelp')}
              </p>
            </div>
            <Switch
              checked={settings.recommendationsEnabled}
              onCheckedChange={(v) => handleSettingsChange({ recommendationsEnabled: v })}
              aria-label={t('settings.billing.insightSettings.recommendationsEnabled')}
            />
          </div>

          <Separator />

          {/* Erfolgs-Toast nach Anpassung */}
          <div className="grid grid-cols-[200px_1fr] items-start gap-4">
            <div>
              <p className="text-sm font-medium text-foreground">
                {t('settings.billing.insightSettings.toastOnApply')}
              </p>
              <p className="mt-0.5 text-xs text-muted-foreground" style={{ textWrap: 'pretty' }}>
                {t('settings.billing.insightSettings.toastOnApplyHelp')}
              </p>
            </div>
            <Switch
              checked={settings.toastOnApply}
              onCheckedChange={(v) => handleSettingsChange({ toastOnApply: v })}
              aria-label={t('settings.billing.insightSettings.toastOnApply')}
            />
          </div>
        </div>
      </section>
    </div>
  )
}

// ───────────────────────── Subcomponents ─────────────────────────

interface RecommendationsEmptyStateProps {
  recommendationsEnabled: boolean
  dismissedCount: number
}

/**
 * Drei EmptyState-Zustände für die Recommendations-Sektion:
 * 1. Feature deaktiviert → Hinweis + Action-Link
 * 2. Alle verworfen → Hinweis mit Schwellen-Link
 * 3. Alles im grünen Bereich → positives Feedback
 */
function RecommendationsEmptyState({ recommendationsEnabled, dismissedCount }: RecommendationsEmptyStateProps) {
  const { t } = useTranslation()

  const handleScrollToSettings = () => {
    document.getElementById('section-insight-settings')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }

  if (!recommendationsEnabled) {
    return (
      <div className="flex items-center gap-3 px-4 py-5">
        <AlertCircle className="h-4 w-4 shrink-0 text-muted-foreground" aria-hidden="true" />
        <span className="text-sm text-muted-foreground">
          {t('settings.billing.recommendations.empty.disabled')}
        </span>
        <button
          type="button"
          onClick={handleScrollToSettings}
          className="ml-auto shrink-0 text-xs text-muted-foreground underline-offset-2 hover:text-foreground hover:underline focus-visible:rounded-sm focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
        >
          {t('settings.billing.recommendations.empty.action.enable')} ↓
        </button>
      </div>
    )
  }

  if (dismissedCount > 0) {
    return (
      <div className="flex items-center gap-3 px-4 py-5">
        <X className="h-4 w-4 shrink-0 text-muted-foreground" aria-hidden="true" />
        <span className="text-sm text-muted-foreground">
          {t('settings.billing.recommendations.empty.dismissed')}
        </span>
        <button
          type="button"
          onClick={handleScrollToSettings}
          className="ml-auto shrink-0 text-xs text-muted-foreground underline-offset-2 hover:text-foreground hover:underline focus-visible:rounded-sm focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
        >
          {t('settings.billing.recommendations.empty.action.adjustSettings')} ↓
        </button>
      </div>
    )
  }

  return (
    <div className="flex items-center gap-3 px-4 py-5">
      <CheckCircle2 className="h-4 w-4 shrink-0 text-success" aria-hidden="true" />
      <span className="text-sm text-muted-foreground">
        {t('settings.billing.recommendations.empty.allGood')}
      </span>
    </div>
  )
}

function InvoiceStatus({ status }: { status: 'paid' | 'open' | 'overdue' }) {
  const { t } = useTranslation()
  const map = {
    paid: {
      icon: CheckCircle2,
      color: 'bg-success-light text-success',
      labelKey: 'settings.billing.history.status.paid',
    },
    open: {
      icon: Receipt,
      color: 'bg-info-light text-info',
      labelKey: 'settings.billing.history.status.open',
    },
    overdue: {
      icon: AlertCircle,
      color: 'bg-error-light text-error',
      labelKey: 'settings.billing.history.status.overdue',
    },
  }
  const cfg = map[status]
  const Icon = cfg.icon
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${cfg.color}`}
    >
      <Icon className="h-3 w-3" aria-hidden="true" />
      {t(cfg.labelKey)}
    </span>
  )
}

/**
 * LicenseAdminHubTab — tenant licensing & module activation (A-3).
 *
 * The provisioning layer: which Cosmi modules are switched on tenant-wide, plan
 * + seat overview. Sits beside the read-only Billing tab (the cost breakdown) —
 * it consumes @/lib/pricing + useTenant rather than duplicating that view.
 * Toggling persists via MSW; real licensing = Luke's track 🔒.
 */
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Info, Layers } from 'lucide-react'
import { Switch } from '@/components/ui/switch'
import { MODULE_PRICES, type ModuleId } from '@/lib/pricing'
import { useTenant } from '@/api/hooks/useBilling'
import { useAdminUsers } from '@/api/hooks/useAdminUsers'
import { useTenantModules, useToggleModule } from '@/api/hooks/useTenantModules'
import type { TenantModule, ModuleGroupId } from '@/api/admin-types'
import { MODULE_GROUP_ORDER, moduleLabelKey } from './moduleMeta'

export default function LicenseAdminHubTab() {
  const { t, i18n } = useTranslation()
  const { data: modules = [], isLoading } = useTenantModules()
  const { data: tenant } = useTenant()
  const { data: users = [] } = useAdminUsers()
  const toggleModule = useToggleModule()

  const seatsUsed = useMemo(
    () => users.filter((u) => u.status === 'active' || u.status === 'invited').length,
    [users],
  )
  const seatsTotal = tenant?.totalSeats ?? seatsUsed
  const activeCount = modules.filter((m) => m.active).length

  const byGroup = useMemo(() => {
    const map = {} as Record<ModuleGroupId, TenantModule[]>
    for (const g of MODULE_GROUP_ORDER) map[g] = []
    for (const m of modules) (map[m.group] ??= []).push(m)
    return map
  }, [modules])

  const renewal = tenant?.billingPeriodEnd
    ? new Intl.DateTimeFormat(i18n.language || 'de', { day: '2-digit', month: 'long', year: 'numeric' }).format(
        new Date(tenant.billingPeriodEnd),
      )
    : '—'

  const planName = tenant?.planType === 'orbit' ? 'Orbit' : 'Cosmi'
  const seatPct = seatsTotal > 0 ? Math.min(100, Math.round((seatsUsed / seatsTotal) * 100)) : 0
  const seatBar = seatPct >= 100 ? 'bg-error' : seatPct >= 80 ? 'bg-warning' : 'bg-primary'

  return (
    <div className="px-6 py-6">
      {/* Header */}
      <div className="mb-5">
        <h2 className="text-base font-semibold text-foreground">{t('admin.license.title')}</h2>
        <p className="mt-0.5 text-sm text-muted-foreground">{t('admin.license.subtitle')}</p>
      </div>

      {/* Plan & seats overview */}
      <div className="mb-6 grid gap-4 sm:grid-cols-3">
        <div className="rounded-xl border border-border bg-card p-4">
          <p className="text-xs uppercase tracking-wider text-muted-foreground">{t('admin.license.plan')}</p>
          <div className="mt-1 flex items-center gap-2">
            <span className="text-lg font-semibold text-foreground">{planName}</span>
            {tenant?.status && (
              <span className="rounded-full bg-success-light px-2 py-0.5 text-[11px] font-medium text-success">
                {t(`admin.license.status.${tenant.status}`)}
              </span>
            )}
          </div>
          <p className="mt-1 text-xs text-muted-foreground">{t('admin.license.planRenewal', { date: renewal })}</p>
        </div>

        <div className="rounded-xl border border-border bg-card p-4">
          <p className="text-xs uppercase tracking-wider text-muted-foreground">{t('admin.license.seatsLabel')}</p>
          <p className="mt-1 text-lg font-semibold text-foreground tabular-nums">
            {seatsUsed} <span className="text-sm font-normal text-muted-foreground">/ {seatsTotal}</span>
          </p>
          <div className="mt-2 h-1.5 w-full overflow-hidden rounded-full bg-secondary">
            <div className={`h-full rounded-full ${seatBar} transition-all`} style={{ width: `${seatPct}%` }} />
          </div>
        </div>

        <div className="rounded-xl border border-border bg-card p-4">
          <p className="text-xs uppercase tracking-wider text-muted-foreground">{t('admin.license.modulesLabel')}</p>
          <p className="mt-1 text-lg font-semibold text-foreground tabular-nums">
            {activeCount} <span className="text-sm font-normal text-muted-foreground">/ {modules.length}</span>
          </p>
          <p className="mt-1 text-xs text-muted-foreground">{t('admin.license.activeModules')}</p>
        </div>
      </div>

      {/* Demo note */}
      <div className="mb-4 flex items-center gap-2 text-xs text-muted-foreground">
        <Info className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
        {t('admin.license.demoNote')}
      </div>

      {/* Module groups */}
      {isLoading ? (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 9 }).map((_, i) => (
            <div key={i} className="h-[88px] animate-pulse rounded-xl bg-secondary/40" />
          ))}
        </div>
      ) : (
        <div className="space-y-6">
          {MODULE_GROUP_ORDER.map((group) => {
            const items = byGroup[group] ?? []
            if (items.length === 0) return null
            return (
              <section key={group}>
                <h3 className="mb-2.5 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                  {t(`admin.license.group.${group}`)}
                </h3>
                <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                  {items.map((m) => (
                    <ModuleCard
                      key={m.moduleId}
                      module={m}
                      onToggle={(active) => toggleModule.mutate({ moduleId: m.moduleId, active })}
                    />
                  ))}
                </div>
              </section>
            )
          })}
        </div>
      )}
    </div>
  )
}

function ModuleCard({ module: m, onToggle }: { module: TenantModule; onToggle: (active: boolean) => void }) {
  const { t } = useTranslation()
  const price = MODULE_PRICES[m.moduleId as ModuleId] ?? 0
  const label = t(moduleLabelKey(m.moduleId))

  return (
    <div
      className={`flex items-start justify-between gap-3 rounded-xl border p-4 transition-colors ${
        m.active ? 'border-border bg-card' : 'border-dashed border-border bg-secondary/20'
      }`}
    >
      <div className={`min-w-0 ${m.active ? '' : 'opacity-70'}`}>
        <div className="flex items-center gap-2">
          <Layers className="h-4 w-4 shrink-0 text-muted-foreground" aria-hidden="true" />
          <span className="truncate text-sm font-medium text-foreground">{label}</span>
        </div>
        <p className="mt-1.5 text-xs text-muted-foreground">{t('admin.license.pricePerUser', { price })}</p>
        <p className="mt-0.5 text-xs text-muted-foreground">
          {m.active ? t('admin.license.assigned', { count: m.assignedSeats }) : t('admin.license.moduleInactive')}
        </p>
      </div>
      <Switch
        checked={m.active}
        onCheckedChange={onToggle}
        aria-label={t('admin.license.toggleAria', { module: label })}
      />
    </div>
  )
}

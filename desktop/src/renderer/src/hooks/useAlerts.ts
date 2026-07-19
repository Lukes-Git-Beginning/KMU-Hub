/**
 * useAlerts — aggregates cross-module alert signals into a unified list.
 *
 * Sources (each flag-gated dashboard-locally, fail-open):
 *   • vertraege  — contracts with status "expiring"
 *   • finance    — invoices with status "overdue"
 *   • helpdesk   — tickets with slaOverdue === true
 *
 * Module gating mirrors WidgetContainer's isWidgetAllowed pattern:
 *   - dashboard-local fail-open (flags unavailable → include the source)
 *   - DEV: respects window.__cosmi_qa_flags__ for QA overrides
 *   - NEVER touches useFeatureFlags/FeatureGate wiring (app-wide fail-closed)
 */
import { useMemo } from 'react'
import { useVertraegeStore } from '@/stores/vertraege'
import { useTickets } from '@/api/hooks/useHelpdesk'
import { useInvoices } from '@/api/hooks/useFinance'
import { useFeatureFlags } from '@/api/hooks/useFeatureFlags'
import { useCapabilitySet } from '@/hooks/useCapability'
import { moduleViewKey, type ModuleKey } from '@/config/capabilities'

export type AlertSeverity = 'warning' | 'critical' | 'info'

export interface DashboardAlert {
  id: string
  severity: AlertSeverity
  /** i18n key for the alert title */
  titleKey: string
  /** ICU-plural compatible i18n key for action label */
  actionKey: string
  /** Interpolation variables for the title key */
  titleVars?: Record<string, string | number>
  /** Route link */
  path: string
  /** Lucide icon name (string so this file stays icon-import-free) */
  iconName: 'AlertCircle' | 'AlertTriangle' | 'Info' | 'FileWarning' | 'Receipt'
}

// ---------------------------------------------------------------------------
// Dashboard-local flag helper (fail-open, DEV QA override)
// Not a React hook — plain function, no use-prefix to avoid rules-of-hooks false-positive.
// ---------------------------------------------------------------------------

function isModuleAllowed(moduleId: string, flags: Record<string, boolean> | undefined, flagsLoading: boolean, flagsError: unknown): boolean {
  // DEV QA override
  if (import.meta.env.DEV) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const qaOverride = (window as any).__cosmi_qa_flags__
    if (qaOverride && typeof qaOverride === 'object' && `modules.${moduleId}` in qaOverride) {
      return Boolean(qaOverride[`modules.${moduleId}`])
    }
  }
  // Fail-open: flags not yet available → include source
  if (flagsError || flagsLoading || !flags) return true
  return flags[`modules.${moduleId}`] ?? false
}

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------

export function useAlerts(): DashboardAlert[] {
  const { flags, isLoading: flagsLoading, error: flagsError } = useFeatureFlags()

  // RBAC module visibility (R-3 dashboard level 2) — default-deny on top of
  // the fail-open flag layer: no `<module>:module:view` grant, no alert.
  const { has: hasCapability, ready: rbacReady } = useCapabilitySet()

  // Store data
  const contracts = useVertraegeStore((s) => s.contracts)

  // Helpdesk — open tickets from real API (fail-open: undefined when loading)
  const { data: openTicketsData } = useTickets({ status: 'open', page_size: 50 })
  const { data: pendingTicketsData } = useTickets({ status: 'pending', page_size: 50 })

  // Finance — overdue invoices (API hook, may error gracefully)
  const { data: invoicesData } = useInvoices({ status: 'overdue', page_size: 50 })

  return useMemo(() => {
    const alerts: DashboardAlert[] = []

    const rbacVisible = (module: ModuleKey): boolean =>
      rbacReady && hasCapability(moduleViewKey(module))

    // ── Contracts: expiring ──────────────────────────────────────────────
    const vertraegeAllowed =
      rbacVisible('vertraege') && isModuleAllowed('vertraege', flags, flagsLoading, flagsError)
    if (vertraegeAllowed) {
      const expiring = contracts.filter((c) => c.status === 'expiring')
      if (expiring.length > 0) {
        alerts.push({
          id: 'contracts-expiring',
          severity: 'warning',
          titleKey: 'dashboard.alerts.contractExpiring',
          titleVars: { count: expiring.length },
          actionKey: 'dashboard.alerts.reviewContracts',
          path: '/vertraege',
          iconName: 'FileWarning',
        })
      }
    }

    // ── Finance: overdue invoices ─────────────────────────────────────────
    const financeAllowed =
      rbacVisible('finance') && isModuleAllowed('finance', flags, flagsLoading, flagsError)
    if (financeAllowed) {
      // Try API-fetched data first, fall back gracefully
      const overdueCount: number =
        (invoicesData as { invoices?: unknown[] } | undefined)?.invoices?.length ?? 0
      if (overdueCount > 0) {
        alerts.push({
          id: 'invoices-overdue',
          severity: 'critical',
          titleKey: 'dashboard.alerts.invoiceOverdue',
          titleVars: { count: overdueCount },
          actionKey: 'dashboard.alerts.viewInvoices',
          path: '/finanzen',
          iconName: 'Receipt',
        })
      }
    }

    // ── Helpdesk: open/pending tickets (SLA breached = those with due_at in past)
    const helpdeskAllowed =
      rbacVisible('helpdesk') && isModuleAllowed('helpdesk', flags, flagsLoading, flagsError)
    if (helpdeskAllowed) {
      const now = Date.now()
      const allActiveTickets = [
        ...(openTicketsData?.tickets ?? []),
        ...(pendingTicketsData?.tickets ?? []),
      ]
      const slaOverdueCount = allActiveTickets.filter(
        (t) => t.due_at && new Date(t.due_at).getTime() < now,
      ).length
      if (slaOverdueCount > 0) {
        alerts.push({
          id: 'helpdesk-sla',
          severity: 'critical',
          titleKey: 'dashboard.alerts.slaBreached',
          titleVars: { count: slaOverdueCount },
          actionKey: 'dashboard.alerts.viewTickets',
          path: '/helpdesk',
          iconName: 'AlertTriangle',
        })
      }
    }

    return alerts
  }, [contracts, openTicketsData, pendingTicketsData, invoicesData, flags, flagsLoading, flagsError, hasCapability, rbacReady])
}

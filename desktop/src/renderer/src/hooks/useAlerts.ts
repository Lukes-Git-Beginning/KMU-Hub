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
import type { LucideIcon } from 'lucide-react'
import { useVertraegeStore } from '@/stores/vertraege'
import { useHelpdeskStore } from '@/stores/helpdesk'
import { useInvoices } from '@/api/hooks/useFinance'
import { useFeatureFlags } from '@/api/hooks/useFeatureFlags'

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

  // Store data
  const contracts = useVertraegeStore((s) => s.contracts)
  const tickets = useHelpdeskStore((s) => s.tickets)

  // Finance — overdue invoices (API hook, may error gracefully)
  const { data: invoicesData } = useInvoices({ status: 'overdue', page_size: 50 })

  return useMemo(() => {
    const alerts: DashboardAlert[] = []

    // ── Contracts: expiring ──────────────────────────────────────────────
    const vertraegeAllowed = isModuleAllowed('vertraege', flags, flagsLoading, flagsError)
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
    const financeAllowed = isModuleAllowed('finance', flags, flagsLoading, flagsError)
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

    // ── Helpdesk: SLA breached ─────────────────────────────────────────
    const helpdeskAllowed = isModuleAllowed('helpdesk', flags, flagsLoading, flagsError)
    if (helpdeskAllowed) {
      const slaOverdue = tickets.filter(
        (t) => t.slaOverdue && t.status !== 'resolved' && t.status !== 'closed',
      )
      if (slaOverdue.length > 0) {
        alerts.push({
          id: 'helpdesk-sla',
          severity: 'critical',
          titleKey: 'dashboard.alerts.slaBreached',
          titleVars: { count: slaOverdue.length },
          actionKey: 'dashboard.alerts.viewTickets',
          path: '/helpdesk',
          iconName: 'AlertTriangle',
        })
      }
    }

    return alerts
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [contracts, tickets, invoicesData, flags, flagsLoading, flagsError])
}

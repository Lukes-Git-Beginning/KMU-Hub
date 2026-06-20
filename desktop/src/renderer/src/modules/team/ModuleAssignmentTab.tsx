/**
 * ModuleAssignmentTab.tsx — Module-assignment matrix for admins and IT-support.
 *
 * Shows a sticky-header / sticky-first-column table of users × modules.
 * Cells toggle grant state. Warning cells (zugeteilt but inactive) highlighted.
 * Supports pre-filtering via initialFilter prop (driven by navigation intent from Billing).
 */
import { useState, useMemo, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useQuery } from '@tanstack/react-query'
import { Search, AlertTriangle, Package } from 'lucide-react'
import { toast } from 'sonner'
import {
  useModuleAssignments,
  useGrantModule,
  useRevokeModule,
  useBulkRevokeInactive,
  useRestoreGrants,
} from '@/api/hooks/useModuleAssignments'
import { useInsightSettings } from '@/api/hooks/useBilling'
import { authenticatedRequest } from '@/api/utils/authenticatedFetch'
import { DEFAULT_INSIGHT_SETTINGS, MODULE_PRICES, type ModuleId } from '@/lib/pricing'
import type { UserModuleGrant } from '@/lib/pricing'
import { ConfirmDialog } from '@/components/shared'
import { formatEUR } from '@/lib/pricing'

// Team members for the assignment matrix, sourced from the real users endpoint.
interface TeamUser {
  userId: string
  userName: string
}

interface RawTeamUser {
  id: string
  email: string
  first_name: string
  last_name: string
}

function useTeamUsers() {
  return useQuery<TeamUser[]>({
    queryKey: ['team', 'users'],
    queryFn: async () => {
      const data = await authenticatedRequest<{ users: RawTeamUser[] }>({
        method: 'GET',
        path: '/api/v1/users',
      })
      return (data.users ?? []).map((u) => ({
        userId: u.id,
        userName: `${u.first_name} ${u.last_name}`.trim() || u.email,
      }))
    },
  })
}

// ───────────────────────── Module display config ─────────────────────────

export const MODULE_DISPLAY_NAMES: Record<string, string> = {
  crm: 'CRM',
  tasks: 'Aufgaben',
  finance: 'Buchhaltung',
  calendar: 'Kalender',
  documents: 'Dokumente',
  chat: 'Chat',
  meetings: 'Meetings',
  mail: 'Mail',
  dialer: 'Dialer',
  team: 'Team',
  zeiterfassung: 'Zeiterfassung',
  schichten: 'Schichten',
  projects: 'Projekte',
  inventar: 'Inventar',
  einkauf: 'Einkauf',
  helpdesk: 'Helpdesk',
  fuhrpark: 'Fuhrpark',
  vertraege: 'Verträge',
  produktion: 'Produktion',
  berichte: 'Berichte',
  formulare: 'Formulare',
  wiki: 'Wiki',
  rapporte: 'Rapporte',
  vermietung: 'Vermietung',
}

// Short labels for table header
const MODULE_SHORT_NAMES: Record<string, string> = {
  crm: 'CRM',
  tasks: 'Aufg.',
  finance: 'Buch.',
  calendar: 'Kal.',
  documents: 'Dok.',
  chat: 'Chat',
  meetings: 'Meet.',
  mail: 'Mail',
  dialer: 'Dial.',
  team: 'Team',
  zeiterfassung: 'Zeit',
  schichten: 'Schicht.',
  projects: 'Proj.',
  inventar: 'Inv.',
  einkauf: 'Eink.',
  helpdesk: 'Help.',
  fuhrpark: 'Fuhr.',
  vertraege: 'Vertr.',
  produktion: 'Prod.',
  berichte: 'Ber.',
  formulare: 'Form.',
  wiki: 'Wiki',
  rapporte: 'Rapp.',
  vermietung: 'Verm.',
}

// ───────────────────────── Types ─────────────────────────

interface ConfirmRevokeState {
  userId: string
  userName: string
  moduleId: ModuleId
  moduleName: string
  grant: UserModuleGrant
}

interface BulkConfirmItem {
  userId: string
  userName: string
  moduleId: ModuleId
  grant: UserModuleGrant
}

export interface ModuleAssignmentInitialFilter {
  moduleId?: string
  userIds?: string[]
}

interface Props {
  initialFilter?: ModuleAssignmentInitialFilter
  onSelectMember?: (userId: string, userName: string) => void
}

// ───────────────────────── Helpers ─────────────────────────

function getInitials(name: string): string {
  return name
    .split(' ')
    .map((n) => n[0])
    .join('')
    .slice(0, 2)
    .toUpperCase()
}

function isInactive(lastActiveAt: string | null, minInactivityDays: number): boolean {
  if (!lastActiveAt) return true
  const diffDays = (Date.now() - new Date(lastActiveAt).getTime()) / 86_400_000
  return diffDays > minInactivityDays
}

function isRecentlyActive(lastActiveAt: string | null): boolean {
  if (!lastActiveAt) return false
  const diffDays = (Date.now() - new Date(lastActiveAt).getTime()) / 86_400_000
  return diffDays < 7
}

// ───────────────────────── Component ─────────────────────────

export function ModuleAssignmentTab({ initialFilter, onSelectMember }: Props) {
  const { t } = useTranslation()
  const { data: allGrants = [] } = useModuleAssignments()
  const { data: allUsers = [] } = useTeamUsers()
  const { data: insightSettings } = useInsightSettings()
  const minInactivityDays = insightSettings?.minInactivityDays ?? DEFAULT_INSIGHT_SETTINGS.minInactivityDays

  const grantModule = useGrantModule()
  const revokeModule = useRevokeModule()
  const bulkRevoke = useBulkRevokeInactive()
  const restoreGrants = useRestoreGrants()

  // ── Filter state ──
  const [searchQuery, setSearchQuery] = useState('')
  const [moduleFilter, setModuleFilter] = useState<string>('all')
  const [warningFilter, setWarningFilter] = useState(false)

  // ── Dialog state ──
  const [confirmRevoke, setConfirmRevoke] = useState<ConfirmRevokeState | null>(null)
  const [bulkConfirm, setBulkConfirm] = useState<BulkConfirmItem[] | null>(null)

  // ── Apply initialFilter on mount ──
  useEffect(() => {
    if (initialFilter?.moduleId) {
      setModuleFilter(initialFilter.moduleId)
    }
  }, [initialFilter?.moduleId])

  // ── Derive module list from grants (only modules with ≥1 grant) ──
  const activeModuleIds = useMemo<ModuleId[]>(() => {
    const seen = new Set<ModuleId>()
    for (const g of allGrants) seen.add(g.moduleId)
    // Sort by display name
    return [...seen].sort((a, b) =>
      (MODULE_DISPLAY_NAMES[a] ?? a).localeCompare(MODULE_DISPLAY_NAMES[b] ?? b),
    )
  }, [allGrants])

  const displayedModules = useMemo<ModuleId[]>(() => {
    if (moduleFilter === 'all') return activeModuleIds
    return activeModuleIds.filter((m) => m === moduleFilter)
  }, [activeModuleIds, moduleFilter])

  // ── Build grant lookup: userId:moduleId → grant ──
  const grantMap = useMemo(() => {
    const map = new Map<string, UserModuleGrant>()
    for (const g of allGrants) {
      map.set(`${g.userId}:${g.moduleId}`, g)
    }
    return map
  }, [allGrants])

  // ── User list (from the users endpoint, filtered by initialFilter.userIds if set) ──
  const baseUsers = useMemo(() => {
    if (initialFilter?.userIds && initialFilter.userIds.length > 0) {
      return allUsers.filter((u) => initialFilter.userIds!.includes(u.userId))
    }
    return allUsers
  }, [allUsers, initialFilter?.userIds])

  const filteredUsers = useMemo(() => {
    return baseUsers.filter((u) => {
      // Text search
      if (searchQuery && !u.userName.toLowerCase().includes(searchQuery.toLowerCase())) {
        return false
      }
      // Warning filter: user must have ≥1 inactive grant in visible modules
      if (warningFilter) {
        const hasWarning = displayedModules.some((m) => {
          const grant = grantMap.get(`${u.userId}:${m}`)
          return grant && isInactive(grant.lastActiveAt, minInactivityDays)
        })
        if (!hasWarning) return false
      }
      return true
    })
  }, [baseUsers, searchQuery, warningFilter, displayedModules, grantMap, minInactivityDays])

  // ── Bulk-revoke candidates: inactive grants in current view ──
  const bulkCandidates = useMemo<BulkConfirmItem[]>(() => {
    const candidates: BulkConfirmItem[] = []
    for (const u of filteredUsers) {
      for (const m of displayedModules) {
        const grant = grantMap.get(`${u.userId}:${m}`)
        if (grant && isInactive(grant.lastActiveAt, minInactivityDays)) {
          candidates.push({ userId: u.userId, userName: u.userName, moduleId: m, grant })
        }
      }
    }
    return candidates
  }, [filteredUsers, displayedModules, grantMap, minInactivityDays])

  const bulkSavings = useMemo(() => {
    return bulkCandidates.reduce((sum, c) => sum + (MODULE_PRICES[c.moduleId] ?? 0), 0)
  }, [bulkCandidates])

  const showBulkBar = (warningFilter || moduleFilter !== 'all') && bulkCandidates.length > 0

  // ── Cell click handler ──
  const handleCellClick = useCallback((u: { userId: string; userName: string }, moduleId: ModuleId) => {
    const key = `${u.userId}:${moduleId}`
    const existing = grantMap.get(key)
    const moduleName = MODULE_DISPLAY_NAMES[moduleId] ?? moduleId

    if (existing) {
      // Revoke path
      if (isRecentlyActive(existing.lastActiveAt)) {
        setConfirmRevoke({ userId: u.userId, userName: u.userName, moduleId, moduleName, grant: existing })
      } else {
        // Direct revoke + undo toast
        const snapGrant = existing
        revokeModule.mutate({ userId: u.userId, moduleId })
        const toastId = toast.success(
          t('team.moduleAssignment.toast.revoked', { user: u.userName, module: moduleName }),
          {
            duration: 10_000,
            action: {
              label: t('common.undo', { defaultValue: 'Rückgängig' }),
              onClick: () => {
                restoreGrants.mutate([snapGrant])
                toast.dismiss(toastId)
              },
            },
          },
        )
      }
    } else {
      // Grant path
      grantModule.mutate({ userId: u.userId, userName: u.userName, moduleId })
      toast.success(t('team.moduleAssignment.toast.granted', { user: u.userName, module: moduleName }))
    }
  }, [grantMap, grantModule, revokeModule, restoreGrants, t])

  // ── Confirm revoke handler ──
  const handleConfirmRevoke = useCallback(() => {
    if (!confirmRevoke) return
    const { userId, moduleId, grant, userName, moduleName } = confirmRevoke
    const snapGrant = grant
    revokeModule.mutate({ userId, moduleId })
    setConfirmRevoke(null)
    const toastId = toast.success(
      t('team.moduleAssignment.toast.revoked', { user: userName, module: moduleName }),
      {
        duration: 10_000,
        action: {
          label: t('common.undo', { defaultValue: 'Rückgängig' }),
          onClick: () => {
            restoreGrants.mutate([snapGrant])
            toast.dismiss(toastId)
          },
        },
      },
    )
  }, [confirmRevoke, revokeModule, restoreGrants, t])

  // ── Bulk revoke handler ──
  const handleBulkRevoke = useCallback(() => {
    if (!bulkConfirm) return
    const snapGrants = bulkConfirm.map((c) => c.grant)
    const pairs = bulkConfirm.map((c) => ({ userId: c.userId, moduleId: c.moduleId }))
    bulkRevoke.mutate({ pairs })
    setBulkConfirm(null)
    const toastId = toast.success(
      t('team.moduleAssignment.bulk.toast', {
        count: snapGrants.length,
        savings: formatEUR(bulkSavings),
      }),
      {
        duration: 10_000,
        action: {
          label: t('common.undo', { defaultValue: 'Rückgängig' }),
          onClick: () => {
            restoreGrants.mutate(snapGrants)
            toast.dismiss(toastId)
          },
        },
      },
    )
  }, [bulkConfirm, bulkRevoke, restoreGrants, bulkSavings, t])

  // ── Cell render helper ──
  function renderCell(u: { userId: string; userName: string }, moduleId: ModuleId) {
    const grant = grantMap.get(`${u.userId}:${moduleId}`)
    const moduleName = MODULE_DISPLAY_NAMES[moduleId] ?? moduleId
    const inactive = grant && isInactive(grant.lastActiveAt, minInactivityDays)

    let stateLabel: string
    let cellContent: React.ReactNode
    let cellClass: string

    if (!grant) {
      stateLabel = t('team.moduleAssignment.stateInactive')
      cellContent = (
        <span className="h-4 w-4 rounded-full border-2 border-border/60 block" aria-hidden="true" />
      )
      cellClass = 'hover:bg-secondary/60'
    } else if (inactive) {
      stateLabel = t('team.moduleAssignment.stateUnused')
      cellContent = (
        <AlertTriangle
          className="h-4 w-4 text-warning"
          aria-hidden="true"
        />
      )
      cellClass = 'bg-warning/5 hover:bg-warning/10'
    } else {
      stateLabel = t('team.moduleAssignment.stateActive')
      cellContent = (
        <span
          className="h-4 w-4 rounded-full bg-primary block"
          aria-hidden="true"
        />
      )
      cellClass = 'hover:bg-secondary/60'
    }

    return (
      <td
        key={moduleId}
        className="border-b border-border/40 px-0"
        role="gridcell"
      >
        <button
          type="button"
          onClick={() => handleCellClick(u, moduleId)}
          aria-label={t('team.moduleAssignment.cellAria', {
            module: moduleName,
            user: u.userName,
            state: stateLabel,
          })}
          aria-pressed={!!grant}
          className={`flex h-9 w-full items-center justify-center transition-colors duration-150 ${cellClass} focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-focus-ring`}
        >
          {cellContent}
        </button>
      </td>
    )
  }

  const totalModules = displayedModules.length

  return (
    <div className="flex flex-col min-h-0">
      {/* ── Filter bar ── */}
      <div className="flex flex-wrap items-center gap-3 mb-4">
        {/* Search */}
        <div className="relative flex-1 min-w-[200px] max-w-xs">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" aria-hidden="true" />
          <input
            type="search"
            name="module-assignment-search"
            autoComplete="off"
            placeholder={t('team.moduleAssignment.searchPlaceholder')}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
          />
        </div>

        {/* Module filter */}
        <select
          value={moduleFilter}
          onChange={(e) => setModuleFilter(e.target.value)}
          className="rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring min-w-[160px]"
          aria-label={t('team.moduleAssignment.moduleFilterAll')}
        >
          <option value="all">{t('team.moduleAssignment.moduleFilterAll')}</option>
          {activeModuleIds.map((m) => (
            <option key={m} value={m}>
              {MODULE_DISPLAY_NAMES[m] ?? m}
            </option>
          ))}
        </select>

        {/* Warning toggle */}
        <button
          type="button"
          onClick={() => setWarningFilter((v) => !v)}
          aria-pressed={warningFilter}
          aria-label={t('team.moduleAssignment.warningFilterAria')}
          className={`flex items-center gap-2 rounded-lg border px-3 py-2 text-sm transition-colors duration-150 ${
            warningFilter
              ? 'border-warning/40 bg-warning/10 text-warning-foreground'
              : 'border-border bg-card text-foreground hover:bg-secondary'
          }`}
        >
          <AlertTriangle className="h-3.5 w-3.5" aria-hidden="true" />
          {t('team.moduleAssignment.warningFilter')}
        </button>
      </div>

      {/* ── Matrix table ── */}
      {filteredUsers.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <Package className="h-10 w-10 text-muted-foreground/40 mb-3" aria-hidden="true" />
          <p className="text-sm font-medium text-foreground">{t('team.moduleAssignment.empty.title')}</p>
          <p className="text-xs text-muted-foreground mt-1">{t('team.moduleAssignment.empty.description')}</p>
        </div>
      ) : (
        <div
          className="overflow-auto rounded-lg border border-border"
          style={{ maxHeight: 'calc(100vh - 320px)' }}
        >
          <table
            role="grid"
            aria-rowcount={filteredUsers.length + 1}
            aria-colcount={displayedModules.length + 2}
            className="w-full border-collapse text-sm"
            style={{ tableLayout: 'auto' }}
          >
            <thead className="sticky top-0 z-20 bg-card">
              <tr role="row">
                {/* Sticky first header cell */}
                <th
                  scope="col"
                  role="columnheader"
                  className="sticky left-0 z-30 bg-card border-b border-r border-border px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wider text-muted-foreground min-w-[160px]"
                >
                  {t('team.page.training.colEmployee')}
                </th>
                {/* Module columns */}
                {displayedModules.map((m) => (
                  <th
                    key={m}
                    scope="col"
                    role="columnheader"
                    title={MODULE_DISPLAY_NAMES[m] ?? m}
                    className="border-b border-border/60 px-2 py-2.5 text-center text-[10px] font-semibold uppercase tracking-wider text-muted-foreground min-w-[52px]"
                  >
                    {MODULE_SHORT_NAMES[m] ?? MODULE_DISPLAY_NAMES[m] ?? m}
                  </th>
                ))}
                {/* Active count column */}
                <th
                  scope="col"
                  role="columnheader"
                  className="sticky right-0 z-30 bg-card border-b border-l border-border px-3 py-2.5 text-center text-[10px] font-semibold uppercase tracking-wider text-muted-foreground min-w-[56px]"
                >
                  {t('team.moduleAssignment.activeColumn')}
                </th>
              </tr>
            </thead>
            <tbody>
              {filteredUsers.map((u, rowIdx) => {
                const activeCount = displayedModules.filter((m) => {
                  const g = grantMap.get(`${u.userId}:${m}`)
                  return g && !isInactive(g.lastActiveAt, minInactivityDays)
                }).length
                const grantedCount = displayedModules.filter((m) =>
                  grantMap.has(`${u.userId}:${m}`),
                ).length

                return (
                  <tr
                    key={u.userId}
                    role="row"
                    aria-rowindex={rowIdx + 2}
                    className="group"
                  >
                    {/* Sticky name cell */}
                    <td
                      role="rowheader"
                      className="sticky left-0 z-10 border-b border-r border-border/60 bg-card px-4 py-0"
                    >
                      <button
                        type="button"
                        onClick={() => onSelectMember?.(u.userId, u.userName)}
                        className="flex h-9 w-full items-center gap-2.5 text-left hover:text-primary transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-focus-ring"
                      >
                        <span
                          className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary-light text-[10px] font-medium text-primary select-none"
                          aria-hidden="true"
                        >
                          {getInitials(u.userName)}
                        </span>
                        <span className="truncate text-xs font-medium text-foreground group-hover:text-primary transition-colors">
                          {u.userName}
                        </span>
                      </button>
                    </td>

                    {/* Module cells */}
                    {displayedModules.map((m) => renderCell(u, m))}

                    {/* Active count cell */}
                    <td
                      role="gridcell"
                      className="sticky right-0 z-10 border-b border-l border-border/60 bg-card px-3 py-0"
                    >
                      <div className="flex h-9 items-center justify-center">
                        <span
                          className={`rounded-full px-2 py-0.5 text-[11px] tabular-nums font-medium ${
                            activeCount < grantedCount
                              ? 'bg-warning/10 text-warning-foreground'
                              : 'bg-secondary text-muted-foreground'
                          }`}
                        >
                          {activeCount}/{totalModules}
                        </span>
                      </div>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* ── Legend ── */}
      <div className="flex items-center gap-5 mt-3 text-[11px] text-muted-foreground">
        <span className="flex items-center gap-1.5">
          <span className="h-3 w-3 rounded-full bg-primary shrink-0" />
          {t('team.moduleAssignment.legend.active')}
        </span>
        <span className="flex items-center gap-1.5">
          <span className="h-3 w-3 rounded-full border-2 border-border/60 shrink-0" />
          {t('team.moduleAssignment.legend.inactive')}
        </span>
        <span className="flex items-center gap-1.5">
          <AlertTriangle className="h-3 w-3 text-warning shrink-0" />
          {t('team.moduleAssignment.legend.unused')}
        </span>
      </div>

      {/* ── Bulk-action bar (sticky bottom, animates in/out) ── */}
      <div
        className="mt-4 transition-transform duration-200 ease-out"
        style={{ transform: showBulkBar ? 'translateY(0)' : 'translateY(8px)', opacity: showBulkBar ? 1 : 0, pointerEvents: showBulkBar ? 'auto' : 'none' }}
        aria-hidden={!showBulkBar}
      >
        {showBulkBar && (
          <div className="flex items-center justify-between rounded-xl border border-warning/30 bg-warning/5 px-4 py-3">
            <div className="text-sm text-foreground">
              <span className="font-medium">{bulkCandidates.length}</span>{' '}
              {t('team.moduleAssignment.warningFilter')} &middot; {formatEUR(bulkSavings)}/Monat Ersparnis
            </div>
            <button
              type="button"
              onClick={() => setBulkConfirm(bulkCandidates)}
              className="flex items-center gap-2 rounded-lg bg-warning px-3 py-1.5 text-xs font-medium text-warning-foreground hover:bg-warning/90 transition-colors"
            >
              <AlertTriangle className="h-3.5 w-3.5" />
              {t('team.moduleAssignment.bulk.label', { count: bulkCandidates.length })}
            </button>
          </div>
        )}
      </div>

      {/* ── Confirm single-revoke dialog ── */}
      <ConfirmDialog
        open={!!confirmRevoke}
        onOpenChange={(open) => { if (!open) setConfirmRevoke(null) }}
        title={t('team.moduleAssignment.confirmRevoke.title')}
        description={t('team.moduleAssignment.confirmRevoke.body', {
          user: confirmRevoke?.userName ?? '',
          module: confirmRevoke?.moduleName ?? '',
        })}
        confirmLabel={t('team.moduleAssignment.confirmRevoke.confirm')}
        variant="destructive"
        onConfirm={handleConfirmRevoke}
      />

      {/* ── Bulk confirm dialog ── */}
      {bulkConfirm && (
        <BulkConfirmDialog
          items={bulkConfirm}
          savings={bulkSavings}
          onConfirm={handleBulkRevoke}
          onCancel={() => setBulkConfirm(null)}
        />
      )}
    </div>
  )
}

// ───────────────────────── Bulk Confirm Dialog ─────────────────────────

function BulkConfirmDialog({
  items,
  savings,
  onConfirm,
  onCancel,
}: {
  items: BulkConfirmItem[]
  savings: number
  onConfirm: () => void
  onCancel: () => void
}) {
  const { t } = useTranslation()

  // Group by module for a readable summary
  const byModule = useMemo(() => {
    const map = new Map<string, string[]>()
    for (const item of items) {
      const list = map.get(item.moduleId) ?? []
      list.push(item.userName)
      map.set(item.moduleId, list)
    }
    return [...map.entries()]
  }, [items])

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="bulk-confirm-title"
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      onClick={(e) => { if (e.target === e.currentTarget) onCancel() }}
    >
      <div className="w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-xl">
        <h2 id="bulk-confirm-title" className="text-base font-semibold text-foreground mb-1">
          {t('team.moduleAssignment.bulk.confirmTitle')}
        </h2>
        <p className="text-sm text-muted-foreground mb-4">
          {t('team.moduleAssignment.bulk.confirmBody', {
            count: items.length,
            savings: formatEUR(savings),
          })}
        </p>

        <div className="max-h-48 overflow-y-auto space-y-2 mb-5 rounded-lg border border-border bg-secondary/30 p-3">
          {byModule.map(([moduleId, users]) => (
            <div key={moduleId}>
              <p className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground mb-0.5">
                {MODULE_DISPLAY_NAMES[moduleId] ?? moduleId}
              </p>
              <p className="text-xs text-foreground">{users.join(', ')}</p>
            </div>
          ))}
        </div>

        <div className="flex gap-3 justify-end">
          <button
            type="button"
            onClick={onCancel}
            className="rounded-lg border border-border px-4 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
          >
            {t('common.cancel', { defaultValue: 'Abbrechen' })}
          </button>
          <button
            type="button"
            onClick={onConfirm}
            className="rounded-lg bg-error px-4 py-2 text-sm font-medium text-error-foreground hover:bg-error/90 transition-colors"
          >
            {t('team.moduleAssignment.bulk.confirm')}
          </button>
        </div>
      </div>
    </div>
  )
}

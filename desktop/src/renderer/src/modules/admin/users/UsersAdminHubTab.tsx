/**
 * UsersAdminHubTab — tenant account management (A-1).
 *
 * The access layer: who can sign in, their role, account status and last login —
 * distinct from the HR personnel records in the `team` module (same people, two
 * lenses). List → search → status filter → sort; whole row opens the detail
 * modal; "Benutzer einladen" creates a pending account. All stateful via MSW.
 */
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Search, UserPlus, Users } from 'lucide-react'
import { SortMenu, EmptyState, type SortDirection, type SortFieldOption } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { useAdminUsers } from '@/api/hooks/useAdminUsers'
import { useRoles } from '@/api/hooks/useRbacRoles'
import { useTenant } from '@/api/hooks/useBilling'
import type { AdminUser, AdminUserStatus } from '@/api/admin-types'
import type { Role } from '@/api/rbac-types'
import { roleDisplayName } from '@/lib/rbac-format'
import { useHasCapability } from '@/hooks/useCapability'
import { InviteUserDialog } from './InviteUserDialog'
import { UserDetailModal } from './UserDetailModal'
import { ROLE_ORDER, STATUS_META, initials, formatRelative } from './presentation'

type StatusFilter = 'all' | AdminUserStatus
type SortField = 'name' | 'role' | 'status' | 'lastLogin'

const STATUS_FILTERS: StatusFilter[] = ['all', 'active', 'invited', 'deactivated']
const STATUS_SORT_RANK: Record<AdminUserStatus, number> = { active: 0, invited: 1, deactivated: 2 }

export default function UsersAdminHubTab() {
  const { t, i18n } = useTranslation()
  const { data: users = [], isLoading } = useAdminUsers()
  const { data: allRoles = [] } = useRoles()
  const { data: tenant } = useTenant()
  const canInvite = useHasCapability('admin:user:invite')

  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all')
  const [sortField, setSortField] = useState<SortField>('name')
  const [sortDir, setSortDir] = useState<SortDirection>('asc')
  const [inviteOpen, setInviteOpen] = useState(false)
  const [selectedId, setSelectedId] = useState<string | null>(null)

  // Keep the detail modal bound to the live query data so it reflects mutations.
  const selectedUser = useMemo(
    () => users.find((u) => u.id === selectedId) ?? null,
    [users, selectedId],
  )

  // Seat usage: active + pending invites consume a seat; deactivated do not.
  const seatsUsed = useMemo(
    () => users.filter((u) => u.status === 'active' || u.status === 'invited').length,
    [users],
  )
  const seatsTotal = tenant?.totalSeats ?? seatsUsed

  const counts = useMemo(() => {
    const c = { all: users.length, active: 0, invited: 0, deactivated: 0 }
    for (const u of users) c[u.status] += 1
    return c
  }, [users])

  const sortOptions: SortFieldOption[] = [
    { value: 'name', label: t('admin.users.col.user') },
    { value: 'role', label: t('admin.users.col.role') },
    { value: 'status', label: t('admin.users.col.status') },
    { value: 'lastLogin', label: t('admin.users.col.lastLogin') },
  ]

  const visible = useMemo(() => {
    const q = search.trim().toLowerCase()
    const filtered = users.filter((u) => {
      if (statusFilter !== 'all' && u.status !== statusFilter) return false
      if (!q) return true
      return (
        `${u.firstName} ${u.lastName}`.toLowerCase().includes(q) ||
        u.email.toLowerCase().includes(q)
      )
    })

    // Primary role rank: presets by ROLE_ORDER, custom roles after them.
    const roleRank = (u: AdminUser): number => {
      const primary = u.roles[0] ?? ''
      const idx = ROLE_ORDER.indexOf(primary as (typeof ROLE_ORDER)[number])
      return idx === -1 ? ROLE_ORDER.length : idx
    }

    const dir = sortDir === 'asc' ? 1 : -1
    const cmp = (a: AdminUser, b: AdminUser): number => {
      switch (sortField) {
        case 'role':
          return (roleRank(a) - roleRank(b)) * dir
        case 'status':
          return (STATUS_SORT_RANK[a.status] - STATUS_SORT_RANK[b.status]) * dir
        case 'lastLogin': {
          const av = a.lastLoginAt ? new Date(a.lastLoginAt).getTime() : 0
          const bv = b.lastLoginAt ? new Date(b.lastLoginAt).getTime() : 0
          return (av - bv) * dir
        }
        default:
          return `${a.lastName} ${a.firstName}`.localeCompare(`${b.lastName} ${b.firstName}`, i18n.language) * dir
      }
    }
    return [...filtered].sort(cmp)
  }, [users, search, statusFilter, sortField, sortDir, i18n.language])

  return (
    <div className="px-6 py-6">
      {/* Header */}
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 className="text-base font-semibold text-foreground">{t('admin.users.title')}</h2>
          <p className="mt-0.5 text-sm text-muted-foreground">{t('admin.users.subtitle')}</p>
        </div>
        <div className="flex items-center gap-3">
          <SeatMeter used={seatsUsed} total={seatsTotal} label={t('admin.users.seats', { used: seatsUsed, total: seatsTotal })} />
          {canInvite && (
            <Button onClick={() => setInviteOpen(true)}>
              <UserPlus className="mr-1.5 h-4 w-4" aria-hidden="true" />
              {t('admin.users.inviteCta')}
            </Button>
          )}
        </div>
      </div>

      {/* Toolbar */}
      <div className="mt-5 flex flex-wrap items-center gap-3">
        {/* Status segmented filter */}
        <div className="flex items-center gap-1 rounded-lg border border-border bg-card p-0.5">
          {STATUS_FILTERS.map((s) => (
            <button
              key={s}
              type="button"
              onClick={() => setStatusFilter(s)}
              aria-pressed={statusFilter === s}
              className={`rounded-md px-2.5 py-1 text-xs font-medium transition-colors ${
                statusFilter === s
                  ? 'bg-secondary text-foreground'
                  : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              {t(`admin.users.filter.${s}`)}
              <span className="ml-1.5 tabular-nums text-muted-foreground">{counts[s]}</span>
            </button>
          ))}
        </div>

        {/* Search */}
        <div className="relative min-w-[200px] flex-1 max-w-xs">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
          <input
            type="search"
            name="admin-user-search"
            autoComplete="off"
            placeholder={t('admin.users.searchPlaceholder')}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full rounded-lg border border-border bg-card py-2 pl-9 pr-3 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
          />
        </div>

        <div className="ml-auto">
          <SortMenu
            options={sortOptions}
            field={sortField}
            direction={sortDir}
            onChange={(f, d) => { setSortField(f as SortField); setSortDir(d) }}
          />
        </div>
      </div>

      {/* List */}
      <div className="mt-4 overflow-hidden rounded-xl border border-border">
        {/* Column header */}
        <div className="grid grid-cols-[2.2fr_1.4fr_1fr_1fr] gap-3 border-b border-border bg-secondary/30 px-4 py-2.5 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
          <span>{t('admin.users.col.user')}</span>
          <span>{t('admin.users.col.role')}</span>
          <span>{t('admin.users.col.status')}</span>
          <span>{t('admin.users.col.lastLogin')}</span>
        </div>

        {isLoading ? (
          <div className="divide-y divide-border">
            {Array.from({ length: 8 }).map((_, i) => (
              <div key={i} className="h-[58px] animate-pulse bg-secondary/20" />
            ))}
          </div>
        ) : visible.length === 0 ? (
          <EmptyState
            icon={Users}
            title={t('admin.users.empty.title')}
            description={search || statusFilter !== 'all' ? t('admin.users.empty.filtered') : t('admin.users.empty.description')}
            action={search || statusFilter !== 'all' || !canInvite ? undefined : { label: t('admin.users.inviteCta'), onClick: () => setInviteOpen(true) }}
          />
        ) : (
          <ul className="divide-y divide-border">
            {visible.map((u) => {
              const status = STATUS_META[u.status]
              const StatusIcon = status.icon
              return (
                <li key={u.id}>
                  <button
                    type="button"
                    onClick={() => setSelectedId(u.id)}
                    className="grid w-full grid-cols-[2.2fr_1.4fr_1fr_1fr] items-center gap-3 px-4 py-2.5 text-left transition-colors hover:bg-secondary/40 focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-focus-ring"
                  >
                    {/* User */}
                    <div className="flex min-w-0 items-center gap-3">
                      <span
                        className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-primary-light text-xs font-semibold text-primary select-none"
                        aria-hidden="true"
                      >
                        {initials(u.firstName, u.lastName)}
                      </span>
                      <div className="min-w-0">
                        <p className="truncate text-sm font-medium text-foreground">
                          {u.firstName} {u.lastName}
                        </p>
                        <p className="truncate text-xs text-muted-foreground">{u.email}</p>
                      </div>
                    </div>

                    {/* Roles (multi-role chips, +n overflow) */}
                    <RoleChipCell roleIds={u.roles} roles={allRoles} />

                    {/* Status */}
                    <div>
                      <span className={`inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium ${status.badgeClass}`}>
                        <StatusIcon className="h-3.5 w-3.5" aria-hidden="true" />
                        {t(status.labelKey)}
                      </span>
                    </div>

                    {/* Last login */}
                    <div className="truncate text-sm text-muted-foreground">
                      {formatRelative(u.lastLoginAt, i18n.language, t('admin.users.status.never'))}
                    </div>
                  </button>
                </li>
              )
            })}
          </ul>
        )}
      </div>

      <InviteUserDialog open={inviteOpen} onOpenChange={setInviteOpen} seatsUsed={seatsUsed} seatsTotal={seatsTotal} />
      <UserDetailModal user={selectedUser} open={selectedId !== null} onClose={() => setSelectedId(null)} />
    </div>
  )
}

/** Role cell: up to two chips with role color, "+n" for the rest. */
function RoleChipCell({ roleIds, roles }: { roleIds: string[]; roles: Role[] }) {
  const { t } = useTranslation()
  const shown = roleIds.slice(0, 2)
  const rest = roleIds.length - shown.length
  return (
    <div className="flex min-w-0 flex-wrap items-center gap-1">
      {shown.map((roleId) => {
        const role = roles.find((r) => r.id === roleId)
        return (
          <span
            key={roleId}
            className="inline-flex max-w-full items-center gap-1.5 rounded-full bg-secondary/60 px-2 py-0.5 text-xs text-foreground"
          >
            <span
              className="h-1.5 w-1.5 shrink-0 rounded-full"
              style={{ background: role?.color ?? 'currentColor' }}
              aria-hidden="true"
            />
            <span className="truncate">{role ? roleDisplayName(t, role) : roleId}</span>
          </span>
        )
      })}
      {rest > 0 && (
        <span className="rounded-full bg-secondary/60 px-1.5 py-0.5 text-xs tabular-nums text-muted-foreground">
          +{rest}
        </span>
      )}
    </div>
  )
}

/** Compact seat-usage meter — counter + slim bar, amber at 80%, red at 100%. */
function SeatMeter({ used, total, label }: { used: number; total: number; label: string }) {
  const pct = total > 0 ? Math.min(100, Math.round((used / total) * 100)) : 0
  const barClass = pct >= 100 ? 'bg-error' : pct >= 80 ? 'bg-warning' : 'bg-primary'
  return (
    <div className="hidden sm:block">
      <p className="text-right text-xs text-muted-foreground tabular-nums">{label}</p>
      <div className="mt-1 h-1.5 w-32 overflow-hidden rounded-full bg-secondary">
        <div className={`h-full rounded-full ${barClass} transition-all`} style={{ width: `${pct}%` }} />
      </div>
    </div>
  )
}

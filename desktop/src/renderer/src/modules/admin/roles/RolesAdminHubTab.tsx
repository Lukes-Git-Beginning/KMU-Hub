/**
 * RolesAdminHubTab — roles & RBAC matrix (A-2).
 *
 * The canonical permission surface (supersedes the legacy fake matrix in the
 * settings IT-admin tab). Five fixed roles from @/config/roles as columns,
 * capabilities grouped by functional area as rows. The admin column is locked
 * (full access). Toggles persist via MSW (mock); real enforcement = gateway 🔒.
 */
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Check, Lock, ShieldCheck, Info } from 'lucide-react'
import { DetailModal } from '@/components/shared'
import type { RoleId } from '@/config/roles'
import { useRolePermissions, useSetPermission } from '@/api/hooks/useRolePermissions'
import { useAdminUsers } from '@/api/hooks/useAdminUsers'
import { ROLE_DOT, ROLE_ORDER, roleLabelKey, STATUS_META, initials } from '../users/presentation'

export default function RolesAdminHubTab() {
  const { t } = useTranslation()
  const { data, isLoading } = useRolePermissions()
  const { data: users = [] } = useAdminUsers()
  const setPermission = useSetPermission()
  const [roleDetail, setRoleDetail] = useState<RoleId | null>(null)

  // Users per role (any status) — drives the column counts + the role detail.
  const usersByRole = useMemo(() => {
    const map = {} as Record<RoleId, typeof users>
    for (const r of ROLE_ORDER) map[r] = []
    for (const u of users) (map[u.role] ??= []).push(u)
    return map
  }, [users])

  const matrix = data?.matrix ?? {}
  const groups = data?.groups ?? []

  const isGranted = (capId: string, role: RoleId) =>
    role === 'admin' ? true : Boolean(matrix[capId]?.[role])

  const toggle = (capId: string, role: RoleId) => {
    if (role === 'admin') return
    setPermission.mutate({ capabilityId: capId, role, granted: !isGranted(capId, role) })
  }

  return (
    <div className="px-6 py-6">
      {/* Header */}
      <div className="mb-5">
        <h2 className="text-base font-semibold text-foreground">{t('admin.roles.title')}</h2>
        <p className="mt-0.5 text-sm text-muted-foreground">{t('admin.roles.subtitle')}</p>
      </div>

      {/* Role summary cards */}
      <div className="mb-5 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
        {ROLE_ORDER.map((role) => (
          <button
            key={role}
            type="button"
            onClick={() => setRoleDetail(role)}
            className="rounded-xl border border-border bg-card p-3 text-left transition-colors hover:bg-secondary/40 focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
          >
            <div className="flex items-center gap-2">
              <span className={`h-2.5 w-2.5 shrink-0 rounded-full ${ROLE_DOT[role]}`} aria-hidden="true" />
              <span className="truncate text-sm font-medium text-foreground">{t(roleLabelKey(role))}</span>
            </div>
            <p className="mt-1.5 text-xs text-muted-foreground">
              {t('admin.roles.userCount', { count: usersByRole[role]?.length ?? 0 })}
            </p>
          </button>
        ))}
      </div>

      {/* Demo note */}
      <div className="mb-3 flex items-center gap-2 text-xs text-muted-foreground">
        <Info className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
        {t('admin.roles.demoNote')}
      </div>

      {/* Permission matrix */}
      <div className="overflow-auto rounded-xl border border-border" style={{ maxHeight: 'calc(100vh - 360px)' }}>
        <table className="w-full border-collapse text-sm">
          <thead className="sticky top-0 z-20">
            <tr>
              <th
                scope="col"
                className="sticky left-0 z-30 border-b border-r border-border bg-card px-4 py-3 text-left text-[11px] font-semibold uppercase tracking-wider text-muted-foreground"
              >
                {t('admin.roles.capabilityCol')}
              </th>
              {ROLE_ORDER.map((role) => (
                <th
                  key={role}
                  scope="col"
                  className="border-b border-border bg-card px-3 py-3 text-center align-bottom"
                >
                  <span className="flex flex-col items-center gap-1">
                    <span className={`h-2 w-2 rounded-full ${ROLE_DOT[role]}`} aria-hidden="true" />
                    <span className="text-xs font-semibold text-foreground">{t(roleLabelKey(role))}</span>
                    <span className="text-[10px] tabular-nums text-muted-foreground">
                      {usersByRole[role]?.length ?? 0}
                    </span>
                  </span>
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {isLoading && (
              <tr>
                <td colSpan={ROLE_ORDER.length + 1} className="px-4 py-10">
                  <div className="space-y-2">
                    {Array.from({ length: 6 }).map((_, i) => (
                      <div key={i} className="h-6 animate-pulse rounded bg-secondary/40" />
                    ))}
                  </div>
                </td>
              </tr>
            )}
            {groups.map((group) => (
              <GroupRows
                key={group.id}
                groupId={group.id}
                capabilities={group.capabilities}
                isGranted={isGranted}
                toggle={toggle}
              />
            ))}
          </tbody>
        </table>
      </div>

      <RoleUsersModal
        role={roleDetail}
        users={roleDetail ? usersByRole[roleDetail] ?? [] : []}
        onClose={() => setRoleDetail(null)}
      />
    </div>
  )
}

function GroupRows({
  groupId,
  capabilities,
  isGranted,
  toggle,
}: {
  groupId: string
  capabilities: string[]
  isGranted: (capId: string, role: RoleId) => boolean
  toggle: (capId: string, role: RoleId) => void
}) {
  const { t } = useTranslation()
  return (
    <>
      <tr>
        <th
          scope="colgroup"
          colSpan={ROLE_ORDER.length + 1}
          className="sticky left-0 border-b border-border bg-secondary/40 px-4 py-1.5 text-left text-[11px] font-semibold uppercase tracking-wider text-muted-foreground"
        >
          {t(`admin.roles.group.${groupId}`)}
        </th>
      </tr>
      {capabilities.map((capId) => (
        <tr key={capId} className="group">
          <th
            scope="row"
            className="sticky left-0 z-10 border-b border-r border-border/60 bg-card px-4 py-0 text-left font-normal"
          >
            <span className="block py-2.5 text-sm text-foreground">{t(`admin.roles.cap.${capId}`)}</span>
          </th>
          {ROLE_ORDER.map((role) => {
            const granted = isGranted(capId, role)
            const locked = role === 'admin'
            return (
              <td key={role} className="border-b border-border/40 px-0 text-center">
                <div className="flex items-center justify-center py-2">
                  <PermissionCheckbox
                    granted={granted}
                    locked={locked}
                    onClick={() => toggle(capId, role)}
                    label={t('admin.roles.cellAria', {
                      capability: t(`admin.roles.cap.${capId}`),
                      role: t(roleLabelKey(role)),
                    })}
                  />
                </div>
              </td>
            )
          })}
        </tr>
      ))}
    </>
  )
}

function PermissionCheckbox({
  granted,
  locked,
  onClick,
  label,
}: {
  granted: boolean
  locked: boolean
  onClick: () => void
  label: string
}) {
  return (
    <button
      type="button"
      onClick={locked ? undefined : onClick}
      disabled={locked}
      role="checkbox"
      aria-checked={granted}
      aria-label={label}
      className={`flex h-5 w-5 items-center justify-center rounded-md border transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring ${
        granted
          ? 'border-primary bg-primary text-primary-foreground'
          : 'border-border hover:border-muted-foreground'
      } ${locked ? 'cursor-not-allowed opacity-70' : 'cursor-pointer'}`}
    >
      {locked ? (
        granted && <Lock className="h-3 w-3" aria-hidden="true" />
      ) : (
        granted && <Check className="h-3.5 w-3.5" aria-hidden="true" />
      )}
    </button>
  )
}

function RoleUsersModal({
  role,
  users,
  onClose,
}: {
  role: RoleId | null
  users: { id: string; firstName: string; lastName: string; email: string; status: keyof typeof STATUS_META }[]
  onClose: () => void
}) {
  const { t } = useTranslation()
  if (!role) return null
  return (
    <DetailModal
      open={role !== null}
      onClose={onClose}
      title={t(roleLabelKey(role))}
      subtitle={t(`config.roles.${role}.description`)}
      badge={
        <span className="inline-flex items-center gap-1.5 rounded-full bg-secondary px-2.5 py-1 text-xs font-medium text-muted-foreground">
          <ShieldCheck className="h-3.5 w-3.5" aria-hidden="true" />
          {t('admin.roles.userCount', { count: users.length })}
        </span>
      }
    >
      <div className="p-5">
        {users.length === 0 ? (
          <p className="py-8 text-center text-sm text-muted-foreground">{t('admin.roles.noUsers')}</p>
        ) : (
          <ul className="divide-y divide-border">
            {users.map((u) => {
              const status = STATUS_META[u.status]
              const StatusIcon = status.icon
              return (
                <li key={u.id} className="flex items-center gap-3 py-2.5">
                  <span
                    className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-primary-light text-xs font-semibold text-primary select-none"
                    aria-hidden="true"
                  >
                    {initials(u.firstName, u.lastName)}
                  </span>
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium text-foreground">{u.firstName} {u.lastName}</p>
                    <p className="truncate text-xs text-muted-foreground">{u.email}</p>
                  </div>
                  <span className={`inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium ${status.badgeClass}`}>
                    <StatusIcon className="h-3.5 w-3.5" aria-hidden="true" />
                    {t(status.labelKey)}
                  </span>
                </li>
              )
            })}
          </ul>
        )}
      </div>
    </DetailModal>
  )
}

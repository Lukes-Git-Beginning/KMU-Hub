/**
 * RolesBuilderTab (R-2) — entry of the role builder in the Verwaltung hub.
 * Replaces the legacy A-2 permission matrix. System presets and tenant custom
 * roles as clickable cards (whole card = open editor), clone as the only
 * create path (never from scratch — Zoho/monday market lesson), compare entry,
 * member modal, guarded delete.
 */
import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useQueries } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Copy, GitCompareArrows, Lock, MoreHorizontal, Pencil, Plus, ShieldCheck, Trash2, Users } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { DetailModal } from '@/components/shared'
import type { Role, RoleGrants } from '@/api/rbac-types'
import { fetchRolePermissions } from '@/api/rbac-client'
import { useDeleteRole, useRoles } from '@/api/hooks/useRbacRoles'
import { useAdminUsers } from '@/api/hooks/useAdminUsers'
import { useHasCapability } from '@/hooks/useCapability'
import { rbacErrorMessage, roleDisplayName } from '@/lib/rbac-format'
import { diffCount, diffGrants } from '@/lib/rbac-diff'
import { CUSTOM_ROLE_LIMIT } from '@/mocks/data/rbac'
import { roleDescriptionKey } from '@/config/roles'
import { STATUS_META, initials } from '../users/presentation'
import CloneRoleDialog from './CloneRoleDialog'
import RoleCompareModal from './RoleCompareModal'
import { RolesAuditTab } from './RolesAuditTab'

type RolesSubTab = 'overview' | 'protocol'

export default function RolesBuilderTab() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { data: roles = [], isLoading } = useRoles()
  const { data: users = [] } = useAdminUsers()
  const deleteRole = useDeleteRole()
  const canCreate = useHasCapability('admin:role:create')
  const canDelete = useHasCapability('admin:role:delete')
  const canViewAudit = useHasCapability('security:audit:read')

  const [activeSubTab, setActiveSubTab] = useState<RolesSubTab>('overview')
  const [cloneBase, setCloneBase] = useState<Role | null>(null)
  const [cloneOpen, setCloneOpen] = useState(false)
  const [membersRole, setMembersRole] = useState<Role | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Role | null>(null)
  const [compareOpen, setCompareOpen] = useState(false)

  const presets = useMemo(() => roles.filter((r) => r.isSystem), [roles])
  const customs = useMemo(() => roles.filter((r) => !r.isSystem), [roles])
  const limitReached = customs.length >= CUSTOM_ROLE_LIMIT

  // Grants of custom roles + their presets → "X deviations" badges.
  const grantIds = useMemo(() => {
    const ids = new Set<string>()
    for (const c of customs) {
      ids.add(c.id)
      if (c.basedOn) ids.add(c.basedOn)
    }
    return [...ids]
  }, [customs])

  const grantResults = useQueries({
    queries: grantIds.map((id) => ({
      queryKey: ['rbac', 'roles', id, 'permissions'] as const,
      queryFn: () => fetchRolePermissions(id),
    })),
  })

  const grantsById = useMemo(() => {
    const map: Record<string, RoleGrants> = {}
    grantIds.forEach((id, i) => {
      const data = grantResults[i]?.data
      if (data) map[id] = data
    })
    return map
  }, [grantIds, grantResults])

  const deviationCount = (role: Role): number | null => {
    if (!role.basedOn) return null
    const own = grantsById[role.id]
    const base = grantsById[role.basedOn]
    if (!own || !base) return null
    return diffCount(diffGrants(own, base))
  }

  const openClone = (base: Role | null) => {
    if (limitReached) {
      toast.error(t('rbac.builder.errors.role_limit_reached'))
      return
    }
    setCloneBase(base)
    setCloneOpen(true)
  }

  const confirmDelete = () => {
    if (!deleteTarget) return
    deleteRole.mutate(deleteTarget.id, {
      onSuccess: () => {
        toast.success(t('rbac.builder.deleteDone', { name: deleteTarget.name }))
        setDeleteTarget(null)
      },
      onError: (err) => toast.error(rbacErrorMessage(t, err)),
    })
  }

  const membersOf = (role: Role) => users.filter((u) => u.roles.includes(role.id))

  return (
    <div className="flex flex-col">
      {/* Sub-Tab-Bar: Übersicht | Protokoll (nur wenn audit:read) */}
      {canViewAudit && (
        <div
          role="tablist"
          aria-label={t('rbac.builder.title')}
          className="flex gap-0 border-b border-border px-6"
        >
          <button
            role="tab"
            aria-selected={activeSubTab === 'overview'}
            onClick={() => setActiveSubTab('overview')}
            className={[
              'px-3 py-2.5 text-sm border-b-2 -mb-px transition-colors',
              activeSubTab === 'overview'
                ? 'border-primary text-primary font-medium'
                : 'border-transparent text-muted-foreground hover:text-foreground',
            ].join(' ')}
          >
            {t('rbac.builder.subTab.overview')}
          </button>
          <button
            role="tab"
            aria-selected={activeSubTab === 'protocol'}
            onClick={() => setActiveSubTab('protocol')}
            className={[
              'px-3 py-2.5 text-sm border-b-2 -mb-px transition-colors',
              activeSubTab === 'protocol'
                ? 'border-primary text-primary font-medium'
                : 'border-transparent text-muted-foreground hover:text-foreground',
            ].join(' ')}
          >
            {t('rbac.builder.subTab.protocol')}
          </button>
        </div>
      )}

      {/* Protokoll-Tab-Inhalt */}
      {activeSubTab === 'protocol' && canViewAudit && <RolesAuditTab />}

      {/* Übersicht-Tab-Inhalt */}
      {(activeSubTab === 'overview' || !canViewAudit) && (
      <div className="px-6 py-6">
      {/* Header + actions */}
      <div className="mb-5 flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className="text-base font-semibold text-foreground">{t('rbac.builder.title')}</h2>
          <p className="mt-0.5 text-sm text-muted-foreground">{t('rbac.builder.subtitle')}</p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => setCompareOpen(true)}>
            <GitCompareArrows className="mr-1.5 h-4 w-4" aria-hidden="true" />
            {t('rbac.builder.compare')}
          </Button>
          {canCreate && (
            <Button size="sm" onClick={() => openClone(null)} disabled={limitReached}>
              <Plus className="mr-1.5 h-4 w-4" aria-hidden="true" />
              {t('rbac.builder.newRole')}
            </Button>
          )}
        </div>
      </div>

      {limitReached && (
        <p className="mb-4 rounded-lg border border-warning/40 bg-warning-light px-3 py-2 text-xs text-warning">
          {t('rbac.builder.limitHint', { limit: CUSTOM_ROLE_LIMIT })}
        </p>
      )}

      {isLoading ? (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="h-28 animate-pulse rounded-xl bg-secondary/40" />
          ))}
        </div>
      ) : (
        <>
          {/* System presets */}
          <SectionHeader
            icon={<Lock className="h-3.5 w-3.5" aria-hidden="true" />}
            label={t('rbac.builder.systemRoles')}
            hint={t('rbac.builder.systemRolesHint')}
          />
          <div className="mb-6 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {presets.map((role) => (
              <RoleCard
                key={role.id}
                role={role}
                deviations={null}
                memberCount={membersOf(role).length}
                onOpen={() => navigate(`/admin/roles/${role.id}`)}
                onClone={canCreate ? () => openClone(role) : undefined}
                onMembers={() => setMembersRole(role)}
              />
            ))}
          </div>

          {/* Custom roles */}
          <SectionHeader
            icon={<ShieldCheck className="h-3.5 w-3.5" aria-hidden="true" />}
            label={t('rbac.builder.customRoles')}
            hint={t('rbac.builder.customRolesHint', { count: customs.length, limit: CUSTOM_ROLE_LIMIT })}
          />
          {customs.length === 0 ? (
            <div className="rounded-xl border border-dashed border-border px-6 py-10 text-center">
              <p className="text-sm font-medium text-foreground">{t('rbac.builder.noCustomRoles')}</p>
              <p className="mx-auto mt-1 max-w-md text-xs text-muted-foreground">
                {t('rbac.builder.noCustomRolesHint')}
              </p>
              {canCreate && (
                <Button size="sm" variant="outline" className="mt-4" onClick={() => openClone(null)}>
                  <Copy className="mr-1.5 h-4 w-4" aria-hidden="true" />
                  {t('rbac.builder.cloneFirst')}
                </Button>
              )}
            </div>
          ) : (
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {customs.map((role) => (
                <RoleCard
                  key={role.id}
                  role={role}
                  deviations={deviationCount(role)}
                  basedOnName={
                    role.basedOn
                      ? roleDisplayName(t, roles.find((r) => r.id === role.basedOn) ?? { id: role.basedOn, name: role.basedOn, isSystem: true })
                      : undefined
                  }
                  memberCount={membersOf(role).length}
                  onOpen={() => navigate(`/admin/roles/${role.id}`)}
                  onClone={canCreate ? () => openClone(role) : undefined}
                  onMembers={() => setMembersRole(role)}
                  onDelete={canDelete ? () => setDeleteTarget(role) : undefined}
                />
              ))}
            </div>
          )}
        </>
      )}

      {/* Clone / create */}
      <CloneRoleDialog
        open={cloneOpen}
        onOpenChange={setCloneOpen}
        roles={roles}
        initialBase={cloneBase}
        onCreated={(role) => {
          setCloneOpen(false)
          navigate(`/admin/roles/${role.id}`)
        }}
      />

      {/* Compare */}
      <RoleCompareModal open={compareOpen} onClose={() => setCompareOpen(false)} roles={roles} />

      {/* Members */}
      <RoleMembersModal
        role={membersRole}
        members={membersRole ? membersOf(membersRole) : []}
        onClose={() => setMembersRole(null)}
      />

      {/* Delete confirm */}
      <AlertDialog open={deleteTarget !== null} onOpenChange={(o) => { if (!o) setDeleteTarget(null) }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('rbac.builder.deleteTitle', { name: deleteTarget?.name ?? '' })}</AlertDialogTitle>
            <AlertDialogDescription>
              {deleteTarget && membersOf(deleteTarget).length > 0
                ? t('rbac.builder.deleteBlockedMembers', { count: membersOf(deleteTarget).length })
                : t('rbac.builder.deleteConfirm')}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={confirmDelete}
              disabled={deleteTarget !== null && membersOf(deleteTarget).length > 0}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {t('rbac.builder.deleteAction')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      </div>
      )}
    </div>
  )
}

function SectionHeader({ icon, label, hint }: { icon: React.ReactNode; label: string; hint?: string }) {
  return (
    <div className="mb-2.5 flex items-baseline justify-between gap-3">
      <h3 className="flex items-center gap-1.5 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
        {icon}
        {label}
      </h3>
      {hint && <span className="text-xs text-muted-foreground">{hint}</span>}
    </div>
  )
}

function RoleCard({
  role,
  deviations,
  basedOnName,
  memberCount,
  onOpen,
  onClone,
  onMembers,
  onDelete,
}: {
  role: Role
  deviations: number | null
  basedOnName?: string
  memberCount: number
  onOpen: () => void
  onClone?: () => void
  onMembers: () => void
  onDelete?: () => void
}) {
  const { t } = useTranslation()
  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onOpen}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onOpen()
        }
      }}
      className="group cursor-pointer rounded-xl border border-border bg-card p-4 text-left transition-colors hover:bg-secondary/40 focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
    >
      <div className="flex items-start gap-2.5">
        <span
          className="mt-1 h-2.5 w-2.5 shrink-0 rounded-full"
          style={{ background: role.color }}
          aria-hidden="true"
        />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5">
            <span className="truncate text-sm font-medium text-foreground">{roleDisplayName(t, role)}</span>
            {role.isSystem && (
              <span className="inline-flex shrink-0 items-center gap-1 rounded-full bg-secondary px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">
                <Lock className="h-2.5 w-2.5" aria-hidden="true" />
                {t('rbac.builder.systemBadge')}
              </span>
            )}
          </div>
          <p className="mt-0.5 line-clamp-2 text-xs text-muted-foreground">
            {role.isSystem ? t(roleDescriptionKey(role.id)) : role.description}
          </p>
        </div>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button
              type="button"
              onClick={(e) => e.stopPropagation()}
              aria-label={t('rbac.builder.cardMenu')}
              className="rounded-md p-1 text-muted-foreground opacity-0 transition-opacity hover:bg-secondary hover:text-foreground focus:opacity-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring group-hover:opacity-100"
            >
              <MoreHorizontal className="h-4 w-4" aria-hidden="true" />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" onClick={(e) => e.stopPropagation()}>
            <DropdownMenuItem onClick={onOpen}>
              <Pencil className="mr-2 h-4 w-4" aria-hidden="true" />
              {role.isSystem ? t('rbac.builder.view') : t('rbac.builder.edit')}
            </DropdownMenuItem>
            {onClone && (
              <DropdownMenuItem onClick={onClone}>
                <Copy className="mr-2 h-4 w-4" aria-hidden="true" />
                {t('rbac.builder.clone')}
              </DropdownMenuItem>
            )}
            <DropdownMenuItem onClick={onMembers}>
              <Users className="mr-2 h-4 w-4" aria-hidden="true" />
              {t('rbac.builder.members')}
            </DropdownMenuItem>
            {onDelete && (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={onDelete} className="text-destructive focus:text-destructive">
                  <Trash2 className="mr-2 h-4 w-4" aria-hidden="true" />
                  {t('rbac.builder.delete')}
                </DropdownMenuItem>
              </>
            )}
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      <div className="mt-3 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
        <span>{t('rbac.builder.memberCount', { count: memberCount })}</span>
        <span aria-hidden="true">·</span>
        <span>{t('rbac.builder.capabilityCount', { count: role.capabilityCount })}</span>
        {basedOnName && (
          <>
            <span aria-hidden="true">·</span>
            <span>{t('rbac.builder.basedOnBadge', { name: basedOnName })}</span>
          </>
        )}
        {deviations !== null && deviations > 0 && (
          <span className="inline-flex items-center rounded-full bg-info-light px-1.5 py-0.5 text-[10px] font-medium text-info">
            {t('rbac.builder.deviationCount', { count: deviations })}
          </span>
        )}
      </div>
    </div>
  )
}

function RoleMembersModal({
  role,
  members,
  onClose,
}: {
  role: Role | null
  members: Array<{ id: string; firstName: string; lastName: string; email: string; status: keyof typeof STATUS_META }>
  onClose: () => void
}) {
  const { t } = useTranslation()
  if (!role) return null
  return (
    <DetailModal
      open={role !== null}
      onClose={onClose}
      title={roleDisplayName(t, role)}
      subtitle={role.isSystem ? t(roleDescriptionKey(role.id)) : role.description}
      badge={
        <span className="inline-flex items-center gap-1.5 rounded-full bg-secondary px-2.5 py-1 text-xs font-medium text-muted-foreground">
          <Users className="h-3.5 w-3.5" aria-hidden="true" />
          {t('rbac.builder.memberCount', { count: members.length })}
        </span>
      }
    >
      {members.length === 0 ? (
        <p className="py-8 text-center text-sm text-muted-foreground">{t('rbac.builder.noMembers')}</p>
      ) : (
        <ul className="divide-y divide-border">
          {members.map((u) => {
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
    </DetailModal>
  )
}

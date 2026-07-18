/**
 * UserRolesSection (R-2) — the shared role-assignment block for one account:
 * role chips (preset i18n / custom names + colors), a multi-assign popover
 * (checkbox per role, mutations fire per toggle) and the "Effektive Rechte"
 * modal. Used in the team member profile (HR home turf, Darien 2026-07-18)
 * and the admin user detail. Guardrails: last-admin removal is rejected
 * server-side (`last_admin`), own roles are read-only (self-lockout).
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Check, ChevronDown, ShieldCheck } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { DetailModal } from '@/components/shared'
import { useRoles, useAssignUserRole, useRemoveUserRole, useUserEffectivePermissions } from '@/api/hooks/useRbacRoles'
import { EffectivePermissionsView } from './EffectivePermissionsView'
import { useAuthStore } from '@/stores/auth'
import { usePermissionsStore } from '@/stores/permissions'
import { rbacErrorMessage, roleDisplayName } from '@/lib/rbac-format'

export function UserRolesSection({
  userId,
  roleIds,
  editable,
  displayName,
}: {
  userId: string
  /** Current role ids of the account (from AdminUser.roles / rolesForUser). */
  roleIds: string[]
  /** Whether the viewer may change assignments (capability-gated by caller). */
  editable: boolean
  /** Person name for the effective-rights modal title. */
  displayName: string
}) {
  const { t } = useTranslation()
  const { data: roles = [] } = useRoles()
  const assign = useAssignUserRole()
  const remove = useRemoveUserRole()
  const authUserId = useAuthStore((s) => s.user?.id)
  const [effectiveOpen, setEffectiveOpen] = useState(false)

  const isSelf = userId === authUserId
  const canEditHere = editable && !isSelf

  const refreshIfSelf = () => {
    if (isSelf) void usePermissionsStore.getState().refresh()
  }

  const toggleRole = (roleId: string, currentlyAssigned: boolean) => {
    if (currentlyAssigned) {
      remove.mutate(
        { userId, roleId },
        {
          onSuccess: () => {
            toast.success(t('rbac.assignment.removed'))
            refreshIfSelf()
          },
          onError: (err) => toast.error(rbacErrorMessage(t, err)),
        },
      )
    } else {
      assign.mutate(
        { userId, roleId },
        {
          onSuccess: () => {
            toast.success(t('rbac.assignment.added'))
            refreshIfSelf()
          },
          onError: (err) => toast.error(rbacErrorMessage(t, err)),
        },
      )
    }
  }

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-center gap-2">
        {roleIds.map((roleId) => {
          const role = roles.find((r) => r.id === roleId)
          return (
            <span
              key={roleId}
              className="inline-flex items-center gap-1.5 rounded-full border border-border bg-card px-2.5 py-1 text-xs font-medium text-foreground"
            >
              <span
                className="h-2 w-2 rounded-full"
                style={{ background: role?.color ?? 'currentColor' }}
                aria-hidden="true"
              />
              {role ? roleDisplayName(t, role) : roleId}
            </span>
          )
        })}

        {canEditHere && (
          <Popover>
            <PopoverTrigger asChild>
              <Button variant="outline" size="sm" className="h-7 text-xs" disabled={assign.isPending || remove.isPending}>
                {t('rbac.assignment.manage')}
                <ChevronDown className="ml-1 h-3.5 w-3.5" aria-hidden="true" />
              </Button>
            </PopoverTrigger>
            <PopoverContent align="start" className="w-72 p-2">
              <ul className="max-h-64 overflow-y-auto">
                {roles.map((role) => {
                  const assigned = roleIds.includes(role.id)
                  return (
                    <li key={role.id}>
                      <button
                        type="button"
                        role="checkbox"
                        aria-checked={assigned}
                        onClick={() => toggleRole(role.id, assigned)}
                        className="flex w-full items-center gap-2.5 rounded-lg px-2 py-1.5 text-left text-sm transition-colors hover:bg-secondary/60 focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
                      >
                        <span
                          className={`flex h-4 w-4 shrink-0 items-center justify-center rounded border ${
                            assigned ? 'border-primary bg-primary text-primary-foreground' : 'border-border'
                          }`}
                          aria-hidden="true"
                        >
                          {assigned && <Check className="h-3 w-3" />}
                        </span>
                        <span className="h-2 w-2 shrink-0 rounded-full" style={{ background: role.color }} aria-hidden="true" />
                        <span className="min-w-0 flex-1 truncate">{roleDisplayName(t, role)}</span>
                      </button>
                    </li>
                  )
                })}
              </ul>
              <p className="mt-2 border-t border-border px-2 pt-2 text-[11px] leading-snug text-muted-foreground">
                {t('rbac.assignment.unionHint')}
              </p>
            </PopoverContent>
          </Popover>
        )}
      </div>

      {isSelf && editable && (
        <p className="text-xs text-muted-foreground">{t('rbac.assignment.selfLocked')}</p>
      )}

      <Button variant="outline" size="sm" className="h-7 text-xs" onClick={() => setEffectiveOpen(true)}>
        <ShieldCheck className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
        {t('rbac.assignment.effectiveRights')}
      </Button>

      <EffectivePermissionsModal
        userId={effectiveOpen ? userId : null}
        displayName={displayName}
        onClose={() => setEffectiveOpen(false)}
      />
    </div>
  )
}

/** Per-user effective-rights modal (admin/HR audit view). */
export function EffectivePermissionsModal({
  userId,
  displayName,
  onClose,
}: {
  userId: string | null
  displayName: string
  onClose: () => void
}) {
  const { t } = useTranslation()
  const { data: permissions, isLoading } = useUserEffectivePermissions(userId)

  return (
    <DetailModal
      open={userId !== null}
      onClose={onClose}
      title={t('rbac.assignment.effectiveTitle', { name: displayName })}
      subtitle={t('rbac.assignment.effectiveSubtitle')}
      maxWidth="max-w-3xl"
    >
      {isLoading || !permissions ? (
        <div className="space-y-2 py-4">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="h-8 animate-pulse rounded bg-secondary/40" />
          ))}
        </div>
      ) : (
        <EffectivePermissionsView roles={permissions.roles} capabilities={permissions.capabilities} />
      )}
    </DetailModal>
  )
}

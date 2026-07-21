/**
 * UserDetailModal — full account view for one user inside the shared Cosmi
 * DetailModal. Change role, activate/deactivate access, resend a pending invite.
 * The signed-in admin cannot deactivate or downgrade their own account.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Mail, Send, UserCheck, UserX, Eye, SlidersHorizontal } from 'lucide-react'
import { toast } from 'sonner'
import { useNavigate } from 'react-router-dom'
import { DetailModal, ConfirmDialog } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { useAuthStore, type User } from '@/stores/auth'
import type { AdminUser } from '@/api/admin-types'
import { useUpdateAdminUser, useResendInvite } from '@/api/hooks/useAdminUsers'
import { useHasCapability } from '@/hooks/useCapability'
import { UserRolesSection } from '@/components/shared/rbac/UserRolesSection'
import { useViewAsStore } from '@/stores/viewAs'
import { rolesForUser } from '@/mocks/data/rbac'
import { STATUS_META, initials, formatRelative } from './presentation'

interface UserDetailModalProps {
  user: AdminUser | null
  open: boolean
  onClose: () => void
}

export function UserDetailModal({ user, open, onClose }: UserDetailModalProps) {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const updateUser = useUpdateAdminUser()
  const resendInvite = useResendInvite()
  const authUserId = useAuthStore((s) => s.user?.id)
  const canAssignRoles = useHasCapability('admin:role:assign')
  const canDeactivate = useHasCapability('admin:user:deactivate')
  const canImpersonate = useHasCapability('admin:impersonate:run')
  const canEditOverrides = useHasCapability('admin:user_override:manage')
  const startViewAs = useViewAsStore((s) => s.startViewAs)
  const [confirmDeactivate, setConfirmDeactivate] = useState(false)

  if (!user) return null

  const isSelf = user.id === authUserId

  // Guardrail: View-as not allowed on admins or self
  const targetRoles = rolesForUser(user.id)
  const targetIsAdmin = targetRoles.includes('admin')
  const canViewAs = canImpersonate && !isSelf && !targetIsAdmin

  const handleViewAs = () => {
    // Build a minimal User object from the AdminUser for the session switch
    const targetUser: User = {
      id: user.id,
      email: user.email,
      firstName: user.firstName,
      lastName: user.lastName,
      roles: targetRoles,
      avatarUrl: null,
    }
    startViewAs(targetUser)
    onClose()
    navigate('/')
  }
  const status = STATUS_META[user.status]
  const StatusIcon = status.icon
  const fullName = `${user.firstName} ${user.lastName}`

  const setStatus = (next: AdminUser['status'], message: string) => {
    updateUser.mutate(
      { id: user.id, status: next },
      { onSuccess: () => toast.success(message) },
    )
  }

  const handleResend = () => {
    resendInvite.mutate(user.id, {
      onSuccess: () => toast.success(t('admin.users.detail.inviteResent', { email: user.email })),
    })
  }

  return (
    <>
      <DetailModal
        open={open}
        onClose={onClose}
        title={fullName}
        subtitle={user.jobTitle || undefined}
        badge={
          <span className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium ${status.badgeClass}`}>
            <StatusIcon className="h-3.5 w-3.5" aria-hidden="true" />
            {t(status.labelKey)}
          </span>
        }
      >
        <div className="space-y-6 p-5">
          {/* Identity card */}
          <div className="flex items-center gap-3.5">
            <span
              className="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-primary-light text-sm font-semibold text-primary select-none"
              aria-hidden="true"
            >
              {initials(user.firstName, user.lastName)}
            </span>
            <div className="min-w-0">
              <p className="truncate text-sm font-semibold text-foreground">{fullName}</p>
              <a
                href={`mailto:${user.email}`}
                className="flex items-center gap-1.5 truncate text-sm text-muted-foreground hover:text-primary transition-colors"
              >
                <Mail className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
                {user.email}
              </a>
            </div>
          </div>

          {/* Roles (multi-role since R-2 — assignment itself is HR's home turf
              in the team module; IT with admin:role:assign can manage here too) */}
          <section className="space-y-2">
            <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              {t('admin.users.detail.roleSection')}
            </h3>
            <UserRolesSection
              userId={user.id}
              roleIds={user.roles}
              editable={canAssignRoles}
              displayName={fullName}
              hasOverrides={user.hasOverrides}
            />
            {canEditOverrides && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => {
                  onClose()
                  navigate(`/admin/users/${user.id}/overrides`)
                }}
              >
                <SlidersHorizontal className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
                {t('rbac.override.editAction')}
              </Button>
            )}
          </section>

          {/* Account meta */}
          <section className="space-y-2">
            <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              {t('admin.users.detail.accountSection')}
            </h3>
            <dl className="grid grid-cols-2 gap-x-4 gap-y-3 text-sm">
              <div>
                <dt className="text-xs text-muted-foreground">{t('admin.users.col.lastLogin')}</dt>
                <dd className="mt-0.5 text-foreground">
                  {formatRelative(user.lastLoginAt, i18n.language, t('admin.users.status.never'))}
                </dd>
              </div>
              {user.status === 'invited' && user.invitedAt && (
                <div>
                  <dt className="text-xs text-muted-foreground">{t('admin.users.detail.invitedAt')}</dt>
                  <dd className="mt-0.5 text-foreground">
                    {formatRelative(user.invitedAt, i18n.language, '—')}
                  </dd>
                </div>
              )}
            </dl>
          </section>

          {/* Access actions */}
          <section className="space-y-2">
            <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              {t('admin.users.detail.accessSection')}
            </h3>
            <div className="flex flex-wrap gap-2">
              {canViewAs && (
                <Button variant="outline" size="sm" onClick={handleViewAs}>
                  <Eye className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
                  {t('rbac.viewAs.action')}
                </Button>
              )}
              {user.status === 'invited' && (
                <Button variant="outline" size="sm" onClick={handleResend} disabled={resendInvite.isPending}>
                  <Send className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
                  {t('admin.users.detail.resendInvite')}
                </Button>
              )}
              {canDeactivate && (
                user.status === 'deactivated' ? (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setStatus('active', t('admin.users.detail.reactivated', { name: fullName }))}
                    disabled={updateUser.isPending}
                  >
                    <UserCheck className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
                    {t('admin.users.detail.reactivate')}
                  </Button>
                ) : (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setConfirmDeactivate(true)}
                    disabled={isSelf || updateUser.isPending}
                    className="text-error hover:text-error"
                  >
                    <UserX className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
                    {user.status === 'invited' ? t('admin.users.detail.revokeInvite') : t('admin.users.detail.deactivate')}
                  </Button>
                )
              )}
            </div>
            {isSelf && canDeactivate && (
              <p className="text-xs text-muted-foreground">{t('admin.users.detail.selfAccessLocked')}</p>
            )}
          </section>
        </div>
      </DetailModal>

      <ConfirmDialog
        open={confirmDeactivate}
        onOpenChange={setConfirmDeactivate}
        variant="destructive"
        title={user.status === 'invited' ? t('admin.users.detail.revokeConfirmTitle') : t('admin.users.detail.deactivateConfirmTitle')}
        description={
          user.status === 'invited'
            ? t('admin.users.detail.revokeConfirmBody', { name: fullName })
            : t('admin.users.detail.deactivateConfirmBody', { name: fullName })
        }
        confirmLabel={user.status === 'invited' ? t('admin.users.detail.revokeInvite') : t('admin.users.detail.deactivate')}
        onConfirm={() => {
          setConfirmDeactivate(false)
          setStatus(
            'deactivated',
            user.status === 'invited'
              ? t('admin.users.detail.inviteRevoked', { name: fullName })
              : t('admin.users.detail.deactivated', { name: fullName }),
          )
        }}
      />
    </>
  )
}

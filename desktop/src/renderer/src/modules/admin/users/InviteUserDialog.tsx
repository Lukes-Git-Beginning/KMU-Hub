/**
 * InviteUserDialog — invite a new tenant account. Email + roles chosen up
 * front (security semantics are clear: the invitee can only do what you grant
 * now). Multi-role since R-2 (union). Creates a pending (`invited`) account;
 * the server mails the invitee a link to its accept page, and a pending invite
 * can be re-sent from the user detail.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Check, Mail } from 'lucide-react'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useInviteUser } from '@/api/hooks/useAdminUsers'
import { useRoles } from '@/api/hooks/useRbacRoles'
import { roleDescription, roleDisplayName, rolePresetKey } from '@/lib/rbac-format'

interface InviteUserDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Seats already consumed and the licensed total — used for the inline limit hint. */
  seatsUsed: number
  seatsTotal: number
}

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

// The demo mock answers `email_exists`; the server says "user already exists"
// or "pending invitation already exists for this email" (both 409).
const DUPLICATE_RE = /email_exists|already exists/i

export function InviteUserDialog({ open, onOpenChange, seatsUsed, seatsTotal }: InviteUserDialogProps) {
  const { t } = useTranslation()
  const invite = useInviteUser()
  const { data: allRoles = [] } = useRoles()

  const [email, setEmail] = useState('')
  // null = untouched: preselect the member preset. Role ids are UUIDs on the
  // server, so the preset is found by key, not by a hardcoded id.
  const [picked, setPicked] = useState<string[] | null>(null)
  const memberRoleId = allRoles.find((r) => rolePresetKey(r) === 'member')?.id
  const roles = picked ?? (memberRoleId ? [memberRoleId] : [])

  const reset = () => {
    setEmail('')
    setPicked(null)
  }

  const toggleRole = (roleId: string) => {
    setPicked(roles.includes(roleId) ? roles.filter((r) => r !== roleId) : [...roles, roleId])
  }

  const emailValid = EMAIL_RE.test(email.trim())
  const seatsFull = seatsUsed >= seatsTotal

  const handleInvite = () => {
    if (!emailValid || roles.length === 0) return
    invite.mutate(
      { email: email.trim(), roles },
      {
        onSuccess: (user) => {
          toast.success(t('admin.users.invite.sent', { email: user.email }))
          reset()
          onOpenChange(false)
        },
        onError: (err) => {
          const msg = err instanceof Error ? err.message : ''
          toast.error(
            DUPLICATE_RE.test(msg)
              ? t('admin.users.invite.errorExists')
              : t('admin.users.invite.errorGeneric'),
          )
        },
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) reset(); onOpenChange(v) }}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{t('admin.users.invite.title')}</DialogTitle>
          <DialogDescription>{t('admin.users.invite.subtitle')}</DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-1">
          {/* Email */}
          <div className="space-y-1.5">
            <Label htmlFor="invite-email">{t('admin.users.invite.email')}</Label>
            <div className="relative">
              <Mail className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
              <Input
                id="invite-email"
                type="email"
                autoFocus
                autoComplete="off"
                placeholder={t('admin.users.invite.emailPlaceholder')}
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                onKeyDown={(e) => { if (e.key === 'Enter' && emailValid) handleInvite() }}
                className="pl-9"
              />
            </div>
          </div>

          {/* Roles (multi-select, union semantics) */}
          <div className="space-y-1.5">
            <Label>{t('admin.users.invite.role')}</Label>
            <ul className="max-h-48 overflow-y-auto rounded-lg border border-border p-1">
              {allRoles.map((role) => {
                const selected = roles.includes(role.id)
                return (
                  <li key={role.id}>
                    <button
                      type="button"
                      role="checkbox"
                      aria-checked={selected}
                      onClick={() => toggleRole(role.id)}
                      className="flex w-full items-center gap-2.5 rounded-md px-2 py-1.5 text-left text-sm transition-colors hover:bg-secondary/60 focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
                    >
                      <span
                        className={`flex h-4 w-4 shrink-0 items-center justify-center rounded border ${
                          selected ? 'border-primary bg-primary text-primary-foreground' : 'border-border'
                        }`}
                        aria-hidden="true"
                      >
                        {selected && <Check className="h-3 w-3" />}
                      </span>
                      <span className="h-2 w-2 shrink-0 rounded-full" style={{ background: role.color }} aria-hidden="true" />
                      <span className="min-w-0 flex-1 truncate">{roleDisplayName(t, role)}</span>
                    </button>
                  </li>
                )
              })}
            </ul>
            <p className="text-xs text-muted-foreground">
              {(() => {
                const only = roles.length === 1 ? allRoles.find((r) => r.id === roles[0]) : undefined
                return only?.isSystem ? roleDescription(t, only) : t('rbac.assignment.unionHint')
              })()}
            </p>
          </div>

          {/* Seat hint — communicate the limit inline, before the invite is sent. */}
          <div
            className={`rounded-lg border px-3 py-2 text-xs ${
              seatsFull
                ? 'border-warning/40 bg-warning-light text-warning'
                : 'border-border bg-secondary/40 text-muted-foreground'
            }`}
          >
            {seatsFull
              ? t('admin.users.invite.seatsFull', { total: seatsTotal })
              : t('admin.users.invite.seatsHint', { used: seatsUsed, total: seatsTotal })}
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => { reset(); onOpenChange(false) }}>
            {t('common.cancel')}
          </Button>
          <Button onClick={handleInvite} disabled={!emailValid || roles.length === 0 || invite.isPending}>
            {invite.isPending ? t('admin.users.invite.sending') : t('admin.users.invite.send')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

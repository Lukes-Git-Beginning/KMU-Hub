/**
 * InviteUserDialog — invite a new tenant account. Email + role chosen up front
 * (security semantics are clear: the invitee can only do what you grant now).
 * Creates a pending (`invited`) account via MSW. Real e-mail/token = Luke 🔒.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Mail } from 'lucide-react'
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { RoleId } from '@/config/roles'
import { useInviteUser } from '@/api/hooks/useAdminUsers'
import { ROLE_DOT, ROLE_ORDER, roleLabelKey } from './presentation'

interface InviteUserDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Seats already consumed and the licensed total — used for the inline limit hint. */
  seatsUsed: number
  seatsTotal: number
}

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

export function InviteUserDialog({ open, onOpenChange, seatsUsed, seatsTotal }: InviteUserDialogProps) {
  const { t } = useTranslation()
  const invite = useInviteUser()

  const [email, setEmail] = useState('')
  const [role, setRole] = useState<RoleId>('member')

  const reset = () => {
    setEmail('')
    setRole('member')
  }

  const emailValid = EMAIL_RE.test(email.trim())
  const seatsFull = seatsUsed >= seatsTotal

  const handleInvite = () => {
    if (!emailValid) return
    invite.mutate(
      { email: email.trim(), role },
      {
        onSuccess: (user) => {
          toast.success(t('admin.users.invite.sent', { email: user.email }))
          reset()
          onOpenChange(false)
        },
        onError: (err) => {
          const msg = err instanceof Error ? err.message : ''
          toast.error(
            msg === 'email_exists'
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

          {/* Role */}
          <div className="space-y-1.5">
            <Label htmlFor="invite-role">{t('admin.users.invite.role')}</Label>
            <Select value={role} onValueChange={(v) => setRole(v as RoleId)}>
              <SelectTrigger id="invite-role">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {ROLE_ORDER.map((r) => (
                  <SelectItem key={r} value={r}>
                    <span className="flex items-center gap-2">
                      <span className={`h-2 w-2 shrink-0 rounded-full ${ROLE_DOT[r]}`} aria-hidden="true" />
                      {t(roleLabelKey(r))}
                    </span>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">{t(`rbac.roles.${role}.description`)}</p>
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
          <Button onClick={handleInvite} disabled={!emailValid || invite.isPending}>
            {invite.isPending ? t('admin.users.invite.sending') : t('admin.users.invite.send')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

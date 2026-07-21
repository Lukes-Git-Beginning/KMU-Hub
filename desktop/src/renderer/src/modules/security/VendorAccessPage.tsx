/**
 * VendorAccessPage — Anbieter-Zugriff (RBAC R-5 B).
 *
 * Drei Sektionen:
 *  - Offene Anfragen (pending / counter_proposed)
 *  - Aktiver Zugang (active)
 *  - Verlauf (declined / expired / revoked / completed)
 *
 * Gated via security:vendor_access:manage.
 */
import { useState, type ElementType, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { useFormatDate } from '@/hooks/useFormatters'
import {
  Shield,
  ShieldAlert,
  ShieldCheck,
  ShieldOff,
  Clock,
  User,
  Ticket,
  AlertTriangle,
  CheckCircle2,
  XCircle,
  CalendarClock,
  RotateCcw,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { ConfirmDialog } from '@/components/shared/ConfirmDialog'
import { EmptyState } from '@/components/shared/EmptyState'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from '@/components/ui/dialog'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import {
  useVendorAccessList,
  useApproveVendorAccess,
  useDeclineVendorAccess,
  useCounterProposeVendorAccess,
  useRevokeVendorAccess,
} from '@/api/vendor-access'
import {
  VENDOR_ACCESS_AREAS,
} from '@/api/vendor-access-types'
import type { VendorAccessRequest, VendorAccessStatus } from '@/api/vendor-access-types'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function hasSensitiveScope(scope: string[]): boolean {
  return scope.some(
    (id) => VENDOR_ACCESS_AREAS.find((a) => a.id === id)?.sensitive === true,
  )
}

function scopeAreas(scope: string[]): typeof VENDOR_ACCESS_AREAS {
  return VENDOR_ACCESS_AREAS.filter((a) => scope.includes(a.id))
}

function outsideAreas(scope: string[]): typeof VENDOR_ACCESS_AREAS {
  return VENDOR_ACCESS_AREAS.filter((a) => !scope.includes(a.id))
}

function daysRemaining(expiresAt: string): number {
  const diff = new Date(expiresAt).getTime() - Date.now()
  return Math.max(0, Math.ceil(diff / (1000 * 60 * 60 * 24)))
}

function todayIso(): string {
  const d = new Date()
  d.setDate(d.getDate() + 1)
  return d.toISOString().split('T')[0]
}

// ---------------------------------------------------------------------------
// Status Pill
// ---------------------------------------------------------------------------

const STATUS_CONFIG: Record<
  VendorAccessStatus,
  { labelKey: string; className: string }
> = {
  pending: {
    labelKey: 'rbac.vendorAccess.status.pending',
    className: 'bg-warning/15 text-warning-foreground',
  },
  counter_proposed: {
    labelKey: 'rbac.vendorAccess.status.counter_proposed',
    className: 'bg-info/15 text-info',
  },
  active: {
    labelKey: 'rbac.vendorAccess.status.active',
    className: 'bg-success/15 text-success-foreground',
  },
  declined: {
    labelKey: 'rbac.vendorAccess.status.declined',
    className: 'bg-muted text-muted-foreground',
  },
  expired: {
    labelKey: 'rbac.vendorAccess.status.expired',
    className: 'bg-muted text-muted-foreground',
  },
  revoked: {
    labelKey: 'rbac.vendorAccess.status.revoked',
    className: 'bg-error-light text-destructive',
  },
  completed: {
    labelKey: 'rbac.vendorAccess.status.completed',
    className: 'bg-secondary text-secondary-foreground',
  },
}

function StatusPill({ status }: { status: VendorAccessStatus }) {
  const { t } = useTranslation()
  const cfg = STATUS_CONFIG[status]
  return (
    <span
      className={[
        'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
        cfg.className,
      ].join(' ')}
    >
      {t(cfg.labelKey)}
    </span>
  )
}

// ---------------------------------------------------------------------------
// Area chips
// ---------------------------------------------------------------------------

function AreaChip({
  labelKey,
  sensitive,
  muted,
}: {
  labelKey: string
  sensitive?: boolean
  muted?: boolean
}) {
  const { t } = useTranslation()
  return (
    <span
      className={[
        'inline-flex items-center rounded-md px-2 py-0.5 text-xs',
        muted
          ? 'bg-muted text-muted-foreground line-through'
          : sensitive
            ? 'bg-destructive/10 text-destructive font-medium'
            : 'bg-secondary text-secondary-foreground',
      ].join(' ')}
    >
      {t(labelKey)}
    </span>
  )
}

// ---------------------------------------------------------------------------
// Scope lists
// ---------------------------------------------------------------------------

function ScopeLists({ scope }: { scope: string[] }) {
  const { t } = useTranslation()
  const included = scopeAreas(scope)
  const excluded = outsideAreas(scope)

  return (
    <div className="mt-3 space-y-2">
      <div>
        <p className="mb-1.5 text-xs font-medium text-foreground">
          {t('rbac.vendorAccess.scopeAccess')}
        </p>
        <div className="flex flex-wrap gap-1.5">
          {included.map((a) => (
            <AreaChip key={a.id} labelKey={a.labelKey} sensitive={a.sensitive} />
          ))}
        </div>
      </div>
      {excluded.length > 0 && (
        <div>
          <p className="mb-1.5 text-xs font-medium text-muted-foreground">
            {t('rbac.vendorAccess.scopeNoAccess')}
          </p>
          <div className="flex flex-wrap gap-1.5">
            {excluded.map((a) => (
              <AreaChip key={a.id} labelKey={a.labelKey} muted />
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Sensitive warning block
// ---------------------------------------------------------------------------

function SensitiveWarning() {
  const { t } = useTranslation()
  return (
    <div className="mt-3 flex items-start gap-2.5 rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2.5">
      <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-destructive" />
      <p className="text-sm text-destructive">
        {t('rbac.vendorAccess.sensitiveWarning')}
      </p>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Approve dialog (with optional sensitive ack)
// ---------------------------------------------------------------------------

interface ApproveDialogProps {
  request: VendorAccessRequest
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: (sensitiveAck: boolean) => void
  isPending: boolean
}

function ApproveDialog({
  request,
  open,
  onOpenChange,
  onConfirm,
  isPending,
}: ApproveDialogProps) {
  const { t } = useTranslation()
  const [ack, setAck] = useState(false)
  const isSensitive = hasSensitiveScope(request.scope)

  function handleConfirm() {
    onConfirm(ack)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <ShieldCheck className="h-5 w-5 text-primary" />
            {t('rbac.vendorAccess.approveDialog.title')}
          </DialogTitle>
          <DialogDescription>
            {t('rbac.vendorAccess.approveDialog.description', { reason: request.reason })}
          </DialogDescription>
        </DialogHeader>

        {isSensitive && (
          <div className="rounded-lg border border-destructive/30 bg-destructive/5 p-3">
            <p className="mb-3 text-sm font-medium text-destructive">
              {t('rbac.vendorAccess.approveDialog.sensitiveHint')}
            </p>
            <div className="flex items-start gap-2">
              <Checkbox
                id="sensitive-ack"
                checked={ack}
                onCheckedChange={(v) => setAck(v === true)}
              />
              <Label htmlFor="sensitive-ack" className="cursor-pointer text-sm leading-snug">
                {t('rbac.vendorAccess.approveDialog.sensitiveAckLabel')}
              </Label>
            </div>
          </div>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t('common.cancel')}
          </Button>
          <Button
            onClick={handleConfirm}
            disabled={isPending || (isSensitive && !ack)}
          >
            {t('rbac.vendorAccess.approveDialog.confirm')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// Counter-propose dialog
// ---------------------------------------------------------------------------

interface CounterProposeDialogProps {
  request: VendorAccessRequest
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: (proposedStart: string) => void
  isPending: boolean
}

function CounterProposeDialog({
  request,
  open,
  onOpenChange,
  onConfirm,
  isPending,
}: CounterProposeDialogProps) {
  const { t } = useTranslation()
  const [proposedStart, setProposedStart] = useState('')
  const minDate = todayIso()

  function handleConfirm() {
    if (!proposedStart) return
    onConfirm(proposedStart)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <CalendarClock className="h-5 w-5 text-primary" />
            {t('rbac.vendorAccess.counterDialog.title')}
          </DialogTitle>
          <DialogDescription>
            {t('rbac.vendorAccess.counterDialog.description', { reason: request.reason })}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-2">
          <Label htmlFor="proposed-start">
            {t('rbac.vendorAccess.counterDialog.startLabel')}
          </Label>
          <Input
            id="proposed-start"
            type="date"
            min={minDate}
            value={proposedStart}
            onChange={(e) => setProposedStart(e.target.value)}
          />
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t('common.cancel')}
          </Button>
          <Button
            onClick={handleConfirm}
            disabled={isPending || !proposedStart}
          >
            {t('rbac.vendorAccess.counterDialog.confirm')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// Pending request card
// ---------------------------------------------------------------------------

interface PendingCardProps {
  request: VendorAccessRequest
}

function PendingCard({ request }: PendingCardProps) {
  const { t } = useTranslation()
  const formatDate = useFormatDate()
  const [approveOpen, setApproveOpen] = useState(false)
  const [declineOpen, setDeclineOpen] = useState(false)
  const [counterOpen, setCounterOpen] = useState(false)

  const approve = useApproveVendorAccess()
  const decline = useDeclineVendorAccess()
  const counterPropose = useCounterProposeVendorAccess()

  const isSensitive = hasSensitiveScope(request.scope)
  const isCounterProposed = request.status === 'counter_proposed'

  return (
    <div className="rounded-xl border border-border bg-card p-5">
      {/* Header */}
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="text-sm font-semibold text-foreground">{request.reason}</h3>
            {request.ticket_ref && (
              <span className="inline-flex items-center gap-1 rounded-md bg-secondary px-2 py-0.5 text-xs text-muted-foreground">
                <Ticket className="h-3 w-3" />
                {request.ticket_ref}
              </span>
            )}
          </div>
          <p className="mt-1 text-sm text-muted-foreground">{request.description}</p>
        </div>
        <StatusPill status={request.status} />
      </div>

      {/* Counter-proposed notice */}
      {isCounterProposed && request.counter_proposed_start && (
        <div className="mt-3 flex items-center gap-2 rounded-lg bg-info/10 px-3 py-2 text-sm text-info">
          <CalendarClock className="h-4 w-4 shrink-0" />
          {t('rbac.vendorAccess.counterProposedNotice', {
            date: formatDate(request.counter_proposed_start, 'medium'),
          })}
        </div>
      )}

      {/* Meta */}
      <div className="mt-3 flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
        <span className="flex items-center gap-1">
          <Clock className="h-3.5 w-3.5" />
          {t('rbac.vendorAccess.meta.period', {
            start: formatDate(request.requested_start, 'short'),
            days: request.duration_days,
            end: formatDate(request.expires_at, 'short'),
          })}
        </span>
        <span className="flex items-center gap-1">
          <User className="h-3.5 w-3.5" />
          {request.agents.map((a) => a.name).join(', ')}
        </span>
      </div>

      {/* Sensitive warning */}
      {isSensitive && <SensitiveWarning />}

      {/* Scope lists */}
      <ScopeLists scope={request.scope} />

      {/* Actions — hidden if counter_proposed (waiting for Zentria) */}
      {!isCounterProposed && (
        <div className="mt-4 flex flex-wrap gap-2">
          <Button
            size="sm"
            onClick={() => setApproveOpen(true)}
          >
            <CheckCircle2 className="mr-1.5 h-3.5 w-3.5" />
            {t('rbac.vendorAccess.action.approve')}
          </Button>
          <Button
            size="sm"
            variant="outline"
            onClick={() => setCounterOpen(true)}
          >
            <CalendarClock className="mr-1.5 h-3.5 w-3.5" />
            {t('rbac.vendorAccess.action.counterPropose')}
          </Button>
          <Button
            size="sm"
            variant="ghost"
            className="text-muted-foreground hover:text-destructive"
            onClick={() => setDeclineOpen(true)}
          >
            <XCircle className="mr-1.5 h-3.5 w-3.5" />
            {t('rbac.vendorAccess.action.decline')}
          </Button>
        </div>
      )}

      {/* Dialogs */}
      <ApproveDialog
        request={request}
        open={approveOpen}
        onOpenChange={setApproveOpen}
        isPending={approve.isPending}
        onConfirm={(sensitiveAck) => {
          approve.mutate(
            { id: request.id, data: sensitiveAck ? { sensitive_ack: true } : {} },
            { onSuccess: () => setApproveOpen(false) },
          )
        }}
      />

      <ConfirmDialog
        open={declineOpen}
        onOpenChange={setDeclineOpen}
        title={t('rbac.vendorAccess.declineDialog.title')}
        description={t('rbac.vendorAccess.declineDialog.description', {
          reason: request.reason,
        })}
        confirmLabel={t('rbac.vendorAccess.action.decline')}
        variant="warning"
        onConfirm={() => {
          decline.mutate(request.id, { onSuccess: () => setDeclineOpen(false) })
        }}
      />

      <CounterProposeDialog
        request={request}
        open={counterOpen}
        onOpenChange={setCounterOpen}
        isPending={counterPropose.isPending}
        onConfirm={(proposedStart) => {
          counterPropose.mutate(
            { id: request.id, data: { proposed_start: proposedStart } },
            { onSuccess: () => setCounterOpen(false) },
          )
        }}
      />
    </div>
  )
}

// ---------------------------------------------------------------------------
// Active access card
// ---------------------------------------------------------------------------

interface ActiveCardProps {
  request: VendorAccessRequest
}

function ActiveCard({ request }: ActiveCardProps) {
  const { t } = useTranslation()
  const formatDate = useFormatDate()
  const [revokeOpen, setRevokeOpen] = useState(false)
  const revoke = useRevokeVendorAccess()

  const daysLeft = daysRemaining(request.expires_at)

  return (
    <div className="rounded-xl border border-border bg-card p-5">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="text-sm font-semibold text-foreground">{request.reason}</h3>
            {request.ticket_ref && (
              <span className="inline-flex items-center gap-1 rounded-md bg-secondary px-2 py-0.5 text-xs text-muted-foreground">
                <Ticket className="h-3 w-3" />
                {request.ticket_ref}
              </span>
            )}
          </div>
          <p className="mt-1 text-sm text-muted-foreground">{request.description}</p>
        </div>
        <StatusPill status="active" />
      </div>

      <div className="mt-3 flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
        <span className="flex items-center gap-1 font-medium text-foreground">
          <Clock className="h-3.5 w-3.5" />
          {t('rbac.vendorAccess.meta.daysLeft', { count: daysLeft })}
        </span>
        <span className="flex items-center gap-1">
          {t('rbac.vendorAccess.meta.expiresOn', {
            date: formatDate(request.expires_at, 'medium'),
          })}
        </span>
        <span className="flex items-center gap-1">
          <User className="h-3.5 w-3.5" />
          {request.agents.map((a) => a.name).join(', ')}
        </span>
      </div>

      {request.approved_by && (
        <p className="mt-1 text-xs text-muted-foreground">
          {t('rbac.vendorAccess.meta.approvedBy', { name: request.approved_by })}
        </p>
      )}

      <ScopeLists scope={request.scope} />

      <div className="mt-4">
        <Button
          size="sm"
          variant="outline"
          className="border-destructive/40 text-destructive hover:bg-destructive/5"
          onClick={() => setRevokeOpen(true)}
        >
          <ShieldOff className="mr-1.5 h-3.5 w-3.5" />
          {t('rbac.vendorAccess.action.revoke')}
        </Button>
      </div>

      <ConfirmDialog
        open={revokeOpen}
        onOpenChange={setRevokeOpen}
        title={t('rbac.vendorAccess.revokeDialog.title')}
        description={t('rbac.vendorAccess.revokeDialog.description', {
          reason: request.reason,
        })}
        confirmLabel={t('rbac.vendorAccess.action.revoke')}
        variant="destructive"
        onConfirm={() => {
          revoke.mutate(request.id, { onSuccess: () => setRevokeOpen(false) })
        }}
      />
    </div>
  )
}

// ---------------------------------------------------------------------------
// History row
// ---------------------------------------------------------------------------

function HistoryRow({ request }: { request: VendorAccessRequest }) {
  const { t } = useTranslation()
  const formatDate = useFormatDate()

  return (
    <div className="flex items-center justify-between gap-4 py-3 border-b border-border last:border-0">
      <div className="min-w-0">
        <p className="text-sm font-medium text-foreground truncate">{request.reason}</p>
        <p className="mt-0.5 text-xs text-muted-foreground">
          {formatDate(request.requested_start, 'short')}
          {' — '}
          {formatDate(request.expires_at, 'short')}
          {' · '}
          {request.agents.map((a) => a.name).join(', ')}
        </p>
      </div>
      <StatusPill status={request.status} />
    </div>
  )
}

// ---------------------------------------------------------------------------
// Section wrapper
// ---------------------------------------------------------------------------

function Section({
  title,
  icon: Icon,
  children,
}: {
  title: string
  icon: ElementType
  children: ReactNode
}) {
  return (
    <section>
      <div className="mb-4 flex items-center gap-2">
        <Icon className="h-4 w-4 text-muted-foreground" />
        <h2 className="text-sm font-semibold text-foreground">{title}</h2>
      </div>
      {children}
    </section>
  )
}

// ---------------------------------------------------------------------------
// Main page
// ---------------------------------------------------------------------------

export default function VendorAccessPage() {
  const { t } = useTranslation()
  const { data: requests = [], isLoading } = useVendorAccessList()

  const pending = requests.filter(
    (r) => r.status === 'pending' || r.status === 'counter_proposed',
  )
  const active = requests.filter((r) => r.status === 'active')
  const history = requests.filter(
    (r) =>
      r.status === 'declined' ||
      r.status === 'expired' ||
      r.status === 'revoked' ||
      r.status === 'completed',
  )

  if (isLoading) {
    return (
      <div className="px-6 py-6 space-y-4">
        {[1, 2].map((i) => (
          <div
            key={i}
            className="h-40 rounded-xl bg-muted animate-pulse"
          />
        ))}
      </div>
    )
  }

  return (
    <div className="px-6 py-6 space-y-8">
      {/* Header */}
      <div className="flex items-start gap-3">
        <div className="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/10">
          <Shield className="h-4.5 w-4.5 text-primary" />
        </div>
        <div>
          <h1 className="text-base font-semibold text-foreground">
            {t('rbac.vendorAccess.pageTitle')}
          </h1>
          <p className="mt-0.5 text-sm text-muted-foreground">
            {t('rbac.vendorAccess.pageDescription')}
          </p>
        </div>
      </div>

      {/* Offene Anfragen */}
      <Section title={t('rbac.vendorAccess.section.pending')} icon={ShieldAlert}>
        {pending.length === 0 ? (
          <EmptyState
            icon={CheckCircle2}
            title={t('rbac.vendorAccess.empty.pending.title')}
            description={t('rbac.vendorAccess.empty.pending.description')}
          />
        ) : (
          <div className="space-y-4">
            {pending.map((r) => (
              <PendingCard key={r.id} request={r} />
            ))}
          </div>
        )}
      </Section>

      {/* Aktiver Zugang */}
      <Section title={t('rbac.vendorAccess.section.active')} icon={ShieldCheck}>
        {active.length === 0 ? (
          <EmptyState
            icon={Shield}
            title={t('rbac.vendorAccess.empty.active.title')}
            description={t('rbac.vendorAccess.empty.active.description')}
          />
        ) : (
          <div className="space-y-4">
            {active.map((r) => (
              <ActiveCard key={r.id} request={r} />
            ))}
          </div>
        )}
      </Section>

      {/* Verlauf */}
      <Section title={t('rbac.vendorAccess.section.history')} icon={RotateCcw}>
        {history.length === 0 ? (
          <EmptyState
            icon={Shield}
            title={t('rbac.vendorAccess.empty.history.title')}
          />
        ) : (
          <div className="rounded-xl border border-border bg-card px-4">
            {history.map((r) => (
              <HistoryRow key={r.id} request={r} />
            ))}
          </div>
        )}
      </Section>
    </div>
  )
}

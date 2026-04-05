import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Plus, Palmtree, Thermometer, Home, GraduationCap, HelpCircle,
  X, Calendar, Loader2, AlertTriangle, Upload,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { cn } from '@/lib/cn'
import { toast } from 'sonner'
import {
  useLeaveRequests,
  useLeaveBalance,
  useLeaveTypes,
  useCreateLeaveRequest,
  useCancelLeaveRequest,
  useRecordSickLeave,
} from '@/api/hooks/hr-hooks'
import { useAuthStore } from '@/stores/auth'
import type { LeaveRequestStatus } from '@/api/hr-types'

type FilterKey = 'all' | LeaveRequestStatus

const LEAVE_TYPE_ICONS: Record<string, typeof Palmtree> = {
  urlaub: Palmtree,
  krankheit: Thermometer,
  homeoffice: Home,
  weiterbildung: GraduationCap,
  sonderurlaub: HelpCircle,
}

export default function AbwesenheitenTab() {
  const { t } = useTranslation()

  const STATUS_BADGES: Record<string, { label: string; className: string }> = {
    pending: { label: t('profil.absence.statusPending'), className: 'bg-warning-light text-warning-foreground' },
    approved: { label: t('profil.absence.statusApproved'), className: 'bg-success-light text-success' },
    rejected: { label: t('profil.absence.statusRejected'), className: 'bg-error-light text-destructive' },
    cancelled: { label: t('profil.absence.statusCancelled'), className: 'bg-gray-100 text-gray-600 dark:bg-gray-900/30 dark:text-gray-400' },
  }
  const [filter, setFilter] = useState<FilterKey>('all')
  const [showDialog, setShowDialog] = useState(false)
  const [showSickDialog, setShowSickDialog] = useState(false)
  const [formLeaveTypeId, setFormLeaveTypeId] = useState('')
  const [formStart, setFormStart] = useState('')
  const [formEnd, setFormEnd] = useState('')
  const [formReason, setFormReason] = useState('')
  const [formHalfDayStart, setFormHalfDayStart] = useState(false)
  const [formHalfDayPeriodStart, setFormHalfDayPeriodStart] = useState<'morning' | 'afternoon'>('morning')
  const [formHalfDayEnd, setFormHalfDayEnd] = useState(false)
  const [formHalfDayPeriodEnd, setFormHalfDayPeriodEnd] = useState<'morning' | 'afternoon'>('afternoon')

  // Sick leave form
  const [sickStart, setSickStart] = useState('')
  const [sickEnd, setSickEnd] = useState('')
  const [sickReason, setSickReason] = useState('')

  const user = useAuthStore((s) => s.user)

  // TanStack Query hooks
  const { data: requestsData, isLoading: requestsLoading } = useLeaveRequests(
    user ? { employee_id: user.id } : undefined,
  )
  const { data: balance } = useLeaveBalance()
  const { data: leaveTypes } = useLeaveTypes()
  const createMutation = useCreateLeaveRequest()
  const cancelMutation = useCancelLeaveRequest()
  const sickMutation = useRecordSickLeave()

  const requests = requestsData?.requests ?? []
  const filtered = filter === 'all' ? requests : requests.filter((r) => r.status === filter)
  const sorted = [...filtered].sort((a, b) => b.createdAt.localeCompare(a.createdAt))

  // Stats
  const vacationUsed = balance?.used ?? 0
  const vacationTotal = balance?.totalEntitlement ?? 0
  const vacationRemaining = balance?.remaining ?? 0
  const carriedOver = balance?.carriedOver ?? 0
  const carryoverExpired = balance?.carryoverExpired ?? false
  const vacationPercent = vacationTotal > 0 ? Math.round((vacationUsed / vacationTotal) * 100) : 0

  // Carryover warning: show if carried over > 0 and it's past February 1
  const now = new Date()
  const showCarryoverWarning = carriedOver > 0 && (now.getMonth() >= 1) // February = month 1

  const handleSubmit = () => {
    if (!formStart || !formEnd || !formLeaveTypeId) {
      toast.error(t('profil.absence.errorRequiredFields'))
      return
    }
    if (formEnd < formStart) {
      toast.error(t('profil.absence.errorEndAfterStart'))
      return
    }
    createMutation.mutate(
      {
        leaveTypeId: formLeaveTypeId,
        startDate: formStart,
        endDate: formEnd,
        isHalfDayStart: formHalfDayStart,
        halfDayPeriodStart: formHalfDayStart ? formHalfDayPeriodStart : undefined,
        isHalfDayEnd: formHalfDayEnd,
        halfDayPeriodEnd: formHalfDayEnd ? formHalfDayPeriodEnd : undefined,
        reason: formReason.trim() || undefined,
      },
      {
        onSuccess: () => {
          setShowDialog(false)
          resetForm()
        },
      },
    )
  }

  const handleSickSubmit = () => {
    if (!sickStart || !sickEnd) {
      toast.error(t('profil.absence.errorStartEndRequired'))
      return
    }
    sickMutation.mutate(
      {
        startDate: sickStart,
        endDate: sickEnd,
        reason: sickReason.trim() || undefined,
      },
      {
        onSuccess: (data) => {
          setShowSickDialog(false)
          setSickStart('')
          setSickEnd('')
          setSickReason('')
          // Prompt AU document upload if required
          if (data.request.auDocumentRequired) {
            toast.warning(t('profil.absence.auDocumentRequired'), {
              duration: 8000,
            })
          }
        },
      },
    )
  }

  const resetForm = () => {
    setFormLeaveTypeId('')
    setFormStart('')
    setFormEnd('')
    setFormReason('')
    setFormHalfDayStart(false)
    setFormHalfDayEnd(false)
  }

  const formatDate = (dateStr: string) => {
    const d = new Date(dateStr)
    return `${d.getDate().toString().padStart(2, '0')}.${(d.getMonth() + 1).toString().padStart(2, '0')}.${d.getFullYear()}`
  }

  return (
    <div className="max-w-3xl mx-auto p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-bold text-foreground">{t('profil.absence.title')}</h2>
        <div className="flex items-center gap-2">
          <Button size="sm" variant="outline" onClick={() => setShowSickDialog(true)} className="gap-2">
            <Thermometer className="h-4 w-4" />
            {t('profil.absence.sickReport')}
          </Button>
          <Button size="sm" onClick={() => setShowDialog(true)} className="gap-2">
            <Plus className="h-4 w-4" />
            {t('profil.absence.newRequest')}
          </Button>
        </div>
      </div>

      {/* Carryover Warning */}
      {showCarryoverWarning && !carryoverExpired && (
        <div className="rounded-lg border border-warning bg-warning-light/30 p-3 flex items-start gap-2">
          <AlertTriangle className="h-4 w-4 text-warning shrink-0 mt-0.5" />
          <div>
            <p className="text-xs font-medium text-warning">{t('profil.absence.carryoverTitle')}</p>
            <p className="text-xs text-muted-foreground">
              {t('profil.absence.carryoverWarning', { days: carriedOver })}
            </p>
          </div>
        </div>
      )}
      {carryoverExpired && carriedOver > 0 && (
        <div className="rounded-lg border border-error bg-error-light/30 p-3 flex items-start gap-2">
          <AlertTriangle className="h-4 w-4 text-error shrink-0 mt-0.5" />
          <div>
            <p className="text-xs font-medium text-error">{t('profil.absence.carryoverExpiredTitle')}</p>
            <p className="text-xs text-muted-foreground">
              {t('profil.absence.carryoverExpired', { days: carriedOver })}
            </p>
          </div>
        </div>
      )}

      {/* Filter */}
      <div className="flex flex-wrap gap-2">
        <button
          onClick={() => setFilter('all')}
          className={cn(
            'px-3 py-1.5 rounded-full text-xs font-medium border transition-colors',
            filter === 'all'
              ? 'bg-primary text-primary-foreground border-primary'
              : 'bg-card text-muted-foreground border-border hover:border-primary/50',
          )}
        >
          {t('profil.absence.filterAll')} ({requests.length})
        </button>
        {(['pending', 'approved', 'rejected', 'cancelled'] as const).map((status) => {
          const count = requests.filter((r) => r.status === status).length
          return (
            <button
              key={status}
              onClick={() => setFilter(status)}
              className={cn(
                'px-3 py-1.5 rounded-full text-xs font-medium border transition-colors',
                filter === status
                  ? 'bg-primary text-primary-foreground border-primary'
                  : 'bg-card text-muted-foreground border-border hover:border-primary/50',
              )}
            >
              {STATUS_BADGES[status].label} ({count})
            </button>
          )
        })}
      </div>

      {/* Requests List */}
      <div className="space-y-3">
        {requestsLoading && (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="h-6 w-6 animate-spin text-primary" />
          </div>
        )}
        {!requestsLoading && sorted.length === 0 && (
          <div className="text-center py-12 text-muted-foreground">
            <Calendar className="h-10 w-10 mx-auto mb-3 opacity-50" />
            <p className="font-medium">{t('profil.absence.noAbsences')}</p>
            <p className="text-sm">{t('profil.absence.createNewRequest')}</p>
          </div>
        )}
        {sorted.map((request) => {
          const typeKey = request.leaveType?.key ?? ''
          const Icon = LEAVE_TYPE_ICONS[typeKey] ?? HelpCircle
          const statusBadge = STATUS_BADGES[request.status] ?? STATUS_BADGES.pending
          return (
            <div
              key={request.id}
              className="flex items-start gap-4 p-4 rounded-xl border border-border bg-card hover:border-primary/30 transition-colors"
            >
              <div className="mt-0.5 text-primary">
                <Icon className="h-5 w-5" />
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <span className="font-medium text-sm text-foreground">{request.leaveType?.name ?? t('profil.absence.title')}</span>
                  <span className="text-sm text-muted-foreground">
                    {formatDate(request.startDate)} - {formatDate(request.endDate)}
                  </span>
                  <span className="text-xs text-muted-foreground">
                    ({request.totalDays} {request.totalDays === 1 ? t('profil.absence.day') : t('profil.absence.days')})
                    {request.isHalfDayStart && ` ${t('profil.absence.halfDayStart')}`}
                    {request.isHalfDayEnd && ` ${t('profil.absence.halfDayEnd')}`}
                  </span>
                </div>
                {request.reason && <p className="text-sm text-muted-foreground">{request.reason}</p>}
                {request.approvalComment && (
                  <p className="text-xs text-muted-foreground mt-1 italic">{t('profil.absence.comment')}: {request.approvalComment}</p>
                )}
                {request.auDocumentRequired && !request.auDocumentFileId && (
                  <div className="flex items-center gap-1.5 mt-1 text-xs text-warning">
                    <Upload className="h-3 w-3" />
                    <span>{t('profil.absence.auPending')}</span>
                  </div>
                )}
              </div>
              <div className="flex items-center gap-2">
                <span className={cn('px-2.5 py-0.5 rounded-full text-xs font-medium', statusBadge.className)}>
                  {statusBadge.label}
                </span>
                {request.status === 'pending' && (
                  <button
                    onClick={() => cancelMutation.mutate(request.id)}
                    className="p-1 rounded hover:bg-destructive/10 text-muted-foreground hover:text-destructive transition-colors"
                    disabled={cancelMutation.isPending}
                  >
                    <X className="h-4 w-4" />
                  </button>
                )}
              </div>
            </div>
          )
        })}
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        {/* Vacation Balance */}
        <div className="rounded-xl border border-border bg-card p-5">
          <h3 className="text-sm font-semibold text-foreground mb-3">{t('profil.absence.remainingVacation')}</h3>
          <div className="flex items-end gap-2 mb-3">
            <span className="text-3xl font-bold text-primary">{vacationRemaining}</span>
            <span className="text-sm text-muted-foreground mb-1">/ {vacationTotal} {t('profil.absence.days')}</span>
          </div>
          <div className="w-full h-3 rounded-full bg-secondary overflow-hidden">
            <div
              className="h-full rounded-full bg-primary transition-all"
              style={{ width: `${vacationPercent}%` }}
            />
          </div>
          <div className="flex justify-between mt-2">
            <p className="text-xs text-muted-foreground">{t('profil.absence.daysTaken', { count: vacationUsed })}</p>
            {carriedOver > 0 && (
              <p className={cn('text-xs', carryoverExpired ? 'text-error' : 'text-muted-foreground')}>
                {carriedOver} {t('profil.absence.carryover')}{carryoverExpired ? ` (${t('profil.absence.expired')})` : ''}
              </p>
            )}
          </div>
        </div>

        {/* Yearly Overview */}
        <div className="rounded-xl border border-border bg-card p-5">
          <h3 className="text-sm font-semibold text-foreground mb-3">
            {t('profil.absence.yearlyOverview')} {balance?.year ?? new Date().getFullYear()}
          </h3>
          <div className="space-y-2">
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">{t('profil.absence.entitlement')}</span>
              <span className="font-medium text-foreground">{vacationTotal} {t('profil.absence.days')}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">{t('profil.absence.taken')}</span>
              <span className="font-medium text-foreground">{vacationUsed} {t('profil.absence.days')}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">{t('profil.absence.remaining')}</span>
              <span className="font-medium text-primary">{vacationRemaining} {t('profil.absence.days')}</span>
            </div>
            {carriedOver > 0 && (
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">{t('profil.absence.carryover')}</span>
                <span className={cn('font-medium', carryoverExpired ? 'text-error line-through' : 'text-foreground')}>
                  {carriedOver} {t('profil.absence.days')}
                </span>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* New Request Dialog */}
      <Dialog open={showDialog} onOpenChange={(v) => { if (!v) resetForm(); setShowDialog(v) }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('profil.absence.requestNewTitle')}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">{t('profil.absence.leaveType')} *</label>
              <Select value={formLeaveTypeId} onValueChange={setFormLeaveTypeId}>
                <SelectTrigger>
                  <SelectValue placeholder={t('profil.absence.selectType')} />
                </SelectTrigger>
                <SelectContent>
                  {(leaveTypes ?? []).map((t) => (
                    <SelectItem key={t.id} value={t.id}>
                      <span className="flex items-center gap-2">
                        <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: t.color }} />
                        {t.name}
                      </span>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">{t('profil.absence.from')} *</label>
                <Input type="date" value={formStart} onChange={(e) => setFormStart(e.target.value)} />
                <div className="flex items-center gap-2 mt-1">
                  <Checkbox
                    id="halfDayStart"
                    checked={formHalfDayStart}
                    onCheckedChange={(v) => setFormHalfDayStart(v === true)}
                  />
                  <Label htmlFor="halfDayStart" className="text-xs font-normal cursor-pointer">{t('profil.absence.halfDay')}</Label>
                  {formHalfDayStart && (
                    <Select value={formHalfDayPeriodStart} onValueChange={(v) => setFormHalfDayPeriodStart(v as 'morning' | 'afternoon')}>
                      <SelectTrigger className="h-7 w-32 text-xs">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="morning">{t('profil.absence.morning')}</SelectItem>
                        <SelectItem value="afternoon">{t('profil.absence.afternoon')}</SelectItem>
                      </SelectContent>
                    </Select>
                  )}
                </div>
              </div>
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">{t('profil.absence.to')} *</label>
                <Input type="date" value={formEnd} onChange={(e) => setFormEnd(e.target.value)} />
                <div className="flex items-center gap-2 mt-1">
                  <Checkbox
                    id="halfDayEnd"
                    checked={formHalfDayEnd}
                    onCheckedChange={(v) => setFormHalfDayEnd(v === true)}
                  />
                  <Label htmlFor="halfDayEnd" className="text-xs font-normal cursor-pointer">{t('profil.absence.halfDay')}</Label>
                  {formHalfDayEnd && (
                    <Select value={formHalfDayPeriodEnd} onValueChange={(v) => setFormHalfDayPeriodEnd(v as 'morning' | 'afternoon')}>
                      <SelectTrigger className="h-7 w-32 text-xs">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="morning">{t('profil.absence.morning')}</SelectItem>
                        <SelectItem value="afternoon">{t('profil.absence.afternoon')}</SelectItem>
                      </SelectContent>
                    </Select>
                  )}
                </div>
              </div>
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">{t('profil.absence.reason')} ({t('common.optional')})</label>
              <textarea
                value={formReason}
                onChange={(e) => setFormReason(e.target.value)}
                rows={3}
                placeholder={t('profil.absence.reasonPlaceholder')}
                className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => { resetForm(); setShowDialog(false) }}>{t('common.cancel')}</Button>
            <Button onClick={handleSubmit} disabled={createMutation.isPending}>
              {createMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : null}
              {t('profil.absence.submitRequest')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Sick Leave Dialog */}
      <Dialog open={showSickDialog} onOpenChange={setShowSickDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('profil.absence.sickReportTitle')}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">{t('profil.absence.from')} *</label>
                <Input type="date" value={sickStart} onChange={(e) => setSickStart(e.target.value)} />
              </div>
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">{t('profil.absence.to')} *</label>
                <Input type="date" value={sickEnd} onChange={(e) => setSickEnd(e.target.value)} />
              </div>
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">{t('profil.absence.note')} ({t('common.optional')})</label>
              <textarea
                value={sickReason}
                onChange={(e) => setSickReason(e.target.value)}
                rows={2}
                placeholder={t('profil.absence.optionalNote')}
                className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              />
            </div>
            <p className="text-xs text-muted-foreground">
              {t('profil.absence.sickThresholdNote')}
            </p>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowSickDialog(false)}>{t('common.cancel')}</Button>
            <Button onClick={handleSickSubmit} disabled={sickMutation.isPending}>
              {sickMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : null}
              {t('profil.absence.submitSickReport')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

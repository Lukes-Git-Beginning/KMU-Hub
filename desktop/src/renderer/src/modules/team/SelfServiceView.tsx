import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  User,
  Calendar,
  FileText,
  Download,
  Clock,
  Briefcase,
  Mail,
  Phone,
  MapPin,
  Building2,
  Send,
  CheckCircle2,
  AlertCircle,
  ChevronRight,
  Loader2,
} from 'lucide-react'
import { toast } from 'sonner'
import { formatDate } from '@/lib/format'
import { InlineStat } from '@/components/shared'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select'
import {
  useSelfProfile,
  useLeaveBalance,
  useLeaveRequests,
  useLeaveTypes,
  useCreateLeaveRequest,
} from '@/api/hooks/hr-hooks'

// ============================================================
// Types
// ============================================================

type SelfServiceTab = 'profil' | 'antraege' | 'dokumente' | 'zeitkonto'

interface SalaryStatement {
  id: string
  month: string
  label: string
  gross: number
  net: number
}

interface TimeAccount {
  label: string
  hours: number
  type: 'positive' | 'negative' | 'neutral'
}

// ============================================================
// Demo data without a backend yet (payroll + time account = Luke later)
// ============================================================

const SALARY_STATEMENTS: SalaryStatement[] = [
  { id: 'ss-1', month: '2026-01', label: 'Januar 2026', gross: 7200, net: 4856 },
  { id: 'ss-2', month: '2025-12', label: 'Dezember 2025', gross: 7200, net: 4856 },
  { id: 'ss-3', month: '2025-11', label: 'November 2025', gross: 7200, net: 4856 },
  { id: 'ss-4', month: '2025-10', label: 'Oktober 2025', gross: 7200, net: 4856 },
  { id: 'ss-5', month: '2025-09', label: 'September 2025', gross: 7200, net: 4856 },
  { id: 'ss-6', month: '2025-08', label: 'August 2025', gross: 7200, net: 4856 },
]

const TIME_ACCOUNTS: TimeAccount[] = [
  { label: 'Überstunden (Monat)', hours: 4.5, type: 'positive' },
  { label: 'Überstunden (gesamt)', hours: 18.25, type: 'positive' },
  { label: 'Gleitzeit-Saldo', hours: 2.75, type: 'positive' },
  { label: 'Resturlaub (Tage)', hours: 22, type: 'neutral' },
]

const requestStatusConfig: Record<string, { labelKey: string; color: string }> = {
  pending: { labelKey: 'team.selfService.statusPending', color: 'bg-warning-light text-warning' },
  approved: { labelKey: 'team.selfService.statusApproved', color: 'bg-success-light text-success' },
  rejected: { labelKey: 'team.selfService.statusRejected', color: 'bg-error-light text-error' },
}

const formatEUR = (amount: number) =>
  new Intl.NumberFormat('de-DE', { style: 'currency', currency: 'EUR', minimumFractionDigits: 0, maximumFractionDigits: 0 }).format(amount)

function getInitials(name: string): string {
  const parts = (name ?? '').trim().split(/\s+/).filter(Boolean)
  if (parts.length >= 2) return `${parts[0][0]}${parts[parts.length - 1][0]}`.toUpperCase()
  return (name ?? '?').substring(0, 2).toUpperCase()
}

// ============================================================
// Component
// ============================================================

export function SelfServiceView() {
  const { t } = useTranslation()
  const [tab, setTab] = useState<SelfServiceTab>('profil')
  const [dialogOpen, setDialogOpen] = useState(false)
  const [formType, setFormType] = useState('')
  const [formStart, setFormStart] = useState('')
  const [formEnd, setFormEnd] = useState('')
  const [formReason, setFormReason] = useState('')

  const { data: self } = useSelfProfile()
  const { data: balance } = useLeaveBalance()
  const { data: leaveTypes } = useLeaveTypes()
  const { data: requestsData } = useLeaveRequests(self?.userId ? { user_id: self.userId } : undefined)
  const createRequest = useCreateLeaveRequest()

  const profile = useMemo(() => {
    const contractKey = {
      full_time: 'team.contractType.fullTime', part_time: 'team.contractType.partTime',
      mini_job: 'team.contractType.miniJob', intern: 'team.contractType.internship',
      temporary: 'team.contractType.temporary',
    }[self?.contractType ?? 'full_time'] ?? 'team.contractType.fullTime'
    return {
      name: self?.userName ?? '—',
      initials: self?.initials ?? getInitials(self?.userName ?? ''),
      role: self?.positionTitle ?? '—',
      department: self?.department ?? '—',
      email: self?.userEmail ?? '—',
      phone: self?.phone ?? '—',
      location: self?.location ?? self?.addressCity ?? '—',
      joinDate: self?.startDate ?? '',
      manager: self?.managerName ?? '—',
      contractLabel: `${t(contractKey)}${self?.workload ? ` (${self.workload}%)` : ''}`,
      employeeId: self?.id ?? '—',
    }
  }, [self, t])

  // Build leave balance rows from the API shape (entitlement/taken/remaining/sick_days/special_leave).
  const balanceRows = useMemo(() => {
    const b = (balance ?? {}) as Record<string, number>
    const vacationTotal = b.entitlement ?? 30
    const vacationTaken = b.taken ?? 0
    const vacationRemaining = b.remaining ?? Math.max(0, vacationTotal - vacationTaken)
    return [
      { type: t('team.selfService.balanceVacation'), total: vacationTotal, used: vacationTaken, remaining: vacationRemaining, color: 'text-primary', isSick: false },
      { type: t('team.selfService.balanceSick'), total: 0, used: b.sick_days ?? 0, remaining: 0, color: 'text-error', isSick: true },
      { type: t('team.selfService.balanceSpecial'), total: 5, used: b.special_leave ?? 0, remaining: Math.max(0, 5 - (b.special_leave ?? 0)), color: 'text-info', isSick: false },
    ]
  }, [balance, t])

  const requests = useMemo(() => {
    const raw = (requestsData?.requests ?? []) as Array<Record<string, unknown>>
    return raw.map((r) => ({
      id: String(r.id),
      type: String(r.type ?? r.leaveTypeName ?? '—'),
      startDate: String(r.start_date ?? r.startDate ?? ''),
      endDate: String(r.end_date ?? r.endDate ?? ''),
      days: Number(r.days ?? r.totalDays ?? 0),
      status: String(r.status ?? 'pending'),
      reason: String(r.note ?? r.reason ?? ''),
    }))
  }, [requestsData])

  const openRequestDialog = (typeId = '') => {
    setFormType(typeId)
    setFormStart('')
    setFormEnd('')
    setFormReason('')
    setDialogOpen(true)
  }

  const submitRequest = () => {
    if (!formType || !formStart || !formEnd) {
      toast.error(t('team.selfService.errorRequired'))
      return
    }
    if (formEnd < formStart) {
      toast.error(t('team.selfService.errorEndAfterStart'))
      return
    }
    createRequest.mutate(
      { leaveTypeId: formType, startDate: formStart, endDate: formEnd, reason: formReason.trim() || undefined },
      { onSuccess: () => setDialogOpen(false) },
    )
  }

  // Download a salary statement as a real (demo) text blob.
  const downloadStatement = (ss: SalaryStatement) => {
    const content =
      `${t('team.selfService.salaryStatements')}\n${ss.label}\n\n` +
      `${t('team.integration.gross')}: ${formatEUR(ss.gross)}\n${t('team.selfService.net')}: ${formatEUR(ss.net)}\n\n` +
      `${profile.name} · ${profile.employeeId}\n`
    const blob = new Blob([content], { type: 'text/plain;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `Gehaltsabrechnung-${ss.month}.txt`
    a.click()
    URL.revokeObjectURL(url)
    toast.success(t('team.selfService.downloadStarted', { label: ss.label }))
  }

  const quickActions = [
    { labelKey: 'team.selfService.requestLeave', icon: Calendar, typeId: leaveTypes?.find((l) => l.name === 'Urlaub')?.id ?? '' },
    { labelKey: 'team.selfService.reportHomeOffice', icon: MapPin, typeId: leaveTypes?.find((l) => l.name === 'Homeoffice')?.id ?? '' },
    { labelKey: 'team.selfService.reportSick', icon: AlertCircle, typeId: leaveTypes?.find((l) => l.name === 'Krankheit')?.id ?? '' },
    { labelKey: 'team.selfService.reportOvertime', icon: Clock, typeId: leaveTypes?.find((l) => l.name === 'Sonderurlaub')?.id ?? '' },
  ]

  return (
    <div className="space-y-4">
      {/* Banner */}
      <div className="rounded-lg border border-primary/20 bg-primary-light/30 p-4">
        <div className="flex items-center gap-4">
          <div className="flex h-14 w-14 items-center justify-center rounded-full bg-primary text-lg font-bold text-primary-foreground">
            {profile.initials}
          </div>
          <div>
            <h2 className="text-lg font-semibold text-foreground">{profile.name}</h2>
            <p className="text-sm text-muted-foreground">{profile.role} · {profile.department}</p>
            <p className="text-xs text-muted-foreground">{t('team.selfService.employeeId')}: {profile.employeeId}</p>
          </div>
          <div className="ml-auto flex items-center gap-3">
            {balanceRows.filter((b) => !b.isSick).slice(0, 2).map((lb) => (
              <div key={lb.type} className="text-center">
                <p className={`text-lg font-bold ${lb.color}`}>{lb.remaining}</p>
                <p className="text-[10px] text-muted-foreground">{lb.type} {t('team.selfService.remainingShort')}</p>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-4 border-b border-border">
        {([
          { key: 'profil' as const, label: t('team.selfService.myProfile'), icon: User },
          { key: 'antraege' as const, label: t('team.selfService.myRequests'), icon: Send },
          { key: 'dokumente' as const, label: t('team.selfService.salaryStatements'), icon: FileText },
          { key: 'zeitkonto' as const, label: t('team.selfService.timeAccount'), icon: Clock },
        ]).map((tabItem) => {
          const Icon = tabItem.icon
          return (
            <button
              key={tabItem.key}
              onClick={() => setTab(tabItem.key)}
              className={`flex items-center gap-1.5 border-b-2 px-1 pb-2 text-sm transition-colors ${
                tab === tabItem.key ? 'border-primary text-primary font-medium' : 'border-transparent text-muted-foreground hover:text-foreground'
              }`}
            >
              <Icon className="h-3.5 w-3.5" />
              {tabItem.label}
            </button>
          )
        })}
      </div>

      {/* Profile Tab */}
      {tab === 'profil' && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="rounded-lg border border-border bg-card p-5">
            <h3 className="text-sm font-medium text-foreground mb-4">{t('team.selfService.personalData')}</h3>
            <div className="space-y-3">
              {[
                { icon: User, label: t('team.selfService.name'), value: profile.name },
                { icon: Mail, label: t('team.selfService.emailLabel'), value: profile.email },
                { icon: Phone, label: t('team.selfService.phoneLabel'), value: profile.phone },
                { icon: MapPin, label: t('team.member.location'), value: profile.location },
                { icon: Building2, label: t('team.member.department'), value: profile.department },
                { icon: Briefcase, label: t('team.member.contractType'), value: profile.contractLabel },
                { icon: Calendar, label: t('team.selfService.joinDate'), value: profile.joinDate ? formatDate(profile.joinDate) : '—' },
                { icon: User, label: t('team.selfService.manager'), value: profile.manager },
              ].map((item) => {
                const Icon = item.icon
                return (
                  <div key={item.label} className="flex items-center gap-3">
                    <Icon className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                    <span className="text-xs text-muted-foreground w-28">{item.label}</span>
                    <span className="text-sm text-foreground">{item.value}</span>
                  </div>
                )
              })}
            </div>
            <button
              onClick={() => toast.info(t('team.selfService.changeRequestInfo'))}
              className="mt-4 flex items-center gap-1.5 text-xs text-primary hover:underline"
            >
              {t('team.selfService.requestChange')}
              <ChevronRight className="h-3 w-3" />
            </button>
          </div>

          <div className="space-y-4">
            <div className="rounded-lg border border-border bg-card p-5">
              <h3 className="text-sm font-medium text-foreground mb-4">{t('team.selfService.leaveAccount')}</h3>
              <div className="space-y-3">
                {balanceRows.map((lb) => (
                  <div key={lb.type}>
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-xs text-muted-foreground">{lb.type}</span>
                      <span className={`text-xs font-medium ${lb.color}`}>
                        {lb.remaining}{lb.total > 0 ? `/${lb.total}` : ''} {lb.isSick ? t('team.selfService.daysTaken') : t('team.selfService.daysRemaining')}
                      </span>
                    </div>
                    {lb.total > 0 && (
                      <div className="h-1.5 rounded-full bg-secondary">
                        <div
                          className="h-full rounded-full bg-primary transition-all"
                          style={{ width: `${Math.min(100, (lb.used / lb.total) * 100)}%` }}
                        />
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>

            <div className="rounded-lg border border-border bg-card p-5">
              <h3 className="text-sm font-medium text-foreground mb-3">{t('team.selfService.quickActions')}</h3>
              <div className="grid grid-cols-2 gap-2">
                {quickActions.map((action) => {
                  const Icon = action.icon
                  return (
                    <button
                      key={action.labelKey}
                      onClick={() => openRequestDialog(action.typeId)}
                      className="flex items-center gap-2 rounded-lg border border-border p-3 text-sm text-foreground hover:bg-secondary transition-colors text-left"
                    >
                      <Icon className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                      <span className="text-xs">{t(action.labelKey)}</span>
                    </button>
                  )
                })}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Requests Tab */}
      {tab === 'antraege' && (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <p className="text-sm text-muted-foreground">{t('team.selfService.requestsCount', { count: requests.length })}</p>
            <button
              onClick={() => openRequestDialog()}
              className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
            >
              <Send className="h-3.5 w-3.5" />
              {t('team.selfService.newRequest')}
            </button>
          </div>
          {requests.length === 0 ? (
            <div className="rounded-lg border border-dashed border-border py-12 text-center text-muted-foreground">
              <Calendar className="h-10 w-10 mx-auto mb-3 opacity-50" />
              <p className="text-sm">{t('team.selfService.noRequests')}</p>
            </div>
          ) : (
            <div className="space-y-3">
              {requests.map((req) => {
                const st = requestStatusConfig[req.status] ?? requestStatusConfig.pending
                return (
                  <div key={req.id} className="rounded-lg border border-border bg-card p-4">
                    <div className="flex items-start justify-between">
                      <div>
                        <div className="flex items-center gap-2 mb-1">
                          <span className="text-sm font-medium text-foreground">{req.type}</span>
                          <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${st.color}`}>{t(st.labelKey)}</span>
                        </div>
                        <div className="flex items-center gap-2 text-xs text-muted-foreground">
                          <Calendar className="h-3 w-3" />
                          <span>
                            {formatDate(req.startDate)}
                            {req.startDate !== req.endDate && ` – ${formatDate(req.endDate)}`}
                            {' '}({req.days} {req.days === 1 ? t('team.approval.day') : t('team.approval.days')})
                          </span>
                        </div>
                        {req.reason && <p className="text-xs text-muted-foreground mt-1">{req.reason}</p>}
                      </div>
                      {req.status === 'approved' && <CheckCircle2 className="h-5 w-5 text-success flex-shrink-0" />}
                      {req.status === 'pending' && <Clock className="h-5 w-5 text-warning flex-shrink-0" />}
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </div>
      )}

      {/* Documents Tab */}
      {tab === 'dokumente' && (
        <div className="space-y-3">
          <p className="text-sm text-muted-foreground">{t('team.selfService.recentStatements')}</p>
          <div className="rounded-lg border border-border overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-secondary/50">
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">{t('team.selfService.month')}</th>
                    <th className="px-4 py-3 text-right font-medium text-muted-foreground">{t('team.integration.gross')}</th>
                    <th className="px-4 py-3 text-right font-medium text-muted-foreground">{t('team.selfService.net')}</th>
                    <th className="px-4 py-3 text-right font-medium text-muted-foreground">{t('common.actions')}</th>
                  </tr>
                </thead>
                <tbody>
                  {SALARY_STATEMENTS.map((ss) => (
                    <tr key={ss.id} className="border-b border-border-muted hover:bg-secondary/20">
                      <td className="px-4 py-3 font-medium text-foreground">{ss.label}</td>
                      <td className="px-4 py-3 text-right text-muted-foreground tabular-nums">{formatEUR(ss.gross)}</td>
                      <td className="px-4 py-3 text-right font-medium text-foreground tabular-nums">{formatEUR(ss.net)}</td>
                      <td className="px-4 py-3 text-right">
                        <button
                          onClick={() => downloadStatement(ss)}
                          className="inline-flex items-center gap-1 rounded-md border border-border px-2 py-1 text-xs text-foreground hover:bg-secondary transition-colors"
                        >
                          <Download className="h-3 w-3" />
                          {t('team.selfService.download')}
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}

      {/* Time Account Tab */}
      {tab === 'zeitkonto' && (
        <div className="space-y-4">
          <dl className="mb-4 flex flex-wrap items-end gap-x-10 gap-y-3 border-b border-border pb-4">
            {TIME_ACCOUNTS.map((ta) => (
              <InlineStat
                key={ta.label}
                label={ta.label}
                value={`${ta.type === 'positive' ? '+' : ''}${ta.hours}${ta.label.includes('Tage') ? '' : 'h'}`}
                accent={ta.type === 'positive' ? 'success' : ta.type === 'negative' ? 'error' : 'default'}
              />
            ))}
          </dl>

          <div className="rounded-lg border border-border bg-card p-5">
            <h3 className="text-sm font-medium text-foreground mb-3">{t('team.selfService.workTimeAccount')}</h3>
            <div className="space-y-2">
              {[
                { day: 'Mo, 17.02.', soll: '8:00', ist: '8:30', diff: '+0:30' },
                { day: 'Di, 18.02.', soll: '8:00', ist: '8:45', diff: '+0:45' },
                { day: 'Mi, 19.02.', soll: '8:00', ist: '7:30', diff: '-0:30' },
                { day: 'Do, 20.02.', soll: '8:00', ist: '9:00', diff: '+1:00' },
                { day: 'Fr, 21.02.', soll: '8:00', ist: '7:45', diff: '-0:15' },
              ].map((entry) => (
                <div key={entry.day} className="flex items-center justify-between py-2 border-b border-border-muted last:border-0">
                  <span className="text-sm text-foreground w-24">{entry.day}</span>
                  <span className="text-xs text-muted-foreground">{t('team.selfService.target')}: {entry.soll}</span>
                  <span className="text-xs text-muted-foreground">{t('team.selfService.actual')}: {entry.ist}</span>
                  <span className={`text-xs font-medium tabular-nums ${
                    entry.diff.startsWith('+') ? 'text-success' : 'text-error'
                  }`}>
                    {entry.diff}
                  </span>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* New Request Dialog */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('team.selfService.requestDialogTitle')}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">{t('team.selfService.leaveType')} *</label>
              <Select value={formType} onValueChange={setFormType}>
                <SelectTrigger>
                  <SelectValue placeholder={t('team.selfService.selectType')} />
                </SelectTrigger>
                <SelectContent>
                  {(leaveTypes ?? []).map((lt) => (
                    <SelectItem key={lt.id} value={lt.id}>
                      <span className="flex items-center gap-2">
                        <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: lt.color }} />
                        {lt.name}
                      </span>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">{t('team.selfService.from')} *</label>
                <Input type="date" value={formStart} onChange={(e) => setFormStart(e.target.value)} />
              </div>
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">{t('team.selfService.to')} *</label>
                <Input type="date" value={formEnd} onChange={(e) => setFormEnd(e.target.value)} />
              </div>
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">{t('team.selfService.reason')} ({t('common.optional')})</label>
              <textarea
                value={formReason}
                onChange={(e) => setFormReason(e.target.value)}
                rows={3}
                placeholder={t('team.selfService.reasonPlaceholder')}
                className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>{t('common.cancel')}</Button>
            <Button onClick={submitRequest} disabled={createRequest.isPending}>
              {createRequest.isPending ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : null}
              {t('team.selfService.submitRequest')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

/**
 * Offboard-Dialog (R-4 §0.5) — zweistufiger Dialog für Mitarbeiter-Austritt.
 * Schritt 1: Formular (letzter Arbeitstag, Austrittsdatum, Austrittsart, Grund, Nachbesetzung).
 * Schritt 2: Konsequenzliste + Abhängigkeits-Check (Vorgesetzte/Genehmiger) + Pflicht-Successor.
 *
 * Einstieg NUR im Profil-Header (ItemActions), KEIN Button in Listen.
 * Gated: team:employee:offboard (admin + hr_admin).
 */
import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
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
import { Switch } from '@/components/ui/switch'
import { AlertTriangle, ChevronRight, LogOut, Users } from 'lucide-react'
import { useEmployees } from '@/api/hooks/hr-hooks'
import type { EmployeeProfile } from '@/api/hr-types'
import { formatDate } from '@/lib/format'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type ExitType =
  | 'resignation'
  | 'termination'
  | 'fixed_term_expired'
  | 'mutual_termination'
  | 'retirement'

const EXIT_TYPE_KEYS: Record<ExitType, string> = {
  resignation: 'team.offboard.exitType.resignation',
  termination: 'team.offboard.exitType.termination',
  fixed_term_expired: 'team.offboard.exitType.fixedTermExpired',
  mutual_termination: 'team.offboard.exitType.mutualTermination',
  retirement: 'team.offboard.exitType.retirement',
}

interface OffboardFormData {
  lastWorkDay: string
  exitDate: string
  exitType: ExitType
  reason: string
  backfill: boolean
}

interface OffboardEmployeeDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  employee: EmployeeProfile
  onConfirm: (data: OffboardFormData & { successorUserId?: string }) => void
  isPending?: boolean
}

// ---------------------------------------------------------------------------
// Main dialog
// ---------------------------------------------------------------------------

export function OffboardEmployeeDialog({
  open,
  onOpenChange,
  employee,
  onConfirm,
  isPending = false,
}: OffboardEmployeeDialogProps) {
  const { t } = useTranslation()
  const [step, setStep] = useState<1 | 2>(1)

  const today = new Date().toISOString().split('T')[0]
  const [form, setForm] = useState<OffboardFormData>({
    lastWorkDay: today,
    exitDate: today,
    exitType: 'resignation',
    reason: '',
    backfill: false,
  })
  const [successorUserId, setSuccessorUserId] = useState('')

  const { data: employeesData } = useEmployees()
  const allEmployees = employeesData?.employees ?? []

  // Abhängigkeits-Check: hat jemand diese Person als managerUserId?
  const dependents = useMemo(
    () =>
      allEmployees.filter(
        (e) =>
          e.userId !== employee.userId &&
          e.status !== 'inactive' &&
          (e.managerUserId === employee.userId ||
            // also check by employee.id (some seeds use id-based managerId)
            e.managerUserId === employee.id),
      ),
    [allEmployees, employee.userId, employee.id],
  )

  const hasDependents = dependents.length > 0
  const successorRequired = hasDependents
  const canConfirm = !successorRequired || !!successorUserId

  // Candidates for successor: active employees except the departing one
  const successorCandidates = useMemo(
    () =>
      allEmployees.filter(
        (e) => e.userId !== employee.userId && e.status !== 'inactive',
      ),
    [allEmployees, employee.userId],
  )

  function patch(p: Partial<OffboardFormData>) {
    setForm((prev) => {
      const next = { ...prev, ...p }
      // Sync exitDate to lastWorkDay when not manually overridden
      if (p.lastWorkDay && next.exitDate === prev.lastWorkDay) {
        next.exitDate = p.lastWorkDay
      }
      return next
    })
  }

  function handleConfirm() {
    onConfirm({
      ...form,
      successorUserId: successorUserId || undefined,
    })
  }

  function handleClose() {
    if (!isPending) {
      onOpenChange(false)
      // Reset state after close
      setTimeout(() => {
        setStep(1)
        setSuccessorUserId('')
        setForm({ lastWorkDay: today, exitDate: today, exitType: 'resignation', reason: '', backfill: false })
      }, 200)
    }
  }

  const employeeName = employee.userName ?? t('team.member.employee')

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-lg max-h-[85vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-destructive">
            <LogOut className="h-4 w-4" />
            {t('team.offboard.title', { name: employeeName })}
          </DialogTitle>
        </DialogHeader>

        {/* Step indicator */}
        <div className="flex items-center gap-2 px-0 pb-2">
          <StepDot active={step === 1} done={step === 2} label="1" />
          <div className="h-px flex-1 bg-border" />
          <StepDot active={step === 2} done={false} label="2" />
        </div>

        <div className="flex-1 overflow-y-auto">
          {step === 1 ? (
            <Step1Form form={form} patch={patch} />
          ) : (
            <Step2Confirm
              form={form}
              employee={employee}
              dependents={dependents}
              successorCandidates={successorCandidates}
              successorUserId={successorUserId}
              onSuccessorChange={setSuccessorUserId}
            />
          )}
        </div>

        <DialogFooter className="gap-2">
          {step === 1 ? (
            <>
              <Button variant="outline" onClick={handleClose} disabled={isPending}>
                {t('common.cancel')}
              </Button>
              <Button
                onClick={() => setStep(2)}
                variant="destructive"
                className="gap-1.5"
              >
                {t('common.next')}
                <ChevronRight className="h-3.5 w-3.5" />
              </Button>
            </>
          ) : (
            <>
              <Button variant="outline" onClick={() => setStep(1)} disabled={isPending}>
                {t('common.back')}
              </Button>
              <Button
                onClick={handleConfirm}
                disabled={!canConfirm || isPending}
                variant="destructive"
                className="gap-1.5"
              >
                <LogOut className="h-3.5 w-3.5" />
                {isPending ? t('team.offboard.confirming') : t('team.offboard.confirm')}
              </Button>
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// Step 1: Formular
// ---------------------------------------------------------------------------

function Step1Form({
  form,
  patch,
}: {
  form: OffboardFormData
  patch: (p: Partial<OffboardFormData>) => void
}) {
  const { t } = useTranslation()

  return (
    <div className="space-y-4 py-2">
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1.5">
          <Label className="text-xs">{t('team.offboard.lastWorkDay')}</Label>
          <Input
            type="date"
            value={form.lastWorkDay}
            onChange={(e) => patch({ lastWorkDay: e.target.value })}
            className="h-8 text-xs"
          />
        </div>
        <div className="space-y-1.5">
          <Label className="text-xs">{t('team.offboard.exitDate')}</Label>
          <Input
            type="date"
            value={form.exitDate}
            min={form.lastWorkDay}
            onChange={(e) => patch({ exitDate: e.target.value })}
            className="h-8 text-xs"
          />
        </div>
      </div>

      <div className="space-y-1.5">
        <Label className="text-xs">{t('team.offboard.exitType')}</Label>
        <Select value={form.exitType} onValueChange={(v) => patch({ exitType: v as ExitType })}>
          <SelectTrigger className="h-8 text-xs">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {(Object.entries(EXIT_TYPE_KEYS) as [ExitType, string][]).map(([k, labelKey]) => (
              <SelectItem key={k} value={k} className="text-xs">
                {t(labelKey)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="space-y-1.5">
        <Label className="text-xs">
          {t('team.offboard.reason')}
          <span className="ml-1 text-muted-foreground">({t('common.optional')})</span>
        </Label>
        <textarea
          value={form.reason}
          onChange={(e) => patch({ reason: e.target.value })}
          placeholder={t('team.offboard.reasonPlaceholder')}
          rows={3}
          className="w-full rounded-md border border-border bg-input-background px-3 py-2 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring resize-none"
        />
      </div>

      <div className="flex items-center justify-between rounded-lg border border-border p-3">
        <div>
          <p className="text-xs font-medium text-foreground">{t('team.offboard.backfill')}</p>
          <p className="text-[10px] text-muted-foreground">{t('team.offboard.backfillHint')}</p>
        </div>
        <Switch
          checked={form.backfill}
          onCheckedChange={(v) => patch({ backfill: v })}
        />
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Step 2: Bestätigung + Konsequenzliste + Abhängigkeits-Check
// ---------------------------------------------------------------------------

function Step2Confirm({
  form,
  employee,
  dependents,
  successorCandidates,
  successorUserId,
  onSuccessorChange,
}: {
  form: OffboardFormData
  employee: EmployeeProfile
  dependents: EmployeeProfile[]
  successorCandidates: EmployeeProfile[]
  successorUserId: string
  onSuccessorChange: (v: string) => void
}) {
  const { t } = useTranslation()

  const consequences = [
    t('team.offboard.consequence.loginLocked', { date: formatDate(form.exitDate) }),
    t('team.offboard.consequence.seatFreed'),
    t('team.offboard.consequence.rolesRevoked'),
  ]

  return (
    <div className="space-y-4 py-2">
      {/* Summary */}
      <div className="rounded-lg border border-border bg-secondary/30 p-3 space-y-1.5 text-xs">
        <SummaryRow label={t('team.offboard.lastWorkDay')} value={formatDate(form.lastWorkDay)} />
        <SummaryRow label={t('team.offboard.exitDate')} value={formatDate(form.exitDate)} />
        <SummaryRow label={t('team.offboard.exitType')} value={t(EXIT_TYPE_KEYS[form.exitType])} />
        {form.backfill && <SummaryRow label={t('team.offboard.backfill')} value={t('common.yes')} />}
      </div>

      {/* Consequences */}
      <div className="space-y-1.5">
        <p className="text-xs font-semibold text-foreground">{t('team.offboard.consequences')}</p>
        <ul className="space-y-1">
          {consequences.map((c, i) => (
            <li key={i} className="flex items-start gap-2 text-xs text-muted-foreground">
              <span className="mt-0.5 h-1.5 w-1.5 shrink-0 rounded-full bg-destructive/70" />
              {c}
            </li>
          ))}
        </ul>
      </div>

      {/* Abhängigkeits-Check */}
      {dependents.length > 0 && (
        <div className="rounded-lg border border-warning/40 bg-warning/5 p-3 space-y-2">
          <div className="flex items-start gap-2">
            <AlertTriangle className="h-3.5 w-3.5 mt-0.5 shrink-0 text-warning" />
            <div className="space-y-1">
              <p className="text-xs font-medium text-warning-foreground">
                {t('team.offboard.dependentsWarning', { count: dependents.length })}
              </p>
              <p className="text-[10px] text-muted-foreground">
                {t('team.offboard.dependentsHint')}
              </p>
            </div>
          </div>

          {/* Dependent names */}
          <div className="flex flex-wrap gap-1 pl-5">
            {dependents.slice(0, 5).map((d) => (
              <span
                key={d.id}
                className="flex items-center gap-1 rounded-full bg-secondary px-2 py-0.5 text-[10px] text-foreground"
              >
                <Users className="h-2.5 w-2.5" />
                {d.userName ?? d.id}
              </span>
            ))}
            {dependents.length > 5 && (
              <span className="text-[10px] text-muted-foreground">
                +{dependents.length - 5} {t('common.more')}
              </span>
            )}
          </div>

          {/* Pflicht-Successor-Select */}
          <div className="space-y-1 pl-5">
            <Label className="text-xs font-medium text-foreground">
              {t('team.offboard.successorLabel')}
              <span className="ml-1 text-destructive">*</span>
            </Label>
            <Select value={successorUserId} onValueChange={onSuccessorChange}>
              <SelectTrigger className="h-8 text-xs">
                <SelectValue placeholder={t('team.offboard.successorPlaceholder')} />
              </SelectTrigger>
              <SelectContent>
                {successorCandidates.map((c) => (
                  <SelectItem key={c.userId ?? c.id} value={c.userId ?? c.id} className="text-xs">
                    {c.userName ?? c.id}
                    {c.positionTitle ? ` — ${c.positionTitle}` : ''}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className="text-[10px] text-muted-foreground">
              {t('team.offboard.successorHint')}
            </p>
          </div>
        </div>
      )}

      {/* Destructive confirmation notice */}
      <div className="rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-xs text-destructive">
        {t('team.offboard.destructiveHint', { name: employee.userName ?? t('team.member.employee') })}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function StepDot({ active, done, label }: { active: boolean; done: boolean; label: string }) {
  return (
    <div
      className={`flex h-6 w-6 items-center justify-center rounded-full text-xs font-medium transition-colors ${
        active
          ? 'bg-destructive text-white'
          : done
            ? 'bg-success text-white'
            : 'bg-secondary text-muted-foreground'
      }`}
    >
      {label}
    </div>
  )
}

function SummaryRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-2">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-medium text-foreground">{value}</span>
    </div>
  )
}

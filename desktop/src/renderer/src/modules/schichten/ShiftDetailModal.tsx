/**
 * ShiftDetailModal — opened when clicking an occupied cell in the week grid
 * (previously only an info toast). Shows the assignment's full context
 * (employee, template, times, surcharge, ArbZG hints) and offers the two
 * actions the market standard (Planday/Deputy) puts on a shift: remove the
 * assignment and request a swap (wires the previously unused
 * useCreateSwapRequest mutation).
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  User,
  CalendarDays,
  Clock,
  Coffee,
  Timer,
  Percent,
  PartyPopper,
  ShieldAlert,
  ArrowLeftRight,
  Trash2,
  AlertTriangle,
  XCircle,
  CheckCircle2,
} from 'lucide-react'
import { DetailModal } from '@/components/shared'
import type { ShiftAssignment, ShiftTemplate } from '@/stores/schichten'
import {
  getSurchargeLabel,
  isHoliday,
  templateNetMinutes,
  SHIFT_STYLE_MAP,
  type ArbZGViolation,
  type SchichtenEmployee,
} from './schichten-shared'

interface ShiftDetailModalProps {
  open: boolean
  onClose: () => void
  assignment: ShiftAssignment | null
  template: ShiftTemplate | null
  employeeRole: string
  /** Resolved API shift id — unassign/swap need it; actions hide when missing. */
  shiftId: string | undefined
  violations: ArbZGViolation[]
  /** Swap targets (everyone except the assigned employee). */
  swapCandidates: SchichtenEmployee[]
  /** Tenant policy: whether shift-swap requests are allowed at all. */
  swapEnabled: boolean
  onUnassign: () => void
  isUnassigning: boolean
  onCreateSwap: (swapWithEmployeeId: string, reason: string) => void
  isCreatingSwap: boolean
}

export function ShiftDetailModal({
  open,
  onClose,
  assignment,
  template,
  employeeRole,
  shiftId,
  violations,
  swapCandidates,
  swapEnabled,
  onUnassign,
  isUnassigning,
  onCreateSwap,
  isCreatingSwap,
}: ShiftDetailModalProps) {
  const { t } = useTranslation()
  const [swapOpen, setSwapOpen] = useState(false)
  const [swapWith, setSwapWith] = useState('')
  const [swapReason, setSwapReason] = useState('')

  if (!assignment || !template) return null

  const surcharge = getSurchargeLabel(assignment.templateId, assignment.date)
  const holiday = isHoliday(assignment.date)
  const netMin = templateNetMinutes(template)
  const netLabel = `${Math.floor(netMin / 60)}h${netMin % 60 > 0 ? ` ${netMin % 60}min` : ''}`
  const style = SHIFT_STYLE_MAP[assignment.templateId]
  const StyleIcon = style?.icon ?? Clock
  const isConfirmed = assignment.status === 'confirmed'
  const dateLabel = new Date(assignment.date + 'T00:00:00').toLocaleDateString('de-DE', {
    weekday: 'long', day: '2-digit', month: 'long', year: 'numeric',
  })

  const handleSwapSubmit = () => {
    if (!swapWith) return
    onCreateSwap(swapWith, swapReason)
    setSwapOpen(false)
    setSwapWith('')
    setSwapReason('')
  }

  const metaItems = [
    { icon: User, label: t('schichten.detail.mitarbeiter'), value: `${assignment.userName}${employeeRole ? ` · ${employeeRole}` : ''}` },
    { icon: CalendarDays, label: t('schichten.detail.datum'), value: dateLabel },
    { icon: Clock, label: t('schichten.detail.zeit'), value: `${template.startTime} – ${template.endTime}` },
    { icon: Coffee, label: t('schichten.detail.pause'), value: t('schichten.vorlagen.pause', { min: template.breakMinutes }) },
    { icon: Timer, label: t('schichten.detail.netto'), value: netLabel },
  ]

  return (
    <DetailModal
      open={open}
      onClose={onClose}
      title={assignment.templateName}
      subtitle={dateLabel}
      badge={
        <span
          className={`ml-1 flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${
            isConfirmed ? 'bg-success-light text-success' : 'bg-secondary text-muted-foreground'
          }`}
        >
          {isConfirmed ? <CheckCircle2 className="h-3 w-3" /> : <Clock className="h-3 w-3" />}
          {isConfirmed ? t('schichten.detail.status.veroeffentlicht') : t('schichten.detail.status.entwurf')}
        </span>
      }
      footer={
        shiftId ? (
          <div className="flex items-center justify-between gap-2">
            {swapEnabled ? (
              <button
                onClick={() => setSwapOpen((v) => !v)}
                className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
              >
                <ArrowLeftRight className="h-3.5 w-3.5" />
                {t('schichten.detail.tauschBeantragen')}
              </button>
            ) : (
              <span className="text-[11px] text-muted-foreground">{t('schichten.detail.tauschDeaktiviert')}</span>
            )}
            <button
              onClick={onUnassign}
              disabled={isUnassigning}
              className="flex items-center gap-1.5 rounded-lg border border-error/30 px-3 py-1.5 text-xs text-error hover:bg-error-light transition-colors disabled:opacity-50"
            >
              <Trash2 className="h-3.5 w-3.5" />
              {t('schichten.detail.zuweisungEntfernen')}
            </button>
          </div>
        ) : undefined
      }
    >
      <div className="space-y-5">
        {/* Shift identity */}
        <div className={`flex items-center gap-3 rounded-lg border p-3 ${style?.bg ?? 'bg-secondary'} ${style?.border ?? 'border-border'}`}>
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-card/60">
            <StyleIcon className={`h-5 w-5 ${style?.text ?? 'text-foreground'}`} />
          </div>
          <div className="min-w-0">
            <p className={`text-sm font-semibold ${style?.text ?? 'text-foreground'}`}>{assignment.templateName}</p>
            <p className="text-xs text-muted-foreground">
              {template.startTime} – {template.endTime}
            </p>
          </div>
          {surcharge && (
            <div className="ml-auto flex items-center gap-1 rounded-full bg-warning-light px-2 py-0.5">
              <Percent className="h-3 w-3 text-warning" />
              <span className="text-[10px] font-medium text-warning">{surcharge.rate} {surcharge.label}</span>
            </div>
          )}
        </div>

        {/* Meta grid */}
        <div className="grid grid-cols-2 gap-x-4 gap-y-3">
          {metaItems.map((item) => {
            const Icon = item.icon
            return (
              <div key={item.label} className="min-w-0">
                <p className="flex items-center gap-1 text-[11px] text-muted-foreground">
                  <Icon className="h-3 w-3" />
                  {item.label}
                </p>
                <p className="mt-0.5 truncate text-sm text-foreground">{item.value}</p>
              </div>
            )
          })}
          {holiday && (
            <div className="min-w-0">
              <p className="flex items-center gap-1 text-[11px] text-muted-foreground">
                <PartyPopper className="h-3 w-3" />
                {t('schichten.detail.feiertag')}
              </p>
              <p className="mt-0.5 truncate text-sm text-error">{holiday}</p>
            </div>
          )}
        </div>

        {/* ArbZG hints for this employee */}
        {violations.length > 0 && (
          <div className="rounded-lg border border-warning/30 bg-warning-light/30 p-3">
            <p className="mb-1.5 flex items-center gap-1.5 text-xs font-medium text-warning">
              <ShieldAlert className="h-3.5 w-3.5" />
              {t('schichten.detail.arbzgHinweise', { count: violations.length })}
            </p>
            <div className="space-y-1">
              {violations.map((v, i) => (
                <p key={i} className="flex items-start gap-1.5 text-xs text-muted-foreground">
                  {v.severity === 'error'
                    ? <XCircle className="mt-0.5 h-3 w-3 flex-shrink-0 text-error" />
                    : <AlertTriangle className="mt-0.5 h-3 w-3 flex-shrink-0 text-warning" />}
                  {v.message}
                </p>
              ))}
            </div>
          </div>
        )}

        {/* Swap request form (inline) */}
        {swapOpen && shiftId && (
          <div className="rounded-lg border border-border bg-card p-3">
            <p className="mb-2 flex items-center gap-1.5 text-xs font-medium text-foreground">
              <ArrowLeftRight className="h-3.5 w-3.5" />
              {t('schichten.detail.tauschTitel')}
            </p>
            <div className="space-y-3">
              <div>
                <label className="mb-1 block text-[11px] font-medium text-muted-foreground">
                  {t('schichten.detail.tauschMit')}
                </label>
                <select
                  value={swapWith}
                  onChange={(e) => setSwapWith(e.target.value)}
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                >
                  <option value="">{t('schichten.detail.tauschMitSelect')}</option>
                  {swapCandidates.map((e) => (
                    <option key={e.id} value={e.id}>{e.name} ({e.role})</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="mb-1 block text-[11px] font-medium text-muted-foreground">
                  {t('schichten.detail.tauschGrund')}
                </label>
                <textarea
                  value={swapReason}
                  onChange={(e) => setSwapReason(e.target.value)}
                  placeholder={t('schichten.detail.tauschGrundPlaceholder')}
                  rows={2}
                  className="w-full resize-none rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                />
              </div>
              <div className="flex justify-end gap-2">
                <button
                  onClick={() => setSwapOpen(false)}
                  className="rounded-lg border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
                >
                  {t('common.cancel')}
                </button>
                <button
                  onClick={handleSwapSubmit}
                  disabled={!swapWith || isCreatingSwap}
                  className="rounded-lg bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
                >
                  {t('schichten.detail.tauschSenden')}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </DetailModal>
  )
}

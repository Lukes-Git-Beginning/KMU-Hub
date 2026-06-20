import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { AlertTriangle, CalendarClock } from 'lucide-react'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import type { ReportDocument, ReportFormat } from '@/api/berichte-types'
import { useCreateSchedule, useSchedules, useUpdateSchedule } from '@/api/hooks/useBerichte'
import { useBerichtePrefsStore } from '@/stores/berichtePrefs'
import { CURRENT_USER } from '@/mocks/data/shared-ids'
import {
  DEFAULT_RHYTHM,
  WEEKDAYS,
  type Rhythm,
  type RhythmKind,
  cronToRhythm,
  pad2,
  rhythmToCron,
} from '../schedule-utils'

interface ScheduleReportModalProps {
  doc: ReportDocument
  open: boolean
  onClose: () => void
}

const RHYTHM_KINDS: RhythmKind[] = ['daily', 'weekly', 'monthly', 'quarterly']
const FORMATS: ReportFormat[] = ['pdf', 'xlsx', 'csv']

/**
 * Per-document scheduling modal (R-4). Couples a ReportSchedule to the report
 * document via `definition_id = doc.id`, and offers a friendly rhythm picker
 * (daily/weekly/monthly/quarterly) instead of a raw cron field.
 *
 * Scheduling is only allowed once the document is released — the editor button
 * guards this, but the modal repeats the guard defensively.
 */
export function ScheduleReportModal({ doc, open, onClose }: ScheduleReportModalProps) {
  const { t } = useTranslation()
  const schedulesQuery = useSchedules({ definition_id: doc.id })
  const createMutation = useCreateSchedule()
  const updateMutation = useUpdateSchedule()
  const defaultFormat = useBerichtePrefsStore((s) => s.defaultFormat)

  const existing = schedulesQuery.data?.schedules.find((s) => s.definition_id === doc.id) ?? null
  const isReleased = doc.status === 'released'

  const [rhythm, setRhythm] = useState<Rhythm>(DEFAULT_RHYTHM)
  const [format, setFormat] = useState<ReportFormat>(defaultFormat)
  const [active, setActive] = useState(true)
  const [recipients, setRecipients] = useState<string[]>([CURRENT_USER.email])

  // Hydrate the form from an existing schedule (or reset to defaults) whenever
  // the modal opens for a different document / schedule.
  useEffect(() => {
    if (!open) return
    if (existing) {
      setRhythm(cronToRhythm(existing.cron_expression))
      setFormat(existing.format)
      setActive(existing.active)
      setRecipients(existing.recipients.length ? existing.recipients : [CURRENT_USER.email])
    } else {
      setRhythm(DEFAULT_RHYTHM)
      setFormat(defaultFormat)
      setActive(true)
      setRecipients([CURRENT_USER.email])
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, existing?.id])

  const timeValue = `${pad2(rhythm.hour)}:${pad2(rhythm.minute)}`
  const setTime = (value: string) => {
    const [h, m] = value.split(':').map((n) => parseInt(n, 10))
    setRhythm((r) => ({ ...r, hour: Number.isNaN(h) ? r.hour : h, minute: Number.isNaN(m) ? r.minute : m }))
  }

  const handleSave = () => {
    if (!isReleased) return
    if (recipients.length === 0) {
      toast.error(t('berichte.docs.schedule.errorRecipients'))
      return
    }
    const cron = rhythmToCron(rhythm)
    if (existing) {
      updateMutation.mutate(
        { id: existing.id, cron_expression: cron, recipients, format, active },
        {
          onSuccess: () => {
            toast.success(t('berichte.docs.schedule.saved'))
            onClose()
          },
          onError: (err) => toast.error((err as Error).message),
        },
      )
    } else {
      createMutation.mutate(
        {
          definition_id: doc.id,
          name: doc.title,
          cron_expression: cron,
          recipients,
          format,
          active,
        },
        {
          onSuccess: () => {
            toast.success(t('berichte.docs.schedule.saved'))
            onClose()
          },
          onError: (err) => toast.error((err as Error).message),
        },
      )
    }
  }

  const saving = createMutation.isPending || updateMutation.isPending

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-w-lg gap-0 p-0">
        <DialogHeader className="border-b border-border px-6 py-4">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary-light">
              <CalendarClock className="h-4 w-4 text-primary" />
            </div>
            <div className="min-w-0">
              <DialogTitle className="text-sm">{t('berichte.docs.schedule.title')}</DialogTitle>
              <DialogDescription className="truncate">{doc.title}</DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <div className="max-h-[70vh] space-y-5 overflow-y-auto px-6 py-5">
          {!isReleased && (
            <div className="flex items-start gap-2.5 rounded-lg border border-warning/25 bg-warning-light px-3.5 py-3 text-xs text-foreground">
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-warning" aria-hidden="true" />
              <p>{t('berichte.docs.schedule.guardBanner')}</p>
            </div>
          )}

          {/* Rhythm picker */}
          <div>
            <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
              {t('berichte.docs.schedule.rhythm')}
            </label>
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
              {RHYTHM_KINDS.map((kind) => (
                <button
                  key={kind}
                  type="button"
                  disabled={!isReleased}
                  onClick={() => setRhythm((r) => ({ ...r, kind }))}
                  className={`rounded-lg border px-3 py-2 text-xs font-medium transition-colors disabled:opacity-50 ${
                    rhythm.kind === kind
                      ? 'border-primary bg-primary-light text-primary'
                      : 'border-border text-muted-foreground hover:bg-secondary'
                  }`}
                >
                  {t(`berichte.docs.schedule.kind.${kind}`)}
                </button>
              ))}
            </div>
          </div>

          {/* Rhythm detail + time */}
          <div className="grid grid-cols-2 gap-3">
            {rhythm.kind === 'weekly' && (
              <div>
                <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
                  {t('berichte.docs.schedule.weekday')}
                </label>
                <select
                  value={rhythm.weekday}
                  disabled={!isReleased}
                  onChange={(e) => setRhythm((r) => ({ ...r, weekday: e.target.value }))}
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring disabled:opacity-50"
                >
                  {WEEKDAYS.map((d) => (
                    <option key={d} value={d}>
                      {t(`berichte.docs.schedule.dow.${d}`)}
                    </option>
                  ))}
                </select>
              </div>
            )}
            {(rhythm.kind === 'monthly' || rhythm.kind === 'quarterly') && (
              <div>
                <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
                  {t('berichte.docs.schedule.dayOfMonth')}
                </label>
                <select
                  value={rhythm.day}
                  disabled={!isReleased}
                  onChange={(e) => setRhythm((r) => ({ ...r, day: parseInt(e.target.value, 10) }))}
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring disabled:opacity-50"
                >
                  {Array.from({ length: 28 }, (_, i) => i + 1).map((d) => (
                    <option key={d} value={d}>
                      {t('berichte.docs.schedule.dayN', { day: d })}
                    </option>
                  ))}
                </select>
              </div>
            )}
            <div>
              <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
                {t('berichte.docs.schedule.time')}
              </label>
              <input
                type="time"
                value={timeValue}
                disabled={!isReleased}
                onChange={(e) => setTime(e.target.value)}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring disabled:opacity-50"
              />
            </div>
          </div>

          {/* Format */}
          <div>
            <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
              {t('berichte.docs.schedule.format')}
            </label>
            <div className="flex gap-2">
              {FORMATS.map((f) => (
                <button
                  key={f}
                  type="button"
                  disabled={!isReleased}
                  onClick={() => setFormat(f)}
                  className={`flex-1 rounded-lg border px-3 py-2 text-sm uppercase transition-colors disabled:opacity-50 ${
                    format === f
                      ? 'border-primary bg-primary-light text-primary'
                      : 'border-border text-muted-foreground hover:bg-secondary'
                  }`}
                >
                  {f}
                </button>
              ))}
            </div>
          </div>

          {/* Active toggle */}
          <div className="flex items-center justify-between rounded-lg border border-border bg-card px-4 py-3">
            <div>
              <p className="text-sm font-medium text-foreground">
                {t('berichte.docs.schedule.active')}
              </p>
              <p className="text-xs text-muted-foreground">
                {t('berichte.docs.schedule.activeHint')}
              </p>
            </div>
            <button
              type="button"
              disabled={!isReleased}
              onClick={() => setActive((a) => !a)}
              aria-label={t('berichte.docs.schedule.active')}
              aria-pressed={active}
              className={`flex h-6 w-10 shrink-0 items-center rounded-full p-0.5 transition-colors disabled:opacity-50 ${
                active ? 'bg-primary' : 'bg-secondary'
              }`}
            >
              <div
                className={`h-5 w-5 rounded-full bg-white shadow transition-transform ${
                  active ? 'translate-x-4' : 'translate-x-0'
                }`}
              />
            </button>
          </div>
        </div>

        <DialogFooter className="border-t border-border px-6 py-4">
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground transition-colors hover:bg-secondary"
          >
            {t('berichte.docs.schedule.cancel')}
          </button>
          <button
            type="button"
            onClick={handleSave}
            disabled={!isReleased || saving}
            className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground transition-colors hover:bg-button-primary-hover disabled:opacity-60"
          >
            <CalendarClock className="h-4 w-4" />
            {t('berichte.docs.schedule.save')}
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

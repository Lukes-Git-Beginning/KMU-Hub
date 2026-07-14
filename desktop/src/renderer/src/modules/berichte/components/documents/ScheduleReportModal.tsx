import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { AlertTriangle, CalendarClock, Clock, History, Mail, Search, Send, X } from 'lucide-react'
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
import {
  useCreateSchedule,
  useRunScheduleNow,
  useSchedules,
  useUpdateSchedule,
} from '@/api/hooks/useBerichte'
import { useBerichtePrefsStore } from '@/stores/berichtePrefs'
import { useBerichteTenantStore } from '@/stores/berichteTenant'
import { CURRENT_USER } from '@/mocks/data/shared-ids'
import { EMPLOYEES } from '@/mocks/mock-db'
import { formatDateTime } from '@/lib/format'
import {
  DEFAULT_RHYTHM,
  EMAIL_RE,
  WEEKDAYS,
  buildRunHistory,
  computeNextRun,
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

/** Internal users available as recipients (demo: from the mock employee DB). */
interface InternalUser {
  name: string
  email: string
  initials: string
  jobTitle: string
}
const INTERNAL_USERS: InternalUser[] = EMPLOYEES.map((e) => ({
  name: `${e.firstName} ${e.lastName}`,
  email: e.email,
  initials: e.initials,
  jobTitle: e.jobTitle,
}))
const INTERNAL_BY_EMAIL = new Map(INTERNAL_USERS.map((u) => [u.email, u]))

const RUN_STATUS_STYLES: Record<string, string> = {
  success: 'bg-success-light text-success',
  failed: 'bg-error-light text-error',
  skipped: 'bg-warning-light text-warning',
}

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
  const runNowMutation = useRunScheduleNow()
  const defaultFormat = useBerichtePrefsStore((s) => s.defaultFormat)
  const allowedFormats = useBerichteTenantStore((s) => s.allowedFormats)

  // Only formats permitted by the tenant settings are offered (fall back to all
  // if an admin disabled every format).
  const availableFormats = FORMATS.filter((f) => allowedFormats.includes(f))
  const formats = availableFormats.length > 0 ? availableFormats : FORMATS

  const existing = schedulesQuery.data?.schedules.find((s) => s.definition_id === doc.id) ?? null
  const isReleased = doc.status === 'released'

  const [rhythm, setRhythm] = useState<Rhythm>(DEFAULT_RHYTHM)
  const [format, setFormat] = useState<ReportFormat>(defaultFormat)
  const [active, setActive] = useState(true)
  const [recipients, setRecipients] = useState<string[]>([CURRENT_USER.email])
  const [userQuery, setUserQuery] = useState('')
  const [emailInput, setEmailInput] = useState('')

  // Hydrate the form from an existing schedule (or reset to defaults) whenever
  // the modal opens for a different document / schedule.
  useEffect(() => {
    if (!open) return
    setUserQuery('')
    setEmailInput('')
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

  // Internal-user typeahead: match name/email, exclude already-picked users.
  const userMatches = useMemo(() => {
    const q = userQuery.trim().toLowerCase()
    if (!q) return []
    return INTERNAL_USERS.filter(
      (u) =>
        !recipients.includes(u.email) &&
        (u.name.toLowerCase().includes(q) || u.email.toLowerCase().includes(q)),
    ).slice(0, 6)
  }, [userQuery, recipients])

  const addRecipient = (email: string) => {
    setRecipients((prev) => (prev.includes(email) ? prev : [...prev, email]))
  }
  const removeRecipient = (email: string) =>
    setRecipients((prev) => prev.filter((e) => e !== email))

  const addExternalEmail = () => {
    const email = emailInput.trim().toLowerCase()
    if (!email) return
    if (!EMAIL_RE.test(email)) {
      toast.error(t('berichte.docs.schedule.errorEmail'))
      return
    }
    if (recipients.includes(email)) {
      toast.error(t('berichte.docs.schedule.errorEmailDup'))
      return
    }
    addRecipient(email)
    setEmailInput('')
  }

  // Live "next run" preview from the currently chosen rhythm; run history is
  // only meaningful once a schedule exists on the server.
  const nextRun = isReleased ? computeNextRun(rhythmToCron(rhythm)) : null
  const runHistory = existing ? buildRunHistory(existing) : []

  const handleRunNow = () => {
    if (!existing) return
    runNowMutation.mutate(existing.id, {
      onSuccess: () => toast.success(t('berichte.docs.schedule.sentNow')),
      onError: (err) => toast.error((err as Error).message),
    })
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

          {/* Recipients — internal users (typeahead) + external email addresses */}
          <div>
            <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
              {t('berichte.docs.schedule.recipients')}
            </label>

            {/* Internal user typeahead */}
            <div className="relative">
              <Search className="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
              <input
                type="text"
                value={userQuery}
                disabled={!isReleased}
                onChange={(e) => setUserQuery(e.target.value)}
                placeholder={t('berichte.docs.schedule.searchUser')}
                className="w-full rounded-lg border border-border bg-card py-2 pl-9 pr-3 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring disabled:opacity-50"
              />
              {userMatches.length > 0 && (
                <div className="absolute z-20 mt-1 max-h-60 w-full overflow-y-auto rounded-lg border border-border bg-card py-1 shadow-lg">
                  {userMatches.map((u) => (
                    <button
                      key={u.email}
                      type="button"
                      onClick={() => {
                        addRecipient(u.email)
                        setUserQuery('')
                      }}
                      className="flex w-full items-center gap-2.5 px-3 py-1.5 text-left transition-colors hover:bg-secondary"
                    >
                      <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-primary-light text-[10px] font-semibold text-primary">
                        {u.initials}
                      </span>
                      <span className="min-w-0">
                        <span className="block truncate text-sm text-foreground">{u.name}</span>
                        <span className="block truncate text-xs text-muted-foreground">
                          {u.jobTitle}
                        </span>
                      </span>
                    </button>
                  ))}
                </div>
              )}
            </div>

            {/* External email input */}
            <div className="mt-2 flex gap-2">
              <div className="relative flex-1">
                <Mail className="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
                <input
                  type="email"
                  value={emailInput}
                  disabled={!isReleased}
                  onChange={(e) => setEmailInput(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      e.preventDefault()
                      addExternalEmail()
                    }
                  }}
                  placeholder={t('berichte.docs.schedule.externalEmail')}
                  className="w-full rounded-lg border border-border bg-card py-2 pl-9 pr-3 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring disabled:opacity-50"
                />
              </div>
              <button
                type="button"
                disabled={!isReleased}
                onClick={addExternalEmail}
                className="rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground disabled:opacity-50"
              >
                {t('berichte.docs.schedule.add')}
              </button>
            </div>

            {/* Recipient chips */}
            {recipients.length > 0 ? (
              <div className="mt-2.5 flex flex-wrap gap-1.5">
                {recipients.map((email) => {
                  const internal = INTERNAL_BY_EMAIL.get(email)
                  return (
                    <span
                      key={email}
                      className="flex items-center gap-1.5 rounded-full bg-secondary py-0.5 pl-1 pr-2 text-xs text-foreground"
                    >
                      {internal ? (
                        <span className="flex h-5 w-5 items-center justify-center rounded-full bg-primary-light text-[9px] font-semibold text-primary">
                          {internal.initials}
                        </span>
                      ) : (
                        <Mail className="ml-1 h-3 w-3 text-muted-foreground" />
                      )}
                      <span className="max-w-[12rem] truncate">
                        {internal ? internal.name : email}
                      </span>
                      <button
                        type="button"
                        disabled={!isReleased}
                        onClick={() => removeRecipient(email)}
                        aria-label={t('berichte.docs.schedule.removeRecipient')}
                        className="rounded-full p-0.5 transition-colors hover:bg-error-light hover:text-error disabled:opacity-50"
                      >
                        <X className="h-2.5 w-2.5" />
                      </button>
                    </span>
                  )
                })}
              </div>
            ) : (
              <p className="mt-1.5 text-[10px] text-muted-foreground/60">
                {t('berichte.docs.schedule.errorRecipients')}
              </p>
            )}
          </div>

          {/* Format */}
          <div>
            <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
              {t('berichte.docs.schedule.format')}
            </label>
            <div className="flex gap-2">
              {formats.map((f) => (
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

          {/* Next-run preview (read-only, from the chosen rhythm) */}
          <div className="flex items-center justify-between rounded-lg border border-border bg-secondary/40 px-4 py-3">
            <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <CalendarClock className="h-3.5 w-3.5" />
              {t('berichte.docs.schedule.nextRun')}
            </span>
            <span className="text-sm font-medium text-foreground">
              {active && nextRun ? formatDateTime(nextRun.toISOString()) : '—'}
            </span>
          </div>

          {/* Run history + manual "send now" — only once a schedule exists */}
          {existing && (
            <div>
              <div className="mb-2 flex items-center justify-between">
                <span className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
                  <History className="h-3.5 w-3.5" />
                  {t('berichte.docs.schedule.history')}
                </span>
                <button
                  type="button"
                  onClick={handleRunNow}
                  disabled={runNowMutation.isPending}
                  className="flex items-center gap-1.5 rounded-lg border border-border px-2.5 py-1.5 text-xs text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground disabled:opacity-50"
                >
                  <Send className="h-3.5 w-3.5" />
                  {t('berichte.docs.schedule.sendNow')}
                </button>
              </div>
              {runHistory.length === 0 ? (
                <p className="rounded-lg border border-border bg-card px-4 py-3 text-xs text-muted-foreground">
                  {t('berichte.docs.schedule.noRuns')}
                </p>
              ) : (
                <ul className="overflow-hidden rounded-lg border border-border">
                  {runHistory.map((run, i) => (
                    <li
                      key={i}
                      className="flex items-center justify-between gap-3 border-b border-border-muted bg-card px-4 py-2.5 text-sm last:border-0"
                    >
                      <span className="flex items-center gap-2 text-foreground">
                        <Clock className="h-3 w-3 text-muted-foreground" />
                        {formatDateTime(run.at)}
                      </span>
                      <span className="flex items-center gap-2">
                        <span className="text-xs text-muted-foreground">
                          {t('berichte.docs.schedule.recipientCount', {
                            count: existing.recipients.length,
                          })}
                        </span>
                        <span
                          className={`rounded-full px-1.5 py-0.5 text-[10px] font-medium ${
                            RUN_STATUS_STYLES[run.status] ?? 'bg-secondary text-muted-foreground'
                          }`}
                        >
                          {t(`berichte.docs.schedule.runStatus.${run.status}`)}
                        </span>
                      </span>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          )}
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

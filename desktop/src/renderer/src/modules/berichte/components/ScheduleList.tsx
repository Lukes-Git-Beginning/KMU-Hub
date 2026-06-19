import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Bell, CalendarClock, Clock, Mail, Plus, Trash2, X } from 'lucide-react'
import { SortMenu, type SortDirection, type SortFieldOption } from '@/components/shared/SortMenu'
import { useBerichtePrefsStore } from '@/stores/berichtePrefs'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { EmptyState } from '@/components/shared/EmptyState'
import { EmptyReports } from '@/components/shared/illustrations'
import type { ReportDefinition, ReportFormat } from '@/api/berichte-types'
import {
  useCreateSchedule,
  useDeleteSchedule,
  useSchedules,
  useToggleSchedule,
} from '@/api/hooks/useBerichte'
import { formatDateTime } from '@/lib/format'

interface ScheduleListProps {
  definitions: ReportDefinition[]
}

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

// formatDateTime imported from @/lib/format (locale-aware)

const STATUS_STYLES: Record<string, string> = {
  success: 'bg-success-light text-success',
  failed: 'bg-error-light text-error',
  skipped: 'bg-warning-light text-warning',
}

const DOW_MAP: Record<string, number> = { SUN: 0, MON: 1, TUE: 2, WED: 3, THU: 4, FRI: 5, SAT: 6 }

/**
 * Demo-grade next-run estimate from a 5-field cron expression
 * (min hour dom mon dow). Month is ignored for the demo; iterates day-by-day.
 */
function computeNextRun(cron: string, from: Date = new Date()): Date | null {
  const parts = cron.trim().split(/\s+/)
  if (parts.length < 5) return null
  const [minF, hourF, domF, , dowF] = parts
  const min = minF === '*' ? 0 : parseInt(minF, 10)
  const hour = hourF === '*' ? null : parseInt(hourF, 10)
  const dom = domF === '*' ? null : parseInt(domF, 10)
  const dow = dowF === '*' ? null : DOW_MAP[dowF.toUpperCase()] ?? parseInt(dowF, 10)
  if (Number.isNaN(min)) return null
  for (let i = 0; i < 366; i++) {
    const d = new Date(from)
    d.setDate(from.getDate() + i)
    d.setHours(hour ?? from.getHours() + (i === 0 ? 1 : 0), min, 0, 0)
    if (dom !== null && d.getDate() !== dom) continue
    if (dow !== null && d.getDay() !== dow) continue
    if (d > from) return d
  }
  return null
}

export function ScheduleList({ definitions }: ScheduleListProps) {
  const { t } = useTranslation()
  const schedulesQuery = useSchedules()
  const toggleMutation = useToggleSchedule()
  const createMutation = useCreateSchedule()
  const deleteMutation = useDeleteSchedule()
  const defaultFormat = useBerichtePrefsStore((s) => s.defaultFormat)

  const [dialogOpen, setDialogOpen] = useState(false)
  const [defId, setDefId] = useState('')
  const [name, setName] = useState('')
  const [cronExpression, setCronExpression] = useState('0 8 * * MON')
  const [format, setFormat] = useState<ReportFormat>(defaultFormat)
  const [sortField, setSortField] = useState('name')
  const [sortDir, setSortDir] = useState<SortDirection>('asc')
  const [emailInput, setEmailInput] = useState('')
  const [recipients, setRecipients] = useState<string[]>([])
  const [alertThreshold, setAlertThreshold] = useState('')

  const schedules = schedulesQuery.data?.schedules ?? []
  const activeCount = schedules.filter((s) => s.active).length

  const sortFields: SortFieldOption[] = [
    { value: 'name', label: t('berichte.geplant.table.name') },
    { value: 'cron', label: t('berichte.geplant.table.zeitplan') },
    { value: 'status', label: t('berichte.geplant.table.aktiv') },
  ]
  const sortedSchedules = useMemo(() => {
    const arr = [...schedules]
    arr.sort((a, b) => {
      let cmp = 0
      if (sortField === 'name') cmp = a.name.localeCompare(b.name)
      else if (sortField === 'cron') cmp = a.cron_expression.localeCompare(b.cron_expression)
      else if (sortField === 'status') cmp = Number(a.active) - Number(b.active)
      return sortDir === 'asc' ? cmp : -cmp
    })
    return arr
  }, [schedules, sortField, sortDir])

  const handleToggle = (id: string, next: boolean) => {
    toggleMutation.mutate(
      { id, active: next },
      {
        onSuccess: () =>
          toast.info(
            next
              ? t('berichte.geplant.aktiviert')
              : t('berichte.geplant.deaktiviertToast'),
          ),
      },
    )
  }

  const handleDelete = (id: string) => {
    deleteMutation.mutate(id, {
      onSuccess: () =>
        toast.success(
          t('berichte.geplant.geloescht', { defaultValue: 'Geplanter Bericht gelöscht.' }),
        ),
      onError: (err) => toast.error((err as Error).message),
    })
  }

  const resetDialog = () => {
    setDefId('')
    setName('')
    setCronExpression('0 8 * * MON')
    setFormat(defaultFormat)
    setEmailInput('')
    setRecipients([])
    setAlertThreshold('')
  }

  const openDialog = () => {
    resetDialog()
    setDialogOpen(true)
  }

  const addRecipient = () => {
    const email = emailInput.trim()
    if (!email) return
    if (!EMAIL_RE.test(email)) {
      toast.error(t('berichte.dialog.errorEmail'))
      return
    }
    if (recipients.includes(email)) {
      toast.error(t('berichte.dialog.errorEmailDoppelt'))
      return
    }
    setRecipients((prev) => [...prev, email])
    setEmailInput('')
  }

  const removeRecipient = (email: string) =>
    setRecipients((prev) => prev.filter((e) => e !== email))

  const handleSave = () => {
    if (!defId) {
      toast.error(t('berichte.dialog.errorBericht'))
      return
    }
    if (recipients.length === 0) {
      toast.error(t('berichte.dialog.mindestEmpfaenger'))
      return
    }
    if (!name.trim()) {
      toast.error(
        t('berichte.dialog.errorName', { defaultValue: 'Bitte einen Namen angeben.' }),
      )
      return
    }
    createMutation.mutate(
      {
        definition_id: defId,
        name: name.trim(),
        cron_expression: cronExpression.trim(),
        recipients,
        format,
        params: alertThreshold.trim() ? { alert_threshold: Number(alertThreshold) } : {},
        active: true,
      },
      {
        onSuccess: () => {
          toast.success(t('berichte.dialog.successToast'))
          setDialogOpen(false)
        },
        onError: (err) => toast.error((err as Error).message),
      },
    )
  }

  return (
    <>
      <div className="mb-4 flex items-center justify-between">
        <p className="text-sm text-muted-foreground">
          {t('berichte.geplant.aktivInfo', { active: activeCount, total: schedules.length })}
        </p>
        <div className="flex items-center gap-2">
          {schedules.length > 0 && (
            <SortMenu
              options={sortFields}
              field={sortField}
              direction={sortDir}
              onChange={(f, d) => {
                setSortField(f)
                setSortDir(d)
              }}
            />
          )}
          <button
            onClick={openDialog}
            className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground transition-colors hover:bg-button-primary-hover"
          >
            <Plus className="h-4 w-4" />
            {t('berichte.geplant.neuerBericht')}
          </button>
        </div>
      </div>

      <div className="overflow-hidden rounded-lg border border-border bg-card">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border">
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                  {t('berichte.geplant.table.name')}
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                  {t('berichte.geplant.table.zeitplan')}
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                  {t('berichte.geplant.table.empfaenger')}
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                  {t('berichte.geplant.table.letzterLauf')}
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                  {t('berichte.geplant.table.naechsterLauf', { defaultValue: 'Nächster Lauf' })}
                </th>
                <th className="px-4 py-3 text-center text-xs font-medium text-muted-foreground">
                  {t('berichte.geplant.table.aktiv')}
                </th>
                <th className="w-10 px-4 py-3" />
              </tr>
            </thead>
            <tbody>
              {sortedSchedules.map((s) => (
                <tr
                  key={s.id}
                  className="border-b border-border-muted transition-colors last:border-0 hover:bg-secondary/50"
                >
                  <td className="px-4 py-3">
                    <span className="inline-flex items-center gap-1.5 font-medium text-foreground">
                      {s.name}
                      {typeof (s.params as { alert_threshold?: number }).alert_threshold === 'number' && (
                        <Bell
                          className="h-3 w-3 text-primary"
                          aria-label={t('berichte.dialog.alertAktiv', { defaultValue: 'Alert aktiv' })}
                        />
                      )}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <code className="rounded bg-secondary px-2 py-0.5 font-mono text-[11px] text-muted-foreground">
                      {s.cron_expression}
                    </code>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap gap-1">
                      {s.recipients.map((email) => (
                        <span
                          key={email}
                          className="rounded-full bg-secondary px-2 py-0.5 text-xs text-muted-foreground"
                        >
                          {email}
                        </span>
                      ))}
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                      <Clock className="h-3 w-3" />
                      {s.last_run_at
                        ? formatDateTime(s.last_run_at)
                        : t('berichte.geplant.nieGelaufen', { defaultValue: 'Noch nicht gelaufen' })}
                      {s.last_run_status && (
                        <span
                          className={`rounded-full px-1.5 py-0.5 text-[10px] font-medium ${
                            STATUS_STYLES[s.last_run_status] ?? 'bg-secondary text-muted-foreground'
                          }`}
                        >
                          {s.last_run_status}
                        </span>
                      )}
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                      <CalendarClock className="h-3 w-3" />
                      {s.active
                        ? formatDateTime(computeNextRun(s.cron_expression)?.toISOString() ?? null)
                        : t('berichte.geplant.pausiert', { defaultValue: 'Pausiert' })}
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex justify-center">
                      <button
                        onClick={() => handleToggle(s.id, !s.active)}
                        className={`flex h-6 w-10 items-center rounded-full p-0.5 transition-colors ${
                          s.active ? 'bg-primary' : 'bg-secondary'
                        }`}
                        aria-label={
                          s.active
                            ? t('berichte.geplant.toggle.deaktivieren')
                            : t('berichte.geplant.toggle.aktivieren')
                        }
                      >
                        <div
                          className={`h-5 w-5 rounded-full bg-white shadow transition-transform ${
                            s.active ? 'translate-x-4' : 'translate-x-0'
                          }`}
                        />
                      </button>
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <button
                      onClick={() => handleDelete(s.id)}
                      className="rounded-lg p-1.5 text-muted-foreground transition-colors hover:bg-error-light hover:text-error"
                      aria-label={t('common.delete', { defaultValue: 'Löschen' })}
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {!schedulesQuery.isLoading && schedules.length === 0 && (
          <EmptyState
            illustration={<EmptyReports />}
            title={t('berichte.geplant.empty')}
            description={t('berichte.geplant.emptyHint')}
          />
        )}
      </div>

      {/* Create dialog */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="max-w-lg gap-0 p-0">
          <DialogHeader className="border-b border-border px-6 py-4">
            <div className="flex items-center gap-3">
              <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary-light">
                <CalendarClock className="h-4 w-4 text-primary" />
              </div>
              <div>
                <DialogTitle className="text-sm">{t('berichte.dialog.title')}</DialogTitle>
                <DialogDescription>{t('berichte.dialog.subtitle')}</DialogDescription>
              </div>
            </div>
          </DialogHeader>

          <div className="space-y-5 px-6 py-5">
            {/* Report picker */}
            <div>
              <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
                {t('berichte.dialog.berichtSelect')}
              </label>
              <select
                value={defId}
                onChange={(e) => setDefId(e.target.value)}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              >
                <option value="">{t('berichte.dialog.berichtSelectPlaceholder')}</option>
                {definitions.map((def) => (
                  <option key={def.id} value={def.id}>
                    {def.name}
                  </option>
                ))}
              </select>
            </div>

            {/* Name */}
            <div>
              <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
                {t('berichte.dialog.name', { defaultValue: 'Name' })}
              </label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Monatlicher Umsatzbericht"
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>

            {/* Cron */}
            <div>
              <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
                {t('berichte.dialog.zeitplan')}
              </label>
              <input
                type="text"
                value={cronExpression}
                onChange={(e) => setCronExpression(e.target.value)}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 font-mono text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
              <p className="mt-1 text-[10px] text-muted-foreground/60">
                {t('berichte.dialog.cronHint', {
                  defaultValue: 'Cron-Expression, z. B. "0 8 * * MON" für jeden Montag um 8:00 Uhr.',
                })}
              </p>
            </div>

            {/* Format */}
            <div>
              <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
                {t('berichte.erstellen.ausgabeformat')}
              </label>
              <div className="flex gap-2">
                {(['pdf', 'xlsx', 'csv'] as const).map((f) => (
                  <button
                    key={f}
                    type="button"
                    onClick={() => setFormat(f)}
                    className={`flex-1 rounded-lg border px-3 py-2 text-sm uppercase transition-colors ${
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

            {/* Alert threshold (optional, demo) */}
            <div>
              <label className="mb-1.5 flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
                <Bell className="h-3 w-3" />
                {t('berichte.dialog.alertSchwelle', { defaultValue: 'Alert-Schwellwert (optional)' })}
              </label>
              <input
                type="number"
                value={alertThreshold}
                onChange={(e) => setAlertThreshold(e.target.value)}
                placeholder={t('berichte.dialog.alertPlaceholder', { defaultValue: 'z. B. 100000' })}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
              <p className="mt-1 text-[10px] text-muted-foreground/60">
                {t('berichte.dialog.alertHint', {
                  defaultValue:
                    'Benachrichtigung, sobald der Hauptwert des Berichts diese Schwelle überschreitet.',
                })}
              </p>
            </div>

            {/* Recipients */}
            <div>
              <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
                {t('berichte.dialog.empfaenger')}
              </label>
              <div className="flex gap-2">
                <div className="relative flex-1">
                  <Mail className="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
                  <input
                    type="email"
                    value={emailInput}
                    onChange={(e) => setEmailInput(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') {
                        e.preventDefault()
                        addRecipient()
                      }
                    }}
                    placeholder={t('berichte.dialog.emailPlaceholder')}
                    className="w-full rounded-lg border border-border bg-card py-2 pl-9 pr-3 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                  />
                </div>
                <button
                  onClick={addRecipient}
                  className="rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
                >
                  {t('berichte.dialog.hinzufuegen')}
                </button>
              </div>
              {recipients.length > 0 && (
                <div className="mt-2 flex flex-wrap gap-1.5">
                  {recipients.map((email) => (
                    <span
                      key={email}
                      className="flex items-center gap-1 rounded-full bg-secondary px-2 py-0.5 text-xs text-foreground"
                    >
                      {email}
                      <button
                        onClick={() => removeRecipient(email)}
                        className="ml-0.5 rounded-full p-0.5 transition-colors hover:bg-error-light hover:text-error"
                      >
                        <X className="h-2.5 w-2.5" />
                      </button>
                    </span>
                  ))}
                </div>
              )}
              {recipients.length === 0 && (
                <p className="mt-1.5 text-[10px] text-muted-foreground/60">
                  {t('berichte.dialog.mindestEmpfaenger')}
                </p>
              )}
            </div>
          </div>

          <DialogFooter className="border-t border-border px-6 py-4">
            <button
              onClick={() => setDialogOpen(false)}
              className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground transition-colors hover:bg-secondary"
            >
              {t('common.cancel')}
            </button>
            <button
              onClick={handleSave}
              disabled={createMutation.isPending}
              className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground transition-colors hover:bg-button-primary-hover disabled:opacity-60"
            >
              <CalendarClock className="h-4 w-4" />
              {t('berichte.dialog.speichern')}
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}

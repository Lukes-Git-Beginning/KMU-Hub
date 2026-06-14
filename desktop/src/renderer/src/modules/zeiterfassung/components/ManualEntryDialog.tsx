/**
 * Manual time-entry dialog (HR-API backed, mock-first).
 *
 * Lets the user log a back-dated entry with project/customer attribution,
 * service/activity, billable flag and note — the clockodo-style taxonomy.
 */
import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2, Receipt } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog'
import { cn } from '@/lib/cn'
import { toast } from 'sonner'
import { useTimeProjects, useCreateTimeEntry } from '@/api/hooks/hr-hooks'
import { formatWorkMinutes } from '@/lib/worktime'

interface ManualEntryDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

function todayInput(): string {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

function minutesBetween(start: string, end: string): number {
  const [sh, sm] = start.split(':').map(Number)
  const [eh, em] = end.split(':').map(Number)
  if ([sh, sm, eh, em].some(Number.isNaN)) return 0
  return eh * 60 + em - (sh * 60 + sm)
}

export function ManualEntryDialog({ open, onOpenChange }: ManualEntryDialogProps) {
  const { t } = useTranslation()
  const { data: projects } = useTimeProjects()
  const createEntry = useCreateTimeEntry()

  const [date, setDate] = useState(todayInput())
  const [start, setStart] = useState('09:00')
  const [end, setEnd] = useState('17:00')
  const [breakMin, setBreakMin] = useState(30)
  const [projectId, setProjectId] = useState('')
  const [activity, setActivity] = useState('')
  const [billable, setBillable] = useState(false)
  const [note, setNote] = useState('')

  // Default the billable flag from the selected project.
  useEffect(() => {
    const p = projects?.find((x) => x.id === projectId)
    if (p) setBillable(p.billableDefault)
  }, [projectId, projects])

  const gross = minutesBetween(start, end)
  const net = Math.max(0, gross - breakMin)
  const isValid = date && start && end && gross > 0 && date <= todayInput()

  const reset = () => {
    setDate(todayInput()); setStart('09:00'); setEnd('17:00'); setBreakMin(30)
    setProjectId(''); setActivity(''); setBillable(false); setNote('')
  }

  const handleSubmit = () => {
    if (gross <= 0) { toast.error(t('zeiterfassung.manual.endAfterStart')); return }
    if (date > todayInput()) { toast.error(t('zeiterfassung.manual.noFutureDate')); return }
    createEntry.mutate(
      {
        clockIn: new Date(`${date}T${start}`).toISOString(),
        clockOut: new Date(`${date}T${end}`).toISOString(),
        breakMinutes: breakMin,
        projectId: projectId || undefined,
        activity: activity.trim() || undefined,
        billable,
        note: note.trim() || undefined,
      },
      {
        onSuccess: () => { onOpenChange(false); reset() },
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{t('zeiterfassung.manual.title')}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div className="space-y-1.5">
            <Label>{t('zeiterfassung.manual.date')}</Label>
            <Input type="date" value={date} max={todayInput()} onChange={(e) => setDate(e.target.value)} />
          </div>

          <div className="grid grid-cols-3 gap-3">
            <div className="space-y-1.5">
              <Label>{t('zeiterfassung.manual.start')}</Label>
              <Input type="time" value={start} onChange={(e) => setStart(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label>{t('zeiterfassung.manual.end')}</Label>
              <Input type="time" value={end} onChange={(e) => setEnd(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label>{t('zeiterfassung.manual.break')}</Label>
              <Input type="number" min={0} value={breakMin} onChange={(e) => setBreakMin(Math.max(0, Number(e.target.value)))} />
            </div>
          </div>

          {gross > 0 ? (
            <p className="text-xs text-muted-foreground">
              {t('zeiterfassung.manual.net')}:{' '}
              <span className="font-medium tabular-nums text-foreground">{formatWorkMinutes(net)}</span>
              {breakMin > 0 && <span className="text-muted-foreground"> ({formatWorkMinutes(gross)} − {breakMin}min)</span>}
            </p>
          ) : (
            <p className="text-xs text-destructive">{t('zeiterfassung.manual.endAfterStart')}</p>
          )}

          <div className="space-y-1.5">
            <Label>{t('zeiterfassung.manual.project')}</Label>
            <Select value={projectId} onValueChange={setProjectId}>
              <SelectTrigger>
                <SelectValue placeholder={t('zeiterfassung.manual.selectProject')} />
              </SelectTrigger>
              <SelectContent>
                {(projects ?? []).map((p) => (
                  <SelectItem key={p.id} value={p.id}>
                    <span className="flex items-center gap-2">
                      <span className="h-2.5 w-2.5 shrink-0 rounded-full" style={{ backgroundColor: p.color }} />
                      <span className="text-muted-foreground">{p.customerName}</span>
                      <span>· {p.name}</span>
                    </span>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-1.5">
            <Label>{t('zeiterfassung.manual.activity')}</Label>
            <Input
              value={activity}
              onChange={(e) => setActivity(e.target.value)}
              placeholder={t('zeiterfassung.manual.activityPlaceholder')}
            />
          </div>

          {/* Billable toggle */}
          <button
            type="button"
            onClick={() => setBillable((v) => !v)}
            className={cn(
              'flex w-full items-center justify-between rounded-lg border px-3 py-2 text-sm transition-colors',
              billable ? 'border-success/40 bg-success/5' : 'border-border hover:bg-secondary',
            )}
          >
            <span className="flex items-center gap-2">
              <Receipt className={cn('h-4 w-4', billable ? 'text-success' : 'text-muted-foreground')} />
              <span className="text-foreground">{t('zeiterfassung.manual.billable')}</span>
            </span>
            <span
              className={cn(
                'rounded-full px-2 py-0.5 text-xs font-medium',
                billable ? 'bg-success/15 text-success' : 'bg-secondary text-muted-foreground',
              )}
            >
              {billable ? t('common.yes') : t('common.no')}
            </span>
          </button>

          <div className="space-y-1.5">
            <Label>{t('zeiterfassung.manual.note')}</Label>
            <Input
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder={t('zeiterfassung.manual.notePlaceholder')}
              onKeyDown={(e) => e.key === 'Enter' && isValid && handleSubmit()}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>{t('common.cancel')}</Button>
          <Button onClick={handleSubmit} disabled={!isValid || createEntry.isPending}>
            {createEntry.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            {t('zeiterfassung.manual.createEntry')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

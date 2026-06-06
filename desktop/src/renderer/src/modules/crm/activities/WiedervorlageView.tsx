/**
 * Wiedervorlage (follow-up) agenda — open activities grouped by due date.
 *
 * Buckets: overdue / today / this week / later / no date. Overdue is flagged
 * red. Each item can be completed (removing it) or rescheduled inline (pushing
 * its due date). Complements the flat activities list view.
 */
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Check, CalendarClock, AlertTriangle } from 'lucide-react'
import { toast } from 'sonner'
import {
  useActivities,
  useCompleteActivity,
  useUpdateActivity,
} from '@/api/hooks/useActivities'
import { Skeleton } from '@/components/ui/skeleton'
import { EmptyState } from '@/components/shared'
import { activityTypeIcon, activityTypeLabel } from './activityUtils'

type Bucket = 'overdue' | 'today' | 'thisWeek' | 'later' | 'noDate'
const BUCKET_ORDER: Bucket[] = ['overdue', 'today', 'thisWeek', 'later', 'noDate']

interface ActivityRow {
  id?: string
  subject?: string
  type?: string
  activity_type?: string
  due_date?: string
  completed?: boolean
  contact_name?: string
  company_name?: string
  deal_name?: string
}

function startOfToday(): number {
  const d = new Date()
  d.setHours(0, 0, 0, 0)
  return d.getTime()
}

function bucketFor(due?: string): Bucket {
  if (!due) return 'noDate'
  const today = startOfToday()
  const t = new Date(due).getTime()
  const dayMs = 86_400_000
  if (t < today) return 'overdue'
  if (t < today + dayMs) return 'today'
  if (t < today + 7 * dayMs) return 'thisWeek'
  return 'later'
}

function relativeDue(due: string, t: (k: string, o?: Record<string, unknown>) => string): string {
  const today = startOfToday()
  const d = new Date(due)
  d.setHours(0, 0, 0, 0)
  const diff = Math.round((d.getTime() - today) / 86_400_000)
  if (diff === 0) return t('crm.activities.bucket.today')
  if (diff < 0) return t('crm.activities.agenda.overdueDays', { days: Math.abs(diff) })
  if (diff === 1) return t('crm.activities.agenda.inDays', { days: 1 })
  return t('crm.activities.agenda.inDays', { days: diff })
}

export default function WiedervorlageView() {
  const { t } = useTranslation()
  const { data, isLoading } = useActivities({ page_size: 200, sort_by: 'due_date', sort_desc: false })
  const completeActivity = useCompleteActivity()
  const updateActivity = useUpdateActivity()

  const open = useMemo(
    () => ((data?.activities ?? []) as ActivityRow[]).filter((a) => a.completed !== true),
    [data?.activities]
  )

  const grouped = useMemo(() => {
    const map: Record<Bucket, ActivityRow[]> = { overdue: [], today: [], thisWeek: [], later: [], noDate: [] }
    for (const a of open) map[bucketFor(a.due_date)].push(a)
    return map
  }, [open])

  const handleComplete = (id?: string) => {
    if (!id) return
    completeActivity.mutate(id, { onSuccess: () => toast.success(t('crm.activities.toast.completed')) })
  }
  const handleReschedule = (id: string | undefined, due_date: string) => {
    if (!id || !due_date) return
    updateActivity.mutate(
      { id, due_date },
      { onSuccess: () => toast.success(t('crm.activities.toast.rescheduled')) }
    )
  }

  if (isLoading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 5 }).map((_, i) => <Skeleton key={i} className="h-14 w-full" />)}
      </div>
    )
  }

  if (open.length === 0) {
    return <EmptyState icon={Check} title={t('crm.activities.agenda.emptyTitle')} description={t('crm.activities.agenda.empty')} />
  }

  return (
    <div className="space-y-6">
      {BUCKET_ORDER.map((bucket) => {
        const items = grouped[bucket]
        if (items.length === 0) return null
        const isOverdue = bucket === 'overdue'
        return (
          <section key={bucket}>
            <div className="mb-2 flex items-center gap-2">
              {isOverdue && <AlertTriangle className="h-4 w-4 text-destructive" />}
              <h3 className={`text-sm font-semibold ${isOverdue ? 'text-destructive' : 'text-foreground'}`}>
                {t(`crm.activities.bucket.${bucket}`)}
              </h3>
              <span className="rounded-full bg-secondary px-1.5 text-xs text-muted-foreground">{items.length}</span>
            </div>
            <div className="space-y-2">
              {items.map((a) => {
                const Icon = activityTypeIcon(a.activity_type ?? a.type)
                const link = a.contact_name || a.company_name || a.deal_name
                return (
                  <div
                    key={a.id}
                    className={`flex items-center gap-3 rounded-lg border bg-card p-3 transition-colors ${
                      isOverdue ? 'border-destructive/30' : 'border-border'
                    }`}
                  >
                    <button
                      onClick={() => handleComplete(a.id)}
                      disabled={completeActivity.isPending}
                      className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full border border-border text-muted-foreground transition-colors hover:border-success hover:bg-success-light hover:text-success"
                      title={t('crm.activities.markComplete')}
                    >
                      <Check className="h-3.5 w-3.5" />
                    </button>
                    <Icon className="h-4 w-4 shrink-0 text-muted-foreground" />
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium text-foreground">{a.subject}</p>
                      <p className="truncate text-xs text-muted-foreground">
                        {t(activityTypeLabel(a.activity_type ?? a.type))}
                        {link ? ` · ${link}` : ''}
                      </p>
                    </div>
                    {a.due_date && (
                      <span className={`hidden shrink-0 items-center gap-1 text-xs sm:flex ${isOverdue ? 'text-destructive' : 'text-muted-foreground'}`}>
                        <CalendarClock className="h-3.5 w-3.5" />
                        {relativeDue(a.due_date, t)}
                      </span>
                    )}
                    <label className="shrink-0" title={t('crm.activities.agenda.reschedule')}>
                      <span className="sr-only">{t('crm.activities.agenda.reschedule')}</span>
                      <input
                        type="date"
                        value={a.due_date ? a.due_date.slice(0, 10) : ''}
                        onChange={(e) => handleReschedule(a.id, e.target.value)}
                        className="h-8 rounded-md border border-border bg-transparent px-2 text-xs text-muted-foreground outline-none focus:border-primary"
                      />
                    </label>
                  </div>
                )
              })}
            </div>
          </section>
        )
      })}
    </div>
  )
}

/**
 * ProjectPortfolioView — cross-project portfolio table.
 *
 * A dense overview of all projects with status, task progress, due date,
 * owner and a health signal derived from schedule-vs-progress (no budget
 * field on the backend yet, so health is honest: "behind schedule" compares
 * elapsed time against completed-task ratio). All data from useProjects.
 */
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { cn } from '@/lib'

interface PortfolioProject {
  id?: string
  name?: string
  project_key?: string
  status?: string
  progress?: number
  task_count?: number
  completed_task_count?: number
  start_date?: string
  end_date?: string
  owner_name?: string
}

interface ProjectPortfolioViewProps {
  projects: PortfolioProject[]
}

type Health = 'done' | 'overdue' | 'atRisk' | 'onTrack'

const STATUS_STYLE: Record<string, string> = {
  active: 'bg-primary-light text-primary',
  completed: 'bg-success/15 text-success',
  on_hold: 'bg-warning/15 text-warning-foreground',
  archived: 'bg-secondary text-muted-foreground',
}

const HEALTH_STYLE: Record<Health, string> = {
  done: 'bg-success',
  onTrack: 'bg-success',
  atRisk: 'bg-warning',
  overdue: 'bg-destructive',
}

function deriveHealth(p: PortfolioProject): Health {
  const progress = p.progress ?? 0
  if (p.status === 'completed' || progress >= 100) return 'done'
  const now = Date.now()
  const end = p.end_date ? new Date(p.end_date).getTime() : null
  if (end && end < now) return 'overdue'
  // Schedule vs. progress: how much of the timeline has elapsed?
  const start = p.start_date ? new Date(p.start_date).getTime() : null
  if (start && end && end > start) {
    const elapsed = ((now - start) / (end - start)) * 100
    if (elapsed - progress > 20) return 'atRisk'
  }
  return 'onTrack'
}

export default function ProjectPortfolioView({ projects }: ProjectPortfolioViewProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()

  function formatDue(date?: string): { label: string; overdue: boolean } {
    if (!date) return { label: '–', overdue: false }
    const d = new Date(date)
    const overdue = d.getTime() < Date.now()
    return {
      label: d.toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit', year: '2-digit' }),
      overdue,
    }
  }

  return (
    <div className="overflow-hidden rounded-lg border border-border">
      {/* Header */}
      <div className="grid grid-cols-[1.6fr_0.8fr_1.4fr_0.9fr_0.9fr_1fr] items-center gap-3 border-b border-border bg-secondary/40 px-4 py-2 text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
        <span>{t('work.portfolio.project')}</span>
        <span>{t('work.portfolio.status')}</span>
        <span>{t('work.portfolio.progress')}</span>
        <span className="text-right">{t('work.portfolio.dueDate')}</span>
        <span>{t('work.portfolio.owner')}</span>
        <span>{t('work.portfolio.health')}</span>
      </div>

      {/* Rows */}
      <div className="divide-y divide-border">
        {projects.map((p) => {
          const progress = p.progress ?? 0
          const done = p.completed_task_count ?? 0
          const total = p.task_count ?? 0
          const due = formatDue(p.end_date)
          const health = deriveHealth(p)
          return (
            <button
              key={p.id}
              type="button"
              onClick={() => p.id && navigate(`/work/projects/${p.id}`)}
              className="grid w-full grid-cols-[1.6fr_0.8fr_1.4fr_0.9fr_0.9fr_1fr] items-center gap-3 px-4 py-2.5 text-left transition-colors hover:bg-accent/50"
            >
              {/* Project name + key */}
              <div className="flex items-center gap-2 min-w-0">
                <span className="truncate text-sm font-medium text-foreground">{p.name}</span>
                <span className="shrink-0 font-mono text-[10px] text-muted-foreground">{p.project_key}</span>
              </div>

              {/* Status */}
              <span>
                <span
                  className={cn(
                    'inline-flex rounded-full px-2 py-0.5 text-[11px] font-medium',
                    STATUS_STYLE[p.status ?? 'active'] ?? STATUS_STYLE.active,
                  )}
                >
                  {t(`work.portfolio.statusValue.${p.status ?? 'active'}`)}
                </span>
              </span>

              {/* Progress */}
              <div className="flex items-center gap-2">
                <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-muted">
                  <div className="h-full rounded-full bg-primary transition-all" style={{ width: `${progress}%` }} />
                </div>
                <span className="w-14 shrink-0 text-right text-[11px] tabular-nums text-muted-foreground">
                  {done}/{total}
                </span>
              </div>

              {/* Due date */}
              <span className={cn('text-right text-xs tabular-nums', due.overdue ? 'font-medium text-destructive' : 'text-muted-foreground')}>
                {due.label}
              </span>

              {/* Owner */}
              <span className="truncate text-xs text-muted-foreground">{p.owner_name ?? '–'}</span>

              {/* Health */}
              <span className="flex items-center gap-1.5">
                <span className={cn('h-2 w-2 shrink-0 rounded-full', HEALTH_STYLE[health])} />
                <span className="text-xs text-muted-foreground">{t(`work.portfolio.healthValue.${health}`)}</span>
              </span>
            </button>
          )
        })}
      </div>
    </div>
  )
}

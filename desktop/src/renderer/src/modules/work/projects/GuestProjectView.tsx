/**
 * GuestProjectView — Standalone read-only project view for external guests.
 *
 * Displays project name, description, task progress, milestones, and
 * recent status updates in a clean minimal layout without sidebar or
 * full navigation. Intended for guest/client access links.
 * Mock data for design — backend swap: real project data from guest API.
 */
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router-dom'
import {
  CheckCircle2,
  Circle,
  Clock,
  Milestone,
  ExternalLink,
  CalendarDays,
  Users,
  TrendingUp,
  AlertCircle,
  Loader2,
} from 'lucide-react'
import { cn } from '@/lib'
import { Badge } from '@/components/ui/badge'
import {
  useProject,
  useProjectGuestOverview,
  type GuestMilestone,
  type GuestStatusUpdate,
} from '@/api/hooks/useProjects'

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface GuestProjectViewProps {
  /** Project ID — falls back to the route param when rendered as a route. */
  projectId?: string
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function GuestProjectView({ projectId: propProjectId }: GuestProjectViewProps) {
  const { t } = useTranslation()
  const params = useParams<{ id: string }>()
  const projectId = propProjectId ?? params.id ?? ''

  const { data: projectData, isLoading } = useProject(projectId)
  const { data: overview } = useProjectGuestOverview(projectId)
  const project = projectData?.project

  const totalTasks = (project?.task_count as number) ?? 0
  const completedTasks = (project?.completed_task_count as number) ?? 0
  const milestones: GuestMilestone[] = overview?.milestones ?? []
  const statusUpdates: GuestStatusUpdate[] = overview?.statusUpdates ?? []

  const progressPercent = useMemo(
    () => (totalTasks > 0 ? Math.round((completedTasks / totalTasks) * 100) : 0),
    [completedTasks, totalTasks]
  )

  if (isLoading || !project) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background text-muted-foreground">
        <Loader2 className="h-6 w-6 animate-spin" />
      </div>
    )
  }

  const projectKey = (project.project_key as string) ?? ''

  return (
    <div className="min-h-screen bg-background">
      {/* Header bar */}
      <header className="border-b border-border bg-card">
        <div className="max-w-4xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="flex items-center justify-center h-9 w-9 rounded-lg bg-primary text-primary-foreground text-sm font-bold">
              {projectKey.substring(0, 2)}
            </div>
            <div>
              <h1 className="text-lg font-semibold text-foreground">
                {project.name}
              </h1>
              <p className="text-xs text-muted-foreground">
                {t('work.guest.subtitle')}
              </p>
            </div>
          </div>
          <Badge variant="outline" className="text-xs">
            <ExternalLink className="h-3 w-3 mr-1" />
            {t('work.guest.readOnly')}
          </Badge>
        </div>
      </header>

      {/* Main content */}
      <main className="max-w-4xl mx-auto px-6 py-8 space-y-8">
        {/* Project description */}
        <section>
          <p className="text-sm text-muted-foreground leading-relaxed">
            {project.description}
          </p>
        </section>

        {/* Key metrics */}
        <section className="grid grid-cols-4 gap-4">
          <MetricCard
            icon={TrendingUp}
            label={t('work.guest.progress')}
            value={`${progressPercent}%`}
            sublabel={t('work.guest.taskCount', { completed: completedTasks, total: totalTasks })}
          />
          <MetricCard
            icon={Users}
            label={t('work.guest.team')}
            value={`${(project.member_count as number) ?? 0}`}
            sublabel={t('work.guest.employees')}
          />
          <MetricCard
            icon={CalendarDays}
            label={t('work.guest.startDate')}
            value={formatDate((project.start_date as string) ?? '')}
            sublabel=""
          />
          <MetricCard
            icon={CalendarDays}
            label={t('work.guest.targetDate')}
            value={formatDate((project.end_date as string) ?? '')}
            sublabel=""
          />
        </section>

        {/* Overall progress bar */}
        <section>
          <div className="flex items-center justify-between text-sm mb-2">
            <span className="font-medium text-foreground">{t('work.guest.overallProgress')}</span>
            <span className="font-mono text-muted-foreground">{progressPercent}%</span>
          </div>
          <div className="h-3 w-full rounded-full bg-muted overflow-hidden">
            <div
              className="h-full rounded-full bg-primary transition-all"
              style={{ width: `${progressPercent}%` }}
            />
          </div>
        </section>

        {/* Milestones */}
        <section>
          <h2 className="text-base font-semibold text-foreground mb-4 flex items-center gap-2">
            <Milestone className="h-4 w-4 text-muted-foreground" />
            {t('work.guest.milestones')}
          </h2>
          <div className="space-y-3">
            {milestones.map((ms) => (
              <MilestoneRow key={ms.id} milestone={ms} />
            ))}
          </div>
        </section>

        {/* Status updates */}
        <section>
          <h2 className="text-base font-semibold text-foreground mb-4 flex items-center gap-2">
            <Clock className="h-4 w-4 text-muted-foreground" />
            {t('work.guest.statusUpdates')}
          </h2>
          <div className="space-y-4">
            {statusUpdates.map((update) => (
              <StatusUpdateCard key={update.id} update={update} />
            ))}
          </div>
        </section>
      </main>

      {/* Footer */}
      <footer className="border-t border-border mt-12 py-6">
        <div className="max-w-4xl mx-auto px-6 flex items-center justify-center">
          <p className="text-xs text-muted-foreground">
            Powered by{' '}
            <span className="font-semibold text-foreground">Cosmi</span>
            {' '}&mdash; {t('work.guest.tagline')}
          </p>
        </div>
      </footer>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

function MetricCard({
  icon: Icon,
  label,
  value,
  sublabel,
}: {
  icon: typeof TrendingUp
  label: string
  value: string
  sublabel: string
}) {
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <div className="flex items-center gap-1.5 text-xs text-muted-foreground mb-2">
        <Icon className="h-3.5 w-3.5" />
        {label}
      </div>
      <p className="text-xl font-semibold text-foreground">{value}</p>
      {sublabel && (
        <p className="text-[10px] text-muted-foreground mt-0.5">{sublabel}</p>
      )}
    </div>
  )
}

function MilestoneRow({ milestone }: { milestone: GuestMilestone }) {
  const statusIcon =
    milestone.status === 'completed' ? (
      <CheckCircle2 className="h-4 w-4 text-success flex-shrink-0" />
    ) : milestone.status === 'in-progress' ? (
      <Clock className="h-4 w-4 text-primary flex-shrink-0 animate-pulse" />
    ) : (
      <Circle className="h-4 w-4 text-muted-foreground/40 flex-shrink-0" />
    )

  const { t } = useTranslation()
  const statusBadge =
    milestone.status === 'completed'
      ? t('work.guest.statusCompleted')
      : milestone.status === 'in-progress'
        ? t('work.guest.statusInProgress')
        : t('work.guest.statusPlanned')

  const badgeVariant: 'default' | 'secondary' | 'outline' =
    milestone.status === 'completed'
      ? 'default'
      : milestone.status === 'in-progress'
        ? 'secondary'
        : 'outline'

  return (
    <div className="flex items-center gap-4 rounded-lg border border-border bg-card px-4 py-3">
      {statusIcon}
      <div className="flex-1 min-w-0">
        <p
          className={cn(
            'text-sm font-medium',
            milestone.status === 'completed'
              ? 'text-muted-foreground line-through'
              : 'text-foreground'
          )}
        >
          {milestone.title}
        </p>
        <p className="text-[10px] text-muted-foreground">
          {t('work.guest.dueDate')}: {formatDate(milestone.dueDate)}
        </p>
      </div>

      {/* Progress for in-progress milestones */}
      {milestone.status === 'in-progress' && (
        <div className="flex items-center gap-2 w-28">
          <div className="flex-1 h-1.5 rounded-full bg-muted overflow-hidden">
            <div
              className="h-full rounded-full bg-primary"
              style={{ width: `${milestone.progress}%` }}
            />
          </div>
          <span className="text-[10px] font-mono text-muted-foreground w-8 text-right">
            {milestone.progress}%
          </span>
        </div>
      )}

      <Badge variant={badgeVariant} className="text-[10px] flex-shrink-0">
        {statusBadge}
      </Badge>
    </div>
  )
}

function StatusUpdateCard({ update }: { update: GuestStatusUpdate }) {
  const typeIcon =
    update.type === 'milestone' ? (
      <CheckCircle2 className="h-3.5 w-3.5 text-success" />
    ) : update.type === 'risk' ? (
      <AlertCircle className="h-3.5 w-3.5 text-warning-foreground" />
    ) : null

  const borderClass =
    update.type === 'risk'
      ? 'border-l-yellow-500'
      : update.type === 'milestone'
        ? 'border-l-emerald-500'
        : 'border-l-transparent'

  return (
    <div
      className={cn(
        'rounded-lg border border-border bg-card p-4 border-l-4',
        borderClass
      )}
    >
      <div className="flex items-center gap-2 mb-2">
        {typeIcon}
        <span className="text-xs font-medium text-foreground">
          {update.author}
        </span>
        <span className="text-[10px] text-muted-foreground">
          {formatDate(update.date)}
        </span>
      </div>
      <p className="text-sm text-muted-foreground leading-relaxed">
        {update.text}
      </p>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function formatDate(iso: string): string {
  try {
    return new Date(iso).toLocaleDateString('de-DE', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
    })
  } catch {
    return iso
  }
}

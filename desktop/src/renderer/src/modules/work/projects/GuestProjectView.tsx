/**
 * GuestProjectView — Standalone read-only project view for external guests.
 *
 * Displays project name, description, task progress, milestones, and
 * recent status updates in a clean minimal layout without sidebar or
 * full navigation. Intended for guest/client access links.
 * Mock data for design — backend swap: real project data from guest API.
 */
import { useMemo } from 'react'
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
} from 'lucide-react'
import { cn } from '@/lib'
import { Badge } from '@/components/ui/badge'

// ---------------------------------------------------------------------------
// Mock data
// ---------------------------------------------------------------------------

interface MockMilestone {
  id: string
  title: string
  dueDate: string
  status: 'completed' | 'in-progress' | 'upcoming'
  progress: number
}

interface MockStatusUpdate {
  id: string
  date: string
  author: string
  text: string
  type: 'update' | 'milestone' | 'risk'
}

const MOCK_PROJECT = {
  name: 'Website Redesign 2026',
  description:
    'Kompletter Relaunch der Unternehmenswebsite mit neuem Design-System, verbesserter Performance und modernem Tech-Stack. Inklusive CMS-Migration und SEO-Optimierung.',
  projectKey: 'WEB',
  startDate: '2026-01-15',
  targetDate: '2026-06-30',
  totalTasks: 48,
  completedTasks: 22,
  teamSize: 6,
}

const MOCK_MILESTONES: MockMilestone[] = [
  { id: 'ms1', title: 'Design-System fertiggestellt', dueDate: '2026-02-01', status: 'completed', progress: 100 },
  { id: 'ms2', title: 'Frontend-Grundstruktur', dueDate: '2026-02-28', status: 'completed', progress: 100 },
  { id: 'ms3', title: 'Backend-API v1', dueDate: '2026-03-15', status: 'in-progress', progress: 65 },
  { id: 'ms4', title: 'CMS-Migration', dueDate: '2026-04-15', status: 'upcoming', progress: 0 },
  { id: 'ms5', title: 'QA & Testing', dueDate: '2026-05-15', status: 'upcoming', progress: 0 },
  { id: 'ms6', title: 'Go-Live', dueDate: '2026-06-30', status: 'upcoming', progress: 0 },
]

const MOCK_STATUS_UPDATES: MockStatusUpdate[] = [
  {
    id: 'su1',
    date: '2026-02-20',
    author: 'Anna Mueller',
    text: 'Frontend-Komponenten-Bibliothek zu 85% fertiggestellt. Performance-Benchmarks zeigen deutliche Verbesserungen gegenueber der aktuellen Website.',
    type: 'update',
  },
  {
    id: 'su2',
    date: '2026-02-18',
    author: 'Thomas Fischer',
    text: 'API-Endpoints fuer Kontakte und Produkte implementiert und getestet. Authentifizierung via JWT steht.',
    type: 'update',
  },
  {
    id: 'su3',
    date: '2026-02-15',
    author: 'Anna Mueller',
    text: 'Meilenstein "Frontend-Grundstruktur" erfolgreich abgeschlossen. Routing, State Management und Design Tokens integriert.',
    type: 'milestone',
  },
  {
    id: 'su4',
    date: '2026-02-12',
    author: 'Max Schmidt',
    text: 'CMS-Datenexport zeigt Inkompatibilitaeten bei einigen benutzerdefinierten Feldern. Wird in Sprint 5 adressiert.',
    type: 'risk',
  },
]

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface GuestProjectViewProps {
  /** Project ID (for future API integration). */
  projectId?: string
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function GuestProjectView({ projectId: _projectId }: GuestProjectViewProps) {
  const project = MOCK_PROJECT
  const progressPercent = useMemo(
    () =>
      project.totalTasks > 0
        ? Math.round((project.completedTasks / project.totalTasks) * 100)
        : 0,
    [project.completedTasks, project.totalTasks]
  )

  return (
    <div className="min-h-screen bg-background">
      {/* Header bar */}
      <header className="border-b border-border bg-card">
        <div className="max-w-4xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="flex items-center justify-center h-9 w-9 rounded-lg bg-primary text-primary-foreground text-sm font-bold">
              {project.projectKey.substring(0, 2)}
            </div>
            <div>
              <h1 className="text-lg font-semibold text-foreground">
                {project.name}
              </h1>
              <p className="text-xs text-muted-foreground">
                Projektuebersicht (Gastzugang)
              </p>
            </div>
          </div>
          <Badge variant="outline" className="text-xs">
            <ExternalLink className="h-3 w-3 mr-1" />
            Lesezugriff
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
            label="Fortschritt"
            value={`${progressPercent}%`}
            sublabel={`${project.completedTasks} / ${project.totalTasks} Aufgaben`}
          />
          <MetricCard
            icon={Users}
            label="Team"
            value={`${project.teamSize}`}
            sublabel="Mitarbeiter"
          />
          <MetricCard
            icon={CalendarDays}
            label="Startdatum"
            value={formatDate(project.startDate)}
            sublabel=""
          />
          <MetricCard
            icon={CalendarDays}
            label="Zieldatum"
            value={formatDate(project.targetDate)}
            sublabel=""
          />
        </section>

        {/* Overall progress bar */}
        <section>
          <div className="flex items-center justify-between text-sm mb-2">
            <span className="font-medium text-foreground">Gesamtfortschritt</span>
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
            Meilensteine
          </h2>
          <div className="space-y-3">
            {MOCK_MILESTONES.map((ms) => (
              <MilestoneRow key={ms.id} milestone={ms} />
            ))}
          </div>
        </section>

        {/* Status updates */}
        <section>
          <h2 className="text-base font-semibold text-foreground mb-4 flex items-center gap-2">
            <Clock className="h-4 w-4 text-muted-foreground" />
            Aktuelle Status-Updates
          </h2>
          <div className="space-y-4">
            {MOCK_STATUS_UPDATES.map((update) => (
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
            <span className="font-semibold text-foreground">KMU Hub</span>
            {' '}&mdash; All-in-One CRM fuer DACH-KMUs
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

function MilestoneRow({ milestone }: { milestone: MockMilestone }) {
  const statusIcon =
    milestone.status === 'completed' ? (
      <CheckCircle2 className="h-4 w-4 text-emerald-500 flex-shrink-0" />
    ) : milestone.status === 'in-progress' ? (
      <Clock className="h-4 w-4 text-primary flex-shrink-0 animate-pulse" />
    ) : (
      <Circle className="h-4 w-4 text-muted-foreground/40 flex-shrink-0" />
    )

  const statusBadge =
    milestone.status === 'completed'
      ? 'Abgeschlossen'
      : milestone.status === 'in-progress'
        ? 'In Bearbeitung'
        : 'Geplant'

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
          Faellig: {formatDate(milestone.dueDate)}
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

function StatusUpdateCard({ update }: { update: MockStatusUpdate }) {
  const typeIcon =
    update.type === 'milestone' ? (
      <CheckCircle2 className="h-3.5 w-3.5 text-emerald-500" />
    ) : update.type === 'risk' ? (
      <AlertCircle className="h-3.5 w-3.5 text-yellow-500" />
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

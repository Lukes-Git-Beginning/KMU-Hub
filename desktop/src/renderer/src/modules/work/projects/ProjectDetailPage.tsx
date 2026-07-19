/**
 * Project detail page showing project info, view toggle, and task views.
 *
 * Accessed via /work/projects/:id. Shows project header with name, key,
 * view toggle (List/Kanban/Gantt/Auslastung), settings, hours-to-invoice,
 * and new task button. Content area renders TaskListView, KanbanBoard,
 * GanttChart, or AuslastungReport based on persisted user preference.
 * BudgetSection is shown as a collapsible panel for list/kanban views.
 * Includes TaskDetailPanel slide-over for quick task viewing.
 */
import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useParams, useNavigate, Routes, Route } from 'react-router-dom'
import {
  ArrowLeft,
  LayoutList,
  Columns3,
  GanttChartSquare,
  CalendarDays,
  BarChart3,
  Settings,
  Plus,
  Receipt,
} from 'lucide-react'
import {
  useProject,
  useProjectStatuses,
  useProjectPreference,
  useSetPreference,
} from '@/api/hooks/useProjects'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import ProjectSettingsDialog from './ProjectSettingsDialog'
import TaskListView from '../list/TaskListView'
import KanbanBoard from '../kanban/KanbanBoard'
import TaskCreateDialog from '../components/TaskCreateDialog'
import TaskDetailPanel from '../tasks/TaskDetailPanel'
import TaskDetailPage from '../tasks/TaskDetailPage'
import GanttChart from '../gantt/GanttChart'
import WorkCalendarView from '../calendar/WorkCalendarView'
import HoursToInvoiceDialog from '../components/HoursToInvoiceDialog'
import BudgetSection from '../components/BudgetSection'
import AuslastungReport from '../components/AuslastungReport'
import { useWorkPrefsStore } from '@/stores/workPrefs'
import { useHasCapability } from '@/hooks/useCapability'
import { useProjectCan } from '../useWorkPermissions'
import { RestrictedModeBadge } from '@/components/shared/rbac/RestrictedModeBadge'

/**
 * Wrapper component that handles nested routing for project detail.
 * Renders either the project board view or the full task detail page.
 */
export default function ProjectDetailPage() {
  return (
    <Routes>
      <Route index element={<ProjectBoardView />} />
      <Route path="tasks/:taskId" element={<TaskDetailPage />} />
    </Routes>
  )
}

/**
 * Project board view with task list/Kanban and slide-over task panel.
 */
function ProjectBoardView() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [createOpen, setCreateOpen] = useState(false)
  const [invoiceOpen, setInvoiceOpen] = useState(false)

  const { data, isLoading, error, refetch } = useProject(id ?? '')
  const { data: statusesData } = useProjectStatuses(id ?? '')
  const { data: prefData } = useProjectPreference(id ?? '')
  const setPreference = useSetPreference()

  const project = data?.project
  const statuses = statusesData?.statuses ?? []

  const projectCan = useProjectCan(project)
  const canCreateTask = useHasCapability('work:task:create')
  const canCreateInvoice = useHasCapability('finance:invoice:create')

  // View type: project-specific preference wins; otherwise the user's personal
  // default view (work settings) is used as the initial value.
  type ViewType = 'list' | 'kanban' | 'gantt' | 'calendar' | 'auslastung'
  const personalDefaultView = useWorkPrefsStore((s) => s.defaultView)
  const [view, setView] = useState<ViewType>(personalDefaultView)


  useEffect(() => {
    const vt = prefData?.view_type
    if (vt === 'kanban' || vt === 'list' || vt === 'gantt' || vt === 'calendar') {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- sync local editable state from prop
      setView(vt as ViewType)
    }
  }, [prefData?.view_type])

  function handleViewChange(newView: ViewType) {
    setView(newView)
    if (id) {
      // Persist auslastung as gantt in preferences (not a standard view type)
      const persistView = newView === 'auslastung' ? 'gantt' : newView
      setPreference.mutate({ projectId: id, view_type: persistView as string })
    }
  }

  if (error) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="text-center">
          <p className="text-lg font-semibold text-foreground">
            {t('work.projects.loadError')}
          </p>
          <p className="mt-2 text-sm text-muted-foreground">
            {error instanceof Error ? error.message : t('work.common.unexpectedError')}
          </p>
          <Button variant="outline" className="mt-4" onClick={() => refetch()}>
            {t('common.retry')}
          </Button>
        </div>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="p-6 space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full" />
      </div>
    )
  }

  if (!project) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="text-center">
          <p className="text-lg font-semibold text-foreground">
            {t('work.projects.notFound')}
          </p>
          <Button
            variant="outline"
            className="mt-4"
            onClick={() => navigate('/work/projects')}
          >
            {t('work.projects.backToList')}
          </Button>
        </div>
      </div>
    )
  }

  // Build statuses with proper id for Kanban columns
  const kanbanStatuses = statuses
    .filter((s) => s.id)
    .map((s) => ({
      id: s.id!,
      name: s.name ?? t('work.list.noStatus'),
      color: s.color,
      is_closed: s.is_closed,
    }))

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-border px-6 py-3">
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => navigate('/work/projects')}
          >
            <ArrowLeft className="h-4 w-4 mr-1" />
            {t('common.back')}
          </Button>
          <div className="flex items-center gap-2">
            <h1 className="text-xl font-semibold text-foreground">
              {project.name}
            </h1>
            <Badge variant="outline" className="font-mono text-xs">
              {project.project_key}
            </Badge>
            <RestrictedModeBadge module="work" />
          </div>
        </div>

        <div className="flex items-center gap-2">
          {/* View toggle */}
          <div className="flex items-center rounded-md border border-border">
            <Button
              variant={view === 'list' ? 'secondary' : 'ghost'}
              size="sm"
              className="rounded-r-none"
              onClick={() => handleViewChange('list')}
              title={t('work.views.list')}
            >
              <LayoutList className="h-4 w-4" />
            </Button>
            <Button
              variant={view === 'kanban' ? 'secondary' : 'ghost'}
              size="sm"
              className="rounded-none border-x border-border"
              onClick={() => handleViewChange('kanban')}
              title={t('work.views.kanban')}
            >
              <Columns3 className="h-4 w-4" />
            </Button>
            <Button
              variant={view === 'gantt' ? 'secondary' : 'ghost'}
              size="sm"
              className="rounded-none border-r border-border"
              onClick={() => handleViewChange('gantt')}
              title={t('work.views.gantt')}
            >
              <GanttChartSquare className="h-4 w-4" />
            </Button>
            <Button
              variant={view === 'calendar' ? 'secondary' : 'ghost'}
              size="sm"
              className="rounded-none border-r border-border"
              onClick={() => handleViewChange('calendar')}
              title={t('work.views.calendar')}
            >
              <CalendarDays className="h-4 w-4" />
            </Button>
            <Button
              variant={view === 'auslastung' ? 'secondary' : 'ghost'}
              size="sm"
              className="rounded-l-none"
              onClick={() => handleViewChange('auslastung')}
              title={t('work.views.utilization')}
            >
              <BarChart3 className="h-4 w-4" />
            </Button>
          </div>

          {/* Hours to Invoice — creates a finance invoice draft */}
          {canCreateInvoice && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => setInvoiceOpen(true)}
              title={t('work.invoice.title')}
            >
              <Receipt className="h-4 w-4" />
            </Button>
          )}

          {projectCan.edit && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => setSettingsOpen(true)}
            >
              <Settings className="h-4 w-4" />
            </Button>
          )}

          {canCreateTask && (
            <Button size="sm" className="gap-1" onClick={() => setCreateOpen(true)}>
              <Plus className="h-4 w-4" />
              {t('work.tasks.newTask')}
            </Button>
          )}
        </div>
      </div>

      {/* Budget section (shown for list and kanban views) */}
      {(view === 'list' || view === 'kanban') && (
        <BudgetSection
          projectId={id ?? ''}
          projectName={project.name}
        />
      )}

      {/* Content area: List, Kanban, Gantt, or Auslastung view */}
      <div className="flex-1 min-h-0">
        {view === 'list' ? (
          <TaskListView projectId={id ?? ''} statuses={statuses} />
        ) : view === 'kanban' ? (
          <KanbanBoard projectId={id ?? ''} statuses={kanbanStatuses} />
        ) : view === 'calendar' ? (
          <WorkCalendarView projectId={id ?? ''} />
        ) : view === 'auslastung' ? (
          <AuslastungReport projectId={id ?? ''} />
        ) : (
          <GanttChart projectId={id ?? ''} />
        )}
      </div>

      {/* Task detail slide-over panel */}
      <TaskDetailPanel />

      {/* Dialogs */}
      <ProjectSettingsDialog
        open={settingsOpen}
        onOpenChange={setSettingsOpen}
        projectId={id ?? ''}
      />

      <TaskCreateDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        projectId={id ?? ''}
        statuses={statuses}
      />

      <HoursToInvoiceDialog
        open={invoiceOpen}
        onOpenChange={setInvoiceOpen}
        projectId={id ?? ''}
        projectName={project.name}
      />
    </div>
  )
}

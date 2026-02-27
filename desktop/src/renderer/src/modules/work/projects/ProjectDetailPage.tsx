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
import { useParams, useNavigate, Routes, Route } from 'react-router-dom'
import {
  ArrowLeft,
  LayoutList,
  Columns3,
  GanttChartSquare,
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
import HoursToInvoiceDialog from '../components/HoursToInvoiceDialog'
import BudgetSection from '../components/BudgetSection'
import AuslastungReport from '../components/AuslastungReport'

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

  // View type from preferences (default to list)
  type ViewType = 'list' | 'kanban' | 'gantt' | 'auslastung'
  const [view, setView] = useState<ViewType>('list')

  // Sync view from preferences once loaded
  useEffect(() => {
    const vt = prefData?.view_type
    if (vt === 'kanban' || vt === 'list' || vt === 'gantt') {
      setView(vt as ViewType)
    }
  }, [prefData?.view_type])

  function handleViewChange(newView: ViewType) {
    setView(newView)
    if (id) {
      // Persist auslastung as gantt in preferences (not a standard view type)
      const persistView = newView === 'auslastung' ? 'gantt' : newView
      setPreference.mutate({ projectId: id, view_type: persistView as any })
    }
  }

  if (error) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="text-center">
          <p className="text-lg font-semibold text-foreground">
            Fehler beim Laden des Projekts
          </p>
          <p className="mt-2 text-sm text-muted-foreground">
            {error instanceof Error ? error.message : 'Ein unerwarteter Fehler ist aufgetreten.'}
          </p>
          <Button variant="outline" className="mt-4" onClick={() => refetch()}>
            Erneut versuchen
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
            Projekt nicht gefunden
          </p>
          <Button
            variant="outline"
            className="mt-4"
            onClick={() => navigate('/work/projects')}
          >
            Zurueck zur Liste
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
      name: s.name ?? 'Ohne Name',
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
            Zurueck
          </Button>
          <div className="flex items-center gap-2">
            <h1 className="text-xl font-semibold text-foreground">
              {project.name}
            </h1>
            <Badge variant="outline" className="font-mono text-xs">
              {project.project_key}
            </Badge>
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
              title="Listenansicht"
            >
              <LayoutList className="h-4 w-4" />
            </Button>
            <Button
              variant={view === 'kanban' ? 'secondary' : 'ghost'}
              size="sm"
              className="rounded-none border-x border-border"
              onClick={() => handleViewChange('kanban')}
              title="Kanban-Ansicht"
            >
              <Columns3 className="h-4 w-4" />
            </Button>
            <Button
              variant={view === 'gantt' ? 'secondary' : 'ghost'}
              size="sm"
              className="rounded-none border-r border-border"
              onClick={() => handleViewChange('gantt')}
              title="Gantt-Ansicht"
            >
              <GanttChartSquare className="h-4 w-4" />
            </Button>
            <Button
              variant={view === 'auslastung' ? 'secondary' : 'ghost'}
              size="sm"
              className="rounded-l-none"
              onClick={() => handleViewChange('auslastung')}
              title="Auslastung"
            >
              <BarChart3 className="h-4 w-4" />
            </Button>
          </div>

          {/* Hours to Invoice */}
          <Button
            variant="outline"
            size="sm"
            onClick={() => setInvoiceOpen(true)}
            title="Stunden zu Rechnung"
          >
            <Receipt className="h-4 w-4" />
          </Button>

          <Button
            variant="outline"
            size="sm"
            onClick={() => setSettingsOpen(true)}
          >
            <Settings className="h-4 w-4" />
          </Button>

          <Button size="sm" className="gap-1" onClick={() => setCreateOpen(true)}>
            <Plus className="h-4 w-4" />
            Neue Aufgabe
          </Button>
        </div>
      </div>

      {/* Budget section (shown for list and kanban views) */}
      {(view === 'list' || view === 'kanban') && (
        <BudgetSection
          budget={50000}
          projectName={project.name}
        />
      )}

      {/* Content area: List, Kanban, Gantt, or Auslastung view */}
      <div className="flex-1 min-h-0">
        {view === 'list' ? (
          <TaskListView projectId={id ?? ''} statuses={statuses} />
        ) : view === 'kanban' ? (
          <KanbanBoard projectId={id ?? ''} statuses={kanbanStatuses} />
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
        projectName={project.name}
      />
    </div>
  )
}

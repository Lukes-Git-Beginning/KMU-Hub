/**
 * Project detail page showing project info, view toggle, and task list.
 *
 * Accessed via /work/projects/:id. Shows project header with name,
 * key, view toggle (List/Kanban), settings, and new task button.
 * Content area is a placeholder for views built in 06-06.
 */
import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  ArrowLeft,
  LayoutList,
  Columns3,
  Settings,
  Plus,
  FolderKanban,
} from 'lucide-react'
import { useProject, useProjectStatuses } from '@/api/hooks/useProjects'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import ProjectSettingsDialog from './ProjectSettingsDialog'

export default function ProjectDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [view, setView] = useState<'list' | 'kanban'>('list')
  const [settingsOpen, setSettingsOpen] = useState(false)

  const { data, isLoading, error, refetch } = useProject(id ?? '')
  const { data: statusesData } = useProjectStatuses(id ?? '')

  const project = data?.project
  const statuses = statusesData?.statuses ?? []

  function showComingSoon() {
    alert('Kommt bald')
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
              onClick={() => setView('list')}
            >
              <LayoutList className="h-4 w-4" />
            </Button>
            <Button
              variant={view === 'kanban' ? 'secondary' : 'ghost'}
              size="sm"
              className="rounded-l-none"
              onClick={() => setView('kanban')}
            >
              <Columns3 className="h-4 w-4" />
            </Button>
          </div>

          <Button
            variant="outline"
            size="sm"
            onClick={() => setSettingsOpen(true)}
          >
            <Settings className="h-4 w-4" />
          </Button>

          <Button size="sm" className="gap-1" onClick={showComingSoon}>
            <Plus className="h-4 w-4" />
            Neue Aufgabe
          </Button>
        </div>
      </div>

      {/* Content area -- placeholder for task views (06-06) */}
      <div className="flex-1 flex items-center justify-center p-6">
        <div className="text-center">
          <FolderKanban className="mx-auto h-12 w-12 text-muted-foreground" />
          <p className="mt-4 text-lg font-medium text-foreground">
            {view === 'list' ? 'Listenansicht' : 'Kanban-Board'}
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            Aufgabenansicht wird in Kuerze implementiert.
          </p>
          {statuses.length > 0 && (
            <div className="mt-4 flex flex-wrap justify-center gap-2">
              {statuses.map((status) => (
                <Badge
                  key={status.id}
                  variant="secondary"
                  className="text-xs"
                  style={
                    status.color
                      ? { backgroundColor: `${status.color}20`, color: status.color, borderColor: status.color }
                      : undefined
                  }
                >
                  {status.name}
                  {status.is_default && ' (Standard)'}
                </Badge>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Settings dialog */}
      <ProjectSettingsDialog
        open={settingsOpen}
        onOpenChange={setSettingsOpen}
        projectId={id ?? ''}
      />
    </div>
  )
}

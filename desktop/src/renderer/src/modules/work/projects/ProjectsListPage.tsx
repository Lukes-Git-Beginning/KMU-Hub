/**
 * Projects list page with search, grid display, and project creation.
 *
 * Displays projects as cards showing name, key, description,
 * member/task counts, and owner. Supports search and pagination.
 */
import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Plus, Search, FolderKanban, ChevronLeft, ChevronRight } from 'lucide-react'
import { useProjects, useCreateProject } from '@/api/hooks/useProjects'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import ProjectCreateDialog from './ProjectCreateDialog'

const PAGE_SIZE = 12

export default function ProjectsListPage() {
  const navigate = useNavigate()
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [page, setPage] = useState(1)
  const [createOpen, setCreateOpen] = useState(false)

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search)
      setPage(1)
    }, 300)
    return () => clearTimeout(timer)
  }, [search])

  const { data, isLoading, error, refetch } = useProjects({
    page,
    page_size: PAGE_SIZE,
    search: debouncedSearch || undefined,
  })

  const createProject = useCreateProject()

  const projects = data?.projects ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  if (error) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="text-center">
          <p className="text-lg font-semibold text-foreground">
            Fehler beim Laden der Projekte
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

  return (
    <div className="p-6 space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Projekte</h1>
        <Button onClick={() => setCreateOpen(true)} className="gap-2">
          <Plus className="h-4 w-4" />
          Neues Projekt
        </Button>
      </div>

      {/* Search bar */}
      <div className="relative max-w-sm">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          placeholder="Projekte suchen..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="pl-9"
        />
      </div>

      {/* Project cards */}
      {isLoading ? (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-40 w-full rounded-lg" />
          ))}
        </div>
      ) : projects.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16">
          <FolderKanban className="h-12 w-12 text-muted-foreground" />
          <p className="mt-4 text-lg font-medium text-foreground">
            Keine Projekte gefunden
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            {debouncedSearch
              ? 'Versuche einen anderen Suchbegriff.'
              : 'Erstelle dein erstes Projekt, um loszulegen.'}
          </p>
          {!debouncedSearch && (
            <Button className="mt-4 gap-2" onClick={() => setCreateOpen(true)}>
              <Plus className="h-4 w-4" />
              Erstes Projekt erstellen
            </Button>
          )}
        </div>
      ) : (
        <>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
            {projects.map((project) => (
              <Card
                key={project.id}
                className="cursor-pointer transition-shadow hover:shadow-md"
                onClick={() => navigate(`/work/projects/${project.id}`)}
              >
                <CardHeader className="pb-2">
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-base truncate">
                      {project.name}
                    </CardTitle>
                    <Badge variant="outline" className="shrink-0 text-xs font-mono">
                      {project.project_key}
                    </Badge>
                  </div>
                </CardHeader>
                <CardContent>
                  <p className="text-sm text-muted-foreground line-clamp-2 mb-3">
                    {project.description || 'Keine Beschreibung'}
                  </p>
                  <div className="flex items-center justify-between text-xs text-muted-foreground">
                    <span>
                      {project.member_count ?? 0} Mitglied{(project.member_count ?? 0) !== 1 ? 'er' : ''}
                    </span>
                    <span>
                      {project.task_count ?? 0} Aufgabe{(project.task_count ?? 0) !== 1 ? 'n' : ''}
                    </span>
                    {project.owner_name && (
                      <span className="truncate max-w-[120px]">{project.owner_name}</span>
                    )}
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between">
              <p className="text-sm text-muted-foreground">
                {total} Projekt{total !== 1 ? 'e' : ''} gesamt
              </p>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page <= 1}
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                >
                  <ChevronLeft className="h-4 w-4" />
                </Button>
                <span className="text-sm text-muted-foreground">
                  Seite {page} von {totalPages}
                </span>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page >= totalPages}
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                >
                  <ChevronRight className="h-4 w-4" />
                </Button>
              </div>
            </div>
          )}
        </>
      )}

      {/* Create project dialog */}
      <ProjectCreateDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onSubmit={async (values) => {
          await createProject.mutateAsync(values)
          setCreateOpen(false)
        }}
        isSubmitting={createProject.isPending}
      />
    </div>
  )
}

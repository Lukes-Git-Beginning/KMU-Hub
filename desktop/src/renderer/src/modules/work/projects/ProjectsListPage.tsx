/**
 * Projects list page with search, grid display, project creation,
 * and template management.
 *
 * Displays projects as cards showing name, key, description,
 * member/task counts, and owner. Supports search, pagination,
 * creating from templates, and filtering template projects.
 */
import { useState, useEffect } from 'react'
import { PageHeader } from '@/components/shared/PageHeader'
import { useNavigate } from 'react-router-dom'
import {
  Plus,
  Search,
  FolderKanban,
  ChevronLeft,
  ChevronRight,
  Copy,
  LayoutTemplate,
} from 'lucide-react'
import { cn } from '@/lib'
import {
  useProjects,
  useCreateProject,
  useCreateFromTemplate,
} from '@/api/hooks/useProjects'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import ProjectCreateDialog from './ProjectCreateDialog'

const PAGE_SIZE = 12

export default function ProjectsListPage() {
  const navigate = useNavigate()
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [page, setPage] = useState(1)
  const [createOpen, setCreateOpen] = useState(false)
  const [showTemplates, setShowTemplates] = useState(false)
  const [templateDialogOpen, setTemplateDialogOpen] = useState(false)
  const [templateName, setTemplateName] = useState('')
  const [templateKey, setTemplateKey] = useState('')
  const [selectedTemplateId, setSelectedTemplateId] = useState('')

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
    templates_only: showTemplates || undefined,
  })

  // Also fetch templates for the "create from template" dialog
  const { data: templatesData } = useProjects({
    templates_only: true,
    page_size: 50,
  })

  const createProject = useCreateProject()
  const createFromTemplate = useCreateFromTemplate()

  const projects = data?.projects ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const templates = templatesData?.projects ?? []

  async function handleCreateFromTemplate() {
    if (!selectedTemplateId || !templateName.trim() || !templateKey.trim()) return
    await createFromTemplate.mutateAsync({
      template_id: selectedTemplateId,
      name: templateName.trim(),
      project_key: templateKey.trim().toUpperCase(),
    })
    setTemplateDialogOpen(false)
    setSelectedTemplateId('')
    setTemplateName('')
    setTemplateKey('')
  }

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
      <PageHeader
        title="Projekte"
        icon={FolderKanban}
        moduleId="projects"
        actions={
          <div className="flex items-center gap-2">
            {templates.length > 0 && (
              <Button
                variant="outline"
                onClick={() => setTemplateDialogOpen(true)}
                className="gap-2"
              >
                <Copy className="h-4 w-4" />
                Aus Vorlage erstellen
              </Button>
            )}
            <Button onClick={() => setCreateOpen(true)} className="gap-2">
              <Plus className="h-4 w-4" />
              Neues Projekt
            </Button>
          </div>
        }
      />

      {/* Search bar + template toggle */}
      <div className="flex items-center gap-3">
        <div className="relative max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Projekte suchen..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
          />
        </div>

        <Button
          variant="outline"
          size="sm"
          className={cn(
            'h-8 gap-1 text-xs',
            showTemplates && 'border-primary text-primary'
          )}
          onClick={() => {
            setShowTemplates(!showTemplates)
            setPage(1)
          }}
        >
          <LayoutTemplate className="h-3.5 w-3.5" />
          Vorlagen
        </Button>
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
            {showTemplates ? 'Keine Vorlagen gefunden' : 'Keine Projekte gefunden'}
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            {debouncedSearch
              ? 'Versuche einen anderen Suchbegriff.'
              : showTemplates
                ? 'Speichere ein Projekt als Vorlage, um es hier zu sehen.'
                : 'Erstelle dein erstes Projekt, um loszulegen.'}
          </p>
          {!debouncedSearch && !showTemplates && (
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
                    <div className="flex items-center gap-1.5 shrink-0">
                      {project.is_template && (
                        <Badge variant="secondary" className="text-xs">
                          Vorlage
                        </Badge>
                      )}
                      <Badge variant="outline" className="text-xs font-mono">
                        {project.project_key}
                      </Badge>
                    </div>
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

      {/* Create from template dialog */}
      <Dialog open={templateDialogOpen} onOpenChange={setTemplateDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Projekt aus Vorlage erstellen</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 pt-2">
            {/* Template selection */}
            <div className="space-y-1.5">
              <label className="text-sm font-medium">Vorlage</label>
              <div className="max-h-40 overflow-y-auto space-y-1 rounded-md border border-border p-1">
                {templates.length === 0 ? (
                  <p className="px-2 py-3 text-sm text-muted-foreground text-center">
                    Keine Vorlagen vorhanden.
                  </p>
                ) : (
                  templates.map((tpl) => (
                    <button
                      key={tpl.id}
                      type="button"
                      className={cn(
                        'flex w-full items-center gap-2 rounded px-2 py-2 text-sm hover:bg-accent transition-colors',
                        tpl.id === selectedTemplateId && 'bg-accent'
                      )}
                      onClick={() => setSelectedTemplateId(tpl.id ?? '')}
                    >
                      <LayoutTemplate className="h-4 w-4 text-muted-foreground shrink-0" />
                      <span className="flex-1 text-left truncate">{tpl.name}</span>
                      <Badge variant="outline" className="text-xs font-mono shrink-0">
                        {tpl.project_key}
                      </Badge>
                    </button>
                  ))
                )}
              </div>
            </div>

            {/* New project name */}
            <div className="space-y-1.5">
              <label className="text-sm font-medium">Projektname</label>
              <Input
                placeholder="Neuer Projektname..."
                value={templateName}
                onChange={(e) => setTemplateName(e.target.value)}
              />
            </div>

            {/* New project key */}
            <div className="space-y-1.5">
              <label className="text-sm font-medium">Projektschluessel</label>
              <Input
                placeholder="z.B. ACME"
                value={templateKey}
                onChange={(e) => setTemplateKey(e.target.value.toUpperCase())}
                className="font-mono"
                maxLength={10}
              />
              <p className="text-xs text-muted-foreground">
                Wird automatisch in Grossbuchstaben umgewandelt.
              </p>
            </div>

            <div className="flex justify-end gap-2">
              <Button
                variant="outline"
                onClick={() => setTemplateDialogOpen(false)}
              >
                Abbrechen
              </Button>
              <Button
                onClick={handleCreateFromTemplate}
                disabled={
                  !selectedTemplateId ||
                  !templateName.trim() ||
                  !templateKey.trim() ||
                  createFromTemplate.isPending
                }
              >
                {createFromTemplate.isPending ? 'Erstelle...' : 'Aus Vorlage erstellen'}
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}

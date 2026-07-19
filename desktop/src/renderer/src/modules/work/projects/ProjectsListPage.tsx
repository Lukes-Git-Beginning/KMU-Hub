/**
 * Projects list page with search, grid display, project creation,
 * and template management.
 *
 * Displays projects as cards showing name, key, description,
 * member/task counts, and owner. Supports search, pagination,
 * creating from templates, and filtering template projects.
 */
import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { PageHeader } from '@/components/shared/PageHeader'
import { EmptyState } from '@/components/shared'
import { EmptyGeneric } from '@/components/shared/illustrations'
import { useNavigate } from 'react-router-dom'
import {
  Plus,
  Search,
  FolderKanban,
  ChevronLeft,
  ChevronRight,
  Copy,
  LayoutTemplate,
  LayoutGrid,
  Table2,
} from 'lucide-react'
import ProjectPortfolioView from './ProjectPortfolioView'
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
import { useHasCapability } from '@/hooks/useCapability'
import { RestrictedModeBadge } from '@/components/shared/rbac/RestrictedModeBadge'

const PAGE_SIZE = 12

export default function ProjectsListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [page, setPage] = useState(1)
  const [createOpen, setCreateOpen] = useState(false)
  const [showTemplates, setShowTemplates] = useState(false)
  const [viewMode, setViewMode] = useState<'cards' | 'portfolio'>('cards')
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
    page_size: viewMode === 'portfolio' ? 100 : PAGE_SIZE,
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
  const canCreateProject = useHasCapability('work:project:create')

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

  return (
    <div className="p-6 space-y-4">
      <PageHeader
        title={t('work.projects.title')}
        icon={FolderKanban}
        moduleId="projects"
        actions={
          <div className="flex items-center gap-2">
            <RestrictedModeBadge module="work" />
            {canCreateProject && templates.length > 0 && (
              <Button
                variant="outline"
                onClick={() => setTemplateDialogOpen(true)}
                className="gap-2"
              >
                <Copy className="h-4 w-4" />
                {t('work.projects.createFromTemplate')}
              </Button>
            )}
            {canCreateProject && (
              <Button onClick={() => setCreateOpen(true)} className="gap-2">
                <Plus className="h-4 w-4" />
                {t('work.projects.newProject')}
              </Button>
            )}
          </div>
        }
      />

      {/* Search bar + template toggle */}
      <div className="flex items-center gap-3">
        <div className="relative max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder={t('work.projects.searchPlaceholder')}
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
          {t('work.projects.templates')}
        </Button>

        {/* View toggle: cards | portfolio */}
        <div className="ml-auto flex items-center rounded-md border border-border">
          <button
            type="button"
            onClick={() => setViewMode('cards')}
            title={t('work.portfolio.viewCards')}
            className={cn(
              'flex h-8 items-center gap-1 rounded-l-md px-2.5 text-xs transition-colors',
              viewMode === 'cards' ? 'bg-secondary font-medium text-foreground' : 'text-muted-foreground hover:bg-secondary/50'
            )}
          >
            <LayoutGrid className="h-3.5 w-3.5" />
            {t('work.portfolio.viewCards')}
          </button>
          <button
            type="button"
            onClick={() => setViewMode('portfolio')}
            title={t('work.portfolio.viewPortfolio')}
            className={cn(
              'flex h-8 items-center gap-1 rounded-r-md px-2.5 text-xs transition-colors',
              viewMode === 'portfolio' ? 'bg-secondary font-medium text-foreground' : 'text-muted-foreground hover:bg-secondary/50'
            )}
          >
            <Table2 className="h-3.5 w-3.5" />
            {t('work.portfolio.viewPortfolio')}
          </button>
        </div>
      </div>

      {/* Project cards */}
      {isLoading ? (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-40 w-full rounded-lg" />
          ))}
        </div>
      ) : projects.length === 0 ? (
        <EmptyState
          illustration={<EmptyGeneric />}
          title={showTemplates ? t('work.projects.noTemplates') : t('work.projects.noProjects')}
          description={
            debouncedSearch
              ? t('work.projects.tryOtherSearch')
              : showTemplates
                ? t('work.projects.noTemplatesHint')
                : t('work.projects.noProjectsHint')
          }
          action={!debouncedSearch && !showTemplates && canCreateProject ? { label: t('work.projects.createFirst'), onClick: () => setCreateOpen(true) } : undefined}
        />
      ) : viewMode === 'portfolio' ? (
        <ProjectPortfolioView projects={projects} />
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
                          {t('work.projects.template')}
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
                    {project.description || t('work.projects.noDescription')}
                  </p>
                  <div className="flex items-center justify-between text-xs text-muted-foreground">
                    <span>
                      {t('work.projects.memberCount', { count: project.member_count ?? 0 })}
                    </span>
                    <span>
                      {t('work.projects.taskCount', { count: project.task_count ?? 0 })}
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
                {t('work.projects.totalProjects', { count: total })}
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
                  {t('work.pagination.page', { page, totalPages })}
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
            <DialogTitle>{t('work.projects.createFromTemplate')}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 pt-2">
            {/* Template selection */}
            <div className="space-y-1.5">
              <label className="text-sm font-medium">{t('work.projects.template')}</label>
              <div className="max-h-40 overflow-y-auto space-y-1 rounded-md border border-border p-1">
                {templates.length === 0 ? (
                  <p className="px-2 py-3 text-sm text-muted-foreground text-center">
                    {t('work.projects.noTemplatesAvailable')}
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
              <label className="text-sm font-medium">{t('work.projects.projectName')}</label>
              <Input
                placeholder={t('work.projects.newProjectNamePlaceholder')}
                value={templateName}
                onChange={(e) => setTemplateName(e.target.value)}
              />
            </div>

            {/* New project key */}
            <div className="space-y-1.5">
              <label className="text-sm font-medium">{t('work.projects.projectKey')}</label>
              <Input
                placeholder="z.B. ACME"
                value={templateKey}
                onChange={(e) => setTemplateKey(e.target.value.toUpperCase())}
                className="font-mono"
                maxLength={10}
              />
              <p className="text-xs text-muted-foreground">
                {t('work.projects.keyAutoUppercase')}
              </p>
            </div>

            <div className="flex justify-end gap-2">
              <Button
                variant="outline"
                onClick={() => setTemplateDialogOpen(false)}
              >
                {t('common.cancel')}
              </Button>
              {canCreateProject && (
                <Button
                  onClick={handleCreateFromTemplate}
                  disabled={
                    !selectedTemplateId ||
                    !templateName.trim() ||
                    !templateKey.trim() ||
                    createFromTemplate.isPending
                  }
                >
                  {createFromTemplate.isPending ? t('work.projects.creating') : t('work.projects.createFromTemplate')}
                </Button>
              )}
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}

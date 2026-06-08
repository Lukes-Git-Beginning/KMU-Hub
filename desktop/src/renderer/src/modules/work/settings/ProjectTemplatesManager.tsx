/**
 * ProjectTemplatesManager — tenant-wide project template administration.
 *
 * Embedded in the work settings panel. Lists project templates, allows inline
 * rename, and spinning up a new project from a template. Templates are projects
 * with is_template=true (backend). Deleting a template needs a backend endpoint
 * (tracked in backend-gaps.md).
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { LayoutTemplate, Plus, X } from 'lucide-react'
import { toast } from 'sonner'
import { useProjects, useUpdateProject, useCreateFromTemplate } from '@/api/hooks/useProjects'

export function ProjectTemplatesManager() {
  const { t } = useTranslation()
  const { data, isLoading } = useProjects({ templates_only: true })
  const templates = data?.projects ?? []
  const updateProject = useUpdateProject()
  const createFromTemplate = useCreateFromTemplate()

  const [creatingFrom, setCreatingFrom] = useState<string | null>(null)
  const [name, setName] = useState('')
  const [projectKey, setProjectKey] = useState('')

  const handleCreate = (templateId: string) => {
    if (!name.trim() || !projectKey.trim()) return
    createFromTemplate.mutate(
      { template_id: templateId, name: name.trim(), project_key: projectKey.trim().toUpperCase() },
      {
        onSuccess: () => {
          toast.success(t('work.settings.templates.created', { name: name.trim() }))
          setCreatingFrom(null)
          setName('')
          setProjectKey('')
        },
      },
    )
  }

  if (isLoading) {
    return <p className="text-sm text-muted-foreground">{t('common.loading')}</p>
  }

  return (
    <div className="space-y-3">
      <p className="text-xs text-muted-foreground">{t('work.settings.templates.count', { count: templates.length })}</p>

      {templates.length === 0 ? (
        <div className="rounded-lg border border-dashed border-border p-6 text-center">
          <LayoutTemplate className="mx-auto h-7 w-7 text-muted-foreground/30" />
          <p className="mt-2 text-sm text-muted-foreground">{t('work.settings.templates.empty')}</p>
          <p className="text-xs text-muted-foreground/70">{t('work.settings.templates.emptyHint')}</p>
        </div>
      ) : (
        <div className="space-y-1.5">
          {templates.map((tpl) => (
            <div key={tpl.id} className="rounded-lg border border-border bg-card p-3">
              <div className="flex items-center gap-2.5">
                <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary-light">
                  <LayoutTemplate className="h-4 w-4 text-primary" />
                </span>
                <div className="min-w-0 flex-1">
                  <input
                    defaultValue={tpl.name}
                    key={`${tpl.id}-${tpl.name}`}
                    onBlur={(e) => {
                      const v = e.target.value.trim()
                      if (v && v !== tpl.name) updateProject.mutate({ id: tpl.id as string, name: v })
                    }}
                    className="w-full rounded border border-transparent bg-transparent px-1 py-0.5 text-sm font-medium text-foreground outline-none hover:border-border focus:border-primary"
                  />
                  {tpl.description && (
                    <p className="truncate px-1 text-xs text-muted-foreground">{tpl.description as string}</p>
                  )}
                </div>
                <button
                  onClick={() => { setCreatingFrom(tpl.id as string); setName(''); setProjectKey('') }}
                  className="flex shrink-0 items-center gap-1.5 rounded-md border border-border px-2.5 py-1.5 text-xs font-medium text-foreground transition-colors hover:bg-secondary"
                >
                  <Plus className="h-3.5 w-3.5" />
                  {t('work.settings.templates.useTemplate')}
                </button>
              </div>

              {creatingFrom === tpl.id && (
                <div className="mt-3 flex flex-wrap items-center gap-2 rounded-lg border border-primary/30 bg-primary/5 p-3">
                  <input
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder={t('work.settings.templates.projectName')}
                    autoFocus
                    className="h-8 min-w-0 flex-1 rounded-md border border-border bg-transparent px-2 text-sm outline-none focus:border-primary"
                  />
                  <input
                    value={projectKey}
                    onChange={(e) => setProjectKey(e.target.value)}
                    placeholder={t('work.settings.templates.projectKey')}
                    maxLength={6}
                    className="h-8 w-24 rounded-md border border-border bg-transparent px-2 text-sm uppercase outline-none focus:border-primary"
                  />
                  <button
                    onClick={() => setCreatingFrom(null)}
                    aria-label={t('common.cancel')}
                    className="rounded-md p-1.5 text-muted-foreground transition-colors hover:text-foreground"
                  >
                    <X className="h-3.5 w-3.5" />
                  </button>
                  <button
                    onClick={() => handleCreate(tpl.id as string)}
                    disabled={!name.trim() || !projectKey.trim()}
                    className="rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
                  >
                    {t('work.settings.templates.create')}
                  </button>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

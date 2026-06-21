/**
 * Template manager (WP-5) — create / edit / delete reusable article templates.
 * Replaces the three hardcoded scaffolds with stateful MSW-backed templates.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Plus, Pencil, Trash2, ArrowLeft } from 'lucide-react'
import type { WikiTemplate } from '@/types/wiki'
import {
  useTemplates,
  useCreateTemplate,
  useUpdateTemplate,
  useDeleteTemplate,
} from '@/api/hooks/useWiki'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { ConfirmDialog } from '@/components/shared'
import { RichTextEditor } from '@/components/shared/RichTextEditor'
import { cn } from '@/lib'
import { WIKI_ICON_MAP, WIKI_ICON_NAMES, WikiArticleIcon } from './wikiIdentity'

interface WikiTemplateManagerProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

interface FormState {
  name: string
  description: string
  icon: string
  content: string
}

const EMPTY_FORM: FormState = { name: '', description: '', icon: 'FileText', content: '<p></p>' }

export function WikiTemplateManager({ open, onOpenChange }: WikiTemplateManagerProps) {
  const { t } = useTranslation()
  const { data: templates } = useTemplates()
  const createTemplate = useCreateTemplate()
  const updateTemplate = useUpdateTemplate()
  const deleteTemplate = useDeleteTemplate()

  const [editingId, setEditingId] = useState<string | 'new' | null>(null)
  const [form, setForm] = useState<FormState>(EMPTY_FORM)
  const [deleteTarget, setDeleteTarget] = useState<WikiTemplate | null>(null)

  const startNew = () => {
    setForm(EMPTY_FORM)
    setEditingId('new')
  }

  const startEdit = (tpl: WikiTemplate) => {
    setForm({ name: tpl.name, description: tpl.description, icon: tpl.icon, content: tpl.content })
    setEditingId(tpl.id)
  }

  const backToList = () => setEditingId(null)

  const handleSave = async () => {
    if (!form.name.trim()) return
    try {
      if (editingId === 'new') {
        await createTemplate.mutateAsync({
          name: form.name.trim(),
          description: form.description.trim(),
          icon: form.icon,
          content: form.content,
          category: 'custom',
        })
        toast.success(t('wiki.template.created'))
      } else if (editingId) {
        await updateTemplate.mutateAsync({
          id: editingId,
          name: form.name.trim(),
          description: form.description.trim(),
          icon: form.icon,
          content: form.content,
        })
        toast.success(t('wiki.template.updated'))
      }
      setEditingId(null)
    } catch {
      toast.error(t('wiki.template.saveError'))
    }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await deleteTemplate.mutateAsync(deleteTarget.id)
      toast.success(t('wiki.template.deleted'))
    } catch {
      toast.error(t('wiki.template.deleteError'))
    }
    setDeleteTarget(null)
  }

  const list = templates ?? []
  const saving = createTemplate.isPending || updateTemplate.isPending

  return (
    <>
      <Dialog open={open} onOpenChange={(o) => { onOpenChange(o); if (!o) setEditingId(null) }}>
        <DialogContent className="max-w-xl">
          {editingId ? (
            // ---- Edit / create form ----
            <>
              <DialogHeader>
                <button
                  onClick={backToList}
                  className="mb-1 flex items-center gap-1 text-xs text-muted-foreground transition-colors hover:text-foreground"
                >
                  <ArrowLeft className="h-3.5 w-3.5" />
                  {t('wiki.template.backToList')}
                </button>
                <DialogTitle>
                  {editingId === 'new' ? t('wiki.template.newTitle') : t('wiki.template.editTitle')}
                </DialogTitle>
                <DialogDescription>{t('wiki.template.formDescription')}</DialogDescription>
              </DialogHeader>

              <div className="space-y-3 pt-1">
                <div className="flex gap-2">
                  <div className="flex-1">
                    <label className="mb-1 block text-xs font-medium text-foreground">{t('wiki.template.nameLabel')}</label>
                    <input
                      value={form.name}
                      onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                      placeholder={t('wiki.template.namePlaceholder')}
                      autoFocus
                      className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground focus:border-primary"
                    />
                  </div>
                </div>

                <div>
                  <label className="mb-1 block text-xs font-medium text-foreground">{t('wiki.template.descriptionLabel')}</label>
                  <input
                    value={form.description}
                    onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
                    placeholder={t('wiki.template.descriptionPlaceholder')}
                    className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground focus:border-primary"
                  />
                </div>

                <div>
                  <label className="mb-1 block text-xs font-medium text-foreground">{t('wiki.template.iconLabel')}</label>
                  <div className="grid grid-cols-9 gap-1">
                    {WIKI_ICON_NAMES.map((name) => {
                      const Icon = WIKI_ICON_MAP[name]
                      const active = form.icon === name
                      return (
                        <button
                          key={name}
                          type="button"
                          onClick={() => setForm((f) => ({ ...f, icon: name }))}
                          className={cn(
                            'flex h-7 w-7 items-center justify-center rounded-md transition-colors',
                            active ? 'bg-primary/15 text-primary' : 'text-muted-foreground hover:bg-secondary',
                          )}
                          aria-label={name}
                        >
                          <Icon className="h-3.5 w-3.5" />
                        </button>
                      )
                    })}
                  </div>
                </div>

                <div>
                  <label className="mb-1 block text-xs font-medium text-foreground">{t('wiki.template.contentLabel')}</label>
                  <RichTextEditor
                    content={form.content}
                    onChange={(html) => setForm((f) => ({ ...f, content: html }))}
                    compact
                    minHeight="120px"
                    maxHeight="220px"
                    placeholder={t('wiki.template.contentPlaceholder')}
                  />
                </div>

                <div className="flex justify-end gap-2 pt-1">
                  <button
                    onClick={backToList}
                    className="h-9 rounded-md border border-border px-4 text-sm text-foreground transition-colors hover:bg-accent"
                  >
                    {t('common.cancel')}
                  </button>
                  <button
                    onClick={handleSave}
                    disabled={!form.name.trim() || saving}
                    className="h-9 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
                  >
                    {t('common.save')}
                  </button>
                </div>
              </div>
            </>
          ) : (
            // ---- List ----
            <>
              <DialogHeader>
                <DialogTitle>{t('wiki.template.manageTitle')}</DialogTitle>
                <DialogDescription>{t('wiki.template.manageDescription')}</DialogDescription>
              </DialogHeader>

              <div className="space-y-1.5 pt-1">
                {list.map((tpl) => (
                  <div
                    key={tpl.id}
                    className="group flex items-center gap-3 rounded-lg border border-border px-3 py-2.5 transition-colors hover:border-primary/30"
                  >
                    <WikiArticleIcon icon={tpl.icon} title={tpl.name} size="md" />
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium text-foreground">{tpl.name}</p>
                      <p className="truncate text-xs text-muted-foreground">{tpl.description}</p>
                    </div>
                    <div className="flex shrink-0 items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100">
                      <button
                        onClick={() => startEdit(tpl)}
                        className="rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                        title={t('common.edit')}
                      >
                        <Pencil className="h-3.5 w-3.5" />
                      </button>
                      <button
                        onClick={() => setDeleteTarget(tpl)}
                        className="rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive"
                        title={t('common.delete')}
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  </div>
                ))}

                <button
                  onClick={startNew}
                  className="flex w-full items-center justify-center gap-2 rounded-lg border border-dashed border-border py-2.5 text-sm text-muted-foreground transition-colors hover:border-primary/40 hover:text-foreground"
                >
                  <Plus className="h-4 w-4" />
                  {t('wiki.template.newTemplate')}
                </button>
              </div>
            </>
          )}
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(o) => { if (!o) setDeleteTarget(null) }}
        title={t('wiki.template.deleteTitle')}
        description={t('wiki.template.deleteDescription', { name: deleteTarget?.name })}
        confirmLabel={t('common.delete')}
        variant="destructive"
        onConfirm={handleDelete}
      />
    </>
  )
}

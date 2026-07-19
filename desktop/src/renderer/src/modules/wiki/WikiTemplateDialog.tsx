import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Settings2, FileText } from 'lucide-react'
import { toast } from 'sonner'
import { sanitizeHtml } from '@/lib/sanitize'
import { useWikiStore } from '@/stores/wiki'
import { useCreateArticle, useTemplates } from '@/api/hooks/useWiki'
import { useHasCapability } from '@/hooks/useCapability'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { cn } from '@/lib'
import { WikiArticleIcon } from './wikiIdentity'
import { WikiTemplateManager } from './WikiTemplateManager'

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface WikiTemplateDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function WikiTemplateDialog({ open, onOpenChange }: WikiTemplateDialogProps) {
  const { t } = useTranslation()
  const createArticle = useCreateArticle()
  const { data: templates } = useTemplates()
  const selectedCategoryId = useWikiStore((s) => s.selectedCategoryId)
  const canManageTemplates = useHasCapability('wiki:template:manage')

  const [title, setTitle] = useState('')
  // null = blank document; otherwise the picked template id.
  const [selectedTemplate, setSelectedTemplate] = useState<string | null>(null)
  const [managerOpen, setManagerOpen] = useState(false)

  const tpl = templates?.find((tm) => tm.id === selectedTemplate) ?? null

  const handleCreate = async () => {
    if (!title.trim()) return
    const content = tpl?.content ?? `<p>${t('wiki.article.defaultContent')}</p>`
    try {
      await createArticle.mutateAsync({
        title: title.trim(),
        content: { html: content } as Record<string, unknown>,
        icon: tpl?.icon,
        category_id: selectedCategoryId ?? undefined,
        published: false,
      })
      toast.success(t('wiki.article.created'))
      onOpenChange(false)
      setTitle('')
      setSelectedTemplate(null)
    } catch {
      toast.error(t('wiki.article.createError'))
    }
  }

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <div className="flex items-start justify-between gap-3">
              <div>
                <DialogTitle>{t('wiki.article.newTitle')}</DialogTitle>
                <DialogDescription>{t('wiki.article.newDescription')}</DialogDescription>
              </div>
              {canManageTemplates && (
                <button
                  onClick={() => setManagerOpen(true)}
                  className="flex shrink-0 items-center gap-1.5 rounded-md border border-border px-2.5 py-1.5 text-xs text-muted-foreground transition-colors hover:border-primary/40 hover:text-foreground"
                >
                  <Settings2 className="h-3.5 w-3.5" />
                  {t('wiki.template.manage')}
                </button>
              )}
            </div>
          </DialogHeader>

          {/* Title */}
          <div className="pt-1">
            <label className="mb-1 block text-sm font-medium text-foreground">{t('wiki.article.titleLabel')}</label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder={t('wiki.article.titlePlaceholder')}
              className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground focus:border-primary"
              autoFocus
              onKeyDown={(e) => { if (e.key === 'Enter') handleCreate() }}
            />
          </div>

          {/* Template picker + preview */}
          <div className="grid grid-cols-[1fr_1.15fr] gap-3">
            {/* Template list */}
            <div className="space-y-1">
              <p className="mb-1 text-xs font-medium text-foreground">{t('wiki.article.templateLabel')}</p>
              <div className="max-h-72 space-y-1 overflow-y-auto pr-1">
                {/* Blank */}
                <button
                  onClick={() => setSelectedTemplate(null)}
                  className={cn(
                    'flex w-full items-center gap-2.5 rounded-lg border px-2.5 py-2 text-left transition-colors',
                    selectedTemplate === null
                      ? 'border-primary bg-primary/5'
                      : 'border-border hover:bg-accent',
                  )}
                >
                  <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-secondary text-muted-foreground">
                    <FileText className="h-4 w-4" />
                  </span>
                  <span className="min-w-0">
                    <span className="block truncate text-sm font-medium text-foreground">{t('wiki.template.blank')}</span>
                    <span className="block truncate text-[11px] text-muted-foreground">{t('wiki.template.blankHint')}</span>
                  </span>
                </button>

                {(templates ?? []).map((tm) => {
                  const active = selectedTemplate === tm.id
                  return (
                    <button
                      key={tm.id}
                      onClick={() => setSelectedTemplate(tm.id)}
                      className={cn(
                        'flex w-full items-center gap-2.5 rounded-lg border px-2.5 py-2 text-left transition-colors',
                        active ? 'border-primary bg-primary/5' : 'border-border hover:bg-accent',
                      )}
                    >
                      <WikiArticleIcon icon={tm.icon} title={tm.name} size="md" />
                      <span className="min-w-0">
                        <span className="block truncate text-sm font-medium text-foreground">{tm.name}</span>
                        <span className="block truncate text-[11px] text-muted-foreground">{tm.description}</span>
                      </span>
                    </button>
                  )
                })}
              </div>
            </div>

            {/* Preview */}
            <div className="flex flex-col">
              <p className="mb-1 text-xs font-medium text-foreground">{t('wiki.template.preview')}</p>
              <div className="max-h-72 flex-1 overflow-y-auto rounded-lg border border-border bg-secondary/20 p-3">
                {tpl ? (
                  <div
                    className="tiptap-content prose prose-sm max-w-none dark:prose-invert"
                    dangerouslySetInnerHTML={{
                      __html: sanitizeHtml(tpl.content, {
                        ADD_TAGS: ['details', 'summary', 'figure', 'figcaption'],
                        ADD_ATTR: ['data-callout', 'data-variant', 'data-details', 'data-details-content', 'open'],
                      }),
                    }}
                  />
                ) : (
                  <div className="flex h-full min-h-32 flex-col items-center justify-center gap-1.5 text-center">
                    <FileText className="h-6 w-6 text-muted-foreground/40" />
                    <p className="text-xs text-muted-foreground">{t('wiki.template.blankPreview')}</p>
                  </div>
                )}
              </div>
            </div>
          </div>

          {/* Actions */}
          <div className="flex justify-end gap-2 pt-1">
            <button
              onClick={() => { onOpenChange(false); setTitle(''); setSelectedTemplate(null) }}
              className="h-9 rounded-md border border-border px-4 text-sm text-foreground transition-colors hover:bg-accent"
            >
              {t('common.cancel')}
            </button>
            <button
              onClick={handleCreate}
              disabled={!title.trim() || createArticle.isPending}
              className="h-9 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
            >
              {t('common.create')}
            </button>
          </div>
        </DialogContent>
      </Dialog>

      <WikiTemplateManager open={managerOpen} onOpenChange={setManagerOpen} />
    </>
  )
}

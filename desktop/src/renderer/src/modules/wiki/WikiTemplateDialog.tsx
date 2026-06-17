import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { FileText, AlertTriangle, BookOpen } from 'lucide-react'
import { toast } from 'sonner'
import { WIKI_TEMPLATES, useWikiStore } from '@/stores/wiki'
import { useCreateArticle } from '@/api/hooks/useWiki'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'

// ---------------------------------------------------------------------------
// Icon map for templates
// ---------------------------------------------------------------------------

const templateIconMap: Record<string, typeof FileText> = {
  FileText,
  AlertTriangle,
  BookOpen,
}

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
  const selectedCategoryId = useWikiStore((s) => s.selectedCategoryId)

  const [title, setTitle] = useState('')
  const [selectedTemplate, setSelectedTemplate] = useState<string | null>(null)

  const handleCreate = async () => {
    if (!title.trim()) return
    const template = selectedTemplate ? WIKI_TEMPLATES.find((tmpl) => tmpl.id === selectedTemplate) : null
    const content = template?.content ?? `<p>${t('wiki.article.defaultContent')}</p>`
    try {
      await createArticle.mutateAsync({
        title: title.trim(),
        content: { plain: content } as Record<string, unknown>,
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
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{t('wiki.article.newTitle')}</DialogTitle>
          <DialogDescription>
            {t('wiki.article.newDescription')}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 pt-2">
          {/* Title */}
          <div>
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

          {/* Template selection */}
          <div>
            <label className="mb-1.5 block text-sm font-medium text-foreground">
              {t('wiki.article.templateLabel')} <span className="text-muted-foreground font-normal">({t('common.optional')})</span>
            </label>
            <div className="grid grid-cols-3 gap-2">
              {WIKI_TEMPLATES.map((tmpl) => {
                const TIcon = templateIconMap[tmpl.icon] ?? FileText
                const isActive = selectedTemplate === tmpl.id
                return (
                  <button
                    key={tmpl.id}
                    onClick={() => setSelectedTemplate(isActive ? null : tmpl.id)}
                    className={`flex flex-col items-center gap-1.5 rounded-lg border px-3 py-3 text-center transition-colors ${
                      isActive
                        ? 'border-primary bg-primary/5 text-primary'
                        : 'border-border text-muted-foreground hover:bg-accent'
                    }`}
                  >
                    <TIcon className="h-5 w-5" />
                    <span className="text-xs font-medium">{tmpl.name}</span>
                    <span className="text-[10px] leading-tight">{tmpl.description}</span>
                  </button>
                )
              })}
            </div>
          </div>

          {/* Actions */}
          <div className="flex justify-end gap-2 pt-1">
            <button
              onClick={() => { onOpenChange(false); setTitle(''); setSelectedTemplate(null) }}
              className="h-9 rounded-md border border-border px-4 text-sm text-foreground hover:bg-accent transition-colors"
            >
              {t('common.cancel')}
            </button>
            <button
              onClick={handleCreate}
              disabled={!title.trim()}
              className="h-9 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              {t('common.create')}
            </button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

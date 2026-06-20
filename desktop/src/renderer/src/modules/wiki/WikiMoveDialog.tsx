import { useState, useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { FolderInput, Check, Loader2, BookOpen } from 'lucide-react'
import { toast } from 'sonner'
import type { WikiArticle, WikiCategory } from '@/types/wiki'
import { useUpdateArticle } from '@/api/hooks/useWiki'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'

// ---------------------------------------------------------------------------
// Helpers — flatten the category tree into a depth-ordered list
// ---------------------------------------------------------------------------

interface FlatCategory {
  category: WikiCategory
  depth: number
}

function flattenCategories(categories: WikiCategory[]): FlatCategory[] {
  const out: FlatCategory[] = []
  const walk = (parentId: string | null, depth: number) => {
    categories
      .filter((c) => (c.parentId ?? null) === parentId)
      .sort((a, b) => a.sortOrder - b.sortOrder)
      .forEach((category) => {
        out.push({ category, depth })
        walk(category.id, depth + 1)
      })
  }
  walk(null, 0)
  return out
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface WikiMoveDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  article: WikiArticle | null
  categories: WikiCategory[]
}

export function WikiMoveDialog({ open, onOpenChange, article, categories }: WikiMoveDialogProps) {
  const { t } = useTranslation()
  const updateArticle = useUpdateArticle()
  // null sentinel = "no category"; undefined = nothing chosen yet.
  const [target, setTarget] = useState<string | null>(null)

  const flat = useMemo(() => flattenCategories(categories), [categories])

  const articleId = article?.id
  const currentCategoryId = article?.categoryId || null
  useEffect(() => {
    setTarget(currentCategoryId)
  }, [articleId, currentCategoryId])

  if (!article) return null

  const handleConfirm = async () => {
    if (target === currentCategoryId) {
      onOpenChange(false)
      return
    }
    try {
      await updateArticle.mutateAsync({ id: article.id, category_id: target })
      toast.success(t('wiki.move.moved'))
      onOpenChange(false)
    } catch {
      toast.error(t('wiki.move.moveError'))
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>{t('wiki.move.title')}</DialogTitle>
          <DialogDescription>
            {t('wiki.move.description', { title: article.title })}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-1 pt-2 max-h-72 overflow-y-auto">
          {/* No category */}
          <button
            onClick={() => setTarget(null)}
            className={`flex w-full items-center gap-2.5 rounded-md border px-3 py-2 text-left transition-colors ${
              target === null ? 'border-primary bg-primary/5' : 'border-border hover:bg-accent'
            }`}
          >
            <BookOpen className={`h-4 w-4 shrink-0 ${target === null ? 'text-primary' : 'text-muted-foreground'}`} />
            <span className={`flex-1 text-sm ${target === null ? 'text-primary font-medium' : 'text-foreground'}`}>
              {t('wiki.move.none')}
            </span>
            {target === null && <Check className="h-3.5 w-3.5 shrink-0 text-primary" />}
          </button>

          {/* Category tree */}
          {flat.map(({ category, depth }) => {
            const isActive = target === category.id
            const isCurrent = currentCategoryId === category.id
            return (
              <button
                key={category.id}
                onClick={() => setTarget(category.id)}
                style={{ paddingLeft: 12 + depth * 16 }}
                className={`flex w-full items-center gap-2.5 rounded-md border py-2 pr-3 text-left transition-colors ${
                  isActive ? 'border-primary bg-primary/5' : 'border-border hover:bg-accent'
                }`}
              >
                <FolderInput className={`h-4 w-4 shrink-0 ${isActive ? 'text-primary' : 'text-muted-foreground'}`} />
                <span className={`min-w-0 flex-1 truncate text-sm ${isActive ? 'text-primary font-medium' : 'text-foreground'}`}>
                  {category.name}
                </span>
                {isCurrent && (
                  <span className="shrink-0 rounded-full bg-secondary px-1.5 py-0.5 text-[10px] text-muted-foreground">
                    {t('wiki.move.current')}
                  </span>
                )}
                {isActive && <Check className="h-3.5 w-3.5 shrink-0 text-primary" />}
              </button>
            )
          })}
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-2 pt-2">
          <button
            onClick={() => onOpenChange(false)}
            className="h-9 rounded-md border border-border px-4 text-sm text-foreground hover:bg-accent transition-colors"
          >
            {t('common.cancel')}
          </button>
          <button
            onClick={handleConfirm}
            disabled={updateArticle.isPending}
            className="inline-flex h-9 items-center gap-2 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
          >
            {updateArticle.isPending && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
            {t('wiki.move.confirm')}
          </button>
        </div>
      </DialogContent>
    </Dialog>
  )
}

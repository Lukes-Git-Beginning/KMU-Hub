import { useState } from 'react'
import {
  Pin,
  Send,
  Archive,
  Edit3,
  Trash2,
  History,
  Share2,
  Eye,
  Clock,
  FolderInput,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import type { WikiArticle, WikiCategory } from '@/types/wiki'
import { useUpdateArticle } from '@/api/hooks/useWiki'
import { useWikiStore } from '@/stores/wiki'
import { ItemActions } from '@/components/shared'
import { formatDate as libFormatDate } from '@/lib/format'
import { WikiMoveDialog } from './WikiMoveDialog'
import { WikiTagEditor } from './WikiTagEditor'
import { WikiArticleIcon, coverBackground } from './wikiIdentity'

// ---------------------------------------------------------------------------
// Status config
// ---------------------------------------------------------------------------

const statusConfig: Record<string, { key: string; bg: string }> = {
  draft: { key: 'wiki.status.draft', bg: 'bg-secondary text-muted-foreground' },
  published: { key: 'wiki.status.published', bg: 'bg-success-light text-success' },
  archived: { key: 'wiki.status.archived', bg: 'bg-warning-light text-warning' },
}

function formatShortDate(dateStr: string): string {
  const normalized = dateStr.includes('T') ? dateStr : dateStr + 'T00:00:00'
  return libFormatDate(normalized, { day: '2-digit', month: 'short', year: 'numeric' })
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface WikiArticleHeaderProps {
  article: WikiArticle
  /** All categories — needed for the move dialog. */
  categories: WikiCategory[]
  /** All tags across articles — drives the tag-editor suggestions. */
  allTags: string[]
  /** Live version count from the per-article versions query. */
  versionCount?: number
  /** Live view count (incremented on detail GET). */
  viewCount?: number
  onEdit: () => void
  onDelete: () => void
  onToggleVersions: () => void
  onShare: () => void
}

export function WikiArticleHeader({
  article,
  categories,
  allTags,
  versionCount,
  viewCount,
  onEdit,
  onDelete,
  onToggleVersions,
  onShare,
}: WikiArticleHeaderProps) {
  const { t } = useTranslation()
  const updateMutation = useUpdateArticle()
  const togglePin = useWikiStore((s) => s.togglePin)
  const [moveOpen, setMoveOpen] = useState(false)
  const [editingTitle, setEditingTitle] = useState(false)
  const [draftTitle, setDraftTitle] = useState(article.title)

  const st = statusConfig[article.status]
  const coverBg = coverBackground(article.coverUrl)

  const commitTitle = async () => {
    const title = draftTitle.trim()
    setEditingTitle(false)
    if (!title || title === article.title) {
      setDraftTitle(article.title)
      return
    }
    try {
      await updateMutation.mutateAsync({ id: article.id, title })
      toast.success(t('wiki.header.titleRenamed'))
    } catch {
      setDraftTitle(article.title)
      toast.error(t('wiki.header.renameError'))
    }
  }

  return (
    <div className="border-b border-border">
      {/* Cover banner (WP-3) */}
      {coverBg && <div className="h-24 w-full" style={{ background: coverBg }} />}

      <div className="px-5 py-3">
        {/* Row 1: Icon + Title + actions */}
        <div className="flex items-start gap-3">
          <WikiArticleIcon
            icon={article.icon}
            title={article.title}
            size="md"
            className={coverBg ? '-mt-7 shadow-sm ring-2 ring-card' : 'mt-0.5'}
          />
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              {article.isPinned && <Pin className="h-3.5 w-3.5 shrink-0 text-primary" />}
            {editingTitle ? (
              <input
                value={draftTitle}
                onChange={(e) => setDraftTitle(e.target.value)}
                onBlur={commitTitle}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') commitTitle()
                  if (e.key === 'Escape') {
                    setDraftTitle(article.title)
                    setEditingTitle(false)
                  }
                }}
                autoFocus
                aria-label={t('wiki.header.renameTitle')}
                className="min-w-0 flex-1 rounded border border-primary/50 bg-background px-1.5 py-0.5 text-lg font-semibold text-foreground outline-none"
              />
            ) : (
              <button
                onClick={() => {
                  setDraftTitle(article.title)
                  setEditingTitle(true)
                }}
                title={t('wiki.header.renameTitle')}
                className="min-w-0 truncate rounded text-left text-lg font-semibold text-foreground hover:bg-accent/50 px-1 -mx-1 transition-colors"
              >
                {article.title}
              </button>
            )}
            <span className={`shrink-0 rounded-full px-2 py-0.5 text-[10px] font-medium ${st?.bg ?? ''}`}>
              {st ? t(st.key) : article.status}
            </span>
          </div>

          {/* Row 2: Meta info */}
          <div className="mt-1 flex items-center gap-2 text-xs text-muted-foreground flex-wrap">
            <span>{article.authorName}</span>
            <span>·</span>
            <Clock className="h-3 w-3" />
            <span>{formatShortDate(article.lastEditedAt)}</span>
            <span>·</span>
            <Eye className="h-3 w-3" />
            <span>{t('wiki.header.views', { count: viewCount ?? article.viewCount })}</span>
            <span>·</span>
            <History className="h-3 w-3" />
            <span>v{versionCount ?? (article.versions ?? []).length}</span>
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-1 shrink-0">
          <button
            onClick={onEdit}
            className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
            title={t('common.edit')}
          >
            <Edit3 className="h-4 w-4" />
          </button>
          <button
            onClick={onToggleVersions}
            className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
            title={t('wiki.header.versionHistory')}
          >
            <History className="h-4 w-4" />
          </button>
          <button
            onClick={onShare}
            className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
            title={t('wiki.header.share')}
          >
            <Share2 className="h-4 w-4" />
          </button>
          <ItemActions
            items={[
              {
                label: article.isPinned ? t('wiki.actions.unpin') : t('wiki.actions.pin'),
                icon: Pin,
                onClick: () => {
                  togglePin(article.id)
                  toast.success(article.isPinned ? t('wiki.actions.unpinned') : t('wiki.actions.pinned'))
                },
              },
              { label: t('wiki.actions.move'), icon: FolderInput, onClick: () => setMoveOpen(true) },
              ...(article.status === 'draft'
                ? [{
                    label: t('wiki.actions.publish'),
                    icon: Send,
                    onClick: () => {
                      updateMutation.mutate({ id: article.id, published: true })
                      toast.success(t('wiki.actions.published'))
                    },
                  }]
                : []),
              ...(article.status === 'published'
                ? [{
                    label: t('wiki.actions.archive'),
                    icon: Archive,
                    // archive = unpublish (closest available API equivalent)
                    onClick: () => {
                      updateMutation.mutate({ id: article.id, published: false })
                      toast.success(t('wiki.actions.archived'))
                    },
                  }]
                : []),
              { label: t('common.delete'), icon: Trash2, onClick: onDelete, variant: 'destructive' as const },
            ]}
          />
        </div>
      </div>

        {/* Tags (editable) */}
        <WikiTagEditor articleId={article.id} tags={article.tags ?? []} suggestions={allTags} />
      </div>

      <WikiMoveDialog
        open={moveOpen}
        onOpenChange={setMoveOpen}
        article={article}
        categories={categories}
      />
    </div>
  )
}

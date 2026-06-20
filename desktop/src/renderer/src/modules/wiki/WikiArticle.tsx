import { useState } from 'react'
import { sanitizeHtml } from '@/lib/sanitize'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { FileText } from 'lucide-react'
import type { WikiArticle as WikiArticleType } from '@/types/wiki'
import { useWikiStore } from '@/stores/wiki'
import { useUpdateArticle, useArticleVersions, useRestoreVersion } from '@/api/hooks/useWiki'
import { adaptVersion } from '@/api/wiki-adapter'
import { WikiArticleHeader } from './WikiArticleHeader'
import { WikiEditor } from './WikiEditor'
import { WikiVersionHistory } from './WikiVersionHistory'
import { WikiAttachments } from './WikiAttachments'

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface WikiArticleProps {
  article: WikiArticleType
  onDelete: () => void
  onShare: () => void
}

export function WikiArticle({ article, onDelete, onShare }: WikiArticleProps) {
  const { t } = useTranslation()
  const isEditing = useWikiStore((s) => s.isEditing)
  const setEditing = useWikiStore((s) => s.setEditing)
  const setSelectedArticle = useWikiStore((s) => s.setSelectedArticle)
  const updateArticleMutation = useUpdateArticle()
  const restoreVersionMutation = useRestoreVersion()
  const [editContent, setEditContent] = useState(article.content)
  const [showVersions, setShowVersions] = useState(false)

  // Version history is loaded per-article via its own query.
  const { data: versionsData } = useArticleVersions(article.id)
  const versions = (versionsData ?? [])
    .map(adaptVersion)
    .sort((a, b) => b.version - a.version)

  const handleSave = async () => {
    try {
      await updateArticleMutation.mutateAsync({
        id: article.id,
        content: { html: editContent } as Record<string, unknown>,
      })
      setEditing(false)
      toast.success(t('wiki.article.saved'))
    } catch {
      toast.error(t('wiki.article.saveError'))
    }
  }

  const handleCancel = () => {
    setEditContent(article.content)
    setEditing(false)
  }

  const handleStartEdit = () => {
    setEditContent(article.content)
    setEditing(true)
  }

  // Internal [[wiki link]] clicks open the target article.
  const handleBodyClick = (e: React.MouseEvent<HTMLDivElement>) => {
    const el = (e.target as HTMLElement).closest('[data-wiki-link]')
    if (!el) return
    const targetId = el.getAttribute('data-article-id')
    if (targetId) {
      e.preventDefault()
      setSelectedArticle(targetId)
      setEditing(false)
    }
  }

  const handleRestore = async (versionId: string) => {
    try {
      await restoreVersionMutation.mutateAsync({ articleId: article.id, versionId })
      toast.success(t('wiki.version.restored'))
    } catch {
      toast.error(t('wiki.version.restoreError'))
    }
  }

  return (
    <div className="flex flex-1 overflow-hidden">
      {/* Article content */}
      <div className="flex flex-1 flex-col overflow-hidden">
        {/* Header */}
        <WikiArticleHeader
          article={article}
          versionCount={versions.length}
          onEdit={handleStartEdit}
          onDelete={onDelete}
          onToggleVersions={() => setShowVersions(!showVersions)}
          onShare={onShare}
        />

        {/* Content / Editor */}
        {isEditing ? (
          <WikiEditor
            key={article.id}
            content={editContent}
            onChange={setEditContent}
            onSave={handleSave}
            onCancel={handleCancel}
            saving={updateArticleMutation.isPending}
          />
        ) : (
          <div className="flex-1 overflow-y-auto px-5 py-4">
            {article.content.trim() ? (
              <div
                className="prose prose-sm max-w-none dark:prose-invert"
                onClick={handleBodyClick}
                dangerouslySetInnerHTML={{
                  __html: sanitizeHtml(article.content, {
                    ADD_ATTR: ['data-wiki-link', 'data-article-id', 'data-slug', 'data-mention-id'],
                  }),
                }}
              />
            ) : (
              <div className="flex flex-col items-center justify-center gap-2 py-10 text-center">
                <FileText className="h-8 w-8 text-muted-foreground/40" />
                <p className="text-sm text-muted-foreground">{t('wiki.article.empty')}</p>
                <button
                  onClick={handleStartEdit}
                  className="mt-1 h-8 rounded-md border border-border px-3 text-xs text-foreground hover:bg-accent transition-colors"
                >
                  {t('wiki.article.startWriting')}
                </button>
              </div>
            )}

            {/* Attachments */}
            <WikiAttachments articleId={article.id} />
          </div>
        )}
      </div>

      {/* Version history slide-in */}
      <WikiVersionHistory
        versions={versions}
        open={showVersions}
        onClose={() => setShowVersions(false)}
        onRestore={handleRestore}
        restoringId={restoreVersionMutation.isPending ? restoreVersionMutation.variables?.versionId : undefined}
      />
    </div>
  )
}

import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { cn } from '@/lib'
import type { WikiArticle as WikiArticleType, WikiCategory } from '@/types/wiki'
import { useWikiStore } from '@/stores/wiki'
import { useWikiPrefsStore } from '@/stores/wikiPrefs'
import { useUpdateArticle, useArticleVersions, useRestoreVersion, useArticle } from '@/api/hooks/useWiki'
import { adaptVersion } from '@/api/wiki-adapter'
import { DocumentReader, type DocRow } from '@/components/shared/document'
import { wikiBlockRegistry } from './wiki-blocks'
import { useWikiUsers } from './useWikiUsers'
import { WikiEmptyCanvas } from './WikiEmptyCanvas'
import { WikiArticleHeader } from './WikiArticleHeader'
import { WikiEditor } from './WikiEditor'
import { WikiVersionHistory } from './WikiVersionHistory'
import { WikiAttachments } from './WikiAttachments'

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface WikiArticleProps {
  article: WikiArticleType
  categories: WikiCategory[]
  /** All tags across articles — drives the tag-editor suggestions. */
  allTags: string[]
  onDelete: () => void
  onShare: () => void
}

export function WikiArticle({ article, categories, allTags, onDelete, onShare }: WikiArticleProps) {
  const { t } = useTranslation()
  const isEditing = useWikiStore((s) => s.isEditing)
  const setEditing = useWikiStore((s) => s.setEditing)
  const readingWidth = useWikiPrefsStore((s) => s.readingWidth)
  const updateArticleMutation = useUpdateArticle()
  const restoreVersionMutation = useRestoreVersion()
  const [editRows, setEditRows] = useState<DocRow[]>(article.body)
  const [editTitle, setEditTitle] = useState(article.title)
  const [editIcon, setEditIcon] = useState(article.icon)
  const [editCover, setEditCover] = useState(article.coverUrl)
  const [showVersions, setShowVersions] = useState(false)

  const resolveUser = useWikiUsers()

  // Detail GET increments the view counter server-side; use its live count.
  const { data: detail } = useArticle(article.id)
  const viewCount = detail?.view_count ?? article.viewCount

  // Version history is loaded per-article via its own query.
  const { data: versionsData } = useArticleVersions(article.id)
  const versions = (versionsData ?? [])
    .map(adaptVersion)
    .map((v) => ({ ...v, editorName: resolveUser(v.editorName) || v.editorName }))
    .sort((a, b) => b.version - a.version)

  const handleSave = async (changeNote?: string) => {
    const title = editTitle.trim()
    try {
      await updateArticleMutation.mutateAsync({
        id: article.id,
        // Persist the block document as the JSONB content field.
        content: { rows: editRows } as Record<string, unknown>,
        // Persist the title + identity alongside the body when they changed
        // (null clears an icon/cover the writer removed).
        ...(title && title !== article.title ? { title } : {}),
        ...(editIcon !== article.icon ? { icon: editIcon ?? null } : {}),
        ...(editCover !== article.coverUrl ? { cover_url: editCover ?? null } : {}),
        change_note: changeNote?.trim() || undefined,
      })
      setEditing(false)
      toast.success(t('wiki.article.saved'))
    } catch {
      toast.error(t('wiki.article.saveError'))
    }
  }

  const handleCancel = () => {
    setEditRows(article.body)
    setEditTitle(article.title)
    setEditIcon(article.icon)
    setEditCover(article.coverUrl)
    setEditing(false)
  }

  const handleStartEdit = () => {
    setEditRows(article.body)
    setEditTitle(article.title)
    setEditIcon(article.icon)
    setEditCover(article.coverUrl)
    setEditing(true)
  }

  const handleRestore = async (versionId: string) => {
    try {
      await restoreVersionMutation.mutateAsync({ articleId: article.id, versionId })
      toast.success(t('wiki.version.restored'))
    } catch {
      toast.error(t('wiki.version.restoreError'))
    }
  }

  const hasBody = article.body.length > 0

  return (
    <div className="flex flex-1 overflow-hidden">
      {/* Article content */}
      <div className="flex flex-1 flex-col overflow-hidden">
        {/* Header — read mode only; the editor carries its own title heading. */}
        {!isEditing && (
          <WikiArticleHeader
            article={article}
            categories={categories}
            allTags={allTags}
            versionCount={versions.length}
            viewCount={viewCount}
            onEdit={handleStartEdit}
            onDelete={onDelete}
            onToggleVersions={() => setShowVersions(!showVersions)}
            onShare={onShare}
          />
        )}

        {/* Content / Editor */}
        {isEditing ? (
          <WikiEditor
            key={article.id}
            title={editTitle}
            onTitleChange={setEditTitle}
            rows={editRows}
            onChange={setEditRows}
            icon={editIcon}
            coverUrl={editCover}
            onIconChange={setEditIcon}
            onCoverChange={setEditCover}
            onSave={handleSave}
            onCancel={handleCancel}
            saving={updateArticleMutation.isPending}
          />
        ) : (
          <div className="flex-1 overflow-y-auto px-6 py-6">
            <div className={cn('mx-auto', readingWidth === 'wide' ? 'max-w-none' : 'max-w-[72ch]')}>
              {hasBody ? (
                <DocumentReader rows={article.body} registry={wikiBlockRegistry} mode="web" />
              ) : (
                <WikiEmptyCanvas onStart={handleStartEdit} />
              )}

              {/* Attachments */}
              <WikiAttachments articleId={article.id} />
            </div>
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

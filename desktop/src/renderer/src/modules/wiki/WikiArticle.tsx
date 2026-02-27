import { useState } from 'react'
import { toast } from 'sonner'
import type { WikiArticle as WikiArticleType } from '@/types/wiki'
import { useWikiStore } from '@/stores/wiki'
import { WikiArticleHeader } from './WikiArticleHeader'
import { WikiEditor } from './WikiEditor'
import { WikiVersionHistory } from './WikiVersionHistory'

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface WikiArticleProps {
  article: WikiArticleType
  onDelete: () => void
  onShare: () => void
}

export function WikiArticle({ article, onDelete, onShare }: WikiArticleProps) {
  const updateArticle = useWikiStore((s) => s.updateArticle)
  const isEditing = useWikiStore((s) => s.isEditing)
  const setEditing = useWikiStore((s) => s.setEditing)
  const [editContent, setEditContent] = useState(article.content)
  const [showVersions, setShowVersions] = useState(false)

  const handleSave = () => {
    updateArticle(article.id, {
      content: editContent,
      lastEditedBy: 'Du',
    })
    setEditing(false)
    toast.success('Artikel gespeichert')
  }

  const handleCancel = () => {
    setEditContent(article.content)
    setEditing(false)
  }

  const handleStartEdit = () => {
    setEditContent(article.content)
    setEditing(true)
  }

  return (
    <div className="flex flex-1 overflow-hidden">
      {/* Article content */}
      <div className="flex flex-1 flex-col overflow-hidden">
        {/* Header */}
        <WikiArticleHeader
          article={article}
          onEdit={handleStartEdit}
          onDelete={onDelete}
          onToggleVersions={() => setShowVersions(!showVersions)}
          onShare={onShare}
        />

        {/* Content / Editor */}
        {isEditing ? (
          <WikiEditor
            content={editContent}
            onChange={setEditContent}
            onSave={handleSave}
            onCancel={handleCancel}
          />
        ) : (
          <div className="flex-1 overflow-y-auto px-5 py-4">
            <div
              className="prose prose-sm max-w-none dark:prose-invert"
              dangerouslySetInnerHTML={{ __html: article.content }}
            />
          </div>
        )}
      </div>

      {/* Version history slide-in */}
      <WikiVersionHistory
        versions={article.versions}
        open={showVersions}
        onClose={() => setShowVersions(false)}
      />
    </div>
  )
}

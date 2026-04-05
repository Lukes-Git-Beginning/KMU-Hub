/**
 * Wiki — Internal Knowledge Base.
 *
 * Three-column layout:
 *   Left  (w-56):   WikiSidebar — tree navigation, search, category management
 *   Center (flex-1): Article list or article detail (read/edit mode)
 *   Right  (w-64):   WikiVersionHistory — slide-in panel (when toggled)
 *
 * All data from useWikiStore (mock). Backend swap: replace store reads
 * with TanStack Query hooks, keep components identical.
 */
import { useState, useMemo } from 'react'
import {
  BookOpen,
  Pin,
  Eye,
  History,
  Tag,
  Clock,
} from 'lucide-react'
import { useWikiStore } from '@/stores/wiki'
import type { WikiArticle as WikiArticleType } from '@/types/wiki'
import { EmptyState, ConfirmDialog } from '@/components/shared'
import { WikiSidebar } from './WikiSidebar'
import { WikiArticle } from './WikiArticle'
import { WikiTemplateDialog } from './WikiTemplateDialog'
import { WikiCategoryDialog } from './WikiCategoryDialog'
import { WikiShareDialog } from './WikiShareDialog'

// ---------------------------------------------------------------------------
// Status config
// ---------------------------------------------------------------------------

const statusConfig: Record<string, { label: string; bg: string }> = {
  draft: { label: 'Entwurf', bg: 'bg-secondary text-muted-foreground' },
  published: { label: 'Veröffentlicht', bg: 'bg-success-light text-success' },
  archived: { label: 'Archiviert', bg: 'bg-warning-light text-warning' },
}

function formatDate(dateStr: string): string {
  try {
    return new Date(dateStr + (dateStr.includes('T') ? '' : 'T00:00:00')).toLocaleDateString('de-DE')
  } catch {
    return dateStr
  }
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function WikiPage() {
  const articles = useWikiStore((s) => s.articles)
  const selectedArticleId = useWikiStore((s) => s.selectedArticleId)
  const selectedCategoryId = useWikiStore((s) => s.selectedCategoryId)
  const searchQuery = useWikiStore((s) => s.searchQuery)
  const setSelectedArticle = useWikiStore((s) => s.setSelectedArticle)
  const deleteArticle = useWikiStore((s) => s.deleteArticle)
  const incrementViewCount = useWikiStore((s) => s.incrementViewCount)

  // Dialogs
  const [showNewArticle, setShowNewArticle] = useState(false)
  const [showNewCategory, setShowNewCategory] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<WikiArticleType | null>(null)
  const [shareTarget, setShareTarget] = useState<WikiArticleType | null>(null)

  // Filtered + sorted articles
  const filteredArticles = useMemo(() => {
    let result = articles
    if (selectedCategoryId) {
      result = result.filter((a) => a.categoryId === selectedCategoryId)
    }
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase()
      result = result.filter(
        (a) =>
          a.title.toLowerCase().includes(q) ||
          (a.tags ?? []).some((t) => t.toLowerCase().includes(q)) ||
          a.authorName.toLowerCase().includes(q),
      )
    }
    return result.sort((a, b) => {
      if (a.isPinned !== b.isPinned) return a.isPinned ? -1 : 1
      return b.lastEditedAt.localeCompare(a.lastEditedAt)
    })
  }, [articles, selectedCategoryId, searchQuery])

  const selectedArticle = articles.find((a) => a.id === selectedArticleId) ?? null

  const handleSelectArticle = (article: WikiArticleType) => {
    setSelectedArticle(article.id)
    incrementViewCount(article.id)
  }

  const handleDelete = () => {
    if (!deleteTarget) return
    deleteArticle(deleteTarget.id)
    if (selectedArticleId === deleteTarget.id) {
      setSelectedArticle(null)
    }
    setDeleteTarget(null)
  }

  return (
    <>
      <div className="flex h-full overflow-hidden animate-fade-up">
        {/* Left: Sidebar with tree navigation */}
        <WikiSidebar
          onNewArticle={() => setShowNewArticle(true)}
          onNewCategory={() => setShowNewCategory(true)}
        />

        {/* Center: Article list + detail */}
        <div className="flex flex-1 overflow-hidden">
          {/* Article list */}
          <div className={`flex flex-col overflow-hidden ${selectedArticle ? 'w-80 shrink-0 border-r border-border' : 'flex-1'}`}>
            {/* List header */}
            <div className="flex items-center justify-between px-3 py-2.5 border-b border-border">
              <span className="text-xs font-medium text-muted-foreground">
                {filteredArticles.length} Artikel
              </span>
            </div>

            {/* List */}
            <div className="flex-1 overflow-y-auto">
              {filteredArticles.length === 0 ? (
                <EmptyState
                  icon={BookOpen}
                  title="Keine Artikel"
                  description={searchQuery ? 'Keine Treffer für diese Suche.' : 'Erstelle deinen ersten Wiki-Artikel.'}
                />
              ) : (
                <div className="divide-y divide-border/50">
                  {filteredArticles.map((article) => {
                    const isSelected = selectedArticleId === article.id
                    const st = statusConfig[article.status]
                    return (
                      <button
                        key={article.id}
                        onClick={() => handleSelectArticle(article)}
                        className={`flex w-full flex-col px-3 py-2.5 text-left transition-colors border-l-[3px] ${
                          isSelected ? 'border-l-primary bg-primary/[0.03]' : 'border-l-transparent hover:bg-accent/50'
                        }`}
                      >
                        {/* Title row */}
                        <div className="flex items-center gap-1.5">
                          {article.isPinned && <Pin className="h-3 w-3 shrink-0 text-primary" />}
                          <span className="text-sm font-medium text-foreground truncate">{article.title}</span>
                        </div>

                        {/* Meta row */}
                        <div className="mt-0.5 flex items-center gap-2 text-[10px] text-muted-foreground">
                          <span>{article.authorName}</span>
                          <span>·</span>
                          <Clock className="h-2.5 w-2.5" />
                          <span>{formatDate(article.lastEditedAt)}</span>
                        </div>

                        {/* Status + tags row */}
                        <div className="mt-1 flex items-center gap-1.5">
                          <span className={`rounded-full px-2 py-0.5 text-[9px] font-medium ${st?.bg ?? ''}`}>
                            {st?.label ?? article.status}
                          </span>
                          {(article.tags ?? []).slice(0, 2).map((tag) => (
                            <span key={tag} className="inline-flex items-center gap-0.5 rounded-full bg-secondary px-1.5 py-0.5 text-[9px] text-muted-foreground">
                              <Tag className="h-2 w-2" />{tag}
                            </span>
                          ))}
                          <div className="flex-1" />
                          <div className="flex items-center gap-1 text-[10px] text-muted-foreground">
                            <Eye className="h-2.5 w-2.5" />{article.viewCount}
                          </div>
                          <div className="flex items-center gap-1 text-[10px] text-muted-foreground">
                            <History className="h-2.5 w-2.5" />v{(article.versions ?? []).length}
                          </div>
                        </div>
                      </button>
                    )
                  })}
                </div>
              )}
            </div>
          </div>

          {/* Article detail */}
          {selectedArticle && (
            <WikiArticle
              article={selectedArticle}
              onDelete={() => setDeleteTarget(selectedArticle)}
              onShare={() => setShareTarget(selectedArticle)}
            />
          )}
        </div>
      </div>

      {/* Dialogs */}
      <WikiTemplateDialog
        open={showNewArticle}
        onOpenChange={setShowNewArticle}
      />
      <WikiCategoryDialog
        open={showNewCategory}
        onOpenChange={setShowNewCategory}
      />
      <WikiShareDialog
        open={!!shareTarget}
        onOpenChange={(o) => { if (!o) setShareTarget(null) }}
        article={shareTarget}
      />
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(o) => { if (!o) setDeleteTarget(null) }}
        title="Artikel löschen"
        description={`"${deleteTarget?.title}" wird unwiderruflich gelöscht.`}
        confirmLabel="Löschen"
        variant="destructive"
        onConfirm={handleDelete}
      />
    </>
  )
}

/**
 * Wiki — Internal Knowledge Base.
 *
 * Three-column layout:
 *   Left  (w-56):   WikiSidebar — tree navigation, search, category management
 *   Center (flex-1): Article list or article detail (read/edit mode)
 *   Right  (w-64):   WikiVersionHistory — slide-in panel (when toggled)
 *
 * Server state via TanStack Query (useArticles, useCategories).
 * UI state via useWikiStore (selection, search, editing flag).
 * Type mapping via adaptArticle / adaptCategory from api/wiki-adapter.ts.
 */
import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  BookOpen,
  Pin,
  Eye,
  History,
  Tag,
  Clock,
  AlertCircle,
} from 'lucide-react'
import { useWikiStore } from '@/stores/wiki'
import { useArticles, useCategories, useDeleteArticle } from '@/api/hooks/useWiki'
import { adaptArticle, adaptCategory } from '@/api/wiki-adapter'
import type { WikiArticle as WikiArticleType } from '@/types/wiki'
import { EmptyState, ConfirmDialog } from '@/components/shared'
import { WikiSidebar } from './WikiSidebar'
import { WikiArticle } from './WikiArticle'
import { WikiTemplateDialog } from './WikiTemplateDialog'
import { WikiCategoryDialog } from './WikiCategoryDialog'
import { WikiShareDialog } from './WikiShareDialog'
import { formatDate as libFormatDate } from '@/lib/format'

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
  return libFormatDate(normalized)
}

// ---------------------------------------------------------------------------
// Skeleton
// ---------------------------------------------------------------------------

function ArticleListSkeleton() {
  return (
    <div className="divide-y divide-border/50">
      {[1, 2, 3, 4].map((i) => (
        <div key={i} className="flex flex-col px-3 py-2.5 gap-1.5">
          <div className="h-4 w-3/4 rounded bg-muted animate-pulse" />
          <div className="h-3 w-1/2 rounded bg-muted animate-pulse" />
          <div className="h-3 w-1/3 rounded bg-muted animate-pulse" />
        </div>
      ))}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function WikiPage() {
  const { t } = useTranslation()

  // Server state
  const { data: articlesData, isLoading: articlesLoading, isError: articlesError } = useArticles()
  const { data: categoriesData, isLoading: categoriesLoading } = useCategories()
  const deleteArticleMutation = useDeleteArticle()

  // UI state
  const selectedArticleId = useWikiStore((s) => s.selectedArticleId)
  const selectedCategoryId = useWikiStore((s) => s.selectedCategoryId)
  const searchQuery = useWikiStore((s) => s.searchQuery)
  const setSelectedArticle = useWikiStore((s) => s.setSelectedArticle)

  // Dialogs
  const [showNewArticle, setShowNewArticle] = useState(false)
  const [showNewCategory, setShowNewCategory] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<WikiArticleType | null>(null)
  const [shareTarget, setShareTarget] = useState<WikiArticleType | null>(null)

  // Adapt API types → UI types
  const articles = useMemo(
    () => (articlesData?.articles ?? []).map(adaptArticle),
    [articlesData],
  )
  const categories = useMemo(
    () => (categoriesData ?? []).map(adaptCategory),
    [categoriesData],
  )

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
          (a.tags ?? []).some((tag) => tag.toLowerCase().includes(q)),
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
    // viewCount increment not yet available via API
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    await deleteArticleMutation.mutateAsync(deleteTarget.id)
    if (selectedArticleId === deleteTarget.id) setSelectedArticle(null)
    setDeleteTarget(null)
  }

  return (
    <>
      <div className="flex h-full overflow-hidden animate-fade-up">
        {/* Left: Sidebar with tree navigation */}
        <WikiSidebar
          categories={categories}
          articles={articles}
          categoriesLoading={categoriesLoading}
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
                {t('wiki.list.articleCount', { count: filteredArticles.length })}
              </span>
            </div>

            {/* List */}
            <div className="flex-1 overflow-y-auto">
              {articlesError ? (
                <div className="flex items-center gap-2 px-3 py-4 text-sm text-destructive">
                  <AlertCircle className="h-4 w-4 shrink-0" />
                  <span>{t('wiki.list.loadError')}</span>
                </div>
              ) : articlesLoading ? (
                <ArticleListSkeleton />
              ) : filteredArticles.length === 0 ? (
                <EmptyState
                  icon={BookOpen}
                  title={t('wiki.list.noArticles')}
                  description={searchQuery ? t('wiki.list.noSearchResults') : t('wiki.list.createFirst')}
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
                          <Clock className="h-2.5 w-2.5" />
                          <span>{formatShortDate(article.lastEditedAt)}</span>
                        </div>

                        {/* Status + tags row */}
                        <div className="mt-1 flex items-center gap-1.5">
                          <span className={`rounded-full px-2 py-0.5 text-[9px] font-medium ${st?.bg ?? ''}`}>
                            {st ? t(st.key) : article.status}
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
        title={t('wiki.dialog.deleteTitle')}
        description={t('wiki.dialog.deleteDescription', { title: deleteTarget?.title })}
        confirmLabel={t('common.delete')}
        variant="destructive"
        onConfirm={handleDelete}
      />
    </>
  )
}

import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { BookOpen, Plus, FolderPlus } from 'lucide-react'
import { moduleHsl } from '@/components/layout/sidebar/nav-items'
import { useWikiStore } from '@/stores/wiki'
import type { WikiArticle, WikiCategory } from '@/types/wiki'
import { WikiSearch } from './WikiSearch'
import { WikiTreeNode } from './WikiTreeNode'

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface WikiSidebarProps {
  categories: WikiCategory[]
  articles: WikiArticle[]
  categoriesLoading?: boolean
  onNewArticle: () => void
  onNewCategory: () => void
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function WikiSidebar({ categories, articles, categoriesLoading, onNewArticle, onNewCategory }: WikiSidebarProps) {
  const { t } = useTranslation()
  const selectedCategoryId = useWikiStore((s) => s.selectedCategoryId)
  const setSelectedCategory = useWikiStore((s) => s.setSelectedCategory)

  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set((categories ?? []).map((c) => c.id)))

  const totalArticles = articles.length
  const publishedCount = articles.filter((a) => a.status === 'published').length
  const totalViews = articles.reduce((sum, a) => sum + a.viewCount, 0)

  // Article counts per category (live)
  const articleCounts = useMemo(() => {
    const counts: Record<string, number> = {}
    for (const a of articles) {
      counts[a.categoryId] = (counts[a.categoryId] ?? 0) + 1
    }
    return counts
  }, [articles])

  const toggleExpand = (id: string) => {
    setExpandedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  return (
    <div className="flex w-56 shrink-0 flex-col border-r border-border bg-card/50">
      {/* Header */}
      <div className="flex items-center justify-between px-3 py-2.5 border-b border-border">
        <div className="flex items-center gap-2">
          <div
            className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg"
            style={{ backgroundColor: moduleHsl('wiki') + '15', color: moduleHsl('wiki') }}
          >
            <BookOpen className="h-3.5 w-3.5" />
          </div>
          <div>
            <h2 className="text-sm font-semibold" style={{ color: moduleHsl('wiki') }}>Wiki</h2>
            <p className="text-[10px] text-muted-foreground">{t('wiki.sidebar.stats', { articles: totalArticles, views: totalViews })}</p>
          </div>
        </div>
        <div className="flex items-center gap-0.5">
          <button
            onClick={onNewCategory}
            className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
            title={t('wiki.sidebar.newCategory')}
          >
            <FolderPlus className="h-3.5 w-3.5" />
          </button>
          <button
            onClick={onNewArticle}
            className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
            title={t('wiki.sidebar.newArticle')}
          >
            <Plus className="h-3.5 w-3.5" />
          </button>
        </div>
      </div>

      {/* Search */}
      <WikiSearch />

      {/* Tree navigation */}
      <div className="flex-1 overflow-y-auto px-2 pb-2">
        {/* All articles */}
        <button
          onClick={() => setSelectedCategory(null)}
          className={`flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors mb-1 ${
            !selectedCategoryId ? 'bg-primary/10 text-primary font-medium' : 'text-foreground/80 hover:bg-accent'
          }`}
        >
          <BookOpen className="h-4 w-4 shrink-0" />
          <span className="flex-1 text-left">{t('wiki.sidebar.allArticles')}</span>
          <span className="text-[10px] text-muted-foreground">{totalArticles}</span>
        </button>

        <div className="my-1.5 border-t border-border" />

        {/* Categories */}
        {categoriesLoading ? (
          <div className="space-y-1">
            {[1, 2, 3].map((i) => (
              <div key={i} className="h-7 rounded bg-muted animate-pulse" />
            ))}
          </div>
        ) : (
          <div className="space-y-0.5">
            {(categories ?? []).map((cat) => (
              <WikiTreeNode
                key={cat.id}
                category={cat}
                isSelected={selectedCategoryId === cat.id}
                isExpanded={expandedIds.has(cat.id)}
                onSelect={setSelectedCategory}
                onToggle={toggleExpand}
                articleCount={articleCounts[cat.id] ?? 0}
              />
            ))}
          </div>
        )}
      </div>

      {/* Stats footer */}
      <div className="border-t border-border px-3 py-2.5 text-xs text-muted-foreground">
        <div className="flex justify-between">
          <span>{t('wiki.status.published')}</span>
          <span>{publishedCount}</span>
        </div>
        <div className="flex justify-between">
          <span>{t('wiki.sidebar.drafts')}</span>
          <span>{totalArticles - publishedCount}</span>
        </div>
      </div>
    </div>
  )
}

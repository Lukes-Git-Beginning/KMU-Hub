import { useState, useMemo } from 'react'
import {
  Search,
  Plus,
  BookOpen,
  Pin,
  Eye,
  Clock,
  Tag,
  ChevronRight,
  Edit3,
  Trash2,
  Archive,
  Send,
  FileText,
  AlertTriangle,
  Building2,
  GitBranch,
  Monitor,
  Users,
  History,
  Star,
} from 'lucide-react'
import { toast } from 'sonner'
import { useWikiStore } from '@/stores/wiki'
import type { WikiArticle, WikiCategory } from '@/types/wiki'
import { ItemActions, ConfirmDialog, EmptyState } from '@/components/shared'

// ─── Icon map for categories ───────────────────────────────────────
const categoryIconMap: Record<string, typeof BookOpen> = {
  Building2,
  GitBranch,
  Monitor,
  Users,
  BookOpen,
}

// ─── Helpers ───────────────────────────────────────────────────────
function formatDate(dateStr: string): string {
  return new Date(dateStr + (dateStr.includes('T') ? '' : 'T00:00:00')).toLocaleDateString('de-CH')
}

const statusConfig: Record<string, { label: string; bg: string }> = {
  draft: { label: 'Entwurf', bg: 'bg-secondary text-muted-foreground' },
  published: { label: 'Veroeffentlicht', bg: 'bg-success-light text-success' },
  archived: { label: 'Archiviert', bg: 'bg-warning-light text-warning' },
}

// ═══════════════════════════════════════════════════════════════════
// Component
// ═══════════════════════════════════════════════════════════════════

export default function WikiPage() {
  const categories = useWikiStore((s) => s.categories)
  const articles = useWikiStore((s) => s.articles)
  const templates = useWikiStore((s) => s.templates)
  const selectedArticleId = useWikiStore((s) => s.selectedArticleId)
  const selectedCategoryId = useWikiStore((s) => s.selectedCategoryId)
  const searchQuery = useWikiStore((s) => s.searchQuery)
  const setSelectedArticle = useWikiStore((s) => s.setSelectedArticle)
  const setSelectedCategory = useWikiStore((s) => s.setSelectedCategory)
  const setSearchQuery = useWikiStore((s) => s.setSearchQuery)
  const deleteArticle = useWikiStore((s) => s.deleteArticle)
  const publishArticle = useWikiStore((s) => s.publishArticle)
  const archiveArticle = useWikiStore((s) => s.archiveArticle)
  const togglePin = useWikiStore((s) => s.togglePin)
  const incrementViewCount = useWikiStore((s) => s.incrementViewCount)
  const addArticle = useWikiStore((s) => s.addArticle)

  const [deleteTarget, setDeleteTarget] = useState<WikiArticle | null>(null)
  const [showNewArticle, setShowNewArticle] = useState(false)
  const [newTitle, setNewTitle] = useState('')
  const [newTemplateId, setNewTemplateId] = useState<string | null>(null)

  // ─── Filtered articles ───────────────────────────────────────────
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
          a.tags.some((t) => t.toLowerCase().includes(q)) ||
          a.authorName.toLowerCase().includes(q),
      )
    }
    // Pinned first, then by last edited
    return result.sort((a, b) => {
      if (a.isPinned !== b.isPinned) return a.isPinned ? -1 : 1
      return b.lastEditedAt.localeCompare(a.lastEditedAt)
    })
  }, [articles, selectedCategoryId, searchQuery])

  const selectedArticle = articles.find((a) => a.id === selectedArticleId) ?? null

  // ─── Handlers ────────────────────────────────────────────────────
  const handleSelectArticle = (article: WikiArticle) => {
    setSelectedArticle(article.id)
    incrementViewCount(article.id)
  }

  const handleDelete = () => {
    if (!deleteTarget) return
    deleteArticle(deleteTarget.id)
    toast.success('Artikel geloescht')
    setDeleteTarget(null)
  }

  const handleCreateArticle = () => {
    if (!newTitle.trim()) return
    const template = newTemplateId ? templates.find((t) => t.id === newTemplateId) : null
    addArticle({
      title: newTitle.trim(),
      content: template?.content ?? '<p>Neuer Artikel — hier Inhalt hinzufuegen.</p>',
      categoryId: selectedCategoryId ?? 'wc1',
      status: 'draft',
      authorId: 'c1',
      authorName: 'Anna Mueller',
      tags: [],
      isPinned: false,
      lastEditedBy: 'Anna Mueller',
      lastEditedAt: new Date().toISOString().split('T')[0],
    })
    toast.success('Artikel erstellt')
    setShowNewArticle(false)
    setNewTitle('')
    setNewTemplateId(null)
  }

  // ─── Stats ───────────────────────────────────────────────────────
  const totalArticles = articles.length
  const publishedCount = articles.filter((a) => a.status === 'published').length
  const totalViews = articles.reduce((sum, a) => sum + a.viewCount, 0)

  // ═════════════════════════════════════════════════════════════════
  // Render
  // ═════════════════════════════════════════════════════════════════

  return (
    <div className="flex h-full">
      {/* ── Category Sidebar ─────────────────────────────────────── */}
      <div className="flex w-56 shrink-0 flex-col border-r border-border bg-card/50">
        <div className="border-b border-border p-3">
          <h2 className="text-sm font-semibold text-foreground">Wiki</h2>
          <p className="text-xs text-muted-foreground">{totalArticles} Artikel · {totalViews} Aufrufe</p>
        </div>

        <div className="flex-1 overflow-y-auto p-2">
          {/* All articles */}
          <button
            onClick={() => setSelectedCategory(null)}
            className={`flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-colors ${
              !selectedCategoryId ? 'bg-primary/10 text-primary font-medium' : 'text-foreground/80 hover:bg-accent'
            }`}
          >
            <BookOpen className="h-4 w-4" />
            <span>Alle Artikel</span>
            <span className="ml-auto text-xs text-muted-foreground">{totalArticles}</span>
          </button>

          <div className="my-2 border-t border-border" />

          {/* Categories */}
          {categories.map((cat) => {
            const CatIcon = categoryIconMap[cat.icon] ?? BookOpen
            return (
              <button
                key={cat.id}
                onClick={() => setSelectedCategory(cat.id)}
                className={`flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-colors ${
                  selectedCategoryId === cat.id ? 'bg-primary/10 text-primary font-medium' : 'text-foreground/80 hover:bg-accent'
                }`}
              >
                <CatIcon className="h-4 w-4" />
                <span className="truncate">{cat.name}</span>
                <span className="ml-auto text-xs text-muted-foreground">{cat.articleCount}</span>
              </button>
            )
          })}
        </div>

        {/* Stats */}
        <div className="border-t border-border p-3 text-xs text-muted-foreground">
          <div className="flex justify-between">
            <span>Veroeffentlicht</span>
            <span>{publishedCount}</span>
          </div>
          <div className="flex justify-between">
            <span>Entwuerfe</span>
            <span>{totalArticles - publishedCount}</span>
          </div>
        </div>
      </div>

      {/* ── Article List ─────────────────────────────────────────── */}
      <div className="flex min-w-0 flex-1 flex-col">
        {/* Header */}
        <div className="flex items-center gap-3 border-b border-border px-4 py-3">
          <div className="relative flex-1">
            <Search className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Artikel suchen..."
              className="h-9 w-full rounded-md border border-border bg-transparent pl-9 pr-3 text-sm outline-none transition-colors placeholder:text-muted-foreground focus:border-primary"
            />
          </div>
          <button
            onClick={() => setShowNewArticle(true)}
            className="flex h-9 items-center gap-1.5 rounded-md bg-primary px-3 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
          >
            <Plus className="h-4 w-4" />
            Neuer Artikel
          </button>
        </div>

        {/* Article list + detail split */}
        <div className="flex flex-1 overflow-hidden">
          {/* List */}
          <div className={`flex-1 overflow-y-auto ${selectedArticle ? 'max-w-md border-r border-border' : ''}`}>
            {filteredArticles.length === 0 ? (
              <EmptyState
                icon={BookOpen}
                title="Keine Artikel"
                description={searchQuery ? 'Keine Artikel fuer diese Suche gefunden.' : 'Erstelle deinen ersten Wiki-Artikel.'}
              />
            ) : (
              <div className="divide-y divide-border">
                {filteredArticles.map((article) => {
                  const isSelected = selectedArticleId === article.id
                  return (
                    <div
                      key={article.id}
                      onClick={() => handleSelectArticle(article)}
                      className={`cursor-pointer px-4 py-3 transition-colors ${
                        isSelected ? 'bg-primary/5' : 'hover:bg-accent/50'
                      }`}
                    >
                      <div className="flex items-start gap-2">
                        <div className="min-w-0 flex-1">
                          <div className="flex items-center gap-1.5">
                            {article.isPinned && <Pin className="h-3 w-3 shrink-0 text-primary" />}
                            <h3 className="truncate text-sm font-medium text-foreground">{article.title}</h3>
                          </div>
                          <div className="mt-1 flex items-center gap-2 text-xs text-muted-foreground">
                            <span>{article.authorName}</span>
                            <span>·</span>
                            <span>{formatDate(article.lastEditedAt)}</span>
                          </div>
                          <div className="mt-1.5 flex items-center gap-1.5">
                            <span className={`inline-flex rounded-full px-2 py-0.5 text-[10px] font-medium ${statusConfig[article.status]?.bg ?? ''}`}>
                              {statusConfig[article.status]?.label ?? article.status}
                            </span>
                            {article.tags.slice(0, 3).map((tag) => (
                              <span key={tag} className="inline-flex items-center gap-0.5 rounded-full bg-secondary px-2 py-0.5 text-[10px] text-muted-foreground">
                                <Tag className="h-2.5 w-2.5" />{tag}
                              </span>
                            ))}
                          </div>
                        </div>
                        <div className="flex shrink-0 flex-col items-end gap-1 text-xs text-muted-foreground">
                          <div className="flex items-center gap-1">
                            <Eye className="h-3 w-3" />{article.viewCount}
                          </div>
                          <div className="flex items-center gap-1">
                            <History className="h-3 w-3" />v{article.versions.length}
                          </div>
                        </div>
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
          </div>

          {/* ── Article Detail ──────────────────────────────────── */}
          {selectedArticle && (
            <div className="flex flex-1 flex-col overflow-hidden">
              {/* Detail header */}
              <div className="flex items-center justify-between border-b border-border px-5 py-3">
                <div className="min-w-0 flex-1">
                  <h2 className="truncate text-lg font-semibold text-foreground">{selectedArticle.title}</h2>
                  <div className="mt-0.5 flex items-center gap-2 text-xs text-muted-foreground">
                    <span>{selectedArticle.authorName}</span>
                    <span>·</span>
                    <Clock className="h-3 w-3" />
                    <span>{formatDate(selectedArticle.lastEditedAt)}</span>
                    <span>·</span>
                    <Eye className="h-3 w-3" />
                    <span>{selectedArticle.viewCount} Aufrufe</span>
                  </div>
                </div>
                <ItemActions
                  actions={[
                    { label: selectedArticle.isPinned ? 'Unpin' : 'Anpinnen', icon: Pin, onClick: () => togglePin(selectedArticle.id) },
                    ...(selectedArticle.status === 'draft'
                      ? [{ label: 'Veroeffentlichen', icon: Send, onClick: () => { publishArticle(selectedArticle.id); toast.success('Veroeffentlicht') } }]
                      : []),
                    ...(selectedArticle.status === 'published'
                      ? [{ label: 'Archivieren', icon: Archive, onClick: () => { archiveArticle(selectedArticle.id); toast.success('Archiviert') } }]
                      : []),
                    { label: 'Loeschen', icon: Trash2, onClick: () => setDeleteTarget(selectedArticle), variant: 'destructive' as const },
                  ]}
                />
              </div>

              {/* Tags */}
              {selectedArticle.tags.length > 0 && (
                <div className="flex flex-wrap gap-1.5 border-b border-border px-5 py-2">
                  {selectedArticle.tags.map((tag) => (
                    <span key={tag} className="inline-flex items-center gap-1 rounded-full bg-secondary px-2.5 py-0.5 text-xs text-muted-foreground">
                      <Tag className="h-3 w-3" />{tag}
                    </span>
                  ))}
                </div>
              )}

              {/* Content */}
              <div className="flex-1 overflow-y-auto px-5 py-4">
                <div
                  className="prose prose-sm max-w-none dark:prose-invert"
                  dangerouslySetInnerHTML={{ __html: selectedArticle.content }}
                />
              </div>

              {/* Version History */}
              <div className="border-t border-border px-5 py-3">
                <h4 className="mb-2 text-xs font-medium text-muted-foreground">Versionen</h4>
                <div className="space-y-1">
                  {selectedArticle.versions.slice(0, 3).map((v) => (
                    <div key={v.id} className="flex items-center gap-2 text-xs text-muted-foreground">
                      <span className="rounded bg-secondary px-1.5 py-0.5 font-mono text-[10px]">v{v.version}</span>
                      <span>{v.editorName}</span>
                      <span className="text-muted-foreground/60">·</span>
                      <span>{formatDate(v.editedAt)}</span>
                      <span className="text-muted-foreground/60">·</span>
                      <span className="italic">{v.changeNote}</span>
                    </div>
                  ))}
                  {selectedArticle.versions.length > 3 && (
                    <p className="text-[11px] text-muted-foreground/60">
                      +{selectedArticle.versions.length - 3} weitere Versionen
                    </p>
                  )}
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* ── New Article Dialog ───────────────────────────────────── */}
      {showNewArticle && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="w-full max-w-md rounded-lg border border-border bg-background p-6 shadow-xl">
            <h3 className="text-lg font-semibold text-foreground">Neuer Wiki-Artikel</h3>
            <p className="mt-1 text-sm text-muted-foreground">Erstelle einen neuen Artikel fuer das interne Wiki.</p>

            <div className="mt-4 space-y-3">
              <div>
                <label className="mb-1 block text-sm font-medium text-foreground">Titel</label>
                <input
                  type="text"
                  value={newTitle}
                  onChange={(e) => setNewTitle(e.target.value)}
                  placeholder="z.B. Einrichtung Arbeitsplatz"
                  className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground focus:border-primary"
                  autoFocus
                  onKeyDown={(e) => { if (e.key === 'Enter') handleCreateArticle() }}
                />
              </div>

              <div>
                <label className="mb-1 block text-sm font-medium text-foreground">Vorlage (optional)</label>
                <div className="grid grid-cols-3 gap-2">
                  {templates.map((t) => {
                    const TIcon = t.icon === 'AlertTriangle' ? AlertTriangle : t.icon === 'BookOpen' ? BookOpen : FileText
                    return (
                      <button
                        key={t.id}
                        onClick={() => setNewTemplateId(newTemplateId === t.id ? null : t.id)}
                        className={`flex flex-col items-center gap-1 rounded-md border px-2 py-2 text-xs transition-colors ${
                          newTemplateId === t.id
                            ? 'border-primary bg-primary/5 text-primary'
                            : 'border-border text-muted-foreground hover:bg-accent'
                        }`}
                      >
                        <TIcon className="h-4 w-4" />
                        <span className="truncate">{t.name}</span>
                      </button>
                    )
                  })}
                </div>
              </div>
            </div>

            <div className="mt-5 flex justify-end gap-2">
              <button
                onClick={() => { setShowNewArticle(false); setNewTitle(''); setNewTemplateId(null) }}
                className="h-9 rounded-md border border-border px-4 text-sm text-foreground transition-colors hover:bg-accent"
              >
                Abbrechen
              </button>
              <button
                onClick={handleCreateArticle}
                disabled={!newTitle.trim()}
                className="h-9 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
              >
                Erstellen
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ── Delete Confirm ───────────────────────────────────────── */}
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(o) => { if (!o) setDeleteTarget(null) }}
        title="Artikel loeschen"
        description={`"${deleteTarget?.title}" wird unwiderruflich geloescht.`}
        confirmLabel="Loeschen"
        variant="destructive"
        onConfirm={handleDelete}
      />
    </div>
  )
}

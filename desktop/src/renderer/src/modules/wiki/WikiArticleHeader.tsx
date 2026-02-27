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
  Tag,
} from 'lucide-react'
import { toast } from 'sonner'
import type { WikiArticle } from '@/types/wiki'
import { useWikiStore } from '@/stores/wiki'
import { ItemActions } from '@/components/shared'

// ---------------------------------------------------------------------------
// Status config
// ---------------------------------------------------------------------------

const statusConfig: Record<string, { label: string; bg: string }> = {
  draft: { label: 'Entwurf', bg: 'bg-secondary text-muted-foreground' },
  published: { label: 'Veroeffentlicht', bg: 'bg-success-light text-success' },
  archived: { label: 'Archiviert', bg: 'bg-warning-light text-warning' },
}

function formatDate(dateStr: string): string {
  try {
    return new Date(dateStr + (dateStr.includes('T') ? '' : 'T00:00:00')).toLocaleDateString('de-CH', {
      day: '2-digit',
      month: 'short',
      year: 'numeric',
    })
  } catch {
    return dateStr
  }
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface WikiArticleHeaderProps {
  article: WikiArticle
  onEdit: () => void
  onDelete: () => void
  onToggleVersions: () => void
  onShare: () => void
}

export function WikiArticleHeader({
  article,
  onEdit,
  onDelete,
  onToggleVersions,
  onShare,
}: WikiArticleHeaderProps) {
  const togglePin = useWikiStore((s) => s.togglePin)
  const publishArticle = useWikiStore((s) => s.publishArticle)
  const archiveArticle = useWikiStore((s) => s.archiveArticle)

  const st = statusConfig[article.status]

  return (
    <div className="border-b border-border px-5 py-3">
      {/* Row 1: Title + actions */}
      <div className="flex items-start gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            {article.isPinned && <Pin className="h-3.5 w-3.5 shrink-0 text-primary" />}
            <h2 className="text-lg font-semibold text-foreground truncate">{article.title}</h2>
            <span className={`shrink-0 rounded-full px-2 py-0.5 text-[10px] font-medium ${st?.bg ?? ''}`}>
              {st?.label ?? article.status}
            </span>
          </div>

          {/* Row 2: Meta info */}
          <div className="mt-1 flex items-center gap-2 text-xs text-muted-foreground flex-wrap">
            <span>{article.authorName}</span>
            <span>·</span>
            <Clock className="h-3 w-3" />
            <span>{formatDate(article.lastEditedAt)}</span>
            <span>·</span>
            <Eye className="h-3 w-3" />
            <span>{article.viewCount} Aufrufe</span>
            <span>·</span>
            <History className="h-3 w-3" />
            <span>v{article.versions.length}</span>
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-1 shrink-0">
          <button
            onClick={onEdit}
            className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
            title="Bearbeiten"
          >
            <Edit3 className="h-4 w-4" />
          </button>
          <button
            onClick={onToggleVersions}
            className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
            title="Versionshistorie"
          >
            <History className="h-4 w-4" />
          </button>
          <button
            onClick={onShare}
            className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
            title="Teilen"
          >
            <Share2 className="h-4 w-4" />
          </button>
          <ItemActions
            actions={[
              { label: article.isPinned ? 'Unpin' : 'Anpinnen', icon: Pin, onClick: () => togglePin(article.id) },
              ...(article.status === 'draft'
                ? [{ label: 'Veroeffentlichen', icon: Send, onClick: () => { publishArticle(article.id); toast.success('Veroeffentlicht') } }]
                : []),
              ...(article.status === 'published'
                ? [{ label: 'Archivieren', icon: Archive, onClick: () => { archiveArticle(article.id); toast.success('Archiviert') } }]
                : []),
              { label: 'Loeschen', icon: Trash2, onClick: onDelete, variant: 'destructive' as const },
            ]}
          />
        </div>
      </div>

      {/* Tags */}
      {article.tags.length > 0 && (
        <div className="flex flex-wrap gap-1.5 mt-2">
          {article.tags.map((tag) => (
            <span key={tag} className="inline-flex items-center gap-1 rounded-full bg-secondary px-2.5 py-0.5 text-xs text-muted-foreground">
              <Tag className="h-3 w-3" />{tag}
            </span>
          ))}
        </div>
      )}
    </div>
  )
}

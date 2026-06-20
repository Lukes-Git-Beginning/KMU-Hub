import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  ChevronRight,
  ChevronDown,
  BookOpen,
  Building2,
  GitBranch,
  Monitor,
  Users,
  Pencil,
  Trash2,
} from 'lucide-react'
import type { WikiCategory } from '@/types/wiki'
import { useUpdateCategory, useDeleteCategory } from '@/api/hooks/useWiki'
import { ItemActions, ConfirmDialog } from '@/components/shared'

// ---------------------------------------------------------------------------
// Icon map
// ---------------------------------------------------------------------------

const iconMap: Record<string, typeof BookOpen> = {
  Building2,
  GitBranch,
  Monitor,
  Users,
  BookOpen,
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface WikiTreeNodeProps {
  category: WikiCategory
  /** Full category list — children are derived via parentId. */
  categories: WikiCategory[]
  selectedCategoryId: string | null
  expandedIds: Set<string>
  onSelect: (id: string) => void
  onToggle: (id: string) => void
  articleCounts: Record<string, number>
  /** Nesting depth (0 = top level) — drives indentation. */
  level?: number
}

export function WikiTreeNode({
  category,
  categories,
  selectedCategoryId,
  expandedIds,
  onSelect,
  onToggle,
  articleCounts,
  level = 0,
}: WikiTreeNodeProps) {
  const { t } = useTranslation()
  const Icon = iconMap[category.icon] ?? BookOpen
  const updateCategory = useUpdateCategory()
  const deleteCategory = useDeleteCategory()

  const [editing, setEditing] = useState(false)
  const [draftName, setDraftName] = useState(category.name)
  const [confirmDelete, setConfirmDelete] = useState(false)

  const children = categories
    .filter((c) => (c.parentId ?? null) === category.id)
    .sort((a, b) => a.sortOrder - b.sortOrder)
  const hasChildren = children.length > 0
  const isExpanded = expandedIds.has(category.id)
  const isSelected = selectedCategoryId === category.id
  const articleCount = articleCounts[category.id] ?? 0

  const startRename = () => {
    setDraftName(category.name)
    setEditing(true)
  }

  const commitRename = async () => {
    const name = draftName.trim()
    setEditing(false)
    if (!name || name === category.name) return
    try {
      await updateCategory.mutateAsync({ id: category.id, name })
      toast.success(t('wiki.category.renamed'))
    } catch {
      toast.error(t('wiki.category.renameError'))
    }
  }

  const handleDelete = async () => {
    try {
      await deleteCategory.mutateAsync(category.id)
      toast.success(t('wiki.category.deleted'))
    } catch {
      toast.error(t('wiki.category.deleteError'))
    }
  }

  return (
    <div>
      <div
        className={`group relative flex items-center gap-1.5 rounded-md pr-1 text-sm transition-colors ${
          isSelected ? 'bg-primary/10 text-primary font-medium' : 'text-foreground/80 hover:bg-accent'
        }`}
        style={{ paddingLeft: level * 12 }}
      >
        {/* Expand/collapse chevron (only when the node has children) */}
        {hasChildren ? (
          <button
            onClick={() => onToggle(category.id)}
            className="shrink-0 rounded p-0.5 pl-2 hover:text-foreground"
            aria-label={isExpanded ? t('wiki.category.collapse') : t('wiki.category.expand')}
          >
            {isExpanded ? (
              <ChevronDown className="h-3 w-3 text-muted-foreground" />
            ) : (
              <ChevronRight className="h-3 w-3 text-muted-foreground" />
            )}
          </button>
        ) : (
          <span className="shrink-0 p-0.5 pl-2" aria-hidden="true">
            <span className="block h-3 w-3" />
          </span>
        )}

        <Icon className="h-4 w-4 shrink-0" />

        {editing ? (
          <input
            value={draftName}
            onChange={(e) => setDraftName(e.target.value)}
            onBlur={commitRename}
            onKeyDown={(e) => {
              if (e.key === 'Enter') commitRename()
              if (e.key === 'Escape') setEditing(false)
            }}
            autoFocus
            aria-label={t('wiki.category.rename')}
            className="min-w-0 flex-1 rounded border border-primary/50 bg-background px-1 py-0.5 text-sm outline-none"
          />
        ) : (
          <button
            onClick={() => onSelect(category.id)}
            className="flex min-w-0 flex-1 items-center gap-1.5 py-1.5 text-left"
          >
            <span className="truncate flex-1">{category.name}</span>
          </button>
        )}

        {!editing && (
          <>
            <span className="shrink-0 text-[10px] text-muted-foreground group-hover:hidden">
              {articleCount}
            </span>
            <div className="hidden shrink-0 group-hover:block">
              <ItemActions
                triggerClassName="h-6 w-6"
                items={[
                  { label: t('wiki.category.rename'), icon: Pencil, onClick: startRename },
                  { label: t('common.delete'), icon: Trash2, onClick: () => setConfirmDelete(true), variant: 'destructive' },
                ]}
              />
            </div>
          </>
        )}

        <ConfirmDialog
          open={confirmDelete}
          onOpenChange={setConfirmDelete}
          title={t('wiki.category.deleteTitle')}
          description={t('wiki.category.deleteDescription', { name: category.name })}
          confirmLabel={t('common.delete')}
          variant="destructive"
          onConfirm={handleDelete}
        />
      </div>

      {/* Child categories (rendered when expanded) */}
      {hasChildren && isExpanded && (
        <div className="mt-0.5 space-y-0.5">
          {children.map((child) => (
            <WikiTreeNode
              key={child.id}
              category={child}
              categories={categories}
              selectedCategoryId={selectedCategoryId}
              expandedIds={expandedIds}
              onSelect={onSelect}
              onToggle={onToggle}
              articleCounts={articleCounts}
              level={level + 1}
            />
          ))}
        </div>
      )}
    </div>
  )
}

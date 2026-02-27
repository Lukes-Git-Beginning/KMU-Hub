import { ChevronRight, ChevronDown, BookOpen, Building2, GitBranch, Monitor, Users } from 'lucide-react'
import type { WikiCategory } from '@/types/wiki'

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
  isSelected: boolean
  isExpanded: boolean
  onSelect: (id: string) => void
  onToggle: (id: string) => void
  articleCount: number
}

export function WikiTreeNode({
  category,
  isSelected,
  isExpanded,
  onSelect,
  onToggle,
  articleCount,
}: WikiTreeNodeProps) {
  const Icon = iconMap[category.icon] ?? BookOpen

  return (
    <div>
      <button
        onClick={() => onSelect(category.id)}
        className={`flex w-full items-center gap-1.5 rounded-md px-2 py-1.5 text-sm transition-colors ${
          isSelected
            ? 'bg-primary/10 text-primary font-medium'
            : 'text-foreground/80 hover:bg-accent'
        }`}
      >
        {/* Expand/collapse chevron */}
        <button
          onClick={(e) => {
            e.stopPropagation()
            onToggle(category.id)
          }}
          className="shrink-0 rounded p-0.5 hover:bg-accent"
        >
          {isExpanded ? (
            <ChevronDown className="h-3 w-3 text-muted-foreground" />
          ) : (
            <ChevronRight className="h-3 w-3 text-muted-foreground" />
          )}
        </button>

        <Icon className="h-4 w-4 shrink-0" />
        <span className="truncate flex-1 text-left">{category.name}</span>
        <span className="text-[10px] text-muted-foreground shrink-0">{articleCount}</span>
      </button>
    </div>
  )
}

import {
  UserPlus,
  FolderPlus,
  Receipt,
  Video,
  Clock,
  Mail,
  LifeBuoy,
  BookOpen,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import type { QuickAction } from '@/stores/search'
import { cn } from '@/lib/cn'

const iconMap: Record<string, LucideIcon> = {
  UserPlus,
  FolderPlus,
  Receipt,
  Video,
  Clock,
  Mail,
  LifeBuoy,
  BookOpen,
}

interface QuickActionsProps {
  actions: QuickAction[]
  activeIndex: number
  startIndex: number
  onSelect: (action: QuickAction) => void
  onHover: (index: number) => void
}

export function QuickActions({
  actions,
  activeIndex,
  startIndex,
  onSelect,
  onHover,
}: QuickActionsProps) {
  return (
    <div className="px-2 py-1.5">
      <div className="px-2 pb-1 text-xs font-medium text-muted-foreground">
        Schnellaktionen
      </div>
      {actions.map((action, i) => {
        const Icon = iconMap[action.icon] ?? FolderPlus
        const isActive = activeIndex === startIndex + i

        return (
          <button
            key={action.id}
            onClick={() => onSelect(action)}
            onMouseEnter={() => onHover(startIndex + i)}
            className={cn(
              'flex w-full items-center gap-3 rounded-md px-3 py-2 text-left text-sm transition-colors',
              isActive
                ? 'bg-primary/10 text-foreground'
                : 'text-foreground/80 hover:bg-accent',
            )}
          >
            <Icon className="h-4 w-4 shrink-0 text-muted-foreground" />
            <span className="flex-1">{action.label}</span>
            {action.shortcut && (
              <kbd className="rounded border border-border bg-muted px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">
                {action.shortcut}
              </kbd>
            )}
          </button>
        )
      })}
    </div>
  )
}

import { cn } from '@/lib/cn'
import type { LucideIcon } from 'lucide-react'

interface SearchResultItemProps {
  icon: LucideIcon
  title: string
  subtitle: string
  active: boolean
  onClick: () => void
  onMouseEnter: () => void
}

export function SearchResultItem({
  icon: Icon,
  title,
  subtitle,
  active,
  onClick,
  onMouseEnter,
}: SearchResultItemProps) {
  return (
    <button
      onClick={onClick}
      onMouseEnter={onMouseEnter}
      className={cn(
        'flex w-full items-center gap-3 rounded-md px-3 py-2 text-left text-sm transition-colors',
        active
          ? 'bg-primary/10 text-foreground'
          : 'text-foreground/80 hover:bg-accent',
      )}
    >
      <Icon className="h-4 w-4 shrink-0 text-muted-foreground" />
      <div className="min-w-0 flex-1">
        <div className="truncate font-medium">{title}</div>
        {subtitle && (
          <div className="truncate text-xs text-muted-foreground">{subtitle}</div>
        )}
      </div>
    </button>
  )
}

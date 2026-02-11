import { MoreVertical } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { cn } from '@/lib'

export interface ActionItem {
  label: string
  icon?: LucideIcon
  onClick: () => void
  variant?: 'default' | 'destructive'
  disabled?: boolean
  separator?: boolean
}

interface ItemActionsProps {
  items: ActionItem[]
  advancedLabel?: string
  onAdvanced?: () => void
  triggerClassName?: string
}

export function ItemActions({
  items,
  advancedLabel,
  onAdvanced,
  triggerClassName,
}: ItemActionsProps) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className={cn('h-8 w-8 shrink-0', triggerClassName)}
          onClick={(e) => e.stopPropagation()}
        >
          <MoreVertical className="h-4 w-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-48">
        {items.map((item, i) => (
          <div key={i}>
            {item.separator && <DropdownMenuSeparator />}
            <DropdownMenuItem
              onClick={(e) => {
                e.stopPropagation()
                item.onClick()
              }}
              disabled={item.disabled}
              className={cn(
                'cursor-pointer',
                item.variant === 'destructive' && 'text-red-600 focus:text-red-600',
              )}
            >
              {item.icon && <item.icon className="mr-2 h-4 w-4" />}
              {item.label}
            </DropdownMenuItem>
          </div>
        ))}
        {onAdvanced && (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              onClick={(e) => {
                e.stopPropagation()
                onAdvanced()
              }}
              className="cursor-pointer font-medium"
            >
              {advancedLabel || 'Erweiterte Optionen'}
            </DropdownMenuItem>
          </>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

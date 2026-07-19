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
import { useTranslation } from 'react-i18next'

export interface ActionItem {
  label: string
  icon?: LucideIcon
  onClick: () => void
  variant?: 'default' | 'destructive'
  disabled?: boolean
  /** Hover hint for a disabled entry (RBAC exception pattern: the item stays
   *  visible, greyed, and explains itself — e.g. missing send/import right).
   *  Rendered as faux-disabled so the native tooltip still fires. */
  title?: string
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
  const { t } = useTranslation()

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
                if (item.disabled) return
                item.onClick()
              }}
              disabled={item.disabled && !item.title}
              aria-disabled={item.disabled || undefined}
              title={item.disabled ? item.title : undefined}
              className={cn(
                'cursor-pointer',
                item.variant === 'destructive' && 'text-destructive focus:text-destructive',
                item.disabled && item.title && 'cursor-not-allowed opacity-50 focus:bg-transparent'
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
              {advancedLabel || t('shared.itemActions.advancedOptions')}
            </DropdownMenuItem>
          </>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

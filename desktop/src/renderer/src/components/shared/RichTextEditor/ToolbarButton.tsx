import type { LucideIcon } from 'lucide-react'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { cn } from '@/lib'

interface ToolbarButtonProps {
  icon: LucideIcon
  onClick: () => void
  active?: boolean
  disabled?: boolean
  tooltip: string
}

export function ToolbarButton({
  icon: Icon,
  onClick,
  active = false,
  disabled = false,
  tooltip,
}: ToolbarButtonProps) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          onClick={onClick}
          disabled={disabled}
          className={cn(
            'inline-flex h-7 w-7 items-center justify-center rounded',
            'text-muted-foreground transition-colors',
            'hover:bg-accent hover:text-accent-foreground',
            'disabled:pointer-events-none disabled:opacity-30',
            active && 'bg-accent text-accent-foreground',
          )}
        >
          <Icon className="h-4 w-4" />
        </button>
      </TooltipTrigger>
      <TooltipContent side="bottom" className="text-xs">
        {tooltip}
      </TooltipContent>
    </Tooltip>
  )
}

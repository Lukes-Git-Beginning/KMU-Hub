import { ChevronLeft, ChevronRight } from 'lucide-react'
import { cn } from '@/lib/cn'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

interface SidebarBrandingProps {
  collapsed: boolean
  onToggle: () => void
}

export function SidebarBranding({ collapsed, onToggle }: SidebarBrandingProps) {
  return (
    <div className="border-b border-sidebar-border">
      <div className={cn('flex items-center p-4', collapsed ? 'justify-center' : 'justify-between')}>
        {collapsed ? (
          <span className="text-lg font-bold text-sidebar-primary">K</span>
        ) : (
          <div>
            <h1 className="text-sm font-bold text-sidebar-foreground">KMU Digital Hub</h1>
            <p className="text-xs text-sidebar-muted">Schweizer Lösung</p>
          </div>
        )}
      </div>

      <div className="px-3 pb-3">
        <Tooltip delayDuration={collapsed ? 100 : 999999}>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="sm"
              onClick={onToggle}
              className={cn(
                'hidden lg:flex w-full items-center gap-2 text-sidebar-muted hover:bg-sidebar-accent hover:text-sidebar-foreground',
                collapsed && 'justify-center'
              )}
            >
              {collapsed ? (
                <ChevronRight className="h-4 w-4 text-sidebar-primary" />
              ) : (
                <>
                  <ChevronLeft className="h-4 w-4 text-sidebar-primary" />
                  <span className="text-sm">Minimieren</span>
                </>
              )}
            </Button>
          </TooltipTrigger>
          {collapsed && (
            <TooltipContent side="right">Sidebar erweitern</TooltipContent>
          )}
        </Tooltip>
      </div>
    </div>
  )
}

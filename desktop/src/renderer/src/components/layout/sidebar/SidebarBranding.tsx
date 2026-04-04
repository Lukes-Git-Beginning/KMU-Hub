import { LayoutGrid, ChevronLeft, ChevronRight } from 'lucide-react'
import { cn } from '@/lib/cn'
import { branding } from '@/config/branding'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

interface SidebarBrandingProps {
  collapsed: boolean
  onToggle: () => void
  onOpenModules: () => void
}

export function SidebarBranding({ collapsed, onToggle, onOpenModules }: SidebarBrandingProps) {
  return (
    <div className="border-b border-sidebar-border px-3 py-3">
      <div className={cn('flex items-center', collapsed ? 'justify-center' : 'gap-2')}>
        {/* Grid button — opens all-modules panel */}
        <Tooltip delayDuration={300}>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8 shrink-0 text-sidebar-muted hover:bg-sidebar-accent hover:text-sidebar-foreground"
              onClick={onOpenModules}
              aria-label="Alle Module anzeigen"
            >
              <LayoutGrid className="h-4.5 w-4.5" aria-hidden="true" />
            </Button>
          </TooltipTrigger>
          <TooltipContent side={collapsed ? 'right' : 'bottom'}>Alle Module</TooltipContent>
        </Tooltip>

        {/* Branding logo */}
        {!collapsed && (
          <img
            src={branding.cosmi.icon64}
            alt="Cosmi"
            className="max-h-6 w-auto select-none object-contain"
          />
        )}

        {/* Spacer + collapse toggle */}
        {!collapsed && <div className="flex-1" />}
        <Tooltip delayDuration={collapsed ? 100 : 999999}>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              onClick={onToggle}
              className={cn(
                'hidden lg:flex h-7 w-7 shrink-0 text-sidebar-muted hover:bg-sidebar-accent hover:text-sidebar-foreground',
                collapsed && 'mt-2'
              )}
            >
              {collapsed ? (
                <ChevronRight className="h-3.5 w-3.5" />
              ) : (
                <ChevronLeft className="h-3.5 w-3.5" />
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

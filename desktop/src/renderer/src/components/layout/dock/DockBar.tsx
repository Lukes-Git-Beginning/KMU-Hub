import { useRef, useCallback } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { cn } from '@/lib/cn'
import { useFilteredNavItems } from '@/hooks/useFilteredNavItems'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import type { NavItemConfig } from '../sidebar/nav-items'

const MAX_SCALE = 1.5
const MAGNIFY_RANGE = 120 // pixels from center where magnification applies

export function DockBar() {
  const dockRef = useRef<HTMLDivElement>(null)
  const { mainItems, bottomItems } = useFilteredNavItems()

  const handleMouseMove = useCallback((e: React.MouseEvent) => {
    const items = dockRef.current?.querySelectorAll<HTMLElement>('[data-dock-item]')
    if (!items) return
    items.forEach((el) => {
      const rect = el.getBoundingClientRect()
      const center = rect.left + rect.width / 2
      const dist = Math.abs(e.clientX - center)
      const scale =
        dist > MAGNIFY_RANGE
          ? 1
          : 1 + (MAX_SCALE - 1) * Math.cos((dist / MAGNIFY_RANGE) * (Math.PI / 2))
      el.style.transform = `scale(${scale})`
    })
  }, [])

  const handleMouseLeave = useCallback(() => {
    const items = dockRef.current?.querySelectorAll<HTMLElement>('[data-dock-item]')
    items?.forEach((el) => {
      el.style.transform = 'scale(1)'
    })
  }, [])

  const enabledMain = mainItems.filter((i) => i.enabled)
  const enabledBottom = bottomItems.filter((i) => i.enabled)

  return (
    <div className="flex shrink-0 justify-center pb-3 pt-1">
      <nav
        ref={dockRef}
        onMouseMove={handleMouseMove}
        onMouseLeave={handleMouseLeave}
        className="flex items-end gap-1 rounded-2xl border border-border/60 bg-card/80 backdrop-blur-xl px-3 py-2 shadow-xl glass-elevated animate-slide-in-bottom"
      >
        {enabledMain.map((item) => (
          <DockItem key={item.id} item={item} />
        ))}

        {enabledBottom.length > 0 && (
          <div className="mx-1 h-8 w-px bg-border/40" aria-hidden="true" />
        )}

        {enabledBottom.map((item) => (
          <DockItem key={item.id} item={item} />
        ))}
      </nav>
    </div>
  )
}

function DockItem({ item }: { item: NavItemConfig }) {
  const location = useLocation()
  const navigate = useNavigate()
  const Icon = item.icon

  const isActive =
    item.to === '/'
      ? location.pathname === '/'
      : location.pathname.startsWith(item.to)

  return (
    <Tooltip delayDuration={100}>
      <TooltipTrigger asChild>
        <button
          data-dock-item
          onClick={() => navigate(item.to)}
          aria-label={item.label}
          aria-current={isActive ? 'page' : undefined}
          className={cn(
            'relative flex items-center justify-center rounded-xl origin-bottom',
            'h-11 w-11 transition-[transform,background-color] duration-150 ease-out',
            isActive
              ? 'text-primary bg-primary/10'
              : 'text-muted-foreground hover:text-foreground hover:bg-accent/50'
          )}
        >
          <Icon className="h-5 w-5" aria-hidden="true" />

          {/* Badge dot */}
          {item.badge && (
            <span
              className={cn(
                'absolute -top-0.5 -right-0.5 h-2 w-2 rounded-full',
                item.badge.type === 'live'
                  ? 'bg-red-500 animate-pulse'
                  : 'bg-primary'
              )}
            />
          )}

          {/* Active indicator */}
          {isActive && (
            <span className="absolute -bottom-1.5 left-1/2 -translate-x-1/2 h-1 w-1 rounded-full bg-primary" />
          )}
        </button>
      </TooltipTrigger>
      <TooltipContent side="top" className="text-xs">
        {item.label}
      </TooltipContent>
    </Tooltip>
  )
}

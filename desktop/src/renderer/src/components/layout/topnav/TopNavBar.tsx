import { useRef, useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { NavLink } from 'react-router-dom'
import { LayoutGrid } from 'lucide-react'
import { cn } from '@/lib/cn'
import { useFilteredNavItems } from '@/hooks/useFilteredNavItems'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { Button } from '@/components/ui/button'
import { ModuleOverviewPanel } from './ModuleOverviewPanel'
import type { NavItemConfig } from '../sidebar/nav-items'

/**
 * Compact horizontal module tab strip.
 * Rendered directly below the header bar in topnav layout.
 * Supports mouse-wheel horizontal scrolling.
 */
export function TopNavBar() {
  const { t } = useTranslation()
  const scrollRef = useRef<HTMLDivElement>(null)
  const { mainItems, bottomItems } = useFilteredNavItems()
  const enabledItems = mainItems.filter((i) => i.enabled)
  const allEnabled = [...enabledItems, ...bottomItems.filter((i) => i.enabled)]

  const [overviewOpen, setOverviewOpen] = useState(false)

  // Horizontal mouse-wheel scrolling
  const handleWheel = useCallback((e: WheelEvent) => {
    const el = scrollRef.current
    if (!el) return
    // Prevent vertical page scroll, scroll horizontally instead
    if (Math.abs(e.deltaY) > Math.abs(e.deltaX)) {
      e.preventDefault()
      el.scrollLeft += e.deltaY
    }
  }, [])

  useEffect(() => {
    const el = scrollRef.current
    if (!el) return
    el.addEventListener('wheel', handleWheel, { passive: false })
    return () => el.removeEventListener('wheel', handleWheel)
  }, [handleWheel])

  return (
    <nav className="flex shrink-0 h-9 items-center border-b border-border/50 bg-card/50 px-2 gap-1">
      {/* Grid button — opens module overview */}
      <Popover open={overviewOpen} onOpenChange={setOverviewOpen}>
        <PopoverTrigger asChild>
          <Button variant="ghost" size="icon" className="h-7 w-7 shrink-0" aria-label={t('layout.topnav.showModules')}>
            <LayoutGrid className="h-4 w-4" aria-hidden="true" />
          </Button>
        </PopoverTrigger>
        <PopoverContent align="start" className="w-[480px] p-0" sideOffset={8}>
          <ModuleOverviewPanel
            items={allEnabled}
            onSelect={() => setOverviewOpen(false)}
          />
        </PopoverContent>
      </Popover>

      {/* Scrollable module tabs — no scrollbar, wheel-scrollable */}
      <div
        ref={scrollRef}
        className="flex flex-1 min-w-0 items-center gap-0.5 overflow-x-auto scrollbar-hide"
      >
        {enabledItems.map((item) => (
          <TopNavTab key={item.id} item={item} />
        ))}
      </div>
    </nav>
  )
}

function TopNavTab({ item }: { item: NavItemConfig }) {
  const Icon = item.icon

  return (
    <NavLink
      to={item.to}
      end={item.to === '/'}
      onClick={(e) => {
        if (item.onActivate) {
          e.preventDefault()
          item.onActivate()
        }
      }}
      className={({ isActive }) =>
        cn(
          'flex shrink-0 items-center gap-1.5 rounded-md px-2.5 py-1 text-xs font-medium',
          'transition-all duration-150 whitespace-nowrap',
          isActive
            ? 'bg-primary/10 text-primary'
            : 'text-muted-foreground hover:text-foreground hover:bg-accent/50'
        )
      }
    >
      <span
        className="flex items-center justify-center h-4 w-4 rounded"
        style={{
          color: `hsl(${item.color.h} ${item.color.s}% 42%)`,
        }}
      >
        <Icon className="h-3 w-3" />
      </span>
      <span>{item.label}</span>
      {item.badge && (
        <span
          className={cn(
            'h-1.5 w-1.5 rounded-full',
            item.badge.type === 'live' ? 'bg-destructive animate-pulse' : 'bg-primary'
          )}
        />
      )}
    </NavLink>
  )
}

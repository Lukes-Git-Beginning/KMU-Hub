import { useRef, useState, useEffect } from 'react'
import { NavLink } from 'react-router-dom'
import { LayoutGrid, ChevronLeft, ChevronRight } from 'lucide-react'
import { cn } from '@/lib/cn'
import { useOnlineStatus } from '@/hooks/useOnlineStatus'
import { useNotificationWebSocket } from '@/api/hooks/useNotifications'
import { useFilteredNavItems } from '@/hooks/useFilteredNavItems'
import { NotificationBell } from '@/modules/notifications/NotificationBell'
import { PresenceStatusPicker } from '@/features/presence'
import {
  SearchBar,
  HeaderClock,
  ProfileMenu,
} from '@/components/header'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { Button } from '@/components/ui/button'
import { ModuleOverviewPanel } from './ModuleOverviewPanel'
import type { NavItemConfig } from '../sidebar/nav-items'

export function TopNavBar() {
  const scrollRef = useRef<HTMLDivElement>(null)
  const { isOnline } = useOnlineStatus()
  const { mainItems, bottomItems } = useFilteredNavItems()
  const enabledItems = mainItems.filter((i) => i.enabled)
  const allEnabled = [...enabledItems, ...bottomItems.filter((i) => i.enabled)]

  const [canScrollLeft, setCanScrollLeft] = useState(false)
  const [canScrollRight, setCanScrollRight] = useState(false)
  const [overviewOpen, setOverviewOpen] = useState(false)

  useNotificationWebSocket()

  // Detect scroll overflow
  useEffect(() => {
    const el = scrollRef.current
    if (!el) return
    const check = () => {
      setCanScrollLeft(el.scrollLeft > 4)
      setCanScrollRight(el.scrollLeft < el.scrollWidth - el.clientWidth - 4)
    }
    check()
    el.addEventListener('scroll', check, { passive: true })
    const ro = new ResizeObserver(check)
    ro.observe(el)
    return () => {
      el.removeEventListener('scroll', check)
      ro.disconnect()
    }
  }, [enabledItems.length])

  const scroll = (dir: 'left' | 'right') => {
    scrollRef.current?.scrollBy({ left: dir === 'left' ? -200 : 200, behavior: 'smooth' })
  }

  return (
    <header className="flex shrink-0 h-14 items-center border-b border-header-border bg-header-background px-3 gap-2 glass-surface">
      {/* Grid button — opens module overview */}
      <Popover open={overviewOpen} onOpenChange={setOverviewOpen}>
        <PopoverTrigger asChild>
          <Button variant="ghost" size="icon" className="h-9 w-9 shrink-0">
            <LayoutGrid className="h-5 w-5" />
          </Button>
        </PopoverTrigger>
        <PopoverContent align="start" className="w-[480px] p-0" sideOffset={8}>
          <ModuleOverviewPanel
            items={allEnabled}
            onSelect={() => setOverviewOpen(false)}
          />
        </PopoverContent>
      </Popover>

      {/* Scroll left arrow */}
      {canScrollLeft && (
        <button
          onClick={() => scroll('left')}
          className="shrink-0 rounded-md p-1 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
        >
          <ChevronLeft className="h-4 w-4" />
        </button>
      )}

      {/* Scrollable module tabs */}
      <div
        ref={scrollRef}
        className="flex flex-1 min-w-0 items-center gap-0.5 overflow-x-auto topnav-scroll"
      >
        {enabledItems.map((item) => (
          <TopNavTab key={item.id} item={item} />
        ))}
      </div>

      {/* Scroll right arrow */}
      {canScrollRight && (
        <button
          onClick={() => scroll('right')}
          className="shrink-0 rounded-md p-1 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
        >
          <ChevronRight className="h-4 w-4" />
        </button>
      )}

      {/* Separator */}
      <div className="h-6 w-px bg-border/50" />

      {/* Header widgets */}
      <div className="flex shrink-0 items-center gap-1.5">
        <SearchBar />
        <div className="hidden sm:block">
          <HeaderClock />
        </div>
        <PresenceStatusPicker />
        <NotificationBell />
        <Tooltip>
          <TooltipTrigger asChild>
            <span
              className={`inline-block h-2 w-2 rounded-full ${isOnline ? 'bg-emerald-500' : 'bg-destructive'}`}
              aria-label={isOnline ? 'Online' : 'Offline'}
            />
          </TooltipTrigger>
          <TooltipContent>
            {isOnline ? 'Verbunden' : 'Keine Verbindung'}
          </TooltipContent>
        </Tooltip>
        <ProfileMenu />
      </div>
    </header>
  )
}

function TopNavTab({ item }: { item: NavItemConfig }) {
  const Icon = item.icon

  return (
    <NavLink
      to={item.to}
      end={item.to === '/'}
      className={({ isActive }) =>
        cn(
          'flex shrink-0 items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium',
          'transition-all duration-150 whitespace-nowrap',
          isActive
            ? 'bg-primary/10 text-primary'
            : 'text-muted-foreground hover:text-foreground hover:bg-accent/50'
        )
      }
    >
      <Icon className="h-4 w-4" />
      <span>{item.label}</span>
      {item.badge && (
        <span
          className={cn(
            'h-1.5 w-1.5 rounded-full',
            item.badge.type === 'live' ? 'bg-red-500 animate-pulse' : 'bg-primary'
          )}
        />
      )}
    </NavLink>
  )
}

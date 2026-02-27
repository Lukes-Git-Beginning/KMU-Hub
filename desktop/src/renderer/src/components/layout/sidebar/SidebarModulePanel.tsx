import { useNavigate, useLocation } from 'react-router-dom'
import { Pin, PinOff } from 'lucide-react'
import { cn } from '@/lib/cn'
import { useUIStore } from '@/stores/ui'
import type { NavItemConfig } from './nav-items'

interface SidebarModulePanelProps {
  items: NavItemConfig[]
  onSelect?: () => void
}

export function SidebarModulePanel({ items, onSelect }: SidebarModulePanelProps) {
  const navigate = useNavigate()
  const location = useLocation()
  const pinnedModules = useUIStore((s) => s.pinnedModules)
  const togglePinModule = useUIStore((s) => s.togglePinModule)

  return (
    <div className="p-3">
      <p className="text-[11px] font-medium text-muted-foreground uppercase tracking-wider mb-2 px-1">
        Alle Module
      </p>
      <div className="grid grid-cols-3 gap-1.5">
        {items.map((item) => {
          const Icon = item.icon
          const isPinned = pinnedModules.includes(item.id)
          const isActive =
            item.to === '/'
              ? location.pathname === '/'
              : location.pathname.startsWith(item.to)

          const iconStyle = {
            color: `hsl(${item.color.h} ${item.color.s}% 42%)`,
          }

          return (
            <div key={item.id} className="relative group">
              <button
                onClick={() => {
                  navigate(item.to)
                  onSelect?.()
                }}
                className={cn(
                  'flex flex-col items-center gap-1 rounded-lg p-2 w-full text-center transition-all duration-150',
                  isActive
                    ? 'bg-primary/10 text-primary'
                    : 'text-muted-foreground hover:text-foreground hover:bg-accent/50'
                )}
              >
                <div
                  className={cn(
                    'flex items-center justify-center h-8 w-8 rounded-lg',
                    isActive ? 'bg-primary/15' : 'bg-muted/50'
                  )}
                >
                  <Icon className="h-4 w-4" style={iconStyle} />
                </div>
                <span className="text-[10px] font-medium leading-tight truncate w-full">
                  {item.label}
                </span>
              </button>

              {/* Pin/unpin toggle */}
              <button
                onClick={(e) => {
                  e.stopPropagation()
                  togglePinModule(item.id)
                }}
                className={cn(
                  'absolute top-0.5 right-0.5 h-5 w-5 rounded-md flex items-center justify-center transition-opacity',
                  isPinned
                    ? 'text-primary opacity-80 hover:opacity-100'
                    : 'text-muted-foreground opacity-0 group-hover:opacity-60 hover:!opacity-100'
                )}
                title={isPinned ? 'Aus Sidebar entfernen' : 'Zur Sidebar hinzufuegen'}
              >
                {isPinned ? (
                  <Pin className="h-3 w-3" />
                ) : (
                  <PinOff className="h-3 w-3" />
                )}
              </button>
            </div>
          )
        })}
      </div>
    </div>
  )
}

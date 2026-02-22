import { useNavigate, useLocation } from 'react-router-dom'
import { cn } from '@/lib/cn'
import type { NavItemConfig } from '../sidebar/nav-items'

interface ModuleOverviewPanelProps {
  items: NavItemConfig[]
  onSelect?: () => void
}

export function ModuleOverviewPanel({ items, onSelect }: ModuleOverviewPanelProps) {
  const navigate = useNavigate()
  const location = useLocation()

  return (
    <div className="p-4">
      <h3 className="text-sm font-semibold text-foreground mb-3">Alle Module</h3>
      <div className="grid grid-cols-4 gap-2">
        {items.map((item) => {
          const Icon = item.icon
          const isActive =
            item.to === '/'
              ? location.pathname === '/'
              : location.pathname.startsWith(item.to)

          return (
            <button
              key={item.id}
              onClick={() => {
                navigate(item.to)
                onSelect?.()
              }}
              className={cn(
                'flex flex-col items-center gap-1.5 rounded-xl p-3 text-center transition-all duration-150',
                isActive
                  ? 'bg-primary/10 text-primary'
                  : 'text-muted-foreground hover:text-foreground hover:bg-accent/50'
              )}
            >
              <div
                className={cn(
                  'flex items-center justify-center h-10 w-10 rounded-xl',
                  isActive ? 'bg-primary/15' : 'bg-muted/50'
                )}
              >
                <Icon className="h-5 w-5" />
              </div>
              <span className="text-[11px] font-medium leading-tight">
                {item.label}
              </span>
            </button>
          )
        })}
      </div>
    </div>
  )
}

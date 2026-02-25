import { cn } from '@/lib'
import { moduleHsl } from '@/components/layout/sidebar/nav-items'
import { ChevronRight } from 'lucide-react'

interface PageHeaderProps {
  title: string
  description?: string
  actions?: React.ReactNode
  /** Lucide icon shown left of the title */
  icon?: React.ElementType
  /** nav-items module ID — colors the icon + title */
  moduleId?: string
  /** Breadcrumb trail, e.g. ['Finanzen', 'Rechnungen'] */
  breadcrumb?: string[]
  /** Slot for tab bar rendered below the header */
  tabs?: React.ReactNode
  gradient?: boolean
  className?: string
}

export function PageHeader({
  title,
  description,
  actions,
  icon: Icon,
  moduleId,
  breadcrumb,
  tabs,
  gradient = false,
  className,
}: PageHeaderProps) {
  const accentColor = moduleId ? moduleHsl(moduleId) : undefined

  return (
    <div className={cn('animate-fade-up', className)}>
      {/* Breadcrumb */}
      {breadcrumb && breadcrumb.length > 0 && (
        <div className="flex items-center gap-1 text-xs text-muted-foreground mb-2">
          {breadcrumb.map((crumb, i) => (
            <span key={i} className="flex items-center gap-1">
              {i > 0 && <ChevronRight className="h-3 w-3" />}
              <span className={i === breadcrumb.length - 1 ? 'text-foreground font-medium' : ''}>
                {crumb}
              </span>
            </span>
          ))}
        </div>
      )}

      {/* Title row */}
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-center gap-3 min-w-0">
          {/* Colored icon */}
          {Icon && (
            <div
              className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl"
              style={accentColor ? { backgroundColor: accentColor + '15', color: accentColor } : undefined}
            >
              <Icon className="h-5 w-5" />
            </div>
          )}
          <div className="min-w-0">
            <h1
              className="text-2xl font-bold tracking-tight"
              style={accentColor ? { color: accentColor } : undefined}
            >
              {title}
            </h1>
            {description && (
              <p className="mt-0.5 text-sm text-muted-foreground">{description}</p>
            )}
          </div>
        </div>
        {actions && <div className="flex items-center gap-2 shrink-0">{actions}</div>}
      </div>

      {/* Tabs slot */}
      {tabs}
    </div>
  )
}

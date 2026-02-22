import { cn } from '@/lib'

interface PageHeaderProps {
  title: string
  description?: string
  actions?: React.ReactNode
  gradient?: boolean
  className?: string
}

export function PageHeader({
  title,
  description,
  actions,
  gradient = false,
  className,
}: PageHeaderProps) {
  return (
    <div className={cn('flex items-start justify-between gap-4 animate-fade-up', className)}>
      <div className="min-w-0">
        <h1
          className={cn(
            'text-2xl font-bold tracking-tight',
            gradient
              ? 'bg-gradient-to-r from-[var(--accent-1)] to-[var(--accent-2)] bg-clip-text text-transparent'
              : 'text-foreground'
          )}
        >
          {title}
        </h1>
        {description && (
          <p className="mt-1 text-sm text-muted-foreground">{description}</p>
        )}
      </div>
      {actions && <div className="flex items-center gap-2 shrink-0">{actions}</div>}
    </div>
  )
}

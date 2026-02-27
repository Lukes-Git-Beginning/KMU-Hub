import type { LucideIcon } from 'lucide-react'
import type { ReactNode } from 'react'
import { Button } from '@/components/ui/button'

interface EmptyStateProps {
  icon?: LucideIcon
  illustration?: ReactNode
  title: string
  description?: string
  action?: {
    label: string
    onClick: () => void
  }
}

export function EmptyState({ icon: Icon, illustration, title, description, action }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-16 animate-fade-up">
      {illustration ? (
        <div className="mb-2" aria-hidden="true">
          {illustration}
        </div>
      ) : Icon ? (
        <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-primary/10" aria-hidden="true">
          <Icon className="h-8 w-8 text-primary/60" />
        </div>
      ) : null}
      <p className="mt-5 text-lg font-semibold text-foreground">{title}</p>
      {description && (
        <p className="mt-1.5 max-w-sm text-center text-sm text-muted-foreground">
          {description}
        </p>
      )}
      {action && (
        <Button className="mt-5" onClick={action.onClick}>
          {action.label}
        </Button>
      )}
    </div>
  )
}

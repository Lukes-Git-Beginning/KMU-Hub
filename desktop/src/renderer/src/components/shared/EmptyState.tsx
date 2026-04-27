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
        <div className="flex h-14 w-14 items-center justify-center rounded-xl bg-secondary" aria-hidden="true">
          <Icon className="h-7 w-7 text-muted-foreground" />
        </div>
      ) : null}
      <h3 className="mt-5 text-lg font-semibold text-foreground">{title}</h3>
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

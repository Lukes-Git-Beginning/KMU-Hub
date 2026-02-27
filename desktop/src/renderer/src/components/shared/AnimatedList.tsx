import { Children } from 'react'
import { cn } from '@/lib'

interface AnimatedListProps {
  children: React.ReactNode
  className?: string
  /** Delay in ms between each item */
  staggerMs?: number
}

export function AnimatedList({
  children,
  className,
  staggerMs = 50,
}: AnimatedListProps) {
  return (
    <div className={cn('space-y-2', className)}>
      {Children.map(children, (child, i) => (
        <div
          className="animate-fade-up"
          style={{ animationDelay: `${i * staggerMs}ms` }}
        >
          {child}
        </div>
      ))}
    </div>
  )
}

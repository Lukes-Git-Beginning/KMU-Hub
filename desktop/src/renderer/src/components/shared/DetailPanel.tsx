import { useEffect, useCallback } from 'react'
import { X, Maximize2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { cn } from '@/lib'

interface DetailPanelProps {
  open: boolean
  onClose: () => void
  title?: string
  subtitle?: string
  badge?: React.ReactNode
  onExpand?: () => void
  children: React.ReactNode
  footer?: React.ReactNode
  className?: string
  width?: string
}

export function DetailPanel({
  open,
  onClose,
  title,
  subtitle,
  badge,
  onExpand,
  children,
  footer,
  className,
  width = 'w-[400px]',
}: DetailPanelProps) {
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    },
    [onClose]
  )

  useEffect(() => {
    if (open) {
      document.addEventListener('keydown', handleKeyDown)
      return () => document.removeEventListener('keydown', handleKeyDown)
    }
  }, [open, handleKeyDown])

  if (!open) return null

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 z-40 bg-black/20 backdrop-blur-[2px] transition-opacity animate-fade-in"
        onClick={onClose}
      />

      {/* Panel */}
      <div
        className={cn(
          'fixed right-0 top-0 z-50 flex h-full flex-col border-l bg-[var(--card)] shadow-xl glass-elevated',
          'animate-in slide-in-from-right duration-300 ease-[cubic-bezier(0.32,0.72,0,1)]',
          width,
          className
        )}
      >
        {/* Gradient Header Stripe */}
        <div className="h-0.5 w-full bg-gradient-to-r from-[var(--accent-1)] to-[var(--accent-2)] shrink-0" />

        {/* Header */}
        {(title || badge) && (
          <div className="flex items-center justify-between border-b px-4 py-3">
            <div className="flex items-center gap-2 min-w-0">
              {title && (
                <div className="min-w-0">
                  <h3 className="truncate text-sm font-semibold text-foreground">
                    {title}
                  </h3>
                  {subtitle && (
                    <p className="truncate text-xs text-muted-foreground">
                      {subtitle}
                    </p>
                  )}
                </div>
              )}
              {badge}
            </div>
            <div className="flex items-center gap-1 shrink-0">
              {onExpand && (
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-7 w-7"
                  onClick={onExpand}
                >
                  <Maximize2 className="h-3.5 w-3.5" />
                </Button>
              )}
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7"
                onClick={onClose}
              >
                <X className="h-3.5 w-3.5" />
              </Button>
            </div>
          </div>
        )}

        {/* Body */}
        <ScrollArea className="flex-1">
          <div className="p-4">{children}</div>
        </ScrollArea>

        {/* Footer */}
        {footer && (
          <div className="border-t px-4 py-3">{footer}</div>
        )}
      </div>
    </>
  )
}

/**
 * LinkPreviewPopover — a small, reusable preview card shown when a user clicks an
 * inline cross-reference (a wiki [[link]], an @mention, …). It surfaces who/what
 * the target is and a single "more info" action that jumps to it.
 *
 * Module-agnostic: the caller resolves the target (icon, name, subtitle) and
 * supplies the jump action, so berichte and any other document surface can reuse
 * the same pattern. Positioned at a fixed anchor rect, dismissed on outside
 * click or Escape.
 */
import { useEffect, useRef, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { ArrowRight, X } from 'lucide-react'

export interface LinkPreviewPopoverProps {
  /** Bounding rect of the clicked link; null hides the popover. */
  anchor: DOMRect | null
  icon: ReactNode
  name: string
  subtitle?: string
  /** Label for the jump action (e.g. "Weitere Infos"). */
  actionLabel: string
  onAction: () => void
  onClose: () => void
  /** Accessible label for the close button. */
  closeLabel: string
}

export function LinkPreviewPopover({
  anchor,
  icon,
  name,
  subtitle,
  actionLabel,
  onAction,
  onClose,
  closeLabel,
}: LinkPreviewPopoverProps) {
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!anchor) return
    const onDown = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) onClose()
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    // Defer the outside-click listener so the opening click doesn't dismiss it.
    const id = window.setTimeout(() => document.addEventListener('mousedown', onDown), 0)
    document.addEventListener('keydown', onKey)
    return () => {
      window.clearTimeout(id)
      document.removeEventListener('mousedown', onDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [anchor, onClose])

  if (!anchor) return null

  const top = Math.min(anchor.bottom + 6, window.innerHeight - 130)
  const left = Math.max(8, Math.min(anchor.left, window.innerWidth - 296))

  return createPortal(
    <div
      ref={ref}
      style={{ position: 'fixed', top, left, zIndex: 60 }}
      className="w-72 rounded-xl border border-border bg-popover p-3 shadow-lg animate-fade-up"
    >
      <div className="flex items-start gap-2.5">
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-secondary text-muted-foreground">
          {icon}
        </div>
        <div className="min-w-0 flex-1 pt-0.5">
          <p className="truncate text-sm font-semibold text-foreground">{name}</p>
          {subtitle && <p className="truncate text-xs text-muted-foreground">{subtitle}</p>}
        </div>
        <button
          type="button"
          onClick={onClose}
          aria-label={closeLabel}
          className="flex h-6 w-6 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
        >
          <X className="h-3.5 w-3.5" />
        </button>
      </div>
      <button
        type="button"
        onClick={onAction}
        className="mt-2.5 flex w-full items-center justify-center gap-1.5 rounded-lg bg-primary/10 px-3 py-1.5 text-xs font-medium text-primary transition-colors hover:bg-primary/15"
      >
        {actionLabel}
        <ArrowRight className="h-3.5 w-3.5" />
      </button>
    </div>,
    document.body,
  )
}

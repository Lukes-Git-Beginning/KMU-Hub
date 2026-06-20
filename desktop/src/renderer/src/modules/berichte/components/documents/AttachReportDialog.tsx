import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Search, type LucideIcon } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Skeleton } from '@/components/shared'

export interface AttachItem {
  id: string
  label: string
  /** Optional second line (e.g. company / project). */
  hint?: string
}

interface AttachReportDialogProps {
  open: boolean
  onClose: () => void
  /** Report title shown as the dialog subtitle. */
  subtitle: string
  icon: LucideIcon
  title: string
  searchPlaceholder: string
  emptyLabel: string
  items: AttachItem[]
  isLoading: boolean
  pending: boolean
  onPick: (item: AttachItem) => void
}

/**
 * Shared presentation for R-5 "attach report to X" pickers (task, contact, …):
 * a searchable, scrollable list of targets. Data + mutation live in thin
 * wrappers so this stays purely presentational and reusable.
 */
export function AttachReportDialog({
  open,
  onClose,
  subtitle,
  icon: Icon,
  title,
  searchPlaceholder,
  emptyLabel,
  items,
  isLoading,
  pending,
  onPick,
}: AttachReportDialogProps) {
  const { t } = useTranslation()
  const [query, setQuery] = useState('')

  const matches = useMemo(() => {
    const q = query.trim().toLowerCase()
    const list = q
      ? items.filter(
          (it) =>
            it.label.toLowerCase().includes(q) || (it.hint ?? '').toLowerCase().includes(q),
        )
      : items
    return list.slice(0, 8)
  }, [items, query])

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-w-md gap-0 p-0">
        <DialogHeader className="border-b border-border px-6 py-4">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary-light">
              <Icon className="h-4 w-4 text-primary" />
            </div>
            <div className="min-w-0">
              <DialogTitle className="text-sm">{title}</DialogTitle>
              <DialogDescription className="truncate">{subtitle}</DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <div className="px-6 py-5">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
            <input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder={searchPlaceholder}
              aria-label={searchPlaceholder}
              className="w-full rounded-lg border border-border bg-card py-2 pl-9 pr-3 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>

          <div className="mt-3 max-h-72 space-y-1 overflow-y-auto" aria-busy={isLoading}>
            {isLoading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className="h-10 w-full rounded-lg" />
              ))
            ) : matches.length === 0 ? (
              <p className="px-1 py-6 text-center text-sm text-muted-foreground">{emptyLabel}</p>
            ) : (
              matches.map((it) => (
                <button
                  key={it.id}
                  type="button"
                  disabled={pending}
                  onClick={() => onPick(it)}
                  className="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-left transition-colors hover:bg-secondary disabled:opacity-50"
                >
                  <Icon className="h-4 w-4 shrink-0 text-muted-foreground" />
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-sm text-foreground">{it.label}</span>
                    {it.hint && (
                      <span className="block truncate text-xs text-muted-foreground">{it.hint}</span>
                    )}
                  </span>
                </button>
              ))
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

import { cn } from '@/lib'

/**
 * Skeleton — Lade-Gerüst statt Spinner (projektweiter Cosmi-Standard).
 *
 * Zeigt die grobe Form der gleich erscheinenden UI mit einem sanften
 * Türkis-Licht-Sweep (GPU-only, `.skeleton` in styles/animations.css).
 * Basis-Baustein `Skeleton` + fertige Layouts (Text/Card/List/Table).
 *
 * Respektiert `prefers-reduced-motion` automatisch (Sweep wird dann zum
 * ruhigen Platzhalter).
 */

interface SkeletonProps {
  className?: string
}

/** Einzelner Skelett-Block. Form/Größe über className (z.B. `h-4 w-32 rounded-full`). */
export function Skeleton({ className }: SkeletonProps) {
  return <div className={cn('skeleton', className)} aria-hidden="true" />
}

/** Mehrere Textzeilen; die letzte Zeile ist kürzer (wie echter Fließtext). */
export function SkeletonText({ lines = 3, className }: { lines?: number; className?: string }) {
  return (
    <div className={cn('space-y-2', className)} aria-hidden="true">
      {Array.from({ length: lines }).map((_, i) => (
        <Skeleton key={i} className={cn('h-3.5', i === lines - 1 ? 'w-2/3' : 'w-full')} />
      ))}
    </div>
  )
}

/** Karten-Gerüst: Avatar + Titel + zwei Zeilen. Für Listen-/Grid-Karten. */
export function SkeletonCard({ className }: SkeletonProps) {
  return (
    <div className={cn('rounded-lg border border-border bg-card p-4', className)} aria-hidden="true">
      <div className="flex items-center gap-3">
        <Skeleton className="h-10 w-10 rounded-full" />
        <div className="flex-1 space-y-2">
          <Skeleton className="h-3.5 w-1/3" />
          <Skeleton className="h-3 w-1/2" />
        </div>
      </div>
      <div className="mt-4 space-y-2">
        <Skeleton className="h-3 w-full" />
        <Skeleton className="h-3 w-4/5" />
      </div>
    </div>
  )
}

/** Liste aus Karten-Gerüsten. */
export function SkeletonList({ rows = 4, className }: { rows?: number; className?: string }) {
  return (
    <div className={cn('space-y-3', className)} aria-hidden="true">
      {Array.from({ length: rows }).map((_, i) => (
        <SkeletonCard key={i} />
      ))}
    </div>
  )
}

/** Tabellen-Gerüst: Kopfzeile + Datenzeilen. */
export function SkeletonTable({ rows = 6, cols = 4, className }: { rows?: number; cols?: number; className?: string }) {
  return (
    <div className={cn('overflow-hidden rounded-lg border border-border', className)} aria-hidden="true">
      <div className="flex items-center gap-4 border-b border-border bg-secondary/40 px-4 py-3">
        {Array.from({ length: cols }).map((_, i) => (
          <Skeleton key={i} className="h-3 flex-1" />
        ))}
      </div>
      <div className="divide-y divide-border-muted">
        {Array.from({ length: rows }).map((_, r) => (
          <div key={r} className="flex items-center gap-4 px-4 py-3">
            {Array.from({ length: cols }).map((_, c) => (
              <Skeleton key={c} className={cn('h-3.5', c === 0 ? 'flex-[1.5]' : 'flex-1')} />
            ))}
          </div>
        ))}
      </div>
    </div>
  )
}

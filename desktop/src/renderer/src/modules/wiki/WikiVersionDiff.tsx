import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Minus } from 'lucide-react'
import { diffHtml } from './wikiDiff'

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface WikiVersionDiffProps {
  /** Older snapshot (this version). */
  oldContent: string
  /** Newer snapshot (current version). */
  newContent: string
}

/**
 * Renders a line-level text diff between an older version and the current one.
 * Added lines are shown in success colours, removed lines in destructive — so
 * the change is legible before a restore is triggered.
 */
export function WikiVersionDiff({ oldContent, newContent }: WikiVersionDiffProps) {
  const { t } = useTranslation()
  const rows = useMemo(() => diffHtml(oldContent, newContent), [oldContent, newContent])
  const hasChanges = rows.some((r) => r.type !== 'same')

  if (!hasChanges) {
    return (
      <p className="mt-2 rounded-md bg-muted/50 px-2.5 py-2 text-[11px] text-muted-foreground">
        {t('wiki.version.diffNoChanges')}
      </p>
    )
  }

  return (
    <div className="mt-2 space-y-1.5">
      {/* Legend */}
      <div className="flex items-center gap-3 text-[10px] text-muted-foreground">
        <span className="inline-flex items-center gap-1">
          <span className="inline-block h-2 w-2 rounded-sm bg-success" />
          {t('wiki.version.diffAdded')}
        </span>
        <span className="inline-flex items-center gap-1">
          <span className="inline-block h-2 w-2 rounded-sm bg-destructive" />
          {t('wiki.version.diffRemoved')}
        </span>
      </div>

      {/* Diff lines */}
      <div className="overflow-hidden rounded-md border border-border">
        {rows.map((row, i) => {
          if (row.type === 'same') {
            return (
              <div
                key={i}
                className="flex gap-1.5 px-2 py-1 text-[11px] text-muted-foreground/70"
              >
                <span className="w-3 shrink-0 text-center" aria-hidden="true">
                  ·
                </span>
                <span className="min-w-0 break-words">{row.text}</span>
              </div>
            )
          }
          const added = row.type === 'added'
          return (
            <div
              key={i}
              className={`flex gap-1.5 px-2 py-1 text-[11px] ${
                added ? 'bg-success-light text-success' : 'bg-destructive/10 text-destructive'
              }`}
            >
              <span className="w-3 shrink-0 pt-0.5" aria-hidden="true">
                {added ? <Plus className="h-2.5 w-2.5" /> : <Minus className="h-2.5 w-2.5" />}
              </span>
              <span className="min-w-0 break-words">{row.text}</span>
            </div>
          )
        })}
      </div>
    </div>
  )
}

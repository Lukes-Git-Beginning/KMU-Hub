import { useTranslation } from 'react-i18next'
import { Clock, RotateCcw, Loader2 } from 'lucide-react'
import type { WikiVersion } from '@/types/wiki'
import { formatDate as libFormatDate } from '@/lib/format'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function formatShortDate(dateStr: string): string {
  const normalized = dateStr.includes('T') ? dateStr : dateStr + 'T00:00:00'
  return libFormatDate(normalized, { day: '2-digit', month: 'short', year: 'numeric' })
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface WikiVersionItemProps {
  version: WikiVersion
  isCurrent: boolean
  /** Restore this version (omitted for the current version). */
  onRestore?: (versionId: string) => void
  restoring?: boolean
}

export function WikiVersionItem({ version, isCurrent, onRestore, restoring }: WikiVersionItemProps) {
  const { t } = useTranslation()
  return (
    <div className={`group flex items-start gap-2.5 rounded-md px-2.5 py-2 ${
      isCurrent ? 'bg-primary/5' : 'hover:bg-accent/50'
    } transition-colors`}>
      {/* Version badge */}
      <span className={`shrink-0 rounded px-1.5 py-0.5 font-mono text-[10px] font-medium ${
        isCurrent ? 'bg-primary/15 text-primary' : 'bg-secondary text-muted-foreground'
      }`}>
        v{version.version}
      </span>

      {/* Content */}
      <div className="min-w-0 flex-1">
        <p className="text-xs font-medium text-foreground">
          {version.editorName}
          {isCurrent && (
            <span className="ml-1.5 text-[10px] text-primary font-normal">{t('wiki.version.current')}</span>
          )}
        </p>
        {version.changeNote && (
          <p className="text-[11px] text-muted-foreground italic mt-0.5">{version.changeNote}</p>
        )}
        <div className="flex items-center gap-1 mt-0.5 text-[10px] text-muted-foreground">
          <Clock className="h-2.5 w-2.5" />
          {formatShortDate(version.editedAt)}
        </div>
      </div>

      {/* Restore action */}
      {onRestore && (
        <button
          onClick={() => onRestore(version.id)}
          disabled={restoring}
          className="shrink-0 rounded p-1 text-muted-foreground opacity-0 transition-opacity hover:bg-accent hover:text-foreground group-hover:opacity-100 focus-visible:opacity-100 disabled:opacity-100"
          title={t('wiki.version.restore')}
        >
          {restoring ? (
            <Loader2 className="h-3 w-3 animate-spin" />
          ) : (
            <RotateCcw className="h-3 w-3" />
          )}
        </button>
      )}
    </div>
  )
}

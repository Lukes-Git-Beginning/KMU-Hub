import { useTranslation } from 'react-i18next'
import { Clock } from 'lucide-react'
import type { WikiVersion } from '@/types/wiki'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function formatDate(dateStr: string): string {
  try {
    return new Date(dateStr + (dateStr.includes('T') ? '' : 'T00:00:00')).toLocaleDateString('de-DE', {
      day: '2-digit',
      month: 'short',
      year: 'numeric',
    })
  } catch {
    return dateStr
  }
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface WikiVersionItemProps {
  version: WikiVersion
  isCurrent: boolean
}

export function WikiVersionItem({ version, isCurrent }: WikiVersionItemProps) {
  const { t } = useTranslation()
  return (
    <div className={`flex items-start gap-2.5 rounded-md px-2.5 py-2 ${
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
        <p className="text-[11px] text-muted-foreground italic mt-0.5">{version.changeNote}</p>
        <div className="flex items-center gap-1 mt-0.5 text-[10px] text-muted-foreground">
          <Clock className="h-2.5 w-2.5" />
          {formatDate(version.editedAt)}
        </div>
      </div>
    </div>
  )
}

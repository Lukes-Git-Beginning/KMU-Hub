import { useTranslation } from 'react-i18next'
import { X, History } from 'lucide-react'
import type { WikiVersion } from '@/types/wiki'
import { WikiVersionItem } from './WikiVersionItem'

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface WikiVersionHistoryProps {
  versions: WikiVersion[]
  open: boolean
  onClose: () => void
}

export function WikiVersionHistory({ versions, open, onClose }: WikiVersionHistoryProps) {
  const { t } = useTranslation()
  if (!open) return null

  return (
    <div className="flex w-64 shrink-0 flex-col border-l border-border bg-card/30 overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-3 py-2.5 border-b border-border">
        <div className="flex items-center gap-1.5">
          <History className="h-3.5 w-3.5 text-muted-foreground" />
          <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
            {t('wiki.version.title')}
          </h3>
          <span className="text-[10px] text-muted-foreground">({(versions ?? []).length})</span>
        </div>
        <button
          onClick={onClose}
          className="rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
        >
          <X className="h-3.5 w-3.5" />
        </button>
      </div>

      {/* Version list */}
      <div className="flex-1 overflow-y-auto p-2 space-y-1">
        {(versions ?? []).map((v, idx) => (
          <WikiVersionItem
            key={v.id}
            version={v}
            isCurrent={idx === 0}
          />
        ))}
      </div>
    </div>
  )
}

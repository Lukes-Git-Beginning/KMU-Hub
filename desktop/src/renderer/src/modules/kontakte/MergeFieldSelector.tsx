/**
 * MergeFieldSelector — Side-by-side field comparison for duplicate merge.
 *
 * Shows a field from both contacts, lets user pick which value to keep.
 */
import { useTranslation } from 'react-i18next'

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface MergeFieldSelectorProps {
  label: string
  valueA: string
  valueB: string
  selected: 'a' | 'b'
  onSelect: (side: 'a' | 'b') => void
}

export function MergeFieldSelector({
  label,
  valueA,
  valueB,
  selected,
  onSelect,
}: MergeFieldSelectorProps) {
  const { t } = useTranslation()
  if (!valueA && !valueB) return null
  if (valueA === valueB) {
    return (
      <div className="grid grid-cols-[100px_1fr] gap-2 items-center">
        <span className="text-[11px] text-muted-foreground text-right">{label}</span>
        <span className="text-sm text-foreground">{valueA || '—'}</span>
      </div>
    )
  }

  return (
    <div className="grid grid-cols-[100px_1fr_1fr] gap-2 items-center">
      <span className="text-[11px] text-muted-foreground text-right">{label}</span>
      <button
        onClick={() => onSelect('a')}
        className={`rounded-md border px-2.5 py-1.5 text-left text-sm transition-colors ${
          selected === 'a'
            ? 'border-primary bg-primary/5 text-primary font-medium'
            : 'border-border text-foreground/70 hover:bg-accent'
        }`}
      >
        {valueA || <span className="text-muted-foreground italic">{t('kontakte.merge.empty')}</span>}
      </button>
      <button
        onClick={() => onSelect('b')}
        className={`rounded-md border px-2.5 py-1.5 text-left text-sm transition-colors ${
          selected === 'b'
            ? 'border-primary bg-primary/5 text-primary font-medium'
            : 'border-border text-foreground/70 hover:bg-accent'
        }`}
      >
        {valueB || <span className="text-muted-foreground italic">{t('kontakte.merge.empty')}</span>}
      </button>
    </div>
  )
}

import { useTranslation } from 'react-i18next'
import { Check, Loader2, Save } from 'lucide-react'

interface SaveReportBarProps {
  name: string
  onNameChange: (name: string) => void
  onSave: () => void
  isPending: boolean
  isEditing: boolean
  canSave: boolean
}

export function SaveReportBar({ name, onNameChange, onSave, isPending, isEditing, canSave }: SaveReportBarProps) {
  const { t } = useTranslation()
  return (
    <div className="flex items-center gap-2 rounded-xl border border-border bg-card p-2.5">
      <input
        value={name}
        onChange={(e) => onNameChange(e.target.value)}
        placeholder={t('berichte.builder.save.namePlaceholder', { defaultValue: 'Berichtsname…' })}
        className="min-w-0 flex-1 rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
      />
      <button
        type="button"
        onClick={onSave}
        disabled={!canSave || !name.trim() || isPending}
        className="flex shrink-0 items-center gap-2 rounded-lg bg-primary px-4 py-1.5 text-sm text-primary-foreground transition-colors hover:bg-button-primary-hover disabled:opacity-50"
      >
        {isPending ? (
          <Loader2 className="h-4 w-4 animate-spin" />
        ) : isEditing ? (
          <Check className="h-4 w-4" />
        ) : (
          <Save className="h-4 w-4" />
        )}
        {isEditing
          ? t('berichte.builder.save.update', { defaultValue: 'Aktualisieren' })
          : t('berichte.builder.save.save', { defaultValue: 'Speichern' })}
      </button>
    </div>
  )
}

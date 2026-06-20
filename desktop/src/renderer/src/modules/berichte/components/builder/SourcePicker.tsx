import { useTranslation } from 'react-i18next'
import { REPORT_SOURCES } from '../../report-sources/registry'

interface SourcePickerProps {
  value: string | null
  onChange: (sourceId: string) => void
}

export function SourcePicker({ value, onChange }: SourcePickerProps) {
  const { t } = useTranslation()
  return (
    <div>
      <label className="mb-2 block text-xs font-medium text-muted-foreground">
        {t('berichte.builder.source.label', { defaultValue: 'Datenquelle' })}
      </label>
      <div className="grid grid-cols-1 gap-1.5">
        {REPORT_SOURCES.map((s) => {
          const Icon = s.icon
          const active = value === s.id
          return (
            <button
              key={s.id}
              type="button"
              onClick={() => onChange(s.id)}
              className={`flex items-center gap-3 rounded-lg border p-2.5 text-left transition-colors ${
                active
                  ? 'border-primary bg-primary-light'
                  : 'border-border hover:border-primary/30 hover:bg-secondary'
              }`}
            >
              <div
                className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-md ${
                  active ? 'bg-primary text-primary-foreground' : 'bg-secondary text-muted-foreground'
                }`}
              >
                <Icon className="h-4 w-4" />
              </div>
              <div className="min-w-0">
                <p className={`truncate text-sm font-medium ${active ? 'text-primary' : 'text-foreground'}`}>
                  {t(s.labelKey, { defaultValue: s.label })}
                </p>
                <p className="truncate text-xs text-muted-foreground">{s.description}</p>
              </div>
            </button>
          )
        })}
      </div>
    </div>
  )
}

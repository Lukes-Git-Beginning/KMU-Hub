import { useTranslation } from 'react-i18next'
import { CalendarRange } from 'lucide-react'
import type { ReportDateRange } from '@/api/berichte-types'
import type { ReportSource } from '../../report-sources/types'
import { DATE_PRESETS } from './filter-utils'

interface DateRangePickerProps {
  source: ReportSource
  value: ReportDateRange | undefined
  onChange: (range: ReportDateRange | undefined) => void
}

export function DateRangePicker({ source, value, onChange }: DateRangePickerProps) {
  const { t } = useTranslation()
  const dateFields = source.fields.filter((f) => f.dataType === 'date')
  if (dateFields.length === 0) return null
  const field = value?.field ?? dateFields[0].key
  const active = value?.preset

  return (
    <div>
      <div className="mb-1.5 flex items-center gap-1.5">
        <CalendarRange className="h-3.5 w-3.5 text-muted-foreground" />
        <label className="text-xs font-medium text-muted-foreground">
          {t('berichte.builder.dateRange.label', { defaultValue: 'Zeitraum' })}
        </label>
      </div>
      <div className="flex flex-wrap gap-1.5">
        <button
          type="button"
          onClick={() => onChange(undefined)}
          className={`rounded-md border px-2.5 py-1 text-xs transition-colors ${
            !active
              ? 'border-primary bg-primary-light text-primary'
              : 'border-border text-muted-foreground hover:bg-secondary'
          }`}
        >
          {t('berichte.builder.dateRange.all', { defaultValue: 'Gesamt' })}
        </button>
        {DATE_PRESETS.map((p) => (
          <button
            key={p.value}
            type="button"
            onClick={() => onChange({ field, preset: p.value })}
            className={`rounded-md border px-2.5 py-1 text-xs transition-colors ${
              active === p.value
                ? 'border-primary bg-primary-light text-primary'
                : 'border-border text-muted-foreground hover:bg-secondary'
            }`}
          >
            {t(`berichte.builder.dateRange.preset.${p.value}`, { defaultValue: p.label })}
          </button>
        ))}
      </div>
    </div>
  )
}

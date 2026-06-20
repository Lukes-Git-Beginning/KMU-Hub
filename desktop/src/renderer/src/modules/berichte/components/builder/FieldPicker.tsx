import { useTranslation } from 'react-i18next'
import { Calendar, Check, Hash, List, ToggleLeft, Type } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import type { ReportMeasure } from '@/api/berichte-types'
import type { FieldDataType, FieldDefinition, ReportSource } from '../../report-sources/types'
import { dimensionsOf, measuresOf } from '../../report-sources/types'
import { MAX_DIMENSIONS } from './builder-utils'

const TYPE_ICON: Record<FieldDataType, LucideIcon> = {
  number: Hash,
  string: Type,
  date: Calendar,
  enum: List,
  boolean: ToggleLeft,
}

interface FieldPickerProps {
  source: ReportSource
  dimensions: string[]
  measures: ReportMeasure[]
  onToggleDimension: (key: string) => void
  onToggleMeasure: (key: string) => void
}

function FieldRow({
  field,
  checked,
  disabled,
  onToggle,
}: {
  field: FieldDefinition
  checked: boolean
  disabled?: boolean
  onToggle: () => void
}) {
  const { t } = useTranslation()
  const Icon = TYPE_ICON[field.dataType]
  return (
    <button
      type="button"
      onClick={onToggle}
      disabled={disabled}
      className={`flex w-full items-center gap-2.5 rounded-md px-2 py-1.5 text-left text-sm transition-colors ${
        checked ? 'bg-primary-light text-foreground' : 'text-foreground hover:bg-secondary'
      } ${disabled ? 'cursor-not-allowed opacity-40' : ''}`}
    >
      <span
        className={`flex h-4 w-4 shrink-0 items-center justify-center rounded border ${
          checked ? 'border-primary bg-primary text-primary-foreground' : 'border-border'
        }`}
      >
        {checked && <Check className="h-3 w-3" />}
      </span>
      <Icon className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
      <span className="truncate">{t(field.labelKey, { defaultValue: field.label })}</span>
    </button>
  )
}

export function FieldPicker({
  source,
  dimensions,
  measures,
  onToggleDimension,
  onToggleMeasure,
}: FieldPickerProps) {
  const { t } = useTranslation()
  const dims = dimensionsOf(source)
  const meas = measuresOf(source)
  const dimFull = dimensions.length >= MAX_DIMENSIONS
  const measureKeys = new Set(measures.map((m) => m.field))

  return (
    <div className="space-y-4">
      {/* Dimensions */}
      <div>
        <div className="mb-1.5 flex items-center justify-between">
          <label className="text-xs font-medium text-muted-foreground">
            {t('berichte.builder.fields.dimensions', { defaultValue: 'Gruppieren nach' })}
          </label>
          <span className="text-[10px] text-muted-foreground/70">
            {t('berichte.builder.fields.dimensionLimit', {
              defaultValue: '{count}/{max}',
              count: dimensions.length,
              max: MAX_DIMENSIONS,
            })}
          </span>
        </div>
        <div className="space-y-0.5">
          {dims.map((f) => {
            const checked = dimensions.includes(f.key)
            return (
              <FieldRow
                key={f.key}
                field={f}
                checked={checked}
                disabled={!checked && dimFull}
                onToggle={() => onToggleDimension(f.key)}
              />
            )
          })}
        </div>
      </div>

      {/* Measures */}
      <div>
        <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
          {t('berichte.builder.fields.measures', { defaultValue: 'Kennzahlen' })}
        </label>
        <div className="space-y-0.5">
          {meas.map((f) => (
            <FieldRow
              key={f.key}
              field={f}
              checked={measureKeys.has(f.key)}
              onToggle={() => onToggleMeasure(f.key)}
            />
          ))}
        </div>
      </div>
    </div>
  )
}

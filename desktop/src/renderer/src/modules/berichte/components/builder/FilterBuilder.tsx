import { useTranslation } from 'react-i18next'
import { Filter, Plus, X } from 'lucide-react'
import type { FilterOperator, ReportFilter } from '@/api/berichte-types'
import type { FieldDefinition, ReportSource } from '../../report-sources/types'
import { fieldByKey } from '../../report-sources/types'
import { NO_VALUE_OPERATORS, OPERATOR_LABEL, OPERATORS_BY_TYPE } from './filter-utils'

interface FilterBuilderProps {
  source: ReportSource
  filters: ReportFilter[]
  onChange: (filters: ReportFilter[]) => void
}

const SELECT_CLASS =
  'rounded-md border border-border bg-card px-2 py-1 text-xs text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring'

function defaultOperator(field: FieldDefinition): FilterOperator {
  return OPERATORS_BY_TYPE[field.dataType][0]
}

function ValueInput({
  field,
  filter,
  onPatch,
}: {
  field: FieldDefinition
  filter: ReportFilter
  onPatch: (patch: Partial<ReportFilter>) => void
}) {
  if (NO_VALUE_OPERATORS.includes(filter.operator)) return null

  if (field.dataType === 'enum') {
    return (
      <select
        value={String(filter.value ?? '')}
        onChange={(e) => onPatch({ value: e.target.value })}
        className={`${SELECT_CLASS} min-w-0 flex-1`}
      >
        <option value="">—</option>
        {field.enumValues?.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
    )
  }

  if (field.dataType === 'boolean') {
    return (
      <select
        value={String(filter.value ?? 'true')}
        onChange={(e) => onPatch({ value: e.target.value })}
        className={`${SELECT_CLASS} min-w-0 flex-1`}
      >
        <option value="true">Ja</option>
        <option value="false">Nein</option>
      </select>
    )
  }

  if (field.dataType === 'number') {
    if (filter.operator === 'between') {
      return (
        <div className="flex min-w-0 flex-1 items-center gap-1">
          <input
            type="number"
            value={String(filter.value ?? '')}
            onChange={(e) => onPatch({ value: e.target.value })}
            className={`${SELECT_CLASS} w-full`}
          />
          <span className="text-[10px] text-muted-foreground">–</span>
          <input
            type="number"
            value={String(filter.value2 ?? '')}
            onChange={(e) => onPatch({ value2: e.target.value })}
            className={`${SELECT_CLASS} w-full`}
          />
        </div>
      )
    }
    return (
      <input
        type="number"
        value={String(filter.value ?? '')}
        onChange={(e) => onPatch({ value: e.target.value })}
        className={`${SELECT_CLASS} min-w-0 flex-1`}
      />
    )
  }

  // string
  return (
    <input
      type="text"
      value={String(filter.value ?? '')}
      onChange={(e) => onPatch({ value: e.target.value })}
      className={`${SELECT_CLASS} min-w-0 flex-1`}
    />
  )
}

export function FilterBuilder({ source, filters, onChange }: FilterBuilderProps) {
  const { t } = useTranslation()
  const filterable = source.fields.filter((f) => f.dataType !== 'date')

  function addFilter() {
    const f0 = filterable[0]
    if (!f0) return
    onChange([...filters, { field: f0.key, operator: defaultOperator(f0), value: '' }])
  }

  function patchFilter(index: number, patch: Partial<ReportFilter>) {
    onChange(filters.map((f, i) => (i === index ? { ...f, ...patch } : f)))
  }

  function changeField(index: number, key: string) {
    const field = fieldByKey(source, key)
    if (!field) return
    patchFilter(index, { field: key, operator: defaultOperator(field), value: '', value2: undefined })
  }

  function removeFilter(index: number) {
    onChange(filters.filter((_, i) => i !== index))
  }

  return (
    <div>
      <div className="mb-1.5 flex items-center justify-between">
        <div className="flex items-center gap-1.5">
          <Filter className="h-3.5 w-3.5 text-muted-foreground" />
          <label className="text-xs font-medium text-muted-foreground">
            {t('berichte.builder.filter.label', { defaultValue: 'Filter' })}
          </label>
        </div>
        <button
          type="button"
          onClick={addFilter}
          className="flex items-center gap-1 rounded-md px-1.5 py-0.5 text-xs text-primary transition-colors hover:bg-primary-light"
        >
          <Plus className="h-3 w-3" />
          {t('berichte.builder.filter.add', { defaultValue: 'Hinzufügen' })}
        </button>
      </div>

      {filters.length === 0 ? (
        <p className="text-[11px] text-muted-foreground/70">
          {t('berichte.builder.filter.empty', { defaultValue: 'Keine Filter — alle Datensätze.' })}
        </p>
      ) : (
        <div className="space-y-2">
          {filters.map((filter, i) => {
            const field = fieldByKey(source, filter.field)
            const operators = field ? OPERATORS_BY_TYPE[field.dataType] : []
            return (
              <div key={i} className="rounded-lg border border-border-muted bg-secondary/40 p-2">
                <div className="flex items-center gap-1.5">
                  <select
                    value={filter.field}
                    onChange={(e) => changeField(i, e.target.value)}
                    className={`${SELECT_CLASS} min-w-0 flex-1`}
                  >
                    {filterable.map((f) => (
                      <option key={f.key} value={f.key}>
                        {f.label}
                      </option>
                    ))}
                  </select>
                  <button
                    type="button"
                    onClick={() => removeFilter(i)}
                    className="shrink-0 rounded p-1 text-muted-foreground transition-colors hover:bg-secondary hover:text-destructive"
                    aria-label={t('berichte.builder.filter.remove', { defaultValue: 'Filter entfernen' })}
                  >
                    <X className="h-3.5 w-3.5" />
                  </button>
                </div>
                <div className="mt-1.5 flex items-center gap-1.5">
                  <select
                    value={filter.operator}
                    onChange={(e) => patchFilter(i, { operator: e.target.value as FilterOperator, value: '', value2: undefined })}
                    className={`${SELECT_CLASS} shrink-0`}
                  >
                    {operators.map((op) => (
                      <option key={op} value={op}>
                        {OPERATOR_LABEL[op]}
                      </option>
                    ))}
                  </select>
                  {field && <ValueInput field={field} filter={filter} onPatch={(p) => patchFilter(i, p)} />}
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

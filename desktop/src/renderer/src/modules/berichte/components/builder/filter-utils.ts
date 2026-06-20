import type { DateRangePreset, FilterOperator } from '@/api/berichte-types'
import type { FieldDataType } from '../../report-sources/types'

/** Valid operators per field data type (date is handled via DateRangePicker). */
export const OPERATORS_BY_TYPE: Record<FieldDataType, FilterOperator[]> = {
  number: ['eq', 'neq', 'gt', 'lt', 'between', 'isEmpty'],
  string: ['contains', 'startsWith', 'eq', 'isEmpty'],
  date: ['after', 'before', 'inLastDays'],
  enum: ['eq', 'neq'],
  boolean: ['eq'],
}

export const OPERATOR_LABEL: Record<FilterOperator, string> = {
  eq: 'ist',
  neq: 'ist nicht',
  gt: 'größer als',
  lt: 'kleiner als',
  between: 'zwischen',
  contains: 'enthält',
  startsWith: 'beginnt mit',
  isEmpty: 'ist leer',
  in: 'ist eines von',
  notIn: 'ist keines von',
  before: 'vor',
  after: 'nach',
  inLastDays: 'in den letzten Tagen',
}

/** Operators that need no value input. */
export const NO_VALUE_OPERATORS: FilterOperator[] = ['isEmpty']

export const DATE_PRESETS: { value: DateRangePreset; label: string }[] = [
  { value: 'last7', label: 'Letzte 7 Tage' },
  { value: 'last30', label: 'Letzte 30 Tage' },
  { value: 'last90', label: 'Letzte 90 Tage' },
  { value: 'thisMonth', label: 'Dieser Monat' },
  { value: 'lastMonth', label: 'Letzter Monat' },
  { value: 'thisQuarter', label: 'Dieses Quartal' },
  { value: 'thisYear', label: 'Dieses Jahr' },
]

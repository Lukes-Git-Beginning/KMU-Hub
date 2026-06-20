import { LifeBuoy } from 'lucide-react'
import type { ReportSource } from './types'
import { generateRows, intBetween, isoDaysAgo, pick } from './sample-utils'

const STATUS = [
  { value: 'open', label: 'Offen' },
  { value: 'pending', label: 'Wartend' },
  { value: 'solved', label: 'Gelöst' },
  { value: 'closed', label: 'Geschlossen' },
]
const PRIORITY = [
  { value: 'low', label: 'Niedrig' },
  { value: 'normal', label: 'Normal' },
  { value: 'high', label: 'Hoch' },
  { value: 'urgent', label: 'Dringend' },
]
const CATEGORY = [
  { value: 'technik', label: 'Technik' },
  { value: 'abrechnung', label: 'Abrechnung' },
  { value: 'allgemein', label: 'Allgemein' },
  { value: 'feature', label: 'Feature-Wunsch' },
]
const AGENTS = ['Lena Hofer', 'Marco Reis', 'Tom Keller', 'Nina Wolf']

export const helpdeskSource: ReportSource = {
  id: 'helpdesk',
  module: 'helpdesk',
  labelKey: 'berichte.sources.helpdesk.label',
  label: 'Tickets',
  icon: LifeBuoy,
  description: 'Support-Auslastung und Reaktionszeiten',
  defaultViz: 'line',
  fields: [
    { key: 'created_at', labelKey: 'berichte.fields.helpdesk.created_at', label: 'Erstellt am', dataType: 'date', role: 'dimension' },
    { key: 'status', labelKey: 'berichte.fields.helpdesk.status', label: 'Status', dataType: 'enum', role: 'dimension', enumValues: STATUS },
    { key: 'priority', labelKey: 'berichte.fields.helpdesk.priority', label: 'Priorität', dataType: 'enum', role: 'dimension', enumValues: PRIORITY },
    { key: 'category', labelKey: 'berichte.fields.helpdesk.category', label: 'Kategorie', dataType: 'enum', role: 'dimension', enumValues: CATEGORY },
    { key: 'agent', labelKey: 'berichte.fields.helpdesk.agent', label: 'Bearbeiter', dataType: 'string', role: 'dimension' },
    { key: 'first_response_mins', labelKey: 'berichte.fields.helpdesk.first_response_mins', label: 'Reaktionszeit (Min)', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'avg' },
    { key: 'resolution_mins', labelKey: 'berichte.fields.helpdesk.resolution_mins', label: 'Lösungszeit (Min)', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'avg' },
  ],
  sampleRows: (count = 150) =>
    generateRows(404, count, (r) => ({
      created_at: isoDaysAgo(intBetween(r(), 0, 180)),
      status: pick(STATUS, r()).value,
      priority: pick(PRIORITY, r()).value,
      category: pick(CATEGORY, r()).value,
      agent: pick(AGENTS, r()),
      first_response_mins: intBetween(r(), 5, 480),
      resolution_mins: intBetween(r(), 60, 2880),
    })),
}

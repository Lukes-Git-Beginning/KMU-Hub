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
const QUEUE = [
  { value: 'first_level', label: '1st-Level' },
  { value: 'second_level', label: '2nd-Level' },
  { value: 'vertrieb', label: 'Vertrieb' },
  { value: 'technik', label: 'Technik' },
]
const CHANNEL = [
  { value: 'email', label: 'E-Mail' },
  { value: 'phone', label: 'Telefon' },
  { value: 'chat', label: 'Chat' },
  { value: 'portal', label: 'Portal' },
]
const SLA = [
  { value: 'on_track', label: 'Im Plan' },
  { value: 'at_risk', label: 'Gefährdet' },
  { value: 'breached', label: 'Verletzt' },
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
    { key: 'queue', labelKey: 'berichte.fields.helpdesk.queue', label: 'Warteschlange', dataType: 'enum', role: 'dimension', enumValues: QUEUE },
    { key: 'channel', labelKey: 'berichte.fields.helpdesk.channel', label: 'Kanal', dataType: 'enum', role: 'dimension', enumValues: CHANNEL },
    { key: 'sla_status', labelKey: 'berichte.fields.helpdesk.sla_status', label: 'SLA-Status', dataType: 'enum', role: 'dimension', enumValues: SLA },
    { key: 'agent', labelKey: 'berichte.fields.helpdesk.agent', label: 'Bearbeiter', dataType: 'string', role: 'dimension' },
    { key: 'first_response_mins', labelKey: 'berichte.fields.helpdesk.first_response_mins', label: 'Reaktionszeit (Min)', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'avg' },
    { key: 'resolution_mins', labelKey: 'berichte.fields.helpdesk.resolution_mins', label: 'Lösungszeit (Min)', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'avg' },
    { key: 'csat', labelKey: 'berichte.fields.helpdesk.csat', label: 'Zufriedenheit (1–5)', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'avg' },
    { key: 'reopen_count', labelKey: 'berichte.fields.helpdesk.reopen_count', label: 'Wiedereröffnungen', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'sum' },
  ],
  sampleRows: (count = 150) =>
    generateRows(404, count, (r) => ({
      created_at: isoDaysAgo(intBetween(r(), 0, 180)),
      status: pick(STATUS, r()).value,
      priority: pick(PRIORITY, r()).value,
      category: pick(CATEGORY, r()).value,
      queue: pick(QUEUE, r()).value,
      channel: pick(CHANNEL, r()).value,
      sla_status: pick(SLA, r()).value,
      agent: pick(AGENTS, r()),
      first_response_mins: intBetween(r(), 5, 480),
      resolution_mins: intBetween(r(), 60, 2880),
      csat: intBetween(r(), 1, 5),
      reopen_count: intBetween(r(), 0, 3),
    })),
}

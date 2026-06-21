import { MessagesSquare } from 'lucide-react'
import type { ReportSource } from './types'
import { generateRows, intBetween, isoDaysAgo, pick } from './sample-utils'

const CHANNEL = [
  { value: 'email', label: 'E-Mail' },
  { value: 'chat', label: 'Chat' },
  { value: 'notification', label: 'Benachrichtigung' },
]
const DIRECTION = [
  { value: 'inbound', label: 'Eingehend' },
  { value: 'outbound', label: 'Ausgehend' },
]
const STATUS = [
  { value: 'unread', label: 'Ungelesen' },
  { value: 'read', label: 'Gelesen' },
  { value: 'archived', label: 'Archiviert' },
]
const PRIORITY = [
  { value: 'normal', label: 'Normal' },
  { value: 'important', label: 'Wichtig' },
  { value: 'urgent', label: 'Dringend' },
]
const CATEGORY = [
  { value: 'request', label: 'Anfrage' },
  { value: 'complaint', label: 'Beschwerde' },
  { value: 'info', label: 'Information' },
  { value: 'order', label: 'Bestellung' },
]
const STARRED = [
  { value: 'yes', label: 'Markiert' },
  { value: 'no', label: 'Nicht markiert' },
]
const CUSTOMERS = [
  'Müller Maschinenbau GmbH',
  'Fintech Solutions AG',
  'Atelier Nord Design',
  'Pharma Vitalis GmbH',
  'Bau & Plan AG',
  'Greentech Energie GmbH',
]
const ASSIGNEES = ['Lena Hofer', 'Marco Reis', 'Julia Brandt', 'Tom Keller']

export const kommunikationSource: ReportSource = {
  id: 'kommunikation',
  module: 'kommunikation',
  labelKey: 'berichte.sources.kommunikation.label',
  label: 'Posteingang',
  icon: MessagesSquare,
  description: 'Nachrichten-Aufkommen und Bearbeitung',
  defaultViz: 'area',
  fields: [
    { key: 'received_at', labelKey: 'berichte.fields.kommunikation.received_at', label: 'Eingang am', dataType: 'date', role: 'dimension' },
    { key: 'channel', labelKey: 'berichte.fields.kommunikation.channel', label: 'Kanal', dataType: 'enum', role: 'dimension', enumValues: CHANNEL },
    { key: 'direction', labelKey: 'berichte.fields.kommunikation.direction', label: 'Richtung', dataType: 'enum', role: 'dimension', enumValues: DIRECTION },
    { key: 'status', labelKey: 'berichte.fields.kommunikation.status', label: 'Status', dataType: 'enum', role: 'dimension', enumValues: STATUS },
    { key: 'priority', labelKey: 'berichte.fields.kommunikation.priority', label: 'Priorität', dataType: 'enum', role: 'dimension', enumValues: PRIORITY },
    { key: 'category', labelKey: 'berichte.fields.kommunikation.category', label: 'Kategorie', dataType: 'enum', role: 'dimension', enumValues: CATEGORY },
    { key: 'customer', labelKey: 'berichte.fields.kommunikation.customer', label: 'Kontakt', dataType: 'string', role: 'dimension' },
    { key: 'assigned_to', labelKey: 'berichte.fields.kommunikation.assigned_to', label: 'Zugewiesen an', dataType: 'string', role: 'dimension' },
    { key: 'is_starred', labelKey: 'berichte.fields.kommunikation.is_starred', label: 'Markiert', dataType: 'enum', role: 'dimension', enumValues: STARRED },
    { key: 'response_mins', labelKey: 'berichte.fields.kommunikation.response_mins', label: 'Antwortzeit (Min)', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'avg' },
    { key: 'handle_mins', labelKey: 'berichte.fields.kommunikation.handle_mins', label: 'Bearbeitungszeit (Min)', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'avg' },
    { key: 'message_length', labelKey: 'berichte.fields.kommunikation.message_length', label: 'Nachrichtenlänge (Zeichen)', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'avg' },
  ],
  sampleRows: (count = 160) =>
    generateRows(505, count, (r) => ({
      received_at: isoDaysAgo(intBetween(r(), 0, 120)),
      channel: pick(CHANNEL, r()).value,
      direction: pick(DIRECTION, r()).value,
      status: pick(STATUS, r()).value,
      priority: pick(PRIORITY, r()).value,
      category: pick(CATEGORY, r()).value,
      customer: pick(CUSTOMERS, r()),
      assigned_to: pick(ASSIGNEES, r()),
      is_starred: pick(STARRED, r()).value,
      response_mins: intBetween(r(), 2, 600),
      handle_mins: intBetween(r(), 1, 90),
      message_length: intBetween(r(), 80, 2400),
    })),
}

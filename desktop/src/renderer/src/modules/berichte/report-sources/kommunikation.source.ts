import { MessagesSquare } from 'lucide-react'
import type { ReportSource } from './types'
import { generateRows, intBetween, isoDaysAgo, pick } from './sample-utils'

const CHANNEL = [
  { value: 'email', label: 'E-Mail' },
  { value: 'chat', label: 'Chat' },
  { value: 'notification', label: 'Benachrichtigung' },
]
const STATUS = [
  { value: 'unread', label: 'Ungelesen' },
  { value: 'read', label: 'Gelesen' },
  { value: 'archived', label: 'Archiviert' },
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
    { key: 'status', labelKey: 'berichte.fields.kommunikation.status', label: 'Status', dataType: 'enum', role: 'dimension', enumValues: STATUS },
    { key: 'assigned_to', labelKey: 'berichte.fields.kommunikation.assigned_to', label: 'Zugewiesen an', dataType: 'string', role: 'dimension' },
    { key: 'response_mins', labelKey: 'berichte.fields.kommunikation.response_mins', label: 'Antwortzeit (Min)', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'avg' },
  ],
  sampleRows: (count = 160) =>
    generateRows(505, count, (r) => ({
      received_at: isoDaysAgo(intBetween(r(), 0, 120)),
      channel: pick(CHANNEL, r()).value,
      status: pick(STATUS, r()).value,
      assigned_to: pick(ASSIGNEES, r()),
      response_mins: intBetween(r(), 2, 600),
    })),
}

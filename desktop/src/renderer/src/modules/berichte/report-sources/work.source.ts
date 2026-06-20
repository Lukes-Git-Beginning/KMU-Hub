import { KanbanSquare } from 'lucide-react'
import type { ReportSource } from './types'
import { generateRows, intBetween, isoDaysAgo, pick } from './sample-utils'

const STATUS = [
  { value: 'todo', label: 'Zu erledigen' },
  { value: 'in_progress', label: 'In Arbeit' },
  { value: 'review', label: 'Review' },
  { value: 'done', label: 'Erledigt' },
]
const PRIORITY = [
  { value: 'low', label: 'Niedrig' },
  { value: 'normal', label: 'Normal' },
  { value: 'high', label: 'Hoch' },
  { value: 'urgent', label: 'Dringend' },
]
const PROJECTS = ['Onboarding Portal', 'Website Relaunch', 'ERP-Migration', 'Mobile App', 'Kampagne Q3']
const ASSIGNEES = ['Stefan Vogel', 'Lena Hofer', 'Marco Reis', 'Julia Brandt', 'Tom Keller', 'Nina Wolf']

export const workSource: ReportSource = {
  id: 'work',
  module: 'work',
  labelKey: 'berichte.sources.work.label',
  label: 'Projekte & Aufgaben',
  icon: KanbanSquare,
  description: 'Aufgaben, Status und Auslastung',
  defaultViz: 'bar',
  fields: [
    { key: 'due_date', labelKey: 'berichte.fields.work.due_date', label: 'Fälligkeit', dataType: 'date', role: 'dimension' },
    { key: 'status', labelKey: 'berichte.fields.work.status', label: 'Status', dataType: 'enum', role: 'dimension', enumValues: STATUS },
    { key: 'priority', labelKey: 'berichte.fields.work.priority', label: 'Priorität', dataType: 'enum', role: 'dimension', enumValues: PRIORITY },
    { key: 'project', labelKey: 'berichte.fields.work.project', label: 'Projekt', dataType: 'string', role: 'dimension' },
    { key: 'assignee', labelKey: 'berichte.fields.work.assignee', label: 'Zuständig', dataType: 'string', role: 'dimension' },
    { key: 'estimated_hours', labelKey: 'berichte.fields.work.estimated_hours', label: 'Geschätzte Stunden', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'sum' },
    { key: 'progress', labelKey: 'berichte.fields.work.progress', label: 'Fortschritt', dataType: 'number', role: 'measure', format: 'percent', defaultAgg: 'avg' },
  ],
  sampleRows: (count = 130) =>
    generateRows(303, count, (r) => {
      const status = pick(STATUS, r())
      const progress =
        status.value === 'done' ? 100 : status.value === 'todo' ? 0 : intBetween(r(), 10, 90)
      return {
        due_date: isoDaysAgo(intBetween(r(), -30, 180)),
        status: status.value,
        priority: pick(PRIORITY, r()).value,
        project: pick(PROJECTS, r()),
        assignee: pick(ASSIGNEES, r()),
        estimated_hours: intBetween(r(), 1, 40),
        progress,
      }
    }),
}

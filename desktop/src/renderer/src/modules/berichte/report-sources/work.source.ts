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
const CATEGORY = [
  { value: 'bug', label: 'Fehler' },
  { value: 'feature', label: 'Feature' },
  { value: 'chore', label: 'Wartung' },
  { value: 'doc', label: 'Dokumentation' },
]
const BILLABLE = [
  { value: 'yes', label: 'Abrechenbar' },
  { value: 'no', label: 'Nicht abrechenbar' },
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
    { key: 'created_date', labelKey: 'berichte.fields.work.created_date', label: 'Erstellt am', dataType: 'date', role: 'dimension' },
    { key: 'due_date', labelKey: 'berichte.fields.work.due_date', label: 'Fälligkeit', dataType: 'date', role: 'dimension' },
    { key: 'status', labelKey: 'berichte.fields.work.status', label: 'Status', dataType: 'enum', role: 'dimension', enumValues: STATUS },
    { key: 'priority', labelKey: 'berichte.fields.work.priority', label: 'Priorität', dataType: 'enum', role: 'dimension', enumValues: PRIORITY },
    { key: 'category', labelKey: 'berichte.fields.work.category', label: 'Aufgabentyp', dataType: 'enum', role: 'dimension', enumValues: CATEGORY },
    { key: 'project', labelKey: 'berichte.fields.work.project', label: 'Projekt', dataType: 'string', role: 'dimension' },
    { key: 'assignee', labelKey: 'berichte.fields.work.assignee', label: 'Zuständig', dataType: 'string', role: 'dimension' },
    { key: 'billable', labelKey: 'berichte.fields.work.billable', label: 'Abrechenbarkeit', dataType: 'enum', role: 'dimension', enumValues: BILLABLE },
    { key: 'estimated_hours', labelKey: 'berichte.fields.work.estimated_hours', label: 'Geschätzte Stunden', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'sum' },
    { key: 'logged_hours', labelKey: 'berichte.fields.work.logged_hours', label: 'Erfasste Stunden', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'sum' },
    { key: 'progress', labelKey: 'berichte.fields.work.progress', label: 'Fortschritt', dataType: 'number', role: 'measure', format: 'percent', defaultAgg: 'avg' },
    { key: 'subtask_count', labelKey: 'berichte.fields.work.subtask_count', label: 'Unteraufgaben', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'sum' },
  ],
  sampleRows: (count = 130) =>
    generateRows(303, count, (r) => {
      const status = pick(STATUS, r())
      const progress =
        status.value === 'done' ? 100 : status.value === 'todo' ? 0 : intBetween(r(), 10, 90)
      const estimated = intBetween(r(), 1, 40)
      return {
        created_date: isoDaysAgo(intBetween(r(), 0, 240)),
        due_date: isoDaysAgo(intBetween(r(), -30, 180)),
        status: status.value,
        priority: pick(PRIORITY, r()).value,
        category: pick(CATEGORY, r()).value,
        project: pick(PROJECTS, r()),
        assignee: pick(ASSIGNEES, r()),
        billable: pick(BILLABLE, r()).value,
        estimated_hours: estimated,
        logged_hours: Math.round(estimated * (progress / 100) * 10) / 10,
        progress,
        subtask_count: intBetween(r(), 0, 12),
      }
    }),
}

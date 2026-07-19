/**
 * Gantt chart utility functions -- pure date math, positioning, and
 * critical path logic. No React dependencies.
 *
 * All date formatting uses German locale. Position helpers convert
 * between dates and pixel coordinates on the timeline.
 */
import {
  startOfDay,
  endOfDay,
  differenceInCalendarDays,
  addDays,
  subDays,
  addMonths,
  startOfMonth,
  endOfMonth,
  eachDayOfInterval,
  eachWeekOfInterval,
  eachMonthOfInterval,
  format,
  getISOWeek,
  isWeekend,
  isToday,
  isAfter,
  min as minDate,
  max as maxDate,
  parseISO,
} from 'date-fns'
import { de } from 'date-fns/locale'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface GanttTask {
  id: string
  title: string
  project_key?: string
  task_number: number
  start: Date | null
  end: Date | null
  priority: string
  status_name?: string
  status_color?: string
  assignee_name?: string
  assignee_id?: string | null
  parent_task_id?: string | null
  depth: number
  completed_at?: string | null
  is_on_critical_path: boolean
}

export interface GanttDependency {
  id: string
  source_task_id: string
  target_task_id: string
  dependency_type: string
}

export type ZoomLevel = 'day' | 'week' | 'month'

export interface TimelineConfig {
  startDate: Date
  endDate: Date
  zoom: ZoomLevel
  columnWidth: number
}

export interface TimelineColumn {
  date: Date
  label: string
  isWeekend: boolean
  isToday: boolean
}

export interface TaskBar {
  x: number
  width: number
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

export const COLUMN_WIDTHS: Record<ZoomLevel, number> = {
  day: 40,
  week: 120,
  month: 200,
}

export const ROW_HEIGHT = 36
export const BAR_HEIGHT = 20
export const HEADER_HEIGHT = 48
export const TASK_INFO_WIDTH = 280

// Pixels per day at each zoom level
const PIXELS_PER_DAY: Record<ZoomLevel, number> = {
  day: 40,
  week: 120 / 7,
  month: 200 / 30,
}

// ---------------------------------------------------------------------------
// Timeline configuration
// ---------------------------------------------------------------------------

/**
 * Build timeline config from tasks. Scans for earliest start / latest end,
 * then adds 2 weeks of padding on each side. Falls back to +/- 1 month
 * from today if no tasks have dates.
 */
export function buildTimelineConfig(
  tasks: GanttTask[],
  zoom: ZoomLevel
): TimelineConfig {
  const dates: Date[] = []
  for (const t of tasks) {
    if (t.start) dates.push(t.start)
    if (t.end) dates.push(t.end)
  }

  let start: Date
  let end: Date

  if (dates.length > 0) {
    start = subDays(startOfDay(minDate(dates)), 14)
    end = addDays(endOfDay(maxDate(dates)), 14)
  } else {
    const now = new Date()
    start = addMonths(startOfMonth(now), -1)
    end = endOfMonth(addMonths(now, 1))
  }

  return {
    startDate: startOfDay(start),
    endDate: endOfDay(end),
    zoom,
    columnWidth: COLUMN_WIDTHS[zoom],
  }
}

// ---------------------------------------------------------------------------
// Date <-> pixel conversion
// ---------------------------------------------------------------------------

/** Convert a date to an X pixel position relative to the timeline start. */
export function dateToX(date: Date, config: TimelineConfig): number {
  const days = differenceInCalendarDays(date, config.startDate)
  return days * PIXELS_PER_DAY[config.zoom]
}

/** Convert an X pixel position back to a date. */
export function xToDate(x: number, config: TimelineConfig): Date {
  const ppd = PIXELS_PER_DAY[config.zoom]
  const days = Math.round(x / ppd)
  return addDays(config.startDate, days)
}

/** Total width of the timeline in pixels. */
export function getTimelineWidth(config: TimelineConfig): number {
  return dateToX(config.endDate, config)
}

// ---------------------------------------------------------------------------
// Timeline columns
// ---------------------------------------------------------------------------

/** Generate column data for the timeline header. */
export function generateTimelineColumns(
  config: TimelineConfig
): TimelineColumn[] {
  const { startDate, endDate, zoom } = config

  if (zoom === 'day') {
    return eachDayOfInterval({ start: startDate, end: endDate }).map((d) => ({
      date: d,
      label: format(d, 'EEEEEE d.', { locale: de }),
      isWeekend: isWeekend(d),
      isToday: isToday(d),
    }))
  }

  if (zoom === 'week') {
    return eachWeekOfInterval(
      { start: startDate, end: endDate },
      { weekStartsOn: 1 }
    ).map((d) => ({
      date: d,
      label: `KW ${getISOWeek(d)}`,
      isWeekend: false,
      isToday: false,
    }))
  }

  // month
  return eachMonthOfInterval({ start: startDate, end: endDate }).map((d) => ({
    date: d,
    label: format(d, 'MMM yyyy', { locale: de }),
    isWeekend: false,
    isToday: false,
  }))
}

/**
 * Generate top-level grouping headers for dual-row timeline header.
 * - day zoom  -> months spanning their day columns
 * - week zoom -> months spanning their week columns
 * - month zoom -> years spanning their month columns
 */
export function generateGroupHeaders(
  columns: TimelineColumn[],
  config: TimelineConfig
): Array<{ label: string; span: number }> {
  if (columns.length === 0) return []

  const groups: Array<{ label: string; span: number }> = []

  if (config.zoom === 'day' || config.zoom === 'week') {
    let currentKey = ''
    for (const col of columns) {
      const key = format(col.date, 'MMMM yyyy', { locale: de })
      if (key !== currentKey) {
        groups.push({ label: key, span: 1 })
        currentKey = key
      } else {
        groups[groups.length - 1].span++
      }
    }
  } else {
    let currentYear = ''
    for (const col of columns) {
      const key = format(col.date, 'yyyy')
      if (key !== currentYear) {
        groups.push({ label: key, span: 1 })
        currentYear = key
      } else {
        groups[groups.length - 1].span++
      }
    }
  }

  return groups
}

// ---------------------------------------------------------------------------
// Task bar positioning
// ---------------------------------------------------------------------------

/**
 * Compute bar pixel position from a task's start/end dates.
 * Returns null for unscheduled tasks (no start AND no end).
 */
export function taskToBar(
  task: GanttTask,
  config: TimelineConfig
): TaskBar | null {
  let start = task.start
  let end = task.end

  // Fully unscheduled
  if (!start && !end) return null

  // Only end (due date), no start -> use end-1 day
  if (!start && end) {
    start = subDays(end, 1)
  }

  // Only start, no end -> default 3 day duration
  if (start && !end) {
    end = addDays(start, 3)
  }

  const x = dateToX(start!, config)
  const rawWidth = dateToX(end!, config) - x
  const width = Math.max(rawWidth, 12) // minimum 12px

  return { x, width }
}

// ---------------------------------------------------------------------------
// Critical path calculation (forward + backward pass)
// ---------------------------------------------------------------------------

/**
 * Calculate critical path using forward and backward pass.
 * Returns the set of task IDs on the critical path.
 *
 * Only considers 'blocks' dependency type (A blocks B means A must
 * finish before B can start).
 */
export function calculateCriticalPath(
  tasks: GanttTask[],
  dependencies: GanttDependency[]
): Set<string> {
  const criticalIds = new Set<string>()

  // Filter to only scheduled tasks with both dates
  const scheduled = tasks.filter((t) => t.start && t.end)
  if (scheduled.length === 0) return criticalIds

  // Build adjacency: source -> [targets] (source blocks targets)
  const blocksDeps = dependencies.filter((d) => d.dependency_type === 'blocks')
  if (blocksDeps.length === 0) return criticalIds

  const taskMap = new Map<string, GanttTask>()
  for (const t of scheduled) taskMap.set(t.id, t)

  // Predecessors: taskId -> list of task IDs that block it
  const predecessors = new Map<string, string[]>()
  // Successors: taskId -> list of task IDs it blocks
  const successors = new Map<string, string[]>()

  for (const dep of blocksDeps) {
    if (!taskMap.has(dep.source_task_id) || !taskMap.has(dep.target_task_id))
      continue

    const preds = predecessors.get(dep.target_task_id) ?? []
    preds.push(dep.source_task_id)
    predecessors.set(dep.target_task_id, preds)

    const succs = successors.get(dep.source_task_id) ?? []
    succs.push(dep.target_task_id)
    successors.set(dep.source_task_id, succs)
  }

  // Duration in days for each task
  function duration(t: GanttTask): number {
    return Math.max(differenceInCalendarDays(t.end!, t.start!), 1)
  }

  // Forward pass: compute ES (earliest start) and EF (earliest finish)
  const es = new Map<string, number>()
  const ef = new Map<string, number>()

  // Topological order via Kahn's algorithm
  const inDegree = new Map<string, number>()
  for (const t of scheduled) inDegree.set(t.id, 0)
  for (const dep of blocksDeps) {
    if (!taskMap.has(dep.source_task_id) || !taskMap.has(dep.target_task_id))
      continue
    inDegree.set(
      dep.target_task_id,
      (inDegree.get(dep.target_task_id) ?? 0) + 1
    )
  }

  const queue: string[] = []
  for (const [id, deg] of inDegree) {
    if (deg === 0) queue.push(id)
  }

  const order: string[] = []
  while (queue.length > 0) {
    const id = queue.shift()!
    order.push(id)
    for (const succ of successors.get(id) ?? []) {
      const deg = (inDegree.get(succ) ?? 1) - 1
      inDegree.set(succ, deg)
      if (deg === 0) queue.push(succ)
    }
  }

  // If cycle detected (order < scheduled), return empty
  if (order.length < scheduled.length) return criticalIds

  // Forward pass
  for (const id of order) {
    const task = taskMap.get(id)!
    const taskStart = differenceInCalendarDays(task.start!, scheduled[0].start!)
    let earliest = taskStart
    for (const pred of predecessors.get(id) ?? []) {
      earliest = Math.max(earliest, ef.get(pred) ?? 0)
    }
    es.set(id, earliest)
    ef.set(id, earliest + duration(task))
  }

  // Backward pass
  const projectEnd = Math.max(...Array.from(ef.values()))
  const ls = new Map<string, number>()
  const lf = new Map<string, number>()

  for (let i = order.length - 1; i >= 0; i--) {
    const id = order[i]
    const task = taskMap.get(id)!
    let latest = projectEnd
    for (const succ of successors.get(id) ?? []) {
      latest = Math.min(latest, ls.get(succ) ?? projectEnd)
    }
    lf.set(id, latest)
    ls.set(id, latest - duration(task))
  }

  // Critical tasks: ES === LS (zero float/slack)
  for (const id of order) {
    if (es.get(id) === ls.get(id)) {
      criticalIds.add(id)
    }
  }

  return criticalIds
}

// ---------------------------------------------------------------------------
// Scheduled / unscheduled separation
// ---------------------------------------------------------------------------

export function separateScheduledAndUnscheduled(tasks: GanttTask[]): {
  scheduled: GanttTask[]
  unscheduled: GanttTask[]
} {
  const scheduled: GanttTask[] = []
  const unscheduled: GanttTask[] = []
  for (const t of tasks) {
    if (t.end) {
      scheduled.push(t)
    } else {
      unscheduled.push(t)
    }
  }
  return { scheduled, unscheduled }
}

// ---------------------------------------------------------------------------
// API data mapping
// ---------------------------------------------------------------------------

/* eslint-disable @typescript-eslint/no-explicit-any */

/**
 * Convert API task list + dependency list into Gantt-ready data.
 * Initializes is_on_critical_path to false (caller runs calculateCriticalPath).
 */
export function mapApiTasksToGantt(
  apiTasks: any[],
  apiDeps: any[]
): { tasks: GanttTask[]; dependencies: GanttDependency[] } {
  const tasks: GanttTask[] = apiTasks.map((t) => {
    let endDate: Date | null = null
    let startDate: Date | null = null

    if (t.due_date) {
      try {
        endDate = startOfDay(parseISO(t.due_date))
      } catch {
        endDate = null
      }
    }

    if (t.created_at) {
      try {
        startDate = startOfDay(parseISO(t.created_at))
      } catch {
        startDate = null
      }
    }

    // If we have an end but start is after end, clamp
    if (startDate && endDate && isAfter(startDate, endDate)) {
      startDate = subDays(endDate, 1)
    }

    return {
      id: t.id ?? '',
      title: t.title ?? '',
      project_key: t.project_key,
      task_number: t.task_number ?? 0,
      start: startDate,
      end: endDate,
      priority: t.priority ?? 'normal',
      status_name: t.status_name,
      status_color: t.status_color,
      assignee_name: t.assignee_name,
      assignee_id: t.assignee_id ?? null,
      parent_task_id: t.parent_task_id ?? null,
      depth: 0, // flat for Gantt (no nesting in timeline view)
      completed_at: t.completed_at ?? null,
      is_on_critical_path: false,
    }
  })

  // Filter dependencies to only blocks/blocked_by for arrow rendering
  const dependencies: GanttDependency[] = apiDeps
    .filter(
      (d: any) =>
        d.dependency_type === 'blocks' || d.dependency_type === 'blocked_by'
    )
    .map((d: any) => ({
      id: d.id ?? d.dependency_id ?? '',
      source_task_id: d.source_task_id ?? d.task_id ?? '',
      target_task_id: d.target_task_id ?? '',
      dependency_type: d.dependency_type ?? 'blocks',
    }))

  return { tasks, dependencies }
}

// ---------------------------------------------------------------------------
// Priority color mapping
// ---------------------------------------------------------------------------

export function priorityColor(priority: string): string {
  switch (priority) {
    case 'urgent':
      return 'var(--error)'
    case 'high':
      return 'var(--warning)'
    case 'normal':
      return 'var(--info)'
    case 'low':
      return 'var(--muted-foreground)'
    default:
      return 'var(--info)'
  }
}

/** Get bar color: prefer status color, fall back to priority. */
export function getBarColor(task: GanttTask): string {
  if (task.status_color) return task.status_color
  return priorityColor(task.priority)
}

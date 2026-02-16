/**
 * Main Gantt chart container with timeline header and task rows.
 *
 * Orchestrates data fetching (tasks + dependencies), computes critical path,
 * manages zoom level and horizontal scroll, and renders the fixed-left +
 * scrollable-right split layout. Unscheduled tasks appear in a separate
 * section below the timeline.
 */
import { useState, useRef, useMemo, useCallback, useEffect } from 'react'
import { useQueries } from '@tanstack/react-query'
import {
  ZoomIn,
  ZoomOut,
  CalendarDays,
  AlertTriangle,
  ArrowRight,
} from 'lucide-react'
import { format } from 'date-fns'
import { de } from 'date-fns/locale'
import type { components } from '@/api/types'
import { apiClient } from '@/api/client'
import { useTasks } from '@/api/hooks/useTasks'

type TaskResponse = components['schemas']['TaskResponse']
type TaskDependencyResponse = components['schemas']['TaskDependencyResponse']
import { useWorkStore } from '@/stores/work'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { TooltipProvider } from '@/components/ui/tooltip'
import GanttTimeline from './GanttTimeline'
import GanttTaskRow from './GanttTaskRow'
import GanttDependencyArrows from './GanttDependencyArrows'
import {
  type GanttTask,
  type GanttDependency,
  type ZoomLevel,
  type TaskBar,
  buildTimelineConfig,
  dateToX,
  getTimelineWidth,
  mapApiTasksToGantt,
  calculateCriticalPath,
  separateScheduledAndUnscheduled,
  taskToBar,
  generateTimelineColumns,
  ROW_HEIGHT,
  HEADER_HEIGHT,
  TASK_INFO_WIDTH,
} from './gantt-utils'

interface GanttChartProps {
  projectId: string
}

const ZOOM_LEVELS: ZoomLevel[] = ['day', 'week', 'month']
const ZOOM_LABELS: Record<ZoomLevel, string> = {
  day: 'Tag',
  week: 'Woche',
  month: 'Monat',
}

const MAX_GANTT_TASKS = 500

export default function GanttChart({ projectId }: GanttChartProps) {
  const [zoom, setZoom] = useState<ZoomLevel>('week')
  const scrollContainerRef = useRef<HTMLDivElement>(null)
  const openTaskPanel = useWorkStore((s) => s.openTaskPanel)

  // -------------------------------------------------------------------------
  // Data fetching: all project tasks
  // -------------------------------------------------------------------------
  const { data: tasksData, isLoading: tasksLoading } = useTasks({
    project_id: projectId,
    include_completed: true,
    page_size: MAX_GANTT_TASKS,
  })

  const apiTasks = (tasksData as { tasks?: TaskResponse[]; total?: number })?.tasks ?? []

  // Fetch dependencies for tasks that have blocked deps
  const taskIdsWithDeps = useMemo(
    () =>
      apiTasks
        .filter((t: TaskResponse) => t.has_blocked_deps)
        .map((t: TaskResponse) => t.id as string),
    [apiTasks]
  )

  const depQueries = useQueries({
    queries: taskIdsWithDeps.map((taskId: string) => ({
      queryKey: ['task-dependencies', taskId],
      queryFn: async () => {
        const { data, error } = await apiClient.GET(
          '/api/v1/tasks/{id}/dependencies',
          { params: { path: { id: taskId } } }
        )
        if (error) throw error
        return data
      },
      enabled: !!taskId,
    })),
  })

  const depsLoading = depQueries.some((q) => q.isLoading)

  // Merge all dependencies into a flat list
  const allApiDeps = useMemo(() => {
    const deps: TaskDependencyResponse[] = []
    for (const q of depQueries) {
      const qDeps = (q.data as { dependencies?: TaskDependencyResponse[] })?.dependencies ?? []
      for (const d of qDeps) {
        // Deduplicate by id
        if (!deps.find((existing) => existing.id === d.id)) {
          deps.push(d)
        }
      }
    }
    return deps
  }, [depQueries])

  // -------------------------------------------------------------------------
  // Transform API data to Gantt model
  // -------------------------------------------------------------------------
  const { ganttTasks, ganttDeps, criticalIds, scheduled, unscheduled } =
    useMemo(() => {
      if (apiTasks.length === 0) {
        return {
          ganttTasks: [] as GanttTask[],
          ganttDeps: [] as GanttDependency[],
          criticalIds: new Set<string>(),
          scheduled: [] as GanttTask[],
          unscheduled: [] as GanttTask[],
        }
      }

      const { tasks, dependencies } = mapApiTasksToGantt(apiTasks, allApiDeps)
      const critical = calculateCriticalPath(tasks, dependencies)

      // Mark critical path tasks
      for (const t of tasks) {
        t.is_on_critical_path = critical.has(t.id)
      }

      const { scheduled: sched, unscheduled: unsched } =
        separateScheduledAndUnscheduled(tasks)

      return {
        ganttTasks: tasks,
        ganttDeps: dependencies,
        criticalIds: critical,
        scheduled: sched,
        unscheduled: unsched,
      }
    }, [apiTasks, allApiDeps])

  // -------------------------------------------------------------------------
  // Timeline config
  // -------------------------------------------------------------------------
  const config = useMemo(
    () => buildTimelineConfig(ganttTasks, zoom),
    [ganttTasks, zoom]
  )

  const timelineWidth = useMemo(() => getTimelineWidth(config), [config])

  const columns = useMemo(() => generateTimelineColumns(config), [config])

  // Today's X position
  const todayX = useMemo(() => dateToX(new Date(), config), [config])

  // -------------------------------------------------------------------------
  // Task bar positions map (for dependency arrows)
  // -------------------------------------------------------------------------
  const taskPositions = useMemo(() => {
    const map = new Map<string, { bar: TaskBar; rowIndex: number }>()
    scheduled.forEach((task, index) => {
      const bar = taskToBar(task, config)
      if (bar) {
        map.set(task.id, { bar, rowIndex: index })
      }
    })
    return map
  }, [scheduled, config])

  // -------------------------------------------------------------------------
  // Zoom controls
  // -------------------------------------------------------------------------
  const currentZoomIndex = ZOOM_LEVELS.indexOf(zoom)

  function handleZoomIn() {
    if (currentZoomIndex > 0) {
      setZoom(ZOOM_LEVELS[currentZoomIndex - 1])
    }
  }

  function handleZoomOut() {
    if (currentZoomIndex < ZOOM_LEVELS.length - 1) {
      setZoom(ZOOM_LEVELS[currentZoomIndex + 1])
    }
  }

  // Scroll to today
  const scrollToToday = useCallback(() => {
    if (scrollContainerRef.current) {
      const containerWidth = scrollContainerRef.current.clientWidth - TASK_INFO_WIDTH
      scrollContainerRef.current.scrollLeft = Math.max(
        0,
        todayX - containerWidth / 2
      )
    }
  }, [todayX])

  // Auto-scroll to today on first render
  useEffect(() => {
    if (!tasksLoading && scheduled.length > 0) {
      // Small delay to allow DOM render
      const timer = setTimeout(scrollToToday, 100)
      return () => clearTimeout(timer)
    }
  }, [tasksLoading, scheduled.length, scrollToToday])

  // -------------------------------------------------------------------------
  // Visible date range display
  // -------------------------------------------------------------------------
  const visibleRange = useMemo(() => {
    const start = format(config.startDate, 'dd. MMM', { locale: de })
    const end = format(config.endDate, 'dd. MMM yyyy', { locale: de })
    return `${start} - ${end}`
  }, [config])

  // -------------------------------------------------------------------------
  // Task click handler
  // -------------------------------------------------------------------------
  const handleTaskClick = useCallback(
    (taskId: string) => {
      openTaskPanel(taskId)
    },
    [openTaskPanel]
  )

  // -------------------------------------------------------------------------
  // Render
  // -------------------------------------------------------------------------

  if (tasksLoading) {
    return (
      <div className="p-6 space-y-3">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-96 w-full" />
      </div>
    )
  }

  if (apiTasks.length === 0) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="text-center">
          <CalendarDays className="mx-auto h-12 w-12 text-muted-foreground/50" />
          <p className="mt-4 text-lg font-semibold text-foreground">
            Keine Aufgaben vorhanden
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            Erstellen Sie Aufgaben, um die Timeline zu sehen.
          </p>
        </div>
      </div>
    )
  }

  if (apiTasks.length >= MAX_GANTT_TASKS) {
    // Warning for large projects
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="text-center">
          <AlertTriangle className="mx-auto h-12 w-12 text-warning" />
          <p className="mt-4 text-lg font-semibold text-foreground">
            Zu viele Aufgaben für Gantt-Ansicht
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            Dieses Projekt hat mehr als {MAX_GANTT_TASKS} Aufgaben.
            Bitte nutzen Sie die Listen- oder Kanban-Ansicht.
          </p>
        </div>
      </div>
    )
  }

  const scheduledHeight = scheduled.length * ROW_HEIGHT
  const unscheduledHeight = unscheduled.length * ROW_HEIGHT
  const totalContentHeight =
    HEADER_HEIGHT +
    scheduledHeight +
    (unscheduled.length > 0 ? ROW_HEIGHT + unscheduledHeight : 0)

  return (
    <TooltipProvider delayDuration={300}>
      <div className="flex h-full flex-col">
        {/* Controls bar */}
        <div className="flex items-center justify-between border-b border-border px-4 py-2">
          <div className="flex items-center gap-3">
            {/* Zoom buttons */}
            <div className="flex items-center gap-1">
              <Button
                variant="ghost"
                size="sm"
                onClick={handleZoomIn}
                disabled={currentZoomIndex === 0}
                title="Hineinzoomen"
              >
                <ZoomIn className="h-4 w-4" />
              </Button>
              <span className="text-xs font-medium text-muted-foreground min-w-[50px] text-center">
                {ZOOM_LABELS[zoom]}
              </span>
              <Button
                variant="ghost"
                size="sm"
                onClick={handleZoomOut}
                disabled={currentZoomIndex === ZOOM_LEVELS.length - 1}
                title="Herauszoomen"
              >
                <ZoomOut className="h-4 w-4" />
              </Button>
            </div>

            {/* Today button */}
            <Button variant="outline" size="sm" onClick={scrollToToday}>
              Heute
            </Button>

            {/* Date range */}
            <span className="text-xs text-muted-foreground">{visibleRange}</span>
          </div>

          {/* Legend */}
          <div className="flex items-center gap-4 text-xs text-muted-foreground">
            <div className="flex items-center gap-1.5">
              <div className="h-3 w-6 rounded border-2 border-error bg-error/20" />
              <span>Kritischer Pfad</span>
            </div>
            <div className="flex items-center gap-1.5">
              <ArrowRight className="h-3 w-3" />
              <span>Abhängigkeit</span>
            </div>
          </div>
        </div>

        {/* Chart area */}
        <div
          ref={scrollContainerRef}
          className="flex-1 min-h-0 overflow-auto"
        >
          <div style={{ width: TASK_INFO_WIDTH + timelineWidth, minHeight: totalContentHeight }}>
            {/* Timeline header: only over the scrollable part */}
            <div className="flex sticky top-0 z-20 bg-background">
              {/* Fixed left header */}
              <div
                className="flex-shrink-0 flex items-center border-b border-r border-border px-3 text-xs font-medium text-muted-foreground bg-background"
                style={{ width: TASK_INFO_WIDTH, height: HEADER_HEIGHT }}
              >
                Aufgabe
              </div>
              {/* Timeline columns header */}
              <div style={{ width: timelineWidth }}>
                <GanttTimeline config={config} />
              </div>
            </div>

            {/* Scheduled tasks */}
            <div className="relative">
              {/* Today marker line */}
              {todayX > 0 && (
                <div
                  className="absolute top-0 bottom-0 border-l-2 border-dashed border-red-400 z-[5] pointer-events-none"
                  style={{
                    left: TASK_INFO_WIDTH + todayX,
                    height: scheduledHeight,
                  }}
                />
              )}

              {/* Dependency arrows SVG overlay */}
              {ganttDeps.length > 0 && !depsLoading && (
                <div
                  className="absolute pointer-events-none"
                  style={{ left: TASK_INFO_WIDTH, top: 0 }}
                >
                  <GanttDependencyArrows
                    dependencies={ganttDeps}
                    taskPositions={taskPositions}
                    criticalIds={criticalIds}
                    width={timelineWidth}
                    height={scheduledHeight}
                  />
                </div>
              )}

              {/* Weekend column backgrounds */}
              {config.zoom === 'day' && (
                <div
                  className="absolute top-0 pointer-events-none"
                  style={{ left: TASK_INFO_WIDTH, height: scheduledHeight }}
                >
                  {columns.map((col, i) =>
                    col.isWeekend ? (
                      <div
                        key={i}
                        className="absolute top-0 bottom-0 bg-muted/15"
                        style={{
                          left: i * config.columnWidth,
                          width: config.columnWidth,
                          height: scheduledHeight,
                        }}
                      />
                    ) : null
                  )}
                </div>
              )}

              {/* Task rows */}
              {scheduled.map((task, index) => (
                <GanttTaskRow
                  key={task.id}
                  task={task}
                  config={config}
                  rowIndex={index}
                  onTaskClick={handleTaskClick}
                  timelineWidth={timelineWidth}
                />
              ))}
            </div>

            {/* Unscheduled section */}
            {unscheduled.length > 0 && (
              <div>
                {/* Section header */}
                <div
                  className="flex items-center border-y border-border bg-muted/30 px-3 text-xs font-medium text-muted-foreground"
                  style={{ height: ROW_HEIGHT }}
                >
                  Ungeplant ({unscheduled.length})
                </div>

                {/* Unscheduled task rows */}
                {unscheduled.map((task, index) => (
                  <GanttTaskRow
                    key={task.id}
                    task={task}
                    config={config}
                    rowIndex={scheduled.length + index + 1}
                    onTaskClick={handleTaskClick}
                    timelineWidth={timelineWidth}
                  />
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </TooltipProvider>
  )
}

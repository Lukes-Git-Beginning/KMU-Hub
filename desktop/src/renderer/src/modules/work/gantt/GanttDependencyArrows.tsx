/**
 * SVG overlay rendering dependency arrows between task bars.
 *
 * Draws smooth bezier paths from the right edge of source bars to the
 * left edge of target bars. Critical path arrows are red.
 * Uses SVG marker for arrowheads.
 */
import type { GanttDependency, TaskBar } from './gantt-utils'
import { ROW_HEIGHT } from './gantt-utils'

interface GanttDependencyArrowsProps {
  dependencies: GanttDependency[]
  taskPositions: Map<string, { bar: TaskBar; rowIndex: number }>
  criticalIds: Set<string>
  width: number
  height: number
}

export default function GanttDependencyArrows({
  dependencies,
  taskPositions,
  criticalIds,
  width,
  height,
}: GanttDependencyArrowsProps) {
  if (dependencies.length === 0) return null

  return (
    <svg
      className="absolute top-0 left-0 pointer-events-none"
      width={width}
      height={height}
      style={{ overflow: 'visible' }}
    >
      <defs>
        <marker
          id="arrowhead"
          markerWidth="8"
          markerHeight="6"
          refX="8"
          refY="3"
          orient="auto"
        >
          <polygon
            points="0 0, 8 3, 0 6"
            className="fill-muted-foreground"
          />
        </marker>
        <marker
          id="arrowhead-critical"
          markerWidth="8"
          markerHeight="6"
          refX="8"
          refY="3"
          orient="auto"
        >
          <polygon points="0 0, 8 3, 0 6" className="fill-error" />
        </marker>
      </defs>

      {dependencies.map((dep) => {
        const source = taskPositions.get(dep.source_task_id)
        const target = taskPositions.get(dep.target_task_id)
        if (!source || !target) return null

        const isCritical =
          criticalIds.has(dep.source_task_id) &&
          criticalIds.has(dep.target_task_id)

        // Start: right edge of source bar, vertically centered in its row
        const sx = source.bar.x + source.bar.width
        const sy = source.rowIndex * ROW_HEIGHT + ROW_HEIGHT / 2

        // End: left edge of target bar
        const ex = target.bar.x
        const ey = target.rowIndex * ROW_HEIGHT + ROW_HEIGHT / 2

        // Bezier control points for a smooth S-curve
        const dx = Math.abs(ex - sx)
        const cpOffset = Math.max(dx * 0.4, 20)

        const path = `M ${sx} ${sy} C ${sx + cpOffset} ${sy}, ${ex - cpOffset} ${ey}, ${ex} ${ey}`

        return (
          <path
            key={dep.id}
            d={path}
            fill="none"
            stroke={isCritical ? 'var(--error)' : 'var(--muted-foreground)'}
            strokeWidth={isCritical ? 2 : 1.5}
            strokeDasharray={isCritical ? undefined : '4 2'}
            markerEnd={
              isCritical
                ? 'url(#arrowhead-critical)'
                : 'url(#arrowhead)'
            }
            opacity={0.7}
          />
        )
      })}
    </svg>
  )
}

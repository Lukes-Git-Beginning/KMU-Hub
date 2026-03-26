/**
 * Custom react-flow node for conditions.
 *
 * Yellow header with filter icon.
 * Input handle at top, two output handles at bottom (Ja/Nein) for branching.
 */
import { memo } from 'react'
import { Handle, Position, type NodeProps } from '@xyflow/react'
import { Filter } from 'lucide-react'
import type { ConditionConfig } from '@/api/automation-types'

interface ConditionNodeData {
  conditions: ConditionConfig
  [key: string]: unknown
}

function ConditionNodeComponent({ data, selected }: NodeProps) {
  const nodeData = data as ConditionNodeData
  const conditions = nodeData.conditions

  const summary =
    conditions?.mode === 'expression'
      ? conditions.expression
        ? conditions.expression.substring(0, 40) + (conditions.expression.length > 40 ? '...' : '')
        : 'Kein Ausdruck'
      : conditions?.simple
        ? 'Einfache Bedingung'
        : 'Keine Bedingung'

  return (
    <div
      className={`rounded-lg border-2 shadow-sm min-w-[180px] ${
        selected
          ? 'border-yellow-500 ring-2 ring-yellow-500/20'
          : 'border-yellow-300 dark:border-yellow-700'
      }`}
    >
      {/* Input handle */}
      <Handle
        type="target"
        position={Position.Top}
        className="!bg-warning !w-3 !h-3 !border-2 !border-white"
      />

      {/* Header */}
      <div className="flex items-center gap-2 rounded-t-md bg-warning px-3 py-2">
        <Filter className="h-4 w-4 text-white" />
        <span className="text-xs font-semibold text-white">Bedingung</span>
      </div>

      {/* Body */}
      <div className="bg-background rounded-b-md px-3 py-2">
        <p className="text-xs text-foreground font-medium truncate">{summary}</p>
      </div>

      {/* Two output handles for branching: Ja (left) and Nein (right) */}
      <Handle
        type="source"
        position={Position.Bottom}
        id="yes"
        style={{ left: '30%' }}
        className="!bg-green-500 !w-3 !h-3 !border-2 !border-white"
      />
      <Handle
        type="source"
        position={Position.Bottom}
        id="no"
        style={{ left: '70%' }}
        className="!bg-destructive !w-3 !h-3 !border-2 !border-white"
      />

      {/* Labels for handles */}
      <div className="flex justify-between px-3 pb-1 -mt-1">
        <span className="text-[9px] text-green-600 font-medium">Ja</span>
        <span className="text-[9px] text-destructive font-medium">Nein</span>
      </div>
    </div>
  )
}

export const ConditionNode = memo(ConditionNodeComponent)

/**
 * Custom react-flow node for triggers.
 *
 * Blue header with lightning bolt icon.
 * Single output handle at the bottom.
 */
import { memo } from 'react'
import { useTranslation } from 'react-i18next'
import { Handle, Position, type NodeProps } from '@xyflow/react'
import { Zap } from 'lucide-react'

interface TriggerNodeData {
  triggerType: string
  triggerConfig: Record<string, unknown>
  [key: string]: unknown
}

function TriggerNodeComponent({ data, selected }: NodeProps) {
  const { t } = useTranslation()
  const nodeData = data as TriggerNodeData

  return (
    <div
      className={`rounded-lg border-2 shadow-sm min-w-[180px] ${
        selected
          ? 'border-blue-500 ring-2 ring-blue-500/20'
          : 'border-blue-300 dark:border-blue-700'
      }`}
    >
      {/* Header */}
      <div className="flex items-center gap-2 rounded-t-md bg-blue-500 px-3 py-2">
        <Zap className="h-4 w-4 text-white" />
        <span className="text-xs font-semibold text-white">Trigger</span>
      </div>

      {/* Body */}
      <div className="bg-background rounded-b-md px-3 py-2">
        <p className="text-xs text-foreground font-medium truncate">
          {nodeData.triggerType || t('automatisierung.node.notConfigured')}
        </p>
      </div>

      {/* Output handle */}
      <Handle
        type="source"
        position={Position.Bottom}
        className="!bg-blue-500 !w-3 !h-3 !border-2 !border-white"
      />
    </div>
  )
}

export const TriggerNode = memo(TriggerNodeComponent)

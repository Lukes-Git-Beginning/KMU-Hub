/**
 * Custom react-flow node for actions.
 *
 * Green header with play icon.
 * Input handle at top, output handle at bottom for chaining.
 */
import { memo } from 'react'
import { useTranslation } from 'react-i18next'
import { Handle, Position, type NodeProps } from '@xyflow/react'
import { Play } from 'lucide-react'

interface ActionNodeData {
  actionType: string
  actionConfig: Record<string, unknown>
  onError: 'continue' | 'abort'
  [key: string]: unknown
}

function ActionNodeComponent({ data, selected }: NodeProps) {
  const { t } = useTranslation()
  const nodeData = data as ActionNodeData

  return (
    <div
      className={`rounded-lg border-2 shadow-sm min-w-[180px] ${
        selected
          ? 'border-green-500 ring-2 ring-green-500/20'
          : 'border-green-300 dark:border-green-700'
      }`}
    >
      {/* Input handle */}
      <Handle
        type="target"
        position={Position.Top}
        className="!bg-green-500 !w-3 !h-3 !border-2 !border-white"
      />

      {/* Header */}
      <div className="flex items-center gap-2 rounded-t-md bg-green-500 px-3 py-2">
        <Play className="h-4 w-4 text-white" />
        <span className="text-xs font-semibold text-white">{t('automatisierung.node.action')}</span>
      </div>

      {/* Body */}
      <div className="bg-background rounded-b-md px-3 py-2">
        <p className="text-xs text-foreground font-medium truncate">
          {nodeData.actionType || t('automatisierung.node.notConfigured')}
        </p>
        {nodeData.onError === 'continue' && (
          <p className="text-[10px] text-muted-foreground mt-0.5">
            {t('automatisierung.node.continueOnError')}
          </p>
        )}
      </div>

      {/* Output handle */}
      <Handle
        type="source"
        position={Position.Bottom}
        className="!bg-green-500 !w-3 !h-3 !border-2 !border-white"
      />
    </div>
  )
}

export const ActionNode = memo(ActionNodeComponent)

/**
 * Custom node type registry for react-flow automation editor.
 */
import { TriggerNode } from './TriggerNode'
import { ConditionNode } from './ConditionNode'
import { ActionNode } from './ActionNode'

export const nodeTypes = {
  trigger: TriggerNode,
  condition: ConditionNode,
  action: ActionNode,
} as const

/**
 * React Flow visual node editor for automation workflows.
 *
 * Full canvas with custom node types (trigger, condition, action),
 * MiniMap, Controls, and serialization to/from workflow JSON.
 */
import { useCallback, useMemo } from 'react'
import {
  ReactFlow,
  ReactFlowProvider,
  MiniMap,
  Controls,
  Background,
  BackgroundVariant,
  useNodesState,
  useEdgesState,
  addEdge,
  type Connection,
  type Edge,
  type Node,
  type OnConnect,
  type NodeTypes,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { Wand2, Save, Plus } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { useAutomatisierungStore } from '@/stores/automatisierung'
import { useCreateAutomation, useUpdateAutomation } from '@/api/hooks/useAutomation'
import type { Automation, ActionConfig, ConditionConfig } from '@/api/automation-types'
import { nodeTypes as customNodeTypes } from './nodes/nodeTypes'

// ---------------------------------------------------------------------------
// Workflow <-> Node/Edge serialization
// ---------------------------------------------------------------------------

function workflowToNodesAndEdges(workflow: Partial<Automation>): {
  nodes: Node[]
  edges: Edge[]
} {
  const nodes: Node[] = []
  const edges: Edge[] = []

  let y = 0

  // Trigger node
  if (workflow.trigger_type) {
    nodes.push({
      id: 'trigger-0',
      type: 'trigger',
      position: { x: 250, y },
      data: {
        triggerType: workflow.trigger_type,
        triggerConfig: workflow.trigger_config ?? {},
      },
    })
    y += 150
  }

  // Condition node
  const conditions = workflow.conditions as ConditionConfig | undefined
  if (conditions && (conditions.simple || conditions.expression)) {
    nodes.push({
      id: 'condition-0',
      type: 'condition',
      position: { x: 250, y },
      data: { conditions },
    })
    if (nodes.length > 1) {
      edges.push({
        id: 'e-trigger-condition',
        source: 'trigger-0',
        target: 'condition-0',
        animated: true,
      })
    }
    y += 150
  }

  // Action nodes
  const actions = (workflow.actions ?? []) as ActionConfig[]
  const prevNodeId =
    nodes.length > 1 && conditions?.simple
      ? 'condition-0'
      : nodes.length > 0
        ? 'trigger-0'
        : null

  actions.forEach((action, idx) => {
    const nodeId = `action-${idx}`
    nodes.push({
      id: nodeId,
      type: 'action',
      position: { x: 250, y },
      data: {
        actionType: action.type,
        actionConfig: action.config,
        onError: action.on_error,
      },
    })

    const sourceId = idx === 0 ? prevNodeId : `action-${idx - 1}`
    if (sourceId) {
      edges.push({
        id: `e-${sourceId}-${nodeId}`,
        source: sourceId,
        target: nodeId,
        animated: true,
      })
    }

    y += 130
  })

  return { nodes, edges }
}

function nodesAndEdgesToWorkflow(
  nodes: Node[],
  _edges: Edge[],
  current: Partial<Automation>,
): Partial<Automation> {
  const triggerNode = nodes.find((n) => n.type === 'trigger')
  const conditionNode = nodes.find((n) => n.type === 'condition')
  const actionNodes = nodes
    .filter((n) => n.type === 'action')
    .sort((a, b) => a.position.y - b.position.y)

  const updated: Partial<Automation> = { ...current }

  if (triggerNode) {
    updated.trigger_type = triggerNode.data.triggerType as string
    updated.trigger_config = triggerNode.data.triggerConfig as Record<string, unknown>
  }

  if (conditionNode) {
    updated.conditions = conditionNode.data.conditions as ConditionConfig
  }

  updated.actions = actionNodes.map((n) => ({
    type: (n.data.actionType as string) ?? '',
    config: (n.data.actionConfig as Record<string, unknown>) ?? {},
    on_error: (n.data.onError as 'continue' | 'abort') ?? 'abort',
  }))

  return updated
}

// ---------------------------------------------------------------------------
// Editor inner (needs ReactFlowProvider wrapper)
// ---------------------------------------------------------------------------

function EditorInner({ onClose }: { onClose: () => void }) {
  const {
    draftWorkflow,
    setDraftWorkflow,
    setEditorMode,
  } = useAutomatisierungStore()

  const createMutation = useCreateAutomation()
  const updateMutation = useUpdateAutomation()
  const isEditing = !!draftWorkflow?.id

  // Initialize nodes/edges from draft
  const initial = useMemo(
    () => workflowToNodesAndEdges(draftWorkflow ?? {}),
    // eslint-disable-next-line react-hooks/exhaustive-deps -- mount only: initial snapshot of draft
    [],
  )

  const [nodes, setNodes, onNodesChange] = useNodesState(initial.nodes)
  const [edges, setEdges, onEdgesChange] = useEdgesState(initial.edges)

  const onConnect: OnConnect = useCallback(
    (connection: Connection) => {
      setEdges((eds) => addEdge({ ...connection, animated: true }, eds))
    },
    [setEdges],
  )

  // Connection validation
  const isValidConnection = useCallback((connection: Connection) => {
    const sourceType = connection.source?.split('-')[0]
    const targetType = connection.target?.split('-')[0]

    // No backward connections: action->trigger or condition->trigger
    if (targetType === 'trigger') return false
    // Trigger can connect to condition or action
    if (sourceType === 'trigger') return targetType === 'condition' || targetType === 'action'
    // Condition can connect to action
    if (sourceType === 'condition') return targetType === 'action'
    // Action can connect to action
    if (sourceType === 'action') return targetType === 'action'
    return false
  }, [])

  // Delete selected nodes
  const onKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Delete' || e.key === 'Backspace') {
        setNodes((nds) => nds.filter((n) => !n.selected))
        setEdges((eds) => eds.filter((e) => !e.selected))
      }
    },
    [setNodes, setEdges],
  )

  // Add node buttons
  const addTriggerNode = () => {
    const id = `trigger-${Date.now()}`
    setNodes((nds) => [
      ...nds,
      {
        id,
        type: 'trigger',
        position: { x: 250, y: 50 },
        data: { triggerType: '', triggerConfig: {} },
      },
    ])
  }

  const addConditionNode = () => {
    const id = `condition-${Date.now()}`
    setNodes((nds) => [
      ...nds,
      {
        id,
        type: 'condition',
        position: { x: 250, y: 200 },
        data: { conditions: { mode: 'simple', simple: null, expression: '' } },
      },
    ])
  }

  const addActionNode = () => {
    const id = `action-${Date.now()}`
    setNodes((nds) => [
      ...nds,
      {
        id,
        type: 'action',
        position: { x: 250, y: 350 },
        data: { actionType: '', actionConfig: {}, onError: 'abort' },
      },
    ])
  }

  // Save
  const handleSave = () => {
    const updated = nodesAndEdgesToWorkflow(nodes, edges, draftWorkflow ?? {})
    setDraftWorkflow(updated)

    if (!updated.name || !updated.trigger_type) return

    const payload = {
      name: updated.name!,
      description: updated.description,
      scope: updated.scope,
      trigger_type: updated.trigger_type!,
      trigger_config: updated.trigger_config,
      conditions: updated.conditions,
      actions: updated.actions,
      max_steps: updated.max_steps ?? 10,
    }

    if (isEditing && draftWorkflow?.id) {
      updateMutation.mutate(
        { id: draftWorkflow.id, ...payload },
        { onSuccess: () => onClose() },
      )
    } else {
      createMutation.mutate(payload, { onSuccess: () => onClose() })
    }
  }

  const handleSwitchToWizard = () => {
    // Sync current graph back to draft
    const updated = nodesAndEdgesToWorkflow(nodes, edges, draftWorkflow ?? {})
    setDraftWorkflow(updated)
    setEditorMode('wizard')
  }

  const isPending = createMutation.isPending || updateMutation.isPending

  return (
    <div className="flex flex-col h-full" onKeyDown={onKeyDown}>
      {/* Toolbar */}
      <div className="flex items-center justify-between border-b border-border px-4 py-2">
        <div className="flex items-center gap-2">
          <DialogTitle className="text-sm font-semibold text-foreground">
            Visueller Editor
          </DialogTitle>
          <div className="h-4 w-px bg-border" />
          <button
            onClick={addTriggerNode}
            className="flex items-center gap-1 rounded-md bg-blue-500/10 px-2 py-1 text-[11px] font-medium text-blue-600 hover:bg-blue-500/20 transition-colors"
          >
            <Plus className="h-3 w-3" />
            Trigger
          </button>
          <button
            onClick={addConditionNode}
            className="flex items-center gap-1 rounded-md bg-warning/10 px-2 py-1 text-[11px] font-medium text-warning-foreground hover:bg-warning/20 transition-colors"
          >
            <Plus className="h-3 w-3" />
            Bedingung
          </button>
          <button
            onClick={addActionNode}
            className="flex items-center gap-1 rounded-md bg-green-500/10 px-2 py-1 text-[11px] font-medium text-green-600 hover:bg-green-500/20 transition-colors"
          >
            <Plus className="h-3 w-3" />
            Aktion
          </button>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={handleSwitchToWizard}
            className="flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
          >
            <Wand2 className="h-3.5 w-3.5" />
            Zum Wizard wechseln
          </button>
          <button
            onClick={handleSave}
            disabled={isPending}
            className="flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
          >
            <Save className="h-3.5 w-3.5" />
            {isPending ? 'Speichern...' : 'Speichern'}
          </button>
          <button
            onClick={onClose}
            className="rounded-md border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
          >
            Schließen
          </button>
        </div>
      </div>

      {/* Canvas */}
      <div className="flex-1">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onConnect={onConnect}
          isValidConnection={isValidConnection}
          nodeTypes={customNodeTypes as NodeTypes}
          fitView
          deleteKeyCode={['Delete', 'Backspace']}
          className="bg-background"
        >
          <Controls position="bottom-left" />
          <MiniMap
            position="bottom-right"
            nodeColor={(node) => {
              switch (node.type) {
                case 'trigger':
                  return '#3b82f6'
                case 'condition':
                  return '#eab308'
                case 'action':
                  return '#22c55e'
                default:
                  return '#64748b'
              }
            }}
          />
          <Background variant={BackgroundVariant.Dots} gap={16} size={1} />
        </ReactFlow>
      </div>
      <DialogDescription className="sr-only">
        Visueller Editor für Automatisierungs-Workflows
      </DialogDescription>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Exported editor with provider
// ---------------------------------------------------------------------------

export function AutomationEditor({ onClose }: { onClose: () => void }) {
  return (
    <Dialog open onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent className="gap-0 p-0 max-w-full w-screen h-screen rounded-none border-0 [&>button:last-child]:hidden">
        <ReactFlowProvider>
          <EditorInner onClose={onClose} />
        </ReactFlowProvider>
      </DialogContent>
    </Dialog>
  )
}

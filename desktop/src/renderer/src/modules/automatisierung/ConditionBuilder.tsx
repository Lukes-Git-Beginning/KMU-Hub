/**
 * Condition builder component for the automation wizard.
 *
 * Two modes:
 * - "Einfach" (simple): Dropdown UI with recursive AND/OR groups
 * - "Ausdruck" (expression): Raw expression text field with test button
 *
 * Both modes write to the same draft.conditions in the Zustand store.
 */
import { useState, useCallback } from 'react'
import {
  Plus,
  Trash2,
  FolderPlus,
  Play,
  CheckCircle,
  XCircle,
} from 'lucide-react'
import * as Switch from '@radix-ui/react-switch'
import { useAutomatisierungStore } from '@/stores/automatisierung'
import { useTriggerDefinitions, useTestCondition } from '@/api/hooks/useAutomation'
import type {
  ConditionConfig,
  SimpleCondition,
  TriggerField,
} from '@/api/automation-types'

// ---------------------------------------------------------------------------
// Recursive condition group
// ---------------------------------------------------------------------------

interface ConditionGroupProps {
  condition: SimpleCondition
  onChange: (condition: SimpleCondition) => void
  onRemove?: () => void
  fields: TriggerField[]
  depth: number
}

function ConditionGroup({
  condition,
  onChange,
  onRemove,
  fields,
  depth,
}: ConditionGroupProps) {
  const isAnd = !!condition.and
  const isOr = !!condition.or
  const isGroup = isAnd || isOr
  const isLeaf = !isGroup

  // Leaf condition
  if (isLeaf) {
    return (
      <LeafCondition
        condition={condition}
        onChange={onChange}
        onRemove={onRemove}
        fields={fields}
      />
    )
  }

  const children = isAnd ? condition.and! : condition.or!
  const groupType = isAnd ? 'and' : 'or'

  const updateChild = (index: number, child: SimpleCondition) => {
    const updated = [...children]
    updated[index] = child
    onChange({ [groupType]: updated })
  }

  const removeChild = (index: number) => {
    const updated = children.filter((_, i) => i !== index)
    onChange({ [groupType]: updated })
  }

  const addLeaf = () => {
    onChange({
      [groupType]: [...children, { field: '', operator: '', value: '' }],
    })
  }

  const addGroup = () => {
    onChange({
      [groupType]: [...children, { and: [{ field: '', operator: '', value: '' }] }],
    })
  }

  const toggleGroupType = () => {
    if (isAnd) {
      onChange({ or: children })
    } else {
      onChange({ and: children })
    }
  }

  return (
    <div
      className={`rounded-lg border border-border p-3 space-y-2 ${
        depth > 0 ? 'ml-4 border-dashed' : ''
      }`}
    >
      <div className="flex items-center justify-between">
        <button
          onClick={toggleGroupType}
          className="rounded-md bg-secondary px-2.5 py-1 text-xs font-bold text-foreground hover:bg-secondary/80 transition-colors uppercase"
        >
          {isAnd ? 'UND' : 'ODER'}
        </button>
        <div className="flex items-center gap-1">
          <button
            onClick={addLeaf}
            className="flex items-center gap-1 rounded-md border border-border px-2 py-1 text-[11px] text-foreground hover:bg-secondary transition-colors"
          >
            <Plus className="h-3 w-3" />
            Bedingung
          </button>
          <button
            onClick={addGroup}
            className="flex items-center gap-1 rounded-md border border-border px-2 py-1 text-[11px] text-foreground hover:bg-secondary transition-colors"
          >
            <FolderPlus className="h-3 w-3" />
            Gruppe
          </button>
          {onRemove && (
            <button
              onClick={onRemove}
              className="rounded-md p-1 text-muted-foreground hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </button>
          )}
        </div>
      </div>

      {children.map((child, idx) => (
        <ConditionGroup
          key={idx}
          condition={child}
          onChange={(c) => updateChild(idx, c)}
          onRemove={() => removeChild(idx)}
          fields={fields}
          depth={depth + 1}
        />
      ))}

      {children.length === 0 && (
        <p className="text-xs text-muted-foreground py-2 text-center">
          Keine Bedingungen. Klicken Sie auf "Bedingung hinzufuegen".
        </p>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Leaf condition (field, operator, value)
// ---------------------------------------------------------------------------

function LeafCondition({
  condition,
  onChange,
  onRemove,
  fields,
}: {
  condition: SimpleCondition
  onChange: (condition: SimpleCondition) => void
  onRemove?: () => void
  fields: TriggerField[]
}) {
  const selectedField = fields.find((f) => f.key === condition.field)
  const operators = selectedField?.operators ?? []

  return (
    <div className="flex items-center gap-2 rounded-md bg-secondary/30 p-2">
      {/* Field */}
      <select
        value={condition.field ?? ''}
        onChange={(e) => onChange({ ...condition, field: e.target.value, operator: '', value: '' })}
        className="flex-1 rounded-md border border-border bg-background px-2 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-focus-ring"
      >
        <option value="">Feld waehlen...</option>
        {fields.map((f) => (
          <option key={f.key} value={f.key}>
            {f.label}
          </option>
        ))}
      </select>

      {/* Operator */}
      <select
        value={condition.operator ?? ''}
        onChange={(e) => onChange({ ...condition, operator: e.target.value })}
        className="w-32 rounded-md border border-border bg-background px-2 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-focus-ring"
      >
        <option value="">Operator...</option>
        {operators.map((op) => (
          <option key={op} value={op}>
            {op}
          </option>
        ))}
      </select>

      {/* Value */}
      <input
        type={selectedField?.type === 'number' ? 'number' : 'text'}
        value={String(condition.value ?? '')}
        onChange={(e) => onChange({ ...condition, value: e.target.value })}
        placeholder="Wert"
        className="flex-1 rounded-md border border-border bg-background px-2 py-1.5 text-xs text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-1 focus:ring-focus-ring"
      />

      {/* Remove */}
      {onRemove && (
        <button
          onClick={onRemove}
          className="rounded-md p-1 text-muted-foreground hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
        >
          <Trash2 className="h-3.5 w-3.5" />
        </button>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Expression mode
// ---------------------------------------------------------------------------

function ExpressionEditor({ fields }: { fields: TriggerField[] }) {
  const { draftWorkflow, updateDraftConditions } = useAutomatisierungStore()
  const conditions = draftWorkflow?.conditions as ConditionConfig | undefined
  const testMutation = useTestCondition()

  const handleTest = () => {
    if (!conditions?.expression) return
    testMutation.mutate({
      condition: conditions,
      sample_env: {},
    })
  }

  return (
    <div className="space-y-3">
      {/* Expression textarea */}
      <div>
        <label className="text-xs font-medium text-foreground">Ausdruck</label>
        <textarea
          value={conditions?.expression ?? ''}
          onChange={(e) =>
            updateDraftConditions({
              mode: 'expression',
              simple: null,
              expression: e.target.value,
            })
          }
          placeholder='z.B. deal.value > 10000 && deal.stage == "won"'
          rows={4}
          className="mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground font-mono placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring resize-none"
        />
      </div>

      {/* Available variables */}
      <div>
        <h4 className="text-xs font-medium text-muted-foreground mb-1">
          Verfuegbare Variablen
        </h4>
        <div className="flex flex-wrap gap-1">
          {fields.map((f) => (
            <span
              key={f.key}
              className="inline-flex items-center gap-1 rounded-md bg-secondary px-2 py-0.5 text-[11px] font-mono text-foreground"
            >
              {f.key}
              <span className="text-muted-foreground">({f.type})</span>
            </span>
          ))}
        </div>
      </div>

      {/* Test button */}
      <div className="flex items-center gap-3">
        <button
          onClick={handleTest}
          disabled={!conditions?.expression || testMutation.isPending}
          className="flex items-center gap-1.5 rounded-md bg-secondary px-3 py-1.5 text-xs font-medium text-foreground hover:bg-secondary/80 transition-colors disabled:opacity-50"
        >
          <Play className="h-3 w-3" />
          {testMutation.isPending ? 'Teste...' : 'Ausdruck testen'}
        </button>

        {testMutation.data && (
          <span className="flex items-center gap-1 text-xs">
            {testMutation.data.matches ? (
              <>
                <CheckCircle className="h-4 w-4 text-green-500" />
                <span className="text-green-600">Erfolgreich</span>
              </>
            ) : (
              <>
                <XCircle className="h-4 w-4 text-red-500" />
                <span className="text-red-600">
                  Fehlgeschlagen
                  {testMutation.data.error
                    ? `: ${testMutation.data.error}`
                    : ''}
                </span>
              </>
            )}
          </span>
        )}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export function ConditionBuilder() {
  const { draftWorkflow, updateDraftConditions } = useAutomatisierungStore()
  const { data: triggerData } = useTriggerDefinitions()
  const conditions = draftWorkflow?.conditions as ConditionConfig | undefined
  const isExpression = conditions?.mode === 'expression'

  // Get fields from selected trigger type
  const triggers = triggerData?.triggers ?? []
  const selectedTrigger = triggers.find(
    (t) => t.type === draftWorkflow?.trigger_type,
  )
  const fields = selectedTrigger?.fields ?? []

  const handleModeToggle = (useExpression: boolean) => {
    if (useExpression) {
      updateDraftConditions({
        mode: 'expression',
        simple: conditions?.simple ?? null,
        expression: conditions?.expression ?? '',
      })
    } else {
      updateDraftConditions({
        mode: 'simple',
        simple: conditions?.simple ?? { and: [] },
        expression: conditions?.expression ?? '',
      })
    }
  }

  const handleSimpleChange = (simple: SimpleCondition) => {
    updateDraftConditions({
      mode: 'simple',
      simple,
      expression: conditions?.expression ?? '',
    })
  }

  return (
    <div className="space-y-4">
      {/* Mode toggle */}
      <div className="flex items-center gap-3">
        <span
          className={`text-xs font-medium ${!isExpression ? 'text-foreground' : 'text-muted-foreground'}`}
        >
          Einfach
        </span>
        <Switch.Root
          checked={isExpression}
          onCheckedChange={handleModeToggle}
          className="relative inline-flex h-5 w-9 items-center rounded-full transition-colors data-[state=checked]:bg-primary data-[state=unchecked]:bg-secondary"
        >
          <Switch.Thumb className="block h-4 w-4 rounded-full bg-white transition-transform data-[state=checked]:translate-x-4 data-[state=unchecked]:translate-x-0.5 shadow-sm" />
        </Switch.Root>
        <span
          className={`text-xs font-medium ${isExpression ? 'text-foreground' : 'text-muted-foreground'}`}
        >
          Ausdruck
        </span>
      </div>

      {/* Content */}
      {isExpression ? (
        <ExpressionEditor fields={fields} />
      ) : (
        <div>
          {conditions?.simple ? (
            <ConditionGroup
              condition={conditions.simple}
              onChange={handleSimpleChange}
              fields={fields}
              depth={0}
            />
          ) : (
            <div className="space-y-3">
              <p className="text-sm text-muted-foreground">
                Keine Bedingung konfiguriert. Die Automatisierung wird immer ausgefuehrt.
              </p>
              <button
                onClick={() =>
                  handleSimpleChange({
                    and: [{ field: '', operator: '', value: '' }],
                  })
                }
                className="flex items-center gap-1 rounded-md border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
              >
                <Plus className="h-3 w-3" />
                Bedingung hinzufuegen
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

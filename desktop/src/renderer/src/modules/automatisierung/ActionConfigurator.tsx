/**
 * Action configurator for the automation wizard.
 *
 * Ordered list of actions with dynamic parameter forms, template variable
 * insertion, and error handling toggle. Uses useActionDefinitions() hook.
 */
import { useState } from 'react'
import {
  Plus,
  Trash2,
  ChevronUp,
  ChevronDown,
  ArrowDown,
  AlertTriangle,
  Variable,
} from 'lucide-react'
import * as Switch from '@radix-ui/react-switch'
import { useActionDefinitions, useTriggerDefinitions } from '@/api/hooks/useAutomation'
import { useAutomatisierungStore } from '@/stores/automatisierung'
import type {
  ActionConfig,
  ActionDefinition,
  ConfigParam,
} from '@/api/automation-types'

// ---------------------------------------------------------------------------
// Module grouping for action selector
// ---------------------------------------------------------------------------

const MODULE_LABELS: Record<string, string> = {
  crm: 'CRM',
  work: 'Projekte',
  email: 'E-Mail',
  notification: 'Benachrichtigungen',
  calendar: 'Kalender',
  finance: 'Finanzen',
}

// ---------------------------------------------------------------------------
// Action slot
// ---------------------------------------------------------------------------

function ActionSlot({
  action,
  index,
  total,
  actionDefs,
  variableNames,
  onUpdate,
  onRemove,
  onMoveUp,
  onMoveDown,
}: {
  action: ActionConfig
  index: number
  total: number
  actionDefs: ActionDefinition[]
  variableNames: string[]
  onUpdate: (action: ActionConfig) => void
  onRemove: () => void
  onMoveUp: () => void
  onMoveDown: () => void
}) {
  const [showVars, setShowVars] = useState(false)
  const [focusedInput, setFocusedInput] = useState<string | null>(null)

  const actionDef = actionDefs.find((d) => d.type === action.type)

  // Group actions by module for selector
  const grouped: Record<string, ActionDefinition[]> = {}
  for (const def of actionDefs) {
    const mod = def.module || 'other'
    if (!grouped[mod]) grouped[mod] = []
    grouped[mod].push(def)
  }

  const handleTypeChange = (type: string) => {
    onUpdate({ ...action, type, config: {} })
  }

  const handleConfigChange = (key: string, value: unknown) => {
    onUpdate({
      ...action,
      config: { ...action.config, [key]: value },
    })
  }

  const handleErrorToggle = (continueOnError: boolean) => {
    onUpdate({
      ...action,
      on_error: continueOnError ? 'continue' : 'abort',
    })
  }

  const insertVariable = (varName: string) => {
    if (focusedInput) {
      handleConfigChange(
        focusedInput,
        `${action.config[focusedInput] ?? ''}{{${varName}}}`,
      )
    }
  }

  return (
    <div className="rounded-lg border border-border overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between bg-secondary/30 px-3 py-2">
        <div className="flex items-center gap-2">
          <span className="flex items-center justify-center h-6 w-6 rounded-full bg-green-500 text-white text-[11px] font-bold">
            {index + 1}
          </span>
          <span className="text-xs font-medium text-foreground">
            {actionDef?.name ?? (action.type || 'Aktion wählen')}
          </span>
        </div>
        <div className="flex items-center gap-1">
          <button
            onClick={onMoveUp}
            disabled={index === 0}
            className="rounded p-0.5 text-muted-foreground hover:text-foreground disabled:opacity-30 transition-colors"
          >
            <ChevronUp className="h-3.5 w-3.5" />
          </button>
          <button
            onClick={onMoveDown}
            disabled={index === total - 1}
            className="rounded p-0.5 text-muted-foreground hover:text-foreground disabled:opacity-30 transition-colors"
          >
            <ChevronDown className="h-3.5 w-3.5" />
          </button>
          <button
            onClick={onRemove}
            className="rounded p-0.5 text-muted-foreground hover:text-destructive transition-colors"
          >
            <Trash2 className="h-3.5 w-3.5" />
          </button>
        </div>
      </div>

      <div className="p-3 space-y-3">
        {/* Action type selector */}
        <div>
          <label className="text-[11px] font-medium text-muted-foreground">
            Aktionstyp
          </label>
          <select
            value={action.type}
            onChange={(e) => handleTypeChange(e.target.value)}
            className="mt-1 w-full rounded-md border border-border bg-background px-2 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-focus-ring"
          >
            <option value="">Aktion wählen...</option>
            {Object.entries(grouped).map(([mod, defs]) => (
              <optgroup key={mod} label={MODULE_LABELS[mod] ?? mod}>
                {defs.map((def) => (
                  <option key={def.type} value={def.type}>
                    {def.name}
                  </option>
                ))}
              </optgroup>
            ))}
          </select>
        </div>

        {/* Dynamic parameter form */}
        {actionDef && actionDef.params.length > 0 && (
          <div className="space-y-2">
            {actionDef.params.map((param) => (
              <ParamInput
                key={param.key}
                param={param}
                value={action.config[param.key]}
                onChange={(val) => handleConfigChange(param.key, val)}
                onFocus={() => setFocusedInput(param.key)}
              />
            ))}
          </div>
        )}

        {/* Template variable helper */}
        {variableNames.length > 0 && (
          <div>
            <button
              onClick={() => setShowVars(!showVars)}
              className="flex items-center gap-1 text-[11px] text-muted-foreground hover:text-foreground transition-colors"
            >
              <Variable className="h-3 w-3" />
              {showVars ? 'Variablen ausblenden' : 'Verfügbare Variablen'}
            </button>
            {showVars && (
              <div className="mt-1 flex flex-wrap gap-1">
                {variableNames.map((v) => (
                  <button
                    key={v}
                    onClick={() => insertVariable(v)}
                    className="rounded bg-secondary px-1.5 py-0.5 text-[10px] font-mono text-foreground hover:bg-primary hover:text-primary-foreground transition-colors"
                    title={`{{${v}}} einfügen`}
                  >
                    {`{{${v}}}`}
                  </button>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Error handling */}
        <div className="flex items-center justify-between pt-1 border-t border-border">
          <span className="flex items-center gap-1 text-[11px] text-muted-foreground">
            <AlertTriangle className="h-3 w-3" />
            Bei Fehler fortfahren
          </span>
          <Switch.Root
            checked={action.on_error === 'continue'}
            onCheckedChange={handleErrorToggle}
            className="relative inline-flex h-4 w-7 items-center rounded-full transition-colors data-[state=checked]:bg-primary data-[state=unchecked]:bg-secondary"
          >
            <Switch.Thumb className="block h-3 w-3 rounded-full bg-white transition-transform data-[state=checked]:translate-x-3.5 data-[state=unchecked]:translate-x-0.5 shadow-sm" />
          </Switch.Root>
        </div>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Dynamic parameter input
// ---------------------------------------------------------------------------

function ParamInput({
  param,
  value,
  onChange,
  onFocus,
}: {
  param: ConfigParam
  value: unknown
  onChange: (value: unknown) => void
  onFocus: () => void
}) {
  const strValue = value != null ? String(value) : ''

  return (
    <div>
      <label className="text-[11px] font-medium text-muted-foreground">
        {param.label}
        {param.required && <span className="text-destructive ml-0.5">*</span>}
      </label>
      {param.type === 'boolean' ? (
        <div className="mt-1">
          <Switch.Root
            checked={value === true || value === 'true'}
            onCheckedChange={(checked) => onChange(checked)}
            className="relative inline-flex h-5 w-9 items-center rounded-full transition-colors data-[state=checked]:bg-primary data-[state=unchecked]:bg-secondary"
          >
            <Switch.Thumb className="block h-4 w-4 rounded-full bg-white transition-transform data-[state=checked]:translate-x-4 data-[state=unchecked]:translate-x-0.5 shadow-sm" />
          </Switch.Root>
        </div>
      ) : param.type === 'number' ? (
        <input
          type="number"
          value={strValue}
          onChange={(e) => onChange(Number(e.target.value))}
          onFocus={onFocus}
          placeholder={param.default_value != null ? String(param.default_value) : ''}
          className="mt-1 w-full rounded-md border border-border bg-background px-2 py-1.5 text-xs text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-1 focus:ring-focus-ring"
        />
      ) : (
        <input
          type="text"
          value={strValue}
          onChange={(e) => onChange(e.target.value)}
          onFocus={onFocus}
          placeholder={param.default_value != null ? String(param.default_value) : ''}
          className="mt-1 w-full rounded-md border border-border bg-background px-2 py-1.5 text-xs text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-1 focus:ring-focus-ring"
        />
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export function ActionConfigurator() {
  const {
    draftWorkflow,
    addDraftAction,
    removeDraftAction,
    updateDraftAction,
  } = useAutomatisierungStore()
  const { data: actionData } = useActionDefinitions()
  const { data: triggerData } = useTriggerDefinitions()

  const actionDefs = actionData?.actions ?? []
  const actions = (draftWorkflow?.actions ?? []) as ActionConfig[]
  const maxSteps = draftWorkflow?.max_steps ?? 10

  // Build available variable names from trigger fields + previous action outputs
  const triggerDef = (triggerData?.triggers ?? []).find(
    (t) => t.type === draftWorkflow?.trigger_type,
  )
  const triggerVars = (triggerDef?.fields ?? []).map((f) => f.key)

  const getVariablesForStep = (stepIndex: number): string[] => {
    const vars = [...triggerVars]
    for (let i = 0; i < stepIndex; i++) {
      const prevDef = actionDefs.find((d) => d.type === actions[i]?.type)
      if (prevDef) {
        for (const out of prevDef.output_fields) {
          vars.push(`step_${i + 1}_${out.key}`)
          if (i === stepIndex - 1) {
            vars.push(`prev_${out.key}`)
          }
        }
      }
    }
    return vars
  }

  const handleAdd = () => {
    addDraftAction({ type: '', config: {}, on_error: 'abort' })
  }

  const handleMoveUp = (index: number) => {
    if (index === 0) return
    const newActions = [...actions]
    ;[newActions[index - 1], newActions[index]] = [
      newActions[index],
      newActions[index - 1],
    ]
    // Rebuild actions array
    for (let i = 0; i < newActions.length; i++) {
      updateDraftAction(i, newActions[i])
    }
  }

  const handleMoveDown = (index: number) => {
    if (index >= actions.length - 1) return
    const newActions = [...actions]
    ;[newActions[index], newActions[index + 1]] = [
      newActions[index + 1],
      newActions[index],
    ]
    for (let i = 0; i < newActions.length; i++) {
      updateDraftAction(i, newActions[i])
    }
  }

  return (
    <div className="space-y-4">
      {/* Step count */}
      <div className="flex items-center justify-between">
        <span className="text-xs text-muted-foreground">
          {actions.length} von {maxSteps} Aktionen
        </span>
      </div>

      {/* Action list with chain indicators */}
      <div className="space-y-2">
        {actions.map((action, idx) => (
          <div key={idx}>
            <ActionSlot
              action={action}
              index={idx}
              total={actions.length}
              actionDefs={actionDefs}
              variableNames={getVariablesForStep(idx)}
              onUpdate={(a) => updateDraftAction(idx, a)}
              onRemove={() => removeDraftAction(idx)}
              onMoveUp={() => handleMoveUp(idx)}
              onMoveDown={() => handleMoveDown(idx)}
            />
            {/* Chain arrow */}
            {idx < actions.length - 1 && (
              <div className="flex justify-center py-1">
                <ArrowDown className="h-4 w-4 text-muted-foreground" />
              </div>
            )}
          </div>
        ))}
      </div>

      {/* Add button */}
      <button
        onClick={handleAdd}
        disabled={actions.length >= maxSteps}
        className="flex items-center gap-1.5 rounded-md border border-dashed border-border px-4 py-2 text-xs text-muted-foreground hover:text-foreground hover:border-primary transition-colors disabled:opacity-50 disabled:cursor-not-allowed w-full justify-center"
      >
        <Plus className="h-3.5 w-3.5" />
        Aktion hinzufügen
      </button>

      {actions.length === 0 && (
        <p className="text-sm text-muted-foreground text-center py-4">
          Fuegen Sie mindestens eine Aktion hinzu, die ausgefuehrt werden soll.
        </p>
      )}
    </div>
  )
}

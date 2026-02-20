/**
 * 4-step automation creation wizard.
 *
 * Steps: 1. Trigger, 2. Bedingung, 3. Aktion(en), 4. Uebersicht
 *
 * Uses Zustand store for draft workflow state (shared with react-flow editor).
 * On submit calls useCreateAutomation() mutation.
 */
import { useCallback } from 'react'
import { X, ChevronLeft, ChevronRight, Workflow } from 'lucide-react'
import { useAutomatisierungStore } from '@/stores/automatisierung'
import { useCreateAutomation, useUpdateAutomation } from '@/api/hooks/useAutomation'
import type { ConditionConfig, ActionConfig } from '@/api/automation-types'
import { TriggerSelector } from './TriggerSelector'
import { ConditionBuilder } from './ConditionBuilder'
import { ActionConfigurator } from './ActionConfigurator'

// ---------------------------------------------------------------------------
// Step labels
// ---------------------------------------------------------------------------

const STEPS = [
  { label: 'Trigger', description: 'Wann soll die Automatisierung ausgeloest werden?' },
  { label: 'Bedingung', description: 'Unter welchen Bedingungen soll sie laufen?' },
  { label: 'Aktion(en)', description: 'Was soll passieren?' },
  { label: 'Uebersicht', description: 'Zusammenfassung pruefen und speichern' },
]

// ---------------------------------------------------------------------------
// Step indicator
// ---------------------------------------------------------------------------

function StepIndicator({
  currentStep,
  onStepClick,
}: {
  currentStep: number
  onStepClick: (step: number) => void
}) {
  return (
    <div className="flex items-center gap-2 px-6 py-4 border-b border-border">
      {STEPS.map((step, idx) => (
        <div key={idx} className="flex items-center gap-2">
          <button
            onClick={() => idx < currentStep && onStepClick(idx)}
            disabled={idx > currentStep}
            className={`flex items-center gap-2 rounded-full px-3 py-1 text-xs font-medium transition-colors ${
              idx === currentStep
                ? 'bg-primary text-primary-foreground'
                : idx < currentStep
                  ? 'bg-secondary text-foreground cursor-pointer hover:bg-secondary/80'
                  : 'bg-secondary/50 text-muted-foreground cursor-not-allowed'
            }`}
          >
            <span className="flex items-center justify-center h-5 w-5 rounded-full bg-background/20 text-[10px] font-bold">
              {idx + 1}
            </span>
            {step.label}
          </button>
          {idx < STEPS.length - 1 && (
            <div className="h-px w-8 bg-border" />
          )}
        </div>
      ))}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Review step
// ---------------------------------------------------------------------------

function ReviewStep() {
  const { draftWorkflow } = useAutomatisierungStore()

  if (!draftWorkflow) return null

  const conditions = draftWorkflow.conditions as ConditionConfig | undefined
  const actions = (draftWorkflow.actions ?? []) as ActionConfig[]

  return (
    <div className="space-y-6">
      {/* Name & Description */}
      <div className="space-y-3">
        <div>
          <label className="text-xs font-medium text-foreground">Name</label>
          <NameInput />
        </div>
        <div>
          <label className="text-xs font-medium text-foreground">Beschreibung</label>
          <DescriptionInput />
        </div>
        <div>
          <label className="text-xs font-medium text-foreground">Bereich</label>
          <ScopeSelector />
        </div>
      </div>

      {/* Summary cards */}
      <div className="space-y-3">
        <div className="rounded-lg border border-border p-4">
          <h4 className="text-xs font-medium text-muted-foreground mb-1">Trigger</h4>
          <p className="text-sm text-foreground">
            {draftWorkflow.trigger_type || 'Kein Trigger ausgewaehlt'}
          </p>
        </div>

        <div className="rounded-lg border border-border p-4">
          <h4 className="text-xs font-medium text-muted-foreground mb-1">Bedingung</h4>
          <p className="text-sm text-foreground">
            {conditions?.mode === 'expression'
              ? conditions.expression || 'Kein Ausdruck'
              : conditions?.simple
                ? 'Einfache Bedingung konfiguriert'
                : 'Keine Bedingung (immer ausfuehren)'}
          </p>
        </div>

        <div className="rounded-lg border border-border p-4">
          <h4 className="text-xs font-medium text-muted-foreground mb-1">
            Aktionen ({actions.length})
          </h4>
          {actions.length === 0 ? (
            <p className="text-sm text-muted-foreground">Keine Aktionen konfiguriert</p>
          ) : (
            <ul className="space-y-1">
              {actions.map((action, idx) => (
                <li key={idx} className="text-sm text-foreground flex items-center gap-2">
                  <span className="flex items-center justify-center h-5 w-5 rounded-full bg-secondary text-[10px] font-bold">
                    {idx + 1}
                  </span>
                  {action.type}
                  <span className="text-xs text-muted-foreground">
                    ({action.on_error === 'continue' ? 'Fortfahren bei Fehler' : 'Abbrechen bei Fehler'})
                  </span>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  )
}

function NameInput() {
  const { draftWorkflow, setDraftWorkflow } = useAutomatisierungStore()
  return (
    <input
      type="text"
      value={draftWorkflow?.name ?? ''}
      onChange={(e) =>
        setDraftWorkflow({ ...draftWorkflow, name: e.target.value })
      }
      placeholder="z.B. Deal gewonnen -> Rechnung erstellen"
      className="mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
    />
  )
}

function DescriptionInput() {
  const { draftWorkflow, setDraftWorkflow } = useAutomatisierungStore()
  return (
    <textarea
      value={draftWorkflow?.description ?? ''}
      onChange={(e) =>
        setDraftWorkflow({ ...draftWorkflow, description: e.target.value })
      }
      placeholder="Optionale Beschreibung..."
      rows={2}
      className="mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring resize-none"
    />
  )
}

function ScopeSelector() {
  const { draftWorkflow, setDraftWorkflow } = useAutomatisierungStore()
  const scope = draftWorkflow?.scope ?? 'personal'

  return (
    <div className="mt-1 flex gap-2">
      {(['personal', 'team', 'organization'] as const).map((s) => (
        <button
          key={s}
          onClick={() => setDraftWorkflow({ ...draftWorkflow, scope: s })}
          className={`rounded-md px-3 py-1.5 text-xs font-medium transition-colors ${
            scope === s
              ? 'bg-primary text-primary-foreground'
              : 'bg-secondary text-foreground hover:bg-secondary/80'
          }`}
        >
          {s === 'personal'
            ? 'Persoenlich'
            : s === 'team'
              ? 'Team'
              : 'Organisation'}
        </button>
      ))}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Main wizard
// ---------------------------------------------------------------------------

export function AutomationWizard({ onClose }: { onClose: () => void }) {
  const {
    wizardStep,
    setWizardStep,
    nextStep,
    prevStep,
    draftWorkflow,
    setEditorMode,
  } = useAutomatisierungStore()

  const createMutation = useCreateAutomation()
  const updateMutation = useUpdateAutomation()
  const isEditing = !!draftWorkflow?.id

  const canAdvance = useCallback(() => {
    if (wizardStep === 0) {
      return !!draftWorkflow?.trigger_type
    }
    // Steps 1 and 2 are optional (conditions + actions)
    return true
  }, [wizardStep, draftWorkflow])

  const handleSubmit = () => {
    if (!draftWorkflow?.name || !draftWorkflow?.trigger_type) return

    const payload = {
      name: draftWorkflow.name,
      description: draftWorkflow.description,
      scope: draftWorkflow.scope,
      trigger_type: draftWorkflow.trigger_type,
      trigger_config: draftWorkflow.trigger_config,
      conditions: draftWorkflow.conditions,
      actions: draftWorkflow.actions,
      max_steps: draftWorkflow.max_steps ?? 10,
    }

    if (isEditing && draftWorkflow.id) {
      updateMutation.mutate(
        { id: draftWorkflow.id, ...payload },
        { onSuccess: () => onClose() },
      )
    } else {
      createMutation.mutate(payload, { onSuccess: () => onClose() })
    }
  }

  const handleSwitchToEditor = () => {
    setEditorMode('editor')
  }

  const isPending = createMutation.isPending || updateMutation.isPending

  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-background">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-border px-6 py-3">
        <h2 className="text-lg font-semibold text-foreground">
          {isEditing ? 'Automatisierung bearbeiten' : 'Neue Automatisierung'}
        </h2>
        <div className="flex items-center gap-3">
          <button
            onClick={handleSwitchToEditor}
            className="flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
          >
            <Workflow className="h-3.5 w-3.5" />
            Zum Editor wechseln
          </button>
          <button
            onClick={onClose}
            className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
          >
            <X className="h-5 w-5" />
          </button>
        </div>
      </div>

      {/* Step indicator */}
      <StepIndicator currentStep={wizardStep} onStepClick={setWizardStep} />

      {/* Step description */}
      <div className="px-6 py-3 border-b border-border">
        <p className="text-sm text-muted-foreground">{STEPS[wizardStep].description}</p>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto px-6 py-4">
        {wizardStep === 0 && <TriggerSelector />}
        {wizardStep === 1 && <ConditionBuilder />}
        {wizardStep === 2 && <ActionConfigurator />}
        {wizardStep === 3 && <ReviewStep />}
      </div>

      {/* Footer navigation */}
      <div className="flex items-center justify-between border-t border-border px-6 py-3">
        <button
          onClick={prevStep}
          disabled={wizardStep === 0}
          className="flex items-center gap-1 rounded-md border border-border px-4 py-2 text-xs font-medium text-foreground hover:bg-secondary transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <ChevronLeft className="h-3.5 w-3.5" />
          Zurueck
        </button>

        {wizardStep < 3 ? (
          <button
            onClick={nextStep}
            disabled={!canAdvance()}
            className="flex items-center gap-1 rounded-md bg-primary px-4 py-2 text-xs font-medium text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Weiter
            <ChevronRight className="h-3.5 w-3.5" />
          </button>
        ) : (
          <button
            onClick={handleSubmit}
            disabled={
              isPending || !draftWorkflow?.name || !draftWorkflow?.trigger_type
            }
            className="rounded-md bg-primary px-4 py-2 text-xs font-medium text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isPending
              ? 'Speichern...'
              : isEditing
                ? 'Automatisierung aktualisieren'
                : 'Automatisierung erstellen'}
          </button>
        )}
      </div>
    </div>
  )
}

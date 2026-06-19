/**
 * 4-step automation creation wizard.
 *
 * Steps: 1. Trigger, 2. Bedingung, 3. Aktion(en), 4. Übersicht
 *
 * Uses Zustand store for draft workflow state (shared with react-flow editor).
 * On submit calls useCreateAutomation() mutation.
 */
import { useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import {
  X,
  ChevronLeft,
  ChevronRight,
  Workflow,
  Play,
  CheckCircle,
  XCircle,
  Loader2,
} from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { useAutomatisierungStore } from '@/stores/automatisierung'
import { useCreateAutomation, useUpdateAutomation, useDryRunAutomation } from '@/api/hooks/useAutomation'
import type { ConditionConfig, ActionConfig, DryRunResponse } from '@/api/automation-types'
import { TriggerSelector } from './TriggerSelector'
import { ConditionBuilder } from './ConditionBuilder'
import { ActionConfigurator } from './ActionConfigurator'

// ---------------------------------------------------------------------------
// Step labels
// ---------------------------------------------------------------------------

const STEP_KEYS = [
  { labelKey: 'automatisierung.wizard.steps.trigger', descKey: 'automatisierung.wizard.steps.triggerDesc' },
  { labelKey: 'automatisierung.wizard.steps.condition', descKey: 'automatisierung.wizard.steps.conditionDesc' },
  { labelKey: 'automatisierung.wizard.steps.actions', descKey: 'automatisierung.wizard.steps.actionsDesc' },
  { labelKey: 'automatisierung.wizard.steps.review', descKey: 'automatisierung.wizard.steps.reviewDesc' },
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
  const { t } = useTranslation()
  return (
    <div className="flex items-center gap-2 px-6 py-4 border-b border-border">
      {STEP_KEYS.map((step, idx) => (
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
            {t(step.labelKey)}
          </button>
          {idx < STEP_KEYS.length - 1 && (
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
  const { t } = useTranslation()
  const { draftWorkflow } = useAutomatisierungStore()
  const dryRunMutation = useDryRunAutomation()

  if (!draftWorkflow) return null

  const conditions = draftWorkflow.conditions as ConditionConfig | undefined
  const actions = (draftWorkflow.actions ?? []) as ActionConfig[]

  const handleDryRun = () => {
    dryRunMutation.mutate({
      automation_id: draftWorkflow.id ?? '',
      sample_env: {},
      actions: actions.map((a) => ({ type: a.type })),
    })
  }

  const dryResult = dryRunMutation.data as DryRunResponse | undefined

  return (
    <div className="space-y-6">
      {/* Name & Description */}
      <div className="space-y-3">
        <div>
          <label className="text-xs font-medium text-foreground">{t('automatisierung.review.name')}</label>
          <NameInput />
        </div>
        <div>
          <label className="text-xs font-medium text-foreground">{t('automatisierung.review.description')}</label>
          <DescriptionInput />
        </div>
        <div>
          <label className="text-xs font-medium text-foreground">{t('automatisierung.review.scope')}</label>
          <ScopeSelector />
        </div>
      </div>

      {/* Summary cards */}
      <div className="space-y-3">
        <div className="rounded-lg border border-border p-4">
          <h4 className="text-xs font-medium text-muted-foreground mb-1">{t('automatisierung.review.trigger')}</h4>
          <p className="text-sm text-foreground">
            {draftWorkflow.trigger_type || t('automatisierung.review.noTrigger')}
          </p>
        </div>

        <div className="rounded-lg border border-border p-4">
          <h4 className="text-xs font-medium text-muted-foreground mb-1">{t('automatisierung.review.condition')}</h4>
          <p className="text-sm text-foreground">
            {conditions?.mode === 'expression'
              ? conditions.expression || t('automatisierung.review.noExpression')
              : conditions?.simple
                ? t('automatisierung.review.simpleConfigured')
                : t('automatisierung.review.noCondition')}
          </p>
        </div>

        <div className="rounded-lg border border-border p-4">
          <h4 className="text-xs font-medium text-muted-foreground mb-1">
            {t('automatisierung.review.actionsCount', { count: actions.length })}
          </h4>
          {actions.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t('automatisierung.review.noActions')}</p>
          ) : (
            <ul className="space-y-1">
              {actions.map((action, idx) => (
                <li key={idx} className="text-sm text-foreground flex items-center gap-2">
                  <span className="flex items-center justify-center h-5 w-5 rounded-full bg-secondary text-[10px] font-bold">
                    {idx + 1}
                  </span>
                  {action.type}
                  <span className="text-xs text-muted-foreground">
                    ({action.on_error === 'continue' ? t('automatisierung.review.continueOnError') : t('automatisierung.review.abortOnError')})
                  </span>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>

      {/* DryRun simulation */}
      <div className="rounded-lg border border-border p-4 space-y-3">
        <div className="flex items-center justify-between">
          <div>
            <h4 className="text-xs font-medium text-foreground">{t('automatisierung.simulation.title')}</h4>
            <p className="text-[11px] text-muted-foreground">
              {t('automatisierung.simulation.description')}
            </p>
          </div>
          <button
            onClick={handleDryRun}
            disabled={dryRunMutation.isPending}
            className="flex items-center gap-1.5 rounded-md bg-secondary px-3 py-1.5 text-xs font-medium text-foreground hover:bg-secondary/80 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {dryRunMutation.isPending ? (
              <Loader2 className="h-3 w-3 animate-spin" />
            ) : (
              <Play className="h-3 w-3" />
            )}
            {dryRunMutation.isPending ? t('automatisierung.simulation.running') : t('automatisierung.simulation.start')}
          </button>
        </div>

        {dryResult && (
          <div className="space-y-2 pt-1">
            {/* Condition result */}
            <div className="flex items-center gap-2 text-xs">
              <span className="text-muted-foreground">{t('automatisierung.simulation.condition')}:</span>
              {dryResult.condition_result ? (
                <span className="flex items-center gap-1 text-green-600">
                  <CheckCircle className="h-3.5 w-3.5" />
                  {t('automatisierung.simulation.matches')}
                </span>
              ) : (
                <span className="flex items-center gap-1 text-destructive">
                  <XCircle className="h-3.5 w-3.5" />
                  {t('automatisierung.simulation.noMatch')}
                </span>
              )}
            </div>

            {/* Would execute */}
            <div className="flex items-center gap-2 text-xs">
              <span className="text-muted-foreground">{t('automatisierung.simulation.execution')}:</span>
              <span className={dryResult.would_execute ? 'text-green-600' : 'text-muted-foreground'}>
                {dryResult.would_execute
                  ? t('automatisierung.simulation.wouldExecute')
                  : t('automatisierung.simulation.wouldNotExecute')}
              </span>
            </div>

            {/* Simulated steps */}
            {dryResult.steps.length > 0 && (
              <div className="mt-2">
                <p className="text-xs font-medium text-muted-foreground mb-1.5">
                  {t('automatisierung.simulation.simulatedSteps', { count: dryResult.steps.length })}
                </p>
                <div className="space-y-1.5">
                  {dryResult.steps.map((step, idx) => (
                    <div
                      key={idx}
                      className="flex items-start gap-2 rounded-md bg-secondary/30 px-2.5 py-2 text-xs"
                    >
                      <span className="flex shrink-0 items-center justify-center h-5 w-5 rounded-full bg-secondary text-[10px] font-bold">
                        {idx + 1}
                      </span>
                      <div className="flex-1 min-w-0">
                        <p className="font-medium text-foreground">{step.action_type}</p>
                        {step.error ? (
                          <p className="text-destructive mt-0.5">{step.error}</p>
                        ) : (
                          <p className="text-muted-foreground mt-0.5">
                            {step.duration_ms}ms
                          </p>
                        )}
                      </div>
                      {step.error ? (
                        <XCircle className="h-3.5 w-3.5 text-destructive shrink-0 mt-0.5" />
                      ) : (
                        <CheckCircle className="h-3.5 w-3.5 text-green-500 shrink-0 mt-0.5" />
                      )}
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}

        {dryRunMutation.isError && (
          <div className="flex items-center gap-2 text-xs text-destructive">
            <XCircle className="h-3.5 w-3.5" />
            {t('automatisierung.simulation.failed')}: {(dryRunMutation.error as Error)?.message ?? t('automatisierung.simulation.unknownError')}
          </div>
        )}
      </div>
    </div>
  )
}

function NameInput() {
  const { t } = useTranslation()
  const { draftWorkflow, setDraftWorkflow } = useAutomatisierungStore()
  return (
    <input
      type="text"
      value={draftWorkflow?.name ?? ''}
      onChange={(e) =>
        setDraftWorkflow({ ...draftWorkflow, name: e.target.value })
      }
      placeholder={t('automatisierung.wizard.namePlaceholder')}
      className="mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
    />
  )
}

function DescriptionInput() {
  const { t } = useTranslation()
  const { draftWorkflow, setDraftWorkflow } = useAutomatisierungStore()
  return (
    <textarea
      value={draftWorkflow?.description ?? ''}
      onChange={(e) =>
        setDraftWorkflow({ ...draftWorkflow, description: e.target.value })
      }
      placeholder={t('automatisierung.wizard.descriptionPlaceholder')}
      rows={2}
      className="mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring resize-none"
    />
  )
}

function ScopeSelector() {
  const { t } = useTranslation()
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
            ? t('automatisierung.scope.personal')
            : s === 'team'
              ? t('automatisierung.scope.team')
              : t('automatisierung.scope.organization')}
        </button>
      ))}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Main wizard
// ---------------------------------------------------------------------------

export function AutomationWizard({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation()
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
    <Dialog open onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent className="gap-0 p-0 max-w-full w-screen h-screen rounded-none border-0 [&>button:last-child]:hidden">
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-border px-6 py-3">
        <DialogTitle className="text-lg font-semibold text-foreground">
          {isEditing ? t('automatisierung.wizard.editTitle') : t('automatisierung.wizard.newTitle')}
        </DialogTitle>
        <DialogDescription className="sr-only">
          {t('automatisierung.wizard.description')}
        </DialogDescription>
        <div className="flex items-center gap-3">
          <button
            onClick={handleSwitchToEditor}
            className="flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
          >
            <Workflow className="h-3.5 w-3.5" />
            {t('automatisierung.wizard.switchToEditor')}
          </button>
          <button
            onClick={onClose}
            className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
            aria-label={t('common.close')}
          >
            <X className="h-5 w-5" />
          </button>
        </div>
      </div>

      {/* Step indicator */}
      <StepIndicator currentStep={wizardStep} onStepClick={setWizardStep} />

      {/* Step description */}
      <div className="px-6 py-3 border-b border-border">
        <p className="text-sm text-muted-foreground">{t(STEP_KEYS[wizardStep].descKey)}</p>
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
          {t('common.back')}
        </button>

        {wizardStep < 3 ? (
          <button
            onClick={nextStep}
            disabled={!canAdvance()}
            className="flex items-center gap-1 rounded-md bg-primary px-4 py-2 text-xs font-medium text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {t('common.next')}
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
              ? t('automatisierung.saving')
              : isEditing
                ? t('automatisierung.wizard.update')
                : t('automatisierung.wizard.create')}
          </button>
        )}
      </div>
    </div>
      </DialogContent>
    </Dialog>
  )
}

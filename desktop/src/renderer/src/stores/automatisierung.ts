/**
 * Automation UI state store (local only).
 *
 * All server data is managed by TanStack Query hooks in api/hooks/useAutomation.ts.
 * This store only holds ephemeral UI state: wizard steps, editor mode, draft workflow.
 *
 * The draft workflow is the SINGLE SOURCE OF TRUTH for both wizard and editor views.
 */
import { create } from 'zustand'
import type {
  Automation,
  ConditionConfig,
  ActionConfig,
} from '@/api/automation-types'

// ---------------------------------------------------------------------------
// Store interface
// ---------------------------------------------------------------------------

export type EditorMode = 'wizard' | 'editor'

export type TemplateGrouping = 'module' | 'complexity'

interface ExecutionLogFilter {
  status?: string
  dateRange?: [Date, Date]
}

interface AutomatisierungUIState {
  // Wizard state
  wizardStep: number
  setWizardStep: (step: number) => void
  nextStep: () => void
  prevStep: () => void

  // Editor mode toggle
  editorMode: EditorMode
  setEditorMode: (mode: EditorMode) => void

  // Draft workflow -- single source of truth for wizard and editor
  draftWorkflow: Partial<Automation> | null
  setDraftWorkflow: (draft: Partial<Automation> | null) => void
  updateDraftTrigger: (triggerType: string, triggerConfig: Record<string, unknown>) => void
  updateDraftConditions: (conditions: ConditionConfig) => void
  addDraftAction: (action: ActionConfig) => void
  removeDraftAction: (index: number) => void
  updateDraftAction: (index: number, action: ActionConfig) => void

  // Selection
  selectedAutomationId: string | null
  selectAutomation: (id: string | null) => void
  clearSelection: () => void

  // Template gallery
  templateGrouping: TemplateGrouping
  setTemplateGrouping: (grouping: TemplateGrouping) => void

  // Execution log filter
  executionLogFilter: ExecutionLogFilter
  setExecutionLogFilter: (filter: ExecutionLogFilter) => void

  // Reset / load helpers
  resetDraft: () => void
  loadAutomationIntoDraft: (automation: Automation) => void
}

// ---------------------------------------------------------------------------
// Default draft state
// ---------------------------------------------------------------------------

const defaultConditions: ConditionConfig = {
  mode: 'simple',
  simple: null,
  expression: '',
}

function createEmptyDraft(): Partial<Automation> {
  return {
    name: '',
    description: '',
    scope: 'personal',
    trigger_type: '',
    trigger_config: {},
    conditions: { ...defaultConditions },
    actions: [],
    max_steps: 10,
  }
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useAutomatisierungStore = create<AutomatisierungUIState>()((set) => ({
  // Wizard state
  wizardStep: 0,
  setWizardStep: (step) => set({ wizardStep: step }),
  nextStep: () => set((s) => ({ wizardStep: Math.min(s.wizardStep + 1, 3) })),
  prevStep: () => set((s) => ({ wizardStep: Math.max(s.wizardStep - 1, 0) })),

  // Editor mode
  editorMode: 'wizard',
  setEditorMode: (mode) => set({ editorMode: mode }),

  // Draft workflow
  draftWorkflow: null,
  setDraftWorkflow: (draft) => set({ draftWorkflow: draft }),

  updateDraftTrigger: (triggerType, triggerConfig) =>
    set((s) => ({
      draftWorkflow: s.draftWorkflow
        ? { ...s.draftWorkflow, trigger_type: triggerType, trigger_config: triggerConfig }
        : { trigger_type: triggerType, trigger_config: triggerConfig },
    })),

  updateDraftConditions: (conditions) =>
    set((s) => ({
      draftWorkflow: s.draftWorkflow
        ? { ...s.draftWorkflow, conditions }
        : { conditions },
    })),

  addDraftAction: (action) =>
    set((s) => {
      const current = s.draftWorkflow?.actions ?? []
      return {
        draftWorkflow: {
          ...s.draftWorkflow,
          actions: [...current, action],
        },
      }
    }),

  removeDraftAction: (index) =>
    set((s) => {
      const current = s.draftWorkflow?.actions ?? []
      return {
        draftWorkflow: {
          ...s.draftWorkflow,
          actions: current.filter((_, i) => i !== index),
        },
      }
    }),

  updateDraftAction: (index, action) =>
    set((s) => {
      const current = [...(s.draftWorkflow?.actions ?? [])]
      current[index] = action
      return {
        draftWorkflow: {
          ...s.draftWorkflow,
          actions: current,
        },
      }
    }),

  // Selection
  selectedAutomationId: null,
  selectAutomation: (id) => set({ selectedAutomationId: id }),
  clearSelection: () => set({ selectedAutomationId: null }),

  // Template gallery
  templateGrouping: 'module',
  setTemplateGrouping: (grouping) => set({ templateGrouping: grouping }),

  // Execution log filter
  executionLogFilter: {},
  setExecutionLogFilter: (filter) => set({ executionLogFilter: filter }),

  // Reset / load
  resetDraft: () =>
    set({
      wizardStep: 0,
      editorMode: 'wizard',
      draftWorkflow: createEmptyDraft(),
    }),

  loadAutomationIntoDraft: (automation) =>
    set({
      wizardStep: 0,
      draftWorkflow: { ...automation },
    }),
}))

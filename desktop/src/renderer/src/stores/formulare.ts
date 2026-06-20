/**
 * Client-only store for the Formulare module.
 *
 * Persisted state: draft field edits while the editor is open.
 * All server data (schemas, submissions, webhooks) is managed by
 * React Query via useFormulare.ts hooks.
 *
 * Actions (email/task/crm_contact) are local UI state retained in DraftSchema
 * for future Sprint 2 backend wiring — the UI panel is hidden until the backend
 * endpoint exists.
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

// Re-export from API types so the Page can use a single import source.
// These 8 types match the backend whitelist exactly.
export type FormFieldType =
  | 'text'
  | 'textarea'
  | 'email'
  | 'select'
  | 'checkbox'
  | 'radio'
  | 'date'
  | 'number'
  | 'file'

export interface FormField {
  id: string
  type: FormFieldType
  label: string
  required: boolean
  placeholder?: string
  options?: string[] // for select, radio
  conditionalLogic?: {
    fieldId: string
    operator: 'equals' | 'not_equals' | 'contains'
    value: string
  }
  page?: number
}

export interface FormAction {
  type: 'email' | 'task' | 'crm_contact'
  config: Record<string, string>
}

// ---------------------------------------------------------------------------
// Draft schema: local representation while editing in the builder.
// Mirrors FormSchema from API but includes frontend-only fields.
// ---------------------------------------------------------------------------

export interface DraftSchema {
  /** Backend ID — undefined for unsaved new forms */
  id?: string
  title: string
  description: string
  fields: FormField[]
  isPublic: boolean
  pageCount: number
  /** Frontend-only: post-submission actions (not yet persisted to backend) */
  actions: FormAction[]
}

// ---------------------------------------------------------------------------
// Store state
// ---------------------------------------------------------------------------

interface FormulareClientState {
  /** Currently open draft in the editor, null when editor is closed */
  draft: DraftSchema | null

  openDraft: (draft: DraftSchema) => void
  closeDraft: () => void
  updateDraftMeta: (updates: Partial<Pick<DraftSchema, 'title' | 'description' | 'isPublic' | 'pageCount'>>) => void

  addField: (field: Omit<FormField, 'id'>) => void
  removeField: (fieldId: string) => void
  reorderFields: (fields: FormField[]) => void
  updateField: (fieldId: string, updates: Partial<Omit<FormField, 'id'>>) => void

  addAction: (action: FormAction) => void
  updateAction: (index: number, action: FormAction) => void
  removeAction: (index: number) => void
}

let nextFieldId = 1000

export const useFormulareStore = create<FormulareClientState>()(
  persist(
    (set) => ({
      draft: null,

      openDraft: (draft) => set({ draft }),

      closeDraft: () => set({ draft: null }),

      updateDraftMeta: (updates) =>
        set((state) => ({
          draft: state.draft ? { ...state.draft, ...updates } : null,
        })),

      addField: (field) =>
        set((state) => {
          if (!state.draft) return state
          return {
            draft: {
              ...state.draft,
              fields: [
                ...state.draft.fields,
                { ...field, id: `field-${nextFieldId++}` },
              ],
            },
          }
        }),

      removeField: (fieldId) =>
        set((state) => {
          if (!state.draft) return state
          return {
            draft: {
              ...state.draft,
              fields: state.draft.fields.filter((f) => f.id !== fieldId),
            },
          }
        }),

      reorderFields: (fields) =>
        set((state) => {
          if (!state.draft) return state
          return { draft: { ...state.draft, fields } }
        }),

      updateField: (fieldId, updates) =>
        set((state) => {
          if (!state.draft) return state
          return {
            draft: {
              ...state.draft,
              fields: state.draft.fields.map((f) =>
                f.id === fieldId ? { ...f, ...updates } : f,
              ),
            },
          }
        }),

      addAction: (action) =>
        set((state) => {
          if (!state.draft) return state
          return {
            draft: {
              ...state.draft,
              actions: [...state.draft.actions, action],
            },
          }
        }),

      updateAction: (index, action) =>
        set((state) => {
          if (!state.draft) return state
          const actions = [...state.draft.actions]
          actions[index] = action
          return { draft: { ...state.draft, actions } }
        }),

      removeAction: (index) =>
        set((state) => {
          if (!state.draft) return state
          const actions = [...state.draft.actions]
          actions.splice(index, 1)
          return { draft: { ...state.draft, actions } }
        }),
    }),
    { name: 'cosmi-formulare-draft' },
  ),
)

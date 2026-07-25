/**
 * DraftConfigProvider (Modul-Editor v1) — the editor-session state layer.
 *
 * Holds the sparse in-progress draft (labels + value-sets) for one module while
 * the editor window is open, drives the live preview, and scrubs its global
 * i18n side-effects on unmount (R-1 mitigation, IST-EDITOR §Risiken).
 *
 * Overlay model (4th layer): the draft is applied ON TOP of the live tenant
 * layer. Labels must mutate the GLOBAL i18n bundle for the preview (modules read
 * t() from the shared instance); on close without commit we reset exactly the
 * touched keys back to their code default. Value-sets need no global mutation —
 * the sandbox reads them via resolveValueSet(id, false, draftValueSets).
 *
 * Persistence/commit lives in mocks/data/customization-drafts.ts; this provider
 * only produces the payload (buildPayload) that the commit dialog deploys.
 */
import { createContext, useContext, useEffect, useMemo, useReducer, useRef } from 'react'
import type { ReactElement, ReactNode } from 'react'
import type {
  CustomizationDraftPayload,
  LocaleLabelMap,
  ModuleAreasOverlay,
  ValueSet,
  ValueSetMigrations,
} from '@/api/customization-types'
import type { CustomFieldDefinition, DraftCustomFieldMap } from '@/mocks/data/custom-fields'
import { i18n } from '@/i18n/i18n'
import { restoreLabelOverlay } from '@/i18n/useLabelOverlay'
import { resolveLabelOverrides } from '@/mocks/data/customization'

type DraftValueSets = Record<string, Omit<ValueSet, 'layer'>>

interface DraftState {
  labels: LocaleLabelMap
  valueSets: DraftValueSets
  customFields: DraftCustomFieldMap
  moduleAreas: ModuleAreasOverlay
  valueSetMigrations: ValueSetMigrations
}

type DraftAction =
  | { type: 'setLabel'; locale: string; key: string; value: string }
  | { type: 'resetLabel'; locale: string; key: string }
  | { type: 'setValueSet'; id: string; set: Omit<ValueSet, 'layer'> }
  | { type: 'resetValueSet'; id: string }
  | { type: 'setEntityFields'; entity: string; fields: CustomFieldDefinition[] }
  | { type: 'resetEntityFields'; entity: string }
  | { type: 'setModuleArea'; moduleKey: string; areaKey: string; enabled: boolean }
  | { type: 'setValueSetMigration'; setId: string; fromId: string; toId: string }
  | { type: 'clearValueSetMigration'; setId: string; fromId: string }
  | { type: 'reset' }
  | { type: 'load'; payload: CustomizationDraftPayload }

function reducer(state: DraftState, action: DraftAction): DraftState {
  switch (action.type) {
    case 'setLabel': {
      const localeMap = { ...(state.labels[action.locale] ?? {}), [action.key]: action.value }
      return { ...state, labels: { ...state.labels, [action.locale]: localeMap } }
    }
    case 'resetLabel': {
      const localeMap = { ...(state.labels[action.locale] ?? {}) }
      delete localeMap[action.key]
      return { ...state, labels: { ...state.labels, [action.locale]: localeMap } }
    }
    case 'setValueSet':
      return { ...state, valueSets: { ...state.valueSets, [action.id]: action.set } }
    case 'resetValueSet': {
      const next = { ...state.valueSets }
      delete next[action.id]
      const nextMig = { ...state.valueSetMigrations }
      delete nextMig[action.id]
      return { ...state, valueSets: next, valueSetMigrations: nextMig }
    }
    case 'setEntityFields':
      return { ...state, customFields: { ...state.customFields, [action.entity]: action.fields } }
    case 'resetEntityFields': {
      const next = { ...state.customFields }
      delete next[action.entity]
      return { ...state, customFields: next }
    }
    case 'setModuleArea': {
      const moduleMap = { ...(state.moduleAreas[action.moduleKey] ?? {}), [action.areaKey]: action.enabled }
      return { ...state, moduleAreas: { ...state.moduleAreas, [action.moduleKey]: moduleMap } }
    }
    case 'setValueSetMigration': {
      const setMap = { ...(state.valueSetMigrations[action.setId] ?? {}), [action.fromId]: action.toId }
      return { ...state, valueSetMigrations: { ...state.valueSetMigrations, [action.setId]: setMap } }
    }
    case 'clearValueSetMigration': {
      const setMap = { ...(state.valueSetMigrations[action.setId] ?? {}) }
      delete setMap[action.fromId]
      const nextMig = { ...state.valueSetMigrations }
      if (Object.keys(setMap).length > 0) nextMig[action.setId] = setMap
      else delete nextMig[action.setId]
      return { ...state, valueSetMigrations: nextMig }
    }
    case 'reset':
      return { labels: {}, valueSets: {}, customFields: {}, moduleAreas: {}, valueSetMigrations: {} }
    case 'load':
      return {
        labels: structuredClone(action.payload.labels),
        valueSets: structuredClone(action.payload.valueSets),
        customFields: structuredClone(action.payload.customFields ?? {}),
        moduleAreas: structuredClone(action.payload.moduleAreas ?? {}),
        valueSetMigrations: structuredClone(action.payload.valueSetMigrations ?? {}),
      }
  }
}

function countLabels(labels: LocaleLabelMap): number {
  return Object.values(labels).reduce((acc, map) => acc + Object.keys(map).length, 0)
}

function countAreas(moduleAreas: ModuleAreasOverlay): number {
  return Object.values(moduleAreas).reduce((acc, map) => acc + Object.keys(map).length, 0)
}

export interface DraftConfigContextValue {
  moduleKey: string
  labels: LocaleLabelMap
  valueSets: DraftValueSets
  /** Per-entity desired field lists staged in the draft (E-3c). */
  customFields: DraftCustomFieldMap
  /** Module-area visibility overrides staged in the draft (R4). */
  moduleAreas: ModuleAreasOverlay
  /** Staged record migrations for deleted value-set options (R4b). */
  valueSetMigrations: ValueSetMigrations
  /** Whether the draft has any change vs. the live tenant layer. */
  isDirty: boolean
  /** Total touched entries (labels + value-sets + field entities) for the footer counter. */
  changeCount: number
  setDraftLabel: (locale: string, key: string, value: string) => void
  resetDraftLabel: (locale: string, key: string) => void
  setDraftValueSet: (id: string, set: Omit<ValueSet, 'layer'>) => void
  resetDraftValueSet: (id: string) => void
  /** Stage the full desired field list for an entity (E-3c). */
  setDraftEntityFields: (entity: string, fields: CustomFieldDefinition[]) => void
  /** Drop the staged field snapshot for an entity (back to the live baseline). */
  resetDraftEntityFields: (entity: string) => void
  /** Toggle one module sub-area (tab/section) on or off in the draft (R4). */
  setDraftModuleArea: (moduleKey: string, areaKey: string, enabled: boolean) => void
  /** Record that a deleted option's records move to another option (R4b). */
  setDraftValueSetMigration: (setId: string, fromId: string, toId: string) => void
  /** Undo a staged option deletion/migration (restore). */
  clearDraftValueSetMigration: (setId: string, fromId: string) => void
  resetAll: () => void
  /** Build the sparse payload for save/deploy. */
  buildPayload: () => CustomizationDraftPayload
  /** Load an existing draft's payload into the editor (continue a blueprint). */
  loadDraft: (payload: CustomizationDraftPayload) => void
}

const DraftConfigContext = createContext<DraftConfigContextValue | null>(null)

export interface DraftConfigProviderProps {
  moduleKey: string
  /** Initial draft payload when reopening a saved blueprint. */
  initialPayload?: CustomizationDraftPayload
  /** Locale to render the live preview in (defaults to the active app locale). */
  previewLocale?: string
  children: ReactNode
}

export function DraftConfigProvider({
  moduleKey,
  initialPayload,
  previewLocale,
  children,
}: DraftConfigProviderProps): ReactElement {
  const locale = previewLocale ?? i18n.language
  const [state, dispatch] = useReducer(
    reducer,
    initialPayload
      ? {
          labels: structuredClone(initialPayload.labels),
          valueSets: structuredClone(initialPayload.valueSets),
          customFields: structuredClone(initialPayload.customFields ?? {}),
          moduleAreas: structuredClone(initialPayload.moduleAreas ?? {}),
          valueSetMigrations: structuredClone(initialPayload.valueSetMigrations ?? {}),
        }
      : { labels: {}, valueSets: {}, customFields: {}, moduleAreas: {}, valueSetMigrations: {} },
  )

  // Every i18n key we ever wrote to the global bundle — scrubbed on unmount so a
  // draft that is discarded leaves no residue in the live app (R-1 mitigation).
  const touchedKeysRef = useRef<Set<string>>(new Set())

  // Live label preview: re-apply the merged (tenant ⊕ draft) overlay whenever the
  // draft labels change. Modules read t() from the shared instance, so the preview
  // updates via the ICU-Live-Fix. DEBOUNCED: applyLabelOverlay triggers a global
  // re-render (the whole sandbox module) — firing it per keystroke would lag
  // typing. The input reads the draft state synchronously, so it stays responsive;
  // only the preview trails by ~120ms.
  useEffect(() => {
    for (const key of Object.keys(state.labels[locale] ?? {})) touchedKeysRef.current.add(key)
    const id = setTimeout(() => {
      // Scrub every touched key to its default first, then re-apply the current
      // overlay — otherwise a key reset back to default keeps its stale override
      // in the bundle (applyLabelOverlay only writes non-default keys).
      restoreLabelOverlay(
        locale,
        Array.from(touchedKeysRef.current),
        resolveLabelOverrides(locale, false, state.labels),
      )
      // Edit-in-place: EditableText can target ANY i18n key, not only the curated
      // LABEL_WHITELIST (which resolveLabelOverrides is limited to). Apply the raw
      // draft labels directly so those inline edits preview live too. The ICU-Live-
      // Fix (bindI18nStore 'added') re-renders the module on this merge.
      i18n.addResourceBundle(locale, 'translation', state.labels[locale] ?? {}, true, true)
    }, 120)
    return () => clearTimeout(id)
  }, [state.labels, locale])

  // On unmount: scrub the touched keys back to their default, then re-apply the
  // persistent tenant/vendor overlay — so the live app is exactly as before.
  useEffect(() => {
    return () => {
      const scrub = Array.from(touchedKeysRef.current)
      restoreLabelOverlay(locale, scrub, resolveLabelOverrides(locale, false))
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [locale])

  const value = useMemo<DraftConfigContextValue>(() => {
    const changeCount =
      countLabels(state.labels) +
      Object.keys(state.valueSets).length +
      Object.keys(state.customFields).length +
      countAreas(state.moduleAreas)
    return {
      moduleKey,
      labels: state.labels,
      valueSets: state.valueSets,
      customFields: state.customFields,
      moduleAreas: state.moduleAreas,
      valueSetMigrations: state.valueSetMigrations,
      isDirty: changeCount > 0,
      changeCount,
      setDraftLabel: (l, key, val) => dispatch({ type: 'setLabel', locale: l, key, value: val }),
      resetDraftLabel: (l, key) => dispatch({ type: 'resetLabel', locale: l, key }),
      setDraftValueSet: (id, set) => dispatch({ type: 'setValueSet', id, set }),
      resetDraftValueSet: (id) => dispatch({ type: 'resetValueSet', id }),
      setDraftEntityFields: (entity, fields) => dispatch({ type: 'setEntityFields', entity, fields }),
      resetDraftEntityFields: (entity) => dispatch({ type: 'resetEntityFields', entity }),
      setDraftModuleArea: (mk, areaKey, enabled) =>
        dispatch({ type: 'setModuleArea', moduleKey: mk, areaKey, enabled }),
      setDraftValueSetMigration: (setId, fromId, toId) =>
        dispatch({ type: 'setValueSetMigration', setId, fromId, toId }),
      clearDraftValueSetMigration: (setId, fromId) =>
        dispatch({ type: 'clearValueSetMigration', setId, fromId }),
      resetAll: () => dispatch({ type: 'reset' }),
      buildPayload: () => ({
        labels: structuredClone(state.labels),
        valueSets: structuredClone(state.valueSets),
        ...(Object.keys(state.customFields).length > 0 && {
          customFields: structuredClone(state.customFields),
        }),
        ...(countAreas(state.moduleAreas) > 0 && {
          moduleAreas: structuredClone(state.moduleAreas),
        }),
        ...(Object.keys(state.valueSetMigrations).length > 0 && {
          valueSetMigrations: structuredClone(state.valueSetMigrations),
        }),
      }),
      loadDraft: (payload) => dispatch({ type: 'load', payload }),
    }
  }, [moduleKey, state])

  return <DraftConfigContext.Provider value={value}>{children}</DraftConfigContext.Provider>
}

/** Access the editor draft state. Throws outside a DraftConfigProvider. */
export function useDraftConfig(): DraftConfigContextValue {
  const ctx = useContext(DraftConfigContext)
  if (!ctx) throw new Error('useDraftConfig must be used within a DraftConfigProvider')
  return ctx
}

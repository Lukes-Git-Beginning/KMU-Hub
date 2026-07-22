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
  ValueSet,
} from '@/api/customization-types'
import { i18n } from '@/i18n/i18n'
import { applyLabelOverlay, restoreLabelOverlay } from '@/i18n/useLabelOverlay'
import { resolveLabelOverrides } from '@/mocks/data/customization'

type DraftValueSets = Record<string, Omit<ValueSet, 'layer'>>

interface DraftState {
  labels: LocaleLabelMap
  valueSets: DraftValueSets
}

type DraftAction =
  | { type: 'setLabel'; locale: string; key: string; value: string }
  | { type: 'resetLabel'; locale: string; key: string }
  | { type: 'setValueSet'; id: string; set: Omit<ValueSet, 'layer'> }
  | { type: 'resetValueSet'; id: string }
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
      return { ...state, valueSets: next }
    }
    case 'reset':
      return { labels: {}, valueSets: {} }
    case 'load':
      return {
        labels: structuredClone(action.payload.labels),
        valueSets: structuredClone(action.payload.valueSets),
      }
  }
}

function countLabels(labels: LocaleLabelMap): number {
  return Object.values(labels).reduce((acc, map) => acc + Object.keys(map).length, 0)
}

export interface DraftConfigContextValue {
  moduleKey: string
  labels: LocaleLabelMap
  valueSets: DraftValueSets
  /** Whether the draft has any change vs. the live tenant layer. */
  isDirty: boolean
  /** Total touched entries (labels + value-sets) for the footer counter. */
  changeCount: number
  setDraftLabel: (locale: string, key: string, value: string) => void
  resetDraftLabel: (locale: string, key: string) => void
  setDraftValueSet: (id: string, set: Omit<ValueSet, 'layer'>) => void
  resetDraftValueSet: (id: string) => void
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
      ? { labels: structuredClone(initialPayload.labels), valueSets: structuredClone(initialPayload.valueSets) }
      : { labels: {}, valueSets: {} },
  )

  // Every i18n key we ever wrote to the global bundle — scrubbed on unmount so a
  // draft that is discarded leaves no residue in the live app (R-1 mitigation).
  const touchedKeysRef = useRef<Set<string>>(new Set())

  // Live label preview: re-apply the merged (tenant ⊕ draft) overlay whenever the
  // draft labels change. Modules read t() from the shared instance, so this makes
  // the preview update instantly (ICU-Live-Fix carries the re-render).
  useEffect(() => {
    for (const key of Object.keys(state.labels[locale] ?? {})) touchedKeysRef.current.add(key)
    applyLabelOverlay(locale, resolveLabelOverrides(locale, false, state.labels))
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
    const changeCount = countLabels(state.labels) + Object.keys(state.valueSets).length
    return {
      moduleKey,
      labels: state.labels,
      valueSets: state.valueSets,
      isDirty: changeCount > 0,
      changeCount,
      setDraftLabel: (l, key, val) => dispatch({ type: 'setLabel', locale: l, key, value: val }),
      resetDraftLabel: (l, key) => dispatch({ type: 'resetLabel', locale: l, key }),
      setDraftValueSet: (id, set) => dispatch({ type: 'setValueSet', id, set }),
      resetDraftValueSet: (id) => dispatch({ type: 'resetValueSet', id }),
      resetAll: () => dispatch({ type: 'reset' }),
      buildPayload: () => ({
        labels: structuredClone(state.labels),
        valueSets: structuredClone(state.valueSets),
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

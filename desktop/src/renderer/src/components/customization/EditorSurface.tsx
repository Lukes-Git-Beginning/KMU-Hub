/**
 * EditorSurface + EditableText (Modul-Editor, Edit-in-place pivot 2026-07-22).
 *
 * The customization editor stopped being a detached side-panel: you now navigate
 * the real module in the preview and edit elements *in context* by clicking them.
 * This is the mechanism.
 *
 * `EditorSurface` is a context that is ON only inside the editor sandbox. Any
 * label rendered through `<EditableText dkey="i18n.key" />` becomes:
 *   - in the live app (no surface)  → plain t(key), zero behaviour change
 *   - in the editor sandbox         → hover outline + click → inline rename,
 *                                     written to the draft, live across the module
 *
 * "What is editable" = "what is wrapped in EditableText" — the instrumentation is
 * the whitelist, which scales far better than a central key array.
 */
import { createContext, useContext, useEffect, useMemo, useRef, useState } from 'react'
import type { ElementType, ReactElement } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { cn } from '@/lib'
import { resolveValueSet, resolveModuleAreas, resolveModuleAreaLayout } from '@/mocks/data/customization'
import { useCustomFields } from '@/api/hooks/useCustomFields'
import type { ModuleAreaLayout, ModuleAreaMap, ModuleAreasOverlay, ResolvedValueSet, ValueSet, ValueSetMigrations } from '@/api/customization-types'
import type { CustomFieldDefinition, CustomFieldEntity, DraftCustomFieldMap } from '@/mocks/data/custom-fields'

/** The editor's left-rail dimensions. Lives here (not in the editor) so modules
 *  can react to the selected one without importing from the editor. */
export type EditorFocusSection =
  | 'felder'
  | 'begriffe'
  | 'wertelisten'
  | 'bereiche'
  | 'spalten'
  | 'statistik'
  | 'kanäle'

export interface EditorSurfaceValue {
  /** True only inside the editor sandbox. */
  editing: boolean
  /** Write an inline label edit into the draft (locale handled by the provider). */
  setLabel: (key: string, value: string) => void
  /** Whether this key currently carries an unsaved draft edit. */
  isDraft: (key: string) => boolean
  /** Draft value-sets staged in the editor — previewed live by modules that consume them. */
  valueSets: Record<string, Omit<ValueSet, 'layer'>>
  /** Draft module-area visibility overrides — previewed live (tabs/sections on-off). */
  moduleAreas: ModuleAreasOverlay
  /**
   * Write a column's layout (order/width) into the draft. Separate from `setLabel`
   * because it is triggered from the module itself — dragging a column edge in the
   * preview — not from a panel. No-op outside the sandbox.
   */
  setAreaLayout: (areaKey: string, patch: ModuleAreaLayout) => void
  /** Same for several areas in one undo step (`focusKey` = the column being dragged). */
  setAreaLayouts: (patches: Record<string, ModuleAreaLayout>, focusKey: string) => void
  /** Draft record migrations for deleted value-set options — previewed live (R4b). */
  valueSetMigrations: ValueSetMigrations
  /** Draft per-entity custom-field snapshots — previewed live by modules that render custom fields (G2). */
  customFields: DraftCustomFieldMap
  /** Left-rail section the user just selected — the preview follows it (see useEditorFocusEffect). */
  focusSection: EditorFocusSection | null
  /** Bumped on every rail click, so re-selecting the same section re-focuses the preview. */
  focusNonce: number
  /**
   * The other direction: the module reports where the user now IS, so the rail and
   * the properties panel follow the preview (see useEditorContextReport). No-op
   * outside the sandbox.
   */
  reportContext: (section: EditorFocusSection | null) => void
}

const noop = (): void => {}
const EditorSurfaceContext = createContext<EditorSurfaceValue>({
  editing: false,
  setLabel: noop,
  isDraft: () => false,
  valueSets: {},
  moduleAreas: {},
  setAreaLayout: noop,
  setAreaLayouts: noop,
  valueSetMigrations: {},
  customFields: {},
  focusSection: null,
  focusNonce: 0,
  reportContext: noop,
})

export function useEditorSurface(): EditorSurfaceValue {
  return useContext(EditorSurfaceContext)
}

export function EditorSurfaceProvider({
  value,
  children,
}: {
  value: EditorSurfaceValue
  children: React.ReactNode
}): ReactElement {
  return <EditorSurfaceContext.Provider value={value}>{children}</EditorSurfaceContext.Provider>
}

/**
 * A single customizable label. Renders `t(dkey)`; inside the editor sandbox it is
 * click-to-rename in place. Drop-in for `{t('some.key')}` in a module.
 *
 * `interactive` — set when the label ALSO sits inside a control (a tab button, a
 * clickable card). Then a single click falls through to that control (so the
 * preview stays navigable) and renaming happens on double-click. Static labels
 * (table headers, section titles) leave it off → single click renames.
 */
export function EditableText({
  dkey,
  as: As = 'span',
  className,
  interactive = false,
}: {
  dkey: string
  /** Element to render as (default span). */
  as?: ElementType
  className?: string
  /** The label is nested in a clickable control → single click navigates, double click renames. */
  interactive?: boolean
}): ReactElement {
  const { t } = useTranslation()
  const { editing, setLabel, isDraft } = useEditorSurface()
  const [active, setActive] = useState(false)
  const text = t(dkey)

  // Live app: no editor surface → render exactly as before.
  if (!editing) return <As className={className}>{text}</As>

  if (active) {
    return (
      <input
        autoFocus
        defaultValue={text}
        onClick={(e) => e.stopPropagation()}
        onKeyDown={(e) => {
          e.stopPropagation()
          if (e.key === 'Enter') {
            setLabel(dkey, (e.target as HTMLInputElement).value)
            setActive(false)
          } else if (e.key === 'Escape') {
            setActive(false)
          }
        }}
        onBlur={(e) => {
          setLabel(dkey, e.target.value)
          setActive(false)
        }}
        className={cn(
          'min-w-[3ch] rounded-[4px] bg-background px-1 outline-none ring-2 ring-[var(--accent-1)]',
          className,
        )}
        style={{ width: `${Math.max(text.length + 1, 4)}ch` }}
      />
    )
  }

  // Nested in a control: don't hijack the single click (let the tab/card do its
  // job) — edit on double click instead.
  if (interactive) {
    return (
      <As
        onDoubleClick={(e: React.MouseEvent) => {
          e.stopPropagation()
          e.preventDefault()
          setActive(true)
        }}
        title="Doppelklick zum Umbenennen"
        className={cn(
          'rounded-[4px] transition-shadow',
          'hover:ring-2 hover:ring-[var(--accent-1)]/40',
          isDraft(dkey) && 'ring-1 ring-amber-500/50',
          className,
        )}
      >
        {text}
      </As>
    )
  }

  return (
    <As
      role="button"
      tabIndex={0}
      onClick={(e: React.MouseEvent) => {
        e.stopPropagation()
        e.preventDefault()
        setActive(true)
      }}
      onKeyDown={(e: React.KeyboardEvent) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.stopPropagation()
          e.preventDefault()
          setActive(true)
        }
      }}
      title="Klicken zum Umbenennen"
      className={cn(
        'cursor-text rounded-[4px] outline-none ring-offset-1 transition-shadow',
        'hover:ring-2 hover:ring-[var(--accent-1)]/60 focus-visible:ring-2 focus-visible:ring-[var(--accent-1)]',
        isDraft(dkey) && 'ring-1 ring-amber-500/50',
        className,
      )}
    >
      {text}
    </As>
  )
}

/**
 * Guard event handlers so they no-op inside the editor sandbox — the editor is for
 * customizing a module, not operating it (R-1). In the live app (no surface) the
 * handler runs unchanged. Mutating/out-navigating actions (create, delete, send,
 * assign, escalate…) wrap their handler in this; state-only navigation (tab
 * switch, open a detail) stays unwrapped so the preview remains walkable.
 */
export function useEditorGuard(): <A extends unknown[]>(
  fn: (...args: A) => void,
) => (...args: A) => void {
  const { editing } = useEditorSurface()
  const { t } = useTranslation()
  return <A extends unknown[]>(fn: (...args: A) => void) =>
    (...args: A): void => {
      if (editing) {
        toast.info(t('customization.editor.actionBlocked'), { id: 'cosmi-editor-action-blocked' })
        return
      }
      fn(...args)
    }
}

/**
 * Resolve a customizable value-set (status/priority/type chips) for a module.
 * In the live app it resolves the persisted layers (default ⊕ vendor ⊕ tenant);
 * inside the editor sandbox the staged draft is layered on top so edits in the
 * Wertelisten panel preview live. Modules render option labels/colors from this
 * instead of hardcoded i18n enums → the customization actually reaches the UI.
 */
export function useModuleValueSet(id: string): ResolvedValueSet | null {
  const { editing, valueSets } = useEditorSurface()
  const draftOverlay = editing ? valueSets : undefined
  return useMemo(() => resolveValueSet(id, false, draftOverlay), [id, editing, draftOverlay])
}

/**
 * Resolve which sub-areas (tabs/sections) of a module are enabled (R4). Returns
 * the merged explicit map — a consumer treats a missing key as enabled:
 * `useModuleAreas('helpdesk')[areaKey] !== false`. In the editor the staged draft
 * is layered on top so toggling a tab off hides it live in the preview.
 */
export function useModuleAreas(moduleKey: string): ModuleAreaMap {
  const { editing, moduleAreas } = useEditorSurface()
  const draftOverlay = editing ? moduleAreas : undefined
  return useMemo(
    () => resolveModuleAreas(moduleKey, false, draftOverlay),
    [moduleKey, editing, draftOverlay],
  )
}

/** Prefix that separates list-column settings from real areas inside moduleAreas. */
export const COLUMN_AREA_PREFIX = 'col:'

export interface ModuleColumnLayout {
  /** areaKey (incl. `col:` prefix) → configured order/width. Empty = nothing set. */
  layout: Record<string, ModuleAreaLayout>
  /** True only inside the editor sandbox — gates the resize handles. */
  editing: boolean
  /**
   * Whether to SHOW the column dividers (Darien 2026-08-05: "man sieht nicht wo
   * der verstellpunkt ist, da das alles weiß ist"). Only true in the editor while
   * the Spalten section is selected — the live module keeps its plain header, and
   * the other editor sections are not about column widths.
   */
  showDividers: boolean
  /** Persist a dragged width as a share of the table (0–1). */
  setWidth: (columnKey: string, width: number) => void
  /**
   * Freeze the columns' current widths in one step. Needed before the first drag:
   * the moment a table switches to `table-layout: fixed`, every column without an
   * explicit width gets an equal share — which would reshuffle the whole list on
   * the first pixel of the first drag. Sharing the dragged column's undo key keeps
   * freeze + drag a single step.
   */
  freezeWidths: (widths: Record<string, number>, draggedKey: string) => void
}

/**
 * Resolve how a module's list columns are laid out — order and width (Darien
 * 2026-08-05). Same layer model as everything else: live tenant/vendor settings,
 * with the editor draft on top so a drag previews immediately.
 *
 * Rollout note: a module needs exactly three things to become column-configurable —
 * this hook, `orderColumns` on its column array, and `columnWidthStyle` on its
 * headers. Nothing module-specific in here.
 */
export function useModuleColumnLayout(moduleKey: string): ModuleColumnLayout {
  const { editing, focusSection, moduleAreas, setAreaLayout, setAreaLayouts } = useEditorSurface()
  const draftOverlay = editing ? moduleAreas : undefined
  const showDividers = editing && focusSection === 'spalten'
  const layout = useMemo(
    () => resolveModuleAreaLayout(moduleKey, false, draftOverlay),
    [moduleKey, editing, draftOverlay],
  )
  return useMemo(
    () => ({
      layout,
      editing,
      showDividers,
      setWidth: (columnKey: string, width: number) =>
        setAreaLayout(`${COLUMN_AREA_PREFIX}${columnKey}`, { width }),
      freezeWidths: (widths: Record<string, number>, draggedKey: string) =>
        setAreaLayouts(
          Object.fromEntries(
            Object.entries(widths).map(([key, width]) => [`${COLUMN_AREA_PREFIX}${key}`, { width }]),
          ),
          `${COLUMN_AREA_PREFIX}${draggedKey}`,
        ),
    }),
    [layout, editing, showDividers, setAreaLayout, setAreaLayouts],
  )
}

/**
 * Sort a module's columns by the configured order. Columns without one keep their
 * code position, so a half-configured list never scrambles — and once the user
 * drags anything, every column carries an order anyway.
 */
export function orderColumns<T extends { key: string }>(
  columns: T[],
  layout: Record<string, ModuleAreaLayout>,
): T[] {
  return columns
    .map((column, index) => ({ column, index, order: layout[`${COLUMN_AREA_PREFIX}${column.key}`]?.order }))
    .sort((a, b) => (a.order ?? a.index) - (b.order ?? b.index) || a.index - b.index)
    .map((entry) => entry.column)
}

/**
 * The `<th>` style for a configured column. Widths are stored as a share of the
 * table, so they hold up when the window is narrower than the one they were set in.
 */
export function columnWidthStyle(
  layout: Record<string, ModuleAreaLayout>,
  columnKey: string,
): { width?: string } {
  const width = layout[`${COLUMN_AREA_PREFIX}${columnKey}`]?.width
  return width === undefined ? {} : { width: `${(width * 100).toFixed(2)}%` }
}

/**
 * The staged record-migration map for one value-set (R4b): removedOptionId →
 * targetOptionId. Only populated inside the editor sandbox; a module remaps record
 * values through it so deleting-with-reassign previews live. Empty in the live app
 * (the real migration runs on deploy in the backend).
 */
const EMPTY_MIGRATION: Record<string, string> = {}
export function useValueSetMigration(setId: string): Record<string, string> {
  const { editing, valueSetMigrations } = useEditorSurface()
  return editing ? (valueSetMigrations[setId] ?? EMPTY_MIGRATION) : EMPTY_MIGRATION
}

/**
 * Resolve the effective custom-field list for an entity (G2). In the live app it
 * returns the persisted definitions (React Query); inside the editor sandbox the
 * staged draft snapshot is layered on top, so adding/renaming a field in the
 * Felder panel previews live in the module's detail/create forms. A module renders
 * its "Zusatzfelder" section from this instead of a hardcoded list → custom fields
 * actually reach the UI (edit-in-place consistency). Only visible fields are
 * returned, sorted by display order.
 */
/** One resolved choice of a select field — from a bound value list or a free option. */
export interface FieldOption {
  /** Stored value (value-set option id, or the plain option text when unbound). */
  id: string
  label: string
  color?: string
}

/**
 * Resolve what a select field offers (Darien 2026-08-04). Two sources:
 *   - bound to a value list (`valueSetId`) → labels/colours/order come from the
 *     list, so renaming an option in the Wertelisten panel reaches the record AND
 *     the statistics at once. This is what gives a NEW value list a place in the
 *     module — without a field pointing at it, a list has no column to live in.
 *   - unbound → its own free `options` strings, as before.
 *
 * Takes the whole field list and returns key → options, so callers can resolve
 * many fields without calling a hook per field. Inside the editor sandbox the
 * staged draft is layered on, so edits preview live.
 */
export function useFieldOptions(fields: CustomFieldDefinition[]): Record<string, FieldOption[]> {
  const { editing, valueSets } = useEditorSurface()
  const draftOverlay = editing ? valueSets : undefined
  return useMemo(() => {
    const out: Record<string, FieldOption[]> = {}
    for (const f of fields) {
      if (f.valueSetId) {
        const resolved = resolveValueSet(f.valueSetId, false, draftOverlay)
        out[f.key] = (resolved?.options ?? [])
          .filter((o) => o.active)
          .map((o) => ({ id: o.id, label: o.label, color: o.color }))
      } else {
        out[f.key] = f.options.map((o) => ({ id: o, label: o }))
      }
    }
    return out
  }, [fields, draftOverlay])
}

/**
 * Let the preview follow the editor's left rail (Darien 2026-08-04): clicking
 * "Statistik" should put the preview ON the statistics tab, clicking "Zusatzfelder"
 * should open a record detail where those fields live — instead of making the user
 * navigate the module by hand to find what they are editing.
 *
 * A module declares where each dimension is visible; unlisted sections do nothing:
 *
 *   useEditorFocusEffect({
 *     statistik: () => setTab('statistik'),
 *     felder: () => { setTab('tickets'); openFirstTicket() },
 *   })
 *
 * No-op outside the editor sandbox, so this is safe to leave in a live module.
 * Rollout note: this is the standard instrumentation for every module the editor
 * gets pointed at — same shape everywhere.
 */
export function useEditorFocusEffect(
  handlers: Partial<Record<EditorFocusSection, () => void>>,
): void {
  const { editing, focusSection, focusNonce } = useEditorSurface()
  // Handlers close over fresh module state each render; keep the latest in a ref so
  // the focus effect never re-runs just because a callback identity changed.
  const latest = useRef(handlers)
  useEffect(() => {
    latest.current = handlers
  })
  // Same for the section: the rail bumps `focusNonce` on EVERY click, so the nonce
  // alone is the complete "the rail asked for something" signal. Depending on
  // `focusSection` as well would misfire since useEditorContextReport (below) also
  // moves it — a module reporting "I am on Felder now" would re-run the felder
  // handler and yank the preview back to its first record.
  const section = useRef(focusSection)
  useEffect(() => {
    section.current = focusSection
  })
  useEffect(() => {
    if (!editing || !section.current) return
    latest.current[section.current]?.()
  }, [editing, focusNonce])
}

/**
 * The counterpart to useEditorFocusEffect (Darien 2026-08-06): the coupling used
 * to be one-way — the rail moved the preview, but walking the module by hand left
 * rail and properties panel behind ("wenn ich im Modul auf Statistik klicke,
 * wechselt das Menü links und rechts nicht").
 *
 * A module reports the section that its CURRENT view is about:
 *
 *   useEditorContextReport(tab === 'statistik' ? 'statistik' : detailOpen ? 'felder' : null)
 *
 * Only report places that mean exactly one section. A plain list is home to
 * Begriffe, Wertelisten, Bereiche AND Spalten at once → report `null` there, which
 * leaves the rail alone (and only clears it if it was showing a place the user has
 * now left). That rule is what keeps the two directions from fighting each other.
 *
 * No-op outside the sandbox — safe to leave in a live module.
 */
export function useEditorContextReport(section: EditorFocusSection | null): void {
  const { editing, reportContext } = useEditorSurface()
  // Report on change only — a module re-renders constantly, the rail must not.
  const last = useRef<EditorFocusSection | null>(null)
  useEffect(() => {
    if (!editing || last.current === section) return
    last.current = section
    reportContext(section)
  }, [editing, section, reportContext])
}

export function useModuleCustomFields(entity: CustomFieldEntity): CustomFieldDefinition[] {
  const { editing, customFields } = useEditorSurface()
  const { data: live } = useCustomFields(entity)
  const draftSnapshot = editing ? customFields[entity] : undefined
  return useMemo(() => {
    const list = draftSnapshot ?? live ?? []
    return [...list].filter((f) => f.visible).sort((a, b) => a.order - b.order)
  }, [draftSnapshot, live])
}

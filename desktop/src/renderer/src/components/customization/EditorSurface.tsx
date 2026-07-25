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
import { createContext, useContext, useMemo, useState } from 'react'
import type { ElementType, ReactElement } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { cn } from '@/lib'
import { resolveValueSet } from '@/mocks/data/customization'
import type { ResolvedValueSet, ValueSet } from '@/api/customization-types'

export interface EditorSurfaceValue {
  /** True only inside the editor sandbox. */
  editing: boolean
  /** Write an inline label edit into the draft (locale handled by the provider). */
  setLabel: (key: string, value: string) => void
  /** Whether this key currently carries an unsaved draft edit. */
  isDraft: (key: string) => boolean
  /** Draft value-sets staged in the editor — previewed live by modules that consume them. */
  valueSets: Record<string, Omit<ValueSet, 'layer'>>
}

const noop = (): void => {}
const EditorSurfaceContext = createContext<EditorSurfaceValue>({
  editing: false,
  setLabel: noop,
  isDraft: () => false,
  valueSets: {},
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

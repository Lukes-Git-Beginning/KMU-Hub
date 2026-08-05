/**
 * EditorWorkspace (Modul-Editor v1, E-2b) — the editor content, decoupled from
 * any window shell. Rendered full-window by EditorWindowPage (its own OS window,
 * KONZEPT §0 ① decision revised 2026-07-22: window instead of overlay).
 *
 * Self-contained: DraftConfigProvider lives here; the module preview brings its
 * own sandbox QueryClient. Because this is a top-level window (no centered,
 * transform-animated Dialog), the module renders on native pixels — no blur, no
 * nested-router artifacts (both were overlay-only symptoms).
 *
 * Layout (MARKT-EDITOR wireframe): 48px toolbar · amber sandbox banner · three
 * panels (trio-nav · module preview · properties) · commit footer. Trio-panel
 * editing + deploy dialog arrive in E-3 / E-5.
 */
import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useBlocker } from 'react-router-dom'
import { toast } from 'sonner'
import { X, Undo2, Redo2, Wand2, ChevronDown } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib'
import { saveDraft } from '@/mocks/data/customization-drafts'
import type { CustomizationDraft } from '@/api/customization-types'
import { publishDraftMirror, takeStashedDraft } from './customization-sync'
import { DraftConfigProvider, useDraftConfig } from './DraftConfigProvider'
import { EditorTrioNav, type EditorSection } from './EditorTrioNav'
import { EditorPropertiesPanel } from './EditorPropertiesPanel'
import { ModuleSandbox } from './ModuleSandbox'
import { EmbeddedFormEditor } from './EmbeddedFormEditor'
import { DeployDialog } from './DeployDialog'
import type { EditorModuleDef } from './editorModules'

export function EditorWorkspace({
  module,
  onClose,
}: {
  module: EditorModuleDef
  onClose: () => void
}): React.ReactElement {
  // Continuing a saved draft: the hub handed it over on open (see customization-
  // sync). Read once — a later plain open must start from a clean sheet.
  const [resumed] = useState(() => takeStashedDraft(module.key))
  return (
    <DraftConfigProvider moduleKey={module.key} initialPayload={resumed?.payload}>
      <EditorLayout module={module} onClose={onClose} resumed={resumed} />
    </DraftConfigProvider>
  )
}

function EditorLayout({
  module,
  onClose,
  resumed,
}: {
  module: EditorModuleDef
  onClose: () => void
  /** The draft this session continues — kept so saving updates it instead of forking. */
  resumed: CustomizationDraft | null
}): React.ReactElement {
  const { t } = useTranslation()
  const { isDirty, changeCount, buildPayload, canUndo, canRedo, undo, redo } = useDraftConfig()
  const [activeSection, setActiveSection] = useState<EditorSection | null>(null)
  const [deployOpen, setDeployOpen] = useState(false)
  // Bumped on every rail click so re-selecting the same section re-focuses the
  // preview (the user may have navigated the module away in between).
  const [focusNonce, setFocusNonce] = useState(0)
  // Ticket form being edited on the canvas (Darien 2026-08-04): editing a channel's
  // form is part of configuring the module, so the builder takes over the preview
  // area instead of navigating out of the editor.
  const [formEditId, setFormEditId] = useState<string | null>(null)

  const selectSection = (section: EditorSection): void => {
    setActiveSection(section)
    setFocusNonce((n) => n + 1)
    // Leaving the channels section closes the builder — the rail always shows what
    // the canvas is displaying.
    if (section !== 'kanäle') setFormEditId(null)
  }

  // Editor is for editing, not using: block any in-module action that navigates
  // away (email → /mails, call → /chat, out-linking rows). State-based navigation
  // inside the module (tab switch, detail modal) does not route, so it is
  // unaffected. In Electron the editor closes via window.close (not the router),
  // so this never traps the close.
  //
  // Exception (Darien 2026-08-04): the editor's OWN panels may route on purpose —
  // "Formular bearbeiten →" in the Kanäle panel is the web fallback for opening a
  // channel's ticket form. Those carry state.fromEditor and pass through; without
  // this the click was silently swallowed and the form looked uneditable.
  const blocker = useBlocker(({ nextLocation }) => nextLocation.state?.fromEditor !== true)
  useEffect(() => {
    if (blocker.state === 'blocked') blocker.reset()
  }, [blocker])

  const moduleName = t(module.titleKey)
  // A continued draft keeps its own name — renaming it on every save would make
  // the rollout list read as if a second rollout had appeared.
  const draftName = resumed?.name ?? t('customization.editor.draftName', { module: moduleName })
  // Which record this session writes to: the continued one, or the one created by
  // the first save. Without it, every save created another entry in the list.
  const [draftId, setDraftId] = useState<string | undefined>(resumed?.id)

  const handleSaveDraft = (): void => {
    const d = saveDraft({ id: draftId, moduleKey: module.key, name: draftName, payload: buildPayload() })
    setDraftId(d.id)
    publishDraftMirror(d)
    toast.success(t('customization.editor.toast.draftSaved'))
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      {/* Gradient stripe (Cosmi identity) */}
      <div className="h-0.5 w-full shrink-0 bg-gradient-to-r from-[var(--accent-1)] to-[var(--accent-2)]" />

      {/* ── Toolbar (48px) ─────────────────────────────────────────────── */}
      <div className="flex h-12 shrink-0 items-center justify-between gap-3 border-b px-3">
        <div className="flex min-w-0 items-center gap-2">
          <Button variant="ghost" size="icon" className="h-8 w-8 shrink-0" onClick={onClose} aria-label={t('customization.editor.close')}>
            <X className="h-4 w-4" aria-hidden="true" />
          </Button>
          <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-[var(--accent-1)]/10 text-[var(--accent-1)]">
            <Wand2 className="h-3.5 w-3.5" aria-hidden="true" />
          </div>
          <div className="min-w-0">
            <p className="truncate text-sm font-semibold leading-tight text-foreground">
              {t('customization.editor.titleBar', { module: moduleName })}
            </p>
            <p className="truncate text-[11px] leading-tight text-muted-foreground">
              {t('customization.editor.subtitle')}
            </p>
          </div>
        </div>

        <div className="flex shrink-0 items-center gap-1">
          <Button variant="ghost" size="icon" className="h-8 w-8" disabled={!canUndo} onClick={undo} aria-label={t('customization.editor.undo')}>
            <Undo2 className="h-4 w-4" aria-hidden="true" />
          </Button>
          <Button variant="ghost" size="icon" className="h-8 w-8" disabled={!canRedo} onClick={redo} aria-label={t('customization.editor.redo')}>
            <Redo2 className="h-4 w-4" aria-hidden="true" />
          </Button>
        </div>
      </div>

      {/* ── Amber sandbox banner ───────────────────────────────────────── */}
      <div className="flex shrink-0 items-center justify-between gap-3 border-b border-amber-500/20 bg-amber-500/10 px-4 py-1.5">
        <p className="flex items-center gap-2 text-xs font-medium text-amber-700 dark:text-amber-400">
          <span className="inline-block h-1.5 w-1.5 rounded-full bg-amber-500" aria-hidden="true" />
          {t('customization.editor.sandboxBanner')}
        </p>
      </div>

      {/* ── Body: trio-nav · preview · properties ──────────────────────── */}
      <div className="flex min-h-0 flex-1">
        <EditorTrioNav active={activeSection} onSelect={selectSection} />
        <div className="min-w-0 flex-1">
          {/* The canvas shows either the module preview — which follows the rail,
              selecting a dimension navigates to where it is visible — or the ticket
              form builder when a channel's form is being edited. */}
          {formEditId ? (
            <EmbeddedFormEditor formId={formEditId} onBack={() => setFormEditId(null)} />
          ) : (
            <ModuleSandbox module={module} focusSection={activeSection} focusNonce={focusNonce} />
          )}
        </div>
        <EditorPropertiesPanel
          section={activeSection}
          moduleKey={module.key}
          onEditForm={setFormEditId}
          editingFormId={formEditId}
        />
      </div>

      {/* ── Commit footer (48px) ───────────────────────────────────────── */}
      <div className="flex h-12 shrink-0 items-center justify-between gap-3 border-t px-4">
        <p className={cn('text-xs', isDirty ? 'font-medium text-foreground' : 'text-muted-foreground')}>
          {t('customization.editor.footer.changes', { count: changeCount })}
        </p>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" className="h-8" disabled={!isDirty} onClick={handleSaveDraft}>
            {t('customization.editor.footer.saveDraft')}
          </Button>
          <Button size="sm" className="h-8 gap-1.5" disabled={!isDirty} onClick={() => setDeployOpen(true)}>
            {t('customization.editor.footer.apply')}
            <ChevronDown className="h-3.5 w-3.5 opacity-80" aria-hidden="true" />
          </Button>
        </div>
      </div>

      <DeployDialog
        open={deployOpen}
        onClose={() => setDeployOpen(false)}
        draftId={draftId}
        moduleKey={module.key}
        draftName={draftName}
        changeCount={changeCount}
        buildPayload={buildPayload}
        onDeployed={onClose}
      />
    </div>
  )
}

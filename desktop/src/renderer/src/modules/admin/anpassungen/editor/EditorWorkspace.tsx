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
import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useBlocker } from 'react-router-dom'
import { toast } from 'sonner'
import { X, Undo2, Redo2, Wand2, ChevronDown } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { DetailModal } from '@/components/shared/DetailModal'
import { cn } from '@/lib'
import { i18n } from '@/i18n/i18n'
import { saveDraft } from '@/mocks/data/customization-drafts'
import type { CustomizationDraft } from '@/api/customization-types'
import { publishDraftMirror, takeStashedDraft, useResumeDraftListener } from './customization-sync'
import { DraftConfigProvider, useDraftConfig } from './DraftConfigProvider'
import { EditorTrioNav, type EditorSection } from './EditorTrioNav'
import { EditorPropertiesPanel } from './EditorPropertiesPanel'
import { ModuleSandbox } from './ModuleSandbox'
import { EmbeddedFormEditor } from './EmbeddedFormEditor'
import { DeployDialog } from './DeployDialog'
import type { EditorModuleDef } from './editorModules'

/**
 * Sections that live at ONE identifiable place inside the module (a tab of their
 * own, a record detail). Everything else — Begriffe, Wertelisten, Bereiche,
 * Spalten — is judged on the list, which is also the module's default view, so a
 * module returning there says nothing about which of them the user meant.
 */
const LOCATED_SECTIONS = new Set<EditorSection>(['statistik', 'felder', 'kanäle'])

/**
 * Key-order-independent serialisation, used to answer "is there work that has not
 * been saved yet?". `isDirty` cannot answer that — it means "differs from live",
 * which stays true after saving (the draft holds exactly those differences).
 */
function stableStringify(value: unknown): string {
  if (value === null || typeof value !== 'object') return JSON.stringify(value) ?? 'null'
  if (Array.isArray(value)) return `[${value.map(stableStringify).join(',')}]`
  const entries = Object.entries(value as Record<string, unknown>)
    .filter(([, v]) => v !== undefined)
    .sort(([a], [b]) => a.localeCompare(b))
  return `{${entries.map(([k, v]) => `${JSON.stringify(k)}:${stableStringify(v)}`).join(',')}}`
}

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
  const { isDirty, changeCount, buildPayload, canUndo, canRedo, undo, redo, loadDraft, resetAll } = useDraftConfig()
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

  // The way back (Darien 2026-08-06): walking the module by hand moves the rail and
  // the properties panel too. Deliberately WITHOUT bumping focusNonce — that nonce
  // is the rail asking the preview to navigate, and echoing it here would push the
  // preview back where the rail last pointed.
  const reportContext = useCallback((section: EditorSection | null): void => {
    setActiveSection((current) => {
      // "I am nowhere in particular" (the plain list): only clear a section that
      // has a place of its own, because that place is the one just left. Sections
      // that live on the list itself stay selected — the user is still looking at
      // them.
      if (section === null) return current && LOCATED_SECTIONS.has(current) ? null : current
      return section
    })
  }, [])

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
  // Which record this session writes to: the continued one, or the one created by
  // the first save. Without it, every save created another entry in the list.
  const [draftId, setDraftId] = useState<string | undefined>(resumed?.id)
  // The name the user gave this draft (or the one a continued draft brought with
  // it). Undefined = not named yet → the first save asks (Darien 2026-08-09:
  // „sonst heißen drei unterschiedliche Entwürfe für Helpdesk alle gleich und man
  // weiß nicht was welcher war").
  const [draftName, setDraftName] = useState<string | undefined>(resumed?.name)
  const [nameDialogOpen, setNameDialogOpen] = useState(false)
  const [nameInput, setNameInput] = useState('')
  // A draft handed over while this window has unsaved work — the user decides.
  const [pendingResume, setPendingResume] = useState<CustomizationDraft | null>(null)
  /** The payload as it was last written to the record — anything else is unsaved. */
  const [savedSignature, setSavedSignature] = useState<string | null>(
    resumed ? stableStringify(resumed.payload) : null,
  )

  /** True only for work that has NOT been written to the draft record yet. */
  const hasUnsavedWork = (): boolean =>
    savedSignature === null ? isDirty : stableStringify(buildPayload()) !== savedSignature

  /** What the name field starts with: the module plus when it was saved. */
  const suggestedName = (): string =>
    t('customization.editor.draftNameSuggestion', {
      module: moduleName,
      date: new Date().toLocaleString(i18n.language, {
        day: 'numeric',
        month: 'short',
        hour: '2-digit',
        minute: '2-digit',
      }),
    })

  // Only one editor window exists per module, so "Weiter bearbeiten" on an
  // already-open editor arrives as a message instead of a fresh window. Unsaved
  // work wins: loading over it would throw away what is on screen.
  const applyResume = (incoming: CustomizationDraft): void => {
    loadDraft(incoming.payload)
    setDraftId(incoming.id)
    setDraftName(incoming.name)
    setSavedSignature(stableStringify(incoming.payload))
    setPendingResume(null)
    toast.success(t('customization.editor.toast.resumed', { name: incoming.name }))
  }

  useResumeDraftListener(module.key, (incoming) => {
    // Unsaved work is never thrown away silently — but refusing outright meant
    // the user asked for a draft and kept looking at the current state without a
    // way forward. Ask instead. Saved work loads without a question: `isDirty`
    // would say yes forever here, because a saved draft still differs from live.
    if (hasUnsavedWork()) {
      setPendingResume(incoming)
      return
    }
    applyResume(incoming)
  })

  /** Drop the continued draft and start from the live state. */
  const startFresh = (): void => {
    resetAll()
    setDraftId(undefined)
    setDraftName(undefined)
    setSavedSignature(null)
    toast.success(t('customization.editor.toast.freshStart'))
  }

  /** Write to the record — with the name the user gave it. */
  const persistDraft = (name: string): void => {
    const payload = buildPayload()
    const d = saveDraft({ id: draftId, moduleKey: module.key, name, payload })
    setDraftId(d.id)
    setDraftName(name)
    setSavedSignature(stableStringify(payload))
    publishDraftMirror(d)
    toast.success(t('customization.editor.toast.draftSaved'))
  }

  const handleSaveDraft = (): void => {
    // Named once, then quiet: asking on every save would make saving a chore.
    // Renaming later happens on the rollout entry.
    if (draftName) {
      persistDraft(draftName)
      return
    }
    setNameInput(suggestedName())
    setNameDialogOpen(true)
  }

  const confirmName = (): void => {
    const name = nameInput.trim()
    if (!name) {
      toast.warning(t('customization.editor.saveDialog.nameRequired'))
      return
    }
    setNameDialogOpen(false)
    persistDraft(name)
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
        {/* Which draft is on screen. Without this the editor silently continued
            something and looked like it had forgotten the last session — the
            same confusion from the other side. */}
        {draftName && (
          <p className="flex shrink-0 items-center gap-2 text-xs text-amber-700 dark:text-amber-400">
            {t('customization.editor.resumedBanner', { name: draftName })}
            <button
              type="button"
              onClick={startFresh}
              className="rounded px-1.5 py-0.5 font-medium underline decoration-dotted underline-offset-2 transition-colors hover:bg-amber-500/20"
            >
              {t('customization.editor.freshStart')}
            </button>
          </p>
        )}
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
            <ModuleSandbox
              module={module}
              focusSection={activeSection}
              focusNonce={focusNonce}
              onContextChange={reportContext}
            />
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
        draftName={draftName ?? suggestedName()}
        changeCount={changeCount}
        buildPayload={buildPayload}
        onDeployed={onClose}
      />

      {/* Name the draft — once, on the first save. Three drafts of the same module
          were otherwise indistinguishable in the rollout list. */}
      <DetailModal
        open={nameDialogOpen}
        onClose={() => setNameDialogOpen(false)}
        title={t('customization.editor.saveDialog.title')}
        maxWidth="max-w-md"
        footer={
          <div className="flex items-center justify-end gap-2">
            <Button variant="outline" size="sm" onClick={() => setNameDialogOpen(false)}>
              {t('customization.editor.deploy.cancel')}
            </Button>
            <Button size="sm" onClick={confirmName}>
              {t('customization.editor.saveDialog.confirm')}
            </Button>
          </div>
        }
      >
        <label className="flex flex-col gap-1.5">
          <span className="text-xs font-medium text-foreground">
            {t('customization.editor.saveDialog.nameLabel')}
          </span>
          <input
            autoFocus
            value={nameInput}
            onChange={(e) => setNameInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') confirmName()
            }}
            placeholder={t('customization.editor.saveDialog.namePlaceholder')}
            className="h-9 w-full rounded-md border border-border bg-background px-2.5 text-sm outline-none focus:border-primary"
          />
        </label>
        <p className="mt-2 text-xs leading-relaxed text-muted-foreground">
          {t('customization.editor.saveDialog.hint')}
        </p>
      </DetailModal>

      {/* A draft was handed over while this window has unsaved work. */}
      <DetailModal
        open={pendingResume !== null}
        onClose={() => setPendingResume(null)}
        title={t('customization.editor.resumeDialog.title')}
        maxWidth="max-w-md"
        footer={
          <div className="flex items-center justify-end gap-2">
            <Button variant="outline" size="sm" onClick={() => setPendingResume(null)}>
              {t('customization.editor.resumeDialog.cancel')}
            </Button>
            <Button size="sm" onClick={() => pendingResume && applyResume(pendingResume)}>
              {t('customization.editor.resumeDialog.confirm')}
            </Button>
          </div>
        }
      >
        <p className="text-sm leading-relaxed text-muted-foreground">
          {t('customization.editor.resumeDialog.body', { name: pendingResume?.name ?? '' })}
        </p>
      </DetailModal>
    </div>
  )
}

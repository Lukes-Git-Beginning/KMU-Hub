/**
 * EditorFrame (Modul-Editor v1, E-2) — the editor window shell.
 *
 * A near-fullscreen in-app overlay (KONZEPT §0 ①) built on the shared Radix
 * Dialog, kept self-contained so a later move to a real Electron window is just
 * a mount-point swap: DraftConfigProvider + the sandbox's own QueryClient +
 * MemoryRouter live entirely inside this tree, nothing leaks to the live app.
 *
 * Layout (MARKT-EDITOR wireframe): gradient stripe · 48px toolbar · amber
 * sandbox banner · three panels (trio-nav · module preview · properties) ·
 * commit footer. Trio-panel editing + deploy dialog arrive in E-3 / E-5; here
 * the frame, sandbox render and draft plumbing come together.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { X, Undo2, Redo2, Eye, EyeOff, Wand2, ChevronDown } from 'lucide-react'
import { Dialog, DialogContent, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib'
import { commitDraftNow, saveDraft } from '@/mocks/data/customization-drafts'
import { DraftConfigProvider, useDraftConfig } from './DraftConfigProvider'
import { EditorTrioNav, type EditorSection } from './EditorTrioNav'
import { EditorPropertiesPanel } from './EditorPropertiesPanel'
import { ModuleSandbox } from './ModuleSandbox'
import { getEditorModule, type EditorModuleDef } from './editorModules'

interface EditorFrameProps {
  open: boolean
  moduleKey: string | null
  onClose: () => void
}

export function EditorFrame({ open, moduleKey, onClose }: EditorFrameProps): React.ReactElement {
  const { t } = useTranslation()
  const module = moduleKey ? getEditorModule(moduleKey) : undefined

  return (
    <Dialog
      open={open && !!module}
      onOpenChange={(v) => {
        if (!v) onClose()
      }}
    >
      <DialogContent
        showCloseButton={false}
        className="flex h-[94vh] w-[96vw] max-w-none flex-col gap-0 overflow-hidden rounded-2xl border p-0"
      >
        <DialogTitle className="sr-only">{t('customization.editor.a11yTitle')}</DialogTitle>
        <div className="h-0.5 w-full shrink-0 bg-gradient-to-r from-[var(--accent-1)] to-[var(--accent-2)]" />
        {module && <EditorChrome module={module} onClose={onClose} />}
      </DialogContent>
    </Dialog>
  )
}

/** Inner chrome — only mounted with a module, so the draft hooks always have one. */
function EditorChrome({
  module,
  onClose,
}: {
  module: EditorModuleDef
  onClose: () => void
}): React.ReactElement {
  return (
    <DraftConfigProvider moduleKey={module.key}>
      <EditorLayout module={module} onClose={onClose} />
    </DraftConfigProvider>
  )
}

function EditorLayout({
  module,
  onClose,
}: {
  module: EditorModuleDef
  onClose: () => void
}): React.ReactElement {
  const { t } = useTranslation()
  const { isDirty, changeCount, buildPayload } = useDraftConfig()
  const [activeSection, setActiveSection] = useState<EditorSection | null>(null)
  const [previewOnly, setPreviewOnly] = useState(false)

  const moduleName = t(module.titleKey)

  const handleSaveDraft = (): void => {
    saveDraft({ moduleKey: module.key, name: t('customization.editor.draftName', { module: moduleName }), payload: buildPayload() })
    toast.success(t('customization.editor.toast.draftSaved'))
  }

  const handleApply = (): void => {
    // E-5 replaces this with the deploy dialog (now / scheduled / draft).
    commitDraftNow({ moduleKey: module.key, name: t('customization.editor.draftName', { module: moduleName }), payload: buildPayload(), mode: 'now' })
    toast.success(t('customization.editor.toast.applied'))
    onClose()
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
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
          <Button variant="ghost" size="icon" className="h-8 w-8" disabled aria-label={t('customization.editor.undo')}>
            <Undo2 className="h-4 w-4" aria-hidden="true" />
          </Button>
          <Button variant="ghost" size="icon" className="h-8 w-8" disabled aria-label={t('customization.editor.redo')}>
            <Redo2 className="h-4 w-4" aria-hidden="true" />
          </Button>
          <div className="mx-1 h-5 w-px bg-border" />
          <Button
            variant="ghost"
            size="sm"
            className="h-8 gap-1.5"
            onClick={() => setPreviewOnly((v) => !v)}
            aria-pressed={previewOnly}
          >
            {previewOnly ? <EyeOff className="h-3.5 w-3.5" aria-hidden="true" /> : <Eye className="h-3.5 w-3.5" aria-hidden="true" />}
            <span className="text-xs">{t('customization.editor.preview')}</span>
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
        {!previewOnly && <EditorTrioNav active={activeSection} onSelect={setActiveSection} />}
        <div className="min-w-0 flex-1">
          <ModuleSandbox module={module} />
        </div>
        {!previewOnly && <EditorPropertiesPanel section={activeSection} />}
      </div>

      {/* ── Commit footer (40px) ───────────────────────────────────────── */}
      <div className="flex h-12 shrink-0 items-center justify-between gap-3 border-t px-4">
        <p className={cn('text-xs', isDirty ? 'font-medium text-foreground' : 'text-muted-foreground')}>
          {t('customization.editor.footer.changes', { count: changeCount })}
        </p>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" className="h-8" disabled={!isDirty} onClick={handleSaveDraft}>
            {t('customization.editor.footer.saveDraft')}
          </Button>
          <Button size="sm" className="h-8 gap-1.5" disabled={!isDirty} onClick={handleApply}>
            {t('customization.editor.footer.apply')}
            <ChevronDown className="h-3.5 w-3.5 opacity-80" aria-hidden="true" />
          </Button>
        </div>
      </div>
    </div>
  )
}

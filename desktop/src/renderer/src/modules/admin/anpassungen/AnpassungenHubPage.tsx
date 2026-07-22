/**
 * AnpassungenTab — the "Anpassungen" tab in AdminHubPage: the entry into the
 * module editor.
 *
 * The old inline Felder/Begriffe admin lists (v1.1/v1.2) were removed — after the
 * re-scoping, customization happens ONLY inside the module editor, not on the
 * live admin surface (Darien 2026-07-22: "die Felder machen in Cosmi keinen Sinn,
 * sind nur im Baukasten-Editor"). E-4 turns this launch grid into the full module
 * gallery (status/preview, editable manifest per module).
 *
 * Gated on `admin:customization:manage` (admin + it_admin) via AdminHubPage.
 */
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Contact, LifeBuoy, Wand2 } from 'lucide-react'
import { ModuleLoadingFallback } from '@/components/layout/ModuleShell'
import { useCapabilitySet } from '@/hooks/useCapability'
import { EDITOR_MODULES } from './editor/editorModules'

/** Lucide icon per editor-module (matches EditorModuleDef.icon). */
const MODULE_ICON = { contact: Contact, lifeBuoy: LifeBuoy } as const

export default function AnpassungenTab() {
  const { t } = useTranslation()
  const { ready } = useCapabilitySet()
  const navigate = useNavigate()

  // Open the editor in its own OS window (web fallback: same-window route).
  const openEditor = (key: string): void => {
    if (window.electronAPI?.editor) {
      void window.electronAPI.editor.openWindow(key)
    } else {
      navigate(`/editor-window?module=${encodeURIComponent(key)}`)
    }
  }

  if (!ready) return <ModuleLoadingFallback />

  return (
    <div className="flex h-full flex-col overflow-y-auto px-6 py-6">
      <div className="mb-2.5 flex items-center gap-2">
        <div className="flex h-7 w-7 items-center justify-center rounded-md bg-[var(--accent-1)]/10 text-[var(--accent-1)]">
          <Wand2 className="h-4 w-4" aria-hidden="true" />
        </div>
        <h2 className="text-base font-semibold text-foreground">{t('customization.editor.launch.title')}</h2>
        <span className="rounded-full bg-amber-500/15 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-amber-600 dark:text-amber-400">
          {t('customization.editor.launch.beta')}
        </span>
      </div>
      <p className="mb-4 max-w-2xl text-sm leading-relaxed text-muted-foreground">
        {t('customization.editor.launch.subtitle')}
      </p>
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {EDITOR_MODULES.map((mod) => {
          const Icon = MODULE_ICON[mod.icon]
          return (
            <button
              key={mod.key}
              type="button"
              onClick={() => openEditor(mod.key)}
              className="group flex items-center gap-3 rounded-xl border bg-background px-4 py-3.5 text-left transition-colors hover:border-[var(--accent-1)]/40 hover:bg-[var(--accent-1)]/5"
            >
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-muted text-muted-foreground transition-colors group-hover:bg-[var(--accent-1)]/10 group-hover:text-[var(--accent-1)]">
                <Icon className="h-5 w-5" aria-hidden="true" />
              </div>
              <div className="min-w-0">
                <p className="truncate text-sm font-medium text-foreground">{t(mod.titleKey)}</p>
                <p className="truncate text-xs text-muted-foreground">{t('customization.editor.launch.open')}</p>
              </div>
            </button>
          )
        })}
      </div>
    </div>
  )
}

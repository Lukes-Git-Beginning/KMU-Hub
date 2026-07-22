/**
 * EditorWindowPage (Modul-Editor v1, E-2b) — the standalone page loaded into the
 * editor's own OS window at #/editor-window?module=<key>. Mirrors
 * ComposeWindowPage / EmployeeWizardWindowPage.
 *
 * Reads the target module from the hash query, renders EditorWorkspace full
 * height. Closing the editor closes the OS window (or navigates back in the web
 * fallback). Because this is a fresh top-level window, the module preview renders
 * natively — no Dialog transform, hence no blur and no nested-router issue.
 */
import { useSearchParams, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { EditorWorkspace } from './EditorWorkspace'
import { getEditorModule } from './editorModules'

export default function EditorWindowPage(): React.ReactElement {
  const { t } = useTranslation()
  const [params] = useSearchParams()
  const navigate = useNavigate()
  const moduleKey = params.get('module') ?? ''
  const module = getEditorModule(moduleKey)

  const close = (): void => {
    if (window.electronAPI?.window) {
      window.electronAPI.window.close()
    } else {
      // Web/dev fallback (no Electron window to close).
      navigate('/admin/anpassungen')
    }
  }

  if (!module) {
    return (
      <div className="flex h-screen flex-col items-center justify-center gap-2 bg-background text-foreground">
        <p className="text-sm text-muted-foreground">{t('customization.editor.unknownModule')}</p>
        <button
          type="button"
          onClick={close}
          className="text-sm text-primary hover:underline"
        >
          {t('customization.editor.close')}
        </button>
      </div>
    )
  }

  return (
    <div className="flex h-screen flex-col bg-background text-foreground">
      <EditorWorkspace module={module} onClose={close} />
    </div>
  )
}

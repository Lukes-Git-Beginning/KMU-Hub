import { ipcMain, BrowserWindow } from 'electron'
import { join } from 'path'

/**
 * Modul-Editor (Customization v1) — opens the module editor in its own OS window
 * so the customer can edit customizations while using Cosmi in the main window
 * and compare draft vs. live side by side. Mirrors compose.ts / employee-wizard.ts.
 * The target module is passed as a hash query param.
 */
export function registerEditorWindowHandlers(): void {
  ipcMain.handle('editor:open-window', (_event, moduleKey: string) => {
    const editorWin = new BrowserWindow({
      width: 1280,
      height: 860,
      minWidth: 900,
      minHeight: 600,
      title: 'Modul-Editor',
      webPreferences: {
        preload: join(__dirname, '../../preload/index.js'),
        contextIsolation: true,
        nodeIntegration: false,
        sandbox: true,
        webSecurity: true,
      },
    })

    const query = `?module=${encodeURIComponent(String(moduleKey ?? ''))}`
    if (process.env.ELECTRON_RENDERER_URL) {
      editorWin.loadURL(`${process.env.ELECTRON_RENDERER_URL}#/editor-window${query}`)
    } else {
      editorWin.loadFile(join(__dirname, '../../renderer/index.html'), {
        hash: `editor-window${query}`,
      })
    }
  })
}

import { ipcMain, BrowserWindow } from 'electron'
import { join } from 'path'

/**
 * Modul-Editor (Customization v1) — opens the module editor in its own OS window
 * so the customer can edit customizations while using Cosmi in the main window
 * and compare draft vs. live side by side. Mirrors compose.ts / employee-wizard.ts.
 * The target module is passed as a hash query param.
 */

/** Shared shell for the editor-family windows (same chrome, different route). */
function openWindow(hash: string, title: string, width: number, height: number): void {
  const win = new BrowserWindow({
    width,
    height,
    minWidth: 900,
    minHeight: 600,
    title,
    webPreferences: {
      preload: join(__dirname, '../../preload/index.js'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
      webSecurity: true,
    },
  })

  if (process.env.ELECTRON_RENDERER_URL) {
    win.loadURL(`${process.env.ELECTRON_RENDERER_URL}#/${hash}`)
  } else {
    win.loadFile(join(__dirname, '../../renderer/index.html'), { hash })
  }
}

export function registerEditorWindowHandlers(): void {
  ipcMain.handle('editor:open-window', (_event, moduleKey: string) => {
    openWindow(`editor-window?module=${encodeURIComponent(String(moduleKey ?? ''))}`, 'Modul-Editor', 1280, 860)
  })
}

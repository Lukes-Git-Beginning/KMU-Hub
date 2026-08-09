import { ipcMain, BrowserWindow } from 'electron'
import { join } from 'path'

/**
 * Modul-Editor (Customization v1) — opens the module editor in its own OS window
 * so the customer can edit customizations while using Cosmi in the main window
 * and compare draft vs. live side by side. Mirrors compose.ts / employee-wizard.ts.
 * The target module is passed as a hash query param.
 */

/**
 * One editor window per module (Darien 2026-08-05: "ich kann auch 5 oder 10
 * Editor-Fenster öffnen"). Several windows on the same module would each hold
 * their own draft of it — the last one to save would silently win. Clicking the
 * module again therefore raises the window that is already open.
 */
const openWindows = new Map<string, BrowserWindow>()

/**
 * Shared shell for the editor-family windows (same chrome, different route).
 * Returns whether a NEW window was created — the caller needs to know, because a
 * draft handover is left in shared storage for a window that is about to boot,
 * and a window that merely got focused would never read it (see the hub).
 */
function openWindow(hash: string, title: string, width: number, height: number, key: string): boolean {
  const existing = openWindows.get(key)
  if (existing && !existing.isDestroyed()) {
    if (existing.isMinimized()) existing.restore()
    existing.focus()
    return false
  }

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

  openWindows.set(key, win)
  win.on('closed', () => {
    // Only drop the entry if it is still ours — a reopen may have replaced it.
    if (openWindows.get(key) === win) openWindows.delete(key)
  })

  if (process.env.ELECTRON_RENDERER_URL) {
    win.loadURL(`${process.env.ELECTRON_RENDERER_URL}#/${hash}`)
  } else {
    win.loadFile(join(__dirname, '../../renderer/index.html'), { hash })
  }
  return true
}

export function registerEditorWindowHandlers(): void {
  ipcMain.handle('editor:open-window', (_event, moduleKey: string): boolean => {
    const key = String(moduleKey ?? '')
    // 1600 statt 1280: Leiste (240px) und Eigenschaften-Panel (~320px) gehen von
    // der Breite ab, der Rest ist die Modulvorschau. Bei 1280 blieben davon rund
    // 680px — zu wenig für eine achtspaltige Liste, ausgerechnet beim
    // Spalten-Konfigurieren (Darien 2026-08-09). Auf kleineren Bildschirmen
    // klemmt Electron das Fenster selbst auf die Arbeitsfläche.
    return openWindow(`editor-window?module=${encodeURIComponent(key)}`, 'Modul-Editor', 1600, 900, `editor:${key}`)
  })
}

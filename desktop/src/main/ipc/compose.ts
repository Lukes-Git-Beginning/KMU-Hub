import { ipcMain, BrowserWindow } from 'electron'
import { join } from 'path'

export function registerComposeHandlers(): void {
  ipcMain.handle('compose:open-window', (_event) => {
    const composeWin = new BrowserWindow({
      width: 680,
      height: 620,
      minWidth: 480,
      minHeight: 400,
      title: 'E-Mail verfassen',
      webPreferences: {
        preload: join(__dirname, '../../preload/index.js'),
        contextIsolation: true,
        nodeIntegration: false,
        sandbox: true,
        webSecurity: true,
      },
    })

    // Load same app with compose-window hash route
    if (process.env.ELECTRON_RENDERER_URL) {
      composeWin.loadURL(`${process.env.ELECTRON_RENDERER_URL}#/compose-window`)
    } else {
      composeWin.loadFile(join(__dirname, '../../renderer/index.html'), {
        hash: 'compose-window',
      })
    }
  })
}

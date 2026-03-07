import { ipcMain, BrowserWindow } from 'electron'
import { join } from 'path'

export function registerEmployeeWizardHandlers(): void {
  ipcMain.handle('employee-wizard:open-window', (_event) => {
    const composeWin = new BrowserWindow({
      width: 960,
      height: 720,
      minWidth: 720,
      minHeight: 520,
      title: 'Mitarbeiter erstellen',
      webPreferences: {
        preload: join(__dirname, '../../preload/index.js'),
        contextIsolation: true,
        nodeIntegration: false,
        sandbox: true,
        webSecurity: true,
      },
    })

    // Load same app with employee-wizard-window hash route
    if (process.env.ELECTRON_RENDERER_URL) {
      composeWin.loadURL(`${process.env.ELECTRON_RENDERER_URL}#/employee-wizard-window`)
    } else {
      composeWin.loadFile(join(__dirname, '../../renderer/index.html'), {
        hash: 'employee-wizard-window',
      })
    }
  })
}

import { ipcMain, app, BrowserWindow } from 'electron'
import { registerAuthHandlers } from './auth'
import { registerNotificationHandlers } from './notifications'
import { registerWindowHandlers } from './window'
import { registerComposeHandlers } from './compose'

export function registerIPCHandlers(getMainWindow: () => BrowserWindow | null): void {
  registerAuthHandlers()
  registerNotificationHandlers(getMainWindow)
  registerWindowHandlers()
  registerComposeHandlers()

  // App metadata handlers
  ipcMain.handle('app:version', () => app.getVersion())
  ipcMain.handle('app:platform', () => process.platform)
}

import { ipcMain, Notification, BrowserWindow } from 'electron'

export function registerNotificationHandlers(getMainWindow: () => BrowserWindow | null): void {
  ipcMain.handle('notification:show', async (_event, title: string, body: string) => {
    if (!Notification.isSupported()) {
      console.warn('[notifications] Native notifications not supported on this platform')
      return
    }

    const notification = new Notification({ title, body })

    notification.on('click', () => {
      const mainWindow = getMainWindow()
      if (mainWindow) {
        if (mainWindow.isMinimized()) {
          mainWindow.restore()
        }
        mainWindow.focus()
      }
    })

    notification.show()
  })
}

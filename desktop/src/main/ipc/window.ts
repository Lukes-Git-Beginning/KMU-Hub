import { ipcMain, BrowserWindow } from 'electron'

export function registerWindowHandlers(): void {
  ipcMain.on('window:minimize', (event) => {
    const window = BrowserWindow.fromWebContents(event.sender)
    window?.minimize()
  })

  ipcMain.on('window:maximize', (event) => {
    const window = BrowserWindow.fromWebContents(event.sender)
    if (window) {
      if (window.isMaximized()) {
        window.unmaximize()
      } else {
        window.maximize()
      }
    }
  })

  ipcMain.on('window:close', (event) => {
    const window = BrowserWindow.fromWebContents(event.sender)
    window?.close()
  })

  ipcMain.handle('window:is-maximized', (event) => {
    const window = BrowserWindow.fromWebContents(event.sender)
    return window?.isMaximized() ?? false
  })
}

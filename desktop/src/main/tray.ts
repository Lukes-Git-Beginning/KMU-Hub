import { Tray, Menu, app, BrowserWindow, nativeImage } from 'electron'
import { getResourcePath } from './utils'

let tray: Tray | null = null

export function setupTray(getMainWindow: () => BrowserWindow | null): void {
  const iconPath = getResourcePath('tray-icon.png')
  const icon = nativeImage.createFromPath(iconPath)

  if (icon.isEmpty()) {
    console.warn('[tray] No tray icon available, skipping tray setup')
    return
  }

  tray = new Tray(icon)
  tray.setToolTip('Cosmi')

  const contextMenu = Menu.buildFromTemplate([
    {
      label: 'Show/Hide',
      click: () => {
        const mainWindow = getMainWindow()
        if (mainWindow) {
          if (mainWindow.isVisible()) {
            mainWindow.hide()
          } else {
            mainWindow.show()
            mainWindow.focus()
          }
        }
      }
    },
    { type: 'separator' },
    {
      label: 'Quit',
      click: () => {
        app.quit()
      }
    }
  ])

  tray.setContextMenu(contextMenu)

  tray.on('click', () => {
    const mainWindow = getMainWindow()
    if (mainWindow) {
      if (mainWindow.isVisible()) {
        mainWindow.hide()
      } else {
        mainWindow.show()
        mainWindow.focus()
      }
    }
  })
}

export function destroyTray(): void {
  if (tray) {
    tray.destroy()
    tray = null
  }
}

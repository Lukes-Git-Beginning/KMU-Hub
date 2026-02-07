import { Tray, Menu, app, BrowserWindow, nativeImage } from 'electron'

let tray: Tray | null = null

export function setupTray(getMainWindow: () => BrowserWindow | null): void {
  // Create a minimal 16x16 tray icon (1px transparent PNG)
  // This will be replaced with a proper icon asset in a later plan
  const icon = nativeImage.createEmpty()

  // On some platforms, creating a tray with an empty icon may fail silently.
  // Skip tray setup if no icon is available.
  if (icon.isEmpty()) {
    console.warn('[tray] No tray icon available, skipping tray setup')
    return
  }

  tray = new Tray(icon)
  tray.setToolTip('KMU Hub')

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

import 'v8-compile-cache'
import { app, BrowserWindow, session, shell, nativeImage } from 'electron'
import { join } from 'path'
import { registerIPCHandlers } from './ipc/index'
import { createMenu } from './menu'
import { setupTray, destroyTray } from './tray'
import { getResourcePath } from './utils'

let mainWindow: BrowserWindow | null = null

function getMainWindow(): BrowserWindow | null {
  return mainWindow
}

function createWindow(): void {
  mainWindow = new BrowserWindow({
    width: 1400,
    height: 900,
    minWidth: 1024,
    minHeight: 700,
    show: false,
    title: 'Cosmi',
    icon: nativeImage.createFromPath(getResourcePath('app-icon.png')),
    webPreferences: {
      preload: join(__dirname, '../preload/index.js'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
      webSecurity: true
    }
  })

  // Show window when ready to prevent visual flash
  mainWindow.on('ready-to-show', () => {
    mainWindow?.show()
  })

  // Open external links in system browser, deny new window creation
  mainWindow.webContents.setWindowOpenHandler(({ url }) => {
    shell.openExternal(url)
    return { action: 'deny' }
  })

  mainWindow.on('closed', () => {
    mainWindow = null
  })

  // Load renderer: dev server in development, built files in production
  if (process.env.ELECTRON_RENDERER_URL) {
    mainWindow.loadURL(process.env.ELECTRON_RENDERER_URL)
  } else {
    mainWindow.loadFile(join(__dirname, '../renderer/index.html'))
  }
}

// Production API host. In packaged builds the renderer is served from file://
// (an opaque origin), so calls to the remote API are cross-origin and would be
// blocked by the gateway's CORS allowlist. We only talk to our own API host and
// authenticate with bearer tokens (no cookies), so it is safe to inject
// permissive CORS response headers for that host without weakening webSecurity.
// Dev runs against the vite dev-server (ELECTRON_RENDERER_URL set) and needs no patch.
const PROD_API_HOST = import.meta.env.MAIN_VITE_API_URL || 'https://app.zentria.tech'

function patchApiCors(): void {
  if (process.env.ELECTRON_RENDERER_URL || !PROD_API_HOST) return
  session.defaultSession.webRequest.onHeadersReceived((details, callback) => {
    if (!details.url.startsWith(PROD_API_HOST)) {
      callback({})
      return
    }
    callback({
      responseHeaders: {
        ...details.responseHeaders,
        'access-control-allow-origin': ['*'],
        'access-control-allow-headers': ['*'],
        'access-control-allow-methods': ['GET,POST,PUT,DELETE,PATCH,OPTIONS'],
      },
    })
  })
}

app.whenReady().then(() => {
  patchApiCors()
  registerIPCHandlers(getMainWindow)
  createMenu()
  createWindow()
  setupTray(getMainWindow)

  app.on('activate', () => {
    // macOS: re-create window when dock icon clicked and no windows open
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow()
    }
  })
})

app.on('window-all-closed', () => {
  destroyTray()
  // On macOS, apps stay active until explicitly quit
  if (process.platform !== 'darwin') {
    app.quit()
  }
})

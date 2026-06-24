import { contextBridge, ipcRenderer } from 'electron'
import type { ElectronAPI } from './types'

const api: ElectronAPI = {
  auth: {
    getStoredTokens: () => ipcRenderer.invoke('auth:get-tokens'),
    storeTokens: (tokens) => ipcRenderer.invoke('auth:store-tokens', tokens),
    clearTokens: () => ipcRenderer.invoke('auth:clear-tokens')
  },

  notifications: {
    show: (title, body) => ipcRenderer.invoke('notification:show', title, body)
  },

  window: {
    minimize: () => ipcRenderer.send('window:minimize'),
    maximize: () => ipcRenderer.send('window:maximize'),
    close: () => ipcRenderer.send('window:close'),
    isMaximized: () => ipcRenderer.invoke('window:is-maximized')
  },

  app: {
    getVersion: () => ipcRenderer.invoke('app:version'),
    getPlatform: () => ipcRenderer.invoke('app:platform')
  },

  deepLink: {
    onDeepLink: (callback: (url: string) => void) => {
      const handler = (_event: Electron.IpcRendererEvent, url: string) => {
        callback(url)
      }
      ipcRenderer.on('deep-link', handler)
      return () => {
        ipcRenderer.removeListener('deep-link', handler)
      }
    }
  },

  compose: {
    openWindow: () => ipcRenderer.invoke('compose:open-window')
  },

  employeeWizard: {
    openWindow: () => ipcRenderer.invoke('employee-wizard:open-window')
  },

  screenshare: {
    getSources: () => ipcRenderer.invoke('screenshare:get-sources'),
    setSource: (sourceId: string) => ipcRenderer.invoke('screenshare:set-source', sourceId)
  }
}

contextBridge.exposeInMainWorld('electronAPI', api)

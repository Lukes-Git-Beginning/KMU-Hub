/** Type definitions for the Electron IPC bridge exposed to the renderer process. */

export interface TokenPair {
  accessToken: string
  refreshToken: string
}

export interface ElectronAPI {
  /** Auth: retrieve stored tokens (decrypted via safeStorage) */
  auth: {
    getStoredTokens: () => Promise<TokenPair | null>
    storeTokens: (tokens: TokenPair) => Promise<void>
    clearTokens: () => Promise<void>
  }

  /** Notifications: trigger native OS notifications */
  notifications: {
    show: (title: string, body: string) => Promise<void>
  }

  /** Window: frameless window controls (minimize, maximize, close) */
  window: {
    minimize: () => void
    maximize: () => void
    close: () => void
    isMaximized: () => Promise<boolean>
  }

  /** App: metadata and platform info */
  app: {
    getVersion: () => Promise<string>
    getPlatform: () => Promise<NodeJS.Platform>
  }

  /** Deep link: listen for protocol handler callbacks */
  deepLink: {
    onDeepLink: (callback: (url: string) => void) => () => void
  }
}

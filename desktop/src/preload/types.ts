/** Type definitions for the Electron IPC bridge exposed to the renderer process. */

export interface TokenPair {
  accessToken: string
  refreshToken: string
}

/** Screen capture source (Wave 4.1 — Electron screen-source picker). */
export interface ScreenSourceInfo {
  id: string
  name: string
  /** PNG data URL of the source thumbnail. Empty string when unavailable. */
  thumbnailDataUrl: string
  /** 'screen' for full display, 'window' for application window */
  type: 'screen' | 'window'
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

  /** Compose: open compose in a separate OS window */
  compose: {
    openWindow: () => Promise<void>
  }

  /** Employee wizard: open employee creation wizard in a separate OS window */
  employeeWizard: {
    openWindow: () => Promise<void>
  }

  /** Module editor: open the customization editor for a module in its own window */
  editor: {
    openWindow: (moduleKey: string) => Promise<void>
  }

  /**
   * Screenshare: enumerate capture sources for the in-call source-picker (Wave 4.1).
   * Returns all available screens and windows with thumbnail previews.
   * setSource must be called with the chosen sourceId immediately before
   * setScreenShareEnabled() so the main-process handler can resolve the
   * source synchronously (avoids async race in setDisplayMediaRequestHandler).
   */
  screenshare: {
    getSources: () => Promise<ScreenSourceInfo[]>
    setSource: (sourceId: string) => Promise<void>
  }
}

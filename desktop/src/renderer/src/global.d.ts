import type { ElectronAPI } from '../../preload/types'

declare global {
  interface Window {
    /**
     * The preload bridge. Optional on purpose: the same renderer is served as a
     * plain web app, where no preload script runs and this is undefined. Typing
     * it as always-present is what let unguarded calls reach the browser build
     * and throw at runtime -- see lib/platform.ts for the guarded accessors.
     */
    electronAPI?: ElectronAPI
  }
}

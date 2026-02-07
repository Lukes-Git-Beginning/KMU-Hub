/**
 * Hooks for Electron IPC bridge communication.
 *
 * Provides type-safe access to native OS features exposed
 * through the preload contextBridge.
 */
import { useCallback, useEffect, useState } from 'react'

/**
 * Send native OS notifications via Electron Notification API.
 */
export function useNativeNotification() {
  const showNotification = useCallback((title: string, body: string) => {
    window.electronAPI.notifications.show(title, body)
  }, [])

  return { showNotification }
}

/**
 * Control the native window (minimize, maximize, close).
 */
export function useWindowControls() {
  const [isMaximized, setIsMaximized] = useState(false)

  useEffect(() => {
    window.electronAPI.window.isMaximized().then(setIsMaximized)
  }, [])

  const minimize = useCallback(() => {
    window.electronAPI.window.minimize()
  }, [])

  const maximize = useCallback(() => {
    window.electronAPI.window.maximize()
    // Toggle the local state optimistically
    setIsMaximized((prev) => !prev)
  }, [])

  const close = useCallback(() => {
    window.electronAPI.window.close()
  }, [])

  return { minimize, maximize, close, isMaximized }
}

/**
 * Get the current OS platform (win32, darwin, linux).
 */
export function usePlatform() {
  const [platform, setPlatform] = useState<NodeJS.Platform | null>(null)

  useEffect(() => {
    window.electronAPI.app.getPlatform().then(setPlatform)
  }, [])

  return platform
}

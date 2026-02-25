/**
 * Outermost layout wrapper.
 *
 * Manages dark mode, color theme, and accent intensity on <html>,
 * then renders the AppShell fullscreen.
 */
import { useEffect, useSyncExternalStore } from 'react'
import { useUIStore } from '@/stores/ui'
import { useSettingsStore } from '@/stores/settings'
import { AppShell } from './AppShell'

// Detect system dark mode preference reactively
const darkQuery = window.matchMedia('(prefers-color-scheme: dark)')
function subscribeSystemTheme(cb: () => void) {
  darkQuery.addEventListener('change', cb)
  return () => darkQuery.removeEventListener('change', cb)
}
function getSystemIsDark() {
  return darkQuery.matches
}

export function DeskEnvironment() {
  const storeTheme = useUIStore((s) => s.theme)
  const colorTheme = useUIStore((s) => s.colorTheme)
  const accentIntensity = useUIStore((s) => s.accentIntensity)
  const windowStyle = useUIStore((s) => s.windowStyle)
  const fontSize = useSettingsStore((s) => s.appearance.fontSize)
  const systemIsDark = useSyncExternalStore(subscribeSystemTheme, getSystemIsDark)

  const isDark = storeTheme === 'auto' ? systemIsDark : storeTheme === 'dark'

  // Sync .dark class on <html>
  useEffect(() => {
    document.documentElement.classList.toggle('dark', isDark)
  }, [isDark])

  // Remove any leftover glass/crystal classes
  useEffect(() => {
    document.documentElement.classList.remove('ui-glass', 'ui-crystal')
  }, [])

  // Sync color theme class on <html> (graphit = default, no class needed)
  useEffect(() => {
    document.documentElement.classList.remove(
      'theme-sand', 'theme-ozean', 'theme-lavendel',
      'theme-wald', 'theme-rose', 'theme-mitternacht', 'theme-terrakotta'
    )
    if (colorTheme !== 'graphit') {
      document.documentElement.classList.add(`theme-${colorTheme}`)
    }
  }, [colorTheme])

  // Sync accent intensity class on <html>
  useEffect(() => {
    document.documentElement.classList.toggle('accent-vivid', accentIntensity === 'vivid')
  }, [accentIntensity])

  // Sync font size on <html>
  useEffect(() => {
    document.documentElement.style.fontSize = `${fontSize}px`
  }, [fontSize])

  const isBubble = windowStyle === 'bubble'

  return (
    <div className={`h-screen w-screen overflow-hidden relative ${isBubble ? 'bg-neutral-200 dark:bg-neutral-900 p-3' : 'bg-background'}`}>
      <div className={isBubble ? 'h-full w-full overflow-hidden rounded-2xl bg-background shadow-2xl' : 'h-full w-full'}>
        <AppShell />
      </div>
    </div>
  )
}

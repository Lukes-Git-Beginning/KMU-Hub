/**
 * Outermost layout wrapper.
 *
 * Manages dark mode, color theme, accent intensity, glass mode,
 * and desk background on <html>, then renders the AppShell fullscreen.
 */
import { useEffect, useSyncExternalStore } from 'react'
import { useUIStore } from '@/stores/ui'
import { useSettingsStore } from '@/stores/settings'
import { useHydrateModuleSettings } from '@/hooks/useHydrateModuleSettings'
import { AppShell } from './AppShell'

// Background gradient presets
// eslint-disable-next-line react-refresh/only-export-components
export const DESK_BACKGROUNDS: Record<string, { label: string; css: string }> = {
  'gradient-warm': {
    label: 'Sonnenuntergang',
    css: 'linear-gradient(135deg, #f5af19 0%, #f12711 50%, #c41a5a 100%)',
  },
  'gradient-cool': {
    label: 'Ozean',
    css: 'linear-gradient(135deg, #667eea 0%, #764ba2 50%, #6B8DD6 100%)',
  },
  'gradient-forest': {
    label: 'Wald',
    css: 'linear-gradient(135deg, #134E5E 0%, #71B280 50%, #2C7744 100%)',
  },
  'gradient-sunset': {
    label: 'Abendhimmel',
    css: 'linear-gradient(135deg, #ee9ca7 0%, #ffdde1 30%, #f9a8d4 60%, #c084fc 100%)',
  },
  'gradient-night': {
    label: 'Nacht',
    css: 'linear-gradient(135deg, #0f0c29 0%, #302b63 50%, #24243e 100%)',
  },
  'gradient-minimal': {
    label: 'Dezent',
    css: 'linear-gradient(135deg, #e0e0e0 0%, #c9d6dd 50%, #d5dbe0 100%)',
  },
}

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
  const uiLook = useUIStore((s) => s.uiLook)
  const deskBackground = useUIStore((s) => s.deskBackground)
  const fontSize = useSettingsStore((s) => s.appearance.fontSize)
  const systemIsDark = useSyncExternalStore(subscribeSystemTheme, getSystemIsDark)

  const isDark = storeTheme === 'auto' ? systemIsDark : storeTheme === 'dark'

  // Hydrate backend-persisted module preferences once after login (X-4).
  useHydrateModuleSettings()

  // Sync .dark class on <html>
  useEffect(() => {
    document.documentElement.classList.toggle('dark', isDark)
  }, [isDark])

  // Sync glass mode on <html>
  useEffect(() => {
    document.documentElement.classList.toggle('ui-glass', uiLook === 'glass')
    // Remove crystal if it was ever set (we only support solid/glass now)
    document.documentElement.classList.remove('ui-crystal')
  }, [uiLook])

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
  const showBackground = uiLook === 'glass' && deskBackground && DESK_BACKGROUNDS[deskBackground]
  const bgCss = showBackground ? DESK_BACKGROUNDS[deskBackground!].css : undefined

  return (
    <div className={`h-screen w-screen overflow-hidden relative ${isBubble ? 'bg-neutral-200 dark:bg-neutral-900 p-3' : 'bg-background'}`}>
      {/* Background layer — visible through frosted glass */}
      {bgCss && (
        <div
          className="absolute inset-0 z-0 pointer-events-none"
          style={{ background: bgCss }}
        />
      )}
      <div className={`relative z-10 ${isBubble ? 'h-full w-full overflow-hidden rounded-2xl shadow-2xl' : 'h-full w-full'}`}>
        <AppShell />
      </div>
    </div>
  )
}

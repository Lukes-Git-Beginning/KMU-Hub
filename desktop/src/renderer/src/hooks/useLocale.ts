/**
 * Hook for locale operations and formatting utilities.
 *
 * Provides the effective locale (user choice -> browser detection -> DE fallback),
 * locale switching, and metadata about available languages.
 */
import { useMemo, useCallback } from 'react'
import {
  useLocaleStore,
  SUPPORTED_LOCALES,
  DEFAULT_LOCALE,
  type SupportedLocale,
} from '@/stores/locale'

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

/** Display names for each supported locale. */
export const LOCALE_DISPLAY_NAMES: Record<SupportedLocale, string> = {
  de: 'Deutsch',
  en: 'English',
  fr: 'Francais',
  it: 'Italiano',
}

/** Available locales with code and display name, for use in locale pickers. */
export const AVAILABLE_LOCALES = SUPPORTED_LOCALES.map((code) => ({
  code,
  name: LOCALE_DISPLAY_NAMES[code],
}))

// ---------------------------------------------------------------------------
// Browser detection
// ---------------------------------------------------------------------------

/**
 * Detect the user's preferred language from the browser.
 * Returns a supported locale or the default (DE) if not supported.
 */
function detectBrowserLocale(): SupportedLocale {
  try {
    const browserLang = navigator.language.split('-')[0].toLowerCase()
    if ((SUPPORTED_LOCALES as readonly string[]).includes(browserLang)) {
      return browserLang as SupportedLocale
    }
  } catch {
    // navigator.language may not be available in all environments
  }
  return DEFAULT_LOCALE
}

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------

export interface UseLocaleReturn {
  /** The effective locale in use (user choice, or auto-detected). */
  locale: SupportedLocale
  /** Set an explicit locale preference. */
  setLocale: (locale: SupportedLocale) => void
  /** Reset to auto-detect mode (clears user preference). */
  resetToAuto: () => void
  /** All available locales with code and display name. */
  availableLocales: typeof AVAILABLE_LOCALES
  /** True when the locale is auto-detected (no explicit user choice). */
  isAutoDetected: boolean
}

/**
 * Provides the current effective locale and switching utilities.
 *
 * Fallback chain: user explicit choice -> browser language -> DE
 */
export function useLocale(): UseLocaleReturn {
  const storeLocale = useLocaleStore((s) => s.locale)
  const setLocale = useLocaleStore((s) => s.setLocale)
  const resetLocale = useLocaleStore((s) => s.resetLocale)

  const isAutoDetected = storeLocale === null

  const locale = useMemo<SupportedLocale>(() => {
    if (storeLocale !== null) {
      return storeLocale
    }
    return detectBrowserLocale()
  }, [storeLocale])

  const resetToAuto = useCallback(() => {
    resetLocale()
  }, [resetLocale])

  return {
    locale,
    setLocale,
    resetToAuto,
    availableLocales: AVAILABLE_LOCALES,
    isAutoDetected,
  }
}

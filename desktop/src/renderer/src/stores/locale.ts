/**
 * Locale preference store (Zustand with persistence).
 *
 * Stores the user's explicit language choice. When locale is null,
 * the app auto-detects the browser language with fallback to German.
 *
 * Persistence: localStorage key 'cosmi-locale' so the preference
 * survives app restart without requiring a backend round-trip.
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

/** Supported locales in the application. */
export const SUPPORTED_LOCALES = ['de', 'en', 'fr', 'it'] as const

export type SupportedLocale = (typeof SUPPORTED_LOCALES)[number]

/** Default locale when browser detection fails or returns unsupported language. */
export const DEFAULT_LOCALE: SupportedLocale = 'de'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface LocaleState {
  /** User's explicit locale choice. null means auto-detect from browser. */
  locale: SupportedLocale | null

  /** Set an explicit locale preference. */
  setLocale: (locale: SupportedLocale) => void

  /** Reset to auto-detect mode (clears explicit choice). */
  resetLocale: () => void
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useLocaleStore = create<LocaleState>()(
  persist(
    (set) => ({
      locale: null,

      setLocale: (locale: SupportedLocale) => set({ locale }),

      resetLocale: () => set({ locale: null }),
    }),
    {
      name: 'cosmi-locale',
    },
  ),
)

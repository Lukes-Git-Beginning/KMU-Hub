/**
 * Locale preference store (Zustand with persistence + backend sync).
 *
 * Stores the user's explicit language choice. When locale is null,
 * the app auto-detects the browser language with fallback to German.
 *
 * Persistence: Zustand persist writes to localStorage key 'cosmi-locale'
 * for fast startup. On each explicit user change, the preference is also
 * written to the backend (PUT /api/v1/settings/language/user, key "locale")
 * so it survives across devices / browsers.
 *
 * On app boot, initFromServer() in settings.ts reads the language module
 * and calls setLocaleQuiet() here so localStorage is skipped and no PUT
 * is triggered (avoids ping-pong).
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
// Backend sync helper (fire-and-forget, non-critical)
// ---------------------------------------------------------------------------

async function persistLocaleToServer(locale: SupportedLocale): Promise<void> {
  try {
    const { authenticatedRequest } = await import('@/api/utils/authenticatedFetch')
    await authenticatedRequest({
      method: 'PUT',
      path: '/api/v1/settings/language/user',
      body: { settings: { locale } },
    })
  } catch {
    // lean: silent; localStorage value already applied; next initFromServer re-syncs
  }
}

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface LocaleState {
  /** User's explicit locale choice. null means auto-detect from browser. */
  locale: SupportedLocale | null

  /** Set an explicit locale preference (user action) — writes to localStorage + backend. */
  setLocale: (locale: SupportedLocale) => void

  /**
   * Set locale without triggering a backend PUT.
   * Used by settings.ts initFromServer() to hydrate from server value.
   */
  setLocaleQuiet: (locale: SupportedLocale) => void

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

      setLocale: (locale: SupportedLocale) => {
        set({ locale })
        void persistLocaleToServer(locale)
      },

      // lean: no PUT on quiet set; upgrade to PUT if cross-device sync of auto-resets is needed
      setLocaleQuiet: (locale: SupportedLocale) => set({ locale }),

      resetLocale: () => set({ locale: null }),
    }),
    {
      name: 'cosmi-locale',
    },
  ),
)

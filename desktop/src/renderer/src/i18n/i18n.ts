/**
 * i18next initialization for Cosmi Desktop.
 *
 * - keySeparator: false -- keeps flat dot-notation keys ("security.2fa.title")
 * - ICU plugin -- parses existing {count, plural, ...} syntax from react-intl era
 * - Static imports -- all 4 locale bundles bundled (small total size)
 * - No backend/lazy loading -- Electron app, all resources local
 *
 * ICU cache invalidation (v1.2 customization fix):
 *   i18next-icu memoizes compiled formatters keyed by `lng.ns.key` and ships
 *   with bindI18n/bindI18nStore = '' by default -- meaning its cache is NEVER
 *   cleared. Runtime label overrides (addResourceBundle from the Anpassungen
 *   editor) update the resource store but the sidebar / any already-rendered
 *   t() surface kept returning the stale compiled default. Binding clearCache
 *   to the store 'added'/'removed' events (fired by addResourceBundle) plus
 *   'languageChanged' makes overrides take effect live, without a rebuild.
 */
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import ICU from 'i18next-icu'

import messagesDE from '@/i18n/messages/de.json'
import messagesEN from '@/i18n/messages/en.json'
import messagesFR from '@/i18n/messages/fr.json'
import messagesIT from '@/i18n/messages/it.json'

import type { SupportedLocale } from '@/stores/locale'

export function initI18n(locale: SupportedLocale): typeof i18n {
  if (i18n.isInitialized) return i18n

  i18n
    // bindI18nStore 'added removed' clears the ICU memoization cache whenever
    // addResourceBundle mutates the store (runtime label overrides); bindI18n
    // 'languageChanged' covers locale switches. Without these the cache is
    // permanent (i18next-icu default '') and overrides never surface live.
    .use(new ICU({ bindI18nStore: 'added removed', bindI18n: 'languageChanged' }))
    .use(initReactI18next)
    .init({
      lng: locale,
      fallbackLng: 'de',
      keySeparator: false,
      nsSeparator: false,
      interpolation: {
        escapeValue: false,
      },
      resources: {
        de: { translation: messagesDE },
        en: { translation: messagesEN },
        fr: { translation: messagesFR },
        it: { translation: messagesIT },
      },
    })

  return i18n
}

export { i18n }

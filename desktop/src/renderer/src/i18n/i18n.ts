/**
 * i18next initialization for Cosmi Desktop.
 *
 * - keySeparator: false -- keeps flat dot-notation keys ("security.2fa.title")
 * - ICU plugin -- parses existing {count, plural, ...} syntax from react-intl era
 * - Static imports -- all 4 locale bundles bundled (small total size)
 * - No backend/lazy loading -- Electron app, all resources local
 */
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import ICU from 'i18next-icu'

import messagesDE from '@/i18n/messages/de.json'
import messagesEN from '@/i18n/messages/en.json'
import messagesFR from '@/i18n/messages/fr.json'
import messagesIT from '@/i18n/messages/it.json'

// Module-scoped addition bundles (merged into de during i18n migration)
import additionsCRM from '@/i18n/additions/crm.json'
import additionsDashboard from '@/i18n/additions/dashboard.json'
import additionsWork from '@/i18n/additions/work.json'

import type { SupportedLocale } from '@/stores/locale'

/** Merge addition bundles into the base DE messages. */
const mergedDE = {
  ...messagesDE,
  ...additionsCRM,
  ...additionsDashboard,
  ...additionsWork,
}

export function initI18n(locale: SupportedLocale): typeof i18n {
  if (i18n.isInitialized) return i18n

  i18n
    .use(ICU)
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
        de: { translation: mergedDE },
        en: { translation: messagesEN },
        fr: { translation: messagesFR },
        it: { translation: messagesIT },
      },
    })

  return i18n
}

export { i18n }

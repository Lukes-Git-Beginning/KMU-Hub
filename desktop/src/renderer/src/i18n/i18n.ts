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
import additionsFinanzen from '@/i18n/additions/finanzen.json'
import additionsKommunikation from '@/i18n/additions/kommunikation.json'
import additionsProfil from '@/i18n/additions/profil.json'
import additionsSettings from '@/i18n/additions/settings.json'
import additionsTeam from '@/i18n/additions/team.json'
import additionsWork from '@/i18n/additions/work.json'
import additionsKontakte from '@/i18n/additions/kontakte.json'
import additionsKontakte2 from '@/i18n/additions/kontakte-2.json'
import additionsChat from '@/i18n/additions/chat.json'
import additionsWiki from '@/i18n/additions/wiki.json'
import additionsAutomatisierung from '@/i18n/additions/automatisierung.json'
import additionsDokumente from '@/i18n/additions/dokumente.json'
import additionsAdmin from '@/i18n/additions/admin.json'
import additionsMeetings from '@/i18n/additions/meetings.json'
import additionsMails from '@/i18n/additions/mails.json'
import additionsHelpdesk from '@/i18n/additions/helpdesk.json'
import additionsBuchhaltung from '@/i18n/additions/buchhaltung.json'
import additionsKalender from '@/i18n/additions/kalender.json'
import additionsRapporte from '@/i18n/additions/rapporte.json'
import additionsNotifications from '@/i18n/additions/notifications.json'

// Step 3.5: Straggler modules (Wave 1)
import additionsFuhrpark from '@/i18n/additions/fuhrpark.json'
import additionsEinkauf from '@/i18n/additions/einkauf.json'
import additionsInventar from '@/i18n/additions/inventar.json'
import additionsVermietung from '@/i18n/additions/vermietung.json'
import additionsVertraege from '@/i18n/additions/vertraege.json'
import additionsProduktion from '@/i18n/additions/produktion.json'
import additionsFormulare from '@/i18n/additions/formulare.json'
import additionsSchichten from '@/i18n/additions/schichten.json'
import additionsBerichte from '@/i18n/additions/berichte.json'
import additionsVideo from '@/i18n/additions/video.json'

// Component-scoped addition bundles (Step 3: components/ directories)
import additionsLayoutCore from '@/i18n/additions/components-layout-core.json'
import additionsLayoutNav from '@/i18n/additions/components-layout-nav.json'
import additionsHeaderWidgets from '@/i18n/additions/components-header-widgets.json'
import additionsHeaderMenus from '@/i18n/additions/components-header-menus.json'
import additionsHeaderTime from '@/i18n/additions/components-header-time.json'
import additionsSharedSearchEditor from '@/i18n/additions/components-shared-search-editor.json'
import additionsSharedMisc from '@/i18n/additions/components-shared-misc.json'
import additionsWidgets from '@/i18n/additions/components-widgets.json'
import additionsChatDeskOnboarding from '@/i18n/additions/components-chat-desk-onboarding.json'

import type { SupportedLocale } from '@/stores/locale'

/** Merge addition bundles into the base DE messages. */
const mergedDE = {
  ...messagesDE,
  ...additionsCRM,
  ...additionsDashboard,
  ...additionsFinanzen,
  ...additionsKommunikation,
  ...additionsProfil,
  ...additionsSettings,
  ...additionsTeam,
  ...additionsWork,
  ...additionsKontakte,
  ...additionsKontakte2,
  ...additionsChat,
  ...additionsWiki,
  ...additionsAutomatisierung,
  ...additionsDokumente,
  ...additionsAdmin,
  ...additionsMeetings,
  ...additionsMails,
  ...additionsHelpdesk,
  ...additionsBuchhaltung,
  ...additionsKalender,
  ...additionsRapporte,
  ...additionsNotifications,
  // Step 3.5: Straggler modules (Wave 1)
  ...additionsFuhrpark,
  ...additionsEinkauf,
  ...additionsInventar,
  ...additionsVermietung,
  ...additionsVertraege,
  ...additionsProduktion,
  ...additionsFormulare,
  ...additionsSchichten,
  ...additionsBerichte,
  ...additionsVideo,
  // Component additions (Step 3)
  ...additionsLayoutCore,
  ...additionsLayoutNav,
  ...additionsHeaderWidgets,
  ...additionsHeaderMenus,
  ...additionsHeaderTime,
  ...additionsSharedSearchEditor,
  ...additionsSharedMisc,
  ...additionsWidgets,
  ...additionsChatDeskOnboarding,
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

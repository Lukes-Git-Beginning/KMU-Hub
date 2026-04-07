import 'i18next'
import type de from '@/i18n/messages/de.json'

declare module 'i18next' {
  interface CustomTypeOptions {
    defaultNS: 'translation'
    returnNull: false
    resources: {
      translation: typeof de
    }
  }
}

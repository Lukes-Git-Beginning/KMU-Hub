import 'i18next'

declare module 'i18next' {
  interface CustomTypeOptions {
    defaultNS: 'translation'
    // Strict key typing disabled during migration — will be re-enabled
    // after all keys are added to de.json
    returnNull: false
  }
}

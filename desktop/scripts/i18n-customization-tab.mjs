/**
 * i18n-customization-tab.mjs — AdminHubPage Tab-Key für „Anpassungen".
 *
 * ADD: admin.hub.tabs.customization (9. Tab in AdminHubPage)
 *
 * Konventionen:
 *   - Du-Form (Cosmi duzt)
 *   - `{var}` NICHT `{{var}}` (ICU-Plugin)
 *   - ECHTE fr+it-Übersetzungen
 *
 * Ausführen (einmalig) aus desktop/: node scripts/i18n-customization-tab.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))
const msgDir = join(here, '..', 'src', 'renderer', 'src', 'i18n', 'messages')

const ADD = {
  'admin.hub.tabs.customization': {
    de: 'Anpassungen',
    en: 'Customization',
    fr: 'Personnalisation',
    it: 'Personalizzazione',
  },
}

for (const lang of ['de', 'en', 'fr', 'it']) {
  const file = join(msgDir, `${lang}.json`)
  const messages = JSON.parse(readFileSync(file, 'utf8'))
  let added = 0
  for (const [key, byLang] of Object.entries(ADD)) {
    if (key in messages) throw new Error(`${lang}: ADD key already exists: ${key}`)
    messages[key] = byLang[lang]
    added += 1
  }
  writeFileSync(file, JSON.stringify(messages, null, 2) + '\n', 'utf8')
  console.log(`${lang}.json — +${added} added`)
}
console.log('done — customization-tab i18n pass complete')

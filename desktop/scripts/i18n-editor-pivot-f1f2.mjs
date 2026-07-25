/**
 * Editor-Pivot Feedback F1/F2 (Wertelisten-Farben + Status-Value-Set) i18n pass.
 * Ausführen aus desktop/: node scripts/i18n-editor-pivot-f1f2.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))
const msgDir = join(here, '..', 'src', 'renderer', 'src', 'i18n', 'messages')

const ADD = {
  'customization.editor.wertelisten.addOption': { de: 'Neue Option', en: 'Add option', fr: 'Ajouter une option', it: 'Aggiungi opzione' },
  'customization.editor.wertelisten.newOption': { de: 'Neue Option', en: 'New option', fr: 'Nouvelle option', it: 'Nuova opzione' },
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
console.log('done')

/**
 * Modul-Editor E-3 i18n pass — Begriffe-Panel (module-scoped label editor).
 * Ausführen aus desktop/: node scripts/i18n-editor-e3.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))
const msgDir = join(here, '..', 'src', 'renderer', 'src', 'i18n', 'messages')

const ADD = {
  'customization.editor.begriffe.reset': {
    de: 'Zurücksetzen',
    en: 'Reset',
    fr: 'Réinitialiser',
    it: 'Ripristina',
  },
  'customization.editor.begriffe.standardHint': {
    de: 'Standard: {value}',
    en: 'Default: {value}',
    fr: 'Défaut : {value}',
    it: 'Predefinito: {value}',
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
console.log('done')

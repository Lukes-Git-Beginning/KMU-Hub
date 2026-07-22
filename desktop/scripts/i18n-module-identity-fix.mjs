/**
 * Module-identity lockdown (Darien 2026-07-22) i18n pass — empty-state copy for
 * modules that expose no customizable content terms in the Begriffe editor.
 * Ausführen aus desktop/: node scripts/i18n-module-identity-fix.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))
const msgDir = join(here, '..', 'src', 'renderer', 'src', 'i18n', 'messages')

const ADD = {
  'customization.editor.begriffe.noneForModule': {
    de: 'Für dieses Modul gibt es aktuell keine anpassbaren Begriffe. Passe stattdessen Wertelisten oder Felder an.',
    en: 'This module has no customizable terms yet. Adjust value lists or fields instead.',
    fr: 'Ce module n’a pas encore de termes personnalisables. Modifie plutôt les listes de valeurs ou les champs.',
    it: 'Questo modulo non ha ancora termini personalizzabili. Modifica invece gli elenchi di valori o i campi.',
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

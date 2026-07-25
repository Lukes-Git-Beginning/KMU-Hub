/**
 * Editor-Pivot G1 (Werteliste-Option löschen + Reassignment) i18n pass.
 * Ausführen aus desktop/: node scripts/i18n-editor-pivot-g1.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))
const msgDir = join(here, '..', 'src', 'renderer', 'src', 'i18n', 'messages')

const ADD = {
  'customization.editor.wertelisten.deleteOption': {
    de: '„{option}" löschen',
    en: 'Delete “{option}”',
    fr: 'Supprimer « {option} »',
    it: 'Elimina «{option}»',
  },
  'customization.editor.wertelisten.reassignTitle': {
    de: '„{option}" entfernen',
    en: 'Remove “{option}”',
    fr: 'Retirer « {option} »',
    it: 'Rimuovi «{option}»',
  },
  'customization.editor.wertelisten.reassignBody': {
    de: 'Bestehende Einträge mit diesem Wert werden geändert auf:',
    en: 'Existing records with this value will be changed to:',
    fr: 'Les enregistrements existants avec cette valeur passeront à :',
    it: 'I record esistenti con questo valore verranno cambiati in:',
  },
  'customization.editor.wertelisten.reassignConfirm': { de: 'Entfernen', en: 'Remove', fr: 'Retirer', it: 'Rimuovi' },
  'customization.editor.wertelisten.removedTitle': { de: 'Entfernt', en: 'Removed', fr: 'Retiré', it: 'Rimosso' },
}

for (const lang of ['de', 'en', 'fr', 'it']) {
  const file = join(msgDir, `${lang}.json`)
  const messages = JSON.parse(readFileSync(file, 'utf8'))
  let added = 0
  for (const [key, byLang] of Object.entries(ADD)) {
    if (key in messages) continue // idempotent: skip keys already present
    messages[key] = byLang[lang]
    added += 1
  }
  writeFileSync(file, JSON.stringify(messages, null, 2) + '\n', 'utf8')
  console.log(`${lang}.json — +${added} added`)
}
console.log('done')

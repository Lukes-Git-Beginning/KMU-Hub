/**
 * Modul-Editor E-3c i18n pass — Felder-Panel (module-scoped custom-field editor)
 * + the helpdesk_ticket entity label.
 * Ausführen aus desktop/: node scripts/i18n-editor-e3c.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))
const msgDir = join(here, '..', 'src', 'renderer', 'src', 'i18n', 'messages')

const ADD = {
  'customization.fields.entity.helpdesk_ticket': {
    de: 'Tickets',
    en: 'Tickets',
    fr: 'Tickets',
    it: 'Ticket',
  },
  'customization.editor.felder.badgeNew': {
    de: 'Neu',
    en: 'New',
    fr: 'Nouveau',
    it: 'Nuovo',
  },
  'customization.editor.felder.badgeChanged': {
    de: 'Geändert',
    en: 'Changed',
    fr: 'Modifié',
    it: 'Modificato',
  },
  'customization.editor.felder.removedTitle': {
    de: 'Wird entfernt',
    en: 'Being removed',
    fr: 'Sera supprimé',
    it: 'In rimozione',
  },
  'customization.editor.felder.restore': {
    de: 'Wiederherstellen',
    en: 'Restore',
    fr: 'Restaurer',
    it: 'Ripristina',
  },
  'customization.editor.felder.stagedHint': {
    de: 'Feld-Änderungen gehen erst beim Übernehmen live.',
    en: 'Field changes only go live once you apply them.',
    fr: 'Les modifications de champs ne sont actives qu’après application.',
    it: 'Le modifiche ai campi diventano attive solo dopo l’applicazione.',
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

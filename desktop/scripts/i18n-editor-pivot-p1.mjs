/**
 * Editor-Pivot P1 (Helpdesk-Pilot) i18n pass.
 * - helpdesk.tabs.ticketsLabel: der Reiter-Titel OHNE Zähler (Zähler wird separat
 *   gerendert, damit der Reiter per EditableText umbenennbar ist).
 * - customization.editor.actionBlocked: Hinweis, wenn eine Aktion im Editor no-op ist.
 * Ausführen aus desktop/: node scripts/i18n-editor-pivot-p1.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))
const msgDir = join(here, '..', 'src', 'renderer', 'src', 'i18n', 'messages')

const ADD = {
  'helpdesk.tabs.ticketsLabel': { de: 'Tickets', en: 'Tickets', fr: 'Tickets', it: 'Ticket' },
  'customization.editor.actionBlocked': {
    de: 'Im Editor deaktiviert — hier passt du das Modul nur an.',
    en: 'Disabled in the editor — here you only customize the module.',
    fr: 'Désactivé dans l’éditeur — ici tu personnalises seulement le module.',
    it: 'Disattivato nell’editor — qui personalizzi solo il modulo.',
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

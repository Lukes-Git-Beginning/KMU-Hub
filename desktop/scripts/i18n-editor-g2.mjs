/**
 * Editor-Pivot G2 i18n pass (Zusatzfelder / custom fields visible in-place).
 *
 * ADD: 2 keys x4 — new-ticket select placeholder + Felder-panel intro line.
 * OVERRIDE: rename the editor nav label "Felder" → "Zusatzfelder" (self-
 * explaining + consistent with the module's "Zusatzfelder" detail section).
 *
 * Du-Form (Cosmi standard), real umlauts, real fr/it. ICU single braces {var}.
 *
 * Run once from desktop/: node scripts/i18n-editor-g2.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))
const msgDir = join(here, '..', 'src', 'renderer', 'src', 'i18n', 'messages')

const ADD = {
  'helpdesk.newTicket.selectPlaceholder': {
    de: 'Bitte wählen…', en: 'Please select…', fr: 'Veuillez sélectionner…', it: 'Seleziona…',
  },
  'customization.editor.felder.intro': {
    de: 'Zusatzfelder sind eigene Felder, die im Detail und im Anlege-Formular des Moduls erscheinen.',
    en: "Extra fields are your own fields that appear in the module's detail and creation form.",
    fr: 'Les champs supplémentaires sont vos propres champs qui apparaissent dans le détail et le formulaire de création du module.',
    it: 'I campi aggiuntivi sono campi personalizzati che compaiono nel dettaglio e nel modulo di creazione.',
  },
}

const OVERRIDE = {
  'customization.editor.nav.fields': {
    de: 'Zusatzfelder', en: 'Extra fields', fr: 'Champs supplémentaires', it: 'Campi aggiuntivi',
  },
}

for (const lang of ['de', 'en', 'fr', 'it']) {
  const file = join(msgDir, `${lang}.json`)
  const messages = JSON.parse(readFileSync(file, 'utf8'))
  let added = 0
  let changed = 0
  for (const [key, byLang] of Object.entries(ADD)) {
    if (key in messages) throw new Error(`${lang}: ADD key already exists: ${key}`)
    messages[key] = byLang[lang]
    added += 1
  }
  for (const [key, byLang] of Object.entries(OVERRIDE)) {
    if (!(key in messages)) throw new Error(`${lang}: OVERRIDE key missing: ${key}`)
    messages[key] = byLang[lang]
    changed += 1
  }
  writeFileSync(file, JSON.stringify(messages, null, 2) + '\n', 'utf8')
  console.log(`${lang}.json — +${added} added, ~${changed} changed`)
}
console.log('done')

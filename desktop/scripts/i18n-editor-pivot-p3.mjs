/**
 * Editor-Pivot P3 (R4 Tab-Sichtbarkeit / moduleAreas) i18n pass.
 * Ausführen aus desktop/: node scripts/i18n-editor-pivot-p3.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))
const msgDir = join(here, '..', 'src', 'renderer', 'src', 'i18n', 'messages')

const ADD = {
  'customization.editor.nav.areas': { de: 'Bereiche', en: 'Areas', fr: 'Zones', it: 'Aree' },
  'customization.editor.props.areasDesc': {
    de: 'Reiter und Bereiche des Moduls für diesen Mandanten ein- oder ausblenden.',
    en: 'Show or hide the module’s tabs and sections for this tenant.',
    fr: 'Afficher ou masquer les onglets et sections du module pour ce client.',
    it: 'Mostra o nascondi le schede e le sezioni del modulo per questo cliente.',
  },
  'customization.editor.bereiche.empty': {
    de: 'Dieses Modul hat keine ein-/ausblendbaren Bereiche.',
    en: 'This module has no toggleable areas.',
    fr: 'Ce module n’a aucune zone activable.',
    it: 'Questo modulo non ha aree attivabili.',
  },
  'customization.editor.bereiche.hint': {
    de: 'Blende Reiter aus, die dieser Mandant nicht braucht — die Daten bleiben erhalten.',
    en: 'Hide tabs this tenant does not need — the data is kept.',
    fr: 'Masque les onglets dont ce client n’a pas besoin — les données sont conservées.',
    it: 'Nascondi le schede che questo cliente non usa — i dati restano.',
  },
  'customization.editor.bereiche.visible': { de: 'Sichtbar', en: 'Visible', fr: 'Visible', it: 'Visibile' },
  'customization.editor.bereiche.hidden': { de: 'Ausgeblendet', en: 'Hidden', fr: 'Masqué', it: 'Nascosto' },
  'customization.editor.bereiche.show': {
    de: '{area} einblenden',
    en: 'Show {area}',
    fr: 'Afficher {area}',
    it: 'Mostra {area}',
  },
  'customization.editor.bereiche.hide': {
    de: '{area} ausblenden',
    en: 'Hide {area}',
    fr: 'Masquer {area}',
    it: 'Nascondi {area}',
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

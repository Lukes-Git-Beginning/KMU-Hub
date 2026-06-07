// Phase 2 (Kommunikation merge) i18n: area switcher, settings panel, nav label.
// Overwrites the nav label (was "Posteingang") to the unified module name.
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const dir = join(dirname(fileURLToPath(import.meta.url)), '..', 'src', 'renderer', 'src', 'i18n', 'messages')

// Keys that are ADDED only if missing.
const ADD = {
  'moduleSettings.entries.kommunikation': { de: 'Kommunikation', en: 'Communication', fr: 'Communication', it: 'Comunicazione' },
  'kommunikation.bereich.team': { de: 'Team', en: 'Team', fr: 'Équipe', it: 'Team' },
  'kommunikation.bereich.posteingang': { de: 'Posteingang', en: 'Inbox', fr: 'Boîte de réception', it: 'Posta in arrivo' },
  'kommunikation.settings.title': { de: 'Kommunikation', en: 'Communication', fr: 'Communication', it: 'Comunicazione' },
  'kommunikation.settings.desc': {
    de: 'Team-Chat und Kundenposteingang nach deinen Vorlieben einrichten.',
    en: 'Set up team chat and customer inbox to your liking.',
    fr: 'Configurez le chat d’équipe et la boîte de réception client selon vos préférences.',
    it: 'Configura la chat del team e la posta in arrivo dei clienti secondo le tue preferenze.',
  },
  'kommunikation.settings.display.title': { de: 'Anzeige', en: 'Display', fr: 'Affichage', it: 'Visualizzazione' },
  'kommunikation.settings.display.desc': {
    de: 'Startbereich, Dichte und Eingabeverhalten.',
    en: 'Start area, density and input behaviour.',
    fr: 'Zone de départ, densité et comportement de saisie.',
    it: 'Area iniziale, densità e comportamento di inserimento.',
  },
  'kommunikation.settings.display.defaultArea': { de: 'Startbereich', en: 'Start area', fr: 'Zone de départ', it: 'Area iniziale' },
  'kommunikation.settings.display.density': { de: 'Dichte', en: 'Density', fr: 'Densité', it: 'Densità' },
  'kommunikation.settings.display.densityComfortable': { de: 'Komfortabel', en: 'Comfortable', fr: 'Confortable', it: 'Comoda' },
  'kommunikation.settings.display.densityCompact': { de: 'Kompakt', en: 'Compact', fr: 'Compacte', it: 'Compatta' },
  'kommunikation.settings.display.enterToSend': { de: 'Mit Enter senden', en: 'Send with Enter', fr: 'Envoyer avec Entrée', it: 'Invia con Invio' },
}

// Keys that are OVERWRITTEN (label changed meaning after the merge).
const SET = {
  'layout.navItems.kommunikation': { de: 'Kommunikation', en: 'Communication', fr: 'Communication', it: 'Comunicazione' },
}

for (const lang of ['de', 'en', 'fr', 'it']) {
  const file = join(dir, `${lang}.json`)
  const json = JSON.parse(readFileSync(file, 'utf8'))
  let added = 0
  for (const [key, vals] of Object.entries(ADD)) {
    if (!(key in json)) {
      json[key] = vals[lang]
      added++
    }
  }
  for (const [key, vals] of Object.entries(SET)) {
    json[key] = vals[lang]
  }
  writeFileSync(file, JSON.stringify(json, null, 2) + '\n', 'utf8')
  console.log(`${lang}: +${added} added, ${Object.keys(SET).length} overwritten`)
}

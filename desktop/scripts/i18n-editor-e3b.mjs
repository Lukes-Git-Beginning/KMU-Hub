/**
 * Modul-Editor E-3b i18n pass — Wertelisten-Panel (module-scoped value-set editor).
 * Ausführen aus desktop/: node scripts/i18n-editor-e3b.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))
const msgDir = join(here, '..', 'src', 'renderer', 'src', 'i18n', 'messages')

const ADD = {
  'customization.editor.wertelisten.setNameLabel': {
    de: 'Name der Liste',
    en: 'List name',
    fr: 'Nom de la liste',
    it: 'Nome dell’elenco',
  },
  'customization.editor.wertelisten.toggleHidden': {
    de: 'Sichtbarkeit umschalten',
    en: 'Toggle visibility',
    fr: 'Basculer la visibilité',
    it: 'Attiva/disattiva visibilità',
  },
  'customization.editor.wertelisten.optionActive': {
    de: 'Sichtbar',
    en: 'Visible',
    fr: 'Visible',
    it: 'Visibile',
  },
  'customization.editor.wertelisten.optionHidden': {
    de: 'Ausgeblendet — bestehende Daten bleiben erhalten',
    en: 'Hidden — existing data is kept',
    fr: 'Masqué — les données existantes sont conservées',
    it: 'Nascosto — i dati esistenti vengono mantenuti',
  },
  'customization.editor.wertelisten.colorLabel': {
    de: 'Farbe wählen',
    en: 'Choose colour',
    fr: 'Choisir la couleur',
    it: 'Scegli colore',
  },
  'customization.editor.wertelisten.preview': {
    de: 'Vorschau',
    en: 'Preview',
    fr: 'Aperçu',
    it: 'Anteprima',
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

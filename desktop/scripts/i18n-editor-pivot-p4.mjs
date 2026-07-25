/**
 * Editor-Pivot P4 (Chrome / Kontext-Inspektor) i18n pass.
 * Ausführen aus desktop/: node scripts/i18n-editor-pivot-p4.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))
const msgDir = join(here, '..', 'src', 'renderer', 'src', 'i18n', 'messages')

const ADD = {
  'customization.editor.inspector.title': { de: 'So passt du an', en: 'How to customize', fr: 'Comment personnaliser', it: 'Come personalizzare' },
  'customization.editor.inspector.subtitle': {
    de: 'Änderungen bleiben ein Entwurf, bis du sie übernimmst.',
    en: 'Changes stay a draft until you apply them.',
    fr: 'Les modifications restent un brouillon jusqu’à ce que tu les appliques.',
    it: 'Le modifiche restano una bozza finché non le applichi.',
  },
  'customization.editor.inspector.step1': {
    de: 'Klick im Modul direkt auf einen Text — Spalten, Überschriften, Feld-Namen — um ihn umzubenennen.',
    en: 'Click any text in the module — columns, headings, field names — to rename it.',
    fr: 'Clique sur un texte du module — colonnes, titres, noms de champs — pour le renommer.',
    it: 'Clicca un testo nel modulo — colonne, titoli, nomi dei campi — per rinominarlo.',
  },
  'customization.editor.inspector.step2': {
    de: 'Reiter benennst du per Doppelklick um.',
    en: 'Rename tabs with a double click.',
    fr: 'Renomme les onglets par un double clic.',
    it: 'Rinomina le schede con un doppio clic.',
  },
  'customization.editor.inspector.step3': {
    de: 'Links: Felder anlegen, Wertelisten pflegen und Bereiche ein- oder ausblenden.',
    en: 'On the left: add fields, manage value lists, and show or hide areas.',
    fr: 'À gauche : ajoute des champs, gère les listes de valeurs et affiche ou masque des zones.',
    it: 'A sinistra: aggiungi campi, gestisci gli elenchi valori e mostra o nascondi le aree.',
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

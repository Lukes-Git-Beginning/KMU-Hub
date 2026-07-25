/**
 * Editor-Pivot G3 i18n pass (KB on the shared block-document engine).
 *
 * ADD: 2 keys x4 — editable KB title placeholder + default title for a freshly
 * created article. Du-Form, real umlauts, real fr/it. ICU single braces {var}.
 *
 * Run once from desktop/: node scripts/i18n-editor-g3.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))
const msgDir = join(here, '..', 'src', 'renderer', 'src', 'i18n', 'messages')

const ADD = {
  'helpdesk.kb.titlePlaceholder': {
    de: 'Titel des Artikels', en: 'Article title', fr: "Titre de l'article", it: "Titolo dell'articolo",
  },
  'helpdesk.kb.newArticleTitle': {
    de: 'Neuer Artikel', en: 'New article', fr: 'Nouvel article', it: 'Nuovo articolo',
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

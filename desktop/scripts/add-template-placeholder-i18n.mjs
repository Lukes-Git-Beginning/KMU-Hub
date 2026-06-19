import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
const dir = resolve('src/renderer/src/i18n/messages')
const key = 'dokumente.template.placeholderBody'
const vals = {
  de: 'Dieses Dokument wurde aus einer Vorlage erstellt. Inhalt hier einfügen.',
  en: 'This document was created from a template. Add your content here.',
  fr: 'Ce document a été créé à partir d’un modèle. Ajoutez votre contenu ici.',
  it: 'Questo documento è stato creato da un modello. Inserisci qui il contenuto.',
}
const lineKey = (line) => {
  const m = line.match(/^\s*"([^"]+)":/)
  return m ? m[1] : null
}
const report = {}
for (const loc of ['de', 'en', 'fr', 'it']) {
  const file = resolve(dir, `${loc}.json`)
  const obj = JSON.parse(readFileSync(file, 'utf8'))
  if (key in obj) { report[loc] = 'exists'; continue }
  let lines = readFileSync(file, 'utf8').split('\n')
  const idx = lines.findIndex((l) => {
    const k = lineKey(l)
    return k !== null && k > key
  })
  if (idx === -1) throw new Error(`no insertion point in ${loc}`)
  lines = [...lines.slice(0, idx), `  ${JSON.stringify(key)}: ${JSON.stringify(vals[loc])},`, ...lines.slice(idx)]
  const out = lines.join('\n')
  JSON.parse(out)
  writeFileSync(file, out, 'utf8')
  report[loc] = 'added'
}
console.log(JSON.stringify(report))

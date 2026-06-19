import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'

const dir = resolve('src/renderer/src/i18n/messages')

const keys = {
  'team.personnelDocs.download': { de: 'Herunterladen', en: 'Download', fr: 'Télécharger', it: 'Scarica' },
  'team.personnelDocs.fileSize': { de: 'Größe', en: 'Size', fr: 'Taille', it: 'Dimensione' },
  'team.personnelDocs.previewBadge': { de: 'Demo-Vorschau', en: 'Demo preview', fr: 'Aperçu démo', it: 'Anteprima demo' },
  'team.personnelDocs.previewHint': { de: 'Im Produktivbetrieb wird hier das hinterlegte PDF angezeigt.', en: 'In production the stored PDF is shown here.', fr: 'En production, le PDF enregistré est affiché ici.', it: 'In produzione qui viene mostrato il PDF archiviato.' },
  'team.personnelDocs.uploadError': { de: 'Upload fehlgeschlagen', en: 'Upload failed', fr: 'Échec du téléversement', it: 'Caricamento non riuscito' },
}

const locales = ['de', 'en', 'fr', 'it']
const report = {}

function insertAfter(lines, anchorKey, blockLines) {
  const idx = lines.findIndex((l) => l.trimStart().startsWith(`"${anchorKey}":`))
  if (idx === -1) throw new Error(`anchor not found: ${anchorKey}`)
  return [...lines.slice(0, idx + 1), ...blockLines, ...lines.slice(idx + 1)]
}

for (const loc of locales) {
  const file = resolve(dir, `${loc}.json`)
  const obj = JSON.parse(readFileSync(file, 'utf8'))
  let lines = readFileSync(file, 'utf8').split('\n')

  const newKeys = Object.keys(keys).filter((k) => !(k in obj)).sort()
  const block = newKeys.map((k) => `  ${JSON.stringify(k)}: ${JSON.stringify(keys[k][loc])},`)
  if (block.length) lines = insertAfter(lines, 'team.personnelDocs.totalDocuments', block)

  const out = lines.join('\n')
  JSON.parse(out)
  writeFileSync(file, out, 'utf8')
  report[loc] = { added: block.length }
}
console.log(JSON.stringify(report, null, 2))

// Additive i18n for event RSVP (B2). Inserts after the first "kalender." line.
import { readFileSync, writeFileSync } from 'node:fs'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const dir = resolve(dirname(fileURLToPath(import.meta.url)), '../src/renderer/src/i18n/messages')

const keys = {
  'kalender.rsvp.yourResponse': { de: 'Deine Antwort', en: 'Your response', fr: 'Votre réponse', it: 'La tua risposta' },
  'kalender.rsvp.accept': { de: 'Zusagen', en: 'Accept', fr: 'Accepter', it: 'Accetta' },
  'kalender.rsvp.maybe': { de: 'Vielleicht', en: 'Maybe', fr: 'Peut-être', it: 'Forse' },
  'kalender.rsvp.decline': { de: 'Absagen', en: 'Decline', fr: 'Refuser', it: 'Rifiuta' },
  'kalender.rsvp.saved': { de: 'Antwort gespeichert', en: 'Response saved', fr: 'Réponse enregistrée', it: 'Risposta salvata' },
}

const report = {}
for (const loc of ['de', 'en', 'fr', 'it']) {
  const file = resolve(dir, `${loc}.json`)
  const obj = JSON.parse(readFileSync(file, 'utf8'))
  const nk = Object.keys(keys).filter((k) => !(k in obj))
  if (!nk.length) { report[loc] = 0; continue }
  let lines = readFileSync(file, 'utf8').split('\n')
  const idx = lines.findIndex((l) => l.trimStart().startsWith('"kalender.'))
  if (idx === -1) throw new Error(`no kalender anchor in ${loc}`)
  const block = nk.map((k) => `  ${JSON.stringify(k)}: ${JSON.stringify(keys[k][loc])},`)
  lines = [...lines.slice(0, idx + 1), ...block, ...lines.slice(idx + 1)]
  const out = lines.join('\n')
  JSON.parse(out)
  writeFileSync(file, out, 'utf8')
  report[loc] = block.length
}
console.log(JSON.stringify(report))

// Feedback fix: read-only weekly target in personal settings. 3 keys x4.
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const __dirname = dirname(fileURLToPath(import.meta.url))
const dir = join(__dirname, '..', 'src', 'renderer', 'src', 'i18n', 'messages')

const additions = {
  de: {
    'zeiterfassung.settings.personal.weeklyTarget': 'Dein Wochensoll',
    'zeiterfassung.settings.personal.weeklyTargetValue': '{hours} Std/Woche',
    'zeiterfassung.settings.personal.weeklyTargetHint': 'Wird im Team-/HR-Bereich festgelegt (Personalakte).',
  },
  en: {
    'zeiterfassung.settings.personal.weeklyTarget': 'Your weekly target',
    'zeiterfassung.settings.personal.weeklyTargetValue': '{hours} h/week',
    'zeiterfassung.settings.personal.weeklyTargetHint': 'Set in the Team/HR module (personnel file).',
  },
  fr: {
    'zeiterfassung.settings.personal.weeklyTarget': 'Votre objectif hebdo',
    'zeiterfassung.settings.personal.weeklyTargetValue': '{hours} h/semaine',
    'zeiterfassung.settings.personal.weeklyTargetHint': 'Défini dans le module Équipe/RH (dossier du personnel).',
  },
  it: {
    'zeiterfassung.settings.personal.weeklyTarget': 'Il tuo obiettivo settimanale',
    'zeiterfassung.settings.personal.weeklyTargetValue': '{hours} h/settimana',
    'zeiterfassung.settings.personal.weeklyTargetHint': 'Impostato nel modulo Team/HR (fascicolo personale).',
  },
}

for (const [locale, keys] of Object.entries(additions)) {
  const file = join(dir, `${locale}.json`)
  const json = JSON.parse(readFileSync(file, 'utf8'))
  let added = 0
  for (const [k, v] of Object.entries(keys).sort(([a], [b]) => (a < b ? -1 : a > b ? 1 : 0))) {
    if (k in json) { json[k] = v; continue }
    added++
    const entries = Object.entries(json)
    let idx = entries.findIndex(([ek]) => ek > k)
    if (idx === -1) idx = entries.length
    entries.splice(idx, 0, [k, v])
    for (const key of Object.keys(json)) delete json[key]
    for (const [ek, ev] of entries) json[ek] = ev
  }
  writeFileSync(file, JSON.stringify(json, null, 2) + '\n', 'utf8')
  console.log(`${locale}: +${added} keys`)
}

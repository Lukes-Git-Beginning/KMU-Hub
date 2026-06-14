// Adds zeiterfassung shell + balance i18n keys to all 4 locales (sorted, parity).
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const __dirname = dirname(fileURLToPath(import.meta.url))
const dir = join(__dirname, '..', 'src', 'renderer', 'src', 'i18n', 'messages')

const additions = {
  de: {
    'zeiterfassung.shell.title': 'Zeiterfassung',
    'zeiterfassung.shell.subtitle': 'ArbZG-konforme Arbeitszeiterfassung',
    'zeiterfassung.balance.label': 'Stundenkonto',
  },
  en: {
    'zeiterfassung.shell.title': 'Time Tracking',
    'zeiterfassung.shell.subtitle': 'Working-hours tracking, ArbZG-compliant',
    'zeiterfassung.balance.label': 'Time account',
  },
  fr: {
    'zeiterfassung.shell.title': 'Suivi du temps',
    'zeiterfassung.shell.subtitle': 'Suivi du temps de travail conforme (ArbZG)',
    'zeiterfassung.balance.label': 'Compte d’heures',
  },
  it: {
    'zeiterfassung.shell.title': 'Rilevazione presenze',
    'zeiterfassung.shell.subtitle': 'Registrazione orari conforme (ArbZG)',
    'zeiterfassung.balance.label': 'Conto ore',
  },
}

for (const [locale, keys] of Object.entries(additions)) {
  const file = join(dir, `${locale}.json`)
  const json = JSON.parse(readFileSync(file, 'utf8'))
  const entries = Object.entries(json)

  // Order-preserving contiguous block insertion (files are NOT globally sorted).
  const block = Object.entries(keys).sort(([a], [b]) => (a < b ? -1 : a > b ? 1 : 0))
  const firstKey = block[0][0]
  let idx = entries.findIndex(([k]) => k > firstKey)
  if (idx === -1) idx = entries.length
  const added = block.filter(([k]) => !(k in json)).length
  entries.splice(idx, 0, ...block)

  const out = Object.fromEntries(entries)
  writeFileSync(file, JSON.stringify(out, null, 2) + '\n', 'utf8')
  console.log(`${locale}: +${added} keys (total ${Object.keys(out).length}) @${idx}`)
}

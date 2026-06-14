// P3 zeiterfassung i18n: analytics view. 12 keys x4, order-preserving per-key insertion.
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const __dirname = dirname(fileURLToPath(import.meta.url))
const dir = join(__dirname, '..', 'src', 'renderer', 'src', 'i18n', 'messages')

const additions = {
  de: {
    'zeiterfassung.analytics.tab': 'Auswertungen',
    'zeiterfassung.analytics.title': 'Auswertungen',
    'zeiterfassung.analytics.range.week': 'Woche',
    'zeiterfassung.analytics.range.month': 'Monat',
    'zeiterfassung.analytics.total': 'Gesamt',
    'zeiterfassung.analytics.billable': 'Abrechenbar',
    'zeiterfassung.analytics.nonBillable': 'Nicht abrechenbar',
    'zeiterfassung.analytics.overtime': 'Überstunden',
    'zeiterfassung.analytics.avgPerDay': 'Ø / Tag',
    'zeiterfassung.analytics.hoursPerDay': 'Stunden pro Tag',
    'zeiterfassung.analytics.byProject': 'Nach Projekt',
    'zeiterfassung.analytics.billableSplit': 'Abrechenbar vs. nicht',
  },
  en: {
    'zeiterfassung.analytics.tab': 'Reports',
    'zeiterfassung.analytics.title': 'Reports',
    'zeiterfassung.analytics.range.week': 'Week',
    'zeiterfassung.analytics.range.month': 'Month',
    'zeiterfassung.analytics.total': 'Total',
    'zeiterfassung.analytics.billable': 'Billable',
    'zeiterfassung.analytics.nonBillable': 'Non-billable',
    'zeiterfassung.analytics.overtime': 'Overtime',
    'zeiterfassung.analytics.avgPerDay': 'Avg / day',
    'zeiterfassung.analytics.hoursPerDay': 'Hours per day',
    'zeiterfassung.analytics.byProject': 'By project',
    'zeiterfassung.analytics.billableSplit': 'Billable vs. non-billable',
  },
  fr: {
    'zeiterfassung.analytics.tab': 'Analyses',
    'zeiterfassung.analytics.title': 'Analyses',
    'zeiterfassung.analytics.range.week': 'Semaine',
    'zeiterfassung.analytics.range.month': 'Mois',
    'zeiterfassung.analytics.total': 'Total',
    'zeiterfassung.analytics.billable': 'Facturable',
    'zeiterfassung.analytics.nonBillable': 'Non facturable',
    'zeiterfassung.analytics.overtime': 'Heures supp.',
    'zeiterfassung.analytics.avgPerDay': 'Moy. / jour',
    'zeiterfassung.analytics.hoursPerDay': 'Heures par jour',
    'zeiterfassung.analytics.byProject': 'Par projet',
    'zeiterfassung.analytics.billableSplit': 'Facturable vs non',
  },
  it: {
    'zeiterfassung.analytics.tab': 'Analisi',
    'zeiterfassung.analytics.title': 'Analisi',
    'zeiterfassung.analytics.range.week': 'Settimana',
    'zeiterfassung.analytics.range.month': 'Mese',
    'zeiterfassung.analytics.total': 'Totale',
    'zeiterfassung.analytics.billable': 'Fatturabile',
    'zeiterfassung.analytics.nonBillable': 'Non fatturabile',
    'zeiterfassung.analytics.overtime': 'Straordinari',
    'zeiterfassung.analytics.avgPerDay': 'Media / giorno',
    'zeiterfassung.analytics.hoursPerDay': 'Ore al giorno',
    'zeiterfassung.analytics.byProject': 'Per progetto',
    'zeiterfassung.analytics.billableSplit': 'Fatturabile vs no',
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
  console.log(`${locale}: +${added} keys (total ${Object.keys(json).length})`)
}

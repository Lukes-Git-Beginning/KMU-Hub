import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
const dir = resolve('src/renderer/src/i18n/messages')

const keys = {
  'crm.activities.agenda.empty': { de: 'Keine offenen Wiedervorlagen — alles erledigt.', en: 'No open follow-ups — all done.', fr: 'Aucune relance ouverte — tout est fait.', it: 'Nessun promemoria aperto — tutto fatto.' },
  'crm.activities.agenda.emptyTitle': { de: 'Alles erledigt', en: 'All done', fr: 'Tout est fait', it: 'Tutto fatto' },
  'crm.activities.agenda.inDays': { de: 'in {days} T.', en: 'in {days}d', fr: 'dans {days} j', it: 'tra {days}g' },
  'crm.activities.agenda.overdueDays': { de: '{days} T. überfällig', en: '{days}d overdue', fr: '{days} j de retard', it: '{days}g di ritardo' },
  'crm.activities.agenda.reschedule': { de: 'Verschieben auf', en: 'Reschedule to', fr: 'Reporter au', it: 'Riprogramma al' },
  'crm.activities.bucket.later': { de: 'Später', en: 'Later', fr: 'Plus tard', it: 'Più tardi' },
  'crm.activities.bucket.noDate': { de: 'Ohne Termin', en: 'No date', fr: 'Sans date', it: 'Senza data' },
  'crm.activities.bucket.overdue': { de: 'Überfällig', en: 'Overdue', fr: 'En retard', it: 'In ritardo' },
  'crm.activities.bucket.thisWeek': { de: 'Diese Woche', en: 'This week', fr: 'Cette semaine', it: 'Questa settimana' },
  'crm.activities.bucket.today': { de: 'Heute', en: 'Today', fr: "Aujourd'hui", it: 'Oggi' },
  'crm.activities.toast.completed': { de: 'Als erledigt markiert', en: 'Marked as done', fr: 'Marqué comme terminé', it: 'Segnato come completato' },
  'crm.activities.toast.rescheduled': { de: 'Wiedervorlage verschoben', en: 'Follow-up rescheduled', fr: 'Relance reportée', it: 'Promemoria riprogrammato' },
  'crm.activities.view.agenda': { de: 'Wiedervorlage', en: 'Follow-ups', fr: 'Relances', it: 'Promemoria' },
  'crm.activities.view.list': { de: 'Liste', en: 'List', fr: 'Liste', it: 'Elenco' },
}

const report = {}
for (const loc of ['de', 'en', 'fr', 'it']) {
  const file = resolve(dir, `${loc}.json`)
  const obj = JSON.parse(readFileSync(file, 'utf8'))
  let lines = readFileSync(file, 'utf8').split('\n')
  const idx = lines.findIndex((l) => l.trimStart().startsWith('"crm.activities.completed":'))
  if (idx === -1) throw new Error(`anchor missing in ${loc}`)
  const newKeys = Object.keys(keys).filter((k) => !(k in obj)).sort()
  const block = newKeys.map((k) => `  ${JSON.stringify(k)}: ${JSON.stringify(keys[k][loc])},`)
  lines = [...lines.slice(0, idx + 1), ...block, ...lines.slice(idx + 1)]
  const out = lines.join('\n')
  JSON.parse(out)
  writeFileSync(file, out, 'utf8')
  report[loc] = block.length
}
console.log(JSON.stringify(report))

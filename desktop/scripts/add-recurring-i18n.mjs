// Additive i18n for recurring-event display (badge) + edit-scope dialog (B1).
// Inserts after the first "kalender." line per locale; preserves flat format.
import { readFileSync, writeFileSync } from 'node:fs'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const dir = resolve(dirname(fileURLToPath(import.meta.url)), '../src/renderer/src/i18n/messages')

const keys = {
  // Recurrence display labels (used by rruleToDisplay; were missing → raw keys)
  'kalender.recurrence.none': { de: 'Keine', en: 'None', fr: 'Aucune', it: 'Nessuna' },
  'kalender.recurrence.daily': { de: 'Täglich', en: 'Daily', fr: 'Quotidien', it: 'Giornaliero' },
  'kalender.recurrence.weekly': { de: 'Wöchentlich', en: 'Weekly', fr: 'Hebdomadaire', it: 'Settimanale' },
  'kalender.recurrence.monthly': { de: 'Monatlich', en: 'Monthly', fr: 'Mensuel', it: 'Mensile' },
  'kalender.recurrence.yearly': { de: 'Jährlich', en: 'Yearly', fr: 'Annuel', it: 'Annuale' },
  'kalender.recurrence.custom': { de: 'Benutzerdefiniert', en: 'Custom', fr: 'Personnalisé', it: 'Personalizzato' },
  // Recurring-edit scope dialog
  'kalender.recurring.editTitle': { de: 'Serientermin bearbeiten', en: 'Edit recurring event', fr: 'Modifier la série', it: 'Modifica serie' },
  'kalender.recurring.editDescription': { de: 'Welche Termine der Serie möchtest du ändern?', en: 'Which events in the series do you want to change?', fr: 'Quels événements de la série voulez-vous modifier ?', it: 'Quali eventi della serie vuoi modificare?' },
  'kalender.recurring.thisEvent': { de: 'Nur dieser Termin', en: 'This event only', fr: 'Cet événement uniquement', it: 'Solo questo evento' },
  'kalender.recurring.thisAndFuture': { de: 'Dieser und alle folgenden', en: 'This and all following', fr: 'Celui-ci et les suivants', it: 'Questo e i successivi' },
  'kalender.recurring.allEvents': { de: 'Alle Termine der Serie', en: 'All events in the series', fr: 'Tous les événements de la série', it: 'Tutti gli eventi della serie' },
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

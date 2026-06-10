// Additive i18n for kontakte P7/P8 finishing (lead-scoring editor + segment override).
import { readFileSync, writeFileSync } from 'node:fs'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const dir = resolve(dirname(fileURLToPath(import.meta.url)), '../src/renderer/src/i18n/messages')

const keys = {
  // Lead-scoring settings section
  'crm.settings.leadScoring.title': { de: 'Lead-Scoring', en: 'Lead scoring', fr: 'Scoring des leads', it: 'Lead scoring' },
  'crm.settings.leadScoring.desc': { de: 'Punkte-Regeln und Temperatur-Schwellen für die automatische Lead-Bewertung', en: 'Point rules and temperature thresholds for automatic lead scoring', fr: 'Règles de points et seuils de température pour le scoring automatique', it: 'Regole di punteggio e soglie di temperatura per il lead scoring automatico' },
  'crm.leadScoring.sourceTitle': { de: 'Basispunkte je Quelle', en: 'Base points per source', fr: 'Points de base par source', it: 'Punti base per origine' },
  'crm.leadScoring.fieldTitle': { de: 'Punkte je ausgefülltem Feld', en: 'Points per filled field', fr: 'Points par champ rempli', it: 'Punti per campo compilato' },
  'crm.leadScoring.thresholdTitle': { de: 'Temperatur-Schwellen', en: 'Temperature thresholds', fr: 'Seuils de température', it: 'Soglie di temperatura' },
  'crm.leadScoring.source.dialer': { de: 'Dialer', en: 'Dialer', fr: 'Dialer', it: 'Dialer' },
  'crm.leadScoring.source.manual': { de: 'Manuell', en: 'Manual', fr: 'Manuel', it: 'Manuale' },
  'crm.leadScoring.source.csv': { de: 'CSV-Import', en: 'CSV import', fr: 'Import CSV', it: 'Import CSV' },
  'crm.leadScoring.field.email': { de: 'E-Mail', en: 'Email', fr: 'E-mail', it: 'Email' },
  'crm.leadScoring.field.phone': { de: 'Telefon', en: 'Phone', fr: 'Téléphone', it: 'Telefono' },
  'crm.leadScoring.field.company': { de: 'Firma', en: 'Company', fr: 'Société', it: 'Azienda' },
  'crm.leadScoring.field.notes': { de: 'Notiz', en: 'Note', fr: 'Note', it: 'Nota' },
  'crm.leadScoring.hotFrom': { de: 'Heiß ab', en: 'Hot from', fr: 'Chaud à partir de', it: 'Caldo da' },
  'crm.leadScoring.warmFrom': { de: 'Warm ab', en: 'Warm from', fr: 'Tiède à partir de', it: 'Tiepido da' },
  'crm.leadScoring.preview': { de: 'Vorschau (Beispiel-Lead)', en: 'Preview (sample lead)', fr: 'Aperçu (lead exemple)', it: 'Anteprima (lead esempio)' },
  'crm.leadScoring.reset': { de: 'Zurücksetzen', en: 'Reset', fr: 'Réinitialiser', it: 'Reimposta' },
  'crm.leadScoring.temp.hot': { de: 'Heiß', en: 'Hot', fr: 'Chaud', it: 'Caldo' },
  'crm.leadScoring.temp.warm': { de: 'Warm', en: 'Warm', fr: 'Tiède', it: 'Tiepido' },
  'crm.leadScoring.temp.cold': { de: 'Kalt', en: 'Cold', fr: 'Froid', it: 'Freddo' },
  // Manual segment override
  'crm.segment.manual': { de: 'manuell', en: 'manual', fr: 'manuel', it: 'manuale' },
  'crm.segment.setManual': { de: 'Segment manuell setzen', en: 'Set segment manually', fr: 'Définir le segment manuellement', it: 'Imposta segmento manualmente' },
  'crm.segment.auto': { de: 'Automatisch ({segment})', en: 'Automatic ({segment})', fr: 'Automatique ({segment})', it: 'Automatico ({segment})' },
}

const report = {}
for (const loc of ['de', 'en', 'fr', 'it']) {
  const file = resolve(dir, `${loc}.json`)
  const obj = JSON.parse(readFileSync(file, 'utf8'))
  const nk = Object.keys(keys).filter((k) => !(k in obj))
  if (!nk.length) { report[loc] = 0; continue }
  let lines = readFileSync(file, 'utf8').split('\n')
  const idx = lines.findIndex((l) => l.trimStart().startsWith('"crm.'))
  const anchorIdx = idx !== -1 ? idx : lines.findIndex((l) => l.trim().startsWith('"'))
  const block = nk.map((k) => `  ${JSON.stringify(k)}: ${JSON.stringify(keys[k][loc])},`)
  lines = [...lines.slice(0, anchorIdx + 1), ...block, ...lines.slice(anchorIdx + 1)]
  const out = lines.join('\n')
  JSON.parse(out)
  writeFileSync(file, out, 'utf8')
  report[loc] = block.length
}
console.log(JSON.stringify(report))

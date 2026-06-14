// P2 zeiterfassung i18n: manual entry + project/customer. 21 keys x4, order-preserving per-key insertion.
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const __dirname = dirname(fileURLToPath(import.meta.url))
const dir = join(__dirname, '..', 'src', 'renderer', 'src', 'i18n', 'messages')

const additions = {
  de: {
    'api.hr.time.entryCreated': 'Zeiteintrag erstellt',
    'api.hr.time.error.entryCreate': 'Zeiteintrag konnte nicht erstellt werden',
    'zeiterfassung.manual.title': 'Zeiteintrag erfassen',
    'zeiterfassung.manual.newEntry': 'Neuer Eintrag',
    'zeiterfassung.manual.date': 'Datum',
    'zeiterfassung.manual.start': 'Von',
    'zeiterfassung.manual.end': 'Bis',
    'zeiterfassung.manual.break': 'Pause (Min.)',
    'zeiterfassung.manual.net': 'Netto',
    'zeiterfassung.manual.project': 'Projekt',
    'zeiterfassung.manual.selectProject': 'Projekt wählen…',
    'zeiterfassung.manual.activity': 'Leistung',
    'zeiterfassung.manual.activityPlaceholder': 'z. B. Beratung, Entwicklung…',
    'zeiterfassung.manual.billable': 'Abrechenbar',
    'zeiterfassung.manual.billableShort': 'Abrechenbar',
    'zeiterfassung.manual.note': 'Notiz',
    'zeiterfassung.manual.notePlaceholder': 'Optionale Notiz…',
    'zeiterfassung.manual.createEntry': 'Eintrag speichern',
    'zeiterfassung.manual.endAfterStart': 'Ende muss nach Beginn liegen',
    'zeiterfassung.manual.noFutureDate': 'Kein Datum in der Zukunft',
    'zeiterfassung.manual.manualBadge': 'Manuell',
  },
  en: {
    'api.hr.time.entryCreated': 'Time entry created',
    'api.hr.time.error.entryCreate': 'Could not create time entry',
    'zeiterfassung.manual.title': 'Log time entry',
    'zeiterfassung.manual.newEntry': 'New entry',
    'zeiterfassung.manual.date': 'Date',
    'zeiterfassung.manual.start': 'From',
    'zeiterfassung.manual.end': 'To',
    'zeiterfassung.manual.break': 'Break (min)',
    'zeiterfassung.manual.net': 'Net',
    'zeiterfassung.manual.project': 'Project',
    'zeiterfassung.manual.selectProject': 'Select project…',
    'zeiterfassung.manual.activity': 'Activity',
    'zeiterfassung.manual.activityPlaceholder': 'e.g. consulting, development…',
    'zeiterfassung.manual.billable': 'Billable',
    'zeiterfassung.manual.billableShort': 'Billable',
    'zeiterfassung.manual.note': 'Note',
    'zeiterfassung.manual.notePlaceholder': 'Optional note…',
    'zeiterfassung.manual.createEntry': 'Save entry',
    'zeiterfassung.manual.endAfterStart': 'End must be after start',
    'zeiterfassung.manual.noFutureDate': 'No future date',
    'zeiterfassung.manual.manualBadge': 'Manual',
  },
  fr: {
    'api.hr.time.entryCreated': 'Saisie de temps créée',
    'api.hr.time.error.entryCreate': 'Impossible de créer la saisie',
    'zeiterfassung.manual.title': 'Saisir un temps',
    'zeiterfassung.manual.newEntry': 'Nouvelle saisie',
    'zeiterfassung.manual.date': 'Date',
    'zeiterfassung.manual.start': 'De',
    'zeiterfassung.manual.end': 'À',
    'zeiterfassung.manual.break': 'Pause (min)',
    'zeiterfassung.manual.net': 'Net',
    'zeiterfassung.manual.project': 'Projet',
    'zeiterfassung.manual.selectProject': 'Choisir un projet…',
    'zeiterfassung.manual.activity': 'Prestation',
    'zeiterfassung.manual.activityPlaceholder': 'p. ex. conseil, développement…',
    'zeiterfassung.manual.billable': 'Facturable',
    'zeiterfassung.manual.billableShort': 'Facturable',
    'zeiterfassung.manual.note': 'Note',
    'zeiterfassung.manual.notePlaceholder': 'Note facultative…',
    'zeiterfassung.manual.createEntry': 'Enregistrer',
    'zeiterfassung.manual.endAfterStart': 'La fin doit suivre le début',
    'zeiterfassung.manual.noFutureDate': 'Pas de date future',
    'zeiterfassung.manual.manualBadge': 'Manuel',
  },
  it: {
    'api.hr.time.entryCreated': 'Voce di tempo creata',
    'api.hr.time.error.entryCreate': 'Impossibile creare la voce',
    'zeiterfassung.manual.title': 'Registra orario',
    'zeiterfassung.manual.newEntry': 'Nuova voce',
    'zeiterfassung.manual.date': 'Data',
    'zeiterfassung.manual.start': 'Dalle',
    'zeiterfassung.manual.end': 'Alle',
    'zeiterfassung.manual.break': 'Pausa (min)',
    'zeiterfassung.manual.net': 'Netto',
    'zeiterfassung.manual.project': 'Progetto',
    'zeiterfassung.manual.selectProject': 'Scegli progetto…',
    'zeiterfassung.manual.activity': 'Prestazione',
    'zeiterfassung.manual.activityPlaceholder': 'es. consulenza, sviluppo…',
    'zeiterfassung.manual.billable': 'Fatturabile',
    'zeiterfassung.manual.billableShort': 'Fatturabile',
    'zeiterfassung.manual.note': 'Nota',
    'zeiterfassung.manual.notePlaceholder': 'Nota facoltativa…',
    'zeiterfassung.manual.createEntry': 'Salva voce',
    'zeiterfassung.manual.endAfterStart': 'La fine deve essere dopo l’inizio',
    'zeiterfassung.manual.noFutureDate': 'Nessuna data futura',
    'zeiterfassung.manual.manualBadge': 'Manuale',
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

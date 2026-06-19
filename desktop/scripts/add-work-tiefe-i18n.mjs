/**
 * i18n additions for the work module "Tiefe-Pass" (W-1 … W-5).
 * Idempotent: only inserts keys not already present. Re-run safely as the
 * pass grows. Single-brace interpolation ({var}), ICU plural where needed.
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'

const dir = resolve('src/renderer/src/i18n/messages')
const locales = ['de', 'en', 'fr', 'it']

// key -> { de, en, fr, it }. Each inserted after `anchor` (kept grouped).
const additions = {
  // W-1: task detail modal — header type label (key shown as subtitle)
  'work.panel.taskLabel': {
    anchor: 'work.myTasks.moveToProject',
    de: 'Aufgabe',
    en: 'Task',
    fr: 'Tâche',
    it: 'Attività',
  },
  // W-3: move standalone task to a project
  'work.myTasks.movedToProject': {
    anchor: 'work.myTasks.moveToProject',
    de: 'Nach {project} verschoben',
    en: 'Moved to {project}',
    fr: 'Déplacé vers {project}',
    it: 'Spostato in {project}',
  },
  // W-4: tracked hours -> draft invoice
  'work.hoursInvoice.created': {
    anchor: 'work.myTasks.moveToProject',
    de: 'Rechnungsentwurf erstellt ({number})',
    en: 'Draft invoice created ({number})',
    fr: 'Brouillon de facture créé ({number})',
    it: 'Bozza di fattura creata ({number})',
  },
  'work.hoursInvoice.createError': {
    anchor: 'work.myTasks.moveToProject',
    de: 'Rechnung konnte nicht erstellt werden',
    en: 'Could not create invoice',
    fr: 'Impossible de créer la facture',
    it: 'Impossibile creare la fattura',
  },
  'work.hoursInvoice.noEntries': {
    anchor: 'work.myTasks.moveToProject',
    de: 'Keine abrechenbaren Stunden',
    en: 'No billable hours',
    fr: 'Aucune heure facturable',
    it: 'Nessuna ora fatturabile',
  },
  'work.hoursInvoice.customer': {
    anchor: 'work.myTasks.moveToProject',
    de: 'Kunde',
    en: 'Customer',
    fr: 'Client',
    it: 'Cliente',
  },
  'work.hoursInvoice.customerPlaceholder': {
    anchor: 'work.myTasks.moveToProject',
    de: 'Kundenname',
    en: 'Customer name',
    fr: 'Nom du client',
    it: 'Nome cliente',
  },
  'work.hoursInvoice.note': {
    anchor: 'work.myTasks.moveToProject',
    de: 'Abrechnung getrackter Stunden — Projekt {project}',
    en: 'Billing of tracked hours — project {project}',
    fr: 'Facturation des heures suivies — projet {project}',
    it: 'Fatturazione delle ore tracciate — progetto {project}',
  },
  'work.hoursInvoice.emptyHint': {
    anchor: 'work.myTasks.moveToProject',
    de: 'Alle getrackten Stunden dieses Projekts sind bereits abgerechnet.',
    en: 'All tracked hours for this project have already been billed.',
    fr: 'Toutes les heures suivies de ce projet sont déjà facturées.',
    it: 'Tutte le ore tracciate di questo progetto sono già fatturate.',
  },
  // W-5: utilization report empty state
  'work.utilization.noData': {
    anchor: 'work.myTasks.moveToProject',
    de: 'Keine Auslastungsdaten für dieses Projekt.',
    en: 'No utilization data for this project.',
    fr: "Aucune donnée de charge pour ce projet.",
    it: 'Nessun dato di carico per questo progetto.',
  },
}

function insertAfter(lines, anchorKey, blockLines) {
  const idx = lines.findIndex((l) => l.trimStart().startsWith(`"${anchorKey}":`))
  if (idx === -1) throw new Error(`anchor not found: ${anchorKey}`)
  return [...lines.slice(0, idx + 1), ...blockLines, ...lines.slice(idx + 1)]
}

const report = {}
for (const loc of locales) {
  const file = resolve(dir, `${loc}.json`)
  const obj = JSON.parse(readFileSync(file, 'utf8'))
  let lines = readFileSync(file, 'utf8').split('\n')
  let added = 0
  for (const [key, def] of Object.entries(additions)) {
    if (key in obj) continue
    lines = insertAfter(lines, def.anchor, [`  ${JSON.stringify(key)}: ${JSON.stringify(def[loc])},`])
    added++
  }
  const out = lines.join('\n')
  JSON.parse(out) // validate
  writeFileSync(file, out, 'utf8')
  report[loc] = added
}
console.log(JSON.stringify(report, null, 2))

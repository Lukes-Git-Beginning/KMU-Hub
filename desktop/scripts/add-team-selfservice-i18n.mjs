import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'

const dir = resolve('src/renderer/src/i18n/messages')

// New team.selfService.* keys (self-service tab wired to real hooks).
const keys = {
  'team.selfService.actual': { de: 'Ist', en: 'Actual', fr: 'Réel', it: 'Effettivo' },
  'team.selfService.balanceSick': { de: 'Krankheit', en: 'Sick leave', fr: 'Maladie', it: 'Malattia' },
  'team.selfService.balanceSpecial': { de: 'Sonderurlaub', en: 'Special leave', fr: 'Congé spécial', it: 'Congedo speciale' },
  'team.selfService.balanceVacation': { de: 'Urlaub', en: 'Vacation', fr: 'Congés', it: 'Ferie' },
  'team.selfService.changeRequestInfo': { de: 'Änderungswünsche zu deinen Stammdaten bitte an die Personalabteilung.', en: 'Please send changes to your master data to HR.', fr: 'Adressez les modifications de vos données aux RH.', it: 'Invia le modifiche dei tuoi dati alle Risorse Umane.' },
  'team.selfService.changeRequestSubmitted': { de: 'Änderung beantragt', en: 'Change requested', fr: 'Modification demandée', it: 'Modifica richiesta' },
  'team.selfService.dateRange': { de: 'Zeitraum', en: 'Date range', fr: 'Période', it: 'Periodo' },
  'team.selfService.daysRemaining': { de: 'Tage übrig', en: 'days left', fr: 'jours restants', it: 'giorni rimasti' },
  'team.selfService.daysTaken': { de: 'Tage genommen', en: 'days taken', fr: 'jours pris', it: 'giorni presi' },
  'team.selfService.download': { de: 'Herunterladen', en: 'Download', fr: 'Télécharger', it: 'Scarica' },
  'team.selfService.downloadStarted': { de: 'Download gestartet: {label}', en: 'Download started: {label}', fr: 'Téléchargement lancé : {label}', it: 'Download avviato: {label}' },
  'team.selfService.errorEndAfterStart': { de: 'Das Enddatum muss nach dem Startdatum liegen.', en: 'The end date must be after the start date.', fr: 'La date de fin doit être postérieure à la date de début.', it: 'La data di fine deve essere successiva a quella di inizio.' },
  'team.selfService.errorRequired': { de: 'Bitte Art und Zeitraum angeben.', en: 'Please select a type and date range.', fr: 'Veuillez indiquer le type et la période.', it: 'Indica tipo e periodo.' },
  'team.selfService.from': { de: 'Von', en: 'From', fr: 'Du', it: 'Dal' },
  'team.selfService.leaveType': { de: 'Abwesenheitsart', en: 'Leave type', fr: "Type d'absence", it: 'Tipo di assenza' },
  'team.selfService.noRequests': { de: 'Noch keine Anträge gestellt.', en: 'No requests yet.', fr: "Aucune demande pour l'instant.", it: 'Nessuna richiesta finora.' },
  'team.selfService.reason': { de: 'Grund', en: 'Reason', fr: 'Motif', it: 'Motivo' },
  'team.selfService.reasonPlaceholder': { de: 'Optionale Notiz …', en: 'Optional note …', fr: 'Note facultative…', it: 'Nota facoltativa…' },
  'team.selfService.remainingShort': { de: 'übrig', en: 'left', fr: 'restants', it: 'rimasti' },
  'team.selfService.requestDialogTitle': { de: 'Antrag stellen', en: 'Submit request', fr: 'Soumettre une demande', it: 'Invia richiesta' },
  'team.selfService.selectType': { de: 'Art wählen', en: 'Select type', fr: 'Choisir le type', it: 'Seleziona tipo' },
  'team.selfService.statusApproved': { de: 'Genehmigt', en: 'Approved', fr: 'Approuvé', it: 'Approvato' },
  'team.selfService.statusPending': { de: 'Ausstehend', en: 'Pending', fr: 'En attente', it: 'In attesa' },
  'team.selfService.statusRejected': { de: 'Abgelehnt', en: 'Rejected', fr: 'Refusé', it: 'Rifiutato' },
  'team.selfService.submitRequest': { de: 'Antrag einreichen', en: 'Submit request', fr: 'Envoyer la demande', it: 'Invia richiesta' },
  'team.selfService.target': { de: 'Soll', en: 'Target', fr: 'Prévu', it: 'Previsto' },
  'team.selfService.to': { de: 'Bis', en: 'To', fr: 'Au', it: 'Al' },
}

// requestsCount changes from a plain suffix to an ICU plural (used with { count }).
const requestsCount = {
  de: '{count, plural, one {# Antrag} other {# Anträge}}',
  en: '{count, plural, one {# request} other {# requests}}',
  fr: '{count, plural, one {# demande} other {# demandes}}',
  it: '{count, plural, one {# richiesta} other {# richieste}}',
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
  if (block.length) lines = insertAfter(lines, 'team.selfService.changeRequestMock', block)

  // overwrite requestsCount value (ICU plural)
  lines = lines.map((l) =>
    l.trimStart().startsWith('"team.selfService.requestsCount":')
      ? `  "team.selfService.requestsCount": ${JSON.stringify(requestsCount[loc])},`
      : l,
  )

  const out = lines.join('\n')
  JSON.parse(out) // validate
  writeFileSync(file, out, 'utf8')
  report[loc] = { added: block.length }
}
console.log(JSON.stringify(report, null, 2))

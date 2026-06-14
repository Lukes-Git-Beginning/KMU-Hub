// P5 zeiterfassung i18n: team view + week-approval workflow. 28 keys x4.
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const __dirname = dirname(fileURLToPath(import.meta.url))
const dir = join(__dirname, '..', 'src', 'renderer', 'src', 'i18n', 'messages')

const additions = {
  de: {
    'api.hr.time.weekSubmitted': 'Woche eingereicht',
    'api.hr.time.error.weekSubmit': 'Woche konnte nicht eingereicht werden',
    'api.hr.time.weekApproved': 'Woche freigegeben',
    'api.hr.time.error.weekApprove': 'Freigabe fehlgeschlagen',
    'api.hr.time.weekRejected': 'Woche abgelehnt',
    'api.hr.time.error.weekReject': 'Ablehnung fehlgeschlagen',
    'zeiterfassung.team.tab': 'Team',
    'zeiterfassung.team.title': 'Team-Zeiterfassung',
    'zeiterfassung.team.pending': '{count} zur Freigabe',
    'zeiterfassung.team.member': 'Mitarbeiter',
    'zeiterfassung.team.week': 'Diese Woche',
    'zeiterfassung.team.overtime': 'Überstunden',
    'zeiterfassung.team.status': 'Status',
    'zeiterfassung.team.clockedIn': 'Eingestempelt',
    'zeiterfassung.team.approve': 'Freigeben',
    'zeiterfassung.team.reject': 'Ablehnen',
    'zeiterfassung.team.empty': 'Keine Teammitglieder',
    'zeiterfassung.team.emptyDesc': 'Sobald Mitarbeitende Zeiten erfassen, erscheinen sie hier.',
    'zeiterfassung.team.weekStatus.open': 'Offen',
    'zeiterfassung.team.weekStatus.submitted': 'Eingereicht',
    'zeiterfassung.team.weekStatus.approved': 'Freigegeben',
    'zeiterfassung.team.weekStatus.rejected': 'Abgelehnt',
    'zeiterfassung.week.openHint': 'Diese Woche ist noch nicht zur Freigabe eingereicht.',
    'zeiterfassung.week.submittedHint': 'Woche eingereicht — wartet auf Freigabe.',
    'zeiterfassung.week.approvedHint': 'Woche freigegeben.',
    'zeiterfassung.week.rejectedHint': 'Woche abgelehnt — bitte korrigieren und erneut einreichen.',
    'zeiterfassung.week.submit': 'Woche einreichen',
    'zeiterfassung.week.resubmit': 'Erneut einreichen',
  },
  en: {
    'api.hr.time.weekSubmitted': 'Week submitted',
    'api.hr.time.error.weekSubmit': 'Could not submit week',
    'api.hr.time.weekApproved': 'Week approved',
    'api.hr.time.error.weekApprove': 'Approval failed',
    'api.hr.time.weekRejected': 'Week rejected',
    'api.hr.time.error.weekReject': 'Rejection failed',
    'zeiterfassung.team.tab': 'Team',
    'zeiterfassung.team.title': 'Team time tracking',
    'zeiterfassung.team.pending': '{count} to approve',
    'zeiterfassung.team.member': 'Employee',
    'zeiterfassung.team.week': 'This week',
    'zeiterfassung.team.overtime': 'Overtime',
    'zeiterfassung.team.status': 'Status',
    'zeiterfassung.team.clockedIn': 'Clocked in',
    'zeiterfassung.team.approve': 'Approve',
    'zeiterfassung.team.reject': 'Reject',
    'zeiterfassung.team.empty': 'No team members',
    'zeiterfassung.team.emptyDesc': 'Once employees track time, they appear here.',
    'zeiterfassung.team.weekStatus.open': 'Open',
    'zeiterfassung.team.weekStatus.submitted': 'Submitted',
    'zeiterfassung.team.weekStatus.approved': 'Approved',
    'zeiterfassung.team.weekStatus.rejected': 'Rejected',
    'zeiterfassung.week.openHint': "This week hasn't been submitted for approval yet.",
    'zeiterfassung.week.submittedHint': 'Week submitted — awaiting approval.',
    'zeiterfassung.week.approvedHint': 'Week approved.',
    'zeiterfassung.week.rejectedHint': 'Week rejected — please correct and resubmit.',
    'zeiterfassung.week.submit': 'Submit week',
    'zeiterfassung.week.resubmit': 'Resubmit',
  },
  fr: {
    'api.hr.time.weekSubmitted': 'Semaine soumise',
    'api.hr.time.error.weekSubmit': 'Impossible de soumettre la semaine',
    'api.hr.time.weekApproved': 'Semaine approuvée',
    'api.hr.time.error.weekApprove': 'Échec de l’approbation',
    'api.hr.time.weekRejected': 'Semaine refusée',
    'api.hr.time.error.weekReject': 'Échec du refus',
    'zeiterfassung.team.tab': 'Équipe',
    'zeiterfassung.team.title': 'Suivi du temps d’équipe',
    'zeiterfassung.team.pending': '{count} à approuver',
    'zeiterfassung.team.member': 'Employé',
    'zeiterfassung.team.week': 'Cette semaine',
    'zeiterfassung.team.overtime': 'Heures supp.',
    'zeiterfassung.team.status': 'Statut',
    'zeiterfassung.team.clockedIn': 'Pointé',
    'zeiterfassung.team.approve': 'Approuver',
    'zeiterfassung.team.reject': 'Refuser',
    'zeiterfassung.team.empty': 'Aucun membre d’équipe',
    'zeiterfassung.team.emptyDesc': 'Dès que les employés saisissent du temps, ils apparaissent ici.',
    'zeiterfassung.team.weekStatus.open': 'Ouvert',
    'zeiterfassung.team.weekStatus.submitted': 'Soumis',
    'zeiterfassung.team.weekStatus.approved': 'Approuvé',
    'zeiterfassung.team.weekStatus.rejected': 'Refusé',
    'zeiterfassung.week.openHint': 'Cette semaine n’a pas encore été soumise.',
    'zeiterfassung.week.submittedHint': 'Semaine soumise — en attente d’approbation.',
    'zeiterfassung.week.approvedHint': 'Semaine approuvée.',
    'zeiterfassung.week.rejectedHint': 'Semaine refusée — corrigez et resoumettez.',
    'zeiterfassung.week.submit': 'Soumettre la semaine',
    'zeiterfassung.week.resubmit': 'Resoumettre',
  },
  it: {
    'api.hr.time.weekSubmitted': 'Settimana inviata',
    'api.hr.time.error.weekSubmit': 'Impossibile inviare la settimana',
    'api.hr.time.weekApproved': 'Settimana approvata',
    'api.hr.time.error.weekApprove': 'Approvazione non riuscita',
    'api.hr.time.weekRejected': 'Settimana rifiutata',
    'api.hr.time.error.weekReject': 'Rifiuto non riuscito',
    'zeiterfassung.team.tab': 'Team',
    'zeiterfassung.team.title': 'Rilevazione del team',
    'zeiterfassung.team.pending': '{count} da approvare',
    'zeiterfassung.team.member': 'Dipendente',
    'zeiterfassung.team.week': 'Questa settimana',
    'zeiterfassung.team.overtime': 'Straordinari',
    'zeiterfassung.team.status': 'Stato',
    'zeiterfassung.team.clockedIn': 'Timbrato',
    'zeiterfassung.team.approve': 'Approva',
    'zeiterfassung.team.reject': 'Rifiuta',
    'zeiterfassung.team.empty': 'Nessun membro del team',
    'zeiterfassung.team.emptyDesc': 'Quando i dipendenti registrano il tempo, appaiono qui.',
    'zeiterfassung.team.weekStatus.open': 'Aperto',
    'zeiterfassung.team.weekStatus.submitted': 'Inviato',
    'zeiterfassung.team.weekStatus.approved': 'Approvato',
    'zeiterfassung.team.weekStatus.rejected': 'Rifiutato',
    'zeiterfassung.week.openHint': 'Questa settimana non è ancora stata inviata.',
    'zeiterfassung.week.submittedHint': 'Settimana inviata — in attesa di approvazione.',
    'zeiterfassung.week.approvedHint': 'Settimana approvata.',
    'zeiterfassung.week.rejectedHint': 'Settimana rifiutata — correggi e reinvia.',
    'zeiterfassung.week.submit': 'Invia settimana',
    'zeiterfassung.week.resubmit': 'Reinvia',
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

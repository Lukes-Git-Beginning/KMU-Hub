import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'

const dir = resolve('src/renderer/src/i18n/messages')
const anchor = 'kontakte.detail.tags'

// Count-neutral phrasing where counts appear (ICU plural broken project-wide).
const K = {
  'common.add': { de: 'Hinzufügen', en: 'Add', fr: 'Ajouter', it: 'Aggiungi' },

  'team.page.tab.payroll': { de: 'Lohnvorbereitung', en: 'Payroll prep', fr: 'Préparation paie', it: 'Preparazione paghe' },

  // Settings panel
  'team.settings.title': { de: 'Team', en: 'Team', fr: 'Équipe', it: 'Team' },
  'team.settings.subtitle': { de: 'Einstellungen für Personal, HR und Lohn-Anbindung', en: 'Settings for staff, HR and payroll connection', fr: 'Paramètres personnel, RH et paie', it: 'Impostazioni personale, HR e paghe' },
  'team.settings.personal.title': { de: 'Ansicht', en: 'View', fr: 'Affichage', it: 'Visualizzazione' },
  'team.settings.personal.desc': { de: 'Womit das Team-Modul startet', en: 'How the team module opens', fr: 'Ouverture du module équipe', it: 'Apertura del modulo team' },
  'team.settings.hr.title': { de: 'Personal & HR', en: 'Staff & HR', fr: 'Personnel et RH', it: 'Personale e HR' },
  'team.settings.hr.desc': { de: 'Abteilungen, Rollen, Urlaubsarten, Arbeitszeit', en: 'Departments, roles, leave types, working time', fr: 'Services, rôles, congés, temps de travail', it: 'Reparti, ruoli, ferie, orario' },
  'team.settings.payroll.title': { de: 'DATEV-Lohn & Schnittstellen', en: 'DATEV payroll & interfaces', fr: 'Paie DATEV et interfaces', it: 'Paghe DATEV e interfacce' },
  'team.settings.payroll.desc': { de: 'Verbindung, Lohnarten- und Abwesenheits-Zuordnung', en: 'Connection, wage-type and absence mapping', fr: 'Connexion, types de salaire et absences', it: 'Connessione, voci paga e assenze' },

  // Personal prefs
  'team.prefs.startTab.label': { de: 'Start-Ansicht', en: 'Start view', fr: 'Vue de démarrage', it: 'Vista iniziale' },
  'team.prefs.startTab.hint': { de: 'Welcher Tab beim Öffnen erscheint', en: 'Which tab opens first', fr: 'Onglet affiché à l’ouverture', it: 'Scheda mostrata all’apertura' },
  'team.prefs.startTab.last': { de: 'Zuletzt verwendet', en: 'Last used', fr: 'Dernier utilisé', it: 'Ultimo usato' },
  'team.prefs.startTab.members': { de: 'Mitarbeiter', en: 'Members', fr: 'Membres', it: 'Membri' },
  'team.prefs.startTab.requests': { de: 'Anträge', en: 'Requests', fr: 'Demandes', it: 'Richieste' },
  'team.prefs.startTab.absences': { de: 'Abwesenheiten', en: 'Absences', fr: 'Absences', it: 'Assenze' },
  'team.prefs.startTab.payroll': { de: 'Lohnvorbereitung', en: 'Payroll prep', fr: 'Préparation paie', it: 'Preparazione paghe' },
  'team.prefs.view.label': { de: 'Mitarbeiter-Ansicht', en: 'Member view', fr: 'Vue des membres', it: 'Vista membri' },
  'team.prefs.view.hint': { de: 'Standard beim Öffnen der Mitarbeiterliste', en: 'Default for the member list', fr: 'Par défaut pour la liste', it: 'Predefinito per l’elenco' },
  'team.prefs.view.grid': { de: 'Kacheln', en: 'Cards', fr: 'Cartes', it: 'Schede' },
  'team.prefs.view.list': { de: 'Liste', en: 'List', fr: 'Liste', it: 'Elenco' },

  // Payroll settings (config)
  'team.payroll.beraterNr': { de: 'Beraternummer', en: 'Advisor number', fr: 'N° de conseiller', it: 'N° consulente' },
  'team.payroll.orderHint': { de: 'DATEV-Ordnungsbegriff — vom Lohnbüro', en: 'DATEV identifier — from your payroll office', fr: 'Identifiant DATEV — du bureau de paie', it: 'Identificativo DATEV — dall’ufficio paghe' },
  'team.payroll.mandantNr': { de: 'Mandantennummer', en: 'Client number', fr: 'N° de client', it: 'N° cliente' },
  'team.payroll.target': { de: 'Zielsystem', en: 'Target system', fr: 'Système cible', it: 'Sistema di destinazione' },
  'team.payroll.targetLug': { de: 'DATEV Lohn und Gehalt', en: 'DATEV Lohn und Gehalt', fr: 'DATEV Lohn und Gehalt', it: 'DATEV Lohn und Gehalt' },
  'team.payroll.targetLodas': { de: 'DATEV LODAS', en: 'DATEV LODAS', fr: 'DATEV LODAS', it: 'DATEV LODAS' },
  'team.payroll.transfer': { de: 'Übergabeart', en: 'Transfer method', fr: 'Mode de transfert', it: 'Modalità di trasferimento' },
  'team.payroll.transferFile': { de: 'Datei-Export', en: 'File export', fr: 'Export fichier', it: 'Esportazione file' },
  'team.payroll.transferService': { de: 'Datenservice (später)', en: 'Data service (later)', fr: 'Service de données (plus tard)', it: 'Servizio dati (in seguito)' },
  'team.payroll.wageTypes': { de: 'Lohnarten-Zuordnung', en: 'Wage-type mapping', fr: 'Affectation des types de salaire', it: 'Mappatura voci di paga' },
  'team.payroll.wageTypesHint': { de: 'Cosmi-Kategorie → DATEV-Lohnart-Nr. (vom Lohnbüro bestätigen lassen)', en: 'Cosmi category → DATEV wage-type no. (confirm with payroll office)', fr: 'Catégorie Cosmi → n° DATEV (à confirmer)', it: 'Categoria Cosmi → n° DATEV (da confermare)' },
  'team.payroll.absences': { de: 'Abwesenheits-Zuordnung', en: 'Absence mapping', fr: 'Affectation des absences', it: 'Mappatura assenze' },
  'team.payroll.absencesHint': { de: 'Welche Abwesenheiten mit welchem Schlüssel exportiert werden', en: 'Which absences export with which key', fr: 'Quelles absences exporter et avec quelle clé', it: 'Quali assenze esportare e con quale chiave' },
  'team.payroll.absenceKey': { de: 'Schlüssel', en: 'Key', fr: 'Clé', it: 'Chiave' },
  'team.payroll.groups': { de: 'Abrechnungsgruppen', en: 'Payroll groups', fr: 'Groupes de paie', it: 'Gruppi di paga' },
  'team.payroll.groupsHint': { de: 'z. B. Festangestellte und Stundenlöhner getrennt abrechnen', en: 'e.g. process salaried and hourly staff separately', fr: 'p. ex. séparer salariés et horaires', it: 'es. separare stipendiati e orari' },
  'team.payroll.newGroup': { de: 'Neue Gruppe', en: 'New group', fr: 'Nouveau groupe', it: 'Nuovo gruppo' },

  // Payroll run (working surface)
  'team.payroll.run.period': { de: 'Abrechnungszeitraum', en: 'Period', fr: 'Période', it: 'Periodo' },
  'team.payroll.run.group': { de: 'Abrechnungsgruppe', en: 'Payroll group', fr: 'Groupe de paie', it: 'Gruppo di paga' },
  'team.payroll.run.changes': { de: 'Änderungen: {count}', en: 'Changes: {count}', fr: 'Modifications : {count}', it: 'Modifiche: {count}' },
  'team.payroll.run.locked': { de: 'Freigegeben', en: 'Locked', fr: 'Validé', it: 'Bloccato' },
  'team.payroll.run.unlock': { de: 'Freigabe aufheben', en: 'Unlock', fr: 'Annuler', it: 'Sblocca' },
  'team.payroll.run.approve': { de: 'Prüfen & freigeben', en: 'Review & approve', fr: 'Vérifier & valider', it: 'Verifica & approva' },
  'team.payroll.run.export': { de: 'Export {target}', en: 'Export {target}', fr: 'Export {target}', it: 'Esporta {target}' },
  'team.payroll.run.exported': { de: 'Lohnlauf exportiert ({count} Mitarbeiter)', en: 'Payroll run exported ({count} employees)', fr: 'Paie exportée ({count} employés)', it: 'Paghe esportate ({count} dipendenti)' },
  'team.payroll.run.new': { de: 'Neu', en: 'New', fr: 'Nouveau', it: 'Nuovo' },
  'team.payroll.run.salaryChange': { de: 'Gehalt', en: 'Salary', fr: 'Salaire', it: 'Stipendio' },
  'team.payroll.run.leaver': { de: 'Austritt', en: 'Leaver', fr: 'Départ', it: 'Uscita' },
  'team.payroll.run.emptyTitle': { de: 'Keine Mitarbeiter in dieser Gruppe', en: 'No employees in this group', fr: 'Aucun employé dans ce groupe', it: 'Nessun dipendente in questo gruppo' },
  'team.payroll.run.emptyDesc': { de: 'Wähle eine andere Abrechnungsgruppe oder lege Mitarbeiter an.', en: 'Pick another payroll group or add employees.', fr: 'Choisissez un autre groupe ou ajoutez des employés.', it: 'Scegli un altro gruppo o aggiungi dipendenti.' },
  'team.payroll.run.employee': { de: 'Mitarbeiter', en: 'Employee', fr: 'Employé', it: 'Dipendente' },
  'team.payroll.run.masterData': { de: 'Stammdaten', en: 'Master data', fr: 'Données de base', it: 'Dati anagrafici' },
  'team.payroll.run.movementData': { de: 'Bewegungsdaten', en: 'Movement data', fr: 'Données variables', it: 'Dati variabili' },
  'team.payroll.run.oneTime': { de: 'Einmalbezug', en: 'One-time', fr: 'Ponctuel', it: 'Una tantum' },
  'team.payroll.run.hours': { de: '{count} Std', en: '{count} h', fr: '{count} h', it: '{count} h' },
  'team.payroll.run.overtime': { de: '{count} Überstd.', en: '{count} OT', fr: '{count} h supp.', it: '{count} straord.' },
  'team.payroll.run.absenceDays': { de: '{count} Tage Abw.', en: '{count} days abs.', fr: '{count} j abs.', it: '{count} gg ass.' },
  'team.payroll.run.history': { de: 'Verlauf', en: 'History', fr: 'Historique', it: 'Cronologia' },
  'team.payroll.run.historyEmpty': { de: 'Noch keine Lohnläufe exportiert.', en: 'No payroll runs exported yet.', fr: 'Aucune paie exportée.', it: 'Nessuna paga esportata.' },
  'team.payroll.run.employees': { de: '{count} Mitarbeiter', en: '{count} employees', fr: '{count} employés', it: '{count} dipendenti' },
}

const report = {}
for (const loc of ['de', 'en', 'fr', 'it']) {
  const file = resolve(dir, `${loc}.json`)
  const obj = JSON.parse(readFileSync(file, 'utf8'))
  let lines = readFileSync(file, 'utf8').split('\n')
  const nk = Object.keys(K).filter((k) => !(k in obj)).sort()
  if (!nk.length) { report[loc] = 0; continue }
  const block = nk.map((k) => `  ${JSON.stringify(k)}: ${JSON.stringify(K[k][loc])},`)
  const idx = lines.findIndex((l) => l.trimStart().startsWith(`"${anchor}":`))
  if (idx === -1) throw new Error(`anchor ${anchor} missing in ${loc}`)
  lines = [...lines.slice(0, idx + 1), ...block, ...lines.slice(idx + 1)]
  const out = lines.join('\n')
  JSON.parse(out)
  writeFileSync(file, out, 'utf8')
  report[loc] = block.length
}
console.log(JSON.stringify(report))

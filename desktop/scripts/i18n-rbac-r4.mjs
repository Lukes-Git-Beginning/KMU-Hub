/**
 * i18n batch for RBAC R-4 (HR data category depth): catalogue labels for the
 * three new team capability keys (`team:self:propose`, `team:directory:full`,
 * `team:employee:offboard`) and the new absence FILE drawer subject
 * (`team:absence_data:view` — separate from the calendar board key
 * `team:absence:read`). UI keys for the new R-4 surfaces (offboard dialog,
 * change-request inbox, self-service flow) are collected here too once the
 * build lands.
 *
 * Inserts new keys at their alphabetical position relative to the existing
 * order (same as i18n-rbac-r3b5.mjs — no global re-sort). Keys already
 * present are skipped, so listing safety duplicates is harmless.
 *
 * Run: node scripts/i18n-rbac-r4.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))
const MESSAGES_DIR = join(here, '..', 'src', 'renderer', 'src', 'i18n', 'messages')
const LOCALES = ['de', 'en', 'fr', 'it']

/** @type {Record<string, [string, string, string, string]>} key → [de, en, fr, it] */
const ADD = {
  // ── Subjects ──────────────────────────────────────────────────────────────
  'rbac.subject.absence_data': ['Abwesenheits-Salden', 'Absence balances', "Soldes d'absence", 'Saldi assenze'],
  'rbac.subject.directory': ['Mitarbeiter-Verzeichnis', 'Employee directory', 'Annuaire du personnel', 'Elenco del personale'],
  'rbac.subject.self': ['Eigene Daten', 'Own data', 'Données propres', 'Dati propri'],
  // ── Actions ───────────────────────────────────────────────────────────────
  'rbac.action.full': ['Alle anzeigen', 'View all', 'Tout afficher', 'Mostra tutti'],
  'rbac.action.offboard': ['Austritt durchführen', 'Offboard', 'Effectuer la sortie', "Gestire l'uscita"],
  'rbac.action.propose': ['Änderungen vorschlagen', 'Propose changes', 'Proposer des modifications', 'Proporre modifiche'],
  // ── R-4 surface keys (offboard dialog, change-request flow, directory) ────
  'api.hr.employee.error.offboard': ['Fehler beim Austritt', 'Offboarding failed', 'Échec de la sortie', "Errore durante l'uscita"],
  'api.hr.employee.offboarded': ['Austritt eingeleitet', 'Offboarding initiated', 'Sortie lancée', 'Uscita avviata'],
  'common.more': ['mehr', 'more', 'plus', 'altri'],
  'shared.actions': ['Aktionen', 'Actions', 'Actions', 'Azioni'],
  'team.changeRequest.approve': ['Genehmigen', 'Approve', 'Approuver', 'Approva'],
  'team.changeRequest.approvedBy': ['Genehmigt von', 'Approved by', 'Approuvé par', 'Approvato da'],
  'team.changeRequest.cancel': ['Antrag stornieren', 'Cancel request', 'Annuler la demande', 'Annulla richiesta'],
  'team.changeRequest.currentValue': ['Aktueller Wert', 'Current value', 'Valeur actuelle', 'Valore attuale'],
  'team.changeRequest.inboxEmpty': ['Keine offenen Anträge', 'No open requests', 'Aucune demande en attente', 'Nessuna richiesta aperta'],
  'team.changeRequest.inboxEmptyHint': ['Alle Profiländerungen wurden bearbeitet.', 'All profile change requests have been processed.', 'Toutes les demandes de modification ont été traitées.', 'Tutte le richieste di modifica sono state elaborate.'],
  'team.changeRequest.inboxSubtitle': ['{count, plural, one {1 ausstehender Antrag} other {{count} ausstehende Anträge}}', '{count, plural, one {1 pending request} other {{count} pending requests}}', '{count, plural, one {1 demande en attente} other {{count} demandes en attente}}', '{count, plural, one {1 richiesta in attesa} other {{count} richieste in attesa}}'],
  'team.changeRequest.inboxTitle': ['Profil-Änderungsanträge', 'Profile change requests', 'Demandes de modification de profil', 'Richieste di modifica profilo'],
  'team.changeRequest.newValue': ['Neu', 'After', 'Après', 'Dopo'],
  'team.changeRequest.newValuePlaceholder': ['Neuen Wert für {field} eingeben', 'Enter new value for {field}', 'Saisir la nouvelle valeur pour {field}', 'Inserire il nuovo valore per {field}'],
  'team.changeRequest.oldValue': ['Alt', 'Before', 'Avant', 'Prima'],
  'team.changeRequest.pendingBadge': ['Änderung ausstehend', 'Change pending', 'Modification en attente', 'Modifica in attesa'],
  'team.changeRequest.proposeTitle': ['Änderung vorschlagen – {field}', 'Propose change – {field}', 'Proposer une modification – {field}', 'Proporre modifica – {field}'],
  'team.changeRequest.reject': ['Ablehnen', 'Reject', 'Refuser', 'Rifiuta'],
  'team.changeRequest.rejectConfirm': ['Ablehnen', 'Reject', 'Refuser', 'Rifiuta'],
  'team.changeRequest.rejectDialogTitle': ['Antrag ablehnen', 'Reject request', 'Refuser la demande', 'Rifiuta richiesta'],
  'team.changeRequest.rejectedBy': ['Abgelehnt von', 'Rejected by', 'Refusé par', 'Rifiutato da'],
  'team.changeRequest.rejectionReason': ['Ablehnungsgrund', 'Rejection reason', 'Motif du refus', 'Motivo del rifiuto'],
  'team.changeRequest.rejectionReasonLabel': ['Grund für die Ablehnung', 'Reason for rejection', 'Motif du refus', 'Motivo del rifiuto'],
  'team.changeRequest.rejectionReasonPlaceholder': ['Bitte Ablehnung begründen …', 'Please state the reason …', 'Veuillez indiquer le motif …', 'Indicare il motivo …'],
  'team.changeRequest.showDecided': ['Erledigte anzeigen ({count})', 'Show decided ({count})', 'Afficher traitées ({count})', 'Mostra elaborate ({count})'],
  'team.changeRequest.statusApproved': ['Genehmigt', 'Approved', 'Approuvé', 'Approvato'],
  'team.changeRequest.statusPending': ['Ausstehend', 'Pending', 'En attente', 'In attesa'],
  'team.changeRequest.statusRejected': ['Abgelehnt', 'Rejected', 'Refusé', 'Rifiutato'],
  'team.changeRequest.submit': ['Antrag einreichen', 'Submit request', 'Soumettre la demande', 'Invia richiesta'],
  'team.changeRequest.submitHint': ['Dein Antrag wird von HR geprüft. Das Feld ist bis zur Entscheidung gesperrt.', 'Your request will be reviewed by HR. The field stays locked until decided.', "Votre demande sera examinée par les RH. Le champ reste verrouillé jusqu'à la décision.", 'La richiesta sarà esaminata dalle risorse umane. Il campo resta bloccato fino alla decisione.'],
  'team.detail.absenceData': ['Abwesenheiten', 'Absences', 'Absences', 'Assenze'],
  'team.detail.email': ['E-Mail', 'Email', 'E-mail', 'E-mail'],
  'team.directory.restrictedHint': ['Eingeschränkte Ansicht – du siehst dein Arbeitsumfeld', 'Restricted view – you see your work environment', 'Vue restreinte – vous voyez votre environnement de travail', 'Vista limitata – vedi il tuo ambiente di lavoro'],
  'team.offboard.backfill': ['Nachbesetzung geplant', 'Backfill planned', 'Remplacement prévu', 'Sostituzione prevista'],
  'team.offboard.backfillHint': ['Die Position soll neu besetzt werden', 'The position will be filled again', 'Le poste sera pourvu à nouveau', 'La posizione verrà ricoperta di nuovo'],
  'team.offboard.confirm': ['Austritt bestätigen', 'Confirm offboarding', 'Confirmer la sortie', 'Conferma uscita'],
  'team.offboard.confirming': ['Wird verarbeitet …', 'Processing …', 'En cours …', 'In elaborazione …'],
  'team.offboard.consequence.loginLocked': ['Login wird am {date} gesperrt', 'Login will be locked on {date}', 'La connexion sera bloquée le {date}', 'Il login verrà bloccato il {date}'],
  'team.offboard.consequence.rolesRevoked': ['Rollen und Rechte werden entzogen', 'Roles and permissions will be revoked', 'Les rôles et droits seront révoqués', 'Ruoli e permessi verranno revocati'],
  'team.offboard.consequence.seatFreed': ['Lizenzplatz wird freigegeben', 'License seat will be freed', 'La licence sera libérée', 'Il posto licenza verrà liberato'],
  'team.offboard.consequences': ['Konsequenzen', 'Consequences', 'Conséquences', 'Conseguenze'],
  'team.offboard.dependentsHint': ['Verantwortung muss übertragen werden, bevor der Austritt bestätigt werden kann.', 'Responsibilities must be reassigned before confirming.', 'Les responsabilités doivent être transférées avant de confirmer.', 'Le responsabilità devono essere trasferite prima della conferma.'],
  'team.offboard.dependentsWarning': ['{count, plural, one {1 Person berichtet an diese Person} other {{count} Personen berichten an diese Person}}', '{count, plural, one {1 person reports to this person} other {{count} people report to this person}}', '{count, plural, one {1 personne dépend de cette personne} other {{count} personnes dépendent de cette personne}}', '{count, plural, one {1 persona riporta a questa persona} other {{count} persone riportano a questa persona}}'],
  'team.offboard.destructiveHint': ['Dieser Vorgang kann nicht rückgängig gemacht werden. {name} verliert sofort alle Zugänge.', 'This action cannot be undone. {name} will immediately lose all access.', 'Cette action est irréversible. {name} perdra immédiatement tous les accès.', 'Questa azione non può essere annullata. {name} perderà immediatamente tutti gli accessi.'],
  'team.offboard.exitDate': ['Austrittsdatum', 'Exit date', 'Date de sortie', 'Data di uscita'],
  'team.offboard.exitType': ['Austrittsart', 'Exit type', 'Type de départ', 'Tipo di uscita'],
  'team.offboard.exitType.resignation': ['Kündigung Mitarbeiter', 'Resignation', 'Démission', 'Dimissioni'],
  'team.offboard.exitType.termination': ['Kündigung Arbeitgeber', 'Termination', 'Licenciement', 'Licenziamento'],
  'team.offboard.exitType.mutual_termination': ['Aufhebungsvertrag', 'Mutual termination', 'Rupture conventionnelle', 'Rescissione consensuale'],
  'team.offboard.exitType.retirement': ['Renteneintritt', 'Retirement', 'Retraite', 'Pensionamento'],
  'team.offboard.initiate': ['Austritt einleiten', 'Initiate offboarding', 'Lancer la sortie', 'Avvia uscita'],
  'team.offboard.lastWorkDay': ['Letzter Arbeitstag', 'Last working day', 'Dernier jour de travail', 'Ultimo giorno lavorativo'],
  'team.offboard.reason': ['Grund', 'Reason', 'Motif', 'Motivo'],
  'team.offboard.reasonPlaceholder': ['Optionaler Kommentar zum Austritt …', 'Optional comment …', 'Commentaire optionnel …', 'Commento opzionale …'],
  'team.offboard.successorHint': ['Diese Person übernimmt die Vorgesetzten-Rolle für alle betroffenen Mitarbeitenden.', 'This person becomes the manager for all affected employees.', 'Cette personne devient responsable de tous les employés concernés.', 'Questa persona diventa responsabile di tutti i dipendenti interessati.'],
  'team.offboard.successorLabel': ['Verantwortung übernimmt', 'Responsibility transfers to', 'Responsabilité transférée à', 'Responsabilità trasferita a'],
  'team.offboard.successorPlaceholder': ['Mitarbeiter auswählen …', 'Select employee …', 'Sélectionner un employé …', 'Seleziona dipendente …'],
  'team.offboard.title': ['Austritt einleiten – {name}', 'Initiate offboarding – {name}', 'Lancer la sortie – {name}', 'Avvia uscita – {name}'],
  'team.page.action.reactivate': ['Reaktivieren', 'Reactivate', 'Réactiver', 'Riattiva'],
  'team.page.reactivated': ['{name} wurde wieder aktiviert', '{name} has been reactivated', '{name} a été réactivé', '{name} è stato riattivato'],
  'team.page.training.descriptionLabel': ['Beschreibung', 'Description', 'Description', 'Descrizione'],
  'team.page.training.fieldDescription': ['Beschreibung', 'Description', 'Description', 'Descrizione'],
  'team.page.training.fieldDescriptionPlaceholder': ['Kurzbeschreibung der Schulung …', 'Short description …', 'Brève description …', 'Breve descrizione …'],
  'team.page.training.fieldMaterials': ['Materialien', 'Materials', 'Supports', 'Materiali'],
  'team.page.training.fieldObjectives': ['Lernziele', 'Learning objectives', 'Objectifs', 'Obiettivi'],
  'team.page.training.fieldObjectivesPlaceholder': ['Was soll vermittelt werden?', 'What should be taught?', 'Que faut-il transmettre ?', 'Cosa deve essere insegnato?'],
  'team.page.training.materialsLabel': ['Materialien', 'Materials', 'Supports', 'Materiali'],
  'team.page.training.materialsUpload': ['Dateien hinzufügen', 'Add files', 'Ajouter des fichiers', 'Aggiungi file'],
  'team.page.training.noExpiry': ['Ohne Ablauf', 'No expiry', 'Sans expiration', 'Senza scadenza'],
  'team.page.training.noParticipants': ['Keine Teilnehmer erfasst', 'No participants recorded', 'Aucun participant', 'Nessun partecipante'],
  'team.page.training.objectivesLabel': ['Lernziele', 'Learning objectives', 'Objectifs', 'Obiettivi'],
  'team.page.training.participantsLabel': ['Teilnehmer', 'Participants', 'Participants', 'Partecipanti'],
  'team.page.training.validity': ['Gültigkeit', 'Validity', 'Validité', 'Validità'],
  'team.page.viewMode': ['Ansicht', 'View', 'Affichage', 'Vista'],
  'team.page.viewModeGrid': ['Kachel-Ansicht', 'Grid view', 'Vue grille', 'Vista griglia'],
  'team.page.viewModeList': ['Listen-Ansicht', 'List view', 'Vue liste', 'Vista elenco'],
  'team.personnelDocs.noAccess': ['Kein Zugriff', 'No access', 'Accès refusé', 'Accesso negato'],
  'team.personnelDocs.noAccessDesc': ['Du hast keine Berechtigung, diese Personalakte einzusehen.', 'You do not have permission to view this personnel file.', "Vous n'avez pas l'autorisation de consulter ce dossier.", 'Non hai il permesso di visualizzare questo fascicolo.'],
  'team.personnelDocs.noEmployeeSelected': ['Kein Mitarbeiter ausgewählt', 'No employee selected', 'Aucun employé sélectionné', 'Nessun dipendente selezionato'],
  'team.personnelDocs.noEmployeeSelectedDesc': ['Wähle oben einen Mitarbeiter aus, um seine Personalakte zu öffnen.', 'Select an employee above to open their personnel file.', 'Sélectionnez un employé ci-dessus pour ouvrir son dossier.', 'Seleziona un dipendente in alto per aprire il suo fascicolo.'],
  'team.personnelDocs.scopeHint': ['Du siehst nur Akten in deiner Verantwortung', 'You only see files within your responsibility', 'Vous ne voyez que les dossiers de votre périmètre', 'Vedi solo i fascicoli di tua responsabilità'],
  'team.personnelDocs.selectEmployee': ['Mitarbeiter', 'Employee', 'Employé', 'Dipendente'],
  'team.personnelDocs.selectEmployeePlaceholder': ['Akte auswählen …', 'Select file …', 'Sélectionner un dossier …', 'Seleziona un fascicolo …'],
  'team.personnelDocs.title': ['Personalakte', 'Personnel file', 'Dossier du personnel', 'Fascicolo del personale'],
  'team.selfService.addressCity': ['Wohnort', 'City', 'Ville', 'Città'],
  'team.selfService.addressPostalCode': ['PLZ', 'Postal code', 'Code postal', 'CAP'],
  'team.selfService.addressStreet': ['Straße', 'Street', 'Rue', 'Via'],
  'team.selfService.editableFields': ['Änderbare Felder', 'Editable fields', 'Champs modifiables', 'Campi modificabili'],
  'team.selfService.emergencyContactName': ['Notfallkontakt', 'Emergency contact', "Contact d'urgence", "Contatto d'emergenza"],
  'team.selfService.emergencyContactPhone': ['Notfalltelefon', 'Emergency phone', "Téléphone d'urgence", 'Telefono di emergenza'],
  'team.selfService.mobile': ['Mobilnummer', 'Mobile', 'Mobile', 'Cellulare'],
  'team.selfService.noProposehint': ['Änderungen laufen über deine Führungskraft.', 'Request changes through your manager.', 'Les modifications passent par votre responsable.', 'Le modifiche passano tramite il tuo responsabile.'],
  'team.selfService.noStatements': ['Keine Abrechnungen vorhanden.', 'No statements available.', 'Aucun bulletin disponible.', 'Nessuna busta paga disponibile.'],
  'team.selfService.phoneLabel': ['Telefon', 'Phone', 'Téléphone', 'Telefono'],
  'team.selfService.preview': ['Vorschau', 'Preview', 'Aperçu', 'Anteprima'],
  'team.selfService.propose': ['Ändern', 'Propose change', 'Proposer', 'Proponi'],
  'team.selfService.proposeErrorEmpty': ['Bitte neuen Wert eingeben.', 'Please enter a new value.', 'Veuillez saisir une nouvelle valeur.', 'Inserire un nuovo valore.'],
  'team.selfService.salaryNoAccess': ['Keine Berechtigung', 'No permission', "Pas d'autorisation", 'Nessun permesso'],
  'team.selfService.salaryNoAccessHint': ['Gehaltsabrechnungen sind nur sichtbar, wenn deine Rolle sie freischaltet.', 'Salary statements are only visible if your role grants access.', 'Les fiches de paie ne sont visibles que si votre rôle y donne accès.', 'Le buste paga sono visibili solo se il tuo ruolo lo consente.'],
  'team.selfService.salaryPreviewHint': ['Im Produktivbetrieb erscheinen hier die offiziellen Abrechnungen aus der Lohnbuchhaltung.', 'In production the official statements from payroll appear here.', 'En production, les bulletins officiels de la paie apparaissent ici.', 'In produzione qui compaiono le buste paga ufficiali.'],
  'team.selfService.statusCancelled': ['Storniert', 'Cancelled', 'Annulé', 'Annullato'],
  'team.wizard.rolesNoPermission': ['Ohne Berechtigung zur Rollenvergabe wird die Person als „Mitarbeiter" angelegt.', 'Without role-assignment permission the person is created as "Employee".', "Sans permission d'attribution de rôle, la personne est créée comme « Employé ».", 'Senza autorizzazione all\'assegnazione dei ruoli la persona viene creata come "Dipendente".'],
}

for (const [li, locale] of LOCALES.entries()) {
  const file = join(MESSAGES_DIR, `${locale}.json`)
  const data = JSON.parse(readFileSync(file, 'utf8'))

  // Insert new keys at their alphabetical position relative to existing order.
  const pending = Object.keys(ADD)
    .filter((k) => !(k in data))
    .sort()
  const out = {}
  const existing = Object.keys(data)
  let pi = 0
  for (const key of existing) {
    while (pi < pending.length && pending[pi] < key) {
      out[pending[pi]] = ADD[pending[pi]][li]
      pi++
    }
    out[key] = data[key]
  }
  while (pi < pending.length) {
    out[pending[pi]] = ADD[pending[pi]][li]
    pi++
  }

  writeFileSync(file, JSON.stringify(out, null, 2) + '\n', 'utf8')
  console.log(`${locale}: +${pending.length} keys`)
}
console.log('done')

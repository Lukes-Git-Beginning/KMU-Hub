import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
const dir = resolve('src/renderer/src/i18n/messages')

const keys = {
  // Tabs
  'finanzen.tabs.openItems': { de: 'Offene Posten', en: 'Open items', fr: 'Postes ouverts', it: 'Partite aperte' },
  'finanzen.tabs.recurring': { de: 'Wiederkehrend', en: 'Recurring', fr: 'Récurrentes', it: 'Ricorrenti' },

  // Invoice form — exchange rate
  'finanzen.invoiceForm.exchangeRate': { de: 'Wechselkurs', en: 'Exchange rate', fr: 'Taux de change', it: 'Tasso di cambio' },
  'finanzen.invoiceForm.exchangeRateHint': {
    de: '1 {currency} = {rate} EUR · Gesamtbetrag ≈ {eur}',
    en: '1 {currency} = {rate} EUR · total ≈ {eur}',
    fr: '1 {currency} = {rate} EUR · total ≈ {eur}',
    it: '1 {currency} = {rate} EUR · totale ≈ {eur}',
  },

  // Open items (OP-Liste)
  'finanzen.openItems.emptyTitle': { de: 'Keine offenen Posten', en: 'No open items', fr: 'Aucun poste ouvert', it: 'Nessuna partita aperta' },
  'finanzen.openItems.emptyDescription': {
    de: 'Alle ausgestellten Rechnungen sind bezahlt. Offene und überfällige Rechnungen erscheinen hier.',
    en: 'All issued invoices are paid. Open and overdue invoices appear here.',
    fr: 'Toutes les factures émises sont payées. Les factures ouvertes et en retard apparaissent ici.',
    it: 'Tutte le fatture emesse sono pagate. Le fatture aperte e scadute appaiono qui.',
  },
  'finanzen.openItems.totalOpen': { de: 'Gesamt offen', en: 'Total open', fr: 'Total ouvert', it: 'Totale aperto' },
  'finanzen.openItems.totalOverdue': { de: 'Überfällig', en: 'Overdue', fr: 'En retard', it: 'Scadute' },
  'finanzen.openItems.avgOverdue': { de: 'Ø Verzug', en: 'Avg. overdue', fr: 'Retard moy.', it: 'Ritardo medio' },
  'finanzen.openItems.avgOverdueHint': { de: 'über alle überfälligen Posten', en: 'across all overdue items', fr: 'sur tous les postes en retard', it: 'su tutte le partite scadute' },
  'finanzen.openItems.itemCount': {
    de: '{count, plural, one {# Posten} other {# Posten}}',
    en: '{count, plural, one {# item} other {# items}}',
    fr: '{count, plural, one {# poste} other {# postes}}',
    it: '{count, plural, one {# partita} other {# partite}}',
  },
  'finanzen.openItems.days': {
    de: '{count, plural, one {# Tag} other {# Tage}}',
    en: '{count, plural, one {# day} other {# days}}',
    fr: '{count, plural, one {# jour} other {# jours}}',
    it: '{count, plural, one {# giorno} other {# giorni}}',
  },
  'finanzen.openItems.bucketAll': { de: 'Alle', en: 'All', fr: 'Tous', it: 'Tutte' },
  'finanzen.openItems.bucket.current': { de: 'Nicht fällig', en: 'Not due', fr: 'Non échu', it: 'Non scaduto' },
  'finanzen.openItems.bucket.d30': { de: '1–30 Tage', en: '1–30 days', fr: '1–30 jours', it: '1–30 giorni' },
  'finanzen.openItems.bucket.d60': { de: '31–60 Tage', en: '31–60 days', fr: '31–60 jours', it: '31–60 giorni' },
  'finanzen.openItems.bucket.d60plus': { de: '60+ Tage', en: '60+ days', fr: '60+ jours', it: '60+ giorni' },
  'finanzen.openItems.overdueCol': { de: 'Verzug', en: 'Overdue', fr: 'Retard', it: 'Ritardo' },
  'finanzen.openItems.agingCol': { de: 'Fälligkeit', en: 'Aging', fr: 'Ancienneté', it: 'Anzianità' },
  'finanzen.openItems.daysOverdue': {
    de: '{count, plural, one {# Tag überfällig} other {# Tage überfällig}}',
    en: '{count, plural, one {# day overdue} other {# days overdue}}',
    fr: '{count, plural, one {# jour de retard} other {# jours de retard}}',
    it: '{count, plural, one {# giorno di ritardo} other {# giorni di ritardo}}',
  },
  'finanzen.openItems.notDue': {
    de: '{count, plural, one {fällig in # Tag} other {fällig in # Tagen}}',
    en: '{count, plural, one {due in # day} other {due in # days}}',
    fr: '{count, plural, one {échéance dans # jour} other {échéance dans # jours}}',
    it: '{count, plural, one {scade tra # giorno} other {scade tra # giorni}}',
  },
  'finanzen.openItems.fxNote': {
    de: 'Fremdwährungsbeträge sind für die Summen zum hinterlegten Kurs in EUR umgerechnet.',
    en: 'Foreign-currency amounts are converted to EUR at the stored rate for totals.',
    fr: 'Les montants en devises sont convertis en EUR au taux enregistré pour les totaux.',
    it: 'Gli importi in valuta estera sono convertiti in EUR al tasso memorizzato per i totali.',
  },

  // Recurring invoices
  'finanzen.recurring.createTitle': { de: 'Wiederkehrende Rechnung', en: 'Recurring invoice', fr: 'Facture récurrente', it: 'Fattura ricorrente' },
  'finanzen.recurring.editTitle': { de: 'Wiederkehrende Rechnung bearbeiten', en: 'Edit recurring invoice', fr: 'Modifier la facture récurrente', it: 'Modifica fattura ricorrente' },
  'finanzen.recurring.title': { de: 'Bezeichnung', en: 'Title', fr: 'Intitulé', it: 'Titolo' },
  'finanzen.recurring.titlePlaceholder': { de: 'z. B. CRM-Lizenz Müller GmbH', en: 'e.g. CRM licence Müller GmbH', fr: 'ex. licence CRM Müller GmbH', it: 'es. licenza CRM Müller GmbH' },
  'finanzen.recurring.titleCustomerRequired': { de: 'Bezeichnung und Kunde sind erforderlich', en: 'Title and customer are required', fr: "L'intitulé et le client sont requis", it: 'Titolo e cliente sono obbligatori' },
  'finanzen.recurring.interval': { de: 'Intervall', en: 'Interval', fr: 'Intervalle', it: 'Intervallo' },
  'finanzen.recurring.intervals.weekly': { de: 'Wöchentlich', en: 'Weekly', fr: 'Hebdomadaire', it: 'Settimanale' },
  'finanzen.recurring.intervals.monthly': { de: 'Monatlich', en: 'Monthly', fr: 'Mensuelle', it: 'Mensile' },
  'finanzen.recurring.intervals.quarterly': { de: 'Quartalsweise', en: 'Quarterly', fr: 'Trimestrielle', it: 'Trimestrale' },
  'finanzen.recurring.intervals.yearly': { de: 'Jährlich', en: 'Yearly', fr: 'Annuelle', it: 'Annuale' },
  'finanzen.recurring.startDate': { de: 'Startdatum', en: 'Start date', fr: 'Date de début', it: 'Data inizio' },
  'finanzen.recurring.endDate': { de: 'Enddatum (optional)', en: 'End date (optional)', fr: 'Date de fin (facultatif)', it: 'Data fine (facoltativa)' },
  'finanzen.recurring.perInvoice': { de: 'Pro Rechnung', en: 'Per invoice', fr: 'Par facture', it: 'Per fattura' },
  'finanzen.recurring.created': { de: 'Wiederkehrende Rechnung angelegt', en: 'Recurring invoice created', fr: 'Facture récurrente créée', it: 'Fattura ricorrente creata' },
  'finanzen.recurring.updated': { de: 'Wiederkehrende Rechnung aktualisiert', en: 'Recurring invoice updated', fr: 'Facture récurrente mise à jour', it: 'Fattura ricorrente aggiornata' },
  'finanzen.recurring.activeSchedules': { de: 'Aktive Serien', en: 'Active schedules', fr: 'Séries actives', it: 'Pianificazioni attive' },
  'finanzen.recurring.mrr': { de: 'Monatl. Umsatz (geschätzt)', en: 'Monthly revenue (est.)', fr: 'Revenu mensuel (est.)', it: 'Ricavo mensile (stim.)' },
  'finanzen.recurring.scheduleCol': { de: 'Serie', en: 'Schedule', fr: 'Série', it: 'Serie' },
  'finanzen.recurring.intervalCol': { de: 'Intervall', en: 'Interval', fr: 'Intervalle', it: 'Intervallo' },
  'finanzen.recurring.nextRun': { de: 'Nächste', en: 'Next', fr: 'Prochaine', it: 'Prossima' },
  'finanzen.recurring.status.active': { de: 'Aktiv', en: 'Active', fr: 'Active', it: 'Attiva' },
  'finanzen.recurring.status.paused': { de: 'Pausiert', en: 'Paused', fr: 'En pause', it: 'In pausa' },
  'finanzen.recurring.status.ended': { de: 'Beendet', en: 'Ended', fr: 'Terminée', it: 'Terminata' },
  'finanzen.recurring.generatedCount': {
    de: '{count, plural, one {# Rechnung erzeugt} other {# Rechnungen erzeugt}}',
    en: '{count, plural, one {# invoice generated} other {# invoices generated}}',
    fr: '{count, plural, one {# facture générée} other {# factures générées}}',
    it: '{count, plural, one {# fattura generata} other {# fatture generate}}',
  },
  'finanzen.recurring.generateNow': { de: 'Jetzt erzeugen', en: 'Generate now', fr: 'Générer maintenant', it: 'Genera ora' },
  'finanzen.recurring.generated': {
    de: 'Rechnung aus „{title}" erzeugt',
    en: 'Invoice generated from “{title}”',
    fr: 'Facture générée depuis « {title} »',
    it: 'Fattura generata da «{title}»',
  },
  'finanzen.recurring.pause': { de: 'Pausieren', en: 'Pause', fr: 'Mettre en pause', it: 'Metti in pausa' },
  'finanzen.recurring.resume': { de: 'Fortsetzen', en: 'Resume', fr: 'Reprendre', it: 'Riprendi' },
  'finanzen.recurring.paused': { de: 'Serie pausiert', en: 'Schedule paused', fr: 'Série en pause', it: 'Pianificazione in pausa' },
  'finanzen.recurring.resumed': { de: 'Serie fortgesetzt', en: 'Schedule resumed', fr: 'Série reprise', it: 'Pianificazione ripresa' },
  'finanzen.recurring.emptyTitle': { de: 'Noch keine wiederkehrenden Rechnungen', en: 'No recurring invoices yet', fr: 'Pas encore de factures récurrentes', it: 'Ancora nessuna fattura ricorrente' },
  'finanzen.recurring.emptyDescription': {
    de: 'Lege eine Serie an, um Rechnungen automatisch in festen Intervallen zu erzeugen.',
    en: 'Create a schedule to automatically issue invoices at fixed intervals.',
    fr: 'Créez une série pour émettre automatiquement des factures à intervalles fixes.',
    it: 'Crea una pianificazione per emettere fatture automaticamente a intervalli fissi.',
  },
  'finanzen.recurring.generateHint': {
    de: '„Jetzt erzeugen" legt sofort eine Entwurfs-Rechnung an und verschiebt die nächste Fälligkeit.',
    en: '“Generate now” immediately creates a draft invoice and advances the next due date.',
    fr: '« Générer maintenant » crée immédiatement une facture brouillon et avance la prochaine échéance.',
    it: '«Genera ora» crea subito una fattura in bozza e avanza la prossima scadenza.',
  },
  'finanzen.recurring.deleteTitle': { de: 'Serie löschen', en: 'Delete schedule', fr: 'Supprimer la série', it: 'Elimina pianificazione' },
  'finanzen.recurring.deleteDescription': {
    de: '„{title}" wird dauerhaft entfernt. Bereits erzeugte Rechnungen bleiben erhalten.',
    en: '“{title}” will be permanently removed. Invoices already generated remain.',
    fr: '« {title} » sera supprimé définitivement. Les factures déjà générées sont conservées.',
    it: '«{title}» sarà rimosso definitivamente. Le fatture già generate rimangono.',
  },
  'finanzen.recurring.deleted': { de: 'Serie gelöscht', en: 'Schedule deleted', fr: 'Série supprimée', it: 'Pianificazione eliminata' },

  // Credit note / storno
  'finanzen.creditNote.createForInvoice': { de: 'Gutschrift erstellen', en: 'Create credit note', fr: 'Créer un avoir', it: 'Crea nota di credito' },
  'finanzen.creditNote.stornoBadge': { de: 'Storno', en: 'Reversal', fr: 'Annulation', it: 'Storno' },
  'finanzen.creditNote.stornoTitle': { de: 'Rechnung stornieren', en: 'Reverse invoice', fr: 'Annuler la facture', it: 'Storna fattura' },
  'finanzen.creditNote.stornoNotice': {
    de: 'Es wird eine Storno-Gutschrift über den vollen Betrag erstellt und Rechnung {number} auf „storniert" gesetzt.',
    en: 'A full reversal credit note is created and invoice {number} is set to “cancelled”.',
    fr: "Un avoir d'annulation pour le montant total est créé et la facture {number} passe à « annulée ».",
    it: "Viene creata una nota di credito di storno per l'intero importo e la fattura {number} viene impostata su «annullata».",
  },
  'finanzen.creditNote.stornoReasonDefault': {
    de: 'Storno der Rechnung {number}',
    en: 'Reversal of invoice {number}',
    fr: 'Annulation de la facture {number}',
    it: 'Storno della fattura {number}',
  },
  'finanzen.creditNote.stornoDone': {
    de: 'Rechnung {number} storniert',
    en: 'Invoice {number} reversed',
    fr: 'Facture {number} annulée',
    it: 'Fattura {number} stornata',
  },
  'finanzen.creditNote.stornoConfirm': { de: 'Stornieren', en: 'Reverse', fr: 'Annuler', it: 'Storna' },

  // Invoice detail
  'finanzen.invoiceDetail.storno': { de: 'Stornieren', en: 'Reverse', fr: 'Annuler', it: 'Storna' },
  'finanzen.invoiceDetail.linkedCreditNotes': { de: 'Verknüpfte Gutschriften', en: 'Linked credit notes', fr: 'Avoirs liés', it: 'Note di credito collegate' },
}

const anchor = 'finanzen.tabs.quotes'
const report = {}
for (const loc of ['de', 'en', 'fr', 'it']) {
  const file = resolve(dir, `${loc}.json`)
  const obj = JSON.parse(readFileSync(file, 'utf8'))
  let lines = readFileSync(file, 'utf8').split('\n')
  const nk = Object.keys(keys).filter((k) => !(k in obj)).sort()
  if (!nk.length) { report[loc] = 0; continue }
  const block = nk.map((k) => `  ${JSON.stringify(k)}: ${JSON.stringify(keys[k][loc])},`)
  const idx = lines.findIndex((l) => l.trimStart().startsWith(`"${anchor}":`))
  if (idx === -1) throw new Error(`anchor ${anchor} missing in ${loc}`)
  lines = [...lines.slice(0, idx + 1), ...block, ...lines.slice(idx + 1)]
  const out = lines.join('\n')
  JSON.parse(out)
  writeFileSync(file, out, 'utf8')
  report[loc] = block.length
}
console.log(JSON.stringify(report))

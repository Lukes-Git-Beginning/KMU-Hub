// Untracked helper (B-5): migrate all remaining berichte defaultValue keys into
// the 4 message JSONs with real EN/FR/IT translations. Only inserts missing
// keys (skips existing). Anchor: "berichte.chart.vorjahr".
import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'

const DIR = resolve('src/renderer/src/i18n/messages')

// key -> { de, en, fr, it }
const T = {
  'berichte.erstellen.errorBericht': {
    de: 'Bitte einen Bericht auswählen.', en: 'Please select a report.',
    fr: 'Veuillez sélectionner un rapport.', it: 'Seleziona un report.',
  },
  'berichte.erstellen.bericht': { de: 'Bericht', en: 'Report', fr: 'Rapport', it: 'Report' },
  'berichte.erstellen.laedt': { de: 'Lädt...', en: 'Loading...', fr: 'Chargement...', it: 'Caricamento...' },
  'berichte.erstellen.berichtSelect': {
    de: 'Bericht auswählen...', en: 'Select report...',
    fr: 'Sélectionner un rapport...', it: 'Seleziona report...',
  },
  'berichte.erstellen.zeitraumHint': {
    de: 'Leer lassen, um den Standard-Zeitraum des Berichts zu verwenden.',
    en: "Leave empty to use the report's default period.",
    fr: 'Laisser vide pour utiliser la période par défaut du rapport.',
    it: 'Lascia vuoto per usare il periodo predefinito del report.',
  },
  'berichte.chart.noData': {
    de: 'Noch keine Daten geladen.', en: 'No data loaded yet.',
    fr: 'Aucune donnée chargée.', it: 'Nessun dato caricato.',
  },
  'berichte.datev.laedt': {
    de: 'Bericht wird geladen...', en: 'Loading report...',
    fr: 'Chargement du rapport...', it: 'Caricamento report...',
  },
  'berichte.datev.noRows': {
    de: 'Keine Daten im aktuellen Zeitraum.', en: 'No data in the current period.',
    fr: 'Aucune donnée sur la période actuelle.', it: 'Nessun dato nel periodo corrente.',
  },
  'berichte.datev.unavailable': {
    de: 'DATEV-Berichte sind noch nicht konfiguriert. Aktiviere das Backend-Modul und starte das Gateway neu.',
    en: 'DATEV reports are not configured yet. Enable the backend module and restart the gateway.',
    fr: 'Les rapports DATEV ne sont pas encore configurés. Activez le module backend et redémarrez la passerelle.',
    it: 'I report DATEV non sono ancora configurati. Abilita il modulo backend e riavvia il gateway.',
  },
  'berichte.drilldown.verlauf': { de: 'Verlauf (Demo)', en: 'Trend (demo)', fr: 'Évolution (démo)', it: 'Andamento (demo)' },
  'berichte.drilldown.modul': { de: 'Modul', en: 'Module', fr: 'Module', it: 'Modulo' },
  'berichte.drilldown.zeitraum': { de: 'Zeitraum', en: 'Period', fr: 'Période', it: 'Periodo' },
  'berichte.drilldown.hinweis': {
    de: 'Demo-Werte — finale BI-Anbindung folgt.',
    en: 'Demo values — final BI integration to follow.',
    fr: 'Valeurs de démonstration — intégration BI finale à venir.',
    it: 'Valori demo — integrazione BI finale in arrivo.',
  },
  'berichte.geplant.geloescht': {
    de: 'Geplanter Bericht gelöscht.', en: 'Scheduled report deleted.',
    fr: 'Rapport planifié supprimé.', it: 'Report pianificato eliminato.',
  },
  'berichte.geplant.table.naechsterLauf': {
    de: 'Nächster Lauf', en: 'Next run', fr: 'Prochaine exécution', it: 'Prossima esecuzione',
  },
  'berichte.geplant.nieGelaufen': {
    de: 'Noch nicht gelaufen', en: 'Not run yet', fr: 'Jamais exécuté', it: 'Mai eseguito',
  },
  'berichte.geplant.pausiert': { de: 'Pausiert', en: 'Paused', fr: 'En pause', it: 'In pausa' },
  'berichte.dialog.errorName': {
    de: 'Bitte einen Namen angeben.', en: 'Please enter a name.',
    fr: 'Veuillez saisir un nom.', it: 'Inserisci un nome.',
  },
  'berichte.dialog.name': { de: 'Name', en: 'Name', fr: 'Nom', it: 'Nome' },
  'berichte.dialog.namePlaceholder': {
    de: 'Monatlicher Umsatzbericht', en: 'Monthly revenue report',
    fr: "Rapport de chiffre d'affaires mensuel", it: 'Report mensile dei ricavi',
  },
  'berichte.dialog.cronHint': {
    de: 'Cron-Expression, z. B. "0 8 * * MON" für jeden Montag um 8:00 Uhr.',
    en: 'Cron expression, e.g. "0 8 * * MON" for every Monday at 8:00.',
    fr: 'Expression cron, p. ex. "0 8 * * MON" pour chaque lundi à 8h00.',
    it: 'Espressione cron, ad es. "0 8 * * MON" per ogni lunedì alle 8:00.',
  },
  'berichte.dialog.alertSchwelle': {
    de: 'Alert-Schwellwert (optional)', en: 'Alert threshold (optional)',
    fr: "Seuil d'alerte (facultatif)", it: 'Soglia di avviso (facoltativa)',
  },
  'berichte.dialog.alertPlaceholder': { de: 'z. B. 100000', en: 'e.g. 100000', fr: 'p. ex. 100000', it: 'ad es. 100000' },
  'berichte.dialog.alertHint': {
    de: 'Benachrichtigung, sobald der Hauptwert des Berichts diese Schwelle überschreitet.',
    en: "Notification once the report's main value exceeds this threshold.",
    fr: 'Notification dès que la valeur principale du rapport dépasse ce seuil.',
    it: 'Notifica quando il valore principale del report supera questa soglia.',
  },
  'berichte.dialog.alertAktiv': { de: 'Alert aktiv', en: 'Alert active', fr: 'Alerte active', it: 'Avviso attivo' },
  'berichte.settings.personal.format': { de: 'Standard-Format', en: 'Default format', fr: 'Format par défaut', it: 'Formato predefinito' },
  'berichte.settings.personal.formatHint': {
    de: 'Wird beim Erstellen neuer Berichte vorausgewählt.',
    en: 'Pre-selected when creating new reports.',
    fr: 'Présélectionné lors de la création de nouveaux rapports.',
    it: 'Preselezionato durante la creazione di nuovi report.',
  },
  'berichte.settings.personal.period': { de: 'Standard-Zeitraum', en: 'Default period', fr: 'Période par défaut', it: 'Periodo predefinito' },
  'berichte.settings.personal.periodHint': {
    de: 'Vorbelegung für den Zeitraum-Filter neuer Berichte.',
    en: 'Pre-fills the period filter of new reports.',
    fr: 'Pré-remplit le filtre de période des nouveaux rapports.',
    it: 'Precompila il filtro periodo dei nuovi report.',
  },
  'berichte.settings.tenant.allowedFormats': {
    de: 'Erlaubte Export-Formate', en: 'Allowed export formats',
    fr: "Formats d'export autorisés", it: 'Formati di esportazione consentiti',
  },
  'berichte.settings.tenant.allowedFormatsHint': {
    de: 'Formate, die im gesamten Arbeitsbereich exportiert werden dürfen.',
    en: 'Formats that may be exported across the workspace.',
    fr: "Formats pouvant être exportés dans tout l'espace de travail.",
    it: "Formati esportabili nell'intero spazio di lavoro.",
  },
  'berichte.settings.tenant.domains': {
    de: 'Zulässige E-Mail-Domains', en: 'Permitted email domains',
    fr: 'Domaines e-mail autorisés', it: 'Domini e-mail consentiti',
  },
  'berichte.settings.tenant.domainsHint': {
    de: 'Geplante Berichte dürfen nur an diese Domains versendet werden.',
    en: 'Scheduled reports may only be sent to these domains.',
    fr: 'Les rapports planifiés ne peuvent être envoyés qu\'à ces domaines.',
    it: 'I report pianificati possono essere inviati solo a questi domini.',
  },
  'berichte.settings.tenant.domainsPlaceholder': { de: 'z. B. firma.de', en: 'e.g. company.com', fr: 'p. ex. entreprise.fr', it: 'ad es. azienda.it' },
  'berichte.settings.tenant.domainAdd': { de: 'Hinzufügen', en: 'Add', fr: 'Ajouter', it: 'Aggiungi' },
  'berichte.period.last30': { de: 'Letzte 30 Tage', en: 'Last 30 days', fr: '30 derniers jours', it: 'Ultimi 30 giorni' },
  'berichte.period.thisMonth': { de: 'Aktueller Monat', en: 'Current month', fr: 'Mois en cours', it: 'Mese corrente' },
  'berichte.period.thisQuarter': { de: 'Aktuelles Quartal', en: 'Current quarter', fr: 'Trimestre en cours', it: 'Trimestre corrente' },
  'berichte.period.thisYear': { de: 'Aktuelles Jahr', en: 'Current year', fr: 'Année en cours', it: 'Anno corrente' },
}

function insertAfter(text, anchorKey, line) {
  const re = new RegExp(`("${anchorKey.replace(/\./g, '\\.')}":[^\\n]*\\n)`)
  if (!re.test(text)) return null
  return text.replace(re, `$1${line}`)
}

for (const lang of ['de', 'en', 'fr', 'it']) {
  const file = resolve(DIR, `${lang}.json`)
  let text = readFileSync(file, 'utf8')
  let added = 0
  for (const [key, vals] of Object.entries(T)) {
    if (text.includes(`"${key}":`)) continue
    const line = `  "${key}": ${JSON.stringify(vals[lang])},\n`
    const next =
      insertAfter(text, 'berichte.chart.vorjahr', line) ||
      insertAfter(text, 'berichte.chart.umsatzverlauf', line)
    if (next) {
      text = next
      added++
    } else {
      console.error(`[${lang}] no anchor for ${key}`)
    }
  }
  JSON.parse(text)
  writeFileSync(file, text)
  console.log(`[${lang}] +${added} keys, valid JSON`)
}

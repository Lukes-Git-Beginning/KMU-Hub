import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'

const dir = resolve('src/renderer/src/i18n/messages')

const common = {
  'crm.nav.leads': { de: 'Leads', en: 'Leads', fr: 'Leads', it: 'Lead' },
}

const leads = {
  'leads.action.convert': { de: 'Qualifizieren / Umwandeln', en: 'Qualify / convert', fr: 'Qualifier / convertir', it: 'Qualifica / converti' },
  'leads.action.disqualify': { de: 'Verwerfen', en: 'Disqualify', fr: 'Disqualifier', it: 'Squalifica' },
  'leads.action.markContacted': { de: 'Als kontaktiert markieren', en: 'Mark as contacted', fr: 'Marquer comme contacté', it: 'Segna come contattato' },
  'leads.action.reopen': { de: 'Wieder öffnen', en: 'Reopen', fr: 'Rouvrir', it: 'Riapri' },
  'leads.action.setCold': { de: 'Als kalt markieren', en: 'Mark as cold', fr: 'Marquer comme froid', it: 'Segna come freddo' },
  'leads.action.setHot': { de: 'Als heiß markieren', en: 'Mark as hot', fr: 'Marquer comme chaud', it: 'Segna come caldo' },
  'leads.action.setWarm': { de: 'Als warm markieren', en: 'Mark as warm', fr: 'Marquer comme tiède', it: 'Segna come tiepido' },
  'leads.convert.confirm': { de: 'Umwandeln', en: 'Convert', fr: 'Convertir', it: 'Converti' },
  'leads.convert.createCompany': { de: 'Firma anlegen', en: 'Create company', fr: 'Créer une entreprise', it: 'Crea azienda' },
  'leads.convert.createContact': { de: 'Kontakt wird angelegt', en: 'Contact will be created', fr: 'Le contact sera créé', it: 'Il contatto verrà creato' },
  'leads.convert.createDeal': { de: 'Deal anlegen', en: 'Create deal', fr: 'Créer une opportunité', it: 'Crea deal' },
  'leads.convert.dealName': { de: 'Deal-Name', en: 'Deal name', fr: "Nom de l'opportunité", it: 'Nome deal' },
  'leads.convert.description': { de: '{name} qualifizieren und in echte Datensätze umwandeln.', en: 'Qualify {name} and convert into real records.', fr: 'Qualifier {name} et convertir en enregistrements réels.', it: 'Qualifica {name} e converti in record reali.' },
  'leads.convert.error': { de: 'Umwandlung fehlgeschlagen', en: 'Conversion failed', fr: 'Échec de la conversion', it: 'Conversione non riuscita' },
  'leads.convert.noCompany': { de: 'Keine Firma am Lead', en: 'No company on lead', fr: 'Aucune entreprise sur le lead', it: 'Nessuna azienda sul lead' },
  'leads.convert.success': { de: '{name} umgewandelt', en: '{name} converted', fr: '{name} converti', it: '{name} convertito' },
  'leads.convert.title': { de: 'Lead umwandeln', en: 'Convert lead', fr: 'Convertir le lead', it: 'Converti lead' },
  'leads.created': { de: 'Lead angelegt', en: 'Lead created', fr: 'Lead créé', it: 'Lead creato' },
  'leads.empty.description': { de: 'Noch keine Leads. Lege den ersten an oder importiere eine Liste.', en: 'No leads yet. Create the first one or import a list.', fr: 'Aucun lead. Créez le premier ou importez une liste.', it: 'Ancora nessun lead. Crea il primo o importa un elenco.' },
  'leads.empty.filtered': { de: 'Keine Leads in dieser Ansicht.', en: 'No leads in this view.', fr: 'Aucun lead dans cette vue.', it: 'Nessun lead in questa vista.' },
  'leads.empty.title': { de: 'Lead-Inbox leer', en: 'Lead inbox empty', fr: 'Boîte de réception des leads vide', it: 'Inbox lead vuota' },
  'leads.form.company': { de: 'Firma', en: 'Company', fr: 'Entreprise', it: 'Azienda' },
  'leads.form.create': { de: 'Lead anlegen', en: 'Create lead', fr: 'Créer le lead', it: 'Crea lead' },
  'leads.form.description': { de: 'Erfasse einen neuen Lead manuell.', en: 'Capture a new lead manually.', fr: 'Saisir un nouveau lead manuellement.', it: 'Inserisci un nuovo lead manualmente.' },
  'leads.form.email': { de: 'E-Mail', en: 'Email', fr: 'E-mail', it: 'E-mail' },
  'leads.form.firstName': { de: 'Vorname', en: 'First name', fr: 'Prénom', it: 'Nome' },
  'leads.form.lastName': { de: 'Nachname', en: 'Last name', fr: 'Nom', it: 'Cognome' },
  'leads.form.notes': { de: 'Notizen', en: 'Notes', fr: 'Notes', it: 'Note' },
  'leads.form.phone': { de: 'Telefon', en: 'Phone', fr: 'Téléphone', it: 'Telefono' },
  'leads.form.scorePreview': { de: 'Auto-Score (Vorschau)', en: 'Auto score (preview)', fr: 'Score auto (aperçu)', it: 'Punteggio auto (anteprima)' },
  'leads.form.source': { de: 'Quelle', en: 'Source', fr: 'Source', it: 'Origine' },
  'leads.form.title': { de: 'Neuer Lead', en: 'New lead', fr: 'Nouveau lead', it: 'Nuovo lead' },
  'leads.import.button': { de: 'CSV-Import', en: 'CSV import', fr: 'Import CSV', it: 'Importa CSV' },
  'leads.import.confirm': { de: 'Importieren', en: 'Import', fr: 'Importer', it: 'Importa' },
  'leads.import.detected': { de: '{count} Leads erkannt', en: '{count} leads detected', fr: '{count} leads détectés', it: '{count} lead rilevati' },
  'leads.import.help': { de: 'Eine Zeile pro Lead: Vorname, Nachname, Firma, E-Mail, Telefon', en: 'One lead per line: first name, last name, company, email, phone', fr: 'Un lead par ligne : prénom, nom, entreprise, e-mail, téléphone', it: 'Un lead per riga: nome, cognome, azienda, e-mail, telefono' },
  'leads.import.placeholder': { de: 'Max, Mustermann, Muster GmbH, max@muster.de, +49 ...', en: 'Max, Mustermann, Muster Ltd, max@muster.com, +49 ...', fr: 'Max, Mustermann, Muster SARL, max@muster.fr, +33 ...', it: 'Max, Mustermann, Muster Srl, max@muster.it, +39 ...' },
  'leads.import.title': { de: 'Leads importieren (CSV)', en: 'Import leads (CSV)', fr: 'Importer des leads (CSV)', it: 'Importa lead (CSV)' },
  'leads.imported': { de: '{count} Leads importiert', en: '{count} leads imported', fr: '{count} leads importés', it: '{count} lead importati' },
  'leads.new': { de: 'Neuer Lead', en: 'New lead', fr: 'Nouveau lead', it: 'Nuovo lead' },
  'leads.noCompany': { de: 'Keine Firma', en: 'No company', fr: 'Aucune entreprise', it: 'Nessuna azienda' },
  'leads.score': { de: 'Score', en: 'Score', fr: 'Score', it: 'Punteggio' },
  'leads.source.csv': { de: 'CSV-Import', en: 'CSV import', fr: 'Import CSV', it: 'Importa CSV' },
  'leads.source.dialer': { de: 'Dialer', en: 'Dialer', fr: 'Numéroteur', it: 'Combinatore' },
  'leads.source.manual': { de: 'Manuell', en: 'Manual', fr: 'Manuel', it: 'Manuale' },
  'leads.status.all': { de: 'Alle', en: 'All', fr: 'Tous', it: 'Tutti' },
  'leads.status.contacted': { de: 'Kontaktiert', en: 'Contacted', fr: 'Contacté', it: 'Contattato' },
  'leads.status.disqualified': { de: 'Verworfen', en: 'Disqualified', fr: 'Disqualifié', it: 'Squalificato' },
  'leads.status.new': { de: 'Neu', en: 'New', fr: 'Nouveau', it: 'Nuovo' },
  'leads.status.qualified': { de: 'Qualifiziert', en: 'Qualified', fr: 'Qualifié', it: 'Qualificato' },
  'leads.temp.cold': { de: 'Kalt', en: 'Cold', fr: 'Froid', it: 'Freddo' },
  'leads.temp.hot': { de: 'Heiß', en: 'Hot', fr: 'Chaud', it: 'Caldo' },
  'leads.temp.warm': { de: 'Warm', en: 'Warm', fr: 'Tiède', it: 'Tiepido' },
  'leads.toast.statusChanged': { de: 'Status aktualisiert', en: 'Status updated', fr: 'Statut mis à jour', it: 'Stato aggiornato' },
  'leads.toast.tempChanged': { de: 'Bewertung aktualisiert', en: 'Rating updated', fr: 'Évaluation mise à jour', it: 'Valutazione aggiornata' },
}

const locales = ['de', 'en', 'fr', 'it']
const report = {}

// Insert a block of "key": "value", lines right after the line whose key === anchorKey.
function insertAfter(lines, anchorKey, blockLines) {
  const idx = lines.findIndex((l) => l.trimStart().startsWith(`"${anchorKey}":`))
  if (idx === -1) throw new Error(`anchor not found: ${anchorKey}`)
  return [...lines.slice(0, idx + 1), ...blockLines, ...lines.slice(idx + 1)]
}

for (const loc of locales) {
  const file = resolve(dir, `${loc}.json`)
  const obj = JSON.parse(readFileSync(file, 'utf8'))
  let lines = readFileSync(file, 'utf8').split('\n')

  // 1) leads.* block (sorted among themselves) after layout.topnav.showModules
  const leadKeys = Object.keys(leads).filter((k) => !(k in obj)).sort()
  const leadBlock = leadKeys.map((k) => `  ${JSON.stringify(k)}: ${JSON.stringify(leads[k][loc])},`)
  if (leadBlock.length) lines = insertAfter(lines, 'layout.topnav.showModules', leadBlock)

  // 2) crm.nav.leads after crm.nav.deals
  let navAdded = 0
  if (!('crm.nav.leads' in obj)) {
    lines = insertAfter(lines, 'crm.nav.deals', [`  "crm.nav.leads": ${JSON.stringify(common['crm.nav.leads'][loc])},`])
    navAdded = 1
  }

  const out = lines.join('\n')
  JSON.parse(out) // validate
  writeFileSync(file, out, 'utf8')
  report[loc] = { leadsAdded: leadBlock.length, navAdded }
}
console.log(JSON.stringify(report, null, 2))

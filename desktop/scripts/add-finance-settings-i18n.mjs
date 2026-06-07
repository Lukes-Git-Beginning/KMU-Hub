import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'

const dir = resolve('src/renderer/src/i18n/messages')
const anchor = 'kontakte.detail.tags'

// Rename the module-settings entry label to "Buchhaltung".
const LABEL = { de: 'Buchhaltung', en: 'Accounting', fr: 'Comptabilité', it: 'Contabilità' }

const K = {
  'finanzen.settings.title': { de: 'Buchhaltung', en: 'Accounting', fr: 'Comptabilité', it: 'Contabilità' },
  'finanzen.settings.subtitle': { de: 'Einstellungen für Rechnungen, Stammdaten und Integrationen', en: 'Settings for invoicing, master data and integrations', fr: 'Paramètres de facturation, données et intégrations', it: 'Impostazioni per fatturazione, dati e integrazioni' },
  'finanzen.settings.personal.title': { de: 'Ansicht', en: 'View', fr: 'Affichage', it: 'Visualizzazione' },
  'finanzen.settings.personal.desc': { de: 'Womit die Buchhaltung startet', en: 'Where accounting opens', fr: 'Vue d’ouverture de la comptabilité', it: 'Vista di apertura della contabilità' },
  'finanzen.settings.stammdaten.title': { de: 'Firmen-Stammdaten', en: 'Company master data', fr: 'Données de l’entreprise', it: 'Dati aziendali' },
  'finanzen.settings.stammdaten.desc': { de: 'Daten, die auf Rechnungen erscheinen', en: 'Data shown on invoices', fr: 'Données figurant sur les factures', it: 'Dati che appaiono sulle fatture' },
  'finanzen.settings.invoicing.title': { de: 'Rechnungen, Steuer & Mahnwesen', en: 'Invoicing, tax & dunning', fr: 'Facturation, TVA et relances', it: 'Fatturazione, imposte e solleciti' },
  'finanzen.settings.invoicing.desc': { de: 'USt-Sätze, Rechnungsnummern, Zahlungsziele, DATEV-Nummern, Mahnstufen', en: 'VAT rates, invoice numbers, payment terms, DATEV numbers, dunning levels', fr: 'Taux de TVA, numéros de facture, délais de paiement, numéros DATEV, niveaux de relance', it: 'Aliquote IVA, numeri fattura, termini di pagamento, numeri DATEV, livelli di sollecito' },
  'finanzen.settings.integrations.title': { de: 'Integrationen', en: 'Integrations', fr: 'Intégrations', it: 'Integrazioni' },
  'finanzen.settings.integrations.desc': { de: 'Verknüpfung zu DATEV, Bexio & Co.', en: 'Connect DATEV, Bexio & co.', fr: 'Connexion à DATEV, Bexio, etc.', it: 'Collegamento a DATEV, Bexio e co.' },

  'finanzen.prefs.startTab.label': { de: 'Start-Ansicht', en: 'Start view', fr: 'Vue de démarrage', it: 'Vista iniziale' },
  'finanzen.prefs.startTab.hint': { de: 'Welcher Tab beim Öffnen der Buchhaltung erscheint', en: 'Which tab opens when you enter accounting', fr: 'Onglet affiché à l’ouverture de la comptabilité', it: 'Scheda mostrata all’apertura della contabilità' },
  'finanzen.prefs.startTab.last': { de: 'Zuletzt verwendet', en: 'Last used', fr: 'Dernier utilisé', it: 'Ultimo usato' },
  'finanzen.prefs.startTab.dashboard': { de: 'Dashboard', en: 'Dashboard', fr: 'Tableau de bord', it: 'Dashboard' },
  'finanzen.prefs.startTab.invoices': { de: 'Rechnungen', en: 'Invoices', fr: 'Factures', it: 'Fatture' },
  'finanzen.prefs.startTab.quotes': { de: 'Angebote', en: 'Quotes', fr: 'Devis', it: 'Preventivi' },
  'finanzen.prefs.startTab.expenses': { de: 'Ausgaben', en: 'Expenses', fr: 'Dépenses', it: 'Spese' },
  'finanzen.prefs.startTab.dunning': { de: 'Mahnwesen', en: 'Dunning', fr: 'Relances', it: 'Solleciti' },
}

const report = {}
for (const loc of ['de', 'en', 'fr', 'it']) {
  const file = resolve(dir, `${loc}.json`)
  let raw = readFileSync(file, 'utf8')

  // 1) update the finance entry label in place
  raw = raw.replace(
    /("moduleSettings\.entries\.finance":\s*)"[^"]*"/,
    `$1${JSON.stringify(LABEL[loc])}`,
  )

  // 2) insert new keys after the anchor
  const obj = JSON.parse(raw)
  let lines = raw.split('\n')
  const nk = Object.keys(K).filter((k) => !(k in obj)).sort()
  if (nk.length) {
    const block = nk.map((k) => `  ${JSON.stringify(k)}: ${JSON.stringify(K[k][loc])},`)
    const idx = lines.findIndex((l) => l.trimStart().startsWith(`"${anchor}":`))
    if (idx === -1) throw new Error(`anchor ${anchor} missing in ${loc}`)
    lines = [...lines.slice(0, idx + 1), ...block, ...lines.slice(idx + 1)]
  }
  const out = lines.join('\n')
  JSON.parse(out)
  writeFileSync(file, out, 'utf8')
  report[loc] = nk.length
}
console.log(JSON.stringify(report))

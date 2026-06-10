// Additive i18n for the advisory-protocol PDF/print view (kontakte P8).
import { readFileSync, writeFileSync } from 'node:fs'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const dir = resolve(dirname(fileURLToPath(import.meta.url)), '../src/renderer/src/i18n/messages')

const keys = {
  'advisory.print.toolbarTitle': { de: 'Geeignetheitserklärung — Vorschau', en: 'Suitability statement — preview', fr: "Déclaration d'adéquation — aperçu", it: 'Dichiarazione di adeguatezza — anteprima' },
  'advisory.print.action': { de: 'Als PDF / Drucken', en: 'Export PDF / Print', fr: 'PDF / Imprimer', it: 'PDF / Stampa' },
  'advisory.print.docTitle': { de: 'Geeignetheitserklärung', en: 'Suitability statement', fr: "Déclaration d'adéquation", it: 'Dichiarazione di adeguatezza' },
  'advisory.print.docSubtitle': { de: 'Dokumentation der Anlageberatung (§ 64 WpHG / FinVermV)', en: 'Investment advice documentation (§ 64 WpHG / FinVermV)', fr: "Documentation du conseil en investissement (§ 64 WpHG / FinVermV)", it: 'Documentazione della consulenza (§ 64 WpHG / FinVermV)' },
  'advisory.print.client': { de: 'Kunde', en: 'Client', fr: 'Client', it: 'Cliente' },
  'advisory.print.sriValue': { de: 'SRI {n} von 7', en: 'SRI {n} of 7', fr: 'SRI {n} sur 7', it: 'SRI {n} di 7' },
  'advisory.print.signatureAdvisor': { de: 'Unterschrift Berater', en: 'Advisor signature', fr: 'Signature du conseiller', it: 'Firma del consulente' },
  'advisory.print.signatureClient': { de: 'Unterschrift Kunde', en: 'Client signature', fr: 'Signature du client', it: 'Firma del cliente' },
  'advisory.print.legalNote': { de: 'Diese Geeignetheitserklärung wurde dem Kunden vor Vertragsschluss auf einem dauerhaften Datenträger zur Verfügung gestellt.', en: 'This suitability statement was provided to the client on a durable medium before conclusion of the contract.', fr: "Cette déclaration d'adéquation a été remise au client sur un support durable avant la conclusion du contrat.", it: 'La presente dichiarazione di adeguatezza è stata fornita al cliente su un supporto durevole prima della conclusione del contratto.' },
  'advisory.field.time': { de: 'Uhrzeit', en: 'Time', fr: 'Heure', it: 'Orario' },
  'advisory.field.maxLossCapacity': { de: 'Max. Verlusttragfähigkeit', en: 'Max. loss capacity', fr: 'Capacité de perte max.', it: 'Capacità di perdita max.' },
  'advisory.field.esg': { de: 'ESG-Präferenz', en: 'ESG preference', fr: 'Préférence ESG', it: 'Preferenza ESG' },
  'advisory.field.esgDetails': { de: 'ESG-Details', en: 'ESG details', fr: 'Détails ESG', it: 'Dettagli ESG' },
  'advisory.product.name': { de: 'Produkt', en: 'Product', fr: 'Produit', it: 'Prodotto' },
  'advisory.product.costs': { de: 'Kosten', en: 'Costs', fr: 'Coûts', it: 'Costi' },
  'advisory.product.recommended': { de: 'Empfohlen', en: 'Recommended', fr: 'Recommandé', it: 'Consigliato' },
}

const report = {}
for (const loc of ['de', 'en', 'fr', 'it']) {
  const file = resolve(dir, `${loc}.json`)
  const obj = JSON.parse(readFileSync(file, 'utf8'))
  const nk = Object.keys(keys).filter((k) => !(k in obj))
  if (!nk.length) { report[loc] = 0; continue }
  let lines = readFileSync(file, 'utf8').split('\n')
  const idx = lines.findIndex((l) => l.trimStart().startsWith('"advisory.'))
  const anchorIdx = idx !== -1 ? idx : lines.findIndex((l) => l.trim().startsWith('"'))
  const block = nk.map((k) => `  ${JSON.stringify(k)}: ${JSON.stringify(keys[k][loc])},`)
  lines = [...lines.slice(0, anchorIdx + 1), ...block, ...lines.slice(anchorIdx + 1)]
  const out = lines.join('\n')
  JSON.parse(out)
  writeFileSync(file, out, 'utf8')
  report[loc] = block.length
}
console.log(JSON.stringify(report))

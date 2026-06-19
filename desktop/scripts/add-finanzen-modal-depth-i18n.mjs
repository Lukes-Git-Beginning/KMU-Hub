import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
const dir = resolve('src/renderer/src/i18n/messages')
const groups = [
  { anchor: 'shared.detailPanel.close', keys: {
    'shared.detailPanel.back': { de:`Zurück`, en:`Back`, fr:`Retour`, it:`Indietro` },
  }},
  { anchor: 'finanzen.customer', keys: {
    'finanzen.customerAccount.title': { de:`Kundenkonto`, en:`Customer account`, fr:`Compte client`, it:`Conto cliente` },
    'finanzen.customerAccount.ustId': { de:`USt-IdNr.`, en:`VAT ID`, fr:`N° TVA`, it:`P. IVA` },
    'finanzen.customerAccount.openInCrm': { de:`Im CRM öffnen`, en:`Open in CRM`, fr:`Ouvrir dans le CRM`, it:`Apri nel CRM` },
    'finanzen.customerAccount.noCrmMatch': { de:`Kein zugeordneter CRM-Kontakt gefunden`, en:`No matching CRM contact found`, fr:`Aucun contact CRM correspondant`, it:`Nessun contatto CRM corrispondente` },
    'finanzen.customerAccount.revenue': { de:`Umsatz`, en:`Revenue`, fr:`Chiffre d'affaires`, it:`Fatturato` },
    'finanzen.customerAccount.open': { de:`Offen`, en:`Open`, fr:`En attente`, it:`Aperto` },
    'finanzen.customerAccount.documents': { de:`Belege`, en:`Documents`, fr:`Documents`, it:`Documenti` },
    'finanzen.customerAccount.more': { de:`{count, plural, one {+# weiterer Beleg} other {+# weitere Belege}}`, en:`{count, plural, one {+# more document} other {+# more documents}}`, fr:`{count, plural, one {+# autre document} other {+# autres documents}}`, it:`{count, plural, one {+# altro documento} other {+# altri documenti}}` },
    'finanzen.customerAccount.onlyThis': { de:`Keine weiteren Belege dieses Kunden`, en:`No other documents for this customer`, fr:`Aucun autre document pour ce client`, it:`Nessun altro documento per questo cliente` },
    'finanzen.customerAccount.typeInvoice': { de:`Rechnung`, en:`Invoice`, fr:`Facture`, it:`Fattura` },
    'finanzen.customerAccount.typeQuote': { de:`Angebot`, en:`Quote`, fr:`Devis`, it:`Preventivo` },
    'finanzen.customerAccount.typeCreditNote': { de:`Gutschrift`, en:`Credit note`, fr:`Avoir`, it:`Nota di credito` },
  }},
  { anchor: 'finanzen.invoiceDetail.auditCreated', keys: {
    'finanzen.invoiceDetail.fromQuote': { de:`Aus Angebot`, en:`From quote`, fr:`Issue du devis`, it:`Da preventivo` },
    'finanzen.invoiceDetail.fromRecurring': { de:`Aus Abo erzeugt`, en:`Generated from recurring`, fr:`Issue d'un abonnement`, it:`Generata da ricorrenza` },
    'finanzen.invoiceDetail.dunningHistory': { de:`Mahnverlauf`, en:`Dunning history`, fr:`Historique des relances`, it:`Storico solleciti` },
    'finanzen.invoiceDetail.dunningLevel': { de:`{level}. Mahnung`, en:`Reminder {level}`, fr:`Relance {level}`, it:`Sollecito {level}` },
  }},
  { anchor: 'finanzen.recurringDetail.title', keys: {
    'finanzen.recurringDetail.upcomingRuns': { de:`Nächste Läufe`, en:`Upcoming runs`, fr:`Prochaines échéances`, it:`Prossime esecuzioni` },
    'finanzen.recurringDetail.generatedInvoices': { de:`Erzeugte Rechnungen`, en:`Generated invoices`, fr:`Factures générées`, it:`Fatture generate` },
  }},
  { anchor: 'buchhaltung.expenseDetail.title', keys: {
    'buchhaltung.expenseDetail.supplierHistory': { de:`Weitere Ausgaben: {supplier}`, en:`More expenses: {supplier}`, fr:`Autres dépenses : {supplier}`, it:`Altre spese: {supplier}` },
    'buchhaltung.expenseDetail.supplierTotal': { de:`{count, plural, one {# Ausgabe gesamt} other {# Ausgaben gesamt}}`, en:`{count, plural, one {# expense total} other {# expenses total}}`, fr:`{count, plural, one {# dépense au total} other {# dépenses au total}}`, it:`{count, plural, one {# spesa in totale} other {# spese in totale}}` },
    'buchhaltung.expenseDetail.supplierOnly': { de:`Keine weiteren Ausgaben dieses Lieferanten`, en:`No other expenses from this supplier`, fr:`Aucune autre dépense de ce fournisseur`, it:`Nessuna altra spesa di questo fornitore` },
  }},
  { anchor: 'buchhaltung.transactionDetail.reference', keys: {
    'buchhaltung.transactionDetail.linkedInvoice': { de:`Verknüpfte Rechnung`, en:`Linked invoice`, fr:`Facture liée`, it:`Fattura collegata` },
    'buchhaltung.transactionDetail.relatedInCategory': { de:`Weitere in: {category}`, en:`More in: {category}`, fr:`Autres dans : {category}`, it:`Altre in: {category}` },
  }},
]
const report={}
for (const loc of ['de','en','fr','it']) {
  const file=resolve(dir,`${loc}.json`); const obj=JSON.parse(readFileSync(file,'utf8'))
  let lines=readFileSync(file,'utf8').split('\n'); let added=0
  for (const g of groups) {
    const nk=Object.keys(g.keys).filter(k=>!(k in obj)).sort()
    if(!nk.length) continue
    const block=nk.map(k=>`  ${JSON.stringify(k)}: ${JSON.stringify(g.keys[k][loc])},`)
    const idx=lines.findIndex(l=>l.trimStart().startsWith(`"${g.anchor}":`))
    if(idx===-1) throw new Error(`anchor ${g.anchor} missing ${loc}`)
    lines=[...lines.slice(0,idx+1),...block,...lines.slice(idx+1)]; added+=block.length
  }
  const out=lines.join('\n'); JSON.parse(out); writeFileSync(file,out,'utf8'); report[loc]=added
}
console.log(JSON.stringify(report))

import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
const dir = resolve('src/renderer/src/i18n/messages')
const groups = [
  { anchor: 'buchhaltung.editExpense', keys: {
    'buchhaltung.expenseDetail.title': { de:`Ausgabe`, en:`Expense`, fr:`Dépense`, it:`Spesa` },
    'buchhaltung.expenseDetail.noReceipt': { de:`Kein Beleg angehängt`, en:`No receipt attached`, fr:`Aucun justificatif joint`, it:`Nessun giustificativo allegato` },
    'buchhaltung.transactionDetail.title': { de:`Transaktion`, en:`Transaction`, fr:`Transaction`, it:`Transazione` },
    'buchhaltung.transactionDetail.reference': { de:`Referenz`, en:`Reference`, fr:`Référence`, it:`Riferimento` },
    'buchhaltung.txStatus.completed': { de:`Abgeschlossen`, en:`Completed`, fr:`Terminé`, it:`Completato` },
    'buchhaltung.txStatus.pending': { de:`Offen`, en:`Pending`, fr:`En attente`, it:`In sospeso` },
  }},
  { anchor: 'finanzen.recurring.nextRun', keys: {
    'finanzen.recurring.generatedLabel': { de:`Erzeugte Rechnungen`, en:`Generated invoices`, fr:`Factures générées`, it:`Fatture generate` },
    'finanzen.recurringDetail.title': { de:`Wiederkehrende Rechnung`, en:`Recurring invoice`, fr:`Facture récurrente`, it:`Fattura ricorrente` },
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

import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
const dir = resolve('src/renderer/src/i18n/messages')
const groups = [
  { anchor: 'finanzen.banking.connect', keys: {
    'finanzen.banking.connected': { de:`Konto verbunden`, en:`Account connected`, fr:`Compte connecté`, it:`Conto collegato` },
    'finanzen.banking.transactionDetail': { de:`Transaktionsdetails`, en:`Transaction details`, fr:`Détails de la transaction`, it:`Dettagli transazione` },
    'finanzen.banking.type': { de:`Art`, en:`Type`, fr:`Type`, it:`Tipo` },
    'finanzen.banking.credit': { de:`Eingang`, en:`Incoming`, fr:`Entrée`, it:`Entrata` },
    'finanzen.banking.debit': { de:`Ausgang`, en:`Outgoing`, fr:`Sortie`, it:`Uscita` },
    'finanzen.banking.assignedInvoice': { de:`Zugeordnete Rechnung`, en:`Assigned invoice`, fr:`Facture associée`, it:`Fattura associata` },
    'finanzen.banking.assignInvoice': { de:`Rechnung zuordnen`, en:`Assign invoice`, fr:`Associer une facture`, it:`Associa fattura` },
    'finanzen.banking.noOpenInvoices': { de:`Keine offenen Rechnungen vorhanden`, en:`No open invoices available`, fr:`Aucune facture ouverte disponible`, it:`Nessuna fattura aperta disponibile` },
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

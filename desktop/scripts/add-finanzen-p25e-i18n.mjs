import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
const dir = resolve('src/renderer/src/i18n/messages')
const groups = [
  { anchor: 'finanzen.hours.hourlyRate', keys: {
    'finanzen.hours.customerPlaceholder': { de:`Kundenname für die Rechnung`, en:`Customer name for the invoice`, fr:`Nom du client pour la facture`, it:`Nome cliente per la fattura` },
    'finanzen.hours.invoiceNote': { de:`Aus erfassten Stunden erzeugt`, en:`Generated from tracked hours`, fr:`Généré à partir des heures saisies`, it:`Generato dalle ore registrate` },
    'finanzen.hours.noUnbilled': { de:`Keine offenen Stunden zum Abrechnen`, en:`No unbilled hours to invoice`, fr:`Aucune heure à facturer`, it:`Nessuna ora da fatturare` },
  }},
  { anchor: 'finanzen.invoiceDetail.auditCreated', keys: {
    'finanzen.invoiceDetail.auditTitle': { de:`GoBD-Änderungsprotokoll`, en:`GoBD audit log`, fr:`Journal d'audit GoBD`, it:`Registro modifiche GoBD` },
    'finanzen.invoiceDetail.auditCreatedDetail': { de:`Nummer: {number}`, en:`Number: {number}`, fr:`Numéro : {number}`, it:`Numero: {number}` },
    'finanzen.invoiceDetail.auditSentDetail': { de:`An: {email}`, en:`To: {email}`, fr:`À : {email}`, it:`A: {email}` },
    'finanzen.invoiceDetail.auditPaymentDetail': { de:`{amount} per {method}`, en:`{amount} via {method}`, fr:`{amount} via {method}`, it:`{amount} tramite {method}` },
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

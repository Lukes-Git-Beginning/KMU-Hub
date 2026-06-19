import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
const dir = resolve('src/renderer/src/i18n/messages')
const groups = [
  { anchor: 'finanzen.dunning.escalate', keys: {
    'finanzen.dunningDetail.title': { de:`Mahndetails`, en:`Dunning details`, fr:`Détails de la relance`, it:`Dettagli sollecito` },
    'finanzen.dunningDetail.escalationChain': { de:`Eskalationsstufe`, en:`Escalation stage`, fr:`Niveau d'escalade`, it:`Livello di escalation` },
    'finanzen.dunningDetail.created': { de:`Erstellt`, en:`Created`, fr:`Créé`, it:`Creato` },
    'finanzen.dunningDetail.invoiceAmount': { de:`Rechnungsbetrag`, en:`Invoice amount`, fr:`Montant de la facture`, it:`Importo fattura` },
    'finanzen.dunningDetail.totalDue': { de:`Gesamtforderung`, en:`Total due`, fr:`Total dû`, it:`Totale dovuto` },
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

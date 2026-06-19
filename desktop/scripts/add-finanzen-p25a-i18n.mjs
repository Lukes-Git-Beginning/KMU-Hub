import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
const dir = resolve('src/renderer/src/i18n/messages')
const groups = [
  { anchor: 'finanzen.invoiceDetail.title', keys: {
    'finanzen.quoteDetail.title': { de:`Angebot`, en:`Quote`, fr:`Devis`, it:`Preventivo` },
    'finanzen.quoteDetail.quoteDate': { de:`Angebotsdatum`, en:`Quote date`, fr:`Date du devis`, it:`Data preventivo` },
    'finanzen.quoteDetail.validUntil': { de:`Gültig bis`, en:`Valid until`, fr:`Valable jusqu'au`, it:`Valido fino al` },
    'finanzen.quoteDetail.convertedTo': { de:`In Rechnung {number} umgewandelt`, en:`Converted to invoice {number}`, fr:`Converti en facture {number}`, it:`Convertito in fattura {number}` },
    'finanzen.creditNoteDetail.title': { de:`Gutschrift`, en:`Credit note`, fr:`Avoir`, it:`Nota di credito` },
    'finanzen.creditNoteDetail.originalInvoice': { de:`Originalrechnung`, en:`Original invoice`, fr:`Facture d'origine`, it:`Fattura originale` },
    'finanzen.creditNoteDetail.date': { de:`Datum`, en:`Date`, fr:`Date`, it:`Data` },
    'finanzen.creditNoteDetail.reason': { de:`Grund`, en:`Reason`, fr:`Motif`, it:`Motivo` },
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

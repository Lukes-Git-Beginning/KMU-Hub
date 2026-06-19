import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
const dir = resolve('src/renderer/src/i18n/messages')
const groups = [
  { anchor: 'kalender.booking.confirmed', keys: {
    'kalender.bookingDetail.title': { de:`Termindetails`, en:`Appointment details`, fr:`Détails du rendez-vous`, it:`Dettagli appuntamento` },
    'kalender.bookingDetail.statusUpdated': { de:`Status auf „{status}" gesetzt`, en:`Status set to "{status}"`, fr:`Statut défini sur « {status} »`, it:`Stato impostato su "{status}"` },
    'kalender.bookingDetail.schedule': { de:`Termin`, en:`Schedule`, fr:`Rendez-vous`, it:`Appuntamento` },
    'kalender.bookingDetail.staff': { de:`Personal`, en:`Staff`, fr:`Personnel`, it:`Personale` },
    'kalender.bookingDetail.client': { de:`Kunde`, en:`Client`, fr:`Client`, it:`Cliente` },
    'kalender.bookingDetail.serviceLabel': { de:`Leistung`, en:`Service`, fr:`Prestation`, it:`Servizio` },
    'kalender.bookingDetail.price': { de:`Preis`, en:`Price`, fr:`Prix`, it:`Prezzo` },
    'kalender.bookingDetail.notes': { de:`Notizen`, en:`Notes`, fr:`Notes`, it:`Note` },
    'kalender.bookingDetail.noContact': { de:`Keine Kontaktdaten hinterlegt`, en:`No contact details on file`, fr:`Aucune coordonnée enregistrée`, it:`Nessun recapito registrato` },
    'kalender.bookingDetail.customerHistory': { de:`Weitere Termine`, en:`More appointments`, fr:`Autres rendez-vous`, it:`Altri appuntamenti` },
    'kalender.bookingDetail.noHistory': { de:`Keine weiteren Termine dieses Kunden`, en:`No other appointments for this client`, fr:`Aucun autre rendez-vous pour ce client`, it:`Nessun altro appuntamento per questo cliente` },
    'kalender.bookingDetail.confirm': { de:`Bestätigen`, en:`Confirm`, fr:`Confirmer`, it:`Conferma` },
    'kalender.bookingDetail.cancelAppt': { de:`Absagen`, en:`Cancel`, fr:`Annuler`, it:`Annulla` },
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

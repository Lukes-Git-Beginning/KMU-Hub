import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
const dir = resolve('src/renderer/src/i18n/messages')
const anchor = 'kontakte.detail.contactData'
const keys = {
  'kontakte.detail.consent': { de:'Einwilligungen', en:'Consent', fr:'Consentements', it:'Consensi' },
  'kontakte.detail.deals': { de:'Deals', en:'Deals', fr:'Opportunités', it:'Deal' },
  'kontakte.detail.dealsEmpty': { de:'Keine Deals verknüpft', en:'No deals linked', fr:'Aucune opportunité liée', it:'Nessun deal collegato' },
  'kontakte.detail.emails': { de:'E-Mail-Verlauf', en:'Email history', fr:'Historique e-mails', it:'Cronologia e-mail' },
  'kontakte.detail.emailsEmpty': { de:'Keine E-Mails vorhanden', en:'No emails', fr:'Aucun e-mail', it:'Nessuna e-mail' },
  'kontakte.detail.tasks': { de:'Aufgaben', en:'Tasks', fr:'Tâches', it:'Attività' },
  'kontakte.detail.tasksEmpty': { de:'Keine Aufgaben', en:'No tasks', fr:'Aucune tâche', it:'Nessuna attività' },
  'kontakte.detail.timeline': { de:'Chronik', en:'Timeline', fr:'Chronologie', it:'Cronologia' },
  'kontakte.tag.add': { de:'Tag', en:'Tag', fr:'Tag', it:'Tag' },
  'kontakte.tag.addError': { de:'Tag konnte nicht hinzugefügt werden', en:'Could not add tag', fr:"Impossible d'ajouter le tag", it:'Impossibile aggiungere il tag' },
  'kontakte.tag.allAssigned': { de:'Alle Tags zugewiesen', en:'All tags assigned', fr:'Tous les tags assignés', it:'Tutti i tag assegnati' },
  'kontakte.tag.removeError': { de:'Tag konnte nicht entfernt werden', en:'Could not remove tag', fr:'Impossible de retirer le tag', it:'Impossibile rimuovere il tag' },
}
const report={}
for (const loc of ['de','en','fr','it']) {
  const file=resolve(dir,`${loc}.json`); const obj=JSON.parse(readFileSync(file,'utf8'))
  let lines=readFileSync(file,'utf8').split('\n')
  const nk=Object.keys(keys).filter(k=>!(k in obj)).sort()
  if(!nk.length){report[loc]=0;continue}
  const block=nk.map(k=>`  ${JSON.stringify(k)}: ${JSON.stringify(keys[k][loc])},`)
  const idx=lines.findIndex(l=>l.trimStart().startsWith(`"${anchor}":`))
  if(idx===-1) throw new Error(`anchor missing ${loc}`)
  lines=[...lines.slice(0,idx+1),...block,...lines.slice(idx+1)]
  const out=lines.join('\n'); JSON.parse(out); writeFileSync(file,out,'utf8'); report[loc]=block.length
}
console.log(JSON.stringify(report))

import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
const dir = resolve('src/renderer/src/i18n/messages')
const groups = [
  { anchor: 'common.delete', keys: {
    'common.done': { de:'Fertig', en:'Done', fr:'Terminé', it:'Fatto' },
  }},
  { anchor: 'kontakte.action.duplicate', keys: {
    'kontakte.action.assignToGroup': { de:'Zu Gruppe hinzufügen', en:'Add to group', fr:'Ajouter au groupe', it:'Aggiungi al gruppo' },
  }},
  { anchor: 'kontakte.action.favorite', keys: {
    'kontakte.groupAssign.title': { de:'Gruppen von {name}', en:'Groups for {name}', fr:'Groupes de {name}', it:'Gruppi di {name}' },
    'kontakte.groupAssign.description': { de:'Wähle die Gruppen, zu denen dieser Kontakt gehört.', en:'Choose the groups this contact belongs to.', fr:'Choisissez les groupes auxquels appartient ce contact.', it:'Scegli i gruppi a cui appartiene questo contatto.' },
    'kontakte.groupAssign.empty': { de:'Noch keine Gruppen. Lege zuerst eine Gruppe an.', en:'No groups yet. Create a group first.', fr:"Aucun groupe. Créez d'abord un groupe.", it:'Ancora nessun gruppo. Crea prima un gruppo.' },
  }},
]
const report = {}
for (const loc of ['de','en','fr','it']) {
  const file = resolve(dir, `${loc}.json`)
  const obj = JSON.parse(readFileSync(file,'utf8'))
  let lines = readFileSync(file,'utf8').split('\n')
  let added = 0
  for (const g of groups) {
    const nk = Object.keys(g.keys).filter(k=>!(k in obj)).sort()
    if(!nk.length) continue
    const block = nk.map(k=>`  ${JSON.stringify(k)}: ${JSON.stringify(g.keys[k][loc])},`)
    const idx = lines.findIndex(l=>l.trimStart().startsWith(`"${g.anchor}":`))
    if(idx===-1) throw new Error(`anchor ${g.anchor} missing ${loc}`)
    lines=[...lines.slice(0,idx+1),...block,...lines.slice(idx+1)]; added+=block.length
  }
  const out=lines.join('\n'); JSON.parse(out); writeFileSync(file,out,'utf8'); report[loc]=added
}
console.log(JSON.stringify(report))

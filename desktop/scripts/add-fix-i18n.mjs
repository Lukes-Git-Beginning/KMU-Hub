import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
const dir = resolve('src/renderer/src/i18n/messages')
const groups = [
  { anchor: 'kontakte.sort.sortedBy', keys: {
    'kontakte.view.grid': { de:'Raster', en:'Grid', fr:'Grille', it:'Griglia' },
    'kontakte.view.list': { de:'Liste', en:'List', fr:'Liste', it:'Elenco' },
    'kontakte.view.toggleAriaLabel': { de:'Ansicht umschalten', en:'Toggle view', fr:'Changer de vue', it:'Cambia vista' },
  }},
  { anchor: 'crm.activities.subjectPlaceholder', keys: {
    'crm.activities.sort.created_at': { de:'Erstellt', en:'Created', fr:'Créé', it:'Creato' },
    'crm.activities.sort.due_date': { de:'Fällig', en:'Due', fr:'Échéance', it:'Scadenza' },
    'crm.activities.sort.subject': { de:'Betreff', en:'Subject', fr:'Objet', it:'Oggetto' },
  }},
]
const report = {}
for (const loc of ['de','en','fr','it']) {
  const file = resolve(dir, `${loc}.json`)
  const obj = JSON.parse(readFileSync(file,'utf8'))
  let lines = readFileSync(file,'utf8').split('\n')
  let added = 0
  for (const g of groups) {
    const newKeys = Object.keys(g.keys).filter(k=>!(k in obj)).sort()
    if (!newKeys.length) continue
    const block = newKeys.map(k=>`  ${JSON.stringify(k)}: ${JSON.stringify(g.keys[k][loc])},`)
    const idx = lines.findIndex(l=>l.trimStart().startsWith(`"${g.anchor}":`))
    if (idx===-1) throw new Error(`anchor ${g.anchor} missing in ${loc}`)
    lines = [...lines.slice(0,idx+1), ...block, ...lines.slice(idx+1)]
    added += block.length
  }
  const out = lines.join('\n'); JSON.parse(out); writeFileSync(file,out,'utf8')
  report[loc]=added
}
console.log(JSON.stringify(report))

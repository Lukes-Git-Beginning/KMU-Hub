import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
const dir = resolve('src/renderer/src/i18n/messages')
const anchor = 'common.edit'
const keys = {
  'common.sort.ascending': { de:'Aufsteigend', en:'Ascending', fr:'Croissant', it:'Crescente' },
  'common.sort.descending': { de:'Absteigend', en:'Descending', fr:'Décroissant', it:'Decrescente' },
  'common.sort.direction': { de:'Richtung', en:'Direction', fr:'Direction', it:'Direzione' },
  'common.sort.sortBy': { de:'Sortieren nach', en:'Sort by', fr:'Trier par', it:'Ordina per' },
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

import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
const dir = resolve('src/renderer/src/i18n/messages')
const key = 'kontakte.sort.sortBy'
const vals = { de:'Sortieren nach', en:'Sort by', fr:'Trier par', it:'Ordina per' }
const anchor = 'kontakte.sort.name'
const report = {}
for (const loc of ['de','en','fr','it']) {
  const file = resolve(dir, `${loc}.json`)
  const obj = JSON.parse(readFileSync(file,'utf8'))
  if (key in obj) { report[loc]='exists'; continue }
  let lines = readFileSync(file,'utf8').split('\n')
  const idx = lines.findIndex(l=>l.trimStart().startsWith(`"${anchor}":`))
  if (idx===-1) throw new Error(`anchor missing ${loc}`)
  lines = [...lines.slice(0,idx+1), `  ${JSON.stringify(key)}: ${JSON.stringify(vals[loc])},`, ...lines.slice(idx+1)]
  const out = lines.join('\n'); JSON.parse(out); writeFileSync(file,out,'utf8')
  report[loc]='added'
}
console.log(JSON.stringify(report))

import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
const dir = resolve('src/renderer/src/i18n/messages')
const anchor = 'kontakte.detail.tags'
const K = {
  'moduleSettings.entries.moduleAssignment': { de: 'Modul-Zuteilung', en: 'Module assignment', fr: 'Attribution de modules', it: 'Assegnazione moduli' },
  'team.moduleAssignment.subtitle': { de: 'Welche Module welcher Mitarbeiter nutzen darf (wirkt auf die Abrechnung).', en: 'Which modules each employee may use (affects billing).', fr: 'Quels modules chaque employé peut utiliser (impacte la facturation).', it: 'Quali moduli può usare ciascun dipendente (influisce sulla fatturazione).' },
}
const report = {}
for (const loc of ['de','en','fr','it']) {
  const file = resolve(dir, `${loc}.json`)
  const obj = JSON.parse(readFileSync(file,'utf8'))
  let lines = readFileSync(file,'utf8').split('\n')
  const nk = Object.keys(K).filter(k=>!(k in obj)).sort()
  if (!nk.length) { report[loc]=0; continue }
  const block = nk.map(k=>`  ${JSON.stringify(k)}: ${JSON.stringify(K[k][loc])},`)
  const idx = lines.findIndex(l=>l.trimStart().startsWith(`"${anchor}":`))
  if (idx===-1) throw new Error(`anchor missing ${loc}`)
  lines = [...lines.slice(0,idx+1),...block,...lines.slice(idx+1)]
  const out = lines.join('\n'); JSON.parse(out); writeFileSync(file,out,'utf8'); report[loc]=block.length
}
console.log(JSON.stringify(report))

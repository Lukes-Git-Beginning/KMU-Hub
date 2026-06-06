import { readFileSync, readdirSync, statSync } from 'node:fs'
import { resolve, join } from 'node:path'
const de = JSON.parse(readFileSync(resolve('src/renderer/src/i18n/messages/de.json'),'utf8'))
const roots = ['src/renderer/src/modules/kontakte','src/renderer/src/modules/crm','src/renderer/src/components/shared']
const files=[]
function walk(d){for(const e of readdirSync(d)){const p=join(d,e); if(statSync(p).isDirectory())walk(p); else if(p.endsWith('.tsx')||p.endsWith('.ts'))files.push(p)}}
roots.forEach(r=>walk(resolve(r)))
const staticKeys=new Set(), dynPrefixes=new Set()
const reStatic=/\bt\(\s*'([^']+)'/g
const reTpl=/\bt\(\s*`([^`$]*)\$\{/g
for(const f of files){const s=readFileSync(f,'utf8')
  let m; while((m=reStatic.exec(s)))staticKeys.add(m[1])
  while((m=reTpl.exec(s)))dynPrefixes.add(m[1])
}
const missing=[...staticKeys].filter(k=>!(k in de)).sort()
console.log('static keys:',staticKeys.size,' dynamic prefixes:',[...dynPrefixes])
console.log('=== MISSING ('+missing.length+') ===')
console.log(missing.join('\n'))

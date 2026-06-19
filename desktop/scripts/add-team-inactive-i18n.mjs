import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
const dir = resolve('src/renderer/src/i18n/messages')
const val = { de: 'Inaktiv', en: 'Inactive', fr: 'Inactif', it: 'Inattivo' }
for (const loc of ['de', 'en', 'fr', 'it']) {
  const file = resolve(dir, `${loc}.json`)
  const obj = JSON.parse(readFileSync(file, 'utf8'))
  if ('team.page.statusInactive' in obj) continue
  let lines = readFileSync(file, 'utf8').split('\n')
  const idx = lines.findIndex((l) => l.trimStart().startsWith('"team.page.deactivated":'))
  if (idx === -1) throw new Error('anchor not found in ' + loc)
  lines = [...lines.slice(0, idx + 1), `  "team.page.statusInactive": ${JSON.stringify(val[loc])},`, ...lines.slice(idx + 1)]
  const out = lines.join('\n'); JSON.parse(out); writeFileSync(file, out, 'utf8')
}
console.log('done')

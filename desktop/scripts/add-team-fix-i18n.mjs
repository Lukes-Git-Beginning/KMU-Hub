import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'

const dir = resolve('src/renderer/src/i18n/messages')

// 8 keys that wrongly use ICU double-braces {{var}} → must be single-brace {var}.
const doubleBraceKeys = [
  'team.member.modules.lastUsed',
  'team.moduleAssignment.bulk.confirmBody',
  'team.moduleAssignment.bulk.label',
  'team.moduleAssignment.bulk.toast',
  'team.moduleAssignment.cellAria',
  'team.moduleAssignment.confirmRevoke.body',
  'team.moduleAssignment.toast.granted',
  'team.moduleAssignment.toast.revoked',
]

// Missing key team.page.title (was only a defaultValue fallback).
const pageTitle = { de: 'Team', en: 'Team', fr: 'Équipe', it: 'Team' }

const locales = ['de', 'en', 'fr', 'it']
const report = {}

function insertAfter(lines, anchorKey, blockLines) {
  const idx = lines.findIndex((l) => l.trimStart().startsWith(`"${anchorKey}":`))
  if (idx === -1) throw new Error(`anchor not found: ${anchorKey}`)
  return [...lines.slice(0, idx + 1), ...blockLines, ...lines.slice(idx + 1)]
}

for (const loc of locales) {
  const file = resolve(dir, `${loc}.json`)
  const obj = JSON.parse(readFileSync(file, 'utf8'))
  let lines = readFileSync(file, 'utf8').split('\n')
  let fixed = 0

  // Fix double braces only on the affected key lines.
  lines = lines.map((l) => {
    const trimmed = l.trimStart()
    if (doubleBraceKeys.some((k) => trimmed.startsWith(`"${k}":`))) {
      const next = l.replace(/\{\{/g, '{').replace(/\}\}/g, '}')
      if (next !== l) fixed++
      return next
    }
    return l
  })

  // Add team.page.title if missing.
  let titleAdded = 0
  if (!('team.page.title' in obj)) {
    lines = insertAfter(lines, 'team.page.tab.selfservice', [`  "team.page.title": ${JSON.stringify(pageTitle[loc])},`])
    titleAdded = 1
  }

  const out = lines.join('\n')
  JSON.parse(out) // validate
  writeFileSync(file, out, 'utf8')
  report[loc] = { bracesFixed: fixed, titleAdded }
}
console.log(JSON.stringify(report, null, 2))

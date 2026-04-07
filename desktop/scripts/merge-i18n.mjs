/**
 * Merge all i18n additions/*.json into messages/de.json
 * Run: node desktop/scripts/merge-i18n.mjs
 */
import { readFileSync, readdirSync, writeFileSync } from 'fs'
import { join, basename } from 'path'

const i18nDir = join(import.meta.dirname, '..', 'src', 'renderer', 'src', 'i18n')
const messagesDir = join(i18nDir, 'messages')
const additionsDir = join(i18nDir, 'additions')

// 1. Load base de.json
const baseDE = JSON.parse(readFileSync(join(messagesDir, 'de.json'), 'utf-8'))
const baseKeyCount = Object.keys(baseDE).length
console.log(`Base de.json: ${baseKeyCount} keys`)

// 2. Load all additions
const additionFiles = readdirSync(additionsDir).filter(f => f.endsWith('.json')).sort()
console.log(`\nAdditions files: ${additionFiles.length}`)

let totalAdditionKeys = 0
const conflicts = []
const merged = { ...baseDE }

for (const file of additionFiles) {
  const data = JSON.parse(readFileSync(join(additionsDir, file), 'utf-8'))
  const keys = Object.keys(data)
  totalAdditionKeys += keys.length
  console.log(`  ${file}: ${keys.length} keys`)

  for (const key of keys) {
    if (key in baseDE) {
      conflicts.push({ key, file, baseValue: baseDE[key], additionValue: data[key] })
    }
    merged[key] = data[key]
  }
}

// 3. Sort keys alphabetically
const sorted = {}
for (const key of Object.keys(merged).sort()) {
  sorted[key] = merged[key]
}

// 4. Write
writeFileSync(join(messagesDir, 'de.json'), JSON.stringify(sorted, null, 2) + '\n', 'utf-8')

const finalKeyCount = Object.keys(sorted).length
console.log(`\n--- Results ---`)
console.log(`Base keys:      ${baseKeyCount}`)
console.log(`Addition keys:  ${totalAdditionKeys}`)
console.log(`Final keys:     ${finalKeyCount}`)
console.log(`Net new:        ${finalKeyCount - baseKeyCount}`)

if (conflicts.length > 0) {
  console.log(`\nConflicts (${conflicts.length} keys overridden by additions):`)
  for (const c of conflicts) {
    console.log(`  ${c.key} (from ${c.file})`)
    if (c.baseValue !== c.additionValue) {
      console.log(`    base: "${c.baseValue}"`)
      console.log(`    new:  "${c.additionValue}"`)
    }
  }
} else {
  console.log(`\nNo conflicts — all addition keys are new.`)
}

/**
 * i18n batch for RBAC R-3 batch 2 (team actions, dashboard level 2,
 * admin/security tab gating): a single new gate key — the dashboard
 * widget-grid empty state shown when RBAC filtering removed every widget.
 *
 * Inserts new keys at their alphabetical position relative to the existing
 * order (same as i18n-rbac-r3.mjs — no global re-sort).
 *
 * Run: node scripts/i18n-rbac-r3b2.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))
const MESSAGES_DIR = join(here, '..', 'src', 'renderer', 'src', 'i18n', 'messages')
const LOCALES = ['de', 'en', 'fr', 'it']

/** @type {Record<string, [string, string, string, string]>} key → [de, en, fr, it] */
const ADD = {
  'rbac.gate.dashboardEmpty': [
    'Für deine Rolle sind hier keine Widgets verfügbar.',
    'No widgets are available for your role here.',
    'Aucun widget n’est disponible ici pour votre rôle.',
    'Nessun widget disponibile qui per il tuo ruolo.',
  ],
  'rbac.gate.profileRestricted': [
    'Weitere Details sind für deine Rolle nicht sichtbar.',
    'Further details are not visible for your role.',
    'Les autres détails ne sont pas visibles pour votre rôle.',
    'Ulteriori dettagli non sono visibili per il tuo ruolo.',
  ],
}

for (const [li, locale] of LOCALES.entries()) {
  const file = join(MESSAGES_DIR, `${locale}.json`)
  const data = JSON.parse(readFileSync(file, 'utf8'))

  // Insert new keys at their alphabetical position relative to existing order.
  const pending = Object.keys(ADD)
    .filter((k) => !(k in data))
    .sort()
  const out = {}
  const existing = Object.keys(data)
  let pi = 0
  for (const key of existing) {
    while (pi < pending.length && pending[pi] < key) {
      out[pending[pi]] = ADD[pending[pi]][li]
      pi++
    }
    out[key] = data[key]
  }
  while (pi < pending.length) {
    out[pending[pi]] = ADD[pending[pi]][li]
    pi++
  }

  writeFileSync(file, JSON.stringify(out, null, 2) + '\n', 'utf8')
  console.log(`${locale}: +${pending.length} keys`)
}
console.log('done')

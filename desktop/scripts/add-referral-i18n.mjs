import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'

const dir = resolve('src/renderer/src/i18n/messages')
const anchor = 'kontakte.detail.tags'

// Count-neutral phrasing ("{count}× …") — ICU plural is broken project-wide (P0.7).
const K = {
  'kontakte.referral.title': { de: 'Empfohlen von', en: 'Referred by', fr: 'Recommandé par', it: 'Segnalato da' },
  'kontakte.referral.add': { de: 'Empfehler wählen', en: 'Choose referrer', fr: 'Choisir le référent', it: 'Scegli referente' },
  'kontakte.referral.remove': { de: 'Entfernen', en: 'Remove', fr: 'Retirer', it: 'Rimuovi' },
  'kontakte.referral.search': { de: 'Kontakt suchen…', en: 'Search contact…', fr: 'Rechercher un contact…', it: 'Cerca contatto…' },
  'kontakte.referral.noResults': { de: 'Keine Treffer', en: 'No matches', fr: 'Aucun résultat', it: 'Nessun risultato' },
  'kontakte.referral.unknown': { de: 'Unbekannt', en: 'Unknown', fr: 'Inconnu', it: 'Sconosciuto' },
  'kontakte.referral.reportTitle': { de: 'Top-Empfehler', en: 'Top referrers', fr: 'Meilleurs référents', it: 'Migliori segnalatori' },
  'kontakte.referral.reportEmpty': { de: 'Noch keine Empfehlungen erfasst.', en: 'No referrals recorded yet.', fr: 'Aucune recommandation enregistrée.', it: 'Nessuna segnalazione registrata.' },
  'kontakte.referral.reportCount': { de: '{count}× empfohlen', en: 'referred {count}×', fr: '{count}× recommandé', it: '{count}× segnalato' },
}

const report = {}
for (const loc of ['de', 'en', 'fr', 'it']) {
  const file = resolve(dir, `${loc}.json`)
  const obj = JSON.parse(readFileSync(file, 'utf8'))
  let lines = readFileSync(file, 'utf8').split('\n')
  const nk = Object.keys(K).filter((k) => !(k in obj)).sort()
  if (!nk.length) { report[loc] = 0; continue }
  const block = nk.map((k) => `  ${JSON.stringify(k)}: ${JSON.stringify(K[k][loc])},`)
  const idx = lines.findIndex((l) => l.trimStart().startsWith(`"${anchor}":`))
  if (idx === -1) throw new Error(`anchor ${anchor} missing in ${loc}`)
  lines = [...lines.slice(0, idx + 1), ...block, ...lines.slice(idx + 1)]
  const out = lines.join('\n')
  JSON.parse(out)
  writeFileSync(file, out, 'utf8')
  report[loc] = block.length
}
console.log(JSON.stringify(report))

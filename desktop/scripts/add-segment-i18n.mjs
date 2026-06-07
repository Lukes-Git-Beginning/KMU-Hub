import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'

const dir = resolve('src/renderer/src/i18n/messages')
const anchor = 'kontakte.detail.tags'

const K = {
  'crm.segment.a': { de: 'Segment A', en: 'Segment A', fr: 'Segment A', it: 'Segmento A' },
  'crm.segment.b': { de: 'Segment B', en: 'Segment B', fr: 'Segment B', it: 'Segmento B' },
  'crm.segment.c': { de: 'Segment C', en: 'Segment C', fr: 'Segment C', it: 'Segmento C' },
  'crm.segment.badge': { de: 'Segment {segment}', en: 'Segment {segment}', fr: 'Segment {segment}', it: 'Segmento {segment}' },
  'crm.segment.badgeTooltip': { de: 'Mandanten-Segment (nach Umsatzpotenzial)', en: 'Client segment (by revenue potential)', fr: 'Segment client (par potentiel de CA)', it: 'Segmento cliente (per potenziale di fatturato)' },
  'crm.segment.thresholdA': { de: 'Schwelle Segment A (ab €)', en: 'Segment A threshold (from €)', fr: 'Seuil segment A (à partir de €)', it: 'Soglia segmento A (da €)' },
  'crm.segment.thresholdB': { de: 'Schwelle Segment B (ab €)', en: 'Segment B threshold (from €)', fr: 'Seuil segment B (à partir de €)', it: 'Soglia segmento B (da €)' },
  'crm.segment.ruleFrom': { de: 'ab {value}', en: 'from {value}', fr: 'à partir de {value}', it: 'da {value}' },
  'crm.segment.ruleRange': { de: '{from} – {to}', en: '{from} – {to}', fr: '{from} – {to}', it: '{from} – {to}' },
  'crm.segment.ruleUnder': { de: 'unter {value}', en: 'under {value}', fr: 'moins de {value}', it: 'sotto {value}' },
  'crm.settings.segments.title': { de: 'Mandanten-Segmente', en: 'Client segments', fr: 'Segments clients', it: 'Segmenti clienti' },
  'crm.settings.segments.desc': { de: 'Regelbasierte A/B/C-Einteilung nach Umsatzpotenzial.', en: 'Rule-based A/B/C classification by revenue potential.', fr: 'Classement A/B/C par potentiel de chiffre d’affaires.', it: 'Classificazione A/B/C in base al potenziale di fatturato.' },
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

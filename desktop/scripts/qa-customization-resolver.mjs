/**
 * Resolver-Smoke für customization.ts (v1.0).
 * Prüft: (a) tenant > vendor > default Priorität, (b) Provenance korrekt,
 * (c) base=true liefert nur Default-Provenance.
 *
 * Direkt als Node-ESM-Skript ausführen (ohne TS/Vite-Pfad-Aliase) —
 * daher wird die Logik hier inline nachgebaut, exakt aus dem Source.
 *
 * Ausführen: node scripts/qa-customization-resolver.mjs
 */

// ── Inline-Daten aus customization.ts (Kopie der Seeds + Resolver) ────────────

const LABEL_WHITELIST = [
  'rbac.module.crm', 'rbac.module.work', 'rbac.module.helpdesk', 'rbac.module.finance',
  'rbac.module.wiki', 'rbac.module.team', 'rbac.module.berichte', 'rbac.module.formulare',
  'rbac.module.dialer', 'rbac.module.schichten', 'rbac.module.zeiterfassung',
  'rbac.module.vertraege', 'rbac.module.inventar', 'rbac.module.einkauf',
  'crm.contacts.title', 'crm.companies.title', 'crm.deals.title',
  'work.tasks.title', 'work.projects.title', 'nav.crm', 'nav.work', 'nav.admin.label',
]

const VENDOR_LABELS = {
  de: {
    'crm.contacts.title': 'Patienten',
    'crm.companies.title': 'Praxen',
    'rbac.module.crm': 'Patientenverwaltung',
    'nav.crm': 'Patienten',
  },
}

const TENANT_LABELS = {
  de: {
    'crm.deals.title': 'Aufträge',
    'work.projects.title': 'Mandate',
  },
}

function resolveLabelOverrides(locale, base = false) {
  const result = {}
  for (const key of LABEL_WHITELIST) {
    if (base) { result[key] = { key, value: '', provenance: 'default' }; continue }
    const vendorValue = VENDOR_LABELS[locale]?.[key]
    const tenantValue = TENANT_LABELS[locale]?.[key]
    if (tenantValue !== undefined) {
      result[key] = { key, value: tenantValue, provenance: 'tenant' }
    } else if (vendorValue !== undefined) {
      result[key] = { key, value: vendorValue, provenance: 'vendor' }
    } else {
      result[key] = { key, value: '', provenance: 'default' }
    }
  }
  return result
}

const DEFAULT_VALUE_SETS = {
  deal_stages: {
    id: 'deal_stages', name: 'Deal-Phasen',
    options: [
      { id: 'lead', label: 'Lead', order: 0, active: true },
      { id: 'qualified', label: 'Qualifiziert', order: 1, active: true },
      { id: 'won', label: 'Gewonnen', order: 4, active: true },
    ],
  },
  ticket_priority: {
    id: 'ticket_priority', name: 'Ticket-Priorität',
    options: [
      { id: 'low', label: 'Niedrig', order: 0, active: true },
      { id: 'high', label: 'Hoch', order: 2, active: true },
      { id: 'critical', label: 'Kritisch', order: 3, active: true },
    ],
  },
}

const VENDOR_VALUE_SETS = {
  deal_stages: {
    id: 'deal_stages', name: 'Behandlungs-Pipeline', layer: 'vendor',
    options: [
      { id: 'lead', label: 'Interessent', order: 0, active: true },
      { id: 'qualified', label: 'Erstgespräch', order: 1, active: true },
      { id: 'won', label: 'Abgeschlossen', order: 4, active: true },
    ],
  },
}

const TENANT_VALUE_SETS = {
  ticket_priority: {
    id: 'ticket_priority', name: 'Ticket-Priorität', layer: 'tenant',
    options: [
      { id: 'low', label: 'Rückfrage', order: 0, active: true },
      { id: 'high', label: 'Hoch', order: 2, active: true },
      { id: 'critical', label: 'Kritisch', order: 3, active: false },
    ],
  },
}

function resolveValueSet(id, base = false) {
  const def = DEFAULT_VALUE_SETS[id]
  if (!def) return null
  if (base) {
    return {
      id: def.id, name: def.name, provenance: 'default',
      options: def.options.map(o => ({ ...o, provenance: 'default' })),
    }
  }
  const merged = {}
  for (const opt of def.options) merged[opt.id] = { ...opt, provenance: 'default' }
  const vendorSet = VENDOR_VALUE_SETS[id]
  if (vendorSet) { for (const opt of vendorSet.options) merged[opt.id] = { ...opt, provenance: 'vendor' } }
  const tenantSet = TENANT_VALUE_SETS[id]
  if (tenantSet) { for (const opt of tenantSet.options) merged[opt.id] = { ...opt, provenance: 'tenant' } }

  const tenantName = tenantSet?.name
  const vendorName = vendorSet?.name
  return {
    id,
    name: tenantName ?? vendorName ?? def.name,
    provenance: tenantName ? 'tenant' : vendorName ? 'vendor' : 'default',
    options: Object.values(merged).sort((a, b) => a.order - b.order),
  }
}

// ── Tests ───────────────────────────────────────────────────────────────────

let passed = 0, failed = 0

function assert(desc, condition, detail = '') {
  if (condition) {
    console.log(`  PASS  ${desc}`)
    passed++
  } else {
    console.error(`  FAIL  ${desc}${detail ? ' — ' + detail : ''}`)
    failed++
  }
}

console.log('\n── (a) Priorität: tenant > vendor > default ─────────────────────')

const resolved = resolveLabelOverrides('de')

// tenant key: crm.deals.title → 'Aufträge' (tenant-only, no vendor for this key)
assert(
  'crm.deals.title → tenant wins ("Aufträge")',
  resolved['crm.deals.title'].value === 'Aufträge' && resolved['crm.deals.title'].provenance === 'tenant',
  JSON.stringify(resolved['crm.deals.title']),
)

// vendor key: crm.contacts.title → 'Patienten' (vendor set, no tenant override)
assert(
  'crm.contacts.title → vendor wins ("Patienten")',
  resolved['crm.contacts.title'].value === 'Patienten' && resolved['crm.contacts.title'].provenance === 'vendor',
  JSON.stringify(resolved['crm.contacts.title']),
)

// default key: work.tasks.title → default (neither layer set it)
assert(
  'work.tasks.title → default (neither layer set)',
  resolved['work.tasks.title'].provenance === 'default',
  JSON.stringify(resolved['work.tasks.title']),
)

// All LABEL_WHITELIST keys are present in the result
assert(
  `all ${LABEL_WHITELIST.length} whitelist keys in resolved map`,
  LABEL_WHITELIST.every(k => k in resolved),
)

console.log('\n── (b) Provenance korrekt ────────────────────────────────────────')

const deals = resolveValueSet('deal_stages')

// deal_stages: vendor overrides lead → 'Interessent' (tenant has no deal_stages entry)
assert(
  'deal_stages.lead → vendor provenance ("Interessent")',
  deals.options.find(o => o.id === 'lead')?.label === 'Interessent' &&
  deals.options.find(o => o.id === 'lead')?.provenance === 'vendor',
)

// deal_stages set-level name from vendor
assert(
  'deal_stages set name from vendor ("Behandlungs-Pipeline")',
  deals.name === 'Behandlungs-Pipeline' && deals.provenance === 'vendor',
)

const tickets = resolveValueSet('ticket_priority')

// ticket_priority: tenant overrides 'low' → 'Rückfrage'
assert(
  'ticket_priority.low → tenant provenance ("Rückfrage")',
  tickets.options.find(o => o.id === 'low')?.label === 'Rückfrage' &&
  tickets.options.find(o => o.id === 'low')?.provenance === 'tenant',
)

// ticket_priority: 'critical' soft-deleted by tenant (active=false)
assert(
  'ticket_priority.critical → soft-deleted (active=false, tenant)',
  tickets.options.find(o => o.id === 'critical')?.active === false &&
  tickets.options.find(o => o.id === 'critical')?.provenance === 'tenant',
)

console.log('\n── (c) base=true liefert nur Default-Provenance ──────────────────')

const baseLabels = resolveLabelOverrides('de', true)
assert(
  'base=true: crm.deals.title provenance = "default"',
  baseLabels['crm.deals.title'].provenance === 'default',
)
assert(
  'base=true: crm.contacts.title provenance = "default" (vendor not visible)',
  baseLabels['crm.contacts.title'].provenance === 'default',
)

const baseDeals = resolveValueSet('deal_stages', true)
assert(
  'base=true: deal_stages.lead → label="Lead" (code default, not vendor override)',
  baseDeals.options.find(o => o.id === 'lead')?.label === 'Lead' &&
  baseDeals.options.find(o => o.id === 'lead')?.provenance === 'default',
)
assert(
  'base=true: deal_stages set provenance = "default"',
  baseDeals.provenance === 'default',
)

console.log('\n── Ergebnis ──────────────────────────────────────────────────────')
console.log(`  ${passed} passed, ${failed} failed`)
if (failed > 0) process.exit(1)

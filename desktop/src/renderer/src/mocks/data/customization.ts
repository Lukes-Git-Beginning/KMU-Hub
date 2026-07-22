/**
 * Customization data layer (v1.0 Fundament) — stateful in-memory store for
 * label-override and value-set overlay mocks.
 *
 * Overlay-Prinzip: drei Provenance-Stufen (default < vendor < tenant).
 * Nur Abweichungen werden gespeichert (sparse) — identisch zum
 * USER_OVERRIDES / applyUserOverrides-Muster aus R-6.
 *
 * Custom Fields (Dimension A) sind in v1.0 NICHT enthalten — sie haben
 * eigene BE-Persistenz und werden in v1.1 unified. Diese Datei deckt:
 *   B · Label-Overrides (LABEL_WHITELIST, i18n-Overlay)
 *   M · Value-Sets      (Deal-Phasen, Ticket-Prioritäten, Projekt-Status, …)
 */
import type {
  ConfigLayer,
  ConfigProvenance,
  LocaleLabelMap,
  ResolvedLabel,
  ResolvedLabelMap,
  ResolvedValueSet,
  ResolvedValueSetOption,
  ValueSet,
  ValueSetOption,
} from '@/api/customization-types'
import { writeAuditEvent } from './audit-events'

// ── Active layer detection ────────────────────────────────────────────────────

/**
 * Returns which overlay layer the current session writes to.
 *
 * v1.0: always 'tenant' — vendor-session detection via R-5 vendor_access
 * (security:vendor_access + getDemoVendorSession()) comes in v1.1 once
 * the vendor-access flow is fully connected. The vendor layer is already
 * fully supported in the data structures and seeds below.
 */
export function activeConfigLayer(): ConfigLayer {
  return 'tenant'
}

// ── B · Label-Overrides ───────────────────────────────────────────────────────

/**
 * Curated whitelist of i18n keys that tenants/vendors may override in v1.
 *
 * ★ RULE (Darien 2026-07-22): MODULE NAMES ARE IMMUTABLE. The sidebar nav and
 * the module display name (rbac.module.* / layout.navItems.*) are the product's
 * stable anchors — support, docs, training and screenshots depend on them, and a
 * renamable module name drifts into "the same module under three names". Only
 * CONTENT terminology *inside* a module is customizable (object headings, record
 * nouns, categories, field labels — e.g. "Kunden" → "Patienten"). So the
 * whitelist contains object/content headings only, never module/nav identity.
 *
 * Verified against de.json (all keys exist in the static bundles):
 *   - crm.contacts.title / crm.companies.title / crm.deals.title = object headings
 *   - work.tasks.title / work.projects.title = work object headings
 */
export const LABEL_WHITELIST: string[] = [
  // Object/content headings inside modules — the domain nouns KMUs rebrand
  // (e.g. a practice renames "Kontakte" → "Patienten"). Module identity (the
  // sidebar nav + rbac.module.* display name) stays fixed and is NOT listed here.
  'crm.contacts.title',
  'crm.companies.title',
  'crm.deals.title',
  'work.tasks.title',
  'work.projects.title',
  // Edit-in-place instrumented content (Kontakte pilot) — record-noun categories
  // a practice rebrands (Kunden → Patienten). Instrumented via <EditableText>.
  'kontakte.category.all',
  'kontakte.category.employee',
  'kontakte.category.customers',
  'kontakte.category.partner',
]

// LocaleLabelMap (locale → key → value) is defined in customization-types.ts.

/**
 * Vendor layer seeds: Zentria sets these during onboarding for a healthcare
 * demo tenant — demonstrates that CONTENT terminology ("Kontakte" → "Patienten"
 * etc.) comes from the vendor layer (provenance = 'vendor'), so the tenant can
 * still override. Module IDENTITY (nav + rbac.module.*) is intentionally NOT
 * seeded here — module names stay fixed (see LABEL_WHITELIST rule).
 */
export const VENDOR_LABELS: LocaleLabelMap = {
  de: {
    'crm.contacts.title': 'Patienten',
    'crm.companies.title': 'Praxen',
  },
  en: {
    'crm.contacts.title': 'Patients',
    'crm.companies.title': 'Practices',
  },
  fr: {
    'crm.contacts.title': 'Patients',
    'crm.companies.title': 'Cabinets',
  },
  it: {
    'crm.contacts.title': 'Pazienti',
    'crm.companies.title': 'Studi',
  },
}

/**
 * Tenant layer seeds: the customer renames "Deals" to "Aufträge" in DE
 * (common in B2B service businesses) — overrides the vendor label for crm.deals
 * and adds a work label override. Provenance = 'tenant', wins over vendor.
 */
export const TENANT_LABELS: LocaleLabelMap = {
  de: {
    'crm.deals.title': 'Aufträge',
    'work.projects.title': 'Mandate',
  },
  en: {
    'crm.deals.title': 'Orders',
    'work.projects.title': 'Mandates',
  },
  fr: {
    'crm.deals.title': 'Commandes',
    'work.projects.title': 'Mandats',
  },
  it: {
    'crm.deals.title': 'Ordini',
    'work.projects.title': 'Mandati',
  },
}

// Runtime mutation maps (start from seeds, mutated by CRUD ops)
const vendorLabels: LocaleLabelMap = structuredClone(VENDOR_LABELS)
const tenantLabels: LocaleLabelMap = structuredClone(TENANT_LABELS)

/**
 * Resolve all LABEL_WHITELIST keys for a locale, merging vendor → tenant.
 * Returns one ResolvedLabel per key with provenance.
 *
 * base=true → returns only the code default level (empty overlay), used by
 * future editors to show the "vanilla Cosmi" baseline (mirrors ?base=1 in R-6).
 */
export function resolveLabelOverrides(
  locale: string,
  base = false,
  draftOverlay?: LocaleLabelMap,
): ResolvedLabelMap {
  const result: ResolvedLabelMap = {}

  for (const key of LABEL_WHITELIST) {
    // The "default" value is whatever the static i18n bundle holds — we don't
    // store it here, we just mark provenance so the editor can display
    // "(Cosmi-Standard)" and the runtime merge replaces nothing for this key.
    const defaultEntry: ResolvedLabel = { key, value: '', provenance: 'default' }

    if (base) {
      result[key] = defaultEntry
      continue
    }

    // draft wins over tenant (4th layer, only supplied inside the editor sandbox).
    const draftValue = draftOverlay?.[locale]?.[key]
    if (draftValue !== undefined) {
      result[key] = { key, value: draftValue, provenance: 'draft' }
      continue
    }

    const vendorValue = vendorLabels[locale]?.[key]
    const tenantValue = tenantLabels[locale]?.[key]

    if (tenantValue !== undefined) {
      result[key] = { key, value: tenantValue, provenance: 'tenant' }
    } else if (vendorValue !== undefined) {
      result[key] = { key, value: vendorValue, provenance: 'vendor' }
    } else {
      result[key] = defaultEntry
    }
  }

  return result
}

/** Set or update a single label override on the given layer + locale. */
export function setLabelOverride(
  layer: ConfigLayer,
  locale: string,
  key: string,
  value: string,
): void {
  if (!LABEL_WHITELIST.includes(key)) {
    console.warn(`[customization] key "${key}" is not in LABEL_WHITELIST — skipped`)
    return
  }

  const store = layer === 'vendor' ? vendorLabels : tenantLabels
  const oldValue = store[locale]?.[key]

  store[locale] ??= {}
  store[locale][key] = value

  writeAuditEvent({
    action: 'customization.label_set',
    target: key,
    targetType: 'label_override',
    oldValue: oldValue !== undefined ? { locale, key, value: oldValue } : undefined,
    newValue: { locale, key, value, layer },
  })
}

/** Remove a label override from the given layer + locale (falls back to layer below). */
export function clearLabelOverride(layer: ConfigLayer, locale: string, key: string): void {
  const store = layer === 'vendor' ? vendorLabels : tenantLabels
  const oldValue = store[locale]?.[key]

  if (oldValue === undefined) return

  delete store[locale][key]

  writeAuditEvent({
    action: 'customization.label_removed',
    target: key,
    targetType: 'label_override',
    oldValue: { locale, key, value: oldValue, layer },
    newValue: undefined,
  })
}

/** Whether any label overrides exist for this tenant (badge / "has changes" check). */
export function hasLabelOverrides(locale?: string): boolean {
  if (locale) {
    return (
      Object.keys(vendorLabels[locale] ?? {}).length > 0 ||
      Object.keys(tenantLabels[locale] ?? {}).length > 0
    )
  }
  return (
    Object.values(vendorLabels).some((m) => Object.keys(m).length > 0) ||
    Object.values(tenantLabels).some((m) => Object.keys(m).length > 0)
  )
}

/** Reset all label overrides for a layer (rollback to layer below). */
export function clearAllLabelOverrides(layer: ConfigLayer): void {
  const store = layer === 'vendor' ? vendorLabels : tenantLabels
  for (const locale of Object.keys(store)) {
    delete store[locale]
  }
}

// ── M · Value-Sets ────────────────────────────────────────────────────────────

/** Default code-level value-sets (the baseline Cosmi ships with). */
const DEFAULT_VALUE_SETS: Record<string, Omit<ValueSet, 'layer'>> = {
  deal_stages: {
    id: 'deal_stages',
    name: 'Deal-Phasen',
    options: [
      { id: 'lead', label: 'Lead', color: 'hsl(215 16% 47%)', order: 0, active: true },
      { id: 'qualified', label: 'Qualifiziert', color: 'hsl(217 91% 60%)', order: 1, active: true },
      { id: 'proposal', label: 'Angebot', color: 'hsl(38 92% 50%)', order: 2, active: true },
      { id: 'negotiation', label: 'Verhandlung', color: 'hsl(25 95% 53%)', order: 3, active: true },
      { id: 'won', label: 'Gewonnen', color: 'hsl(142 71% 45%)', order: 4, active: true },
      { id: 'lost', label: 'Verloren', color: 'hsl(0 72% 51%)', order: 5, active: true },
    ],
  },
  ticket_priority: {
    id: 'ticket_priority',
    name: 'Ticket-Priorität',
    options: [
      { id: 'low', label: 'Niedrig', color: 'hsl(142 71% 45%)', order: 0, active: true },
      { id: 'medium', label: 'Mittel', color: 'hsl(38 92% 50%)', order: 1, active: true },
      { id: 'high', label: 'Hoch', color: 'hsl(25 95% 53%)', order: 2, active: true },
      { id: 'critical', label: 'Kritisch', color: 'hsl(0 72% 51%)', order: 3, active: true },
    ],
  },
  project_status: {
    id: 'project_status',
    name: 'Projekt-Status',
    options: [
      { id: 'planning', label: 'Planung', color: 'hsl(215 16% 47%)', order: 0, active: true },
      { id: 'active', label: 'Aktiv', color: 'hsl(217 91% 60%)', order: 1, active: true },
      { id: 'on_hold', label: 'Pausiert', color: 'hsl(38 92% 50%)', order: 2, active: true },
      { id: 'completed', label: 'Abgeschlossen', color: 'hsl(142 71% 45%)', order: 3, active: true },
      { id: 'cancelled', label: 'Abgebrochen', color: 'hsl(0 72% 51%)', order: 4, active: true },
    ],
  },
}

/**
 * Vendor value-set seeds: Zentria renames "Qualifiziert" → "Erstgespräch" for
 * the healthcare demo tenant (pipeline stage matches medical onboarding flow).
 * Shows provenance = 'vendor' for these options.
 */
export const VENDOR_VALUE_SETS: Record<string, ValueSet> = {
  deal_stages: {
    id: 'deal_stages',
    name: 'Behandlungs-Pipeline',
    layer: 'vendor',
    options: [
      { id: 'lead', label: 'Interessent', color: 'hsl(215 16% 47%)', order: 0, active: true },
      { id: 'qualified', label: 'Erstgespräch', color: 'hsl(217 91% 60%)', order: 1, active: true },
      { id: 'proposal', label: 'Diagnose', color: 'hsl(38 92% 50%)', order: 2, active: true },
      { id: 'negotiation', label: 'Behandlung', color: 'hsl(25 95% 53%)', order: 3, active: true },
      { id: 'won', label: 'Abgeschlossen', color: 'hsl(142 71% 45%)', order: 4, active: true },
      { id: 'lost', label: 'Kein Bedarf', color: 'hsl(0 72% 51%)', order: 5, active: true },
    ],
  },
}

/**
 * Tenant value-set seeds: the customer keeps the vendor's deal pipeline but
 * overrides ticket_priority — removes "Kritisch" (soft-delete) and renames
 * "Niedrig" → "Rückfrage". Shows provenance = 'tenant' for these options.
 */
export const TENANT_VALUE_SETS: Record<string, ValueSet> = {
  ticket_priority: {
    id: 'ticket_priority',
    name: 'Ticket-Priorität',
    layer: 'tenant',
    options: [
      { id: 'low', label: 'Rückfrage', color: 'hsl(142 71% 45%)', order: 0, active: true },
      { id: 'medium', label: 'Mittel', color: 'hsl(38 92% 50%)', order: 1, active: true },
      { id: 'high', label: 'Hoch', color: 'hsl(25 95% 53%)', order: 2, active: true },
      { id: 'critical', label: 'Kritisch', color: 'hsl(0 72% 51%)', order: 3, active: false },
    ],
  },
}

// Runtime mutation maps (start from seeds)
const vendorValueSets: Record<string, ValueSet> = structuredClone(VENDOR_VALUE_SETS)
const tenantValueSets: Record<string, ValueSet> = structuredClone(TENANT_VALUE_SETS)

/**
 * Resolve one value-set across all layers.
 * The tenant layer wins per option (by id): if a tenant option exists for
 * an id, it replaces the vendor/default option. Options only in vendor carry
 * provenance='vendor'; only in default carry provenance='default'.
 */
export function resolveValueSet(
  id: string,
  base = false,
  draftOverlay?: Record<string, Omit<ValueSet, 'layer'>>,
): ResolvedValueSet | null {
  const def = DEFAULT_VALUE_SETS[id]
  if (!def) return null

  if (base) {
    return {
      id: def.id,
      name: def.name,
      provenance: 'default',
      options: def.options.map((o) => ({ ...o, provenance: 'default' as ConfigProvenance })),
    }
  }

  // Build a merged option map: default → vendor → tenant → draft
  const merged: Record<string, ResolvedValueSetOption> = {}

  for (const opt of def.options) {
    merged[opt.id] = { ...opt, provenance: 'default' }
  }

  const vendorSet = vendorValueSets[id]
  if (vendorSet) {
    for (const opt of vendorSet.options) {
      merged[opt.id] = { ...opt, provenance: 'vendor' }
    }
  }

  const tenantSet = tenantValueSets[id]
  if (tenantSet) {
    for (const opt of tenantSet.options) {
      merged[opt.id] = { ...opt, provenance: 'tenant' }
    }
  }

  // draft wins per option (4th layer, only supplied inside the editor sandbox).
  const draftSet = draftOverlay?.[id]
  if (draftSet) {
    for (const opt of draftSet.options) {
      merged[opt.id] = { ...opt, provenance: 'draft' }
    }
  }

  // Determine set-level name + provenance (draft > tenant > vendor > default)
  const draftName = draftSet?.name
  const tenantName = tenantSet?.name
  const vendorName = vendorSet?.name
  const effectiveName = draftName ?? tenantName ?? vendorName ?? def.name
  const nameProvenance: ConfigProvenance = draftName
    ? 'draft'
    : tenantName
      ? 'tenant'
      : vendorName
        ? 'vendor'
        : 'default'

  return {
    id,
    name: effectiveName,
    provenance: nameProvenance,
    options: Object.values(merged).sort((a, b) => a.order - b.order),
  }
}

/** List all known value-set ids (default set is the source of truth for ids). */
export function listResolvedValueSets(base = false): ResolvedValueSet[] {
  return Object.keys(DEFAULT_VALUE_SETS)
    .map((id) => resolveValueSet(id, base))
    .filter((s): s is ResolvedValueSet => s !== null)
}

/** Upsert a value-set into the given layer (replaces options wholesale). */
export function upsertValueSet(layer: ConfigLayer, set: Omit<ValueSet, 'layer'>): void {
  const store = layer === 'vendor' ? vendorValueSets : tenantValueSets
  const oldSet = store[set.id]

  store[set.id] = { ...set, layer }

  writeAuditEvent({
    action: 'customization.valueset_updated',
    target: set.id,
    targetType: 'value_set',
    oldValue: oldSet
      ? { id: oldSet.id, name: oldSet.name, optionCount: oldSet.options.length }
      : undefined,
    newValue: { id: set.id, name: set.name, optionCount: set.options.length, layer },
  })
}

/** Update a single option within a value-set on the given layer. */
export function upsertValueSetOption(
  layer: ConfigLayer,
  setId: string,
  option: ValueSetOption,
): void {
  const store = layer === 'vendor' ? vendorValueSets : tenantValueSets
  const existing = store[setId]

  if (!existing) {
    // Bootstrap the set from defaults if no layer entry exists yet
    const def = DEFAULT_VALUE_SETS[setId]
    if (!def) return
    store[setId] = { ...def, layer, options: [...def.options] }
  }

  const set = store[setId]
  const idx = set.options.findIndex((o) => o.id === option.id)
  const oldOption = idx >= 0 ? set.options[idx] : undefined

  if (idx >= 0) {
    set.options[idx] = option
  } else {
    set.options.push(option)
  }

  writeAuditEvent({
    action: 'customization.valueset_updated',
    target: `${setId}.${option.id}`,
    targetType: 'value_set_option',
    oldValue: oldOption,
    newValue: { ...option, layer },
  })
}

// ── Aggregate helpers ─────────────────────────────────────────────────────────

/** Whether any customization (labels or value-sets) exists for this tenant. */
export function hasCustomization(): boolean {
  const anyLabels =
    Object.values(vendorLabels).some((m) => Object.keys(m).length > 0) ||
    Object.values(tenantLabels).some((m) => Object.keys(m).length > 0)
  const anyValueSets =
    Object.keys(vendorValueSets).length > 0 || Object.keys(tenantValueSets).length > 0
  return anyLabels || anyValueSets
}

/** Reset all customization data for a layer (rollback / "zurücksetzen"). */
export function clearAllCustomization(layer: ConfigLayer): void {
  if (layer === 'vendor') {
    for (const k of Object.keys(vendorLabels)) delete vendorLabels[k]
    for (const k of Object.keys(vendorValueSets)) delete vendorValueSets[k]
  } else {
    for (const k of Object.keys(tenantLabels)) delete tenantLabels[k]
    for (const k of Object.keys(tenantValueSets)) delete tenantValueSets[k]
  }
}

// ── Draft promotion (Modul-Editor v1) ─────────────────────────────────────────

/** A full clone of the tenant layer — used to roll back a deployed draft. */
export interface TenantSnapshot {
  labels: LocaleLabelMap
  valueSets: Record<string, ValueSet>
}

/** Snapshot the current tenant layer (call BEFORE promoting a draft, for rollback). */
export function snapshotTenant(): TenantSnapshot {
  return {
    labels: structuredClone(tenantLabels),
    valueSets: structuredClone(tenantValueSets),
  }
}

/** Restore a previously captured tenant snapshot (rollback). */
export function restoreTenant(snap: TenantSnapshot): void {
  for (const k of Object.keys(tenantLabels)) delete tenantLabels[k]
  Object.assign(tenantLabels, structuredClone(snap.labels))
  for (const k of Object.keys(tenantValueSets)) delete tenantValueSets[k]
  Object.assign(tenantValueSets, structuredClone(snap.valueSets))
}

/**
 * Merge a draft payload into the tenant layer (commit / scheduled-deploy
 * promotion). Sparse: only the payload's deviations are written. The caller
 * writes the audit event with the draft context. Returns applied counts for
 * the summary. Only LABEL_WHITELIST keys are honoured.
 */
export function applyDraftToTenant(payload: {
  labels: LocaleLabelMap
  valueSets: Record<string, Omit<ValueSet, 'layer'>>
}): { labelCount: number; valueSetCount: number } {
  let labelCount = 0
  for (const [locale, map] of Object.entries(payload.labels)) {
    tenantLabels[locale] ??= {}
    for (const [key, value] of Object.entries(map)) {
      if (!LABEL_WHITELIST.includes(key)) continue
      tenantLabels[locale][key] = value
      labelCount += 1
    }
  }

  let valueSetCount = 0
  for (const [id, set] of Object.entries(payload.valueSets)) {
    tenantValueSets[id] = { ...set, layer: 'tenant' }
    valueSetCount += 1
  }

  return { labelCount, valueSetCount }
}

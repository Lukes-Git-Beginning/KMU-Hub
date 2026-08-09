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
  ModuleAreaLayout,
  ModuleAreaMap,
  ModuleAreaSetting,
  ModuleAreaSettingMap,
  ModuleAreasOverlay,
  ResolvedLabel,
  ResolvedLabelMap,
  ResolvedValueSet,
  ResolvedValueSetOption,
  ValueSet,
  ValueSetMigrations,
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
  // List-column headings (Darien 2026-08-05). Renaming a column previewed fine but
  // was dropped on deploy, because this whitelist is the deploy filter — the rename
  // only ever lived in the sandbox bundle. Every key here is an `EditorModuleDef.
  // listColumns[].labelKey`; keep the two lists in step when a module gains columns.
  'helpdesk.table.ticketNr',
  'helpdesk.table.subject',
  'helpdesk.table.category',
  'helpdesk.table.priority',
  'helpdesk.table.status',
  'helpdesk.table.assignedTo',
  'helpdesk.table.sla',
  'helpdesk.table.createdAt',
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
  ticket_status: {
    id: 'ticket_status',
    name: 'Ticket-Status',
    options: [
      { id: 'open', label: 'Offen', color: 'hsl(38 92% 50%)', order: 0, active: true },
      { id: 'in_progress', label: 'In Bearbeitung', color: 'hsl(217 91% 60%)', order: 1, active: true },
      { id: 'waiting', label: 'Wartend', color: 'hsl(215 16% 47%)', order: 2, active: true },
      { id: 'resolved', label: 'Gelöst', color: 'hsl(142 71% 45%)', order: 3, active: true },
      { id: 'closed', label: 'Geschlossen', color: 'hsl(215 16% 47%)', order: 4, active: true },
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
  const vendorSet = vendorValueSets[id]
  const tenantSet = tenantValueSets[id]
  const draftSet = draftOverlay?.[id]

  // A set may have no code default and still exist — a tenant-created list, or a
  // brand-new draft-only list authored in the editor. Only bail if no layer has it.
  if (!def && !vendorSet && !tenantSet && !draftSet) return null

  if (base) {
    // base = the code default only; a set without one has no baseline.
    if (!def) return null
    return {
      id: def.id,
      name: def.name,
      provenance: 'default',
      options: def.options.map((o) => ({ ...o, provenance: 'default' as ConfigProvenance })),
    }
  }

  // Build a merged option map: default → vendor → tenant → draft
  const merged: Record<string, ResolvedValueSetOption> = {}

  if (def) {
    for (const opt of def.options) {
      merged[opt.id] = { ...opt, provenance: 'default' }
    }
  }

  if (vendorSet) {
    for (const opt of vendorSet.options) {
      merged[opt.id] = { ...opt, provenance: 'vendor' }
    }
  }

  if (tenantSet) {
    for (const opt of tenantSet.options) {
      merged[opt.id] = { ...opt, provenance: 'tenant' }
    }
  }

  // draft wins per option (4th layer, only supplied inside the editor sandbox).
  if (draftSet) {
    for (const opt of draftSet.options) {
      merged[opt.id] = { ...opt, provenance: 'draft' }
    }
  }

  // Determine set-level name + provenance (draft > tenant > vendor > default)
  const draftName = draftSet?.name
  const tenantName = tenantSet?.name
  const vendorName = vendorSet?.name
  const effectiveName = draftName ?? tenantName ?? vendorName ?? def?.name ?? id
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
    moduleKey: draftSet?.moduleKey ?? tenantSet?.moduleKey ?? vendorSet?.moduleKey,
  }
}

/**
 * Ids of the lists a module owns beyond its built-in ones — the lists someone
 * created in that module's editor and deployed. Without this the Wertelisten panel
 * only knew the registry's fixed ids plus whatever sat in the current draft, so a
 * self-created list disappeared from the panel as soon as it went live.
 */
export function listTenantValueSetsForModule(moduleKey: string): string[] {
  return Object.values(tenantValueSets)
    .filter((set) => set.moduleKey === moduleKey)
    .map((set) => set.id)
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

// ── R4 · Module-area visibility (tab/section on-off per tenant) ────────────────

// Sparse overlays: only areas explicitly turned OFF are stored. Empty = all on.
const vendorModuleAreas: ModuleAreasOverlay = {}
const tenantModuleAreas: ModuleAreasOverlay = {}

/**
 * Where records go whose value-set option was removed (R4b), per tenant:
 * setId → removedOptionId → targetOptionId.
 *
 * The editor stages this while previewing a removal; on deploy it lands here and
 * STAYS. That is what makes the promise the removal dialog gives ("Bestehende
 * Einträge werden geändert auf: X") true after "Übernehmen" too — until now the
 * preview remapped and the deployed module fell back to the removed value
 * (Darien 2026-08-06, "die Wertelisten gehen nicht zu 100 %"). The real record
 * UPDATE is the backend's job; this table is the frontend's source of truth for
 * what a stored value now means, and it is what the backend gets handed.
 */
const tenantValueSetMigrations: ValueSetMigrations = {}

/** Where records of this set have been moved to (empty when nothing was removed). */
export function resolveValueSetMigrations(setId: string): Record<string, string> {
  return tenantValueSetMigrations[setId] ?? {}
}

/**
 * Merge two area settings for the SAME key across layers. Objects merge field by
 * field so a draft that only moves a column does not wipe the width the tenant
 * layer already stored; a boolean coming from a higher layer means "just the
 * switch", so it folds into `visible` instead of replacing the layout.
 */
function mergeAreaSetting(base: ModuleAreaSetting | undefined, over: ModuleAreaSetting): ModuleAreaSetting {
  if (base === undefined) return over
  if (typeof base === 'boolean' && typeof over === 'boolean') return over
  const baseObj: ModuleAreaLayout = typeof base === 'boolean' ? { visible: base } : base
  const overObj: ModuleAreaLayout = typeof over === 'boolean' ? { visible: over } : over
  return { ...baseObj, ...overObj }
}

/** Layer-merge the raw settings of one module (vendor ⊕ tenant ⊕ draft). */
function mergeAreaLayers(moduleKey: string, draftOverlay?: ModuleAreasOverlay): ModuleAreaSettingMap {
  const out: ModuleAreaSettingMap = {}
  for (const layer of [vendorModuleAreas, tenantModuleAreas, draftOverlay ?? {}]) {
    for (const [key, setting] of Object.entries(layer[moduleKey] ?? {})) {
      out[key] = mergeAreaSetting(out[key], setting)
    }
  }
  return out
}

/**
 * Resolve which sub-areas of a module are enabled. Merged default(all-on) ⊕ vendor
 * ⊕ tenant ⊕ draft. Returns only the EXPLICIT settings — a consumer treats a
 * missing areaKey as enabled: `resolveModuleAreas(m)[area] !== false`.
 *
 * Always normalises to booleans, including the column-layout objects. That is what
 * keeps areas and statistics untouched by the layout extension: every existing
 * consumer keeps asking `!== false` and keeps getting the right answer.
 */
export function resolveModuleAreas(
  moduleKey: string,
  base = false,
  draftOverlay?: ModuleAreasOverlay,
): ModuleAreaMap {
  if (base) return {}
  const merged = mergeAreaLayers(moduleKey, draftOverlay)
  const out: ModuleAreaMap = {}
  for (const [key, setting] of Object.entries(merged)) {
    if (typeof setting === 'boolean') {
      out[key] = setting
      continue
    }
    // A layout-only entry (order/width, no `visible`) says NOTHING about
    // visibility, so it must not appear here at all: opt-in columns ask `=== true`
    // and would otherwise switch themselves on the moment the list is reordered.
    if (setting.visible !== undefined) out[key] = setting.visible
  }
  return out
}

/**
 * Resolve the LAYOUT half of the same settings (order/width per column) — the part
 * `resolveModuleAreas` flattens away. Only keys that actually carry order or width
 * appear, so a module can treat an empty result as "nothing configured".
 */
export function resolveModuleAreaLayout(
  moduleKey: string,
  base = false,
  draftOverlay?: ModuleAreasOverlay,
): Record<string, ModuleAreaLayout> {
  if (base) return {}
  const out: Record<string, ModuleAreaLayout> = {}
  for (const [key, setting] of Object.entries(mergeAreaLayers(moduleKey, draftOverlay))) {
    if (typeof setting === 'boolean') continue
    if (setting.order === undefined && setting.width === undefined) continue
    out[key] = setting
  }
  return out
}

// ── Aggregate helpers ─────────────────────────────────────────────────────────

/** Whether any customization (labels, value-sets or hidden areas) exists. */
export function hasCustomization(): boolean {
  const anyLabels =
    Object.values(vendorLabels).some((m) => Object.keys(m).length > 0) ||
    Object.values(tenantLabels).some((m) => Object.keys(m).length > 0)
  const anyValueSets =
    Object.keys(vendorValueSets).length > 0 || Object.keys(tenantValueSets).length > 0
  const anyAreas =
    Object.keys(vendorModuleAreas).length > 0 || Object.keys(tenantModuleAreas).length > 0
  return anyLabels || anyValueSets || anyAreas
}

/** Reset all customization data for a layer (rollback / "zurücksetzen"). */
export function clearAllCustomization(layer: ConfigLayer): void {
  if (layer === 'vendor') {
    for (const k of Object.keys(vendorLabels)) delete vendorLabels[k]
    for (const k of Object.keys(vendorValueSets)) delete vendorValueSets[k]
    for (const k of Object.keys(vendorModuleAreas)) delete vendorModuleAreas[k]
  } else {
    for (const k of Object.keys(tenantLabels)) delete tenantLabels[k]
    for (const k of Object.keys(tenantValueSets)) delete tenantValueSets[k]
    for (const k of Object.keys(tenantModuleAreas)) delete tenantModuleAreas[k]
    for (const k of Object.keys(tenantValueSetMigrations)) delete tenantValueSetMigrations[k]
  }
}

// ── Draft promotion (Modul-Editor v1) ─────────────────────────────────────────

/** A full clone of the tenant layer — used to roll back a deployed draft. */
export interface TenantSnapshot {
  labels: LocaleLabelMap
  valueSets: Record<string, ValueSet>
  moduleAreas: ModuleAreasOverlay
  /** Optional so snapshots taken before R4b still restore (they just had none). */
  valueSetMigrations?: ValueSetMigrations
}

/** Snapshot the current tenant layer (call BEFORE promoting a draft, for rollback). */
export function snapshotTenant(): TenantSnapshot {
  return {
    labels: structuredClone(tenantLabels),
    valueSets: structuredClone(tenantValueSets),
    moduleAreas: structuredClone(tenantModuleAreas),
    valueSetMigrations: structuredClone(tenantValueSetMigrations),
  }
}

/** Restore a previously captured tenant snapshot (rollback). */
export function restoreTenant(snap: TenantSnapshot): void {
  for (const k of Object.keys(tenantLabels)) delete tenantLabels[k]
  Object.assign(tenantLabels, structuredClone(snap.labels))
  for (const k of Object.keys(tenantValueSets)) delete tenantValueSets[k]
  Object.assign(tenantValueSets, structuredClone(snap.valueSets))
  for (const k of Object.keys(tenantModuleAreas)) delete tenantModuleAreas[k]
  Object.assign(tenantModuleAreas, structuredClone(snap.moduleAreas ?? {}))
  // Rolling back a removal has to bring the records back too: the option
  // reappears, so the redirect that pointed away from it must go.
  for (const k of Object.keys(tenantValueSetMigrations)) delete tenantValueSetMigrations[k]
  Object.assign(tenantValueSetMigrations, structuredClone(snap.valueSetMigrations ?? {}))
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
  moduleAreas?: ModuleAreasOverlay
  valueSetMigrations?: ValueSetMigrations
}): { labelCount: number; valueSetCount: number; areaCount: number } {
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

  // Removals: keep the redirect, so records keep landing where the removal dialog
  // promised. Chains are collapsed — if "Mittel → Hoch" is already live and this
  // draft removes "Hoch" in favour of "Dringend", then Mittel must go to Dringend
  // as well, otherwise it would point at an option that no longer exists.
  for (const [setId, moves] of Object.entries(payload.valueSetMigrations ?? {})) {
    const target = { ...(tenantValueSetMigrations[setId] ?? {}) }
    for (const [removed, to] of Object.entries(moves)) {
      target[removed] = to
      for (const [earlier, earlierTo] of Object.entries(target)) {
        if (earlierTo === removed) target[earlier] = to
      }
    }
    tenantValueSetMigrations[setId] = target
  }

  let areaCount = 0
  for (const [moduleKey, areaMap] of Object.entries(payload.moduleAreas ?? {})) {
    const target: ModuleAreaSettingMap = { ...(tenantModuleAreas[moduleKey] ?? {}) }
    // Per key, not per map: a draft that only sets a width must not drop the
    // visibility the tenant already carries (and vice versa).
    for (const [areaKey, setting] of Object.entries(areaMap)) {
      target[areaKey] = mergeAreaSetting(target[areaKey], setting)
    }
    tenantModuleAreas[moduleKey] = target
    areaCount += Object.keys(areaMap).length
  }

  return { labelCount, valueSetCount, areaCount }
}

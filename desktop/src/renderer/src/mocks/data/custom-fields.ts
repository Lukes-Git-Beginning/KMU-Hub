/**
 * Custom Fields data layer (v1.1) — unified, tenant-scoped custom field
 * definitions across 5 entities.
 *
 * This is the SEPARATE Custom-Fields persistence track (own BE tables:
 * work_custom_field_definitions + crm custom_field_definitions). It is NOT
 * part of the overlay/provenance system from customization.ts (which covers
 * Label-Overrides and Value-Sets). Custom fields are always tenant-only — no
 * vendor overlay layer here.
 *
 * 🔒 BE note for Luke: the real backend already has two tables:
 *   - work_custom_field_definitions (9 types, Migration 000146/147, RLS native)
 *   - custom_field_definitions (6 types, Migration 000005, CRM entities only)
 * The FE unification task (this file + v1.1 UI + handlers) is the client-side
 * contract. Luke's task = generalise the read-path to serve all 5 entities
 * through a single endpoint family, align type names (dropdown→select,
 * checkbox→boolean), and enforce RLS per RBAC key admin:customization:manage.
 * See backend-gaps §Customization.
 */
import { nanoid } from 'nanoid'
import { writeAuditEvent } from './audit-events'

// ── Types ─────────────────────────────────────────────────────────────────────

/**
 * The entities that support custom fields.
 *
 * `helpdesk_ticket` (added in E-3c, Modul-Editor v1) is an ADDITIVE FE-mock
 * entity so the Helpdesk pilot module has a real Felder panel — it does NOT
 * touch Luke's existing tables (work_custom_field_definitions /
 * custom_field_definitions). BE-side it is a future additive entity family,
 * documented in backend-gaps §Customization (nothing to migrate now).
 */
export type CustomFieldEntity =
  | 'work_task'
  | 'crm_contact'
  | 'crm_company'
  | 'crm_deal'
  | 'crm_activity'
  | 'helpdesk_ticket'

/**
 * The 9 canonical field types (aligned with BE work_custom_field_definitions).
 * NB: legacy FE used "dropdown" → normalise to "select", "checkbox" → "boolean".
 */
export type CustomFieldType =
  | 'text'
  | 'number'
  | 'date'
  | 'boolean'
  | 'select'
  | 'multi_select'
  | 'url'
  | 'email'
  | 'phone'

/** Validation rules (optional, progressive-disclosure tier). */
export interface CustomFieldValidation {
  /** Minimum value/length. */
  min?: number
  /** Maximum value/length. */
  max?: number
  /** Regex pattern (stored as string for JSON serialisation). */
  pattern?: string
}

/** A single custom field definition. */
export interface CustomFieldDefinition {
  id: string
  entity: CustomFieldEntity
  /** Stable programmatic key (snake_case, auto-generated, never changes). */
  key: string
  /** Human-readable label shown in the UI. */
  label: string
  type: CustomFieldType
  /** Whether this field is mandatory when creating/editing the entity. */
  required: boolean
  /** Options for select/multi_select types (ignored for other types). */
  options: string[]
  /** Progressive-disclosure tier: validation constraints. */
  validation?: CustomFieldValidation
  /** Progressive-disclosure tier: default value. */
  defaultValue?: string
  /** Whether this field is shown in list/detail views. */
  visible: boolean
  /** Display order within the entity (ascending). */
  order: number
  /**
   * Whether at least one entity record has a value for this field.
   * Used for soft-delete protection: deleting a field with inUse=true
   * requires explicit user confirmation (consequence dialog).
   */
  inUse: boolean
}

/** Input body for creating a new field. */
export interface CreateCustomFieldInput {
  entity: CustomFieldEntity
  label: string
  type: CustomFieldType
  required?: boolean
  options?: string[]
  validation?: CustomFieldValidation
  defaultValue?: string
  visible?: boolean
}

/** Input body for updating an existing field. */
export type UpdateCustomFieldInput = Partial<Omit<CreateCustomFieldInput, 'entity'>>

// ── Seed data ─────────────────────────────────────────────────────────────────

/** Initial custom field definitions across all 5 entities. */
const SEED_FIELDS: CustomFieldDefinition[] = [
  // ── work_task ────────────────────────────────────────────────────────────
  {
    id: 'cf_wt_001',
    entity: 'work_task',
    key: 'erp_ticket_ref',
    label: 'ERP-Ticketnummer',
    type: 'text',
    required: false,
    options: [],
    visible: true,
    order: 0,
    inUse: true,
  },
  {
    id: 'cf_wt_002',
    entity: 'work_task',
    key: 'estimated_hours',
    label: 'Geschätzter Aufwand (h)',
    type: 'number',
    required: false,
    options: [],
    validation: { min: 0, max: 999 },
    visible: true,
    order: 1,
    inUse: true,
  },
  {
    id: 'cf_wt_003',
    entity: 'work_task',
    key: 'task_category',
    label: 'Aufgabenkategorie',
    type: 'select',
    required: false,
    options: ['Design', 'Entwicklung', 'Review', 'Meeting', 'Sonstiges'],
    visible: true,
    order: 2,
    inUse: false,
  },

  // ── crm_contact ──────────────────────────────────────────────────────────
  {
    id: 'cf_cc_001',
    entity: 'crm_contact',
    key: 'erp_customer_number',
    label: 'Kundennummer ERP',
    type: 'text',
    required: false,
    options: [],
    visible: true,
    order: 0,
    inUse: true,
  },
  {
    id: 'cf_cc_002',
    entity: 'crm_contact',
    key: 'newsletter_consent',
    label: 'Newsletter-Einwilligung',
    type: 'boolean',
    required: false,
    options: [],
    defaultValue: 'false',
    visible: true,
    order: 1,
    inUse: true,
  },
  {
    id: 'cf_cc_003',
    entity: 'crm_contact',
    key: 'preferred_contact_method',
    label: 'Bevorzugter Kontaktweg',
    type: 'select',
    required: false,
    options: ['E-Mail', 'Telefon', 'WhatsApp', 'Post'],
    visible: true,
    order: 2,
    inUse: false,
  },

  // ── crm_company ──────────────────────────────────────────────────────────
  {
    id: 'cf_co_001',
    entity: 'crm_company',
    key: 'vat_id',
    label: 'USt-IdNr.',
    type: 'text',
    required: false,
    options: [],
    validation: { pattern: '^[A-Z]{2}[0-9A-Z]+$' },
    visible: true,
    order: 0,
    inUse: true,
  },
  {
    id: 'cf_co_002',
    entity: 'crm_company',
    key: 'employee_count_bracket',
    label: 'Mitarbeiterzahl (Klasse)',
    type: 'select',
    required: false,
    options: ['1–9', '10–49', '50–199', '200+'],
    visible: true,
    order: 1,
    inUse: false,
  },

  // ── crm_deal ─────────────────────────────────────────────────────────────
  {
    id: 'cf_d_001',
    entity: 'crm_deal',
    key: 'deal_priority',
    label: 'Priorität',
    type: 'select',
    required: false,
    options: ['Niedrig', 'Mittel', 'Hoch', 'Kritisch'],
    visible: true,
    order: 0,
    inUse: true,
  },
  {
    id: 'cf_d_002',
    entity: 'crm_deal',
    key: 'deal_source',
    label: 'Quelle',
    type: 'select',
    required: false,
    options: ['Empfehlung', 'Kalt-Akquise', 'Website', 'Messe', 'Sonstiges'],
    visible: true,
    order: 1,
    inUse: true,
  },
  {
    id: 'cf_d_003',
    entity: 'crm_deal',
    key: 'expected_close_date',
    label: 'Erwartetes Abschlussdatum',
    type: 'date',
    required: false,
    options: [],
    visible: true,
    order: 2,
    inUse: false,
  },

  // ── crm_activity ─────────────────────────────────────────────────────────
  {
    id: 'cf_a_001',
    entity: 'crm_activity',
    key: 'call_recording_url',
    label: 'Gesprächsaufzeichnung (URL)',
    type: 'url',
    required: false,
    options: [],
    visible: false,
    order: 0,
    inUse: false,
  },
  {
    id: 'cf_a_002',
    entity: 'crm_activity',
    key: 'follow_up_tags',
    label: 'Follow-up-Tags',
    type: 'multi_select',
    required: false,
    options: ['Angebot senden', 'Demo einplanen', 'Nachfassen', 'Eskalieren'],
    visible: true,
    order: 1,
    inUse: false,
  },

  // ── helpdesk_ticket (E-3c pilot; additive FE-mock, see type note) ──────────
  {
    id: 'cf_ht_001',
    entity: 'helpdesk_ticket',
    key: 'sla_tier',
    label: 'SLA-Stufe',
    type: 'select',
    required: false,
    options: ['Standard', 'Priorität', 'Kritisch'],
    visible: true,
    order: 0,
    inUse: true,
  },
  {
    id: 'cf_ht_002',
    entity: 'helpdesk_ticket',
    key: 'escalation_reason',
    label: 'Eskalationsgrund',
    type: 'text',
    required: false,
    options: [],
    visible: true,
    order: 1,
    inUse: false,
  },
  {
    id: 'cf_ht_003',
    entity: 'helpdesk_ticket',
    key: 'contact_channel',
    label: 'Kontaktkanal',
    type: 'select',
    required: false,
    options: ['E-Mail', 'Telefon', 'Chat', 'Vor Ort'],
    visible: true,
    order: 2,
    inUse: false,
  },
]

// ── Runtime state (mutable) ───────────────────────────────────────────────────

/** In-memory store: deep-cloned from seeds so tests can reset cleanly. */
let store: CustomFieldDefinition[] = structuredClone(SEED_FIELDS)

// ── Helpers ───────────────────────────────────────────────────────────────────

function toKey(label: string): string {
  return label
    .toLowerCase()
    .replace(/[äöüß]/g, (c) => ({ ä: 'ae', ö: 'oe', ü: 'ue', ß: 'ss' }[c] ?? c))
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_|_$/g, '')
    .slice(0, 48)
}

// ── CRUD ──────────────────────────────────────────────────────────────────────

/** List all custom field definitions, optionally filtered by entity. */
export function listCustomFields(entity?: CustomFieldEntity): CustomFieldDefinition[] {
  const fields = entity ? store.filter((f) => f.entity === entity) : store
  return fields.slice().sort((a, b) => a.order - b.order)
}

/** Get a single field definition by id. */
export function getCustomField(id: string): CustomFieldDefinition | undefined {
  return store.find((f) => f.id === id)
}

/** Create a new custom field definition. */
export function createCustomField(input: CreateCustomFieldInput): CustomFieldDefinition {
  const existingForEntity = store.filter((f) => f.entity === input.entity)
  const maxOrder = existingForEntity.length > 0
    ? Math.max(...existingForEntity.map((f) => f.order))
    : -1

  const field: CustomFieldDefinition = {
    id: `cf_${nanoid(8)}`,
    entity: input.entity,
    key: toKey(input.label),
    label: input.label,
    type: input.type,
    required: input.required ?? false,
    options: input.options ?? [],
    validation: input.validation,
    defaultValue: input.defaultValue,
    visible: input.visible ?? true,
    order: maxOrder + 1,
    inUse: false,
  }

  store.push(field)

  writeAuditEvent({
    action: 'customization.field_created',
    target: field.label,
    targetType: 'custom_field',
    newValue: { id: field.id, entity: field.entity, key: field.key, type: field.type },
  })

  return field
}

/** Update an existing custom field definition. */
export function updateCustomField(
  id: string,
  input: UpdateCustomFieldInput,
): CustomFieldDefinition | null {
  const idx = store.findIndex((f) => f.id === id)
  if (idx < 0) return null

  const old = store[idx]
  const updated: CustomFieldDefinition = {
    ...old,
    ...(input.label !== undefined && { label: input.label }),
    ...(input.type !== undefined && { type: input.type }),
    ...(input.required !== undefined && { required: input.required }),
    ...(input.options !== undefined && { options: input.options }),
    ...(input.validation !== undefined && { validation: input.validation }),
    ...(input.defaultValue !== undefined && { defaultValue: input.defaultValue }),
    ...(input.visible !== undefined && { visible: input.visible }),
  }

  store[idx] = updated

  writeAuditEvent({
    action: 'customization.field_updated',
    target: updated.label,
    targetType: 'custom_field',
    oldValue: { label: old.label, type: old.type, required: old.required },
    newValue: { label: updated.label, type: updated.type, required: updated.required },
  })

  return updated
}

/**
 * Delete a custom field definition.
 * When the field has inUse=true the caller must have shown the consequence
 * dialog first — this function deletes unconditionally (dialog is enforced
 * in the UI layer, not here).
 */
export function deleteCustomField(id: string): boolean {
  const field = store.find((f) => f.id === id)
  if (!field) return false

  store = store.filter((f) => f.id !== id)

  writeAuditEvent({
    action: 'customization.field_deleted',
    target: field.label,
    targetType: 'custom_field',
    oldValue: { id: field.id, entity: field.entity, key: field.key, inUse: field.inUse },
  })

  return true
}

/** Reorder fields within an entity by providing the new ordered id list. */
export function reorderCustomFields(entity: CustomFieldEntity, orderedIds: string[]): void {
  orderedIds.forEach((id, idx) => {
    const field = store.find((f) => f.id === id && f.entity === entity)
    if (field) field.order = idx
  })
}

/** Reset the store to seed data (used by QA scripts). */
export function resetCustomFieldsStore(): void {
  store = structuredClone(SEED_FIELDS)
}

// ── Draft promotion (Modul-Editor v1, E-3c) ────────────────────────────────────
//
// Custom fields live in their own store (own BE persistence), OUTSIDE the tenant
// overlay in customization.ts. So the editor stages field changes as a per-entity
// SNAPSHOT (the desired full field list for a touched entity) and, on deploy,
// diffs the snapshot against the live store — the same mental model as value-sets
// (draft holds the full set), just against a CRUD store instead of an overlay map.

/** A per-entity map of the desired full field list (only touched entities present). */
export type DraftCustomFieldMap = Record<string, CustomFieldDefinition[]>

/** Whether a persisted field (real store id) differs from its desired snapshot. */
function fieldChanged(a: CustomFieldDefinition, b: CustomFieldDefinition): boolean {
  return (
    a.label !== b.label ||
    a.type !== b.type ||
    a.required !== b.required ||
    a.visible !== b.visible ||
    a.order !== b.order ||
    JSON.stringify(a.options) !== JSON.stringify(b.options) ||
    JSON.stringify(a.validation ?? null) !== JSON.stringify(b.validation ?? null) ||
    (a.defaultValue ?? '') !== (b.defaultValue ?? '')
  )
}

/**
 * Apply a draft's field snapshot to the live store by diffing per entity:
 *   - desired field whose id is NOT in the store  → create (draft/temp id dropped)
 *   - desired field present but changed           → update
 *   - store field for the entity absent from desired → delete
 * Returns the number of field-level changes applied (for the deploy summary).
 */
export function applyDraftCustomFields(fieldsByEntity: DraftCustomFieldMap): number {
  let fieldCount = 0
  for (const [entity, desired] of Object.entries(fieldsByEntity)) {
    const current = store.filter((f) => f.entity === (entity as CustomFieldEntity))
    const desiredIds = new Set(desired.map((f) => f.id))

    // Creates + updates
    desired.forEach((field, idx) => {
      const existing = store.find((f) => f.id === field.id)
      if (!existing) {
        const created = createCustomField({
          entity: entity as CustomFieldEntity,
          label: field.label,
          type: field.type,
          required: field.required,
          options: field.options,
          validation: field.validation,
          defaultValue: field.defaultValue,
          visible: field.visible,
        })
        // Preserve the draft-intended order.
        const created2 = store.find((f) => f.id === created.id)
        if (created2) created2.order = idx
        fieldCount += 1
      } else if (fieldChanged(existing, { ...field, order: idx })) {
        updateCustomField(field.id, {
          label: field.label,
          type: field.type,
          required: field.required,
          options: field.options,
          validation: field.validation,
          defaultValue: field.defaultValue,
          visible: field.visible,
        })
        existing.order = idx
        fieldCount += 1
      }
    })

    // Deletes (baseline field removed in the draft)
    for (const f of current) {
      if (!desiredIds.has(f.id)) {
        deleteCustomField(f.id)
        fieldCount += 1
      }
    }
  }
  return fieldCount
}

/** Snapshot the whole field store (call BEFORE promoting a draft, for rollback). */
export function snapshotCustomFields(): CustomFieldDefinition[] {
  return structuredClone(store)
}

/** Restore a previously captured field snapshot (rollback). */
export function restoreCustomFields(snap: CustomFieldDefinition[]): void {
  store = structuredClone(snap)
}

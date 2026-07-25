/**
 * Customization — app-owned API contract types (v1.0 Fundament).
 *
 * Overlay-Prinzip: Code liefert den Default. Config speichert nur Abweichungen,
 * in zwei unabhängigen Schichten. Die Resolution pro Schlüssel lautet:
 *
 *   effektiv = code_default  ⊕  vendor_overlay (Zentria)  ⊕  tenant_overlay (Kunde)
 *                                                              ↑ gewinnt per key
 *
 * Provenance wird IMMER mitgeführt, damit die UI die Herkunft anzeigt
 * ("von Zentria eingerichtet" vs. "selbst geändert") — identisch zum
 * CapabilityGrant.sources-Muster aus R-6.
 *
 * FE consumes these shapes; the MSW mock conforms today, the real backend
 * (Luke's track 🔒, backend-gaps §Customization) conforms tomorrow.
 *
 * Custom Fields (Dimension A) are excluded from v1.0 — they have their own
 * BE persistence (work_custom_field_definitions / crm custom_field_definitions)
 * and will be unified in v1.1. This file covers only:
 *   B · Label-Overrides (Terminologie / Whitelist-Keys)
 *   M · Value-Sets      (Wertelisten / Status-Sets)
 *
 * Custom Fields (Dimension A) join the draft payload in E-3c via DraftCustomFieldMap
 * — they keep their own store (mocks/data/custom-fields), the draft only carries a
 * per-entity snapshot for deploy.
 */
import type { DraftCustomFieldMap } from '@/mocks/data/custom-fields'

// ── Overlay layer ────────────────────────────────────────────────────────────

/**
 * The two overlay layers that sit above the code default.
 * `vendor` = Zentria writes during onboarding (via the GDAP vendor-access from R-5).
 * `tenant` = the customer's own changes.
 * Resolution order: default < vendor < tenant.
 */
export type ConfigLayer = 'vendor' | 'tenant'

/**
 * Where a resolved value originates. Four strata, low → high priority:
 *   'default' — the code-bundled fallback (nothing stored in overlay).
 *   'vendor'  — Zentria set it during onboarding; the customer has not overridden it.
 *   'tenant'  — the customer set it (live for all users).
 *   'draft'   — an in-editor change, only visible inside the editor sandbox until
 *               it is committed/deployed into the tenant layer. Always wins.
 */
export type ConfigProvenance = 'default' | 'vendor' | 'tenant' | 'draft'

// ── B · Label-Overrides (Terminologie) ───────────────────────────────────────

/**
 * A locale-scoped map from i18n key → override text.
 * Sparse: only keys that deviate from the code default are present.
 *
 * Runtime merge uses i18next `addResourceBundle(locale, 'translation',
 * overrides, true, true)` so the override wins in all t() calls
 * without a rebuild. Only LABEL_WHITELIST keys may appear here.
 */
export type LabelOverrideMap = Record<string, string>

/** A locale-scoped overlay map: locale → (i18n key → override text). Sparse. */
export type LocaleLabelMap = Record<string, LabelOverrideMap>

/**
 * A label override resolved to its effective value plus where it came from.
 * `value` is the text to render; `provenance` drives the "von Zentria
 * eingerichtet" / "selbst geändert" / "(Standard)" badge in the editor.
 */
export interface ResolvedLabel {
  /** The effective text (vendor override, or tenant override, or the default). */
  value: string
  provenance: ConfigProvenance
  /** The raw i18n key from LABEL_WHITELIST. */
  key: string
}

/**
 * Full resolved label map returned by GET /customization/labels.
 * One entry per LABEL_WHITELIST key.
 */
export type ResolvedLabelMap = Record<string, ResolvedLabel>

// ── M · Value-Sets (Wertelisten / Status-Sets) ───────────────────────────────

/**
 * One option within a named value-set (e.g. a Deal pipeline stage).
 * `color` is an optional HSL/hex token for status dots.
 * `active = false` = soft-deleted: hidden in pickers but retained for
 * existing records (prevents data loss on Airtable-style hard-delete).
 */
export interface ValueSetOption {
  /** Stable id — never changes after creation (safe for DB FK references). */
  id: string
  label: string
  /** Optional HSL/hex color token for status dots / pipeline columns. */
  color?: string
  /** Display order (ascending). */
  order: number
  /** Soft-delete: false hides from pickers, keeps existing records intact. */
  active: boolean
}

/**
 * A named list of options plus the layer it was defined in.
 * `id` is a well-known stable slug (e.g. "deal_stages", "ticket_priority").
 */
export interface ValueSet {
  id: string
  /** Human-readable name shown in the editor (DE default). */
  name: string
  options: ValueSetOption[]
  /** Which layer owns this definition (vendor | tenant). */
  layer: ConfigLayer
}

/**
 * A value-set resolved across all layers, with per-option provenance.
 * Options from a lower layer that a higher layer did not touch keep
 * their original provenance, while modified/added entries carry the
 * winning layer's provenance. The merged list is order-sorted.
 */
export interface ResolvedValueSetOption extends ValueSetOption {
  provenance: ConfigProvenance
}

export interface ResolvedValueSet {
  id: string
  name: string
  /** Effective options after overlay merge, sorted by `order`. */
  options: ResolvedValueSetOption[]
  /** Provenance of the value-set definition itself. */
  provenance: ConfigProvenance
}

// ── Module-area visibility (R4: tab/section on-off per tenant) ─────────────────

/**
 * Which sub-areas (tabs/sections) of ONE module are enabled for the tenant.
 * Sparse: only areas explicitly turned OFF are stored (absent = enabled). This
 * lets updates ship new areas that are on by default while a tenant can hide the
 * ones it does not use (e.g. a helpdesk without a knowledge base).
 *   areaKey → enabled
 */
export type ModuleAreaMap = Record<string, boolean>

/** Module-area overlay across modules: moduleKey → (areaKey → enabled). */
export type ModuleAreasOverlay = Record<string, ModuleAreaMap>

// ── API input/output shapes ───────────────────────────────────────────────────

/** Body for PUT /customization/labels — replaces/adds keys for one locale. */
export interface UpdateLabelOverridesInput {
  locale: string
  layer: ConfigLayer
  /** Partial map — only the keys to set/update. */
  overrides: LabelOverrideMap
}

/** Body for PUT /customization/value-sets/:id — replaces the options of one set. */
export interface UpsertValueSetInput {
  layer: ConfigLayer
  name?: string
  options: ValueSetOption[]
}

/** Response envelope for GET /customization/labels */
export interface LabelOverridesResponse {
  locale: string
  labels: ResolvedLabelMap
}

/** Response envelope for GET /customization/value-sets */
export interface ValueSetsResponse {
  valueSets: ResolvedValueSet[]
}

/** Response envelope for GET /customization/value-sets/:id */
export interface ValueSetResponse {
  valueSet: ResolvedValueSet
}

// ── Draft & Deployment (Modul-Editor v1) ──────────────────────────────────────

/**
 * The sparse content of an editor draft — only the deviations, mirroring the
 * overlay principle. Applied as the 4th layer on top of the tenant layer while
 * the editor is open, and merged into the tenant layer on commit/deploy.
 */
export interface CustomizationDraftPayload {
  /** Label overrides: locale → key → value. */
  labels: LocaleLabelMap
  /** Value-set overrides by set id (layer omitted — promotes to tenant on deploy). */
  valueSets: Record<string, Omit<ValueSet, 'layer'>>
  /**
   * Custom-field snapshots by entity (E-3c): the desired full field list for a
   * touched entity. Optional — omitted when a draft has no field changes, so
   * existing draft producers stay valid. Fields live outside the tenant overlay
   * (own store), so deploy diffs this snapshot against the live field store.
   */
  customFields?: DraftCustomFieldMap
  /**
   * Module-area visibility overrides (R4): moduleKey → areaKey → enabled. Sparse;
   * omitted when a draft toggles no areas, so existing draft producers stay valid.
   */
  moduleAreas?: ModuleAreasOverlay
}

/**
 * Lifecycle of a saved editor draft (Change-Management für Config).
 *   'draft'      — work in progress, affects no user.
 *   'scheduled'  — approved for a future tenant-wide rollout at `scheduledAt`.
 *   'live'       — promoted into the tenant layer (its changes are effective).
 *   'superseded' — replaced by a newer live version (retained for rollback).
 */
export type DraftStatus = 'draft' | 'scheduled' | 'live' | 'superseded'

/** A persisted editor draft (saved blueprint or scheduled rollout). */
export interface CustomizationDraft {
  id: string
  /** Which module this draft targets (v1 pilot: 'kontakte' | 'helpdesk'). */
  moduleKey: string
  /** Human name for the draft list (e.g. "Praxis-Umbau Q3"). */
  name: string
  status: DraftStatus
  /** ISO timestamp for a scheduled rollout; set only when status='scheduled'. */
  scheduledAt?: string
  payload: CustomizationDraftPayload
  createdAt: string
  updatedAt: string
  /** Actor id who created the draft. */
  createdBy: string
}

/** How the editor commit dialog resolves the current draft. */
export type DeployMode = 'now' | 'scheduled' | 'draft'

/** Body for POST /customization/drafts (save-as-draft). */
export interface SaveDraftInput {
  /** Present when updating an existing draft. */
  id?: string
  moduleKey: string
  name: string
  payload: CustomizationDraftPayload
}

/** Body for POST /customization/drafts/deploy — apply now, schedule, or keep as draft. */
export interface DeployDraftInput {
  moduleKey: string
  name: string
  payload: CustomizationDraftPayload
  mode: DeployMode
  /** Required when mode='scheduled' (ISO date-time). */
  scheduledAt?: string
  /** Optional in-app announcement banner text for affected users. */
  announcement?: string
}

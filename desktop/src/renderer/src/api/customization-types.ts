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
 */

// ── Overlay layer ────────────────────────────────────────────────────────────

/**
 * The two overlay layers that sit above the code default.
 * `vendor` = Zentria writes during onboarding (via the GDAP vendor-access from R-5).
 * `tenant` = the customer's own changes.
 * Resolution order: default < vendor < tenant.
 */
export type ConfigLayer = 'vendor' | 'tenant'

/**
 * Where a resolved value originates. Three values matching the three strata:
 *   'default' — the code-bundled fallback (nothing stored in overlay).
 *   'vendor'  — Zentria set it during onboarding; the customer has not overridden it.
 *   'tenant'  — the customer set it (always wins).
 */
export type ConfigProvenance = 'default' | 'vendor' | 'tenant'

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

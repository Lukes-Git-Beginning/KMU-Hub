/**
 * Shared intake engine — types.
 *
 * The intake engine turns a *form submission* (a `Record<fieldId, value>`) into a
 * *module-specific record* (first consumer: a Helpdesk ticket). It is the common
 * substrate for every Cosmi surface where an editor-built form feeds a domain
 * object — analogous to `shared/document` for block authoring.
 *
 * A form field carries an optional **role** (subject/description/priority/…). The
 * engine reads each field's role, collects the role → answer mapping, drops every
 * unmarked field into an `extras` bucket (→ the record's free custom fields) and
 * hands both to the target's `build()` to produce the create-input.
 *
 * Modules stay decoupled: `shared/intake` never imports a module. Each module
 * *registers* its target (id + roles + build/create) via `registerIntakeTarget`.
 */

/**
 * A field role is an open string keyed per target (a Helpdesk target knows
 * `subject`, an HR-request target might know `applicant`, …). The empty/absent
 * role means the field is an unmarked *extra* → lands in `custom_fields`.
 */
export type IntakeFieldRole = string

/** The reserved role value for "no role assigned → extra/custom field". */
export const INTAKE_ROLE_EXTRA = 'extra' as const

/** Minimal shape the engine needs from a form field (decoupled from FormField). */
export interface IntakeField {
  id: string
  label: string
  role?: IntakeFieldRole
}

/** A role a target exposes, shown as an option in the field-role dropdown. */
export interface IntakeRoleDef {
  /** Role key stored on the field (e.g. `subject`). */
  role: IntakeFieldRole
  /** i18n key for the human label (e.g. `intake.role.subject`). */
  labelKey: string
  /**
   * Optional whitelist of form field types this role accepts. When set, the
   * builder only offers the role for those field types (e.g. `priority` → select).
   * Omit to allow any field type.
   */
  fieldTypes?: string[]
}

/**
 * Result of mapping a submission: the target it belongs to, the built record and
 * the raw role/extras split (kept for debugging / QA assertions).
 */
export interface IntakeMappingResult<TRecord = unknown> {
  targetId: string
  record: TRecord
  mapped: Record<IntakeFieldRole, unknown>
  extras: Record<string, string | number | boolean>
}

/** Context passed to a target's `build()` — channel origin, submitter identity. */
export interface IntakeBuildContext {
  /** Where the submission came from: agent / selfservice (intern) / external. */
  channel?: string
  /** Resolved requester when the channel supplies it (e.g. logged-in user). */
  requester?: { id?: string; name?: string; email?: string; isExternal?: boolean }
}

/**
 * A module-registered intake target. `TRecord` is the module's create-input
 * (e.g. `CreateTicketInput`). Generic on purpose so the registry stays agnostic.
 */
export interface IntakeTargetDef<TRecord = unknown> {
  /** Stable id stored on the form (`FormSchema.intakeTargetId`). */
  id: string
  /** i18n key for the target's display name (e.g. `intake.target.helpdeskTicket`). */
  labelKey: string
  /** Roles this target understands, in display order. */
  roles: IntakeRoleDef[]
  /**
   * Build the module create-input from the collected role values + the extras
   * bucket. Pure — no side effects, no network.
   */
  build: (
    mapped: Record<IntakeFieldRole, unknown>,
    extras: Record<string, string | number | boolean>,
    context: IntakeBuildContext,
  ) => TRecord
  /** Persist the record (module client call). Returns at least the created id. */
  create: (record: TRecord) => Promise<{ id: string }>
  /**
   * React-Query keys to invalidate after a successful create so the module list
   * refreshes (e.g. `[['helpdesk', 'tickets']]`).
   */
  invalidateKeys?: readonly unknown[][]
}

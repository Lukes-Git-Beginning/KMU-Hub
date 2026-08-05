/**
 * Shared intake engine — the pure mapping core.
 *
 * `mapSubmissionToRecord` takes a target id, the form's fields (with roles) and a
 * submission's `answers` (keyed by field id) and produces the module create-input
 * via the registered target's `build()`. Fields with a known role feed the
 * `mapped` bucket; every unmarked field lands in `extras` (→ custom fields).
 *
 * No React, no network — safe to unit-test and to call from any channel.
 */
import { getIntakeTarget } from './registry'
import {
  INTAKE_ROLE_EXTRA,
  type IntakeBuildContext,
  type IntakeField,
  type IntakeMappingResult,
} from './types'

/** Page-break marker used by the form builder — never a real answer field. */
const PAGE_BREAK_LABEL = '__page_break__'

/**
 * Slugify a field label into a stable, human-readable custom-field key
 * (`"Bestell-Nr."` → `bestell_nr`). Falls back to the field id when a label
 * slugifies to nothing (e.g. all punctuation).
 */
export function slugifyKey(label: string, fallback: string): string {
  const slug = label
    .trim()
    .toLowerCase()
    .replace(/[äöü]/g, (c) => ({ ä: 'ae', ö: 'oe', ü: 'ue' })[c] ?? c)
    .replace(/ß/g, 'ss')
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '')
  return slug || fallback
}

/** Coerce an arbitrary answer value into a custom-field-safe primitive. */
function toPrimitive(value: unknown): string | number | boolean {
  if (typeof value === 'boolean' || typeof value === 'number') return value
  if (value == null) return ''
  return String(value)
}

/**
 * Map a submission onto a module record.
 *
 * @throws when no target is registered for `targetId` — callers should guard
 *         with `getIntakeTarget` (the builder only offers registered targets).
 */
export function mapSubmissionToRecord<TRecord = unknown>(
  targetId: string,
  fields: IntakeField[],
  answers: Record<string, unknown>,
  context: IntakeBuildContext = {},
): IntakeMappingResult<TRecord> {
  const target = getIntakeTarget(targetId)
  if (!target) {
    throw new Error(`intake: no target registered for "${targetId}"`)
  }
  const knownRoles = new Set(target.roles.map((r) => r.role))

  const mapped: Record<string, unknown> = {}
  const extras: Record<string, string | number | boolean> = {}

  for (const field of fields) {
    if (field.label === PAGE_BREAK_LABEL) continue
    const answer = answers[field.id]
    const isRole = field.role && field.role !== INTAKE_ROLE_EXTRA && knownRoles.has(field.role)
    if (isRole) {
      mapped[field.role as string] = answer
    } else {
      // Unmarked → extra. Empty answers are dropped so custom_fields stays lean.
      const primitive = toPrimitive(answer)
      if (primitive !== '') {
        extras[slugifyKey(field.label, field.id)] = primitive
      }
    }
  }

  const record = target.build(mapped, extras, context) as TRecord
  return { targetId, record, mapped, extras }
}

/**
 * Shared intake engine — target registry.
 *
 * Modules register their intake targets here (side-effect on import). The engine
 * and the form builder read from the registry; neither imports a module, so the
 * dependency arrow only ever points module → shared.
 */
import type { IntakeRoleDef, IntakeTargetDef } from './types'

const REGISTRY = new Map<string, IntakeTargetDef<unknown>>()

/** Register (or replace) a module's intake target. Idempotent by id. */
export function registerIntakeTarget<TRecord>(def: IntakeTargetDef<TRecord>): void {
  REGISTRY.set(def.id, def as IntakeTargetDef<unknown>)
}

/** Look up a target by id, or `undefined` when nothing is registered for it. */
export function getIntakeTarget(id: string | undefined | null): IntakeTargetDef<unknown> | undefined {
  if (!id) return undefined
  return REGISTRY.get(id)
}

/** All registered targets, for the "what does this form create?" picker. */
export function listIntakeTargets(): IntakeTargetDef<unknown>[] {
  return [...REGISTRY.values()]
}

/** The roles a target exposes, or an empty list when the target is unknown. */
export function getIntakeRoles(id: string | undefined | null): IntakeRoleDef[] {
  return getIntakeTarget(id)?.roles ?? []
}

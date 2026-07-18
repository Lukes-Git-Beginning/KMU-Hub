/**
 * Role identity (RBAC R-1).
 *
 * The 7 system preset roles (KONZEPT §3 Rollen-Modell). Old 5-role set
 * migrated 2026-07: hr → hr_admin, it_support → it_admin; readonly + extern
 * are new. Roles are DATA now — visibility/actions resolve through the
 * effective capability map (stores/permissions.ts + hooks/useCapability.ts),
 * not through hardcoded role lists. Preset definitions + grants live in
 * mocks/data/rbac.ts until Luke's role CRUD lands (backend-gaps §RBAC).
 *
 * Custom tenant roles (R-2) will carry ids outside this union — RoleId covers
 * the system presets only; generic code should treat role ids as string.
 */

export type RoleId =
  | 'admin'
  | 'it_admin'
  | 'hr_admin'
  | 'manager'
  | 'member'
  | 'readonly'
  | 'extern'

/** i18n label key for a system preset; custom roles (R-2) render their own name. */
export function roleLabelKey(roleId: string): string {
  return `rbac.roles.${roleId}.label`
}

/** i18n description key for a system preset. */
export function roleDescriptionKey(roleId: string): string {
  return `rbac.roles.${roleId}.description`
}

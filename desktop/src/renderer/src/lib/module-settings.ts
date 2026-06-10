import type { ModuleId } from './pricing'

/**
 * Settings scope — the 3-level hierarchy of the Settings-Fundament:
 *
 *   tenant-default → Modul-Leiter override (tenant-wide) → user override (personal)
 *
 * - 'personal' settings are always editable by the user (comfort: default view,
 *   sort order, density …). They never affect anyone else.
 * - 'tenant' settings apply to EVERYONE in the tenant. Only a Modul-Leiter (or an
 *   admin) may change them; non-leads see them read-only behind a lock.
 *
 * Whether the current user may edit a tenant-scoped section is resolved by
 * `useIsModuleLead(moduleId)`.
 */
export type SettingsScope = 'personal' | 'tenant'

/**
 * Modules that have a settings panel. Superset of ModuleId: 'dashboard' is a
 * cross-cutting surface (not a priced/leadable module) but still exposes
 * tenant-wide settings — editable by admins only, since it cannot be
 * delegated via LEADABLE_MODULES.
 */
export type SettingsModuleId = ModuleId | 'dashboard'

/**
 * Modules that expose a Modul-Leiter role (i.e. have meaningful tenant-wide
 * settings an admin can delegate). Used by the Team module's per-employee
 * "erweiterte Moduleinstellungen" toggle. Keep in sync with ModuleId.
 */
export const LEADABLE_MODULES: ModuleId[] = [
  'crm',
  'finance',
  'team',
  'dialer',
  'meetings',
  'mail',
  'calendar',
  'documents',
  'helpdesk',
  'inventar',
  'einkauf',
  'produktion',
  'projects',
  'vertraege',
  'schichten',
  'rapporte',
  'fuhrpark',
  'vermietung',
  'zeiterfassung',
  'wiki',
  'berichte',
  'formulare',
]

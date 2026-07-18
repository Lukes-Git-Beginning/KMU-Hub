/**
 * RBAC permission catalogue + default grant matrix (A-2 — LEGACY).
 *
 * ⚠ Superseded by the RBAC R-block: roles are the 7 presets from
 * mocks/data/rbac.ts (custom roles come with the R-2 role builder, which
 * replaces this matrix UI entirely). Kept functional until R-2 lands.
 * readonly/extern hold nothing here (default-deny); admin implicitly holds
 * every capability and is locked in the UI + server.
 */
import type { RoleId } from '@/config/roles'
import type { PermissionGroup, PermissionMatrix } from '@/api/admin-types'

/** Capability groups → ordered capabilities. Labels are i18n'd by id in the UI. */
export const PERMISSION_GROUPS: PermissionGroup[] = [
  { id: 'access', capabilities: ['users.view', 'users.manage', 'roles.manage'] },
  { id: 'crm', capabilities: ['crm.view', 'crm.edit', 'crm.export'] },
  { id: 'projects', capabilities: ['projects.view', 'projects.edit', 'projects.manage'] },
  { id: 'finance', capabilities: ['finance.view', 'finance.manage'] },
  { id: 'team', capabilities: ['team.view', 'team.manage'] },
  { id: 'security', capabilities: ['security.view', 'security.manage'] },
  { id: 'settings', capabilities: ['settings.view', 'settings.manage'] },
]

/** Capabilities whose grant is a write/elevated action — flagged subtly in the UI. */
export const ELEVATED_CAPABILITIES = new Set([
  'users.manage', 'roles.manage', 'crm.export', 'projects.manage',
  'finance.manage', 'team.manage', 'security.manage', 'settings.manage',
])

// Default grants per capability. Admin is omitted here (always true, locked).
const T = (...roles: RoleId[]) => {
  const row: Partial<Record<RoleId, boolean>> = { admin: true }
  for (const r of roles) row[r] = true
  return row
}

export const seedPermissionMatrix = (): PermissionMatrix => ({
  'users.view': T('it_admin', 'manager', 'hr_admin'),
  'users.manage': T('it_admin'),
  'roles.manage': T('it_admin'),
  'crm.view': T('it_admin', 'manager', 'hr_admin', 'member'),
  'crm.edit': T('manager', 'member'),
  'crm.export': T('manager'),
  'projects.view': T('it_admin', 'manager', 'hr_admin', 'member'),
  'projects.edit': T('manager', 'member'),
  'projects.manage': T('manager'),
  'finance.view': T(),
  'finance.manage': T(),
  'team.view': T('manager', 'hr_admin'),
  'team.manage': T('hr_admin'),
  'security.view': T('it_admin'),
  'security.manage': T('it_admin'),
  'settings.view': T('it_admin'),
  'settings.manage': T('it_admin'),
})

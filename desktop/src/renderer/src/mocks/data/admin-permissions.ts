/**
 * RBAC permission catalogue + default grant matrix (A-2).
 *
 * The five roles are canonical and fixed (`@/config/roles` — admin / it_support
 * / manager / hr / member); no custom roles in 1.0. Capabilities are grouped by
 * functional area (resource × action) so the matrix stays readable. The admin
 * role implicitly holds every capability and is locked in the UI + server.
 *
 * Real RBAC enforcement lives in the gateway (Luke's track 🔒); this matrix is a
 * mock-persisted FE model, swap-ready behind the `/api/v1/admin/permissions`
 * contract.
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
  'users.view': T('it_support', 'manager', 'hr'),
  'users.manage': T('it_support'),
  'roles.manage': T(),
  'crm.view': T('it_support', 'manager', 'hr', 'member'),
  'crm.edit': T('manager', 'member'),
  'crm.export': T('manager'),
  'projects.view': T('it_support', 'manager', 'hr', 'member'),
  'projects.edit': T('manager', 'member'),
  'projects.manage': T('manager'),
  'finance.view': T(),
  'finance.manage': T(),
  'team.view': T('manager', 'hr'),
  'team.manage': T('hr'),
  'security.view': T('it_support'),
  'security.manage': T('it_support'),
  'settings.view': T('it_support'),
  'settings.manage': T('it_support'),
})

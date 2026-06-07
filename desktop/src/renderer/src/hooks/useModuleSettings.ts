import { useAuthStore } from '@/stores/auth'
import { useModuleLeadsStore } from '@/stores/moduleLeads'
import type { ModuleId } from '@/lib/pricing'

/**
 * Whether the current user may edit a module's TENANT-wide settings.
 *
 * Admins lead every module implicitly; everyone else must be flagged as
 * Modul-Leiter for that specific module (set by an admin in the Team module).
 * Non-leads see tenant settings read-only with a lock.
 */
export function useIsModuleLead(moduleId: ModuleId): boolean {
  const user = useAuthStore((s) => s.user)
  const isLead = useModuleLeadsStore((s) => s.isLead)
  if (!user) return false
  if (user.roles.includes('admin')) return true
  return isLead(user.id, moduleId)
}

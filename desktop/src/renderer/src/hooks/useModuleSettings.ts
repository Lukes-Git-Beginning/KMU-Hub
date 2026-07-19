import { useAuthStore } from '@/stores/auth'
import { useModuleLeadsStore } from '@/stores/moduleLeads'
import { useHasCapability } from '@/hooks/useCapability'
import type { SettingsModuleId } from '@/lib/module-settings'

/**
 * Whether the current user may edit a module's TENANT-wide settings.
 *
 * R-3: the implicit admin shortcut is capability-based — any role holding
 * `settings:tenant:manage` (admin + it_admin presets, previews included) leads
 * every module; everyone else must be flagged as Modul-Leiter for that
 * specific module (set by an admin in the Team module). Non-leads see tenant
 * settings read-only with a lock (the "value visible, edit locked" pattern).
 */
export function useIsModuleLead(moduleId: SettingsModuleId): boolean {
  const user = useAuthStore((s) => s.user)
  const isLead = useModuleLeadsStore((s) => s.isLead)
  const manageTenant = useHasCapability('settings:tenant:manage')
  if (!user) return false
  if (manageTenant) return true
  return isLead(user.id, moduleId)
}

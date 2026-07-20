import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { ModuleId } from '@/lib/pricing'
import type { SettingsModuleId } from '@/lib/module-settings'

/**
 * Module-Leiter ("erweiterte Moduleinstellungen").
 *
 * A SECOND permission dimension on top of module *access* (see ModuleAssignmentTab,
 * which controls whether a user has a module at all). A user flagged as Modul-Leiter
 * for a module may edit that module's TENANT-wide settings — defaults that apply to
 * everyone — instead of only their personal overrides.
 *
 * 3-level hierarchy: tenant-default → Modul-Leiter override (tenant-wide) → user override.
 *
 * Admins are implicitly lead for every module (see useIsModuleLead). The flag is set
 * per-employee by an admin in the Team module (MemberProfileContent via ProfileSections).
 *
 * MOCK-FIRST: persisted to localStorage. Backend: `tenant_module_leads`
 * (tenant_id, user_id, module_id) — see .planning/backend-gaps.md.
 */
interface ModuleLeadsState {
  /** userId -> module ids where the user is a Modul-Leiter. */
  leads: Record<string, ModuleId[]>
  isLead: (userId: string, moduleId: SettingsModuleId) => boolean
  getLeadModules: (userId: string) => ModuleId[]
  setLead: (userId: string, moduleId: ModuleId, value: boolean) => void
  toggleLead: (userId: string, moduleId: ModuleId) => void
}

export const useModuleLeadsStore = create<ModuleLeadsState>()(
  persist(
    (set, get) => ({
      leads: {
        // Seed: the dev "manager" profile (u-pm) leads finance + team so the
        // lock/unlock states are demonstrable without switching to an admin.
        'u-pm': ['finance', 'team'],
      },

      isLead: (userId, moduleId) => (get().leads[userId] ?? []).includes(moduleId as ModuleId),

      getLeadModules: (userId) => get().leads[userId] ?? [],

      setLead: (userId, moduleId, value) =>
        set((s) => {
          const current = s.leads[userId] ?? []
          const next = value
            ? current.includes(moduleId)
              ? current
              : [...current, moduleId]
            : current.filter((m) => m !== moduleId)
          return { leads: { ...s.leads, [userId]: next } }
        }),

      toggleLead: (userId, moduleId) =>
        set((s) => {
          const current = s.leads[userId] ?? []
          const next = current.includes(moduleId)
            ? current.filter((m) => m !== moduleId)
            : [...current, moduleId]
          return { leads: { ...s.leads, [userId]: next } }
        }),
    }),
    { name: 'cosmi-module-leads' },
  ),
)

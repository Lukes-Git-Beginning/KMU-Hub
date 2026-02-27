import { useMemo } from 'react'
import { useAuthStore } from '@/stores/auth'
import { useProfileStore } from '@/stores/profile'
import { canSeeNavItem } from '@/config/roles'
import { isModuleAllowedForProfile } from '@/config/business-profiles'
import { navItems, type NavItemConfig } from '@/components/layout/sidebar/nav-items'

/**
 * Returns nav items filtered by RBAC + business profile.
 * Shared across all layout variants (Sidebar, Dock, TopNav, Classic).
 */
export function useFilteredNavItems() {
  const user = useAuthStore((s) => s.user)
  const businessProfileId = useProfileStore((s) => s.businessProfileId)
  const devShowAll = useProfileStore((s) => s.devShowAllModules)
  const enabledOptionals = useProfileStore((s) => s.enabledOptionalModules)

  return useMemo(() => {
    const filter = (items: NavItemConfig[]) =>
      items.filter((item) => {
        if (!canSeeNavItem(user, item.id)) return false
        if (devShowAll) return true
        return isModuleAllowedForProfile(item.id, businessProfileId, enabledOptionals)
      })

    const main = navItems.filter((i) => i.section === 'main')
    const bottom = navItems.filter((i) => i.section === 'bottom')

    return {
      mainItems: filter(main),
      bottomItems: filter(bottom),
      allItems: filter(navItems),
    }
  }, [user, businessProfileId, devShowAll, enabledOptionals])
}

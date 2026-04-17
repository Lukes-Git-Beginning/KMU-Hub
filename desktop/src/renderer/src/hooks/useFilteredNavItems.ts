import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '@/stores/auth'
import { useProfileStore } from '@/stores/profile'
import { canSeeNavItem } from '@/config/roles'
import { isModuleAllowedForProfile } from '@/config/business-profiles'
import { navItems, type NavItemConfig } from '@/components/layout/sidebar/nav-items'

/**
 * Returns nav items filtered by RBAC + business profile.
 * Shared across all layout variants (Sidebar, Dock, TopNav, Classic).
 *
 * Resolves i18n label keys at render time (navItems stores keys, not translated strings).
 */
export function useFilteredNavItems() {
  const { t } = useTranslation()
  const user = useAuthStore((s) => s.user)
  const businessProfileId = useProfileStore((s) => s.businessProfileId)
  const devShowAll = useProfileStore((s) => s.devShowAllModules)
  const enabledOptionals = useProfileStore((s) => s.enabledOptionalModules)

  return useMemo(() => {
    const resolve = (item: NavItemConfig): NavItemConfig => ({
      ...item,
      label: t(item.label),
      badge: item.badge
        ? { ...item.badge, value: item.badge.value ? t(item.badge.value) : item.badge.value }
        : item.badge,
    })

    const filter = (items: NavItemConfig[]) =>
      items.filter((item) => {
        if (!canSeeNavItem(user, item.id)) return false
        if (devShowAll) return true
        return isModuleAllowedForProfile(item.id, businessProfileId, enabledOptionals)
      })

    const main = navItems.filter((i) => i.section === 'main')
    const bottom = navItems.filter((i) => i.section === 'bottom')

    return {
      mainItems: filter(main).map(resolve),
      bottomItems: filter(bottom).map(resolve),
      allItems: filter(navItems).map(resolve),
    }
  }, [t, user, businessProfileId, devShowAll, enabledOptionals])
}

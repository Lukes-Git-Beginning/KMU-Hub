import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '@/stores/auth'
import { useProfileStore } from '@/stores/profile'
import { useUIStore } from '@/stores/ui'
import { canSeeNavItem } from '@/config/roles'
import { isModuleAllowedForProfile } from '@/config/business-profiles'
import { navItems, type NavItemConfig } from '@/components/layout/sidebar/nav-items'
import { useFeatureFlags } from '@/api/hooks/useFeatureFlags'
import { useModuleUnreadCounts } from '@/api/hooks/useNotifications'

/**
 * Nav item IDs that correspond to optional backend feature flags.
 * These 14 modules are off by default and controlled via COSMI_MODULE_*_ENABLED env vars.
 * Core items (dashboard, contacts, chat, etc.) are always visible and are NOT in this set.
 */
const MODULE_FLAG_IDS = new Set([
  'wiki', 'berichte', 'formulare', 'helpdesk', 'vertraege',
  'buchhaltung', 'meetings', 'rapporte', 'schichten', 'fuhrpark',
  'vermietung', 'inventar', 'einkauf', 'produktion',
])

/**
 * Returns nav items filtered by RBAC + business profile + server feature flags.
 * Shared across all layout variants (Sidebar, Dock, TopNav, Classic).
 *
 * Resolves i18n label keys at render time (navItems stores keys, not translated strings).
 * While feature flags are loading, module items remain hidden (fallback: safe off-state).
 */
export function useFilteredNavItems() {
  const { t } = useTranslation()
  const user = useAuthStore((s) => s.user)
  const businessProfileId = useProfileStore((s) => s.businessProfileId)
  const devShowAll = useProfileStore((s) => s.devShowAllModules)
  const enabledOptionals = useProfileStore((s) => s.enabledOptionalModules)
  const { isEnabled: isFlagEnabled, isLoading: flagsLoading } = useFeatureFlags()
  const openSettingsOverlay = useUIStore((s) => s.openSettingsOverlay)
  const moduleUnreadCounts = useModuleUnreadCounts()

  return useMemo(() => {
    const resolve = (item: NavItemConfig): NavItemConfig => {
      // Live notification unread count surfaces as a numeric badge. It never
      // overrides a 'live' real-time indicator (e.g. meetings) and falls back
      // to the item's own (translated) badge when there are no unreads.
      const liveUnread = moduleUnreadCounts[item.id] ?? 0
      const badge =
        item.badge?.type === 'live'
          ? item.badge
          : liveUnread > 0
            ? { type: 'text' as const, value: String(liveUnread) }
            : item.badge
              ? { ...item.badge, value: item.badge.value ? t(item.badge.value) : item.badge.value }
              : item.badge
      return {
        ...item,
        label: t(item.label),
        badge,
        // The bottom "Einstellungen" item opens the module-settings overlay
        // instead of navigating to a route.
        onActivate: item.id === 'settings' ? () => openSettingsOverlay() : undefined,
      }
    }

    const filter = (items: NavItemConfig[]) =>
      items.filter((item) => {
        if (!canSeeNavItem(user, item.id)) return false

        // devShowAll bypasses every gate (profile + feature flags).
        // It's the explicit "show me everything" switch for design/dev work.
        if (devShowAll) return true

        // Feature-flag gate: hide optional module items until the flag is enabled.
        // While flags are loading we conservatively hide them (no flash of ungated content).
        if (MODULE_FLAG_IDS.has(item.id)) {
          // video module uses id "meetings" in nav but flag key "modules.video"
          const flagKey = item.id === 'meetings' ? 'modules.video' : `modules.${item.id}`
          if (flagsLoading || !isFlagEnabled(flagKey)) return false
        }

        return isModuleAllowedForProfile(item.id, businessProfileId, enabledOptionals)
      })

    const main = navItems.filter((i) => i.section === 'main')
    const bottom = navItems.filter((i) => i.section === 'bottom')

    return {
      mainItems: filter(main).map(resolve),
      bottomItems: filter(bottom).map(resolve),
      allItems: filter(navItems).map(resolve),
    }
  }, [t, user, businessProfileId, devShowAll, enabledOptionals, isFlagEnabled, flagsLoading, openSettingsOverlay, moduleUnreadCounts])
}

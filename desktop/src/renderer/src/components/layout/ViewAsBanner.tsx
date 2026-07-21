/**
 * ViewAsBanner (RBAC R-5 — Part C.2).
 *
 * Persistent banner shown while a "Als Benutzer anzeigen" session is
 * active. Positioned at the top-center analogous to PermissionPreviewBanner
 * but slightly offset so both can coexist without overlapping.
 *
 * The banner is ALWAYS visible and does NOT scroll away (fixed positioning).
 * This is a product feature — it must work without DEV_BYPASS_AUTH.
 */
import { useTranslation } from 'react-i18next'
import { UserCheck, X } from 'lucide-react'
import { useViewAsStore } from '@/stores/viewAs'

export function ViewAsBanner() {
  const { t } = useTranslation()
  const viewAsUser = useViewAsStore((s) => s.viewAsUser)
  const exitViewAs = useViewAsStore((s) => s.exitViewAs)

  if (!viewAsUser) return null

  const fullName = `${viewAsUser.firstName} ${viewAsUser.lastName}`

  return (
    <div
      className="fixed left-1/2 top-3 z-[71] -translate-x-1/2"
      role="status"
      aria-live="polite"
      // Offset from PermissionPreviewBanner (z-[70]) so both are distinguishable.
      // ViewAs is a stronger admin action → slightly higher z-index.
    >
      <div className="flex items-center gap-2.5 rounded-full border border-warning/40 bg-warning-light py-1.5 pl-3.5 pr-1.5 shadow-lg backdrop-blur">
        <UserCheck className="h-3.5 w-3.5 shrink-0 text-warning" aria-hidden="true" />
        <span className="text-xs font-medium text-warning">
          {t('rbac.viewAs.banner', { name: fullName })}
        </span>
        <button
          type="button"
          onClick={exitViewAs}
          className="inline-flex items-center gap-1 rounded-full bg-warning px-2.5 py-1 text-xs font-medium text-white transition-colors hover:bg-warning/85 focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
          aria-label={t('rbac.viewAs.exit')}
        >
          <X className="h-3 w-3" aria-hidden="true" />
          {t('rbac.viewAs.exit')}
        </button>
      </div>
    </div>
  )
}

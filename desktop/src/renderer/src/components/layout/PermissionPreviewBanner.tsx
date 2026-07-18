/**
 * PermissionPreviewBanner (R-2) — global pill shown while the overlay preview
 * ("Als Rolle anzeigen") is active. The account stays signed in; capability
 * checks resolve against the previewed role until ended here. Fixed top-center
 * so it never collides with the ProfileSwitcher (bottom right).
 */
import { useTranslation } from 'react-i18next'
import { Eye, X } from 'lucide-react'
import { usePermissionsStore } from '@/stores/permissions'

export function PermissionPreviewBanner() {
  const { t } = useTranslation()
  const preview = usePermissionsStore((s) => s.preview)
  const endPreview = usePermissionsStore((s) => s.endPreview)

  if (!preview) return null

  return (
    <div className="fixed left-1/2 top-3 z-[70] -translate-x-1/2" role="status">
      <div className="flex items-center gap-2.5 rounded-full border border-info/40 bg-info-light py-1.5 pl-3.5 pr-1.5 shadow-lg backdrop-blur">
        <Eye className="h-3.5 w-3.5 shrink-0 text-info" aria-hidden="true" />
        <span className="text-xs font-medium text-info">
          {t('rbac.preview.banner', { name: preview.label })}
        </span>
        <button
          type="button"
          onClick={endPreview}
          className="inline-flex items-center gap-1 rounded-full bg-info px-2.5 py-1 text-xs font-medium text-white transition-colors hover:bg-info/85 focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
        >
          <X className="h-3 w-3" aria-hidden="true" />
          {t('rbac.preview.end')}
        </button>
      </div>
    </div>
  )
}

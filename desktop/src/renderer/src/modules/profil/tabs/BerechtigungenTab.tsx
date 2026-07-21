/**
 * BerechtigungenTab — read-only "Effektive Rechte" view of the own account.
 * Thin wrapper since R-2: the shared EffectivePermissionsView renders roles,
 * filter and grouped grants; the same view backs the admin/HR per-user modal.
 */
import { useTranslation } from 'react-i18next'
import { ShieldCheck } from 'lucide-react'
import { usePermissionsStore } from '@/stores/permissions'
import { EffectivePermissionsView } from '@/components/shared/rbac/EffectivePermissionsView'

export default function BerechtigungenTab() {
  const { t } = useTranslation()
  const roles = usePermissionsStore((s) => s.roles)
  const capabilities = usePermissionsStore((s) => s.capabilities)
  const deniedByOverride = usePermissionsStore((s) => s.deniedByOverride)

  return (
    <div className="mx-auto max-w-3xl space-y-4 p-6">
      <section className="space-y-1.5">
        <div className="flex items-center gap-2">
          <ShieldCheck className="h-4 w-4 text-primary" aria-hidden="true" />
          <h2 className="text-sm font-semibold text-foreground">{t('rbac.effective.title')}</h2>
        </div>
        <p className="text-xs text-muted-foreground">{t('rbac.effective.intro')}</p>
      </section>

      <EffectivePermissionsView roles={roles} capabilities={capabilities} deniedByOverride={deniedByOverride} />
    </div>
  )
}

import { ShieldCheck, UserCog, ArrowRight } from 'lucide-react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ModuleSettingsShell } from '@/components/shared/ModuleSettingsShell'

/**
 * Module-settings entry for security & DSGVO.
 *
 * The actual controls live in dedicated surfaces: tenant-wide policies +
 * compliance tools in the Admin-Hub (/admin/security), personal 2FA/sessions
 * in the profile settings (/settings). This panel orients the user and links
 * out, so security is discoverable from the module-settings overlay like every
 * other module — without duplicating the panels.
 */
export function SecuritySettingsPanel() {
  const { t } = useTranslation()

  const linkClass =
    'inline-flex items-center gap-2 rounded-lg border border-border px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors'

  return (
    <ModuleSettingsShell
      moduleId="security"
      titleKey="settings.security.title"
      descriptionKey="security.moduleSettings.subtitle"
      sections={[
        {
          id: 'tenant',
          titleKey: 'security.moduleSettings.policiesTitle',
          descriptionKey: 'security.moduleSettings.policiesDesc',
          scope: 'tenant',
          icon: ShieldCheck,
          children: (
            <Link to="/admin/security" className={linkClass}>
              {t('security.moduleSettings.openHub')}
              <ArrowRight className="h-4 w-4" />
            </Link>
          ),
        },
        {
          id: 'personal',
          titleKey: 'security.moduleSettings.personalTitle',
          descriptionKey: 'security.moduleSettings.personalDesc',
          scope: 'personal',
          icon: UserCog,
          children: (
            <Link to="/settings" className={linkClass}>
              {t('security.moduleSettings.openProfile')}
              <ArrowRight className="h-4 w-4" />
            </Link>
          ),
        },
      ]}
    />
  )
}

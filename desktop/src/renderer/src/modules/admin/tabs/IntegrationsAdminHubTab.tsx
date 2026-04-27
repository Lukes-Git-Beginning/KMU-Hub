/**
 * IntegrationsAdminHubTab — Phase-3-Platzhalter.
 *
 * Workspace-übergreifende Integrationen (Slack, Zapier, Webhooks) kommen in Phase 4.
 * Bexio / DATEV / Lexware wandern in Phase 4 ins Buchhaltung-Modul.
 */
import { useTranslation } from 'react-i18next'
import { Plug } from 'lucide-react'

export default function IntegrationsAdminHubTab() {
  const { t } = useTranslation()
  return (
    <div className="px-6 py-6 max-w-2xl">
      <h2 className="text-sm font-semibold text-foreground mb-0.5">{t('admin.hub.tabs.integrations')}</h2>
      <p className="text-xs text-muted-foreground mb-8">
        {t('admin.hub.placeholderInfo', { source: 'Phase 4 (Buchhaltung-Modul + Mail-Modul)' })}
      </p>

      <div className="flex flex-col items-center gap-3 py-16 text-center">
        <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-secondary">
          <Plug className="h-6 w-6 text-muted-foreground" />
        </div>
        <p className="text-sm font-medium text-foreground">Integrationen kommen in Phase 4</p>
        <p className="text-xs text-muted-foreground max-w-xs">
          Bexio, DATEV und Lexware werden direkt ins Buchhaltungs-Modul integriert.
          Slack, Zapier und Webhook-Integrationen erscheinen hier.
        </p>
      </div>
    </div>
  )
}

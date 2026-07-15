/**
 * IntegrationsAdminHubTab — placeholder for workspace-wide integrations.
 *
 * Bexio / DATEV / Lexware are integrated directly in the Accounting module;
 * Slack / Zapier / webhooks will land here in a later phase.
 */
import { useTranslation } from 'react-i18next'
import { Plug } from 'lucide-react'
import { EmptyState } from '@/components/shared/EmptyState'

export default function IntegrationsAdminHubTab() {
  const { t } = useTranslation()
  return (
    <div className="px-6 py-6">
      <EmptyState
        icon={Plug}
        title={t('admin.hub.integrations.emptyTitle')}
        description={t('admin.hub.integrations.emptyDesc')}
      />
    </div>
  )
}

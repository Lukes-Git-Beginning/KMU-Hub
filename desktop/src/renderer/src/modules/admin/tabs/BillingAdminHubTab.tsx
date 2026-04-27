/**
 * BillingAdminHubTab — Wrapper um BillingSettingsTab.
 * Kein zusätzliches Chrome — der Tab-Inhalt von Phase 1 wird direkt gerendert.
 */
import { BillingSettingsTab } from '@/modules/settings/tabs/BillingSettingsTab'

export default function BillingAdminHubTab() {
  return (
    <div className="px-6 py-6">
      <BillingSettingsTab />
    </div>
  )
}

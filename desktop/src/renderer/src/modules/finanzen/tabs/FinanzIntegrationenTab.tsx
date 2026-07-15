/**
 * FinanzIntegrationenTab — Buchhaltungs-Integrationen im Buchhaltungs-Modul.
 *
 * Extrahiert aus settings/tabs/IntegrationSettingsTab.tsx (Kategorie: buchhaltung).
 * Zeigt nur die finanzspezifischen Integrationen: DATEV, Bexio, Lexware.
 * Nutzt die gleichen Panel-Komponenten wie die Settings-Seite.
 *
 * Apple-Disziplin: listenbasiert, kein Card-in-Card, keine Hero-Sektionen.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plug } from 'lucide-react'
import { useIntegrationStore } from '@/stores/integrations'
import {
  INTEGRATION_REGISTRY,
  type IntegrationDefinition,
} from '@/modules/settings/integrations/integration-registry'
import { IntegrationCard } from '@/modules/settings/integrations/IntegrationCard'
import { DATEVConfigPanel } from '@/modules/settings/integrations/DATEVConfigPanel'
import { BexioSetupWizard } from '@/modules/settings/integrations/BexioSetupWizard'
import { BexioSyncDashboard } from '@/modules/settings/integrations/BexioSyncDashboard'
import { GenericIntegrationPanel } from '@/modules/settings/integrations/GenericIntegrationPanel'
import { useBexioConnectionStatus } from '@/api/hooks/useBexio'

// Nur Buchhaltungs-Integrationen aus dem Registry
const BUCHHALTUNGS_INTEGRATIONS = INTEGRATION_REGISTRY.filter(
  (d) => d.category === 'buchhaltung',
)

export function FinanzIntegrationenTab({ embedded = false }: { embedded?: boolean } = {}) {
  const { t } = useTranslation()
  const [activeIntegrationId, setActiveIntegrationId] = useState<string | null>(null)
  const integrationStore = useIntegrationStore()

  // Bexio uses the real backend flow (setup wizard + sync dashboard) instead of
  // the drill-down panel: card → wizard when disconnected, dashboard when connected.
  const { data: bexioConnection } = useBexioConnectionStatus()
  const [bexioWizardOpen, setBexioWizardOpen] = useState(false)
  const [bexioDashboardOpen, setBexioDashboardOpen] = useState(false)

  const openBexio = () => {
    if (bexioConnection?.connected) setBexioDashboardOpen(true)
    else setBexioWizardOpen(true)
  }

  const getCardStatus = (def: IntegrationDefinition) => {
    if (def.id === 'bexio') return bexioConnection?.connected ? 'connected' : 'disconnected'
    return integrationStore.getStatus(def.id)
  }

  // Drill-down in Integrations-Panel
  if (activeIntegrationId) {
    const def = BUCHHALTUNGS_INTEGRATIONS.find((d) => d.id === activeIntegrationId)
    if (!def) {
      setActiveIntegrationId(null)
      return null
    }

    const handleBack = () => setActiveIntegrationId(null)

    if (def.id === 'datev-rechnungswesen') {
      return <DATEVConfigPanel onBack={handleBack} />
    }
    return <GenericIntegrationPanel definition={def} onBack={handleBack} />
  }

  return (
    <div className="max-w-3xl">
      {!embedded && (
        <div className="mb-6">
          <h2 className="text-base font-semibold text-foreground">
            {t('finanzen.integrationen.title', { defaultValue: 'Buchhaltungs-Integrationen' })}
          </h2>
          <p className="text-sm text-muted-foreground mt-0.5">
            {t('finanzen.integrationen.subtitle', { defaultValue: 'Verknüpfung zu externen Buchhaltungs-Systemen' })}
          </p>
        </div>
      )}

      {BUCHHALTUNGS_INTEGRATIONS.length === 0 ? (
        <div className="flex flex-col items-center gap-3 py-12 text-center">
          <Plug className="h-8 w-8 text-muted-foreground/40" />
          <p className="text-sm text-muted-foreground">
            {t('finanzen.integrationen.empty', { defaultValue: 'Keine Integrationen verfügbar.' })}
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
          {BUCHHALTUNGS_INTEGRATIONS.map((def) => (
            <IntegrationCard
              key={def.id}
              definition={def}
              status={getCardStatus(def)}
              onClick={() => (def.id === 'bexio' ? openBexio() : setActiveIntegrationId(def.id))}
            />
          ))}
        </div>
      )}

      {/* Bexio setup wizard + sync dashboard (real backend flow) */}
      <BexioSetupWizard
        isOpen={bexioWizardOpen}
        onClose={() => setBexioWizardOpen(false)}
      />
      {bexioDashboardOpen && (
        <BexioSyncDashboard
          isOpen={bexioDashboardOpen}
          onClose={() => setBexioDashboardOpen(false)}
        />
      )}
    </div>
  )
}

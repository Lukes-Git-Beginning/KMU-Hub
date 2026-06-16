import { SlidersHorizontal, Building2, Receipt, Plug, BookOpen } from 'lucide-react'
import { ModuleSettingsShell, type ModuleSettingsSection } from '@/components/shared'
import { FinancePersonalPrefs } from '@/modules/finanzen/FinancePersonalPrefs'
import { KontierungSettings } from '@/modules/finanzen/KontierungSettings'
import { StammdatenTab } from '@/modules/finanzen/tabs/StammdatenTab'
import { FinanzIntegrationenTab } from '@/modules/finanzen/tabs/FinanzIntegrationenTab'
import { FinanceSettingsTab } from '@/modules/settings/tabs/FinanceSettingsTab'

/**
 * FinanceSettingsPanel — the "Buchhaltung" entry of the module-settings overlay.
 * Consolidates the former in-page admin tabs (Stammdaten, Integrationen,
 * Settings) into one scope-aware surface. Tenant sections are editable only by a
 * Buchhaltungs-Modul-Leiter / admin.
 */
export function FinanceSettingsPanel() {
  const sections: ModuleSettingsSection[] = [
    {
      id: 'personal',
      titleKey: 'finanzen.settings.personal.title',
      descriptionKey: 'finanzen.settings.personal.desc',
      scope: 'personal',
      icon: SlidersHorizontal,
      children: <FinancePersonalPrefs />,
    },
    {
      id: 'stammdaten',
      titleKey: 'finanzen.settings.stammdaten.title',
      descriptionKey: 'finanzen.settings.stammdaten.desc',
      scope: 'tenant',
      icon: Building2,
      children: <StammdatenTab embedded />,
    },
    {
      id: 'invoicing',
      titleKey: 'finanzen.settings.invoicing.title',
      descriptionKey: 'finanzen.settings.invoicing.desc',
      scope: 'tenant',
      icon: Receipt,
      children: <FinanceSettingsTab embedded />,
    },
    {
      id: 'kontierung',
      titleKey: 'finanzen.settings.kontierung.title',
      descriptionKey: 'finanzen.settings.kontierung.desc',
      scope: 'tenant',
      icon: BookOpen,
      children: <KontierungSettings />,
    },
    {
      id: 'integrations',
      titleKey: 'finanzen.settings.integrations.title',
      descriptionKey: 'finanzen.settings.integrations.desc',
      scope: 'tenant',
      icon: Plug,
      children: <FinanzIntegrationenTab embedded />,
    },
  ]

  return (
    <ModuleSettingsShell
      moduleId="finance"
      titleKey="finanzen.settings.title"
      descriptionKey="finanzen.settings.subtitle"
      sections={sections}
    />
  )
}

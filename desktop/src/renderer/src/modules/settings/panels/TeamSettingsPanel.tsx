import { SlidersHorizontal, Building2, Banknote } from 'lucide-react'
import { ModuleSettingsShell, type ModuleSettingsSection } from '@/components/shared'
import { TeamPersonalPrefs } from '@/modules/team/TeamPersonalPrefs'
import { TeamSettingsTab } from '@/modules/settings/tabs/TeamSettingsTab'
import { PayrollSettings } from '@/modules/team/PayrollSettings'

/**
 * TeamSettingsPanel — the "Team" entry of the module-settings overlay.
 * Persönlich (start tab, member view) + Für-alle (HR config + DATEV-Lohn).
 * Tenant sections editable only by a Team-Modul-Leiter / admin.
 */
export function TeamSettingsPanel() {
  const sections: ModuleSettingsSection[] = [
    {
      id: 'personal',
      titleKey: 'team.settings.personal.title',
      descriptionKey: 'team.settings.personal.desc',
      scope: 'personal',
      icon: SlidersHorizontal,
      children: <TeamPersonalPrefs />,
    },
    {
      id: 'hr',
      titleKey: 'team.settings.hr.title',
      descriptionKey: 'team.settings.hr.desc',
      scope: 'tenant',
      icon: Building2,
      children: <TeamSettingsTab embedded />,
    },
    {
      id: 'payroll',
      titleKey: 'team.settings.payroll.title',
      descriptionKey: 'team.settings.payroll.desc',
      scope: 'tenant',
      icon: Banknote,
      children: <PayrollSettings />,
    },
  ]

  return (
    <ModuleSettingsShell
      moduleId="team"
      titleKey="team.settings.title"
      descriptionKey="team.settings.subtitle"
      sections={sections}
    />
  )
}

/**
 * Modul-Zuteilung as a module-settings entry (COSMI group, next to billing).
 *
 * Module access per employee is a licensing/billing concern, so it lives in the
 * settings overlay rather than the team page. Deep links from Billing /
 * MemberDetailPanel still work: they set a navigation intent + open this entry,
 * which we read once into the initial filter.
 */
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigationStore } from '@/stores/navigation'
import { ModuleAssignmentTab, type ModuleAssignmentInitialFilter } from '@/modules/team/ModuleAssignmentTab'

export function ModuleAssignmentSettingsPanel() {
  const { t } = useTranslation()
  const intent = useNavigationStore((s) => s.intent)
  const clearIntent = useNavigationStore((s) => s.clearIntent)
  const [initialFilter, setInitialFilter] = useState<ModuleAssignmentInitialFilter | undefined>(undefined)

  // Consume an "open-team-modulzuteilung" intent once (deep link with filter).
  useEffect(() => {
    if (intent?.type === 'open-team-modulzuteilung') {
      setInitialFilter(intent.data as ModuleAssignmentInitialFilter)
      clearIntent()
    }
  }, [intent, clearIntent])

  return (
    <div className="max-w-4xl">
      <h2 className="mb-1 text-foreground">{t('moduleSettings.entries.moduleAssignment')}</h2>
      <p className="mb-6 text-sm text-muted-foreground">{t('team.moduleAssignment.subtitle')}</p>
      <ModuleAssignmentTab initialFilter={initialFilter} />
    </div>
  )
}

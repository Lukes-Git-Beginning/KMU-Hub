import { SlidersHorizontal, GitBranch, ListPlus, Tags } from 'lucide-react'
import { ModuleSettingsShell, type ModuleSettingsSection } from '@/components/shared'
import { CrmPersonalPrefs } from '@/modules/kontakte/CrmPersonalPrefs'
import { PipelineStagesEditor } from '@/modules/kontakte/PipelineStagesEditor'
import { CustomFieldsManager } from '@/modules/kontakte/CustomFieldsManager'
import { TagManager } from '@/modules/kontakte/TagManager'

/**
 * CrmSettingsPanel — the "CRM" entry of the module-settings overlay (Kontakte
 * Phase 7). Tenant-scoped: only a CRM Modul-Leiter / admin may edit these.
 */
export function CrmSettingsPanel() {
  const sections: ModuleSettingsSection[] = [
    {
      id: 'personal',
      titleKey: 'crm.settings.personal.title',
      descriptionKey: 'crm.settings.personal.desc',
      scope: 'personal',
      icon: SlidersHorizontal,
      children: <CrmPersonalPrefs />,
    },
    {
      id: 'pipeline',
      titleKey: 'crm.settings.pipeline.title',
      descriptionKey: 'crm.settings.pipeline.desc',
      scope: 'tenant',
      icon: GitBranch,
      children: <PipelineStagesEditor />,
    },
    {
      id: 'customFields',
      titleKey: 'crm.settings.customFields.title',
      descriptionKey: 'crm.settings.customFields.desc',
      scope: 'tenant',
      icon: ListPlus,
      children: <CustomFieldsManager />,
    },
    {
      id: 'tags',
      titleKey: 'crm.settings.tags.title',
      descriptionKey: 'crm.settings.tags.desc',
      scope: 'tenant',
      icon: Tags,
      children: <TagManager />,
    },
  ]

  return (
    <ModuleSettingsShell
      moduleId="crm"
      titleKey="crm.settings.title"
      descriptionKey="crm.settings.subtitle"
      sections={sections}
    />
  )
}

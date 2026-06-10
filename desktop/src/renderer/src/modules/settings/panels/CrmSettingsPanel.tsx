import { SlidersHorizontal, GitBranch, ListPlus, Tags, Gauge, Flame } from 'lucide-react'
import { ModuleSettingsShell, type ModuleSettingsSection } from '@/components/shared'
import { CrmPersonalPrefs } from '@/modules/kontakte/CrmPersonalPrefs'
import { PipelineStagesEditor } from '@/modules/kontakte/PipelineStagesEditor'
import { CustomFieldsManager } from '@/modules/kontakte/CustomFieldsManager'
import { TagManager } from '@/modules/kontakte/TagManager'
import { SegmentSettings } from '@/modules/kontakte/SegmentSettings'
import { LeadScoringSettings } from '@/modules/kontakte/LeadScoringSettings'

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
    {
      id: 'segments',
      titleKey: 'crm.settings.segments.title',
      descriptionKey: 'crm.settings.segments.desc',
      scope: 'tenant',
      icon: Gauge,
      children: <SegmentSettings />,
    },
    {
      id: 'leadScoring',
      titleKey: 'crm.settings.leadScoring.title',
      descriptionKey: 'crm.settings.leadScoring.desc',
      scope: 'tenant',
      icon: Flame,
      children: <LeadScoringSettings />,
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

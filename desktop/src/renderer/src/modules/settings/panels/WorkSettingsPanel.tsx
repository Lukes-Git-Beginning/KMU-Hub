import { SlidersHorizontal, Tags, ListPlus, GitBranch, LayoutTemplate, Clock } from 'lucide-react'
import { ModuleSettingsShell, type ModuleSettingsSection } from '@/components/shared'
import { WorkPersonalPrefs } from '@/modules/work/settings/WorkPersonalPrefs'
import { LabelTaxonomyManager } from '@/modules/work/settings/LabelTaxonomyManager'
import { WorkCustomFieldsManager } from '@/modules/work/settings/WorkCustomFieldsManager'
import { DefaultStatusSetEditor } from '@/modules/work/settings/DefaultStatusSetEditor'
import { ProjectTemplatesManager } from '@/modules/work/settings/ProjectTemplatesManager'
import { TimeRulesSettings } from '@/modules/work/settings/TimeRulesSettings'

/**
 * WorkSettingsPanel — the "Projekte/Aufgaben" entry of the module-settings
 * overlay (work batch Phase 2). Personal prefs are always editable; tenant
 * sections (labels, custom fields, default workflow, templates, time rules)
 * require a module lead / admin.
 */
export function WorkSettingsPanel() {
  const sections: ModuleSettingsSection[] = [
    {
      id: 'personal',
      titleKey: 'work.settings.personal.title',
      descriptionKey: 'work.settings.personal.desc',
      scope: 'personal',
      icon: SlidersHorizontal,
      children: <WorkPersonalPrefs />,
    },
    {
      id: 'labels',
      titleKey: 'work.settings.labels.title',
      descriptionKey: 'work.settings.labels.desc',
      scope: 'tenant',
      icon: Tags,
      children: <LabelTaxonomyManager />,
    },
    {
      id: 'templates',
      titleKey: 'work.settings.templates.title',
      descriptionKey: 'work.settings.templates.desc',
      scope: 'tenant',
      icon: LayoutTemplate,
      children: <ProjectTemplatesManager />,
    },
    {
      id: 'statusSet',
      titleKey: 'work.settings.statusSet.title',
      descriptionKey: 'work.settings.statusSet.desc',
      scope: 'tenant',
      icon: GitBranch,
      children: <DefaultStatusSetEditor />,
    },
    {
      id: 'customFields',
      titleKey: 'work.settings.fields.title',
      descriptionKey: 'work.settings.fields.desc',
      scope: 'tenant',
      icon: ListPlus,
      children: <WorkCustomFieldsManager />,
    },
    {
      id: 'time',
      titleKey: 'work.settings.time.title',
      descriptionKey: 'work.settings.time.desc',
      scope: 'tenant',
      icon: Clock,
      children: <TimeRulesSettings />,
    },
  ]

  return (
    <ModuleSettingsShell
      moduleId="projects"
      titleKey="work.settings.title"
      descriptionKey="work.settings.subtitle"
      sections={sections}
    />
  )
}

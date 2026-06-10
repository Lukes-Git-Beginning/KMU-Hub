import { SlidersHorizontal, HardDrive, FileType2, Share2 } from 'lucide-react'
import { ModuleSettingsShell, type ModuleSettingsSection } from '@/components/shared'
import { DokumentePersonalPrefs } from './DokumentePersonalPrefs'
import {
  DokumenteStorageSettings,
  DokumenteFileTypeSettings,
  DokumenteSharingSettings,
} from './DokumenteTenantSettings'

/**
 * DokumenteSettingsPanel — the "Dokumente" entry of the module-settings
 * overlay. Personal prefs (view, sort, density) are always editable; tenant
 * sections (storage/retention, file types, sharing/OnlyOffice) require a
 * module lead / admin and are mock-first until the backend is wired.
 */
export function DokumenteSettingsPanel() {
  const sections: ModuleSettingsSection[] = [
    {
      id: 'personal',
      titleKey: 'dokumente.settings.personal.title',
      descriptionKey: 'dokumente.settings.personal.desc',
      scope: 'personal',
      icon: SlidersHorizontal,
      children: <DokumentePersonalPrefs />,
    },
    {
      id: 'storage',
      titleKey: 'dokumente.settings.storage.title',
      descriptionKey: 'dokumente.settings.storage.desc',
      scope: 'tenant',
      icon: HardDrive,
      children: <DokumenteStorageSettings />,
    },
    {
      id: 'fileTypes',
      titleKey: 'dokumente.settings.fileTypes.title',
      descriptionKey: 'dokumente.settings.fileTypes.desc',
      scope: 'tenant',
      icon: FileType2,
      children: <DokumenteFileTypeSettings />,
    },
    {
      id: 'sharing',
      titleKey: 'dokumente.settings.sharing.title',
      descriptionKey: 'dokumente.settings.sharing.desc',
      scope: 'tenant',
      icon: Share2,
      children: <DokumenteSharingSettings />,
    },
  ]

  return (
    <ModuleSettingsShell
      moduleId="documents"
      titleKey="dokumente.settings.title"
      descriptionKey="dokumente.settings.subtitle"
      sections={sections}
    />
  )
}

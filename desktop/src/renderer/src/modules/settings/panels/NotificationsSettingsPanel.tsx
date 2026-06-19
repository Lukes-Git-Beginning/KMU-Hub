import { Bell } from 'lucide-react'
import { ModuleSettingsShell } from '@/components/shared/ModuleSettingsShell'
import { NotificationSettingsTab } from '../tabs/NotificationSettingsTab'

/**
 * Module-settings entry for notifications. Reuses the existing personal
 * notification preferences (module × channel matrix, DND, quiet hours, mutes)
 * inside the scope-aware shell, so they are reachable from the module-settings
 * overlay without duplicating the panel.
 *
 * Notifications are a cross-cutting surface (no priced/leadable module), so
 * there is no tenant-wide scope to delegate — only the personal section shows.
 */
export function NotificationsSettingsPanel() {
  return (
    <ModuleSettingsShell
      moduleId="notifications"
      titleKey="settings.notifications.title"
      descriptionKey="settings.notifications.subtitle"
      sections={[
        {
          id: 'personal',
          titleKey: 'notifications.moduleSettings.sectionTitle',
          descriptionKey: 'notifications.moduleSettings.sectionDesc',
          scope: 'personal',
          icon: Bell,
          children: <NotificationSettingsTab embedded />,
        },
      ]}
    />
  )
}

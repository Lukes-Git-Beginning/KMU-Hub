/**
 * Admin page for org-wide CalDAV/CardDAV management.
 *
 * Provides:
 * - Org-wide enable/disable toggle
 * - User audit table (who has CalDAV enabled, password counts)
 * - Admin password revocation per user
 */
import { useTranslation } from 'react-i18next'
import { Switch } from '@/components/ui/switch'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { Calendar, Shield, Trash2, Users } from 'lucide-react'
import {
  useAdminCalDAVSettings,
  useSetAdminCalDAVSettings,
  useAdminCalDAVUsers,
  useAdminRevokeUserPasswords,
} from '@/api/hooks/useCaldav'

export default function CalDAVAdminPage() {
  const { t } = useTranslation()
  const { data: settings, isLoading: settingsLoading } =
    useAdminCalDAVSettings()
  const setSettings = useSetAdminCalDAVSettings()
  const { data: users, isLoading: usersLoading } = useAdminCalDAVUsers()
  const revokePasswords = useAdminRevokeUserPasswords()

  return (
    <div className="p-6 max-w-4xl mx-auto">
      <div className="flex items-center gap-3 mb-1">
        <Calendar className="h-5 w-5 text-muted-foreground" />
        <h1 className="text-lg font-semibold text-foreground">
          {t('admin.caldav.title')}
        </h1>
      </div>
      <p className="text-sm text-muted-foreground mb-6">
        {t('admin.caldav.description')}
      </p>

      {/* Org toggle */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-3">
          <Shield className="h-4 w-4 text-muted-foreground" />
          <h2 className="text-sm font-medium">{t('admin.caldav.orgSettings')}</h2>
        </div>

        <div className="flex items-center justify-between rounded-md border border-border p-4">
          <div>
            <p className="text-sm font-medium">
              {t('admin.caldav.enableToggle')}
            </p>
            <p className="text-xs text-muted-foreground mt-0.5">
              {t('admin.caldav.enableDescription')}
            </p>
          </div>
          {settingsLoading ? (
            <div className="h-5 w-10 bg-muted animate-pulse rounded-full" />
          ) : (
            <Switch
              checked={settings?.enabled ?? false}
              onCheckedChange={(checked) => setSettings.mutate(checked)}
              disabled={setSettings.isPending}
            />
          )}
        </div>
      </section>

      <Separator className="mb-8" />

      {/* User audit */}
      <section>
        <div className="flex items-center gap-2 mb-3">
          <Users className="h-4 w-4 text-muted-foreground" />
          <h2 className="text-sm font-medium">
            {t('admin.caldav.usersWithAccess')}
          </h2>
        </div>

        {usersLoading ? (
          <div className="animate-pulse space-y-2">
            <div className="h-10 bg-muted rounded" />
            <div className="h-10 bg-muted rounded" />
          </div>
        ) : users && users.length > 0 ? (
          <div className="border border-border rounded-md overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-muted/50 text-left">
                  <th className="px-3 py-2 font-medium">{t('admin.caldav.user')}</th>
                  <th className="px-3 py-2 font-medium">{t('admin.caldav.email')}</th>
                  <th className="px-3 py-2 font-medium text-center">
                    {t('admin.caldav.passwords')}
                  </th>
                  <th className="px-3 py-2 font-medium">{t('admin.caldav.lastUsed')}</th>
                  <th className="px-3 py-2 font-medium text-right">
                    {t('common.actions')}
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {users.map((user) => (
                  <tr key={user.id} className="hover:bg-accent/30">
                    <td className="px-3 py-2">
                      {user.first_name} {user.last_name}
                    </td>
                    <td className="px-3 py-2 text-muted-foreground">
                      {user.email}
                    </td>
                    <td className="px-3 py-2 text-center">
                      {user.password_count}
                    </td>
                    <td className="px-3 py-2 text-muted-foreground">
                      {user.last_used
                        ? new Date(user.last_used).toLocaleDateString(
                            'de-DE',
                          )
                        : t('admin.caldav.never')}
                    </td>
                    <td className="px-3 py-2 text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-red-500 hover:text-red-600"
                        onClick={() => revokePasswords.mutate(user.id)}
                        disabled={
                          revokePasswords.isPending ||
                          user.password_count === 0
                        }
                      >
                        <Trash2 className="h-3.5 w-3.5 mr-1" />
                        {t('admin.caldav.revokeAll')}
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">
            {t('admin.caldav.noUsers')}
          </p>
        )}
      </section>
    </div>
  )
}

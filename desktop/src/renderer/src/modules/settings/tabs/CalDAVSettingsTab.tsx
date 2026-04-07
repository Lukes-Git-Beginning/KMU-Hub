/**
 * CalDAV/CardDAV settings tab: app-specific password management,
 * per-client setup wizard, and connection test.
 *
 * All labels in German (Deutschland-First).
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Separator } from '@/components/ui/separator'
import {
  Calendar,
  Copy,
  Key,
  Plus,
  Shield,
  Trash2,
  ChevronDown,
  ChevronRight,
  CheckCircle,
  XCircle,
  RefreshCw,
  Monitor,
  Smartphone,
  Globe,
  Loader2,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  useAppPasswords,
  useCalDAVStatus,
  useCreateAppPassword,
  useRevokeAppPassword,
  useEnableCalDAV,
  useDisableCalDAV,
  useTestCalDAVConnection,
} from '@/api/hooks/useCaldav'
import { API_BASE_URL } from '@/lib/constants'

export function CalDAVSettingsTab() {
  const { t } = useTranslation()
  const { data: status, isLoading: statusLoading } = useCalDAVStatus()
  const { data: passwords, isLoading: passwordsLoading } = useAppPasswords()
  const createPassword = useCreateAppPassword()
  const revokePassword = useRevokeAppPassword()
  const enableCalDAV = useEnableCalDAV()
  const disableCalDAV = useDisableCalDAV()

  const [newLabel, setNewLabel] = useState('')
  const [showNewPassword, setShowNewPassword] = useState<string | null>(null)
  const [expandedClient, setExpandedClient] = useState<string | null>(null)

  const serverURL = API_BASE_URL
  const userId = status?.user_id ?? ''

  const caldavURL = `${serverURL}/caldav/principals/${userId}/calendars/`
  const carddavURL = `${serverURL}/carddav/principals/${userId}/addressbooks/`

  const handleCreatePassword = async () => {
    if (!newLabel.trim()) {
      toast.error(t('settings.caldav.labelRequired'))
      return
    }
    const result = await createPassword.mutateAsync(newLabel.trim())
    setShowNewPassword(result.password)
    setNewLabel('')
    toast.success(t('settings.caldav.passwordCreated'))
  }

  const handleCopy = (text: string, label: string) => {
    navigator.clipboard.writeText(text)
    toast.success(t('settings.caldav.copied', { label }))
  }

  const handleToggle = () => {
    if (status?.user_enabled) {
      disableCalDAV.mutate()
    } else {
      enableCalDAV.mutate()
    }
  }

  const toggleClient = (client: string) => {
    setExpandedClient(expandedClient === client ? null : client)
  }

  if (statusLoading) {
    return (
      <div className="p-6">
        <div className="animate-pulse space-y-4">
          <div className="h-6 bg-muted rounded w-48" />
          <div className="h-4 bg-muted rounded w-96" />
        </div>
      </div>
    )
  }

  return (
    <div className="max-w-2xl p-6">
      <h2 className="text-foreground mb-1">CalDAV / CardDAV</h2>
      <p className="text-sm text-muted-foreground mb-6">
        {t('settings.caldav.subtitle')}
      </p>

      {/* Status section */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Calendar className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">Status</h3>
        </div>

        <div className="space-y-3">
          <div className="flex items-center justify-between rounded-md border border-border p-3">
            <div>
              <p className="text-sm font-medium">{t('settings.caldav.orgLabel')}</p>
              <p className="text-xs text-muted-foreground">
                {status?.org_enabled
                  ? t('settings.caldav.orgEnabled')
                  : t('settings.caldav.orgDisabled')}
              </p>
            </div>
            {status?.org_enabled ? (
              <CheckCircle className="h-5 w-5 text-success" />
            ) : (
              <XCircle className="h-5 w-5 text-destructive" />
            )}
          </div>

          <div className="flex items-center justify-between rounded-md border border-border p-3">
            <div>
              <p className="text-sm font-medium">{t('settings.caldav.personalLabel')}</p>
              <p className="text-xs text-muted-foreground">
                {t('settings.caldav.personalDesc')}
              </p>
            </div>
            <Switch
              checked={status?.user_enabled ?? false}
              onCheckedChange={handleToggle}
              disabled={!status?.org_enabled}
            />
          </div>
        </div>
      </section>

      <Separator className="mb-8" />

      {/* App-specific passwords */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Key className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">
            {t('settings.caldav.appPasswordsTitle')}
          </h3>
        </div>

        <p className="text-xs text-muted-foreground mb-4">
          {t('settings.caldav.appPasswordsDesc')}
        </p>

        {/* Password list */}
        {passwordsLoading ? (
          <div className="animate-pulse h-20 bg-muted rounded" />
        ) : passwords && passwords.length > 0 ? (
          <div className="border border-border rounded-md mb-4 divide-y divide-border">
            {passwords.map((pw) => (
              <div
                key={pw.id}
                className="flex items-center justify-between px-3 py-2"
              >
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium truncate">{pw.label}</p>
                  <p className="text-xs text-muted-foreground">
                    {pw.password_prefix}... | {t('settings.caldav.created')}:{' '}
                    {new Date(pw.created_at).toLocaleDateString('de-DE')}
                    {pw.last_used_at &&
                      ` | ${t('settings.caldav.lastUsed')}: ${new Date(pw.last_used_at).toLocaleDateString('de-DE')}`}
                  </p>
                </div>
                <div className="flex items-center gap-2 ml-2">
                  {pw.revoked_at ? (
                    <span className="text-xs text-destructive font-medium">
                      {t('settings.caldav.revoked')}
                    </span>
                  ) : (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => revokePassword.mutate(pw.id)}
                      className="text-destructive hover:text-destructive"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </Button>
                  )}
                </div>
              </div>
            ))}
          </div>
        ) : (
          <p className="text-sm text-muted-foreground mb-4">
            {t('settings.caldav.noPasswords')}
          </p>
        )}

        {/* New password dialog shown inline */}
        {showNewPassword && (
          <div className="border border-warning/30 bg-warning-light rounded-md p-4 mb-4">
            <p className="text-sm font-medium text-warning mb-2">
              {t('settings.caldav.passwordOnce')}
            </p>
            <div className="flex items-center gap-2">
              <code className="flex-1 text-sm bg-background border border-border rounded px-3 py-2 font-mono select-all break-all">
                {showNewPassword}
              </code>
              <Button
                variant="outline"
                size="sm"
                onClick={() =>
                  handleCopy(showNewPassword, t('settings.caldav.passwordLabel'))
                }
              >
                <Copy className="h-3.5 w-3.5" />
              </Button>
            </div>
            <Button
              variant="ghost"
              size="sm"
              className="mt-2"
              onClick={() => setShowNewPassword(null)}
            >
              {t('common.close')}
            </Button>
          </div>
        )}

        {/* Create new */}
        <div className="flex items-end gap-2">
          <div className="flex-1">
            <Label htmlFor="pw-label" className="text-xs">
              {t('settings.caldav.labelField')}
            </Label>
            <Input
              id="pw-label"
              placeholder={t('settings.caldav.labelPlaceholder')}
              value={newLabel}
              onChange={(e) => setNewLabel(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleCreatePassword()}
            />
          </div>
          <Button
            onClick={handleCreatePassword}
            disabled={createPassword.isPending || !newLabel.trim()}
            size="sm"
          >
            <Plus className="h-3.5 w-3.5 mr-1" />
            {t('settings.caldav.createPassword')}
          </Button>
        </div>
      </section>

      <Separator className="mb-8" />

      {/* Setup wizard */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Shield className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">
            {t('settings.caldav.setupGuideTitle')}
          </h3>
        </div>

        <div className="space-y-2">
          {/* Thunderbird */}
          <ClientGuide
            icon={<Globe className="h-4 w-4" />}
            name="Mozilla Thunderbird"
            expanded={expandedClient === 'thunderbird'}
            onToggle={() => toggleClient('thunderbird')}
          >
            <div className="space-y-3 text-sm">
              <div>
                <p className="font-medium mb-1">{t('settings.caldav.guide.addCalendar')}</p>
                <ol className="list-decimal list-inside space-y-1 text-muted-foreground">
                  <li>{t('settings.caldav.guide.thunderbird.calStep1')}</li>
                  <li>
                    {t('settings.caldav.guide.enterUrl')}
                    <URLField
                      url={caldavURL}
                      onCopy={() => handleCopy(caldavURL, 'CalDAV-URL')}
                    />
                  </li>
                  <li>{t('settings.caldav.guide.usernameHint')}</li>
                  <li>{t('settings.caldav.guide.passwordHint')}</li>
                </ol>
              </div>
              <div>
                <p className="font-medium mb-1">{t('settings.caldav.guide.addAddressbook')}</p>
                <ol className="list-decimal list-inside space-y-1 text-muted-foreground">
                  <li>{t('settings.caldav.guide.thunderbird.abStep1')}</li>
                  <li>
                    {t('settings.caldav.guide.enterUrl')}
                    <URLField
                      url={carddavURL}
                      onCopy={() => handleCopy(carddavURL, 'CardDAV-URL')}
                    />
                  </li>
                </ol>
              </div>
            </div>
          </ClientGuide>

          {/* macOS Calendar */}
          <ClientGuide
            icon={<Monitor className="h-4 w-4" />}
            name="macOS Kalender"
            expanded={expandedClient === 'macos'}
            onToggle={() => toggleClient('macos')}
          >
            <div className="space-y-3 text-sm">
              <ol className="list-decimal list-inside space-y-1 text-muted-foreground">
                <li>{t('settings.caldav.guide.macos.step1')}</li>
                <li>{t('settings.caldav.guide.macos.step2')}</li>
                <li>
                  {t('settings.caldav.guide.macos.step3')}
                  <URLField
                    url={serverURL}
                    onCopy={() => handleCopy(serverURL, 'Server-URL')}
                  />
                </li>
                <li>{t('settings.caldav.guide.usernameHint')}</li>
                <li>{t('settings.caldav.guide.passwordHint')}</li>
              </ol>
              <p className="text-xs text-muted-foreground">
                {t('settings.caldav.guide.macos.pushNote')}
              </p>
            </div>
          </ClientGuide>

          {/* Outlook */}
          <ClientGuide
            icon={<Smartphone className="h-4 w-4" />}
            name="Microsoft Outlook"
            expanded={expandedClient === 'outlook'}
            onToggle={() => toggleClient('outlook')}
          >
            <div className="space-y-3 text-sm">
              <div className="rounded-md bg-warning-light border border-warning/30 p-2 text-xs text-warning">
                {t('settings.caldav.guide.outlook.note')}{' '}
                <span className="font-medium">
                  caldavsynchronizer.org
                </span>
              </div>
              <ol className="list-decimal list-inside space-y-1 text-muted-foreground">
                <li>{t('settings.caldav.guide.outlook.step1')}</li>
                <li>{t('settings.caldav.guide.outlook.step2')}</li>
                <li>
                  {t('settings.caldav.guide.outlook.step3')}
                  <URLField
                    url={caldavURL}
                    onCopy={() => handleCopy(caldavURL, 'CalDAV-URL')}
                  />
                </li>
                <li>{t('settings.caldav.guide.usernameHint')}</li>
                <li>{t('settings.caldav.guide.passwordHint')}</li>
              </ol>
            </div>
          </ClientGuide>
        </div>

        {/* User ID display */}
        <div className="mt-4 rounded-md border border-border p-3">
          <p className="text-xs text-muted-foreground mb-1">
            {t('settings.caldav.userIdHint')}
          </p>
          <div className="flex items-center gap-2">
            <code className="text-sm font-mono bg-muted px-2 py-1 rounded select-all flex-1">
              {userId || '...'}
            </code>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => handleCopy(userId, t('settings.caldav.userIdLabel'))}
              disabled={!userId}
            >
              <Copy className="h-3.5 w-3.5" />
            </Button>
          </div>
        </div>
      </section>

      <Separator className="mb-8" />

      {/* Connection test */}
      <ConnectionTestSection
        orgEnabled={status?.org_enabled ?? false}
        userEnabled={status?.user_enabled ?? false}
      />
    </div>
  )
}

// ---------------------------------------------------------------------------
// Connection test section
// ---------------------------------------------------------------------------

function ConnectionTestSection({
  orgEnabled,
  userEnabled,
}: {
  orgEnabled: boolean
  userEnabled: boolean
}) {
  const { t } = useTranslation()
  const testMutation = useTestCalDAVConnection()

  const handleTest = () => {
    testMutation.mutate()
  }

  return (
    <section>
      <div className="flex items-center gap-2 mb-4">
        <RefreshCw className="h-4 w-4 text-muted-foreground" />
        <h3 className="text-sm font-medium text-foreground">
          {t('settings.caldav.connectionTestTitle')}
        </h3>
      </div>

      <div className="flex items-center gap-3 mb-3">
        <Button
          variant="outline"
          size="sm"
          onClick={handleTest}
          disabled={testMutation.isPending || !orgEnabled || !userEnabled}
        >
          {testMutation.isPending ? (
            <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
          ) : (
            <RefreshCw className="h-3.5 w-3.5 mr-1.5" />
          )}
          {testMutation.isPending ? t('settings.caldav.testing') : t('settings.caldav.testConnection')}
        </Button>
        <p className="text-xs text-muted-foreground">
          {t('settings.caldav.testDesc')}
        </p>
      </div>

      {(!orgEnabled || !userEnabled) && (
        <div className="rounded-md border border-warning/30 bg-warning-light px-3 py-2 text-xs text-warning">
          {!orgEnabled
            ? t('settings.caldav.orgDisabledAdmin')
            : t('settings.caldav.enablePersonalFirst')}
        </div>
      )}

      {testMutation.data && (
        <div
          className={`rounded-md border px-3 py-2.5 text-xs ${
            testMutation.data.success
              ? 'border-success/30 bg-success-light text-success'
              : 'border-destructive/30 bg-error-light text-destructive'
          }`}
        >
          <div className="flex items-center gap-2 mb-1.5">
            {testMutation.data.success ? (
              <CheckCircle className="h-4 w-4" />
            ) : (
              <XCircle className="h-4 w-4" />
            )}
            <span className="font-medium">
              {testMutation.data.success
                ? t('settings.caldav.connectionSuccess')
                : t('settings.caldav.connectionFailed')}
            </span>
          </div>
          <div className="space-y-0.5 ml-6">
            <p className="flex items-center gap-1.5">
              {testMutation.data.caldav_reachable ? (
                <CheckCircle className="h-3 w-3" />
              ) : (
                <XCircle className="h-3 w-3" />
              )}
              CalDAV {testMutation.data.caldav_reachable ? t('settings.caldav.reachable') : t('settings.caldav.notReachable')}
            </p>
            <p className="flex items-center gap-1.5">
              {testMutation.data.carddav_reachable ? (
                <CheckCircle className="h-3 w-3" />
              ) : (
                <XCircle className="h-3 w-3" />
              )}
              CardDAV {testMutation.data.carddav_reachable ? t('settings.caldav.reachable') : t('settings.caldav.notReachable')}
            </p>
            {testMutation.data.message && (
              <p className="mt-1 text-muted-foreground">{testMutation.data.message}</p>
            )}
          </div>
        </div>
      )}

      {testMutation.isError && (
        <div className="rounded-md border border-destructive/30 bg-error-light px-3 py-2 text-xs text-destructive flex items-center gap-2">
          <XCircle className="h-4 w-4" />
          {t('settings.caldav.error')}: {(testMutation.error as Error)?.message ?? t('settings.caldav.testFailed')}
        </div>
      )}
    </section>
  )
}

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

function ClientGuide({
  icon,
  name,
  expanded,
  onToggle,
  children,
}: {
  icon: React.ReactNode
  name: string
  expanded: boolean
  onToggle: () => void
  children: React.ReactNode
}) {
  return (
    <div className="border border-border rounded-md">
      <button
        onClick={onToggle}
        className="flex items-center gap-2 w-full px-3 py-2.5 text-sm font-medium text-left hover:bg-accent/50 transition-colors"
      >
        {icon}
        <span className="flex-1">{name}</span>
        {expanded ? (
          <ChevronDown className="h-4 w-4 text-muted-foreground" />
        ) : (
          <ChevronRight className="h-4 w-4 text-muted-foreground" />
        )}
      </button>
      {expanded && <div className="px-3 pb-3 pt-1">{children}</div>}
    </div>
  )
}

function URLField({
  url,
  onCopy,
}: {
  url: string
  onCopy: () => void
}) {
  const { t } = useTranslation()
  return (
    <span className="inline-flex items-center gap-1 ml-1">
      <code className="text-xs bg-muted px-1.5 py-0.5 rounded font-mono break-all">
        {url}
      </code>
      <button
        onClick={(e) => {
          e.stopPropagation()
          onCopy()
        }}
        className="text-muted-foreground hover:text-foreground"
        title={t('settings.caldav.copyUrl')}
      >
        <Copy className="h-3 w-3" />
      </button>
    </span>
  )
}

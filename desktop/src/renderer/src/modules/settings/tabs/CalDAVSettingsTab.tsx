/**
 * CalDAV/CardDAV settings tab: app-specific password management,
 * per-client setup wizard, and connection test.
 *
 * All labels in German (Deutschland-First).
 */
import { useState } from 'react'
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
      toast.error('Bitte geben Sie eine Bezeichnung ein')
      return
    }
    const result = await createPassword.mutateAsync(newLabel.trim())
    setShowNewPassword(result.password)
    setNewLabel('')
    toast.success('App-Passwort erstellt')
  }

  const handleCopy = (text: string, label: string) => {
    navigator.clipboard.writeText(text)
    toast.success(`${label} kopiert`)
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
        Kalender und Kontakte mit externen Anwendungen synchronisieren
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
              <p className="text-sm font-medium">Organisation</p>
              <p className="text-xs text-muted-foreground">
                {status?.org_enabled
                  ? 'CalDAV/CardDAV ist organisationsweit aktiviert'
                  : 'CalDAV/CardDAV ist organisationsweit deaktiviert (Admin-Einstellung)'}
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
              <p className="text-sm font-medium">Persönlicher Zugang</p>
              <p className="text-xs text-muted-foreground">
                CalDAV/CardDAV für Ihren Account aktivieren
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
            App-spezifische Passwörter
          </h3>
        </div>

        <p className="text-xs text-muted-foreground mb-4">
          Erstellen Sie separate Passwörter für jede Anwendung. Ihr
          Haupt-Passwort wird nicht verwendet.
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
                    {pw.password_prefix}... | Erstellt:{' '}
                    {new Date(pw.created_at).toLocaleDateString('de-DE')}
                    {pw.last_used_at &&
                      ` | Zuletzt: ${new Date(pw.last_used_at).toLocaleDateString('de-DE')}`}
                  </p>
                </div>
                <div className="flex items-center gap-2 ml-2">
                  {pw.revoked_at ? (
                    <span className="text-xs text-destructive font-medium">
                      Widerrufen
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
            Noch keine App-Passwörter erstellt.
          </p>
        )}

        {/* New password dialog shown inline */}
        {showNewPassword && (
          <div className="border border-warning/30 bg-warning-light rounded-md p-4 mb-4">
            <p className="text-sm font-medium text-warning mb-2">
              Dieses Passwort wird nur einmal angezeigt!
            </p>
            <div className="flex items-center gap-2">
              <code className="flex-1 text-sm bg-background border border-border rounded px-3 py-2 font-mono select-all break-all">
                {showNewPassword}
              </code>
              <Button
                variant="outline"
                size="sm"
                onClick={() =>
                  handleCopy(showNewPassword, 'Passwort')
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
              Schliessen
            </Button>
          </div>
        )}

        {/* Create new */}
        <div className="flex items-end gap-2">
          <div className="flex-1">
            <Label htmlFor="pw-label" className="text-xs">
              Bezeichnung
            </Label>
            <Input
              id="pw-label"
              placeholder="z.B. Thunderbird, macOS Kalender"
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
            Neues Passwort erstellen
          </Button>
        </div>
      </section>

      <Separator className="mb-8" />

      {/* Setup wizard */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Shield className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">
            Einrichtungsanleitung
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
                <p className="font-medium mb-1">Kalender hinzufügen:</p>
                <ol className="list-decimal list-inside space-y-1 text-muted-foreground">
                  <li>
                    Einstellungen &gt; Kalender &gt; Neues Netzwerk-Kalender
                    &gt; CalDAV
                  </li>
                  <li>
                    URL eingeben:
                    <URLField
                      url={caldavURL}
                      onCopy={() => handleCopy(caldavURL, 'CalDAV-URL')}
                    />
                  </li>
                  <li>Benutzername: Ihre Benutzer-ID (siehe unten)</li>
                  <li>Passwort: App-spezifisches Passwort (oben erstellen)</li>
                </ol>
              </div>
              <div>
                <p className="font-medium mb-1">Adressbuch hinzufügen:</p>
                <ol className="list-decimal list-inside space-y-1 text-muted-foreground">
                  <li>
                    Adressbuch &gt; Neues CardDAV-Adressbuch
                  </li>
                  <li>
                    URL eingeben:
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
                <li>
                  Systemeinstellungen &gt; Internetaccounts &gt; Anderen Account
                  hinzufügen &gt; CalDAV-Account
                </li>
                <li>Account-Typ: Manuell</li>
                <li>
                  Server:
                  <URLField
                    url={serverURL}
                    onCopy={() => handleCopy(serverURL, 'Server-URL')}
                  />
                </li>
                <li>Benutzername: Ihre Benutzer-ID (siehe unten)</li>
                <li>Passwort: App-spezifisches Passwort (oben erstellen)</li>
              </ol>
              <p className="text-xs text-muted-foreground">
                macOS unterstuetzt Push-Benachrichtigungen. Änderungen werden
                automatisch synchronisiert.
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
                Hinweis: Microsoft Outlook unterstuetzt CalDAV nicht nativ.
                Installieren Sie das kostenlose Plugin &quot;CalDav
                Synchronizer&quot; von{' '}
                <span className="font-medium">
                  caldavsynchronizer.org
                </span>
              </div>
              <ol className="list-decimal list-inside space-y-1 text-muted-foreground">
                <li>CalDav Synchronizer installieren und Outlook neustarten</li>
                <li>CalDav Synchronizer &gt; Neues Profil &gt; CalDAV</li>
                <li>
                  DAV-URL:
                  <URLField
                    url={caldavURL}
                    onCopy={() => handleCopy(caldavURL, 'CalDAV-URL')}
                  />
                </li>
                <li>Benutzername: Ihre Benutzer-ID (siehe unten)</li>
                <li>Passwort: App-spezifisches Passwort (oben erstellen)</li>
              </ol>
            </div>
          </ClientGuide>
        </div>

        {/* User ID display */}
        <div className="mt-4 rounded-md border border-border p-3">
          <p className="text-xs text-muted-foreground mb-1">
            Ihre Benutzer-ID (als Benutzername verwenden):
          </p>
          <div className="flex items-center gap-2">
            <code className="text-sm font-mono bg-muted px-2 py-1 rounded select-all flex-1">
              {userId || '...'}
            </code>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => handleCopy(userId, 'Benutzer-ID')}
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
  const testMutation = useTestCalDAVConnection()

  const handleTest = () => {
    testMutation.mutate()
  }

  return (
    <section>
      <div className="flex items-center gap-2 mb-4">
        <RefreshCw className="h-4 w-4 text-muted-foreground" />
        <h3 className="text-sm font-medium text-foreground">
          Verbindungstest
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
          {testMutation.isPending ? 'Teste...' : 'Verbindung testen'}
        </Button>
        <p className="text-xs text-muted-foreground">
          Prüft ob CalDAV/CardDAV korrekt eingerichtet ist
        </p>
      </div>

      {(!orgEnabled || !userEnabled) && (
        <div className="rounded-md border border-warning/30 bg-warning-light px-3 py-2 text-xs text-warning">
          {!orgEnabled
            ? 'CalDAV/CardDAV ist organisationsweit deaktiviert. Bitte wenden Sie sich an Ihren Administrator.'
            : 'Aktivieren Sie zuerst Ihren persönlichen Zugang (oben), um den Verbindungstest ausführen zu können.'}
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
                ? 'Verbindung erfolgreich'
                : 'Verbindung fehlgeschlagen'}
            </span>
          </div>
          <div className="space-y-0.5 ml-6">
            <p className="flex items-center gap-1.5">
              {testMutation.data.caldav_reachable ? (
                <CheckCircle className="h-3 w-3" />
              ) : (
                <XCircle className="h-3 w-3" />
              )}
              CalDAV {testMutation.data.caldav_reachable ? 'erreichbar' : 'nicht erreichbar'}
            </p>
            <p className="flex items-center gap-1.5">
              {testMutation.data.carddav_reachable ? (
                <CheckCircle className="h-3 w-3" />
              ) : (
                <XCircle className="h-3 w-3" />
              )}
              CardDAV {testMutation.data.carddav_reachable ? 'erreichbar' : 'nicht erreichbar'}
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
          Fehler: {(testMutation.error as Error)?.message ?? 'Verbindungstest fehlgeschlagen'}
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
        title="URL kopieren"
      >
        <Copy className="h-3 w-3" />
      </button>
    </span>
  )
}

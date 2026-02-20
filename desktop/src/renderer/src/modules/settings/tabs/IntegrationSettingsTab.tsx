import { useState } from 'react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Separator } from '@/components/ui/separator'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  Plug,
  Plus,
  Trash2,
  Check,
  X,
  Loader2,
  Send,
  Link,
  Unlink,
  Hash,
  MessageSquare,
} from 'lucide-react'
import { toast } from 'sonner'
import { ConfirmDialog } from '@/components/shared'
import {
  useIntegrationConfigs,
  useCreateIntegration,
  useDeleteIntegration,
  useTestIntegration,
  useAccountLinks,
  useLinkAccount,
  useUnlinkAccount,
  useChannelMappings,
  useCreateChannelMapping,
  useDeleteChannelMapping,
} from '@/api/hooks/useIntegrations'
import type { IntegrationConfig, IntegrationPlatform } from '@/api/integration-types'

const PLATFORM_META: Record<IntegrationPlatform, { label: string; icon: typeof MessageSquare; color: string }> = {
  teams: { label: 'Microsoft Teams', icon: MessageSquare, color: 'text-blue-600' },
  slack: { label: 'Slack', icon: Hash, color: 'text-purple-600' },
  custom_webhook: { label: 'Webhook', icon: Plug, color: 'text-orange-500' },
}

const STATUS_BADGE: Record<string, { label: string; cls: string }> = {
  active: { label: 'Aktiv', cls: 'bg-success-light text-success' },
  inactive: { label: 'Inaktiv', cls: 'bg-secondary text-muted-foreground' },
  error: { label: 'Fehler', cls: 'bg-error-light text-error' },
}

const MODULE_OPTIONS = [
  'CRM', 'Kalender', 'Finanzen', 'Team', 'Projekte', 'Aufgaben', 'Chat',
]

export function IntegrationSettingsTab() {
  const { data: configs, isLoading } = useIntegrationConfigs()
  const createMutation = useCreateIntegration()
  const deleteMutation = useDeleteIntegration()
  const testMutation = useTestIntegration()
  const { data: accountLinks } = useAccountLinks()
  const linkMutation = useLinkAccount()
  const unlinkMutation = useUnlinkAccount()

  const [showAddDialog, setShowAddDialog] = useState(false)
  const [newPlatform, setNewPlatform] = useState<IntegrationPlatform>('slack')
  const [newName, setNewName] = useState('')
  const [newWebhookUrl, setNewWebhookUrl] = useState('')
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null)

  // Account linking
  const [linkToken, setLinkToken] = useState('')
  const [linkPlatform, setLinkPlatform] = useState<string | null>(null)

  // Channel mapping (expanded per integration)
  const [expandedMapping, setExpandedMapping] = useState<string | null>(null)

  const handleCreate = () => {
    if (!newName.trim()) return
    createMutation.mutate(
      {
        platform: newPlatform,
        name: newName.trim(),
        webhook_url: newWebhookUrl.trim() || undefined,
      },
      {
        onSuccess: () => {
          toast.success('Integration erstellt')
          setShowAddDialog(false)
          setNewName('')
          setNewWebhookUrl('')
        },
      },
    )
  }

  const handleTest = (id: string) => {
    testMutation.mutate(id, {
      onSuccess: (result) => {
        if (result.success) {
          toast.success('Verbindung erfolgreich')
        } else {
          toast.error(result.message ?? 'Test fehlgeschlagen')
        }
      },
      onError: (err: Error) => toast.error(err.message),
    })
  }

  const handleDelete = () => {
    if (!deleteTarget) return
    deleteMutation.mutate(deleteTarget, {
      onSuccess: () => {
        toast.success('Integration geloescht')
        setDeleteTarget(null)
      },
    })
  }

  const handleLink = (platform: string) => {
    if (!linkToken.trim()) return
    linkMutation.mutate(
      { platform, token: linkToken.trim() },
      {
        onSuccess: () => {
          toast.success('Account verknuepft')
          setLinkToken('')
          setLinkPlatform(null)
        },
      },
    )
  }

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 text-muted-foreground py-12">
        <Loader2 className="h-4 w-4 animate-spin" />
        <span className="text-sm">Integrationen werden geladen...</span>
      </div>
    )
  }

  return (
    <div className="max-w-2xl">
      <h2 className="text-foreground mb-1">Integrationen</h2>
      <p className="text-sm text-muted-foreground mb-6">
        Externe Plattformen verbinden, Accounts verknuepfen und Channel-Zuordnungen verwalten
      </p>

      {/* ── Platform Configs ─────────────────────────── */}
      <section className="mb-8">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <Plug className="h-4 w-4 text-muted-foreground" />
            <h3 className="text-sm font-medium text-foreground">Plattformen</h3>
          </div>
          <Button size="sm" onClick={() => setShowAddDialog(true)}>
            <Plus className="mr-1.5 h-4 w-4" />
            Hinzufuegen
          </Button>
        </div>

        {(!configs || configs.length === 0) ? (
          <p className="text-xs text-muted-foreground italic">Noch keine Integrationen konfiguriert</p>
        ) : (
          <div className="space-y-3">
            {configs.map((cfg: IntegrationConfig) => {
              const meta = PLATFORM_META[cfg.platform] ?? PLATFORM_META.custom_webhook
              const Icon = meta.icon
              const badge = STATUS_BADGE[cfg.status] ?? STATUS_BADGE.inactive
              return (
                <div key={cfg.id} className="rounded-lg border border-border bg-card p-4">
                  <div className="flex items-start gap-3">
                    <div className={`flex h-10 w-10 items-center justify-center rounded-lg bg-secondary shrink-0`}>
                      <Icon className={`h-5 w-5 ${meta.color}`} />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <p className="text-sm font-medium text-foreground">{cfg.name}</p>
                        <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${badge.cls}`}>
                          {badge.label}
                        </span>
                      </div>
                      <p className="text-xs text-muted-foreground">{meta.label}</p>
                      {cfg.last_tested_at && (
                        <p className="text-[10px] text-muted-foreground mt-0.5">
                          Zuletzt getestet: {new Date(cfg.last_tested_at).toLocaleString('de-DE')}
                        </p>
                      )}
                    </div>
                    <div className="flex items-center gap-1 shrink-0">
                      <Button
                        variant="outline"
                        size="sm"
                        className="h-8 text-xs"
                        onClick={() => handleTest(cfg.id)}
                        disabled={testMutation.isPending}
                      >
                        <Send className="mr-1 h-3 w-3" />
                        Test
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        className="h-8 text-xs"
                        onClick={() => setExpandedMapping(expandedMapping === cfg.id ? null : cfg.id)}
                      >
                        <Hash className="mr-1 h-3 w-3" />
                        Channels
                      </Button>
                      <button
                        onClick={() => setDeleteTarget(cfg.id)}
                        className="rounded p-1.5 text-muted-foreground hover:text-error transition-colors"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  </div>

                  {/* Channel mappings inline */}
                  {expandedMapping === cfg.id && (
                    <div className="mt-3 pt-3 border-t border-border">
                      <ChannelMappingSection integrationId={cfg.id} />
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        )}
      </section>

      <Separator className="mb-8" />

      {/* ── Account Linking ───────────────────────────── */}
      <section>
        <div className="flex items-center gap-2 mb-4">
          <Link className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">Account-Verknuepfung</h3>
        </div>
        <p className="text-xs text-muted-foreground mb-3">
          Verknuepfe deinen KMU Hub Account mit externen Plattformen
        </p>

        <div className="space-y-2">
          {(['teams', 'slack'] as IntegrationPlatform[]).map((platform) => {
            const meta = PLATFORM_META[platform]
            const Icon = meta.icon
            const link = accountLinks?.find((l) => l.platform === platform)
            const isLinked = link?.linked ?? false

            return (
              <div key={platform} className="flex items-center gap-3 rounded-lg border border-border bg-card p-3">
                <Icon className={`h-5 w-5 ${meta.color} shrink-0`} />
                <div className="flex-1 min-w-0">
                  <p className="text-sm text-foreground">{meta.label}</p>
                  <p className="text-xs text-muted-foreground">
                    {isLinked ? 'Verknuepft' : 'Nicht verknuepft'}
                  </p>
                </div>

                {isLinked ? (
                  <Button
                    variant="outline"
                    size="sm"
                    className="text-xs"
                    onClick={() =>
                      unlinkMutation.mutate(platform, {
                        onSuccess: () => toast.success('Verknuepfung aufgehoben'),
                      })
                    }
                    disabled={unlinkMutation.isPending}
                  >
                    <Unlink className="mr-1 h-3 w-3" />
                    Trennen
                  </Button>
                ) : linkPlatform === platform ? (
                  <div className="flex items-center gap-2">
                    <Input
                      value={linkToken}
                      onChange={(e) => setLinkToken(e.target.value)}
                      placeholder="Token"
                      className="w-40 h-8 text-xs font-mono"
                    />
                    <Button size="sm" className="h-8 text-xs" onClick={() => handleLink(platform)}>
                      <Check className="h-3 w-3" />
                    </Button>
                    <Button variant="outline" size="sm" className="h-8 text-xs" onClick={() => setLinkPlatform(null)}>
                      <X className="h-3 w-3" />
                    </Button>
                  </div>
                ) : (
                  <Button
                    variant="outline"
                    size="sm"
                    className="text-xs"
                    onClick={() => setLinkPlatform(platform)}
                  >
                    <Link className="mr-1 h-3 w-3" />
                    Verknuepfen
                  </Button>
                )}
              </div>
            )
          })}
        </div>
      </section>

      {/* ── Add Integration Dialog ────────────────────── */}
      <Dialog open={showAddDialog} onOpenChange={setShowAddDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Integration hinzufuegen</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-1.5">
              <Label>Plattform</Label>
              <select
                value={newPlatform}
                onChange={(e) => setNewPlatform(e.target.value as IntegrationPlatform)}
                className="w-full rounded-lg border border-border bg-input-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              >
                <option value="slack">Slack</option>
                <option value="teams">Microsoft Teams</option>
                <option value="custom_webhook">Custom Webhook</option>
              </select>
            </div>
            <div className="space-y-1.5">
              <Label>Name</Label>
              <Input
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                placeholder="z.B. Firma-Slack"
              />
            </div>
            <div className="space-y-1.5">
              <Label>Webhook-URL</Label>
              <Input
                value={newWebhookUrl}
                onChange={(e) => setNewWebhookUrl(e.target.value)}
                placeholder="https://hooks.slack.com/..."
                className="font-mono text-xs"
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAddDialog(false)}>Abbrechen</Button>
            <Button onClick={handleCreate} disabled={createMutation.isPending || !newName.trim()}>
              {createMutation.isPending && <Loader2 className="mr-1.5 h-4 w-4 animate-spin" />}
              Erstellen
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Confirm delete */}
      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => { if (!open) setDeleteTarget(null) }}
        title="Integration loeschen?"
        description="Alle Channel-Zuordnungen und Verknuepfungen werden ebenfalls entfernt."
        confirmLabel="Loeschen"
        variant="destructive"
        onConfirm={handleDelete}
      />
    </div>
  )
}

// ============================================================
// Channel Mapping Sub-Section
// ============================================================
function ChannelMappingSection({ integrationId }: { integrationId: string }) {
  const { data: mappings, isLoading } = useChannelMappings(integrationId)
  const createMapping = useCreateChannelMapping()
  const deleteMapping = useDeleteChannelMapping()

  const [showAdd, setShowAdd] = useState(false)
  const [channelId, setChannelId] = useState('')
  const [channelName, setChannelName] = useState('')
  const [selectedModules, setSelectedModules] = useState<string[]>([])

  const toggleModule = (mod: string) => {
    setSelectedModules((prev) =>
      prev.includes(mod) ? prev.filter((m) => m !== mod) : [...prev, mod],
    )
  }

  const handleCreate = () => {
    if (!channelId.trim() || !channelName.trim() || selectedModules.length === 0) {
      toast.error('Bitte alle Felder ausfuellen')
      return
    }
    createMapping.mutate(
      {
        integrationId,
        data: { channel_id: channelId.trim(), channel_name: channelName.trim(), modules: selectedModules },
      },
      {
        onSuccess: () => {
          toast.success('Channel-Zuordnung erstellt')
          setShowAdd(false)
          setChannelId('')
          setChannelName('')
          setSelectedModules([])
        },
      },
    )
  }

  if (isLoading) {
    return <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
  }

  return (
    <div className="space-y-2">
      <p className="text-xs font-medium text-muted-foreground">Channel-Zuordnungen</p>

      {(!mappings || mappings.length === 0) ? (
        <p className="text-xs text-muted-foreground italic">Keine Zuordnungen</p>
      ) : (
        <div className="space-y-1.5">
          {mappings.map((m) => (
            <div key={m.id} className="flex items-center gap-2 text-xs rounded border border-border px-2 py-1.5">
              <Hash className="h-3 w-3 text-muted-foreground shrink-0" />
              <span className="text-foreground font-medium">{m.channel_name}</span>
              <span className="text-muted-foreground">→ {m.modules.join(', ')}</span>
              <button
                onClick={() =>
                  deleteMapping.mutate(
                    { integrationId, mappingId: m.id },
                    { onSuccess: () => toast.success('Zuordnung entfernt') },
                  )
                }
                className="ml-auto text-muted-foreground hover:text-error transition-colors"
              >
                <Trash2 className="h-3 w-3" />
              </button>
            </div>
          ))}
        </div>
      )}

      {!showAdd ? (
        <Button variant="outline" size="sm" className="text-xs" onClick={() => setShowAdd(true)}>
          <Plus className="mr-1 h-3 w-3" />
          Zuordnung
        </Button>
      ) : (
        <div className="rounded border border-border bg-secondary/30 p-2 space-y-2">
          <div className="grid grid-cols-2 gap-2">
            <Input
              value={channelId}
              onChange={(e) => setChannelId(e.target.value)}
              placeholder="Channel-ID"
              className="text-xs h-7 font-mono"
            />
            <Input
              value={channelName}
              onChange={(e) => setChannelName(e.target.value)}
              placeholder="Channel-Name"
              className="text-xs h-7"
            />
          </div>
          <div className="flex flex-wrap gap-1.5">
            {MODULE_OPTIONS.map((mod) => (
              <button
                key={mod}
                onClick={() => toggleModule(mod)}
                className={`rounded-full px-2 py-0.5 text-[10px] font-medium transition-colors ${
                  selectedModules.includes(mod)
                    ? 'bg-primary text-primary-foreground'
                    : 'bg-secondary text-muted-foreground hover:bg-secondary/80'
                }`}
              >
                {mod}
              </button>
            ))}
          </div>
          <div className="flex gap-1.5">
            <Button size="sm" className="text-xs h-7" onClick={handleCreate} disabled={createMapping.isPending}>
              Erstellen
            </Button>
            <Button variant="outline" size="sm" className="text-xs h-7" onClick={() => setShowAdd(false)}>
              Abbrechen
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}

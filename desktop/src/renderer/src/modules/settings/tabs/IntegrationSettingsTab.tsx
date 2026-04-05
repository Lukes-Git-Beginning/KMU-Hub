/**
 * IntegrationSettingsTab — Grid of all available integrations with drill-down.
 *
 * Replaces the previous chat-only integration tab with a full grid showing
 * all 12 integrations across 5 categories. Click a card to drill down into
 * the config panel (generic or custom). Existing API-driven integrations
 * (Slack, Teams webhook, Custom Webhook) use ExistingIntegrationWrapper.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ArrowLeft, Plus, Send, Trash2, Hash, Loader2, Link, Unlink, Check, X } from 'lucide-react'
import { toast } from 'sonner'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
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
import {
  INTEGRATION_REGISTRY,
  CATEGORY_ORDER,
  CATEGORY_LABEL_KEYS,
  PLATFORM_MAP,
  type IntegrationDefinition,
} from '../integrations/integration-registry'
import { IntegrationCard } from '../integrations/IntegrationCard'
import { GenericIntegrationPanel } from '../integrations/GenericIntegrationPanel'
import { DATEVConfigPanel } from '../integrations/DATEVConfigPanel'
import { BexioConfigPanel } from '../integrations/BexioConfigPanel'
import { useIntegrationStore } from '@/stores/integrations'

// ---------------------------------------------------------------------------
// Constants for existing integrations
// ---------------------------------------------------------------------------

const PLATFORM_META: Record<string, { label: string; color: string }> = {
  teams: { label: 'Microsoft Teams', color: 'text-blue-600' },
  slack: { label: 'Slack', color: 'text-purple-600' },
  custom_webhook: { label: 'Webhook', color: 'text-orange-500' },
}

const MODULE_OPTIONS = [
  'CRM', 'Kalender', 'Finanzen', 'Team', 'Projekte', 'Aufgaben', 'Chat',
]

// ---------------------------------------------------------------------------
// Main Component
// ---------------------------------------------------------------------------

export function IntegrationSettingsTab() {
  const { t } = useTranslation()
  const [activeIntegrationId, setActiveIntegrationId] = useState<string | null>(null)

  // Existing API hooks
  const { data: apiConfigs } = useIntegrationConfigs()

  // New mock integration store
  const integrationStore = useIntegrationStore()

  // Determine status for each integration card
  const getCardStatus = (def: IntegrationDefinition) => {
    if (def.authMethod === 'existing') {
      const platformKey = PLATFORM_MAP[def.id]
      const match = apiConfigs?.find((c: IntegrationConfig) => c.platform === platformKey)
      return match?.status === 'active' ? 'connected' as const : 'disconnected' as const
    }
    return integrationStore.getStatus(def.id)
  }

  // Drill-down: render the appropriate panel
  if (activeIntegrationId) {
    const def = INTEGRATION_REGISTRY.find((d) => d.id === activeIntegrationId)
    if (!def) {
      setActiveIntegrationId(null)
      return null
    }

    const handleBack = () => setActiveIntegrationId(null)

    // Existing API-driven integrations
    if (def.authMethod === 'existing') {
      return (
        <ExistingIntegrationWrapper
          definition={def}
          onBack={handleBack}
        />
      )
    }

    // Custom panels
    if (def.id === 'datev-rechnungswesen') {
      return <DATEVConfigPanel onBack={handleBack} />
    }
    if (def.id === 'bexio') {
      return <BexioConfigPanel onBack={handleBack} />
    }

    // Generic panel
    return <GenericIntegrationPanel definition={def} onBack={handleBack} />
  }

  // Grid view
  return (
    <div className="max-w-4xl">
      <h2 className="text-foreground mb-1">{t('settings.integrations.title')}</h2>
      <p className="text-sm text-muted-foreground mb-6">
        {t('settings.integrations.subtitle')}
      </p>

      {CATEGORY_ORDER.map((cat) => {
        const integrations = INTEGRATION_REGISTRY.filter((d) => d.category === cat)
        if (integrations.length === 0) return null

        return (
          <section key={cat} className="mb-8">
            <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3">
              {t(CATEGORY_LABEL_KEYS[cat])}
            </h3>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
              {integrations.map((def) => (
                <IntegrationCard
                  key={def.id}
                  definition={def}
                  status={getCardStatus(def)}
                  onClick={() => setActiveIntegrationId(def.id)}
                />
              ))}
            </div>
          </section>
        )
      })}
    </div>
  )
}

// ---------------------------------------------------------------------------
// ExistingIntegrationWrapper — wraps existing API-driven integrations
// ---------------------------------------------------------------------------

function ExistingIntegrationWrapper({
  definition,
  onBack,
}: {
  definition: IntegrationDefinition
  onBack: () => void
}) {
  const { t } = useTranslation()
  const platformKey = PLATFORM_MAP[definition.id] as IntegrationPlatform | undefined
  const { data: configs } = useIntegrationConfigs()
  const testMutation = useTestIntegration()
  const deleteMutation = useDeleteIntegration()
  const createMutation = useCreateIntegration()
  const { data: accountLinks } = useAccountLinks()
  const linkMutation = useLinkAccount()
  const unlinkMutation = useUnlinkAccount()

  const [showAddDialog, setShowAddDialog] = useState(false)
  const [newName, setNewName] = useState('')
  const [newWebhookUrl, setNewWebhookUrl] = useState('')
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null)
  const [expandedMapping, setExpandedMapping] = useState<string | null>(null)
  const [linkToken, setLinkToken] = useState('')
  const [showLinkInput, setShowLinkInput] = useState(false)

  const Icon = definition.icon
  const platformConfigs = configs?.filter(
    (c: IntegrationConfig) => platformKey && c.platform === platformKey,
  ) ?? []

  const handleCreate = () => {
    if (!newName.trim() || !platformKey) return
    createMutation.mutate(
      { platform: platformKey, name: newName.trim(), webhook_url: newWebhookUrl.trim() || undefined },
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
        if (result.success) toast.success('Verbindung erfolgreich')
        else toast.error(result.message ?? 'Test fehlgeschlagen')
      },
      onError: (err: Error) => toast.error(err.message),
    })
  }

  const handleDelete = () => {
    if (!deleteTarget) return
    deleteMutation.mutate(deleteTarget, {
      onSuccess: () => {
        toast.success('Integration gelöscht')
        setDeleteTarget(null)
      },
    })
  }

  const handleLink = () => {
    if (!linkToken.trim() || !platformKey) return
    linkMutation.mutate(
      { platform: platformKey, token: linkToken.trim() },
      {
        onSuccess: () => {
          toast.success('Account verknüpft')
          setLinkToken('')
          setShowLinkInput(false)
        },
      },
    )
  }

  const accountLink = accountLinks?.find((l) => l.platform === platformKey)
  const isLinked = accountLink?.linked ?? false

  return (
    <div className="max-w-2xl">
      {/* Back */}
      <button
        onClick={onBack}
        className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors mb-4"
      >
        <ArrowLeft className="h-4 w-4" />
        Zurück zu Integrationen
      </button>

      {/* Header */}
      <div className="flex items-start gap-4 mb-6">
        <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-secondary">
          <Icon className={`h-6 w-6 ${definition.iconColor}`} />
        </div>
        <div className="flex-1">
          <h2 className="text-lg font-semibold text-foreground">{definition.name}</h2>
          <p className="text-sm text-muted-foreground mt-0.5">{definition.description}</p>
        </div>
        <Button size="sm" onClick={() => setShowAddDialog(true)}>
          <Plus className="mr-1.5 h-4 w-4" />
          Hinzufügen
        </Button>
      </div>

      {/* Existing configs */}
      {platformConfigs.length === 0 ? (
        <p className="text-sm text-muted-foreground italic mb-6">Noch keine Konfigurationen vorhanden</p>
      ) : (
        <div className="space-y-3 mb-6">
          {platformConfigs.map((cfg: IntegrationConfig) => {
            const meta = PLATFORM_META[cfg.platform] ?? { label: cfg.platform, color: '' }
            return (
              <div key={cfg.id} className="rounded-lg border border-border bg-card p-4">
                <div className="flex items-center gap-3">
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-foreground">{cfg.name}</p>
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
                      className="rounded p-1.5 text-muted-foreground hover:text-destructive transition-colors"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </div>
                </div>
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

      {/* Account Linking */}
      {platformKey && platformKey !== 'custom_webhook' && (
        <div className="border-t border-border pt-6 mb-6">
          <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3">Account-Verknüpfung</h3>
          <div className="flex items-center gap-3 rounded-lg border border-border p-3">
            <div className="flex-1">
              <p className="text-sm text-foreground">{isLinked ? 'Account verknüpft' : 'Nicht verknüpft'}</p>
            </div>
            {isLinked ? (
              <Button
                variant="outline"
                size="sm"
                className="text-xs"
                onClick={() =>
                  unlinkMutation.mutate(platformKey, {
                    onSuccess: () => toast.success('Verknüpfung aufgehoben'),
                  })
                }
                disabled={unlinkMutation.isPending}
              >
                <Unlink className="mr-1 h-3 w-3" />
                Trennen
              </Button>
            ) : showLinkInput ? (
              <div className="flex items-center gap-2">
                <Input
                  value={linkToken}
                  onChange={(e) => setLinkToken(e.target.value)}
                  placeholder="Token"
                  className="w-40 h-8 text-xs font-mono"
                />
                <Button size="sm" className="h-8 text-xs" onClick={handleLink}>
                  <Check className="h-3 w-3" />
                </Button>
                <Button variant="outline" size="sm" className="h-8 text-xs" onClick={() => setShowLinkInput(false)}>
                  <X className="h-3 w-3" />
                </Button>
              </div>
            ) : (
              <Button
                variant="outline"
                size="sm"
                className="text-xs"
                onClick={() => setShowLinkInput(true)}
              >
                <Link className="mr-1 h-3 w-3" />
                Verknüpfen
              </Button>
            )}
          </div>
        </div>
      )}

      {/* Add Integration Dialog */}
      <Dialog open={showAddDialog} onOpenChange={setShowAddDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{definition.name} hinzufügen</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-1.5">
              <Label>Name</Label>
              <Input
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                placeholder={`z.B. ${definition.name} - Firma`}
              />
            </div>
            <div className="space-y-1.5">
              <Label>Webhook-URL</Label>
              <Input
                value={newWebhookUrl}
                onChange={(e) => setNewWebhookUrl(e.target.value)}
                placeholder="https://..."
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
        title="Integration löschen?"
        description="Alle Channel-Zuordnungen werden ebenfalls entfernt."
        confirmLabel="Löschen"
        variant="destructive"
        onConfirm={handleDelete}
      />
    </div>
  )
}

// ---------------------------------------------------------------------------
// Channel Mapping Sub-Section (preserved from original)
// ---------------------------------------------------------------------------

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
      toast.error('Bitte alle Felder ausfüllen')
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
              <span className="text-muted-foreground">{'\u2192'} {m.modules.join(', ')}</span>
              <button
                onClick={() =>
                  deleteMapping.mutate(
                    { integrationId, mappingId: m.id },
                    { onSuccess: () => toast.success('Zuordnung entfernt') },
                  )
                }
                className="ml-auto text-muted-foreground hover:text-destructive transition-colors"
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

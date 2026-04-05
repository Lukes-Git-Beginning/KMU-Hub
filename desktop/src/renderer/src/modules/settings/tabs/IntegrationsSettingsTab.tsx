/**
 * Admin Integrations settings tab.
 *
 * Displays platform cards for Teams, Slack, Bexio, Lexware Office, and DATEV,
 * setup wizard dialogs, and an account linking section for the current user.
 *
 * All labels in German (Deutschland-First).
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { Link2, Unlink, Shield, Loader2 } from 'lucide-react'
import {
  useIntegrationConfigs,
  useUpdateIntegrationConfig,
  useAccountLinkStatus,
  useUnlinkAccount,
} from '@/api/hooks/useIntegration'
import type {
  Platform,
  IntegrationConfig,
} from '@/api/integration-types'
import {
  IntegrationCard,
  type ConnectionStatus,
} from '../integrations/IntegrationCard'
import { TeamsSetupWizard } from '../integrations/TeamsSetupWizard'
import { SlackSetupWizard } from '../integrations/SlackSetupWizard'
import { BexioSetupWizard } from '../integrations/BexioSetupWizard'
import { BexioSyncDashboard } from '../integrations/BexioSyncDashboard'
import { LexwareSetupWizard } from '../integrations/LexwareSetupWizard'
import { LexwareSyncDashboard } from '../integrations/LexwareSyncDashboard'
import { DatevSettingsPanel } from '../integrations/DatevSettingsPanel'
import { AccountLinkDialog } from '../integrations/AccountLinkDialog'
import {
  useBexioConnectionStatus,
  useBexioDisconnect,
} from '@/api/hooks/useBexio'
import {
  useLexwareConnectionStatus,
  useLexwareDisconnect,
} from '@/api/hooks/useLexware'
import {
  useDatevConnectionStatus,
} from '@/api/hooks/useDatevUpload'

// ---------------------------------------------------------------------------
// SVG icons for platform logos (inline to avoid external deps)
// ---------------------------------------------------------------------------

function TeamsIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-5 w-5" fill="currentColor">
      <path d="M19.2 6.4h-2.8c.2-.4.4-.9.4-1.4 0-1.7-1.3-3-3-3s-3 1.3-3 3c0 .5.1 1 .4 1.4H8.8c-.9 0-1.6.7-1.6 1.6v5.6c0 2.6 2.1 4.8 4.8 4.8h2c2.6 0 4.8-2.1 4.8-4.8V8c0-.9-.7-1.6-1.6-1.6zM13.8 4c.6 0 1 .4 1 1s-.4 1-1 1-1-.4-1-1 .4-1 1-1zm6 10c0 .9-.7 1.6-1.6 1.6h-.4c.1-.4.2-.8.2-1.2V8c0-.2 0-.4-.1-.6h.3c.9 0 1.6.7 1.6 1.6v5zm1-7.6c.8 0 1.4-.6 1.4-1.4s-.6-1.4-1.4-1.4-1.4.6-1.4 1.4.6 1.4 1.4 1.4z" />
    </svg>
  )
}

function SlackIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-5 w-5" fill="currentColor">
      <path d="M5.042 15.165a2.528 2.528 0 0 1-2.52 2.523A2.528 2.528 0 0 1 0 15.165a2.527 2.527 0 0 1 2.522-2.52h2.52v2.52zm1.271 0a2.527 2.527 0 0 1 2.521-2.52 2.527 2.527 0 0 1 2.521 2.52v6.313A2.528 2.528 0 0 1 8.834 24a2.528 2.528 0 0 1-2.521-2.522v-6.313zM8.834 5.042a2.528 2.528 0 0 1-2.521-2.52A2.528 2.528 0 0 1 8.834 0a2.528 2.528 0 0 1 2.521 2.522v2.52H8.834zm0 1.271a2.528 2.528 0 0 1 2.521 2.521 2.528 2.528 0 0 1-2.521 2.521H2.522A2.528 2.528 0 0 1 0 8.834a2.528 2.528 0 0 1 2.522-2.521h6.312zm10.122 2.521a2.528 2.528 0 0 1 2.522-2.521A2.528 2.528 0 0 1 24 8.834a2.528 2.528 0 0 1-2.522 2.521h-2.522V8.834zm-1.268 0a2.528 2.528 0 0 1-2.523 2.521 2.527 2.527 0 0 1-2.52-2.521V2.522A2.527 2.527 0 0 1 15.165 0a2.528 2.528 0 0 1 2.523 2.522v6.312zm-2.523 10.122a2.528 2.528 0 0 1 2.523 2.522A2.528 2.528 0 0 1 15.165 24a2.527 2.527 0 0 1-2.52-2.522v-2.522h2.52zm0-1.268a2.527 2.527 0 0 1-2.52-2.523 2.526 2.526 0 0 1 2.52-2.52h6.313A2.528 2.528 0 0 1 24 15.165a2.528 2.528 0 0 1-2.522 2.523h-6.313z" />
    </svg>
  )
}

// ---------------------------------------------------------------------------
// Helper to derive connection status from config
// ---------------------------------------------------------------------------

function getConnectionStatus(
  config: IntegrationConfig | undefined,
): ConnectionStatus {
  if (!config) return 'disconnected'
  if (config.is_active) return 'connected'
  return 'paused'
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function IntegrationsSettingsTab() {
  const { t } = useTranslation()
  const { data: configs, isLoading } = useIntegrationConfigs()
  const updateTeams = useUpdateIntegrationConfig('teams')
  const updateSlack = useUpdateIntegrationConfig('slack')

  // Bexio connection
  const { data: bexioConnection } = useBexioConnectionStatus()
  const bexioDisconnect = useBexioDisconnect()

  // Lexware connection
  const { data: lexwareConnection } = useLexwareConnectionStatus()
  const lexwareDisconnect = useLexwareDisconnect()

  // DATEV connection
  const { data: datevConnection } = useDatevConnectionStatus()

  // Wizard state
  const [teamsWizardOpen, setTeamsWizardOpen] = useState(false)
  const [slackWizardOpen, setSlackWizardOpen] = useState(false)
  const [bexioWizardOpen, setBexioWizardOpen] = useState(false)
  const [bexioDashboardOpen, setBexioDashboardOpen] = useState(false)
  const [lexwareWizardOpen, setLexwareWizardOpen] = useState(false)
  const [lexwareDashboardOpen, setLexwareDashboardOpen] = useState(false)
  const [datevPanelOpen, setDatevPanelOpen] = useState(false)

  // Account link dialog state
  const [linkPlatform, setLinkPlatform] = useState<Platform | null>(null)

  // Look up configs by platform
  const teamsConfig = configs?.find((c) => c.platform === 'teams')
  const slackConfig = configs?.find((c) => c.platform === 'slack')

  if (isLoading) {
    return (
      <div className="p-6">
        <div className="animate-pulse space-y-4">
          <div className="h-6 bg-muted rounded w-48" />
          <div className="h-4 bg-muted rounded w-96" />
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 mt-6">
            <div className="h-32 bg-muted rounded" />
            <div className="h-32 bg-muted rounded" />
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="max-w-3xl p-6">
      <h2 className="text-foreground mb-1">Integrationen</h2>
      <p className="text-sm text-muted-foreground mb-6">
        Teams, Slack & externe Dienste verwalten
      </p>

      {/* Admin notice */}
      <div className="flex items-center gap-2 rounded-md border border-border bg-muted/50 px-3 py-2 mb-6">
        <Shield className="h-4 w-4 text-muted-foreground shrink-0" />
        <p className="text-xs text-muted-foreground">
          Nur Administratoren können Integrationen konfigurieren. Alle
          Benutzer können ihr eigenes Konto verknüpfen.
        </p>
      </div>

      {/* Platform cards */}
      <section className="mb-8">
        <h3 className="text-sm font-medium text-foreground mb-3">
          Plattformen
        </h3>
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <IntegrationCard
            name="Microsoft Teams"
            description="Bot Framework Benachrichtigungen und Aktionen"
            icon={<TeamsIcon />}
            status={getConnectionStatus(teamsConfig)}
            isActive={teamsConfig?.is_active ?? false}
            onConfigure={() => setTeamsWizardOpen(true)}
            onToggle={(enabled) =>
              updateTeams.mutate({ is_active: enabled })
            }
            isToggling={updateTeams.isPending}
          />
          <IntegrationCard
            name="Slack"
            description="Block Kit Benachrichtigungen und Slash-Commands"
            icon={<SlackIcon />}
            status={getConnectionStatus(slackConfig)}
            isActive={slackConfig?.is_active ?? false}
            onConfigure={() => setSlackWizardOpen(true)}
            onToggle={(enabled) =>
              updateSlack.mutate({ is_active: enabled })
            }
            isToggling={updateSlack.isPending}
          />

          {/* Bexio integration */}
          <IntegrationCard
            name="Bexio"
            description="Buchhaltung & Rechnungen synchronisieren"
            icon={
              <span className="text-sm font-bold text-foreground">
                Bx
              </span>
            }
            status={
              bexioConnection?.connected
                ? 'connected'
                : 'disconnected'
            }
            isActive={bexioConnection?.connected ?? false}
            onConfigure={() =>
              bexioConnection?.connected
                ? setBexioDashboardOpen(true)
                : setBexioWizardOpen(true)
            }
            onToggle={() => {
              if (bexioConnection?.connected) {
                if (confirm('Bexio-Verbindung trennen?')) {
                  bexioDisconnect.mutate()
                }
              }
            }}
            isToggling={bexioDisconnect.isPending}
          />
          {/* Lexware Office integration */}
          <IntegrationCard
            name="Lexware Office"
            description="Kontakte, Rechnungen & Angebote synchronisieren"
            icon={
              <span className="text-sm font-bold text-foreground">
                Lx
              </span>
            }
            status={
              lexwareConnection?.connected
                ? 'connected'
                : 'disconnected'
            }
            isActive={lexwareConnection?.connected ?? false}
            onConfigure={() =>
              lexwareConnection?.connected
                ? setLexwareDashboardOpen(true)
                : setLexwareWizardOpen(true)
            }
            onToggle={() => {
              if (lexwareConnection?.connected) {
                if (confirm('Lexware-Verbindung trennen?')) {
                  lexwareDisconnect.mutate()
                }
              }
            }}
            isToggling={lexwareDisconnect.isPending}
          />

          {/* DATEV API integration */}
          <IntegrationCard
            name="DATEV"
            description="Buchungsdaten & Belege hochladen"
            icon={
              <span className="text-sm font-bold text-foreground">
                Dv
              </span>
            }
            status={
              datevConnection?.connected
                ? 'connected'
                : 'disconnected'
            }
            isActive={datevConnection?.connected ?? false}
            onConfigure={() => setDatevPanelOpen(true)}
            onToggle={() => {}}
            isToggling={false}
          />
        </div>
      </section>

      <Separator className="mb-8" />

      {/* Account linking section (visible to all users) */}
      <section>
        <h3 className="text-sm font-medium text-foreground mb-1">
          Kontoverknüpfung
        </h3>
        <p className="text-xs text-muted-foreground mb-4">
          Verknüpfen Sie Ihr externes Konto, um Benachrichtigungen direkt in
          Teams oder Slack zu erhalten.
        </p>

        <div className="space-y-2">
          <AccountLinkRow
            platform="teams"
            platformLabel="Microsoft Teams"
            onLink={() => setLinkPlatform('teams')}
          />
          <AccountLinkRow
            platform="slack"
            platformLabel="Slack"
            onLink={() => setLinkPlatform('slack')}
          />
        </div>
      </section>

      {/* Wizard dialogs */}
      <TeamsSetupWizard
        isOpen={teamsWizardOpen}
        onClose={() => setTeamsWizardOpen(false)}
        existingConfig={teamsConfig}
      />
      <SlackSetupWizard
        isOpen={slackWizardOpen}
        onClose={() => setSlackWizardOpen(false)}
        existingConfig={slackConfig}
      />

      {/* Bexio wizard and dashboard */}
      <BexioSetupWizard
        isOpen={bexioWizardOpen}
        onClose={() => setBexioWizardOpen(false)}
      />
      {bexioDashboardOpen && (
        <BexioSyncDashboard
          isOpen={bexioDashboardOpen}
          onClose={() => setBexioDashboardOpen(false)}
        />
      )}

      {/* Lexware wizard and dashboard */}
      <LexwareSetupWizard
        isOpen={lexwareWizardOpen}
        onClose={() => setLexwareWizardOpen(false)}
      />
      {lexwareDashboardOpen && (
        <LexwareSyncDashboard
          isOpen={lexwareDashboardOpen}
          onClose={() => setLexwareDashboardOpen(false)}
        />
      )}

      {/* DATEV settings panel */}
      {datevPanelOpen && (
        <DatevSettingsPanel
          isOpen={datevPanelOpen}
          onClose={() => setDatevPanelOpen(false)}
        />
      )}

      {/* Account link dialog */}
      {linkPlatform && (
        <AccountLinkDialog
          platform={linkPlatform}
          isOpen={!!linkPlatform}
          onClose={() => setLinkPlatform(null)}
        />
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Account link row sub-component
// ---------------------------------------------------------------------------

function AccountLinkRow({
  platform,
  platformLabel,
  onLink,
}: {
  platform: Platform
  platformLabel: string
  onLink: () => void
}) {
  const { data: linkStatus, isLoading } = useAccountLinkStatus(platform)
  const unlinkAccount = useUnlinkAccount(platform)

  return (
    <div className="flex items-center justify-between rounded-md border border-border p-3">
      <div>
        <p className="text-sm font-medium">{platformLabel}</p>
        {isLoading ? (
          <div className="animate-pulse h-3 bg-muted rounded w-32 mt-1" />
        ) : linkStatus?.linked ? (
          <p className="text-xs text-muted-foreground">
            Verknüpft als{' '}
            <span className="font-medium">
              {linkStatus.external_display_name}
            </span>
            {linkStatus.linked_at &&
              ` seit ${new Date(linkStatus.linked_at).toLocaleDateString('de-DE')}`}
          </p>
        ) : (
          <p className="text-xs text-muted-foreground">Nicht verknüpft</p>
        )}
      </div>
      <div>
        {linkStatus?.linked ? (
          <Button
            variant="ghost"
            size="sm"
            onClick={() => unlinkAccount.mutate()}
            disabled={unlinkAccount.isPending}
            className="text-destructive hover:text-destructive"
          >
            {unlinkAccount.isPending ? (
              <Loader2 className="h-3.5 w-3.5 mr-1 animate-spin" />
            ) : (
              <Unlink className="h-3.5 w-3.5 mr-1" />
            )}
            Verknüpfung aufheben
          </Button>
        ) : (
          <Button variant="outline" size="sm" onClick={onLink}>
            <Link2 className="h-3.5 w-3.5 mr-1" />
            Verknüpfen
          </Button>
        )}
      </div>
    </div>
  )
}

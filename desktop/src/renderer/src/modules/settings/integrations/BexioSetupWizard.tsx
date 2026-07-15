/**
 * 4-step Bexio integration setup wizard.
 *
 * Steps:
 * 1. OAuth Verbindung -- Connect to Bexio via OAuth
 * 2. Sync-Richtungen -- Configure sync types and intervals
 * 3. Feld-Zuordnung -- Field mapping editor for contacts
 * 4. Erster Sync -- Trigger initial sync and show results
 */
import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import {
  ArrowLeft,
  ArrowRight,
  CheckCircle,
  ExternalLink,
  Loader2,
  RefreshCw,
} from 'lucide-react'
import {
  useBexioGetAuthURL,
  useBexioConnectionStatus,
  useBexioTriggerSync,
  useBexioSyncStatus,
  useBexioUpdateSyncConfig,
} from '@/api/hooks/useBexio'
import type { BexioSyncConfig } from '@/api/bexio-types'
import { BexioFieldMappingEditor } from './BexioFieldMappingEditor'

interface BexioSetupWizardProps {
  isOpen: boolean
  onClose: () => void
}


const INTERVAL_OPTIONS = [5, 10, 15, 30, 60]
const POLL_INTERVAL_OPTIONS = [1, 5, 10, 15]

export function BexioSetupWizard({ isOpen, onClose }: BexioSetupWizardProps) {
  const { t } = useTranslation()

  const STEPS = [
    { label: t('settings.integrations.bexio.setup.step.connection'), number: 1 },
    { label: t('settings.integrations.bexio.setup.step.syncOptions'), number: 2 },
    { label: t('settings.integrations.bexio.setup.step.fieldMapping'), number: 3 },
    { label: t('settings.integrations.bexio.setup.step.initialSync'), number: 4 },
  ]

  const [step, setStep] = useState(1)
  const [syncConfig, setSyncConfig] = useState<BexioSyncConfig>({
    contact_sync_enabled: true,
    contact_sync_interval_minutes: 15,
    invoice_push_enabled: true,
    quote_push_enabled: true,
    payment_poll_enabled: true,
    payment_poll_interval_minutes: 5,
    invoice_pull_enabled: false,
    invoice_pull_interval_minutes: 15,
  })
  const [syncTriggered, setSyncTriggered] = useState(false)

  const getAuthURL = useBexioGetAuthURL()
  const { data: connectionStatus, refetch: refetchConnection } =
    useBexioConnectionStatus()
  const triggerSync = useBexioTriggerSync()
  const { data: syncStatus } = useBexioSyncStatus(syncTriggered)
  const updateConfig = useBexioUpdateSyncConfig()

  // Poll connection status while on step 1 waiting for OAuth
  const [polling, setPolling] = useState(false)

  useEffect(() => {
    if (!isOpen || step !== 1 || !polling) return
    const interval = setInterval(() => {
      refetchConnection()
    }, 2000)
    return () => clearInterval(interval)
  }, [isOpen, step, polling, refetchConnection])

   
  useEffect(() => {
    if (connectionStatus?.connected && polling) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- reset state on dependency change
      setPolling(false)
    }
  }, [connectionStatus?.connected, polling])

  const handleConnect = useCallback(async () => {
    try {
      const result = await getAuthURL.mutateAsync()
      window.open(result.authorization_url, '_blank')
      setPolling(true)
    } catch {
      // Error handled by hook toast
    }
  }, [getAuthURL])

  const handleNext = () => {
    setStep((s) => Math.min(s + 1, STEPS.length))
  }

  const handleBack = () => {
    setStep((s) => Math.max(s - 1, 1))
  }

  const handleTriggerSync = async () => {
    setSyncTriggered(true)
    await triggerSync.mutateAsync(undefined)
  }

  const handleFinish = async () => {
    try {
      await updateConfig.mutateAsync(syncConfig)
    } catch {
      // Error handled by hook toast
    }
    onClose()
    // Reset wizard state
    setStep(1)
    setSyncTriggered(false)
    setPolling(false)
  }

  const canProceed = () => {
    if (step === 1) return connectionStatus?.connected === true
    return true
  }

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-xl max-h-[85vh]">
        <DialogHeader>
          <DialogTitle>{t('settings.integrations.bexio.setup.title')}</DialogTitle>
          <DialogDescription>
            {step === STEPS.length
              ? t('settings.integrations.bexio.setup.descriptionLastStep')
              : t('settings.integrations.bexio.setup.descriptionStep', { step, total: STEPS.length })}
          </DialogDescription>
        </DialogHeader>

        {/* Step indicator */}
        <div className="flex items-center gap-1 mb-2">
          {STEPS.map((s) => (
            <div
              key={s.number}
              className={`flex items-center gap-1.5 ${
                s.number < STEPS.length ? 'flex-1' : ''
              }`}
            >
              <div
                className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-xs font-medium transition-colors ${
                  s.number === step
                    ? 'bg-primary text-primary-foreground'
                    : s.number < step
                      ? 'bg-primary/20 text-primary'
                      : 'bg-muted text-muted-foreground'
                }`}
              >
                {s.number < step ? (
                  <CheckCircle className="h-4 w-4" />
                ) : (
                  s.number
                )}
              </div>
              <span
                className={`text-xs hidden sm:inline ${
                  s.number === step
                    ? 'text-foreground font-medium'
                    : 'text-muted-foreground'
                }`}
              >
                {s.label}
              </span>
              {s.number < STEPS.length && (
                <div className="flex-1 h-px bg-border mx-1" />
              )}
            </div>
          ))}
        </div>

        {/* Step content — scrolls; header + stepper + nav stay pinned */}
        <div className="min-h-[200px] max-h-[55vh] overflow-y-auto">
          {step === 1 && (
            <StepOAuth
              connected={connectionStatus?.connected ?? false}
              orgName={connectionStatus?.org_name}
              onConnect={handleConnect}
              isConnecting={getAuthURL.isPending}
              isPolling={polling}
            />
          )}
          {step === 2 && (
            <StepSyncConfig
              config={syncConfig}
              onChange={setSyncConfig}
            />
          )}
          {step === 3 && <StepFieldMapping />}
          {step === 4 && (
            <StepInitialSync
              onTrigger={handleTriggerSync}
              isSyncing={triggerSync.isPending}
              syncTriggered={syncTriggered}
              syncStatus={syncStatus}
            />
          )}
        </div>

        {/* Navigation buttons */}
        <div className="flex items-center justify-between pt-2 border-t border-border">
          <Button
            variant="ghost"
            size="sm"
            onClick={handleBack}
            disabled={step === 1}
          >
            <ArrowLeft className="h-3.5 w-3.5 mr-1" />
            {t('common.back')}
          </Button>
          {step < STEPS.length ? (
            <Button
              size="sm"
              onClick={handleNext}
              disabled={!canProceed()}
            >
              {t('common.next')}
              <ArrowRight className="h-3.5 w-3.5 ml-1" />
            </Button>
          ) : (
            <Button size="sm" onClick={handleFinish}>
              {t('settings.integrations.bexio.setup.finish')}
            </Button>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// Step sub-components
// ---------------------------------------------------------------------------

function StepOAuth({
  connected,
  orgName,
  onConnect,
  isConnecting,
  isPolling,
}: {
  connected: boolean
  orgName?: string
  onConnect: () => void
  isConnecting: boolean
  isPolling: boolean
}) {
  const { t } = useTranslation()
  return (
    <div className="space-y-4">
      <div className="rounded-md border border-info/30 bg-info-light p-3">
        <p className="text-sm text-info">
          {t('settings.integrations.bexio.setup.oauthInfo')}
        </p>
      </div>

      {connected ? (
        <div className="flex items-center gap-2 rounded-md border border-success/30 bg-success-light p-3">
          <CheckCircle className="h-4 w-4 text-success shrink-0" />
          <div>
            <p className="text-sm text-success font-medium">
              {t('settings.integrations.bexio.setup.connectionEstablished')}
            </p>
            {orgName && (
              <p className="text-xs text-success mt-0.5">
                {t('settings.integrations.bexio.setup.organisation', { name: orgName })}
              </p>
            )}
          </div>
        </div>
      ) : (
        <div className="space-y-3">
          <Button onClick={onConnect} disabled={isConnecting || isPolling}>
            {isConnecting ? (
              <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
            ) : (
              <ExternalLink className="h-3.5 w-3.5 mr-1.5" />
            )}
            {t('settings.integrations.bexio.setup.connectButton')}
          </Button>

          {isPolling && (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
              {t('settings.integrations.bexio.setup.waitingAuth')}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

function StepSyncConfig({
  config,
  onChange,
}: {
  config: BexioSyncConfig
  onChange: (config: BexioSyncConfig) => void
}) {
  const { t } = useTranslation()
  const update = (partial: Partial<BexioSyncConfig>) => {
    onChange({ ...config, ...partial })
  }

  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">
        {t('settings.integrations.bexio.setup.chooseSyncData')}
      </p>

      <div className="space-y-3">
        <div className="flex items-center justify-between rounded-md border border-border p-3">
          <div>
            <Label className="text-sm font-medium">
              {t('settings.integrations.bexio.setup.contactSyncLabel')}
            </Label>
            <p className="text-xs text-muted-foreground mt-0.5">
              {t('settings.integrations.bexio.setup.contactSyncHelp')}
            </p>
          </div>
          <Switch
            checked={config.contact_sync_enabled}
            onCheckedChange={(v) => update({ contact_sync_enabled: v })}
          />
        </div>

        {config.contact_sync_enabled && (
          <div className="ml-4 flex items-center gap-2">
            <Label className="text-xs text-muted-foreground">
              {t('settings.integrations.bexio.setup.intervalLabel')}
            </Label>
            <select
              className="text-xs rounded-md border border-border bg-background px-2 py-1"
              value={config.contact_sync_interval_minutes}
              onChange={(e) =>
                update({ contact_sync_interval_minutes: Number(e.target.value) })
              }
            >
              {INTERVAL_OPTIONS.map((v) => (
                <option key={v} value={v}>
                  {t('settings.integrations.bexio.setup.minutesOption', { count: v })}
                </option>
              ))}
            </select>
          </div>
        )}

        <div className="flex items-center justify-between rounded-md border border-border p-3">
          <div>
            <Label className="text-sm font-medium">
              {t('settings.integrations.bexio.setup.invoicePushLabel')}
            </Label>
            <p className="text-xs text-muted-foreground mt-0.5">
              {t('settings.integrations.bexio.setup.invoicePushHelp')}
            </p>
          </div>
          <Switch
            checked={config.invoice_push_enabled}
            onCheckedChange={(v) => update({ invoice_push_enabled: v })}
          />
        </div>

        <div className="flex items-center justify-between rounded-md border border-border p-3">
          <div>
            <Label className="text-sm font-medium">
              {t('settings.integrations.bexio.setup.quotePushLabel')}
            </Label>
            <p className="text-xs text-muted-foreground mt-0.5">
              {t('settings.integrations.bexio.setup.quotePushHelp')}
            </p>
          </div>
          <Switch
            checked={config.quote_push_enabled}
            onCheckedChange={(v) => update({ quote_push_enabled: v })}
          />
        </div>

        <div className="flex items-center justify-between rounded-md border border-border p-3">
          <div>
            <Label className="text-sm font-medium">
              {t('settings.integrations.bexio.setup.paymentPollLabel')}
            </Label>
            <p className="text-xs text-muted-foreground mt-0.5">
              {t('settings.integrations.bexio.setup.paymentPollHelp')}
            </p>
          </div>
          <Switch
            checked={config.payment_poll_enabled}
            onCheckedChange={(v) => update({ payment_poll_enabled: v })}
          />
        </div>

        {config.payment_poll_enabled && (
          <div className="ml-4 flex items-center gap-2">
            <Label className="text-xs text-muted-foreground">
              {t('settings.integrations.bexio.setup.intervalLabel')}
            </Label>
            <select
              className="text-xs rounded-md border border-border bg-background px-2 py-1"
              value={config.payment_poll_interval_minutes}
              onChange={(e) =>
                update({
                  payment_poll_interval_minutes: Number(e.target.value),
                })
              }
            >
              {POLL_INTERVAL_OPTIONS.map((v) => (
                <option key={v} value={v}>
                  {t('settings.integrations.bexio.setup.minutesOption', { count: v })}
                </option>
              ))}
            </select>
          </div>
        )}

        <div className="flex items-center justify-between rounded-md border border-border p-3">
          <div>
            <Label className="text-sm font-medium">
              {t('settings.integrations.bexio.setup.invoicePullLabel')}
            </Label>
            <p className="text-xs text-muted-foreground mt-0.5">
              {t('settings.integrations.bexio.setup.invoicePullHelp')}
            </p>
          </div>
          <Switch
            checked={config.invoice_pull_enabled}
            onCheckedChange={(v) => update({ invoice_pull_enabled: v })}
          />
        </div>

        {config.invoice_pull_enabled && (
          <div className="ml-4 flex items-center gap-2">
            <Label className="text-xs text-muted-foreground">
              {t('settings.integrations.bexio.setup.intervalLabel')}
            </Label>
            <select
              className="text-xs rounded-md border border-border bg-background px-2 py-1"
              value={config.invoice_pull_interval_minutes}
              onChange={(e) =>
                update({
                  invoice_pull_interval_minutes: Number(e.target.value),
                })
              }
            >
              {POLL_INTERVAL_OPTIONS.map((v) => (
                <option key={v} value={v}>
                  {t('settings.integrations.bexio.setup.minutesOption', { count: v })}
                </option>
              ))}
            </select>
          </div>
        )}
      </div>
    </div>
  )
}

function StepFieldMapping() {
  const { t } = useTranslation()
  return (
    <div className="space-y-3">
      <p className="text-sm text-muted-foreground">
        {t('settings.integrations.bexio.setup.fieldMappingHint')}
      </p>
      <BexioFieldMappingEditor entityType="contact" compact />
    </div>
  )
}

function StepInitialSync({
  onTrigger,
  isSyncing,
  syncTriggered,
  syncStatus,
}: {
  onTrigger: () => void
  isSyncing: boolean
  syncTriggered: boolean
  syncStatus?: import('@/api/bexio-types').BexioSyncStatus
}) {
  const { t } = useTranslation()

  return (
    <div className="space-y-4">
      <div className="rounded-md border border-success/30 bg-success-light p-3">
        <p className="text-sm text-success font-medium">
          {t('settings.integrations.bexio.setup.configDone')}
        </p>
        <p className="text-xs text-success mt-1">
          {t('settings.integrations.bexio.setup.configDoneHint')}
        </p>
      </div>

      {!syncTriggered ? (
        <Button onClick={onTrigger} disabled={isSyncing}>
          {isSyncing ? (
            <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
          ) : (
            <RefreshCw className="h-3.5 w-3.5 mr-1.5" />
          )}
          {t('settings.integrations.bexio.setup.startSync')}
        </Button>
      ) : (
        <div className="space-y-2">
          {isSyncing ? (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
              {t('settings.integrations.bexio.setup.syncRunning')}
            </div>
          ) : (
            syncStatus && (
              <div className="rounded-md border border-border p-3 space-y-1">
                <p className="text-sm font-medium">
                  {t('settings.integrations.bexio.setup.syncResult')}
                </p>
                <p className="text-xs text-muted-foreground">
                  {t('settings.integrations.bexio.setup.syncResultContacts', {
                    synced: syncStatus.total_contacts_mapped,
                  })}
                </p>
                <p className="text-xs text-muted-foreground">
                  {t('settings.integrations.bexio.setup.syncResultInvoices', {
                    synced: syncStatus.total_invoices_mapped,
                  })}
                </p>
                {syncStatus.last_sync_error && (
                  <p className="text-xs text-destructive">{syncStatus.last_sync_error}</p>
                )}
              </div>
            )
          )}
        </div>
      )}
    </div>
  )
}

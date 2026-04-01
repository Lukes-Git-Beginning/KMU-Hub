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
} from '@/api/hooks/useBexio'
import type { BexioSyncConfig } from '@/api/bexio-types'
import { BexioFieldMappingEditor } from './BexioFieldMappingEditor'

interface BexioSetupWizardProps {
  isOpen: boolean
  onClose: () => void
}

const STEPS = [
  { label: 'Verbindung', number: 1 },
  { label: 'Sync-Optionen', number: 2 },
  { label: 'Feld-Zuordnung', number: 3 },
  { label: 'Erster Sync', number: 4 },
]

const INTERVAL_OPTIONS = [
  { value: 5, label: '5 Minuten' },
  { value: 10, label: '10 Minuten' },
  { value: 15, label: '15 Minuten' },
  { value: 30, label: '30 Minuten' },
  { value: 60, label: '60 Minuten' },
]

const POLL_INTERVAL_OPTIONS = [
  { value: 1, label: '1 Minute' },
  { value: 5, label: '5 Minuten' },
  { value: 10, label: '10 Minuten' },
  { value: 15, label: '15 Minuten' },
]

export function BexioSetupWizard({ isOpen, onClose }: BexioSetupWizardProps) {
  const [step, setStep] = useState(1)
  const [syncConfig, setSyncConfig] = useState<BexioSyncConfig>({
    contact_sync_enabled: true,
    contact_sync_interval_minutes: 15,
    invoice_push_enabled: true,
    quote_push_enabled: true,
    payment_poll_enabled: true,
    payment_poll_interval_minutes: 5,
  })
  const [syncTriggered, setSyncTriggered] = useState(false)

  const getAuthURL = useBexioGetAuthURL()
  const { data: connectionStatus, refetch: refetchConnection } =
    useBexioConnectionStatus()
  const triggerSync = useBexioTriggerSync()
  const { data: syncStatus } = useBexioSyncStatus(syncTriggered)

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
    await triggerSync.mutateAsync()
  }

  const handleFinish = () => {
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
      <DialogContent className="max-w-xl">
        <DialogHeader>
          <DialogTitle>Bexio einrichten</DialogTitle>
          <DialogDescription>
            Verbinden Sie Cosmi mit Ihrem Bexio-Konto in{' '}
            {step === STEPS.length
              ? 'einem letzten'
              : `${step} von ${STEPS.length}`}{' '}
            Schritten
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

        {/* Step content */}
        <div className="min-h-[200px]">
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
            Zurück
          </Button>
          {step < STEPS.length ? (
            <Button
              size="sm"
              onClick={handleNext}
              disabled={!canProceed()}
            >
              Weiter
              <ArrowRight className="h-3.5 w-3.5 ml-1" />
            </Button>
          ) : (
            <Button size="sm" onClick={handleFinish}>
              Fertig
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
  return (
    <div className="space-y-4">
      <div className="rounded-md border border-info/30 bg-info-light p-3">
        <p className="text-sm text-info">
          Verbinden Sie Cosmi mit Ihrem Bexio-Konto ueber OAuth. Sie werden
          zu Bexio weitergeleitet, um den Zugriff zu genehmigen.
        </p>
      </div>

      {connected ? (
        <div className="flex items-center gap-2 rounded-md border border-success/30 bg-success-light p-3">
          <CheckCircle className="h-4 w-4 text-success shrink-0" />
          <div>
            <p className="text-sm text-success font-medium">
              Verbindung hergestellt!
            </p>
            {orgName && (
              <p className="text-xs text-success mt-0.5">
                Organisation: {orgName}
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
            Mit Bexio verbinden
          </Button>

          {isPolling && (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
              Warte auf Bexio-Autorisierung...
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
  const update = (partial: Partial<BexioSyncConfig>) => {
    onChange({ ...config, ...partial })
  }

  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">
        Wählen Sie, welche Daten synchronisiert werden sollen.
      </p>

      <div className="space-y-3">
        <div className="flex items-center justify-between rounded-md border border-border p-3">
          <div>
            <Label className="text-sm font-medium">
              Kontakte bidirektional synchronisieren
            </Label>
            <p className="text-xs text-muted-foreground mt-0.5">
              Kontakte zwischen Cosmi und Bexio abgleichen
            </p>
          </div>
          <Switch
            checked={config.contact_sync_enabled}
            onCheckedChange={(v) => update({ contact_sync_enabled: v })}
          />
        </div>

        {config.contact_sync_enabled && (
          <div className="ml-4 flex items-center gap-2">
            <Label className="text-xs text-muted-foreground">Intervall:</Label>
            <select
              className="text-xs rounded-md border border-border bg-background px-2 py-1"
              value={config.contact_sync_interval_minutes}
              onChange={(e) =>
                update({ contact_sync_interval_minutes: Number(e.target.value) })
              }
            >
              {INTERVAL_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </select>
          </div>
        )}

        <div className="flex items-center justify-between rounded-md border border-border p-3">
          <div>
            <Label className="text-sm font-medium">
              Rechnungen an Bexio senden
            </Label>
            <p className="text-xs text-muted-foreground mt-0.5">
              Erstellte Rechnungen automatisch in Bexio anlegen
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
              Offerten an Bexio senden
            </Label>
            <p className="text-xs text-muted-foreground mt-0.5">
              Erstellte Offerten automatisch in Bexio anlegen
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
              Zahlungsstatus von Bexio abfragen
            </Label>
            <p className="text-xs text-muted-foreground mt-0.5">
              Zahlungseingänge aus Bexio automatisch erkennen
            </p>
          </div>
          <Switch
            checked={config.payment_poll_enabled}
            onCheckedChange={(v) => update({ payment_poll_enabled: v })}
          />
        </div>

        {config.payment_poll_enabled && (
          <div className="ml-4 flex items-center gap-2">
            <Label className="text-xs text-muted-foreground">Intervall:</Label>
            <select
              className="text-xs rounded-md border border-border bg-background px-2 py-1"
              value={config.payment_poll_interval_minutes}
              onChange={(e) =>
                update({
                  payment_poll_interval_minutes: Number(e.target.value),
                })
              }
            >
              {POLL_INTERVAL_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
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
  return (
    <div className="space-y-3">
      <p className="text-sm text-muted-foreground">
        Ordnen Sie die Kontakt-Felder zwischen Cosmi und Bexio zu. Sie
        können diese Zuordnungen später jederzeit anpassen.
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
  const isRunning =
    syncStatus?.contact_sync?.status === 'running' ||
    syncStatus?.invoice_sync?.status === 'running'

  return (
    <div className="space-y-4">
      <div className="rounded-md border border-success/30 bg-success-light p-3">
        <p className="text-sm text-success font-medium">
          Konfiguration abgeschlossen!
        </p>
        <p className="text-xs text-success mt-1">
          Starten Sie jetzt die erste Synchronisierung, um Ihre Daten
          abzugleichen.
        </p>
      </div>

      {!syncTriggered ? (
        <Button onClick={onTrigger} disabled={isSyncing}>
          {isSyncing ? (
            <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
          ) : (
            <RefreshCw className="h-3.5 w-3.5 mr-1.5" />
          )}
          Synchronisierung starten
        </Button>
      ) : (
        <div className="space-y-2">
          {isRunning ? (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
              Synchronisierung laeuft...
            </div>
          ) : (
            syncStatus && (
              <div className="rounded-md border border-border p-3 space-y-1">
                <p className="text-sm font-medium">Ergebnis:</p>
                <p className="text-xs text-muted-foreground">
                  Kontakte: {syncStatus.contact_sync.items_synced} synchronisiert
                  {syncStatus.contact_sync.items_failed > 0 &&
                    `, ${syncStatus.contact_sync.items_failed} Fehler`}
                </p>
                <p className="text-xs text-muted-foreground">
                  Rechnungen: {syncStatus.invoice_sync.items_synced} synchronisiert
                  {syncStatus.invoice_sync.items_failed > 0 &&
                    `, ${syncStatus.invoice_sync.items_failed} Fehler`}
                </p>
              </div>
            )
          )}
        </div>
      )}
    </div>
  )
}

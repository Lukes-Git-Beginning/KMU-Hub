/**
 * 4-step Slack integration setup wizard.
 *
 * Steps:
 * 1. Plattform -- Slack App setup info, manual or OAuth credential input
 * 2. Kanalzuordnung -- Channel mapping editor
 * 3. Test -- Send test notification
 * 4. Fertig -- Summary and activation toggle
 */
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
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
  XCircle,
  ExternalLink,
  Send,
  Loader2,
} from 'lucide-react'
import {
  useCreateIntegrationConfig,
  useUpdateIntegrationConfig,
  useTestIntegrationConfig,
} from '@/api/hooks/useIntegration'
import { API_BASE_URL } from '@/lib/constants'
import type { IntegrationConfig } from '@/api/integration-types'
import { ChannelMappingEditor } from './ChannelMappingEditor'

interface SlackSetupWizardProps {
  isOpen: boolean
  onClose: () => void
  /** Existing config for editing, undefined for new setup */
  existingConfig?: IntegrationConfig
}

type CredentialMode = 'manual' | 'oauth'

const STEPS = [
  { label: 'Plattform', number: 1 },
  { label: 'Kanalzuordnung', number: 2 },
  { label: 'Test', number: 3 },
  { label: 'Fertig', number: 4 },
]

export function SlackSetupWizard({
  isOpen,
  onClose,
  existingConfig,
}: SlackSetupWizardProps) {
  const [step, setStep] = useState(1)
  const [credMode, setCredMode] = useState<CredentialMode>('manual')
  const [botToken, setBotToken] = useState('')
  const [signingSecret, setSigningSecret] = useState('')
  const [clientId, setClientId] = useState('')
  const [clientSecret, setClientSecret] = useState('')
  const [isActive, setIsActive] = useState(true)
  const [testResult, setTestResult] = useState<'success' | 'error' | null>(
    null,
  )

  const createConfig = useCreateIntegrationConfig()
  const updateConfig = useUpdateIntegrationConfig('slack')
  const testConfig = useTestIntegrationConfig('slack')

  const isEditing = !!existingConfig

  const handleNext = async () => {
    if (step === 1 && !isEditing && credMode === 'manual') {
      if (!botToken.trim() || !signingSecret.trim()) return
      await createConfig.mutateAsync({
        platform: 'slack',
        credentials_vault_key: `slack:${clientId.trim() || 'manual'}`,
        metadata: JSON.stringify({
          bot_token_set: true,
          signing_secret_set: true,
          client_id: clientId.trim() || undefined,
          setup_mode: 'manual',
        }),
      })
    }
    setStep((s) => Math.min(s + 1, 4))
  }

  const handleBack = () => {
    setStep((s) => Math.max(s - 1, 1))
  }

  const handleTest = async () => {
    setTestResult(null)
    try {
      const result = await testConfig.mutateAsync()
      setTestResult(result.success ? 'success' : 'error')
    } catch {
      setTestResult('error')
    }
  }

  const handleOAuthInstall = () => {
    // Open OAuth install flow in a new window
    const oauthUrl = `${API_BASE_URL}/api/v1/integrations/slack/oauth/install`
    window.open(oauthUrl, '_blank', 'width=600,height=700')
  }

  const handleFinish = async () => {
    if (isEditing) {
      await updateConfig.mutateAsync({ is_active: isActive })
    }
    onClose()
    // Reset wizard state
    setStep(1)
    setBotToken('')
    setSigningSecret('')
    setClientId('')
    setClientSecret('')
    setTestResult(null)
    setIsActive(true)
    setCredMode('manual')
  }

  const canProceed = () => {
    if (step === 1 && !isEditing && credMode === 'manual') {
      return botToken.trim().length > 0 && signingSecret.trim().length > 0
    }
    return true
  }

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-xl">
        <DialogHeader>
          <DialogTitle>Slack einrichten</DialogTitle>
          <DialogDescription>
            Konfigurieren Sie die Slack-Integration in{' '}
            {STEPS[step - 1].label === 'Fertig'
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
            <StepPlatform
              credMode={credMode}
              setCredMode={setCredMode}
              botToken={botToken}
              setBotToken={setBotToken}
              signingSecret={signingSecret}
              setSigningSecret={setSigningSecret}
              clientId={clientId}
              setClientId={setClientId}
              clientSecret={clientSecret}
              setClientSecret={setClientSecret}
              isEditing={isEditing}
              onOAuthInstall={handleOAuthInstall}
            />
          )}
          {step === 2 && <StepChannelMapping />}
          {step === 3 && (
            <StepTest
              onTest={handleTest}
              testResult={testResult}
              isTesting={testConfig.isPending}
            />
          )}
          {step === 4 && (
            <StepFinish isActive={isActive} setIsActive={setIsActive} />
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
          {step < 4 ? (
            <Button
              size="sm"
              onClick={handleNext}
              disabled={!canProceed() || createConfig.isPending}
            >
              {createConfig.isPending && (
                <Loader2 className="h-3.5 w-3.5 mr-1 animate-spin" />
              )}
              Weiter
              <ArrowRight className="h-3.5 w-3.5 ml-1" />
            </Button>
          ) : (
            <Button size="sm" onClick={handleFinish}>
              Speichern und aktivieren
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

function StepPlatform({
  credMode,
  setCredMode,
  botToken,
  setBotToken,
  signingSecret,
  setSigningSecret,
  clientId,
  setClientId,
  clientSecret,
  setClientSecret,
  isEditing,
  onOAuthInstall,
}: {
  credMode: CredentialMode
  setCredMode: (v: CredentialMode) => void
  botToken: string
  setBotToken: (v: string) => void
  signingSecret: string
  setSigningSecret: (v: string) => void
  clientId: string
  setClientId: (v: string) => void
  clientSecret: string
  setClientSecret: (v: string) => void
  isEditing: boolean
  onOAuthInstall: () => void
}) {
  return (
    <div className="space-y-4">
      <div className="rounded-md border border-purple-500/30 bg-purple-50/10 p-3">
        <p className="text-sm text-purple-700 dark:text-purple-400">
          Erstellen Sie eine Slack App auf api.slack.com und geben Sie die
          Anmeldeinformationen ein.
        </p>
        <a
          href="https://api.slack.com/apps"
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-1 text-xs text-purple-600 dark:text-purple-400 hover:underline mt-2"
        >
          Slack App erstellen
          <ExternalLink className="h-3 w-3" />
        </a>
      </div>

      {isEditing ? (
        <div className="rounded-md bg-muted p-3">
          <p className="text-sm text-muted-foreground">
            Anmeldeinformationen sind bereits konfiguriert. Sie können die
            Kanalzuordnungen im nächsten Schritt anpassen.
          </p>
        </div>
      ) : (
        <>
          {/* Mode selector */}
          <div className="flex gap-2">
            <Button
              variant={credMode === 'manual' ? 'default' : 'outline'}
              size="sm"
              onClick={() => setCredMode('manual')}
            >
              Manuell eingeben
            </Button>
            <Button
              variant={credMode === 'oauth' ? 'default' : 'outline'}
              size="sm"
              onClick={() => setCredMode('oauth')}
            >
              Mit Slack verbinden
            </Button>
          </div>

          {credMode === 'manual' ? (
            <div className="space-y-3">
              <div>
                <Label htmlFor="slack-bot-token" className="text-xs">
                  Bot Token (xoxb-...)
                </Label>
                <Input
                  id="slack-bot-token"
                  type="password"
                  placeholder="xoxb-..."
                  value={botToken}
                  onChange={(e) => setBotToken(e.target.value)}
                  className="font-mono text-sm"
                />
              </div>
              <div>
                <Label htmlFor="slack-signing-secret" className="text-xs">
                  Signing Secret
                </Label>
                <Input
                  id="slack-signing-secret"
                  type="password"
                  placeholder="Signing Secret eingeben"
                  value={signingSecret}
                  onChange={(e) => setSigningSecret(e.target.value)}
                />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <Label htmlFor="slack-client-id" className="text-xs">
                    Client ID (optional)
                  </Label>
                  <Input
                    id="slack-client-id"
                    placeholder="Client ID"
                    value={clientId}
                    onChange={(e) => setClientId(e.target.value)}
                    className="font-mono text-sm"
                  />
                </div>
                <div>
                  <Label htmlFor="slack-client-secret" className="text-xs">
                    Client Secret (optional)
                  </Label>
                  <Input
                    id="slack-client-secret"
                    type="password"
                    placeholder="Client Secret"
                    value={clientSecret}
                    onChange={(e) => setClientSecret(e.target.value)}
                  />
                </div>
              </div>
            </div>
          ) : (
            <div className="space-y-3">
              <p className="text-sm text-muted-foreground">
                Klicken Sie auf den Button, um die Slack-App in Ihrem
                Workspace zu installieren. Sie werden zu Slack weitergeleitet.
              </p>
              <Button variant="outline" onClick={onOAuthInstall}>
                <ExternalLink className="h-3.5 w-3.5 mr-1.5" />
                Mit Slack verbinden
              </Button>
              <p className="text-xs text-muted-foreground">
                Nach der Installation kehren Sie hierher zurück und fahren
                mit der Kanalzuordnung fort.
              </p>
            </div>
          )}
        </>
      )}
    </div>
  )
}

function StepChannelMapping() {
  return (
    <div className="space-y-3">
      <p className="text-sm text-muted-foreground">
        Ordnen Sie Slack-Kanäle den Cosmi Modulen zu. Benachrichtigungen
        der ausgewaehlten Module werden an den jeweiligen Kanal weitergeleitet.
      </p>
      <ChannelMappingEditor platform="slack" />
    </div>
  )
}

function StepTest({
  onTest,
  testResult,
  isTesting,
}: {
  onTest: () => void
  testResult: 'success' | 'error' | null
  isTesting: boolean
}) {
  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">
        Senden Sie eine Testbenachrichtigung, um die Verbindung zu prüfen.
      </p>

      <Button variant="outline" onClick={onTest} disabled={isTesting}>
        {isTesting ? (
          <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
        ) : (
          <Send className="h-3.5 w-3.5 mr-1.5" />
        )}
        Test senden
      </Button>

      {testResult === 'success' && (
        <div className="flex items-center gap-2 rounded-md border border-green-500/30 bg-green-50/10 p-3">
          <CheckCircle className="h-4 w-4 text-green-500 shrink-0" />
          <p className="text-sm text-green-700 dark:text-green-400">
            Testbenachrichtigung erfolgreich gesendet!
          </p>
        </div>
      )}
      {testResult === 'error' && (
        <div className="flex items-center gap-2 rounded-md border border-red-500/30 bg-red-50/10 p-3">
          <XCircle className="h-4 w-4 text-red-500 shrink-0" />
          <p className="text-sm text-red-700 dark:text-red-400">
            Test fehlgeschlagen. Bitte überprüfen Sie die Konfiguration.
          </p>
        </div>
      )}
    </div>
  )
}

function StepFinish({
  isActive,
  setIsActive,
}: {
  isActive: boolean
  setIsActive: (v: boolean) => void
}) {
  return (
    <div className="space-y-4">
      <div className="rounded-md border border-green-500/30 bg-green-50/10 p-3">
        <p className="text-sm text-green-700 dark:text-green-400 font-medium">
          Slack Konfiguration abgeschlossen!
        </p>
        <p className="text-xs text-green-600 dark:text-green-500 mt-1">
          Die Integration kann jetzt aktiviert werden.
        </p>
      </div>

      <div className="flex items-center justify-between rounded-md border border-border p-3">
        <div>
          <p className="text-sm font-medium">Integration aktivieren</p>
          <p className="text-xs text-muted-foreground">
            Benachrichtigungen werden an Slack weitergeleitet
          </p>
        </div>
        <Switch checked={isActive} onCheckedChange={setIsActive} />
      </div>
    </div>
  )
}

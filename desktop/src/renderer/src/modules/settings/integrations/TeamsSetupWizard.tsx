/**
 * 4-step Microsoft Teams integration setup wizard.
 *
 * Steps:
 * 1. Plattform -- Azure AD App registration info + credential input
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
import type { IntegrationConfig } from '@/api/integration-types'
import { ChannelMappingEditor } from './ChannelMappingEditor'

interface TeamsSetupWizardProps {
  isOpen: boolean
  onClose: () => void
  /** Existing config for editing, undefined for new setup */
  existingConfig?: IntegrationConfig
}

const STEPS = [
  { label: 'Plattform', number: 1 },
  { label: 'Kanalzuordnung', number: 2 },
  { label: 'Test', number: 3 },
  { label: 'Fertig', number: 4 },
]

export function TeamsSetupWizard({
  isOpen,
  onClose,
  existingConfig,
}: TeamsSetupWizardProps) {
  const [step, setStep] = useState(1)
  const [appId, setAppId] = useState('')
  const [appPassword, setAppPassword] = useState('')
  const [isActive, setIsActive] = useState(true)
  const [testResult, setTestResult] = useState<'success' | 'error' | null>(
    null,
  )

  const createConfig = useCreateIntegrationConfig()
  const updateConfig = useUpdateIntegrationConfig('teams')
  const testConfig = useTestIntegrationConfig('teams')

  const isEditing = !!existingConfig

  const handleNext = async () => {
    if (step === 1 && !isEditing) {
      // Save credentials on first step completion
      if (!appId.trim() || !appPassword.trim()) return
      await createConfig.mutateAsync({
        platform: 'teams',
        credentials_vault_key: `teams:${appId.trim()}`,
        metadata: JSON.stringify({
          app_id: appId.trim(),
          app_password_set: true,
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

  const handleFinish = async () => {
    if (isEditing) {
      await updateConfig.mutateAsync({ is_active: isActive })
    }
    onClose()
    // Reset wizard state
    setStep(1)
    setAppId('')
    setAppPassword('')
    setTestResult(null)
    setIsActive(true)
  }

  const canProceed = () => {
    if (step === 1 && !isEditing) {
      return appId.trim().length > 0 && appPassword.trim().length > 0
    }
    return true
  }

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-xl">
        <DialogHeader>
          <DialogTitle>Microsoft Teams einrichten</DialogTitle>
          <DialogDescription>
            Konfigurieren Sie die Teams-Integration in{' '}
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
              appId={appId}
              setAppId={setAppId}
              appPassword={appPassword}
              setAppPassword={setAppPassword}
              isEditing={isEditing}
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
            Zurueck
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
  appId,
  setAppId,
  appPassword,
  setAppPassword,
  isEditing,
}: {
  appId: string
  setAppId: (v: string) => void
  appPassword: string
  setAppPassword: (v: string) => void
  isEditing: boolean
}) {
  return (
    <div className="space-y-4">
      <div className="rounded-md border border-blue-500/30 bg-blue-50/10 p-3">
        <p className="text-sm text-blue-700 dark:text-blue-400">
          Erstellen Sie eine Azure AD App-Registrierung und geben Sie die
          Anmeldeinformationen ein.
        </p>
        <a
          href="https://learn.microsoft.com/en-us/microsoftteams/platform/bots/how-to/authentication/bot-sso-register-aad"
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-1 text-xs text-blue-600 dark:text-blue-400 hover:underline mt-2"
        >
          Microsoft-Dokumentation oeffnen
          <ExternalLink className="h-3 w-3" />
        </a>
      </div>

      {isEditing ? (
        <div className="rounded-md bg-muted p-3">
          <p className="text-sm text-muted-foreground">
            Anmeldeinformationen sind bereits konfiguriert. Sie koennen die
            Kanalzuordnungen im naechsten Schritt anpassen.
          </p>
        </div>
      ) : (
        <div className="space-y-3">
          <div>
            <Label htmlFor="teams-app-id" className="text-xs">
              App ID (Application/Client ID)
            </Label>
            <Input
              id="teams-app-id"
              placeholder="z.B. 12345678-abcd-1234-abcd-123456789abc"
              value={appId}
              onChange={(e) => setAppId(e.target.value)}
              className="font-mono text-sm"
            />
          </div>
          <div>
            <Label htmlFor="teams-app-pw" className="text-xs">
              App Password (Client Secret)
            </Label>
            <Input
              id="teams-app-pw"
              type="password"
              placeholder="Client Secret eingeben"
              value={appPassword}
              onChange={(e) => setAppPassword(e.target.value)}
            />
          </div>
        </div>
      )}
    </div>
  )
}

function StepChannelMapping() {
  return (
    <div className="space-y-3">
      <p className="text-sm text-muted-foreground">
        Ordnen Sie Teams-Kanaele den KMU Hub Modulen zu. Benachrichtigungen
        der ausgewaehlten Module werden an den jeweiligen Kanal weitergeleitet.
      </p>
      <ChannelMappingEditor platform="teams" />
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
        Senden Sie eine Testbenachrichtigung, um die Verbindung zu pruefen.
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
            Test fehlgeschlagen. Bitte ueberpruefen Sie die Konfiguration.
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
          Microsoft Teams Konfiguration abgeschlossen!
        </p>
        <p className="text-xs text-green-600 dark:text-green-500 mt-1">
          Die Integration kann jetzt aktiviert werden.
        </p>
      </div>

      <div className="flex items-center justify-between rounded-md border border-border p-3">
        <div>
          <p className="text-sm font-medium">Integration aktivieren</p>
          <p className="text-xs text-muted-foreground">
            Benachrichtigungen werden an Teams weitergeleitet
          </p>
        </div>
        <Switch checked={isActive} onCheckedChange={setIsActive} />
      </div>
    </div>
  )
}

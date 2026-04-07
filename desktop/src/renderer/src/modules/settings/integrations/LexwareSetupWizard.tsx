/**
 * 3-step Lexware Office integration setup wizard.
 *
 * Steps:
 * 1. API-Schluessel -- Enter API key and test connection
 * 2. Feld-Zuordnung -- Configure field mappings (embedded compact editor)
 * 3. Erster Sync -- Trigger initial sync and show progress
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { CheckCircle2, Loader2, ArrowRight, Key } from 'lucide-react'
import {
  useLexwareConnect,
  useLexwareTestConnection,
  useLexwareTriggerSync,
} from '@/api/hooks/useLexware'
import { LexwareFieldMappingEditor } from './LexwareFieldMappingEditor'

interface LexwareSetupWizardProps {
  isOpen: boolean
  onClose: () => void
}

export function LexwareSetupWizard({ isOpen, onClose }: LexwareSetupWizardProps) {
  const { t } = useTranslation()
  const [step, setStep] = useState(1)
  const [apiKey, setApiKey] = useState('')
  const [testResult, setTestResult] = useState<'idle' | 'success' | 'error'>('idle')

  const connect = useLexwareConnect()
  const testConnection = useLexwareTestConnection()
  const triggerSync = useLexwareTriggerSync()

  const handleTest = async () => {
    try {
      const result = await connect.mutateAsync(apiKey)
      if (result.success) {
        const test = await testConnection.mutateAsync()
        setTestResult(test.success ? 'success' : 'error')
      } else {
        setTestResult('error')
      }
    } catch {
      setTestResult('error')
    }
  }

  const handleStartSync = async () => {
    try {
      await triggerSync.mutateAsync('contacts')
    } catch {
      // handled by mutation
    }
    onClose()
  }

  const handleClose = () => {
    setStep(1)
    setApiKey('')
    setTestResult('idle')
    onClose()
  }

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && handleClose()}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t('settings.integrations.lexware.setup.title')}</DialogTitle>
          <DialogDescription>
            {step === 1 && t('settings.integrations.lexware.setup.descStep1')}
            {step === 2 && t('settings.integrations.lexware.setup.descStep2')}
            {step === 3 && t('settings.integrations.lexware.setup.descStep3')}
          </DialogDescription>
        </DialogHeader>

        {/* Step indicators */}
        <div className="flex items-center gap-2 mb-4">
          {[1, 2, 3].map((s) => (
            <div
              key={s}
              className={`h-2 flex-1 rounded-full ${
                s <= step ? 'bg-primary' : 'bg-muted'
              }`}
            />
          ))}
        </div>

        {step === 1 && (
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="api-key">{t('settings.integrations.lexware.setup.apiKeyLabel')}</Label>
              <div className="relative">
                <Key className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  id="api-key"
                  type="password"
                  placeholder={t('settings.integrations.lexware.setup.apiKeyPlaceholder')}
                  value={apiKey}
                  onChange={(e) => {
                    setApiKey(e.target.value)
                    setTestResult('idle')
                  }}
                  className="pl-10"
                />
              </div>
              <p className="text-xs text-muted-foreground">
                {t('settings.integrations.lexware.setup.apiKeyHint')}
              </p>
            </div>

            {testResult === 'success' && (
              <div className="flex items-center gap-2 text-success text-sm">
                <CheckCircle2 className="h-4 w-4" />
                {t('settings.integrations.lexware.setup.connectionSuccess')}
              </div>
            )}
            {testResult === 'error' && (
              <p className="text-sm text-destructive">
                {t('settings.integrations.lexware.setup.connectionError')}
              </p>
            )}
          </div>
        )}

        {step === 2 && (
          <div className="max-h-[400px] overflow-y-auto">
            <LexwareFieldMappingEditor entityType="contact" compact />
          </div>
        )}

        {step === 3 && (
          <div className="text-center py-6 space-y-3">
            <CheckCircle2 className="h-12 w-12 text-success mx-auto" />
            <h3 className="font-medium">{t('settings.integrations.lexware.setup.connectionEstablished')}</h3>
            <p className="text-sm text-muted-foreground">
              {t('settings.integrations.lexware.setup.syncPrompt')}
            </p>
          </div>
        )}

        <DialogFooter>
          {step === 1 && (
            <>
              <Button variant="outline" onClick={handleClose}>
                {t('common.cancel')}
              </Button>
              <Button
                onClick={testResult === 'success' ? () => setStep(2) : handleTest}
                disabled={!apiKey || connect.isPending || testConnection.isPending}
              >
                {(connect.isPending || testConnection.isPending) && (
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                )}
                {testResult === 'success' ? (
                  <>
                    <ArrowRight className="h-4 w-4 mr-2" />
                    {t('common.next')}
                  </>
                ) : (
                  t('settings.integrations.lexware.setup.testConnection')
                )}
              </Button>
            </>
          )}
          {step === 2 && (
            <>
              <Button variant="outline" onClick={() => setStep(1)}>
                {t('common.back')}
              </Button>
              <Button onClick={() => setStep(3)}>
                <ArrowRight className="h-4 w-4 mr-2" />
                {t('common.next')}
              </Button>
            </>
          )}
          {step === 3 && (
            <>
              <Button variant="outline" onClick={handleClose}>
                {t('settings.integrations.lexware.setup.later')}
              </Button>
              <Button onClick={handleStartSync} disabled={triggerSync.isPending}>
                {triggerSync.isPending && (
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                )}
                {t('settings.integrations.lexware.setup.startSync')}
              </Button>
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

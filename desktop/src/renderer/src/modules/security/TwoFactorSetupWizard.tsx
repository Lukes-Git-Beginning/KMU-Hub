/**
 * Two-factor authentication setup wizard.
 *
 * Multi-step modal dialog guiding users through 2FA setup:
 * 1. Introduction -- explain 2FA, click "Enable" to start
 * 2. QR Code -- scan with authenticator app or enter manual secret
 * 3. Verification -- enter 6-digit TOTP code to confirm
 * 4. Recovery Codes -- save 8 backup codes, download/copy options
 *
 * Also provides Regenerate Recovery Codes and Disable 2FA actions
 * when 2FA is already enabled.
 */
import { useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Shield } from 'lucide-react'
import {
  useSetup2FA,
  useVerify2FA,
  useDisable2FA,
  useRegenerateRecoveryCodes,
} from '@/api/hooks/use2FA'

type WizardStep = 'intro' | 'qr' | 'verify' | 'recovery'

const STEP_ORDER: WizardStep[] = ['intro', 'qr', 'verify', 'recovery']

function stepIndex(step: WizardStep): number {
  return STEP_ORDER.indexOf(step) + 1
}

interface TwoFactorSetupWizardProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Whether 2FA is currently enabled on this account. */
  twoFactorEnabled: boolean
  /** Callback when 2FA state changes (enabled/disabled). */
  onStateChange?: () => void
}

export default function TwoFactorSetupWizard({
  open,
  onOpenChange,
  twoFactorEnabled,
  onStateChange,
}: TwoFactorSetupWizardProps) {
  const { t } = useTranslation()
  const [step, setStep] = useState<WizardStep>('intro')
  const [qrCodePng, setQrCodePng] = useState('')
  const [manualSecret, setManualSecret] = useState('')
  const [totpCode, setTotpCode] = useState('')
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([])
  const [disableCode, setDisableCode] = useState('')

  const setupMutation = useSetup2FA()
  const verifyMutation = useVerify2FA()
  const disableMutation = useDisable2FA()
  const regenerateMutation = useRegenerateRecoveryCodes()

  const resetWizard = useCallback(() => {
    setStep('intro')
    setQrCodePng('')
    setManualSecret('')
    setTotpCode('')
    setRecoveryCodes([])
    setDisableCode('')
  }, [])

  const handleClose = useCallback(
    (isOpen: boolean) => {
      if (!isOpen) {
        resetWizard()
      }
      onOpenChange(isOpen)
    },
    [onOpenChange, resetWizard],
  )

  // Step 1 -> 2: Start setup
  const handleStartSetup = useCallback(async () => {
    try {
      const result = await setupMutation.mutateAsync()
      setQrCodePng(result.qr_code_png)
      setManualSecret(result.manual_secret)
      setStep('qr')
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : t('common.error'),
      )
    }
  }, [setupMutation, t])

  // Step 3: Verify TOTP code
  const handleVerify = useCallback(async () => {
    if (totpCode.length < 6) return
    try {
      const result = await verifyMutation.mutateAsync(totpCode)
      setRecoveryCodes(result.codes)
      setStep('recovery')
      onStateChange?.()
      toast.success(t('security.2fa.enabled'))
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : t('security.2fa.invalidCode'),
      )
    }
  }, [totpCode, verifyMutation, t, onStateChange])

  // Disable 2FA
  const handleDisable = useCallback(async () => {
    try {
      await disableMutation.mutateAsync(disableCode)
      onStateChange?.()
      toast.success(t('security.2fa.disable'))
      handleClose(false)
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : t('security.2fa.invalidCode'),
      )
    }
  }, [disableCode, disableMutation, t, onStateChange, handleClose])

  // Regenerate recovery codes
  const handleRegenerate = useCallback(async () => {
    try {
      const result = await regenerateMutation.mutateAsync()
      setRecoveryCodes(result.codes)
      setStep('recovery')
      toast.success(t('security.2fa.regenerateCodes'))
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : t('common.error'),
      )
    }
  }, [regenerateMutation, t])

  // Download recovery codes as .txt
  const handleDownloadCodes = useCallback(() => {
    const text = recoveryCodes.join('\n')
    const blob = new Blob([text], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'cosmi-recovery-codes.txt'
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }, [recoveryCodes])

  // Copy recovery codes to clipboard
  const handleCopyCodes = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(recoveryCodes.join('\n'))
      toast.success(t('common.copied'))
    } catch {
      toast.error(t('common.error'))
    }
  }, [recoveryCodes, t])

  // Render "already enabled" management view
  if (twoFactorEnabled && step === 'intro') {
    return (
      <Dialog open={open} onOpenChange={handleClose}>
        <DialogContent className="max-w-md glass-elevated">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Shield className="h-5 w-5 text-primary" />
              {t('security.2fa.title')}
            </DialogTitle>
            <DialogDescription>
              <span className="inline-flex mt-2 rounded-full bg-success-light px-2.5 py-0.5 text-xs font-medium text-success">
                {t('security.2fa.enabled')}
              </span>
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            {/* Regenerate recovery codes */}
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button variant="outline" className="w-full">
                  {t('security.2fa.regenerateCodes')}
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent className="glass-elevated">
                <AlertDialogHeader>
                  <AlertDialogTitle>
                    {t('security.2fa.regenerateCodes')}
                  </AlertDialogTitle>
                  <AlertDialogDescription>
                    {t('security.2fa.regenerateWarning')}
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>
                    {t('common.cancel')}
                  </AlertDialogCancel>
                  <AlertDialogAction
                    onClick={handleRegenerate}
                    disabled={regenerateMutation.isPending}
                  >
                    {t('common.confirm')}
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>

            {/* Disable 2FA */}
            <div className="space-y-2">
              <Input
                type="text"
                inputMode="numeric"
                maxLength={6}
                placeholder={t('security.2fa.enterCode')}
                value={disableCode}
                onChange={(e) => setDisableCode(e.target.value.replace(/\D/g, ''))}
              />
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <Button variant="destructive" className="w-full" disabled={disableCode.length < 6}>
                    {t('security.2fa.disable')}
                  </Button>
                </AlertDialogTrigger>
                <AlertDialogContent className="glass-elevated">
                  <AlertDialogHeader>
                    <AlertDialogTitle>
                      {t('security.2fa.disable')}
                    </AlertDialogTitle>
                    <AlertDialogDescription>
                      {t('security.2fa.enableDescription')}
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>
                      {t('common.cancel')}
                    </AlertDialogCancel>
                    <AlertDialogAction
                      onClick={handleDisable}
                      disabled={disableMutation.isPending}
                    >
                      {t('common.confirm')}
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    )
  }

  // Render recovery codes (shown after regenerate when already enabled too)
  if (twoFactorEnabled && step === 'recovery') {
    return (
      <Dialog open={open} onOpenChange={handleClose}>
        <DialogContent className="max-w-md glass-elevated">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Shield className="h-5 w-5 text-primary" />
              {t('security.2fa.recoveryCodes')}
            </DialogTitle>
            <DialogDescription>
              {t('security.2fa.recoveryDescription')}
            </DialogDescription>
          </DialogHeader>

          <div className="grid grid-cols-2 gap-2 rounded-lg border border-border bg-secondary/50 p-4 font-mono text-sm text-foreground">
            {recoveryCodes.map((code) => (
              <div key={code} className="text-center">
                {code}
              </div>
            ))}
          </div>

          <DialogFooter className="gap-2">
            <Button variant="outline" onClick={handleDownloadCodes}>
              {t('security.2fa.downloadCodes')}
            </Button>
            <Button variant="outline" onClick={handleCopyCodes}>
              {t('security.2fa.copyCodes')}
            </Button>
            <Button onClick={() => handleClose(false)}>
              {t('common.close')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    )
  }

  // Main wizard for new setup
  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-md glass-elevated">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Shield className="h-5 w-5 text-primary" />
            {t('security.2fa.title')}
          </DialogTitle>
          <DialogDescription>
            <span className="text-xs text-muted-foreground">
              {t('security.2fa.step.install')} {stepIndex(step)}/4
            </span>
          </DialogDescription>
        </DialogHeader>

        {/* Step indicator */}
        <div className="flex gap-1">
          {STEP_ORDER.map((s, i) => (
            <div
              key={s}
              className={`h-1.5 flex-1 rounded-full ${
                i < stepIndex(step) ? 'bg-primary' : 'bg-secondary'
              }`}
            />
          ))}
        </div>

        {/* Step 1: Introduction */}
        {step === 'intro' && (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              {t('security.2fa.enableDescription')}
            </p>
            <Button
              className="w-full"
              onClick={handleStartSetup}
              disabled={setupMutation.isPending}
            >
              {setupMutation.isPending ? (
                <span className="flex items-center gap-2">
                  <span className="h-4 w-4 animate-spin rounded-full border-2 border-primary-foreground border-t-transparent" />
                  {t('common.loading')}
                </span>
              ) : (
                t('security.2fa.enable')
              )}
            </Button>
          </div>
        )}

        {/* Step 2: QR Code */}
        {step === 'qr' && (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              {t('security.2fa.scanQR')}
            </p>
            <div className="flex justify-center">
              <img
                src={`data:image/png;base64,${qrCodePng}`}
                alt="2FA QR Code"
                className="h-48 w-48 rounded-md border"
              />
            </div>
            <div className="space-y-1">
              <p className="text-xs text-muted-foreground">
                {t('security.2fa.manualEntry')}
              </p>
              <code className="block rounded-lg border border-border bg-secondary/50 p-2 text-center text-sm font-mono text-foreground select-all">
                {manualSecret}
              </code>
            </div>
            <DialogFooter>
              <Button onClick={() => setStep('verify')}>
                {t('common.next')}
              </Button>
            </DialogFooter>
          </div>
        )}

        {/* Step 3: Verification */}
        {step === 'verify' && (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              {t('security.2fa.step.verify')}
            </p>
            <Input
              type="text"
              inputMode="numeric"
              maxLength={6}
              placeholder={t('security.2fa.enterCode')}
              value={totpCode}
              onChange={(e) => setTotpCode(e.target.value.replace(/\D/g, ''))}
              autoFocus
            />
            <DialogFooter className="gap-2">
              <Button variant="outline" onClick={() => setStep('qr')}>
                {t('common.back')}
              </Button>
              <Button
                onClick={handleVerify}
                disabled={totpCode.length < 6 || verifyMutation.isPending}
              >
                {verifyMutation.isPending ? (
                  <span className="flex items-center gap-2">
                    <span className="h-4 w-4 animate-spin rounded-full border-2 border-primary-foreground border-t-transparent" />
                    {t('common.loading')}
                  </span>
                ) : (
                  t('security.2fa.verify')
                )}
              </Button>
            </DialogFooter>
          </div>
        )}

        {/* Step 4: Recovery Codes */}
        {step === 'recovery' && (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              {t('security.2fa.recoveryDescription')}
            </p>
            <div className="grid grid-cols-2 gap-2 rounded-lg border border-border bg-secondary/50 p-4 font-mono text-sm text-foreground">
              {recoveryCodes.map((code) => (
                <div key={code} className="text-center">
                  {code}
                </div>
              ))}
            </div>
            <DialogFooter className="gap-2">
              <Button variant="outline" onClick={handleDownloadCodes}>
                {t('security.2fa.downloadCodes')}
              </Button>
              <Button variant="outline" onClick={handleCopyCodes}>
                {t('security.2fa.copyCodes')}
              </Button>
              <Button onClick={() => handleClose(false)}>
                {t('common.close')}
              </Button>
            </DialogFooter>
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}

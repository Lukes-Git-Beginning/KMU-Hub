/**
 * Token-based account linking dialog for Teams and Slack.
 *
 * Users enter the token they received from the /cosmi link command
 * in their external platform. The dialog verifies the token and
 * links the account.
 */
import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { CheckCircle, XCircle, Loader2 } from 'lucide-react'
import { useLinkAccount } from '@/api/hooks/useIntegration'
import type { Platform } from '@/api/integration-types'

interface AccountLinkDialogProps {
  platform: Platform
  isOpen: boolean
  onClose: () => void
}

const PLATFORM_LABELS: Record<Platform, string> = {
  teams: 'Microsoft Teams',
  slack: 'Slack',
}

export function AccountLinkDialog({
  platform,
  isOpen,
  onClose,
}: AccountLinkDialogProps) {
  const { t } = useTranslation()
  const [token, setToken] = useState('')
  const [status, setStatus] = useState<'idle' | 'success' | 'error'>('idle')
  const linkAccount = useLinkAccount()

  const handleClose = () => {
    setToken('')
    setStatus('idle')
    onClose()
  }

  // Auto-close after successful link
  useEffect(() => {
    if (status === 'success') {
      const timer = setTimeout(() => {
        handleClose()
      }, 2000)
      return () => clearTimeout(timer)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- handleClose is stable in practice (state setters + onClose prop); adding it would re-trigger on every render
  }, [status])

  const handleLink = async () => {
    if (!token.trim()) return
    try {
      await linkAccount.mutateAsync({ token: token.trim() })
      setStatus('success')
    } catch {
      setStatus('error')
    }
  }

  const platformLabel = PLATFORM_LABELS[platform]

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && handleClose()}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{t('settings.integrations.accountLink.title')}</DialogTitle>
          <DialogDescription>
            {t('settings.integrations.accountLink.description', { platform: platformLabel })}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {status === 'success' ? (
            <div className="flex flex-col items-center gap-3 py-4">
              <CheckCircle className="h-12 w-12 text-success" />
              <p className="text-sm font-medium text-success">
                {t('settings.integrations.accountLink.linked')}
              </p>
              <p className="text-xs text-muted-foreground">
                {t('settings.integrations.accountLink.autoClose')}
              </p>
            </div>
          ) : (
            <>
              {/* Instructions */}
              <div className="space-y-2">
                <p className="text-sm text-muted-foreground">
                  {t('settings.integrations.accountLink.step1', { platform: platformLabel })}
                </p>
                <code className="block rounded-md bg-muted px-3 py-2 text-sm font-mono">
                  /cosmi link
                </code>
                <p className="text-sm text-muted-foreground">
                  {t('settings.integrations.accountLink.step2')}
                </p>
              </div>

              {/* Token input */}
              <Input
                value={token}
                onChange={(e) => {
                  setToken(e.target.value)
                  if (status === 'error') setStatus('idle')
                }}
                placeholder={t('settings.integrations.accountLink.tokenPlaceholder')}
                className="font-mono text-lg text-center tracking-wider"
                onKeyDown={(e) => e.key === 'Enter' && handleLink()}
              />

              {/* Error message */}
              {status === 'error' && (
                <div className="flex items-center gap-2 rounded-md border border-destructive/30 bg-error-light p-2.5">
                  <XCircle className="h-4 w-4 text-destructive shrink-0" />
                  <p className="text-sm text-destructive">
                    {t('settings.integrations.accountLink.tokenInvalid')}
                  </p>
                </div>
              )}

              {/* Actions */}
              <div className="flex items-center justify-end gap-2">
                <Button variant="ghost" size="sm" onClick={handleClose}>
                  {t('common.cancel')}
                </Button>
                <Button
                  size="sm"
                  onClick={handleLink}
                  disabled={!token.trim() || linkAccount.isPending}
                >
                  {linkAccount.isPending && (
                    <Loader2 className="h-3.5 w-3.5 mr-1 animate-spin" />
                  )}
                  {t('settings.integrations.accountLink.link')}
                </Button>
              </div>
            </>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}

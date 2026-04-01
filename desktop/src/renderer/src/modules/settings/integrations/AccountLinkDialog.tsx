/**
 * Token-based account linking dialog for Teams and Slack.
 *
 * Users enter the token they received from the /cosmi link command
 * in their external platform. The dialog verifies the token and
 * links the account.
 */
import { useState, useEffect } from 'react'
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
          <DialogTitle>Konto verknüpfen</DialogTitle>
          <DialogDescription>
            Verknüpfen Sie Ihr {platformLabel}-Konto mit Cosmi
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {status === 'success' ? (
            <div className="flex flex-col items-center gap-3 py-4">
              <CheckCircle className="h-12 w-12 text-green-500" />
              <p className="text-sm font-medium text-green-700 dark:text-green-400">
                Konto verknüpft!
              </p>
              <p className="text-xs text-muted-foreground">
                Dieses Fenster schliesst sich automatisch...
              </p>
            </div>
          ) : (
            <>
              {/* Instructions */}
              <div className="space-y-2">
                <p className="text-sm text-muted-foreground">
                  1. Geben Sie den folgenden Befehl in {platformLabel} ein:
                </p>
                <code className="block rounded-md bg-muted px-3 py-2 text-sm font-mono">
                  /cosmi link
                </code>
                <p className="text-sm text-muted-foreground">
                  2. Sie erhalten einen Token. Geben Sie ihn hier ein:
                </p>
              </div>

              {/* Token input */}
              <Input
                value={token}
                onChange={(e) => {
                  setToken(e.target.value)
                  if (status === 'error') setStatus('idle')
                }}
                placeholder="Token eingeben"
                className="font-mono text-lg text-center tracking-wider"
                onKeyDown={(e) => e.key === 'Enter' && handleLink()}
              />

              {/* Error message */}
              {status === 'error' && (
                <div className="flex items-center gap-2 rounded-md border border-red-500/30 bg-red-50/10 p-2.5">
                  <XCircle className="h-4 w-4 text-red-500 shrink-0" />
                  <p className="text-sm text-red-700 dark:text-red-400">
                    Token ungültig oder abgelaufen. Bitte versuchen Sie es
                    erneut.
                  </p>
                </div>
              )}

              {/* Actions */}
              <div className="flex items-center justify-end gap-2">
                <Button variant="ghost" size="sm" onClick={handleClose}>
                  Abbrechen
                </Button>
                <Button
                  size="sm"
                  onClick={handleLink}
                  disabled={!token.trim() || linkAccount.isPending}
                >
                  {linkAccount.isPending && (
                    <Loader2 className="h-3.5 w-3.5 mr-1 animate-spin" />
                  )}
                  Verknüpfen
                </Button>
              </div>
            </>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}

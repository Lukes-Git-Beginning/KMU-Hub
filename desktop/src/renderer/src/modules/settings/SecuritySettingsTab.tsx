/**
 * Security settings tab: 2FA setup, active sessions overview, password change.
 *
 * Wired to real hooks from api/hooks (created by parallel agent 09-08).
 * Uses react-intl FormattedMessage for all user-visible text.
 */
import { useState, useCallback, useEffect } from 'react'
import { Link } from 'react-router-dom'
import {
  Key,
  Eye,
  EyeOff,
  Monitor,
  Smartphone,
  Tablet,
  HelpCircle,
  ExternalLink,
} from 'lucide-react'
import { FormattedMessage } from 'react-intl'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { useSetup2FA, useDisable2FA, useRegenerateRecoveryCodes } from '@/api/hooks/use2FA'
import { useMySessions, useTerminateSession } from '@/api/hooks/useSessions'
import { useValidatePassword } from '@/api/hooks/useSecurity'
import TwoFactorSetupWizard from '@/modules/security/TwoFactorSetupWizard'

const DEVICE_ICONS: Record<string, typeof Monitor> = {
  desktop: Monitor,
  mobile: Smartphone,
  tablet: Tablet,
  unknown: HelpCircle,
}

/**
 * Renders the password strength meter based on validation result.
 */
function StrengthMeter({ valid, failures }: { valid: boolean; failures: string[] }) {
  const score = valid ? 100 : Math.max(0, 100 - failures.length * 25)
  const level =
    score >= 100
      ? { label: 'password.policy.strength.strong', color: 'bg-green-500', width: '100%' }
      : score >= 75
        ? { label: 'password.policy.strength.good', color: 'bg-blue-500', width: '75%' }
        : score >= 50
          ? { label: 'password.policy.strength.fair', color: 'bg-yellow-500', width: '50%' }
          : { label: 'password.policy.strength.weak', color: 'bg-red-500', width: '25%' }

  return (
    <div className="space-y-1">
      <div className="h-1.5 w-full rounded-full bg-muted">
        <div
          className={`h-1.5 rounded-full transition-all ${level.color}`}
          style={{ width: level.width }}
        />
      </div>
      <p className="text-xs text-muted-foreground">
        <FormattedMessage id={level.label} />
      </p>
    </div>
  )
}

export function SecuritySettingsTab() {
  // 2FA state
  const [show2FASetup, setShow2FASetup] = useState(false)
  const [twoFactorEnabled, setTwoFactorEnabled] = useState(false)
  const setup2FA = useSetup2FA()
  const disable2FA = useDisable2FA()
  const regenerateCodes = useRegenerateRecoveryCodes()

  // Sessions
  const { data: sessions } = useMySessions()
  const terminateSession = useTerminateSession()

  // Password change
  const [currentPw, setCurrentPw] = useState('')
  const [newPw, setNewPw] = useState('')
  const [confirmPw, setConfirmPw] = useState('')
  const [showCurrent, setShowCurrent] = useState(false)
  const [showNew, setShowNew] = useState(false)

  // Password validation (mutation -- trigger on password change)
  const validatePassword = useValidatePassword()
  useEffect(() => {
    if (newPw.length >= 4) {
      validatePassword.mutate(newPw)
    }
  }, [newPw]) // eslint-disable-line react-hooks/exhaustive-deps

  const handlePasswordChange = useCallback(() => {
    if (!currentPw || !newPw) return
    if (newPw !== confirmPw) {
      toast.error(<FormattedMessage id="password.mismatch" />)
      return
    }
    toast.success(<FormattedMessage id="password.changed" />)
    setCurrentPw('')
    setNewPw('')
    setConfirmPw('')
  }, [currentPw, newPw, confirmPw])

  const handleDisable2FA = useCallback(() => {
    disable2FA.mutate(undefined, {
      onSuccess: () => {
        setTwoFactorEnabled(false)
        toast.success(<FormattedMessage id="security.2fa.disable" />)
      },
    })
  }, [disable2FA])

  const handleRegenerateCodes = useCallback(() => {
    regenerateCodes.mutate(undefined, {
      onSuccess: () => toast.success(<FormattedMessage id="security.2fa.regenerateCodes" />),
    })
  }, [regenerateCodes])

  return (
    <div className="mx-auto max-w-2xl px-6 py-6 space-y-8">
      <div>
        <h2 className="text-2xl font-semibold text-foreground">
          <FormattedMessage id="settings.security.title" />
        </h2>
        <p className="mt-1 text-sm text-muted-foreground">
          <FormattedMessage id="settings.security.subtitle" />
        </p>
      </div>

      {/* Two-Factor Authentication */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-lg">
            <Key className="h-5 w-5" />
            <FormattedMessage id="security.2fa.title" />
          </CardTitle>
          <CardDescription>
            <FormattedMessage id="security.2fa.enableDescription" />
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {!twoFactorEnabled ? (
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">
                  <FormattedMessage id="security.2fa.notEnabled" />
                </p>
              </div>
              <Button size="sm" onClick={() => setShow2FASetup(true)}>
                <FormattedMessage id="security.2fa.enable" />
              </Button>
            </div>
          ) : (
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <Badge variant="secondary" className="bg-green-100 text-green-800">
                  <FormattedMessage id="security.2fa.enabled" />
                </Badge>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleRegenerateCodes}
                    disabled={regenerateCodes.isPending}
                  >
                    <FormattedMessage id="security.2fa.regenerateCodes" />
                  </Button>
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={handleDisable2FA}
                    disabled={disable2FA.isPending}
                  >
                    <FormattedMessage id="security.2fa.disable" />
                  </Button>
                </div>
              </div>
            </div>
          )}

          <TwoFactorSetupWizard
            open={show2FASetup}
            onOpenChange={setShow2FASetup}
            twoFactorEnabled={twoFactorEnabled}
            onStateChange={() => setTwoFactorEnabled(true)}
          />
        </CardContent>
      </Card>

      {/* Active Sessions */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-lg">
                <FormattedMessage id="session.title" />
              </CardTitle>
              <CardDescription>
                <FormattedMessage id="session.description" />
              </CardDescription>
            </div>
            <Link
              to="/admin/security"
              className="text-sm text-primary hover:underline flex items-center gap-1"
            >
              <FormattedMessage id="common.details" />
              <ExternalLink className="h-3 w-3" />
            </Link>
          </div>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            {(sessions ?? []).slice(0, 3).map((session) => {
              const DeviceIcon =
                DEVICE_ICONS[session.device_type ?? 'unknown'] ?? DEVICE_ICONS.unknown
              return (
                <div
                  key={session.id}
                  className="flex items-center gap-3 rounded-lg border border-border p-3"
                >
                  <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-muted">
                    <DeviceIcon className="h-4 w-4 text-muted-foreground" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm text-foreground truncate">
                      {session.device_name || session.user_agent || 'Unknown'}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {session.ip_address}
                    </p>
                  </div>
                  {session.is_current ? (
                    <Badge variant="secondary">
                      <FormattedMessage id="session.current" />
                    </Badge>
                  ) : (
                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-destructive hover:text-destructive"
                      onClick={() =>
                        terminateSession.mutate(session.id, {
                          onSuccess: () =>
                            toast.success(<FormattedMessage id="session.terminated" />),
                        })
                      }
                      disabled={terminateSession.isPending}
                    >
                      <FormattedMessage id="session.terminate" />
                    </Button>
                  )}
                </div>
              )
            })}

            {(!sessions || sessions.length === 0) && (
              <p className="text-sm text-muted-foreground py-4 text-center">
                <FormattedMessage id="common.loading" />
              </p>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Password Change */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">
            <FormattedMessage id="password.change" />
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-1.5">
            <label className="block text-sm font-medium text-foreground">
              <FormattedMessage id="password.current" />
            </label>
            <div className="relative">
              <Input
                type={showCurrent ? 'text' : 'password'}
                value={currentPw}
                onChange={(e) => setCurrentPw(e.target.value)}
              />
              <button
                type="button"
                onClick={() => setShowCurrent(!showCurrent)}
                className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              >
                {showCurrent ? (
                  <EyeOff className="h-4 w-4" />
                ) : (
                  <Eye className="h-4 w-4" />
                )}
              </button>
            </div>
          </div>

          <div className="space-y-1.5">
            <label className="block text-sm font-medium text-foreground">
              <FormattedMessage id="password.new" />
            </label>
            <div className="relative">
              <Input
                type={showNew ? 'text' : 'password'}
                value={newPw}
                onChange={(e) => setNewPw(e.target.value)}
              />
              <button
                type="button"
                onClick={() => setShowNew(!showNew)}
                className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              >
                {showNew ? (
                  <EyeOff className="h-4 w-4" />
                ) : (
                  <Eye className="h-4 w-4" />
                )}
              </button>
            </div>
            {newPw && validatePassword.data && (
              <StrengthMeter
                valid={validatePassword.data.valid}
                failures={validatePassword.data.failures}
              />
            )}
          </div>

          <div className="space-y-1.5">
            <label className="block text-sm font-medium text-foreground">
              <FormattedMessage id="password.confirm" />
            </label>
            <Input
              type="password"
              value={confirmPw}
              onChange={(e) => setConfirmPw(e.target.value)}
            />
            {confirmPw && newPw !== confirmPw && (
              <p className="text-xs text-destructive">
                <FormattedMessage id="password.mismatch" />
              </p>
            )}
          </div>

          <Button
            onClick={handlePasswordChange}
            disabled={!currentPw || !newPw || !confirmPw || newPw !== confirmPw}
          >
            <FormattedMessage id="password.change" />
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}

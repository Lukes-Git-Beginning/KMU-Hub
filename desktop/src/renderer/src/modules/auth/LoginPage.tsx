/**
 * Login page with email/password form and 2FA TOTP prompt.
 *
 * Authenticates against the backend API and redirects to the
 * app shell on success. Uses Electron safeStorage for token persistence.
 *
 * State machine: 'credentials' -> '2fa_prompt' -> redirect to app
 * When the backend returns requiresTwoFactor, the page shows a TOTP
 * input instead of redirecting. Users can also use recovery codes.
 */
import { useState, useCallback, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { FormattedMessage, useIntl } from 'react-intl'
import { useAuthStore } from '@/stores/auth'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'

type LoginStage = 'credentials' | '2fa_prompt'

export default function LoginPage() {
  const intl = useIntl()

  // Credentials state
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

  // 2FA state
  const [stage, setStage] = useState<LoginStage>('credentials')
  const [totpCode, setTotpCode] = useState('')
  const [useRecovery, setUseRecovery] = useState(false)
  const [pendingToken, setPendingToken] = useState<string | null>(null)

  const login = useAuthStore((s) => s.login)
  const complete2FALogin = useAuthStore((s) => s.complete2FALogin)
  const navigate = useNavigate()

  const handleCredentialsSubmit = useCallback(
    async (e: FormEvent) => {
      e.preventDefault()
      setError(null)
      setIsSubmitting(true)

      try {
        await login(email, password)
        // If we get here without 2FA, login succeeded -- redirect
        navigate('/', { replace: true })
      } catch (err) {
        if (err instanceof Error && err.message === '2FA_REQUIRED') {
          // Login returned a pending token -- switch to 2FA prompt
          const token = useAuthStore.getState().pendingToken
          setPendingToken(token)
          setStage('2fa_prompt')
        } else {
          setError(
            err instanceof Error
              ? err.message
              : intl.formatMessage({ id: 'auth.loginFailed' }),
          )
        }
      } finally {
        setIsSubmitting(false)
      }
    },
    [email, password, login, navigate, intl],
  )

  const handle2FASubmit = useCallback(
    async (e: FormEvent) => {
      e.preventDefault()
      if (!pendingToken || !totpCode) return
      setError(null)
      setIsSubmitting(true)

      try {
        await complete2FALogin(pendingToken, totpCode)
        navigate('/', { replace: true })
      } catch (err) {
        setError(
          err instanceof Error
            ? err.message
            : intl.formatMessage({ id: 'security.2fa.invalidCode' }),
        )
      } finally {
        setIsSubmitting(false)
      }
    },
    [pendingToken, totpCode, complete2FALogin, navigate, intl],
  )

  const handleBackToCredentials = useCallback(() => {
    setStage('credentials')
    setTotpCode('')
    setUseRecovery(false)
    setPendingToken(null)
    setError(null)
  }, [])

  // 2FA prompt view
  if (stage === '2fa_prompt') {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background p-4">
        <Card className="w-full max-w-sm">
          <CardHeader className="text-center">
            <CardTitle className="text-2xl font-bold tracking-tight">
              <FormattedMessage id="security.2fa.title" />
            </CardTitle>
            <CardDescription>
              {useRecovery ? (
                <FormattedMessage id="security.2fa.useRecoveryCode" />
              ) : (
                <FormattedMessage id="security.2fa.loginPrompt" />
              )}
            </CardDescription>
          </CardHeader>

          <CardContent>
            <form onSubmit={handle2FASubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="totp-code">
                  {useRecovery ? (
                    <FormattedMessage id="security.2fa.useRecoveryCode" />
                  ) : (
                    <FormattedMessage id="security.2fa.enterCode" />
                  )}
                </Label>
                <Input
                  id="totp-code"
                  type="text"
                  inputMode={useRecovery ? 'text' : 'numeric'}
                  maxLength={useRecovery ? 20 : 6}
                  placeholder={useRecovery ? 'xxxxxxxxxx' : '000000'}
                  value={totpCode}
                  onChange={(e) =>
                    setTotpCode(
                      useRecovery
                        ? e.target.value
                        : e.target.value.replace(/\D/g, ''),
                    )
                  }
                  disabled={isSubmitting}
                  required
                  autoFocus
                />
              </div>

              {error && (
                <p className="text-sm text-destructive">{error}</p>
              )}

              <Button
                type="submit"
                className="w-full"
                disabled={isSubmitting || !totpCode}
              >
                {isSubmitting ? (
                  <span className="flex items-center gap-2">
                    <span className="h-4 w-4 animate-spin rounded-full border-2 border-primary-foreground border-t-transparent" />
                    <FormattedMessage id="common.loading" />
                  </span>
                ) : (
                  <FormattedMessage id="security.2fa.verify" />
                )}
              </Button>

              <div className="flex items-center justify-between text-sm">
                <button
                  type="button"
                  className="text-muted-foreground underline-offset-2 hover:underline"
                  onClick={() => {
                    setUseRecovery(!useRecovery)
                    setTotpCode('')
                    setError(null)
                  }}
                >
                  {useRecovery ? (
                    <FormattedMessage id="security.2fa.enterCode" />
                  ) : (
                    <FormattedMessage id="security.2fa.useRecoveryCode" />
                  )}
                </button>
                <button
                  type="button"
                  className="text-muted-foreground underline-offset-2 hover:underline"
                  onClick={handleBackToCredentials}
                >
                  <FormattedMessage id="common.back" />
                </button>
              </div>
            </form>
          </CardContent>
        </Card>
      </div>
    )
  }

  // Credentials view (default)
  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <Card className="w-full max-w-sm">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl font-bold tracking-tight">
            KMU Hub
          </CardTitle>
          <CardDescription>
            <FormattedMessage id="auth.login" />
          </CardDescription>
        </CardHeader>

        <CardContent>
          <form onSubmit={handleCredentialsSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="email">
                <FormattedMessage id="auth.email" />
              </Label>
              <Input
                id="email"
                type="email"
                placeholder="name@firma.de"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                disabled={isSubmitting}
                required
                autoFocus
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="password">
                <FormattedMessage id="auth.password" />
              </Label>
              <Input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                disabled={isSubmitting}
                required
              />
            </div>

            {error && (
              <p className="text-sm text-destructive">{error}</p>
            )}

            <Button
              type="submit"
              className="w-full"
              disabled={isSubmitting}
            >
              {isSubmitting ? (
                <span className="flex items-center gap-2">
                  <span className="h-4 w-4 animate-spin rounded-full border-2 border-primary-foreground border-t-transparent" />
                  <FormattedMessage id="common.loading" />
                </span>
              ) : (
                <FormattedMessage id="auth.login" />
              )}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}

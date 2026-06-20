/**
 * Login page with email/password form and 2FA TOTP prompt.
 *
 * Authenticates against the backend API and redirects to the
 * app shell on success. Uses Electron safeStorage for token persistence.
 *
 * State machine: 'credentials' -> '2fa_prompt' -> redirect to app
 * When the backend returns requiresTwoFactor, the page shows a TOTP
 * input instead of redirecting. Users can also use recovery codes.
 *
 * The "stay signed in" checkbox controls whether tokens are persisted
 * via safeStorage across app restarts (default: true / persistent).
 */
import { useState, useCallback, type FormEvent } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '@/stores/auth'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
import { AuthLayout } from './AuthLayout'

type LoginStage = 'credentials' | '2fa_prompt'

export default function LoginPage() {
  const { t } = useTranslation()

  // Credentials state
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [rememberMe, setRememberMe] = useState(true)
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
        await login(email, password, rememberMe)
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
              : t('auth.loginFailed'),
          )
        }
      } finally {
        setIsSubmitting(false)
      }
    },
    [email, password, rememberMe, login, navigate, t],
  )

  const handle2FASubmit = useCallback(
    async (e: FormEvent) => {
      e.preventDefault()
      if (!pendingToken || !totpCode) return
      setError(null)
      setIsSubmitting(true)

      try {
        await complete2FALogin(pendingToken, totpCode, rememberMe)
        navigate('/', { replace: true })
      } catch (err) {
        setError(
          err instanceof Error
            ? err.message
            : t('security.2fa.invalidCode'),
        )
      } finally {
        setIsSubmitting(false)
      }
    },
    [pendingToken, totpCode, rememberMe, complete2FALogin, navigate, t],
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
      <AuthLayout>
        <div className="animate-fade-up space-y-8">
          {/* Header */}
          <div className="space-y-1.5">
            <h1 className="text-2xl font-semibold tracking-tight text-foreground">
              {t('security.2fa.title')}
            </h1>
            <p className="text-sm text-muted-foreground">
              {useRecovery ? t('security.2fa.useRecoveryCode') : t('security.2fa.loginPrompt')}
            </p>
          </div>

          <form onSubmit={handle2FASubmit} className="space-y-5">
            <div className="space-y-2">
              <Label htmlFor="totp-code" className="text-sm font-medium">
                {useRecovery ? t('security.2fa.useRecoveryCode') : t('security.2fa.enterCode')}
              </Label>
              <Input
                id="totp-code"
                type="text"
                inputMode={useRecovery ? 'text' : 'numeric'}
                maxLength={useRecovery ? 20 : 6}
                placeholder={useRecovery ? 'xxxxxxxxxx' : '000 000'}
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
                className="h-11 text-base tracking-widest"
              />
            </div>

            {error && (
              <p className="animate-fade-in text-sm text-destructive">{error}</p>
            )}

            <Button
              type="submit"
              className="h-11 w-full text-sm font-medium"
              disabled={isSubmitting || !totpCode}
            >
              {isSubmitting ? (
                <span className="flex items-center gap-2">
                  <span className="h-4 w-4 animate-spin rounded-full border-2 border-primary-foreground border-t-transparent" />
                  {t('common.loading')}
                </span>
              ) : (
                t('security.2fa.verify')
              )}
            </Button>

            <div className="flex items-center justify-between">
              <button
                type="button"
                className="text-sm text-muted-foreground underline-offset-4 transition-colors hover:text-foreground hover:underline"
                onClick={() => {
                  setUseRecovery(!useRecovery)
                  setTotpCode('')
                  setError(null)
                }}
              >
                {useRecovery ? t('security.2fa.enterCode') : t('security.2fa.useRecoveryCode')}
              </button>
              <button
                type="button"
                className="text-sm text-muted-foreground underline-offset-4 transition-colors hover:text-foreground hover:underline"
                onClick={handleBackToCredentials}
              >
                {t('common.back')}
              </button>
            </div>
          </form>
        </div>
      </AuthLayout>
    )
  }

  // Credentials view (default)
  return (
    <AuthLayout>
      <div className="animate-fade-up space-y-8">
        {/* Logo + headline */}
        <div className="space-y-3">
          <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-primary/10 animate-scale-in-bounce">
            <svg
              width="22"
              height="22"
              viewBox="0 0 24 24"
              fill="none"
              className="text-primary"
              aria-hidden="true"
            >
              <path d="M12 2L2 7l10 5 10-5-10-5z" fill="currentColor" opacity="0.25" />
              <path
                d="M2 17l10 5 10-5"
                stroke="currentColor"
                strokeWidth="1.75"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
              <path
                d="M2 12l10 5 10-5"
                stroke="currentColor"
                strokeWidth="1.75"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </svg>
          </div>

          <div className="space-y-1">
            <h1 className="text-2xl font-semibold tracking-tight text-foreground">
              {t('auth.login')}
            </h1>
            <p className="text-sm text-muted-foreground">Cosmi &mdash; by Zentria</p>
          </div>
        </div>

        {/* Form */}
        <form onSubmit={handleCredentialsSubmit} className="space-y-5">
          {/* Email field */}
          <div className="space-y-2">
            <Label htmlFor="email" className="text-sm font-medium">
              {t('auth.email')}
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
              className="h-11"
            />
          </div>

          {/* Password field */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label htmlFor="password" className="text-sm font-medium">
                {t('auth.password')}
              </Label>
              <Link
                to="/forgot-password"
                className="text-xs text-muted-foreground underline-offset-4 transition-colors hover:text-foreground hover:underline"
              >
                {t('auth.forgotPassword')}
              </Link>
            </div>
            <Input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              disabled={isSubmitting}
              required
              className="h-11"
            />
          </div>

          {/* Remember me */}
          <div className="flex items-center gap-2.5">
            <Checkbox
              id="remember-me"
              checked={rememberMe}
              onCheckedChange={(checked) => setRememberMe(checked === true)}
              disabled={isSubmitting}
            />
            <Label
              htmlFor="remember-me"
              className="cursor-pointer select-none text-sm font-normal text-muted-foreground"
            >
              {t('auth.rememberMe')}
            </Label>
          </div>

          {error && (
            <p className="animate-fade-in text-sm text-destructive">{error}</p>
          )}

          <Button
            type="submit"
            className="h-11 w-full text-sm font-medium"
            disabled={isSubmitting}
          >
            {isSubmitting ? (
              <span className="flex items-center gap-2">
                <span className="h-4 w-4 animate-spin rounded-full border-2 border-primary-foreground border-t-transparent" />
                {t('common.loading')}
              </span>
            ) : (
              t('auth.login')
            )}
          </Button>

          <p className="text-center text-sm text-muted-foreground">
            {t('auth.noAccount')}{' '}
            <Link
              to="/register"
              className="font-medium text-primary underline-offset-4 transition-colors hover:underline"
            >
              {t('auth.register')}
            </Link>
          </p>
        </form>
      </div>
    </AuthLayout>
  )
}

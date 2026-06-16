/**
 * Reset-password page: set a new password using the single-use token from the
 * email link (?token=...). Calls the public /api/v1/auth/reset-password endpoint.
 *
 * Note: the production email link is a web URL (PASSWORD_RESET_BASE_URL). In the
 * Electron desktop app this route is reached via the hash query
 * (#/reset-password?token=...) — the web landing page mirrors this flow.
 */
import { useState, useCallback, type FormEvent } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { resetPassword } from '@/api/auth-reset-client'
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
import { AuthLayout } from './AuthLayout'

function Logo() {
  return (
    <div className="mx-auto mb-3 flex h-14 w-14 items-center justify-center rounded-2xl bg-primary/10 animate-scale-in-bounce">
      <svg width="28" height="28" viewBox="0 0 24 24" fill="none" className="text-primary">
        <path d="M12 2L2 7l10 5 10-5-10-5z" fill="currentColor" opacity="0.2" />
        <path d="M2 17l10 5 10-5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
        <path d="M2 12l10 5 10-5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
      </svg>
    </div>
  )
}

export default function ResetPasswordPage() {
  const { t } = useTranslation()
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token') ?? ''
  const navigate = useNavigate()

  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

  const handleSubmit = useCallback(
    async (e: FormEvent) => {
      e.preventDefault()
      setError(null)

      if (password !== confirmPassword) {
        setError(t('auth.passwordMismatch'))
        return
      }
      if (password.length < 8) {
        setError(t('auth.passwordTooShort'))
        return
      }

      setIsSubmitting(true)
      try {
        await resetPassword(token, password)
        toast.success(t('auth.reset.success'))
        navigate('/login', { replace: true })
      } catch (err) {
        setError(err instanceof Error ? err.message : t('auth.reset.failed'))
      } finally {
        setIsSubmitting(false)
      }
    },
    [token, password, confirmPassword, navigate, t],
  )

  // No token in the URL → the link is malformed; guide the user to request a new one.
  if (!token) {
    return (
      <AuthLayout>
        <Card className="w-full max-w-sm border-0 shadow-lg bg-card/80 backdrop-blur-sm">
          <CardHeader className="text-center">
            <Logo />
            <CardTitle className="text-2xl font-bold tracking-tight">
              {t('auth.reset.title')}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4 text-center">
            <p className="text-sm text-destructive">{t('auth.reset.missingToken')}</p>
            <Link
              to="/forgot-password"
              className="inline-block text-sm text-primary underline-offset-2 hover:underline"
            >
              {t('auth.forgot.title')}
            </Link>
          </CardContent>
        </Card>
      </AuthLayout>
    )
  }

  return (
    <AuthLayout>
      <Card className="w-full max-w-sm border-0 shadow-lg bg-card/80 backdrop-blur-sm">
        <CardHeader className="text-center">
          <Logo />
          <CardTitle className="text-2xl font-bold tracking-tight">
            {t('auth.reset.title')}
          </CardTitle>
          <CardDescription className="animate-fade-up" style={{ animationDelay: '200ms' }}>
            {t('auth.reset.subtitle')}
          </CardDescription>
        </CardHeader>

        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="password">{t('auth.reset.newPassword')}</Label>
              <Input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                disabled={isSubmitting}
                required
                autoFocus
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="confirmPassword">{t('auth.confirmPassword')}</Label>
              <Input
                id="confirmPassword"
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                disabled={isSubmitting}
                required
              />
            </div>

            {error && <p className="text-sm text-destructive">{error}</p>}

            <Button type="submit" className="w-full" disabled={isSubmitting}>
              {isSubmitting ? (
                <span className="flex items-center gap-2">
                  <span className="h-4 w-4 animate-spin rounded-full border-2 border-primary-foreground border-t-transparent" />
                  {t('common.loading')}
                </span>
              ) : (
                t('auth.reset.submit')
              )}
            </Button>

            <p className="text-center text-sm text-muted-foreground">
              <Link to="/login" className="text-primary underline-offset-2 hover:underline">
                {t('auth.backToLogin')}
              </Link>
            </p>
          </form>
        </CardContent>
      </Card>
    </AuthLayout>
  )
}

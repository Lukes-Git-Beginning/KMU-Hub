/**
 * Forgot-password page: request a password-reset link by email.
 *
 * Reached from the login screen ("Passwort vergessen?"). Calls the public
 * /api/v1/auth/forgot-password endpoint and always shows a generic confirmation
 * (enumeration-safe) so the page never reveals whether an address is registered.
 */
import { useState, useCallback, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { requestPasswordReset } from '@/api/auth-reset-client'
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

export default function ForgotPasswordPage() {
  const { t } = useTranslation()
  const [email, setEmail] = useState('')
  const [sent, setSent] = useState(false)
  const [isSubmitting, setIsSubmitting] = useState(false)

  const handleSubmit = useCallback(
    async (e: FormEvent) => {
      e.preventDefault()
      setIsSubmitting(true)
      try {
        await requestPasswordReset(email)
      } catch {
        // Enumeration-safe: show the generic confirmation even on a network
        // error so the UI never reveals whether the address exists.
      } finally {
        setSent(true)
        setIsSubmitting(false)
      }
    },
    [email],
  )

  return (
    <AuthLayout>
      <Card className="w-full max-w-sm border-0 shadow-lg bg-card/80 backdrop-blur-sm">
        <CardHeader className="text-center">
          <div className="mx-auto mb-3 flex h-14 w-14 items-center justify-center rounded-2xl bg-primary/10 animate-scale-in-bounce">
            <svg width="28" height="28" viewBox="0 0 24 24" fill="none" className="text-primary">
              <path d="M12 2L2 7l10 5 10-5-10-5z" fill="currentColor" opacity="0.2" />
              <path d="M2 17l10 5 10-5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
              <path d="M2 12l10 5 10-5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
          </div>
          <CardTitle className="text-2xl font-bold tracking-tight">
            {t('auth.forgot.title')}
          </CardTitle>
          <CardDescription className="animate-fade-up" style={{ animationDelay: '200ms' }}>
            {t('auth.forgot.subtitle')}
          </CardDescription>
        </CardHeader>

        <CardContent>
          {sent ? (
            <div className="space-y-4 text-center">
              <p className="text-sm text-muted-foreground">{t('auth.forgot.sent')}</p>
              <Link
                to="/login"
                className="inline-block text-sm text-primary underline-offset-2 hover:underline"
              >
                {t('auth.backToLogin')}
              </Link>
            </div>
          ) : (
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="email">{t('auth.email')}</Label>
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

              <Button type="submit" className="w-full" disabled={isSubmitting || !email}>
                {isSubmitting ? (
                  <span className="flex items-center gap-2">
                    <span className="h-4 w-4 animate-spin rounded-full border-2 border-primary-foreground border-t-transparent" />
                    {t('common.loading')}
                  </span>
                ) : (
                  t('auth.forgot.submit')
                )}
              </Button>

              <p className="text-center text-sm text-muted-foreground">
                <Link to="/login" className="text-primary underline-offset-2 hover:underline">
                  {t('auth.backToLogin')}
                </Link>
              </p>
            </form>
          )}
        </CardContent>
      </Card>
    </AuthLayout>
  )
}

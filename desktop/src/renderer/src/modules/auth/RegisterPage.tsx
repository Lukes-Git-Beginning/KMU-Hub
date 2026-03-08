import { useState, useCallback, type FormEvent } from 'react'
import { useNavigate, Link } from 'react-router-dom'
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

export default function RegisterPage() {
  const intl = useIntl()

  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

  const register = useAuthStore((s) => s.register)
  const navigate = useNavigate()

  const handleSubmit = useCallback(
    async (e: FormEvent) => {
      e.preventDefault()
      setError(null)

      if (password !== confirmPassword) {
        setError(intl.formatMessage({ id: 'auth.passwordMismatch' }))
        return
      }

      if (password.length < 8) {
        setError(intl.formatMessage({ id: 'auth.passwordTooShort' }))
        return
      }

      setIsSubmitting(true)

      try {
        await register(email, password, firstName, lastName)
        navigate('/', { replace: true })
      } catch (err) {
        setError(
          err instanceof Error
            ? err.message
            : intl.formatMessage({ id: 'auth.registerFailed' }),
        )
      } finally {
        setIsSubmitting(false)
      }
    },
    [firstName, lastName, email, password, confirmPassword, register, navigate, intl],
  )

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <Card className="w-full max-w-sm">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl font-bold tracking-tight animate-scale-in-bounce">
            KMU Hub
          </CardTitle>
          <CardDescription className="animate-fade-up" style={{ animationDelay: '200ms' }}>
            <FormattedMessage id="auth.register" />
          </CardDescription>
        </CardHeader>

        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-2">
                <Label htmlFor="firstName">
                  <FormattedMessage id="auth.firstName" />
                </Label>
                <Input
                  id="firstName"
                  type="text"
                  value={firstName}
                  onChange={(e) => setFirstName(e.target.value)}
                  disabled={isSubmitting}
                  required
                  autoFocus
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="lastName">
                  <FormattedMessage id="auth.lastName" />
                </Label>
                <Input
                  id="lastName"
                  type="text"
                  value={lastName}
                  onChange={(e) => setLastName(e.target.value)}
                  disabled={isSubmitting}
                  required
                />
              </div>
            </div>

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

            <div className="space-y-2">
              <Label htmlFor="confirmPassword">
                <FormattedMessage id="auth.confirmPassword" />
              </Label>
              <Input
                id="confirmPassword"
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
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
                <FormattedMessage id="auth.register" />
              )}
            </Button>

            <p className="text-center text-sm text-muted-foreground">
              <FormattedMessage id="auth.alreadyHaveAccount" />{' '}
              <Link to="/login" className="text-primary underline-offset-2 hover:underline">
                <FormattedMessage id="auth.login" />
              </Link>
            </p>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}

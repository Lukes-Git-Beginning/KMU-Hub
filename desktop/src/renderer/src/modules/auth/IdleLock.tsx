/**
 * IdleLock — DSGVO-compliant idle auto-lock overlay.
 *
 * Mounts as a sibling to the authenticated app shell and shows a
 * full-screen re-authentication overlay whenever the user has been
 * inactive for IDLE_TIMEOUT_MS milliseconds.
 *
 * Design constraints (from CLAUDE.md / motion.ts):
 *  - Only transform / opacity animations (GPU only, no width/height/margin).
 *  - No emojis; personality via SVG + wording.
 *  - Motion tokens imported from lib/motion.ts.
 *  - i18n via useTranslation(), keys in namespace "translation" (flat dot-notation).
 */
import { useState, useEffect, useRef, useCallback, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '@/stores/auth'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { duration, easing } from '@/lib/motion'

/** Inactivity threshold before the lock overlay appears. */
export const IDLE_TIMEOUT_MS = 15 * 60 * 1000

/** DOM events that count as user activity. */
const ACTIVITY_EVENTS = [
  'mousemove',
  'mousedown',
  'keydown',
  'touchstart',
  'scroll',
  'wheel',
] as const

// CSS transition values derived from motion tokens (no magic numbers in components).
const OVERLAY_TRANSITION = `opacity ${duration.modal}s cubic-bezier(${easing.outUI.join(',')})`
const CARD_TRANSITION = `opacity ${duration.modal}s cubic-bezier(${easing.outUI.join(',')}), transform ${duration.modal}s cubic-bezier(${easing.outUI.join(',')})`

export function IdleLock() {
  const { t } = useTranslation()

  const user = useAuthStore((s) => s.user)
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const login = useAuthStore((s) => s.login)
  const logout = useAuthStore((s) => s.logout)

  const [locked, setLocked] = useState(false)
  const [visible, setVisible] = useState(false) // controls CSS transition
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

  // Holds the setTimeout handle so we can clear + reset it on activity.
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const passwordInputRef = useRef<HTMLInputElement>(null)

  /** Reset the inactivity timer. Called on every activity event. */
  const resetTimer = useCallback(() => {
    if (timerRef.current !== null) {
      clearTimeout(timerRef.current)
    }
    timerRef.current = setTimeout(() => {
      setLocked(true)
    }, IDLE_TIMEOUT_MS)
  }, [])

  // Start / stop the idle timer based on auth state.
  useEffect(() => {
    if (!isAuthenticated) {
      // Not logged in — no timer needed.
      if (timerRef.current !== null) {
        clearTimeout(timerRef.current)
        timerRef.current = null
      }
      return
    }

    // Attach activity listeners (passive for scroll/touch perf).
    const passiveEvents = new Set(['scroll', 'touchstart', 'wheel'])
    const handler = () => resetTimer()

    ACTIVITY_EVENTS.forEach((event) => {
      window.addEventListener(event, handler, {
        passive: passiveEvents.has(event),
        capture: true,
      })
    })

    // Kick off the first timer.
    resetTimer()

    return () => {
      ACTIVITY_EVENTS.forEach((event) => {
        window.removeEventListener(event, handler, { capture: true })
      })
      if (timerRef.current !== null) {
        clearTimeout(timerRef.current)
        timerRef.current = null
      }
    }
  }, [isAuthenticated, resetTimer])

  // Animate in when locked state changes.
  useEffect(() => {
    if (locked) {
      // Mount first, then trigger transition on next frame.
      setVisible(false)
      requestAnimationFrame(() => {
        requestAnimationFrame(() => setVisible(true))
      })
      // Focus the password input after animation starts.
      const focusTimer = setTimeout(() => {
        passwordInputRef.current?.focus()
      }, Math.round(duration.modal * 1000))
      return () => clearTimeout(focusTimer)
    } else {
      setVisible(false)
    }
  }, [locked])

  const handleUnlock = useCallback(
    async (e: FormEvent) => {
      e.preventDefault()
      if (!user?.email || !password) return

      setError(null)
      setIsSubmitting(true)

      try {
        await login(user.email, password)
        // Success — close overlay and restart timer.
        setLocked(false)
        setPassword('')
        resetTimer()
      } catch {
        setError(t('appLock.wrongPassword'))
        setPassword('')
        passwordInputRef.current?.focus()
      } finally {
        setIsSubmitting(false)
      }
    },
    [user?.email, password, login, resetTimer, t],
  )

  const handleLogout = useCallback(async () => {
    setLocked(false)
    setPassword('')
    await logout()
  }, [logout])

  // Nothing to render when not locked or not authenticated.
  if (!isAuthenticated || !locked) return null

  const displayName =
    user
      ? `${user.firstName ?? ''} ${user.lastName ?? ''}`.trim() || user.email
      : ''

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label={t('appLock.title')}
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 9999,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        backgroundColor: 'hsl(var(--background) / 0.92)',
        backdropFilter: 'blur(16px) saturate(1.4)',
        WebkitBackdropFilter: 'blur(16px) saturate(1.4)',
        opacity: visible ? 1 : 0,
        transition: OVERLAY_TRANSITION,
      }}
    >
      <div
        style={{
          width: '100%',
          maxWidth: '26rem',
          padding: '0 1rem',
          transform: visible ? 'translateY(0) scale(1)' : 'translateY(12px) scale(0.97)',
          opacity: visible ? 1 : 0,
          transition: CARD_TRANSITION,
        }}
      >
        {/* Card */}
        <div
          style={{
            borderRadius: '1rem',
            border: '1px solid hsl(var(--border) / 0.6)',
            backgroundColor: 'hsl(var(--card))',
            boxShadow:
              '0 24px 64px -12px hsl(var(--foreground) / 0.12), 0 0 0 1px hsl(var(--border) / 0.3)',
            padding: '2rem',
          }}
        >
          {/* Lock icon — custom SVG, no emoji */}
          <div
            style={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              marginBottom: '1.5rem',
              gap: '0.75rem',
            }}
          >
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                width: '3.5rem',
                height: '3.5rem',
                borderRadius: '0.875rem',
                backgroundColor: 'hsl(var(--primary) / 0.1)',
              }}
            >
              <svg
                width="28"
                height="28"
                viewBox="0 0 24 24"
                fill="none"
                aria-hidden="true"
                style={{ color: 'hsl(var(--primary))' }}
              >
                <rect
                  x="3"
                  y="11"
                  width="18"
                  height="11"
                  rx="2"
                  stroke="currentColor"
                  strokeWidth="1.75"
                />
                <path
                  d="M7 11V7a5 5 0 0 1 10 0v4"
                  stroke="currentColor"
                  strokeWidth="1.75"
                  strokeLinecap="round"
                />
                <circle cx="12" cy="16" r="1.25" fill="currentColor" />
              </svg>
            </div>

            <div style={{ textAlign: 'center' }}>
              <h1
                style={{
                  margin: 0,
                  fontSize: '1.25rem',
                  fontWeight: 700,
                  letterSpacing: '-0.015em',
                  color: 'hsl(var(--foreground))',
                }}
              >
                {t('appLock.title')}
              </h1>
              <p
                style={{
                  margin: '0.25rem 0 0',
                  fontSize: '0.875rem',
                  color: 'hsl(var(--muted-foreground))',
                }}
              >
                {t('appLock.subtitle', { name: displayName })}
              </p>
            </div>
          </div>

          {/* Re-auth form */}
          <form onSubmit={handleUnlock} style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
              <Label htmlFor="idle-lock-password">
                {t('appLock.passwordLabel')}
              </Label>
              <Input
                id="idle-lock-password"
                ref={passwordInputRef}
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                disabled={isSubmitting}
                required
                autoComplete="current-password"
              />
            </div>

            {error && (
              <p
                role="alert"
                style={{
                  margin: 0,
                  fontSize: '0.875rem',
                  color: 'hsl(var(--destructive))',
                }}
              >
                {error}
              </p>
            )}

            <Button type="submit" className="w-full" disabled={isSubmitting || !password}>
              {isSubmitting ? (
                <span style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                  <span
                    style={{
                      display: 'inline-block',
                      width: '1rem',
                      height: '1rem',
                      borderRadius: '50%',
                      border: '2px solid hsl(var(--primary-foreground) / 0.4)',
                      borderTopColor: 'hsl(var(--primary-foreground))',
                      animation: 'spin 0.7s linear infinite',
                    }}
                  />
                  {t('common.loading')}
                </span>
              ) : (
                t('appLock.unlock')
              )}
            </Button>

            <div style={{ display: 'flex', justifyContent: 'center', paddingTop: '0.25rem' }}>
              <button
                type="button"
                onClick={handleLogout}
                style={{
                  background: 'none',
                  border: 'none',
                  padding: '0.25rem 0',
                  fontSize: '0.875rem',
                  color: 'hsl(var(--muted-foreground))',
                  cursor: 'pointer',
                  textDecoration: 'underline',
                  textDecorationColor: 'transparent',
                  textUnderlineOffset: '2px',
                  transition: `color ${duration.fast}s`,
                }}
                onMouseEnter={(e) => {
                  ;(e.currentTarget as HTMLButtonElement).style.textDecorationColor = 'currentColor'
                }}
                onMouseLeave={(e) => {
                  ;(e.currentTarget as HTMLButtonElement).style.textDecorationColor = 'transparent'
                }}
              >
                {t('appLock.logout')}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}

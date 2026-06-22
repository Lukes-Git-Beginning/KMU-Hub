/**
 * App-weites Öffnungs-/Flug-Overlay (über dem Router).
 *
 * Maskiert den weißen Suspense-/Skelett-Flash beim Übergang ins Dashboard und
 * ersetzt ihn durch eine „Flug-nach-vorn / rein-ins-Bild"-Animation. Zwei
 * Einstiegs-Pfade, beide enden in DERSELBEN Fly-in-/Zoom-Phase (siehe
 * `stores/launch.ts`):
 *
 *  - Fall A (Start MIT Token): CosmiLaunch-Logo-Intro läuft beim App-Start
 *    (währenddessen läuft `auth.initialize()`), danach Fly-in → Dashboard.
 *    Ausgelöst synchron in `main.tsx` via `initStartupLaunch()`.
 *  - Fall B (post-login): nur die Fly-in-Phase, ausgelöst aus `LoginPage`.
 *
 * Die Choreografie ist GPU-only (transform/opacity, Tokens/Keyframes in
 * `styles/animations.css`). Bei `prefers-reduced-motion` wird gar kein Overlay
 * gestartet (weder Boot noch `LoginPage`) → sofort rein, kein Zoom.
 */
import { useCallback, useEffect, type AnimationEvent } from 'react'
import { cn } from '@/lib/cn'
import { useAuthStore } from '@/stores/auth'
import { useLaunchStore } from '@/stores/launch'
import { SpaceBackground } from './SpaceBackground'
import { StaticBrandMark } from './AuthLayout'
import CosmiLaunch from './CosmiLaunch'

/** Dev-/QA-Build (DEV_BYPASS_AUTH): Start gilt als authentifiziert. */
const DEV_AUTHED_START = import.meta.env.DEV

/** Fallback, falls CosmiLaunch.onComplete nie feuert (vgl. AuthLayout). */
const INTRO_FALLBACK_MS = 9000
/** Fallback, falls animationend des Fly-ins nie feuert. > --dur-flyin. */
const FLYIN_FALLBACK_MS = 1400

export function LaunchOverlay() {
  const active = useLaunchStore((s) => s.active)
  const mode = useLaunchStore((s) => s.mode)
  const phase = useLaunchStore((s) => s.phase)
  const toFlyin = useLaunchStore((s) => s.toFlyin)
  const end = useLaunchStore((s) => s.end)

  // Fall A: Logo-Intro fertig → in die Fly-in-Phase (oder Login freigeben,
  // falls das Token nach der parallelen Validierung doch ungültig war).
  const handleIntroDone = useCallback(() => {
    const authed = DEV_AUTHED_START || useAuthStore.getState().isAuthenticated
    if (authed) toFlyin()
    else end()
  }, [toFlyin, end])

  // Sicherheits-Fallbacks, falls onComplete/animationend ausbleiben.
  useEffect(() => {
    if (!active) return
    if (phase === 'intro') {
      const id = window.setTimeout(handleIntroDone, INTRO_FALLBACK_MS)
      return () => window.clearTimeout(id)
    }
    const id = window.setTimeout(end, FLYIN_FALLBACK_MS)
    return () => window.clearTimeout(id)
  }, [active, phase, handleIntroDone, end])

  // Fly-in zu Ende → Overlay schließen. Nur die Szenen-Animation zählt; die
  // Logo-Zoom-Animation bubblet ebenfalls hoch und wird hier ignoriert.
  const handleSceneAnimEnd = useCallback(
    (e: AnimationEvent<HTMLDivElement>) => {
      if (e.animationName === 'cosmi-flyin-scene') end()
    },
    [end],
  )

  if (!active) return null

  const flying = phase === 'flyin'

  return (
    <div
      className={cn('cosmi-startup', flying && 'is-flying')}
      onAnimationEnd={flying ? handleSceneAnimEnd : undefined}
      aria-hidden
    >
      <SpaceBackground />
      <div className={cn('cosmi-startup-logo', flying && 'is-flying')}>
        {mode === 'full' ? (
          // Persistente Instanz: hält nach onComplete ihren Endframe (COSMI-
          // Wortmarke) → nahtloser Übergang in den Zoom.
          <CosmiLaunch size={400} shiftLogo="none" onWordmark={() => {}} onComplete={handleIntroDone} />
        ) : (
          <StaticBrandMark />
        )}
      </div>
    </div>
  )
}

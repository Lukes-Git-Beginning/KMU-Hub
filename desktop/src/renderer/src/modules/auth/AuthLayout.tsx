import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { cn } from '@/lib/cn'
import { branding } from '@/config/branding'
import { SpaceBackground } from './SpaceBackground'
import CosmiLaunch from './CosmiLaunch'

/**
 * Auth-/Launch-Fläche für alle Auth-Routen (Login, 2FA, Register, Passwort-Reset).
 *
 * Beim ersten Erscheinen pro App-Start läuft die CosmiLaunch-Animation
 * bildschirm-zentriert; danach gleitet das Logo in die rechte Hälfte (GPU-
 * transform) und das Login-Fenster faded links auf demselben Weltraum-
 * Hintergrund ein. Bei `prefers-reduced-motion` oder erneutem Besuch (Flag in
 * sessionStorage) wird direkt der gesetzte Zustand gezeigt — ohne Intro.
 */

const LAUNCH_FLAG = 'cosmi:launch-played'
/** Fallback, falls onComplete der Animation nie feuert (defensiv). */
const SETTLE_FALLBACK_MS = 9000

function prefersReducedMotion(): boolean {
  return (
    typeof window !== 'undefined' &&
    typeof window.matchMedia === 'function' &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches
  )
}

function launchAlreadyPlayed(): boolean {
  try {
    return sessionStorage.getItem(LAUNCH_FLAG) === '1'
  } catch {
    return false
  }
}

function markLaunchPlayed(): void {
  try {
    sessionStorage.setItem(LAUNCH_FLAG, '1')
  } catch {
    // sessionStorage nicht verfügbar → Intro läuft ggf. erneut, unkritisch.
  }
}

export function AuthLayout({ children }: { children: ReactNode }) {
  // Einmal beim Mount entscheiden, ob das Intro spielt.
  const playIntro = useMemo(() => !prefersReducedMotion() && !launchAlreadyPlayed(), [])
  const [phase, setPhase] = useState<'intro' | 'settled'>(playIntro ? 'intro' : 'settled')
  const settledRef = useRef(false)

  const settle = useCallback(() => {
    if (settledRef.current) return
    settledRef.current = true
    markLaunchPlayed()
    setPhase('settled')
  }, [])

  useEffect(() => {
    if (!playIntro) {
      markLaunchPlayed()
      return
    }
    const id = window.setTimeout(settle, SETTLE_FALLBACK_MS)
    return () => window.clearTimeout(id)
  }, [playIntro, settle])

  const settled = phase === 'settled'

  return (
    <div className="relative min-h-screen overflow-hidden bg-[#090c14] text-white">
      <SpaceBackground />

      {/* Marke — startet bildschirm-zentriert, gleitet in die rechte Hälfte */}
      <div className={cn('cosmi-launch-logo', settled && 'is-settled')}>
        {playIntro ? (
          <CosmiLaunch size={400} shiftLogo="none" onWordmark={() => {}} onComplete={settle} />
        ) : (
          <StaticBrandMark />
        )}
      </div>

      {/* Login-Fenster — faded links ein, sobald das Intro gesetzt ist */}
      <div className={cn('cosmi-auth-form', settled && 'is-visible')}>
        <div className="w-[min(92vw,380px)] rounded-3xl border border-white/10 bg-background/95 p-8 text-foreground shadow-2xl backdrop-blur-xl">
          {children}
        </div>
      </div>
    </div>
  )
}

/** Statische Marke für reduced-motion / Wiederbesuch (kein Re-Play der Formung).
 *  Auch vom Fly-in-Overlay (Fall B) als zentriertes Logo wiederverwendet. */
export function StaticBrandMark() {
  return (
    <div className="flex select-none flex-col items-center gap-4">
      <img
        src={branding.cosmi.icon128}
        alt=""
        className="h-20 w-20 object-contain"
        aria-hidden
      />
      <span className="text-3xl font-light tracking-[0.3em] text-white/90">COSMI</span>
    </div>
  )
}

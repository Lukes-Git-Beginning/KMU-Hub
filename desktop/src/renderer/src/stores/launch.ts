/**
 * Launch-/Transition-Store (Zustand).
 *
 * Steuert das app-weite Öffnungs-/Flug-Overlay, das den weißen Suspense-/
 * Skelett-Flash beim Übergang ins Dashboard maskiert. Zwei Einstiegs-Pfade,
 * beide enden in DERSELBEN Fly-in-/Zoom-Animation:
 *
 *  - Fall A (`mode: 'full'`)  — Start MIT Token: CosmiLaunch-Logo-Intro läuft,
 *    danach Fly-in (Logo zoomt nach vorn + faded), dann ist das Dashboard offen.
 *  - Fall B (`mode: 'flyin'`) — nach erfolgreichem Login: nur die Fly-in-Phase
 *    (ohne vorheriges Logo-Intro), legt sich über das ladende Dashboard.
 *
 * Das Overlay (`LaunchOverlay`) hängt app-weit über dem Router und liest
 * `active`/`mode`/`phase`. Die eigentliche Choreografie ist GPU-only (CSS in
 * `styles/animations.css`).
 *
 * Fall A wird per `initStartupLaunch()` VOR dem ersten Render entschieden
 * (`main.tsx`), damit das Overlay schon den ersten Frame abdeckt — ein
 * `useEffect` liefe erst nach dem Paint und ließe den Flash kurz sichtbar.
 */
import { create } from 'zustand'

/** Phase des Overlays. `intro` = CosmiLaunch-Logoformung (nur Fall A). */
export type LaunchPhase = 'intro' | 'flyin'

/** Modus = welcher Einstiegs-Pfad das Overlay ausgelöst hat. */
export type LaunchMode = 'full' | 'flyin'

interface LaunchState {
  /** Overlay sichtbar? */
  active: boolean
  /** Welcher Pfad das Overlay gestartet hat (null = inaktiv). */
  mode: LaunchMode | null
  /** Aktuelle Phase innerhalb des aktiven Overlays. */
  phase: LaunchPhase

  /** Fall A: Logo-Intro starten (läuft während `auth.initialize()`). */
  startFull: () => void
  /** Fall B: direkt in die Fly-in-/Öffnungs-Animation (post-login). */
  startFlyin: () => void
  /** Fall A: Logo-Intro fertig → in die gemeinsame Fly-in-Phase wechseln. */
  toFlyin: () => void
  /** Overlay beenden → unmounten, Dashboard liegt frei. */
  end: () => void
}

export const useLaunchStore = create<LaunchState>()((set) => ({
  active: false,
  mode: null,
  phase: 'intro',

  startFull() {
    set({ active: true, mode: 'full', phase: 'intro' })
  },
  startFlyin() {
    set({ active: true, mode: 'flyin', phase: 'flyin' })
  },
  toFlyin() {
    set({ phase: 'flyin' })
  },
  end() {
    set({ active: false, mode: null, phase: 'intro' })
  },
}))

/** `prefers-reduced-motion: reduce` aktiv? (defensiv, SSR-/Test-sicher) */
export function prefersReducedMotion(): boolean {
  return (
    typeof window !== 'undefined' &&
    typeof window.matchMedia === 'function' &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches
  )
}

/** Geteiltes Session-Flag mit AuthLayout — verhindert doppeltes Logo-Intro. */
const LAUNCH_FLAG = 'cosmi:launch-played'

/** Routen ohne Startup-Splash: Auth-/Gast-Screens, Utility-Popouts, Previews. */
const GUEST_BOOT_ROUTES = new Set([
  '/login',
  '/register',
  '/forgot-password',
  '/reset-password',
  '/launch-preview',
  '/flyin-preview',
  '/compose-window',
  '/employee-wizard-window',
  '/editor-window',
])

function bootHashPath(): string {
  const h = (window.location.hash || '').replace(/^#/, '')
  return h.split('?')[0] || '/'
}

function launchAlreadyPlayed(): boolean {
  try {
    return sessionStorage.getItem(LAUNCH_FLAG) === '1'
  } catch {
    return false
  }
}

export function markLaunchPlayed(): void {
  try {
    sessionStorage.setItem(LAUNCH_FLAG, '1')
  } catch {
    // sessionStorage nicht verfügbar → Intro läuft ggf. erneut, unkritisch.
  }
}

/**
 * Fall A entscheiden — beim App-Start VOR dem ersten Render aufrufen.
 *
 * Dev-Build (DEV_BYPASS_AUTH): Start gilt synchron als „mit Token" → Splash.
 * Prod-Build: nur wenn ein gespeichertes Token vorliegt (lokaler IPC; das
 * ~7s-Logo-Intro überbrückt die parallele Token-Validierung). Kein Splash bei
 * reduced-motion, auf Gast-/Utility-Routen oder wenn diese Session schon lief.
 */
export function initStartupLaunch(): void {
  if (typeof window === 'undefined') return
  if (prefersReducedMotion()) return
  if (launchAlreadyPlayed()) return
  if (GUEST_BOOT_ROUTES.has(bootHashPath())) return

  if (import.meta.env.DEV) {
    markLaunchPlayed()
    useLaunchStore.getState().startFull()
    return
  }

  void (async () => {
    try {
      const stored = await window.electronAPI?.auth?.getStoredTokens?.()
      if (stored) {
        markLaunchPlayed()
        useLaunchStore.getState().startFull()
      }
    } catch {
      // Kein Token lesbar → kein Splash, regulärer Login-/Dashboard-Pfad.
    }
  })()
}

import { useMemo, type CSSProperties } from 'react'
import { cn } from '@/lib/cn'

/**
 * Durchgehender Weltraum-Hintergrund für die Auth-/Launch-Fläche.
 *
 * Bewusst IDENTISCH zur ehemaligen CosmiLaunch-„Kachel": derselbe sehr subtile
 * Radial-Gradient (BG_MID → BG_DARK) plus weiße, sanft twinkelnde Sterne. Die
 * Kachel selbst ist in CosmiLaunch entfernt → alles verschmilzt zu einem Bild.
 */

// Exakt die Werte aus CosmiLaunch.tsx (Single Source: dort C_COLOR/BG_*).
const BG_MID = '#0c0f18'
const BG_DARK = '#090c14'

/** Deterministischer PRNG → stabile Sternpositionen über Re-Renders/HMR. */
function mulberry32(seed: number): () => number {
  let s = seed >>> 0
  return () => {
    s = (s + 0x6d2b79f5) >>> 0
    let t = Math.imul(s ^ (s >>> 15), 1 | s)
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

interface Star {
  x: number
  y: number
  r: number
  o: number
  dur: number
  delay: number
}

function makeStars(count: number, seed: number): Star[] {
  const rnd = mulberry32(seed)
  return Array.from({ length: count }, () => ({
    x: rnd() * 100,
    y: rnd() * 100,
    r: 0.4 + rnd() * 1.0,
    o: 0.14 + rnd() * 0.38, // matcht genStars: rnd()*0.38 + 0.14 → 0.14–0.52
    dur: 2.8 + rnd() * 3.8, // Twinkle-Dauer (s)
    delay: -rnd() * 6, // negativ → Phasen sofort gestaffelt
  }))
}

export function SpaceBackground({ className }: { className?: string }) {
  const stars = useMemo(() => makeStars(135, 20260622), [])

  return (
    <div
      aria-hidden
      className={cn('pointer-events-none absolute inset-0 overflow-hidden', className)}
      style={{
        background: `radial-gradient(92% 80% at 50% 36%, ${BG_MID} 0%, ${BG_DARK} 100%)`,
      }}
    >
      <svg
        className="absolute inset-0 h-full w-full"
        preserveAspectRatio="xMidYMid slice"
        aria-hidden
      >
        {stars.map((s, i) => (
          <circle
            key={i}
            className="cosmi-star"
            cx={`${s.x}%`}
            cy={`${s.y}%`}
            r={s.r}
            fill="#ffffff"
            opacity={s.o}
            style={
              {
                '--o': s.o,
                animationDuration: `${s.dur}s`,
                animationDelay: `${s.delay}s`,
              } as CSSProperties
            }
          />
        ))}
      </svg>
    </div>
  )
}

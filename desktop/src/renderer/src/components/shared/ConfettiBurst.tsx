/**
 * ConfettiBurst — celebratory confetti particle animation.
 *
 * Renders a burst of colored confetti particles that fall from
 * the center of the screen. Used for deal-won celebrations.
 * Respects prefers-reduced-motion (shows nothing).
 */
import { useState, useEffect, useCallback } from 'react'

const COLORS = ['#f97316', '#10b981', '#4f46e5', '#ec4899', '#f59e0b', '#8b5cf6']
const PARTICLE_COUNT = 35

interface Particle {
  id: number
  x: number
  color: string
  delay: number
  duration: number
  rotation: number
  size: number
  drift: number
}

interface ConfettiBurstProps {
  /** When true, triggers the burst */
  active: boolean
  /** Called when animation finishes */
  onComplete?: () => void
}

export function ConfettiBurst({ active, onComplete }: ConfettiBurstProps) {
  const [particles, setParticles] = useState<Particle[]>([])

  const stableOnComplete = useCallback(() => onComplete?.(), [onComplete])

  useEffect(() => {
    if (!active) {
      setParticles([])
      return
    }

    // Respect reduced motion
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
      stableOnComplete()
      return
    }

    const newParticles: Particle[] = Array.from({ length: PARTICLE_COUNT }, (_, i) => ({
      id: i,
      x: 35 + Math.random() * 30,
      color: COLORS[Math.floor(Math.random() * COLORS.length)],
      delay: Math.random() * 250,
      duration: 900 + Math.random() * 700,
      rotation: Math.random() * 720 - 360,
      size: 5 + Math.random() * 7,
      drift: (Math.random() - 0.5) * 120,
    }))

    setParticles(newParticles)

    const timer = setTimeout(() => {
      setParticles([])
      stableOnComplete()
    }, 2000)

    return () => clearTimeout(timer)
  }, [active, stableOnComplete])

  if (particles.length === 0) return null

  return (
    <div className="pointer-events-none fixed inset-0 z-[100] overflow-hidden" aria-hidden>
      {particles.map((p) => (
        <div
          key={p.id}
          className="absolute"
          style={{
            top: '25%',
            left: `${p.x}%`,
            width: p.size,
            height: p.size * 0.6,
            backgroundColor: p.color,
            borderRadius: '2px',
            animation: `confetti-fall ${p.duration}ms cubic-bezier(0.25, 0.46, 0.45, 0.94) ${p.delay}ms both`,
            ['--confetti-rot' as string]: `${p.rotation}deg`,
            ['--confetti-drift' as string]: `${p.drift}px`,
          }}
        />
      ))}
    </div>
  )
}

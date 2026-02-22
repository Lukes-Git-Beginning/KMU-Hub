/**
 * 3D tilt effect hook for hero cards.
 *
 * Tracks mouse position over the element and applies a subtle
 * perspective rotation + scale. Sets --tilt-x/--tilt-y CSS vars
 * for the specular shine overlay (.tilt-shine).
 *
 * Respects prefers-reduced-motion automatically.
 */
import { useRef, useCallback, useEffect } from 'react'

interface TiltOptions {
  /** Max rotation degrees (default 4) */
  maxAngle?: number
  /** Scale multiplier on hover (default 1.02) */
  scale?: number
  /** Whether effect is enabled (default true) */
  enabled?: boolean
}

export function useTiltEffect<T extends HTMLElement = HTMLDivElement>(
  options: TiltOptions = {}
) {
  const { maxAngle = 4, scale = 1.02, enabled = true } = options
  const ref = useRef<T>(null)
  const rafRef = useRef(0)

  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      const el = ref.current
      if (!el) return

      cancelAnimationFrame(rafRef.current)
      rafRef.current = requestAnimationFrame(() => {
        const rect = el.getBoundingClientRect()
        const x = (e.clientX - rect.left) / rect.width
        const y = (e.clientY - rect.top) / rect.height

        const rotateX = (0.5 - y) * maxAngle * 2
        const rotateY = (x - 0.5) * maxAngle * 2

        el.style.transform = `perspective(800px) rotateX(${rotateX.toFixed(2)}deg) rotateY(${rotateY.toFixed(2)}deg) scale3d(${scale}, ${scale}, ${scale})`
        el.style.setProperty('--tilt-x', `${(x * 100).toFixed(1)}%`)
        el.style.setProperty('--tilt-y', `${(y * 100).toFixed(1)}%`)
      })
    },
    [maxAngle, scale]
  )

  const handleMouseLeave = useCallback(() => {
    const el = ref.current
    if (!el) return
    cancelAnimationFrame(rafRef.current)
    el.style.transform = ''
    el.style.removeProperty('--tilt-x')
    el.style.removeProperty('--tilt-y')
  }, [])

  useEffect(() => {
    const el = ref.current
    if (!el || !enabled) return

    // Respect reduced motion preference
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return

    el.addEventListener('mousemove', handleMouseMove)
    el.addEventListener('mouseleave', handleMouseLeave)

    return () => {
      el.removeEventListener('mousemove', handleMouseMove)
      el.removeEventListener('mouseleave', handleMouseLeave)
      cancelAnimationFrame(rafRef.current)
    }
  }, [handleMouseMove, handleMouseLeave, enabled])

  return ref
}

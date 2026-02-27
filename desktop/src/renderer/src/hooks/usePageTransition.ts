/**
 * CSS-based page transition hook.
 *
 * Returns a class name that triggers a fade-up entrance animation
 * whenever the route key changes. Works with prefers-reduced-motion.
 */
import { useState, useEffect, useRef } from 'react'

export function usePageTransition(routeKey: string): string {
  const [transitionClass, setTransitionClass] = useState('page-enter')
  const prevKey = useRef(routeKey)

  useEffect(() => {
    if (routeKey !== prevKey.current) {
      // Trigger re-entrance animation on route change
      setTransitionClass('')
      const frame = requestAnimationFrame(() => {
        setTransitionClass('page-enter')
      })
      prevKey.current = routeKey
      return () => cancelAnimationFrame(frame)
    }
  }, [routeKey])

  return transitionClass
}

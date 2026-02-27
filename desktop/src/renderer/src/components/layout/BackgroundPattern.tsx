/**
 * Background pattern layer — renders a themed SVG pattern behind the app.
 *
 * Visible through translucent UI in glass/crystal mode.
 * In solid mode, the pattern is hidden but sticker classes activate on <html>.
 */
import { useEffect } from 'react'
import { useUIStore } from '@/stores/ui'
import { BACKGROUND_PATTERNS } from '@/config/background-patterns'

export function BackgroundPattern() {
  const backgroundPattern = useUIStore((s) => s.backgroundPattern)
  const uiLook = useUIStore((s) => s.uiLook)

  const pattern = BACKGROUND_PATTERNS[backgroundPattern]
  const isActive = pattern && pattern.id !== 'none'
  const showBg = isActive && uiLook !== 'solid'

  // Sync sticker + pattern classes on <html>
  useEffect(() => {
    const el = document.documentElement

    // Remove all sticker classes
    el.classList.remove(
      'sticker-pflanzen', 'sticker-hunde', 'sticker-geometrisch',
      'sticker-wellen', 'sticker-abstrakt', 'has-bg-pattern'
    )

    if (isActive) {
      el.classList.add('has-bg-pattern')
      if (pattern.stickerClass) {
        el.classList.add(pattern.stickerClass)
      }
    }
  }, [backgroundPattern, isActive, pattern])

  if (!showBg || !pattern.bgClass) return null

  return (
    <div
      className={`absolute inset-0 z-[1] pointer-events-none ${pattern.bgClass}`}
      aria-hidden="true"
    />
  )
}

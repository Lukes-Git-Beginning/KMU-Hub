/**
 * TextReveal — word-by-word staggered entrance animation.
 *
 * Splits text into words and wraps each in a span with staggered
 * fade-up animation. Used for hero greeting text on the Dashboard.
 */
import { useMemo } from 'react'

interface TextRevealProps {
  text: string
  className?: string
  /** Delay between each word in ms (default 70) */
  wordDelay?: number
  /** Base delay before animation starts in ms (default 0) */
  baseDelay?: number
}

export function TextReveal({
  text,
  className = '',
  wordDelay = 70,
  baseDelay = 0,
}: TextRevealProps) {
  const words = useMemo(() => text.split(' '), [text])

  return (
    <span className={className} aria-label={text}>
      {words.map((word, i) => (
        <span
          key={i}
          className="inline-block animate-fade-up"
          style={{
            animationDelay: `${baseDelay + i * wordDelay}ms`,
            animationFillMode: 'both',
          }}
          aria-hidden
        >
          {word}
          {i < words.length - 1 ? '\u00A0' : ''}
        </span>
      ))}
    </span>
  )
}

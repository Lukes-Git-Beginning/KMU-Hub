/**
 * AnimatedCheckmark — SVG circle + checkmark with draw animation.
 *
 * Circle draws first (500ms), then checkmark draws (300ms, 350ms delay).
 * Used for success states like invoice-sent confirmation.
 * Respects prefers-reduced-motion (shows static checkmark).
 */

interface AnimatedCheckmarkProps {
  size?: number
  className?: string
  color?: string
}

export function AnimatedCheckmark({
  size = 48,
  className = '',
  color = 'var(--success)',
}: AnimatedCheckmarkProps) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 52 52"
      className={className}
      role="img"
      aria-label="Erfolgreich"
    >
      <circle
        cx="26"
        cy="26"
        r="24"
        fill="none"
        stroke={color}
        strokeWidth="2"
        className="anim-check-circle"
      />
      <path
        d="M15 27l7 7 15-15"
        fill="none"
        stroke={color}
        strokeWidth="3"
        strokeLinecap="round"
        strokeLinejoin="round"
        className="anim-check-tick"
      />
    </svg>
  )
}

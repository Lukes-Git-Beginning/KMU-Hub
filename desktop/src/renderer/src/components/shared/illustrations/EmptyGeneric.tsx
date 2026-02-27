/** Generic empty state illustration — minimalist box, theme-aware.
 *  Use as fallback when no specific illustration exists for a module. */
export function EmptyGeneric({ className = '' }: { className?: string }) {
  return (
    <svg width="140" height="120" viewBox="0 0 140 120" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
      {/* Open box */}
      <path
        d="M30 50l40-22 40 22v36l-40 22-40-22V50z"
        className="stroke-muted-foreground/25"
        strokeWidth="1.5"
        strokeLinejoin="round"
        fill="none"
      />
      {/* Box center line */}
      <line x1="70" y1="50" x2="70" y2="108" className="stroke-muted-foreground/15" strokeWidth="1" />
      {/* Box lid flaps */}
      <path d="M30 50l40 22 40-22" className="stroke-muted-foreground/20" strokeWidth="1.2" strokeLinejoin="round" fill="none" />
      {/* Lid open lines */}
      <path d="M30 50l-4-12 44-16 40 16-4 12" className="stroke-primary/25" strokeWidth="1.3" strokeLinejoin="round" fill="none" />
      {/* Sparkle */}
      <path d="M96 30l2-6 2 6 6 2-6 2-2 6-2-6-6-2 6-2z" className="fill-primary/20" />
    </svg>
  )
}

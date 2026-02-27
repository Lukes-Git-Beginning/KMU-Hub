/** Empty pipeline/funnel illustration — minimalist lineart, theme-aware */
export function EmptyDeals({ className = '' }: { className?: string }) {
  return (
    <svg width="140" height="120" viewBox="0 0 140 120" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
      {/* Funnel shape */}
      <path
        d="M30 28h80l-20 35v30l-20 10-20-10V63L30 28z"
        className="stroke-muted-foreground/30"
        strokeWidth="1.5"
        strokeLinejoin="round"
        fill="none"
      />
      {/* Funnel horizontal lines */}
      <line x1="38" y1="40" x2="102" y2="40" className="stroke-muted-foreground/15" strokeWidth="1" strokeDasharray="4 3" />
      <line x1="46" y1="52" x2="94" y2="52" className="stroke-muted-foreground/15" strokeWidth="1" strokeDasharray="4 3" />
      {/* Primary accent circle at bottom */}
      <circle cx="70" cy="80" r="6" className="fill-primary/12 stroke-primary/35" strokeWidth="1.3" />
      {/* Small arrow pointing down into funnel */}
      <path d="M70 20v-6M66 18l4 4 4-4" className="stroke-primary/30" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

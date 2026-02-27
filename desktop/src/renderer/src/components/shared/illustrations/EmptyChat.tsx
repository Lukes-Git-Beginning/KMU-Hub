/** Empty chat bubbles illustration — minimalist lineart, theme-aware */
export function EmptyChat({ className = '' }: { className?: string }) {
  return (
    <svg width="140" height="120" viewBox="0 0 140 120" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
      {/* Left bubble */}
      <path
        d="M22 34a6 6 0 016-6h40a6 6 0 016 6v24a6 6 0 01-6 6H40l-10 10V64h-2a6 6 0 01-6-6V34z"
        className="stroke-muted-foreground/25"
        strokeWidth="1.5"
        fill="none"
      />
      {/* Dots in left bubble */}
      <circle cx="42" cy="46" r="2" className="fill-muted-foreground/18" />
      <circle cx="52" cy="46" r="2" className="fill-muted-foreground/18" />
      <circle cx="62" cy="46" r="2" className="fill-muted-foreground/18" />
      {/* Right bubble */}
      <path
        d="M66 58a6 6 0 016-6h40a6 6 0 016 6v24a6 6 0 01-6 6h-8v10l-10-10h-22a6 6 0 01-6-6V58z"
        className="stroke-primary/30"
        strokeWidth="1.5"
        fill="none"
      />
      {/* Dashed lines in right bubble */}
      <line x1="78" y1="66" x2="106" y2="66" className="stroke-primary/20" strokeWidth="1.5" strokeDasharray="3 3" />
      <line x1="78" y1="74" x2="98" y2="74" className="stroke-primary/15" strokeWidth="1.5" strokeDasharray="3 3" />
    </svg>
  )
}

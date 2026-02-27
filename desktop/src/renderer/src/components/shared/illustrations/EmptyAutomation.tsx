/** Empty automation/gears illustration — minimalist lineart, theme-aware */
export function EmptyAutomation({ className = '' }: { className?: string }) {
  return (
    <svg width="140" height="120" viewBox="0 0 140 120" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
      {/* Large gear */}
      <circle cx="58" cy="52" r="18" className="stroke-muted-foreground/25" strokeWidth="1.5" fill="none" />
      <circle cx="58" cy="52" r="8" className="stroke-muted-foreground/20" strokeWidth="1.3" fill="none" />
      {/* Gear teeth (large) */}
      <line x1="58" y1="30" x2="58" y2="34" className="stroke-muted-foreground/25" strokeWidth="3" strokeLinecap="round" />
      <line x1="58" y1="70" x2="58" y2="74" className="stroke-muted-foreground/25" strokeWidth="3" strokeLinecap="round" />
      <line x1="36" y1="52" x2="40" y2="52" className="stroke-muted-foreground/25" strokeWidth="3" strokeLinecap="round" />
      <line x1="76" y1="52" x2="80" y2="52" className="stroke-muted-foreground/25" strokeWidth="3" strokeLinecap="round" />
      {/* Small gear */}
      <circle cx="92" cy="72" r="12" className="stroke-primary/30" strokeWidth="1.5" fill="none" />
      <circle cx="92" cy="72" r="5" className="fill-primary/10 stroke-primary/25" strokeWidth="1.2" />
      {/* Small gear teeth */}
      <line x1="92" y1="57" x2="92" y2="60" className="stroke-primary/30" strokeWidth="2.5" strokeLinecap="round" />
      <line x1="92" y1="84" x2="92" y2="87" className="stroke-primary/30" strokeWidth="2.5" strokeLinecap="round" />
      <line x1="77" y1="72" x2="80" y2="72" className="stroke-primary/30" strokeWidth="2.5" strokeLinecap="round" />
      <line x1="104" y1="72" x2="107" y2="72" className="stroke-primary/30" strokeWidth="2.5" strokeLinecap="round" />
      {/* Connection arrow */}
      <path d="M78 62l6 4" className="stroke-muted-foreground/18" strokeWidth="1.2" strokeLinecap="round" strokeDasharray="2 2" />
    </svg>
  )
}

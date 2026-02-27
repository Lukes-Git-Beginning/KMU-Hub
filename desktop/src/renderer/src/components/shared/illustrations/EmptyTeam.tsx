/** Empty org chart / people illustration — minimalist lineart, theme-aware */
export function EmptyTeam({ className = '' }: { className?: string }) {
  return (
    <svg width="140" height="120" viewBox="0 0 140 120" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
      {/* Center person */}
      <circle cx="70" cy="36" r="10" className="fill-primary/12 stroke-primary/35" strokeWidth="1.5" />
      <path d="M55 60c0-8.3 6.7-15 15-15s15 6.7 15 60" className="stroke-primary/35" strokeWidth="1.5" strokeLinecap="round" fill="none" />
      {/* Left ghost person */}
      <circle cx="36" cy="72" r="7" className="stroke-muted-foreground/18" strokeWidth="1.3" fill="none" strokeDasharray="3 2" />
      <path d="M26 92c0-5.5 4.5-10 10-10s10 4.5 10 10" className="stroke-muted-foreground/15" strokeWidth="1.3" strokeLinecap="round" strokeDasharray="3 2" fill="none" />
      {/* Right ghost person */}
      <circle cx="104" cy="72" r="7" className="stroke-muted-foreground/18" strokeWidth="1.3" fill="none" strokeDasharray="3 2" />
      <path d="M94 92c0-5.5 4.5-10 10-10s10 4.5 10 10" className="stroke-muted-foreground/15" strokeWidth="1.3" strokeLinecap="round" strokeDasharray="3 2" fill="none" />
      {/* Connection lines */}
      <line x1="55" y1="55" x2="43" y2="68" className="stroke-muted-foreground/12" strokeWidth="1" strokeDasharray="3 3" />
      <line x1="85" y1="55" x2="97" y2="68" className="stroke-muted-foreground/12" strokeWidth="1" strokeDasharray="3 3" />
    </svg>
  )
}

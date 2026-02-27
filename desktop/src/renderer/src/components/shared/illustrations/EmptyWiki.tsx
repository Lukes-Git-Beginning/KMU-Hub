/** Empty book/pages illustration — minimalist lineart, theme-aware */
export function EmptyWiki({ className = '' }: { className?: string }) {
  return (
    <svg width="140" height="120" viewBox="0 0 140 120" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
      {/* Open book — left page */}
      <path
        d="M70 30C58 28 40 26 28 30v62c12-3 30-2 42 2V30z"
        className="stroke-muted-foreground/25"
        strokeWidth="1.5"
        strokeLinejoin="round"
        fill="none"
      />
      {/* Open book — right page */}
      <path
        d="M70 30c12-2 30-4 42 0v62c-12-3-30-2-42 2V30z"
        className="stroke-muted-foreground/25"
        strokeWidth="1.5"
        strokeLinejoin="round"
        fill="none"
      />
      {/* Spine */}
      <line x1="70" y1="30" x2="70" y2="94" className="stroke-muted-foreground/20" strokeWidth="1" />
      {/* Left page lines */}
      <line x1="36" y1="44" x2="62" y2="42" className="stroke-muted-foreground/12" strokeWidth="1" strokeDasharray="3 3" />
      <line x1="36" y1="54" x2="62" y2="52" className="stroke-muted-foreground/10" strokeWidth="1" strokeDasharray="3 3" />
      <line x1="36" y1="64" x2="58" y2="62" className="stroke-muted-foreground/8" strokeWidth="1" strokeDasharray="3 3" />
      {/* Right page — primary accent */}
      <line x1="78" y1="42" x2="104" y2="44" className="stroke-primary/18" strokeWidth="1" strokeDasharray="3 3" />
      <line x1="78" y1="52" x2="104" y2="54" className="stroke-primary/15" strokeWidth="1" strokeDasharray="3 3" />
      <line x1="78" y1="62" x2="100" y2="64" className="stroke-primary/12" strokeWidth="1" strokeDasharray="3 3" />
    </svg>
  )
}

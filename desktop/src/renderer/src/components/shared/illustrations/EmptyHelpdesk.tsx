/** Empty ticket queue illustration — minimalist lineart, theme-aware */
export function EmptyHelpdesk({ className = '' }: { className?: string }) {
  return (
    <svg width="140" height="120" viewBox="0 0 140 120" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
      {/* Ticket shape */}
      <path
        d="M30 32a4 4 0 014-4h72a4 4 0 014 4v14a8 8 0 000 16v26a4 4 0 01-4 4H34a4 4 0 01-4-4V62a8 8 0 000-16V32z"
        className="stroke-muted-foreground/30"
        strokeWidth="1.5"
        fill="none"
      />
      {/* Perforation line */}
      <line x1="30" y1="54" x2="110" y2="54" className="stroke-muted-foreground/15" strokeWidth="1" strokeDasharray="4 4" />
      {/* Checkmark circle */}
      <circle cx="70" cy="38" r="8" className="fill-primary/10 stroke-primary/35" strokeWidth="1.3" />
      <path d="M65 38l3 3 7-7" className="stroke-primary/45" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" fill="none" />
      {/* Bottom detail lines */}
      <line x1="44" y1="66" x2="96" y2="66" className="stroke-muted-foreground/18" strokeWidth="1.5" strokeDasharray="3 3" />
      <line x1="50" y1="74" x2="90" y2="74" className="stroke-muted-foreground/12" strokeWidth="1.5" strokeDasharray="3 3" />
      <line x1="54" y1="82" x2="86" y2="82" className="stroke-muted-foreground/10" strokeWidth="1.5" strokeDasharray="3 3" />
    </svg>
  )
}

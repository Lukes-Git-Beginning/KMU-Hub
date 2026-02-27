/** Empty calendar illustration — minimalist lineart, theme-aware */
export function EmptyCalendar({ className = '' }: { className?: string }) {
  return (
    <svg width="140" height="120" viewBox="0 0 140 120" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
      {/* Calendar body */}
      <rect x="28" y="28" width="84" height="76" rx="6" className="stroke-muted-foreground/30" strokeWidth="1.5" fill="none" />
      {/* Header bar */}
      <rect x="28" y="28" width="84" height="20" rx="6" className="fill-primary/8 stroke-muted-foreground/30" strokeWidth="1.5" />
      {/* Calendar hooks */}
      <line x1="50" y1="22" x2="50" y2="34" className="stroke-muted-foreground/30" strokeWidth="2" strokeLinecap="round" />
      <line x1="90" y1="22" x2="90" y2="34" className="stroke-muted-foreground/30" strokeWidth="2" strokeLinecap="round" />
      {/* Grid lines */}
      <line x1="56" y1="48" x2="56" y2="104" className="stroke-muted-foreground/10" strokeWidth="0.8" />
      <line x1="84" y1="48" x2="84" y2="104" className="stroke-muted-foreground/10" strokeWidth="0.8" />
      <line x1="28" y1="67" x2="112" y2="67" className="stroke-muted-foreground/10" strokeWidth="0.8" />
      <line x1="28" y1="86" x2="112" y2="86" className="stroke-muted-foreground/10" strokeWidth="0.8" />
      {/* Empty day highlight */}
      <rect x="60" y="71" width="20" height="12" rx="2" className="fill-primary/10 stroke-primary/25" strokeWidth="1" />
    </svg>
  )
}

/** Empty chart/reports illustration — minimalist lineart, theme-aware */
export function EmptyReports({ className = '' }: { className?: string }) {
  return (
    <svg width="140" height="120" viewBox="0 0 140 120" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
      {/* Chart frame */}
      <rect x="28" y="22" width="84" height="76" rx="6" className="stroke-muted-foreground/25" strokeWidth="1.5" fill="none" />
      {/* Y axis */}
      <line x1="44" y1="34" x2="44" y2="86" className="stroke-muted-foreground/20" strokeWidth="1.2" />
      {/* X axis */}
      <line x1="44" y1="86" x2="100" y2="86" className="stroke-muted-foreground/20" strokeWidth="1.2" />
      {/* Empty bars (dashed outlines) */}
      <rect x="52" y="58" width="10" height="28" rx="2" className="stroke-muted-foreground/15" strokeWidth="1" strokeDasharray="3 2" fill="none" />
      <rect x="66" y="46" width="10" height="40" rx="2" className="stroke-primary/20" strokeWidth="1" strokeDasharray="3 2" fill="none" />
      <rect x="80" y="54" width="10" height="32" rx="2" className="stroke-muted-foreground/15" strokeWidth="1" strokeDasharray="3 2" fill="none" />
      {/* Trend line */}
      <path d="M52 68l14-16 14 8" className="stroke-primary/25" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" fill="none" strokeDasharray="4 3" />
    </svg>
  )
}

/** Empty clipboard/checklist illustration — minimalist lineart, theme-aware */
export function EmptyTasks({ className = '' }: { className?: string }) {
  return (
    <svg
      width="140"
      height="120"
      viewBox="0 0 140 120"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
    >
      {/* Clipboard body */}
      <rect
        x="35"
        y="20"
        width="70"
        height="88"
        rx="6"
        className="stroke-muted-foreground/30"
        strokeWidth="1.5"
        fill="none"
      />
      {/* Clipboard clip */}
      <rect
        x="55"
        y="14"
        width="30"
        height="14"
        rx="4"
        className="stroke-muted-foreground/30 fill-background"
        strokeWidth="1.5"
      />
      <rect x="63" y="18" width="14" height="4" rx="2" className="fill-primary/25" />
      {/* Empty checkbox rows */}
      <rect x="46" y="42" width="12" height="12" rx="2.5" className="stroke-primary/35" strokeWidth="1.3" fill="none" />
      <line x1="64" y1="48" x2="92" y2="48" className="stroke-muted-foreground/20" strokeWidth="1.5" strokeDasharray="3 3" />

      <rect x="46" y="62" width="12" height="12" rx="2.5" className="stroke-muted-foreground/20" strokeWidth="1.3" fill="none" />
      <line x1="64" y1="68" x2="88" y2="68" className="stroke-muted-foreground/15" strokeWidth="1.5" strokeDasharray="3 3" />

      <rect x="46" y="82" width="12" height="12" rx="2.5" className="stroke-muted-foreground/15" strokeWidth="1.3" fill="none" />
      <line x1="64" y1="88" x2="85" y2="88" className="stroke-muted-foreground/10" strokeWidth="1.5" strokeDasharray="3 3" />
    </svg>
  )
}

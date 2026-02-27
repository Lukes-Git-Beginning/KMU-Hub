/** Empty address book illustration — minimalist lineart, theme-aware */
export function EmptyContacts({ className = '' }: { className?: string }) {
  return (
    <svg
      width="140"
      height="120"
      viewBox="0 0 140 120"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
    >
      {/* Book body */}
      <rect
        x="30"
        y="18"
        width="80"
        height="90"
        rx="6"
        className="stroke-muted-foreground/30"
        strokeWidth="1.5"
        fill="none"
      />
      {/* Book spine */}
      <line x1="50" y1="18" x2="50" y2="108" className="stroke-muted-foreground/20" strokeWidth="1.5" />
      {/* Spine rings */}
      <circle cx="50" cy="38" r="3" className="stroke-muted-foreground/25" strokeWidth="1.2" fill="none" />
      <circle cx="50" cy="63" r="3" className="stroke-muted-foreground/25" strokeWidth="1.2" fill="none" />
      <circle cx="50" cy="88" r="3" className="stroke-muted-foreground/25" strokeWidth="1.2" fill="none" />
      {/* Person silhouette — primary colored */}
      <circle cx="80" cy="50" r="10" className="fill-primary/15 stroke-primary/40" strokeWidth="1.5" />
      <path
        d="M67 78c0-7.2 5.8-13 13-13s13 5.8 13 13"
        className="stroke-primary/40"
        strokeWidth="1.5"
        strokeLinecap="round"
        fill="none"
      />
      {/* Dashed lines — placeholder for contact details */}
      <line x1="64" y1="90" x2="96" y2="90" className="stroke-muted-foreground/20" strokeWidth="1.5" strokeDasharray="3 3" />
      <line x1="68" y1="97" x2="92" y2="97" className="stroke-muted-foreground/15" strokeWidth="1.5" strokeDasharray="3 3" />
    </svg>
  )
}

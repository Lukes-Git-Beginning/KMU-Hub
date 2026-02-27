/** Empty mailbox illustration — minimalist lineart, theme-aware */
export function EmptyMails({ className = '' }: { className?: string }) {
  return (
    <svg
      width="140"
      height="120"
      viewBox="0 0 140 120"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
    >
      {/* Envelope body */}
      <rect
        x="25"
        y="32"
        width="90"
        height="60"
        rx="6"
        className="stroke-muted-foreground/30"
        strokeWidth="1.5"
        fill="none"
      />
      {/* Envelope flap */}
      <path
        d="M25 38l45 28 45-28"
        className="stroke-primary/35"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        fill="none"
      />
      {/* Inner flap highlight */}
      <path
        d="M25 92l30-22M115 92l-30-22"
        className="stroke-muted-foreground/15"
        strokeWidth="1.2"
        strokeLinecap="round"
        fill="none"
      />
      {/* Small decorative dots — "no mail" indicator */}
      <circle cx="62" cy="66" r="1.5" className="fill-muted-foreground/20" />
      <circle cx="70" cy="66" r="1.5" className="fill-muted-foreground/25" />
      <circle cx="78" cy="66" r="1.5" className="fill-muted-foreground/20" />
    </svg>
  )
}

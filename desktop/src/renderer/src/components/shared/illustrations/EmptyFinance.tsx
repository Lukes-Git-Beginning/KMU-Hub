/** Empty invoice/receipt illustration — minimalist lineart, theme-aware */
export function EmptyFinance({ className = '' }: { className?: string }) {
  return (
    <svg width="140" height="120" viewBox="0 0 140 120" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
      {/* Receipt/invoice body */}
      <path
        d="M38 16h64a4 4 0 014 4v80l-8-5-8 5-8-5-8 5-8-5-8 5-8-5-8 5V20a4 4 0 014-4z"
        className="stroke-muted-foreground/30"
        strokeWidth="1.5"
        fill="none"
      />
      {/* Euro symbol */}
      <text x="62" y="48" className="fill-primary/30" fontSize="22" fontWeight="600" fontFamily="sans-serif">€</text>
      {/* Line items */}
      <line x1="48" y1="60" x2="92" y2="60" className="stroke-muted-foreground/20" strokeWidth="1.5" strokeDasharray="3 3" />
      <line x1="48" y1="70" x2="82" y2="70" className="stroke-muted-foreground/15" strokeWidth="1.5" strokeDasharray="3 3" />
      <line x1="48" y1="80" x2="88" y2="80" className="stroke-muted-foreground/12" strokeWidth="1.5" strokeDasharray="3 3" />
    </svg>
  )
}

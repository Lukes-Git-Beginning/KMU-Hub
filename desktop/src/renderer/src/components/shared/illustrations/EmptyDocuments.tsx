/** Empty folder illustration — minimalist lineart, theme-aware */
export function EmptyDocuments({ className = '' }: { className?: string }) {
  return (
    <svg width="140" height="120" viewBox="0 0 140 120" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
      {/* Folder back */}
      <path
        d="M24 36a4 4 0 014-4h24l8 10h52a4 4 0 014 4v54a4 4 0 01-4 4H28a4 4 0 01-4-4V36z"
        className="stroke-muted-foreground/30"
        strokeWidth="1.5"
        fill="none"
      />
      {/* Folder tab */}
      <path d="M24 42h92" className="stroke-muted-foreground/20" strokeWidth="1" />
      {/* File icon inside */}
      <rect x="56" y="54" width="28" height="34" rx="3" className="stroke-primary/30" strokeWidth="1.3" fill="none" />
      <path d="M72 54v10h12" className="stroke-primary/25" strokeWidth="1.2" strokeLinecap="round" strokeLinejoin="round" fill="none" />
      {/* Dashed lines on file */}
      <line x1="62" y1="72" x2="78" y2="72" className="stroke-muted-foreground/15" strokeWidth="1" strokeDasharray="2 2" />
      <line x1="62" y1="78" x2="74" y2="78" className="stroke-muted-foreground/12" strokeWidth="1" strokeDasharray="2 2" />
    </svg>
  )
}

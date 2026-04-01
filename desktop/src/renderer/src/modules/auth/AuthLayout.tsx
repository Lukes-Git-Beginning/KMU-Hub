import type { ReactNode } from 'react'

const FEATURES = [
  'Projekte, Aufgaben & Zeiterfassung',
  'Team, Schichten & HR',
  'Finanzen, Verträge & Dokumente',
  'Chat, E-Mail & Meetings',
]

export function AuthLayout({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-screen">
      {/* Brand panel — hidden on small viewports */}
      <div className="hidden lg:flex lg:w-[480px] xl:w-[540px] relative overflow-hidden flex-col justify-between bg-primary p-10 text-primary-foreground">
        {/* Decorative shapes */}
        <div className="pointer-events-none absolute inset-0">
          <div className="absolute -top-24 -right-24 h-64 w-64 rounded-full bg-white/5" />
          <div className="absolute bottom-16 -left-16 h-48 w-48 rounded-full bg-white/5" />
          <div className="absolute top-1/2 right-12 h-32 w-32 rotate-45 rounded-2xl bg-white/[0.03]" />
        </div>

        <div className="relative z-10">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-white/15">
              <svg width="22" height="22" viewBox="0 0 24 24" fill="none" className="text-white">
                <path d="M12 2L2 7l10 5 10-5-10-5z" fill="currentColor" opacity="0.3" />
                <path d="M2 17l10 5 10-5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                <path d="M2 12l10 5 10-5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
            </div>
            <span className="text-xl font-bold tracking-tight">Cosmi</span>
          </div>
        </div>

        <div className="relative z-10 space-y-6">
          <h1 className="text-3xl font-extrabold leading-tight tracking-tight xl:text-4xl">
            Alles für dein
            <br />
            Unternehmen.
            <br />
            <span className="text-white/70">Eine Plattform.</span>
          </h1>

          <ul className="space-y-3">
            {FEATURES.map((f) => (
              <li key={f} className="flex items-center gap-3 text-sm text-white/80">
                <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-white/15 text-[10px] font-bold">
                  &#10003;
                </span>
                {f}
              </li>
            ))}
          </ul>
        </div>

        <p className="relative z-10 text-xs text-white/40">
          EU-Datensouveraenitaet &middot; DSGVO-konform &middot; Self-Hosted oder Cloud
        </p>
      </div>

      {/* Form panel */}
      <div className="flex flex-1 items-center justify-center bg-background p-6">
        <div className="w-full max-w-sm">
          {children}
        </div>
      </div>
    </div>
  )
}

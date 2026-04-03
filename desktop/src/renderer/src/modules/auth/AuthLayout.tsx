import type { ReactNode } from 'react'
import { branding } from '@/config/branding'

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
          <img src={branding.cosmi.icon128} alt="Cosmi" className="h-20 w-auto" />
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

        <div className="relative z-10">
          <p className="text-xs text-white/40 mb-2">
            EU-Datensouveränität &middot; DSGVO-konform &middot; Self-Hosted oder Cloud
          </p>
          <p className="text-xs text-white/50">by Zentria</p>
        </div>
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

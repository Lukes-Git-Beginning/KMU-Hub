import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { branding } from '@/config/branding'

export function AuthLayout({ children }: { children: ReactNode }) {
  const { t } = useTranslation()

  const FEATURES = [
    t('auth.feature.projects'),
    t('auth.feature.team'),
    t('auth.feature.finance'),
    t('auth.feature.communication'),
  ]

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

        {/* Logo */}
        <div className="relative z-10">
          <img
            src={branding.cosmi.icon128}
            alt="Cosmi"
            className="h-auto w-auto max-h-[52px] object-contain"
          />
        </div>

        {/* Headline + feature list */}
        <div className="relative z-10 space-y-7">
          <h1 className="text-3xl font-bold leading-[1.15] tracking-tight xl:text-[2.25rem]">
            {t('auth.hero.line1')}
            <br />
            {t('auth.hero.line2')}
            <br />
            <span className="font-normal text-white/60">{t('auth.hero.line3')}</span>
          </h1>

          <ul className="space-y-3">
            {FEATURES.map((f) => (
              <li key={f} className="flex items-center gap-3 text-[13px] leading-snug text-white/75">
                <span className="flex h-[18px] w-[18px] shrink-0 items-center justify-center rounded-full bg-white/15">
                  <svg width="9" height="7" viewBox="0 0 9 7" fill="none" aria-hidden="true">
                    <path
                      d="M1 3.5l2.2 2.2L8 1"
                      stroke="currentColor"
                      strokeWidth="1.5"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    />
                  </svg>
                </span>
                {f}
              </li>
            ))}
          </ul>
        </div>

        {/* Footer */}
        <div className="relative z-10">
          <p className="mb-1.5 text-[11px] leading-relaxed text-white/35">
            {t('auth.footer.badges')}
          </p>
          <p className="text-[11px] text-white/45">by Zentria</p>
        </div>
      </div>

      {/* Form panel */}
      <div className="flex flex-1 items-center justify-center bg-background p-8">
        <div className="w-full max-w-[360px]">
          {children}
        </div>
      </div>
    </div>
  )
}

import { useTranslation } from 'react-i18next'
import { Clock } from 'lucide-react'
import ZeiterfassungTab from '@/modules/profil/tabs/ZeiterfassungTab'
import { StundenkontoBadge } from './components/StundenkontoBadge'

export default function ZeiterfassungPage() {
  const { t } = useTranslation()

  return (
    <div className="flex h-full flex-col">
      {/* Module shell header */}
      <header className="flex items-center justify-between gap-4 border-b border-border px-6 py-4">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary/10">
            <Clock className="h-5 w-5 text-primary" />
          </div>
          <div>
            <h1 className="text-lg font-semibold text-foreground">
              {t('zeiterfassung.shell.title')}
            </h1>
            <p className="text-xs text-muted-foreground">
              {t('zeiterfassung.shell.subtitle')}
            </p>
          </div>
        </div>
        <StundenkontoBadge />
      </header>

      {/* Functional core (clock-in/out, ArbZG, today/week/corrections) */}
      <div className="min-h-0 flex-1">
        <ZeiterfassungTab />
      </div>
    </div>
  )
}

/**
 * NoAccessView (R-3) — the light "Kein Zugriff" page a deep link lands on when
 * the role lacks a module's level-1 visibility (Google "You need access"
 * pattern; a blank page or silent redirect is the market's common weakness).
 * Mounted per module route via the ModuleGate wrapper in App.tsx.
 */
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { ShieldQuestion } from 'lucide-react'
import type { ModuleKey } from '@/config/capabilities'
import { moduleLabel } from '@/lib/rbac-format'

export function NoAccessView({ module }: { module: ModuleKey }) {
  const { t } = useTranslation()

  return (
    <div className="flex h-full items-center justify-center p-8">
      <div className="max-w-sm text-center">
        <span className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-secondary">
          <ShieldQuestion className="h-6 w-6 text-muted-foreground" aria-hidden="true" />
        </span>
        <h1 className="text-base font-medium text-foreground">
          {t('rbac.gate.noAccessTitle', { module: moduleLabel(t, module) })}
        </h1>
        <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
          {t('rbac.gate.noAccessBody')}
        </p>
        <Link
          to="/"
          className="mt-5 inline-flex items-center rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
        >
          {t('rbac.gate.noAccessCta')}
        </Link>
      </div>
    </div>
  )
}

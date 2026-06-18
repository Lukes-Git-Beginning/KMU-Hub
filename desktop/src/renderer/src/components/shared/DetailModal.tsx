import { X, ArrowLeft } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Dialog, DialogContent, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib'

/**
 * DetailModal — der einheitliche „Cosmi-Fenster"-Stil zum Öffnen von Detail-
 * Ansichten (Rechnung, Ausgabe, Kontakt, Dokument …). Zentriertes Modal mit
 * Gradient-Stripe, Header + Close und intern scrollendem Body — analog zur
 * Kontakte-/Dokumente-Detailansicht.
 *
 * API-kompatibel zu `DetailPanel` (open/onClose/title/subtitle/badge/footer/
 * children), damit Slide-over-Details ohne Umbau auf das Modal umstellbar sind.
 */
interface DetailModalProps {
  open: boolean
  onClose: () => void
  title?: string
  subtitle?: string
  badge?: React.ReactNode
  children: React.ReactNode
  footer?: React.ReactNode
  className?: string
  /** Tailwind max-w-* Klasse. Default max-w-2xl (Einzel-Detail); max-w-4xl für 360°-Ansichten. */
  maxWidth?: string
  /**
   * Optionaler „Zurück"-Handler. Wird gesetzt, sobald das Modal Teil einer
   * Navigations-Kette ist (z.B. Rechnung → Quell-Angebot). Rendert links im
   * Header einen Zurück-Chevron zusätzlich zum Schließen-Button.
   */
  onBack?: () => void
}

export function DetailModal({
  open,
  onClose,
  title,
  subtitle,
  badge,
  children,
  footer,
  className,
  maxWidth = 'max-w-2xl',
  onBack,
}: DetailModalProps) {
  const { t } = useTranslation()
  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) onClose() }}>
      <DialogContent
        showCloseButton={false}
        className={cn('flex max-h-[90vh] flex-col gap-0 overflow-hidden rounded-2xl p-0', maxWidth, className)}
      >
        {/* a11y: Radix benötigt einen Titel — sichtbar gerendert im Header unten, hier sr-only als Fallback */}
        <DialogTitle className="sr-only">{title ?? t('shared.detailPanel.panelLabel')}</DialogTitle>

        {/* Gradient-Stripe (Cosmi-Identität) */}
        <div className="h-0.5 w-full shrink-0 bg-gradient-to-r from-[var(--accent-1)] to-[var(--accent-2)]" />

        {(title || badge || onBack) && (
          <div className="flex items-center justify-between border-b px-5 py-3.5">
            <div className="flex min-w-0 items-center gap-2">
              {onBack && (
                <Button
                  variant="ghost"
                  size="icon"
                  className="group -ml-1.5 h-7 w-7 shrink-0"
                  onClick={onBack}
                  aria-label={t('shared.detailPanel.back')}
                >
                  <ArrowLeft
                    className="h-3.5 w-3.5 transition-transform group-hover:-translate-x-0.5"
                    aria-hidden="true"
                  />
                </Button>
              )}
              {title && (
                <div className="min-w-0">
                  <h3 className="truncate text-sm font-semibold text-foreground">{title}</h3>
                  {subtitle && <p className="truncate text-xs text-muted-foreground">{subtitle}</p>}
                </div>
              )}
              {badge}
            </div>
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7 shrink-0"
              onClick={onClose}
              aria-label={t('shared.detailPanel.close')}
            >
              <X className="h-3.5 w-3.5" aria-hidden="true" />
            </Button>
          </div>
        )}

        {/* Native scroll (reliable) — the global scrollbar-hide rule keeps it invisible. */}
        <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain">
          <div className="p-5">{children}</div>
        </div>

        {footer && <div className="border-t px-5 py-3">{footer}</div>}
      </DialogContent>
    </Dialog>
  )
}

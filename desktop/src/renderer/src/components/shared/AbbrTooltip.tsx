import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider } from '@/components/ui/tooltip'
import { cn } from '@/lib'

/**
 * AbbrTooltip — subtile Hover-Erklärung für Abkürzungen (SLA, CSAT, …).
 *
 * Rendert die Abkürzung mit dezenter gepunkteter Unterstreichung; on-hover
 * erscheint die ausgeschriebene Form + ein kurzer Erklärsatz. Wiederverwendbar
 * über ganz Cosmi — neue Begriffe einfach im GLOSSARY ergänzen (Text ist
 * i18n-fähig via `glossary.<TERM>.full` / `.description`, mit DE-Fallback).
 */
const GLOSSARY: Record<string, { full: string; description: string }> = {
  SLA: {
    full: 'Service Level Agreement',
    description: 'Zugesicherte Reaktions- und Lösungszeit für ein Ticket.',
  },
  CSAT: {
    full: 'Customer Satisfaction Score',
    description: 'Kundenzufriedenheit, gemessen über eine kurze Bewertung nach Ticket-Abschluss.',
  },
  KB: {
    full: 'Knowledge Base',
    description: 'Wissensdatenbank mit Artikeln zur Selbsthilfe und internen Lösungen.',
  },
  FRT: {
    full: 'First Response Time',
    description: 'Zeit bis zur ersten Antwort auf ein neues Ticket.',
  },
}

interface AbbrTooltipProps {
  /** Abkürzungs-Schlüssel, z.B. "SLA". Schlägt im Glossar nach. */
  term: string
  /** Optionales Anzeige-Label statt des Terms (sonst wird `term` angezeigt). */
  children?: ReactNode
  className?: string
}

export function AbbrTooltip({ term, children, className }: AbbrTooltipProps) {
  const { t } = useTranslation()
  const entry = GLOSSARY[term]

  // Kein Glossar-Eintrag → reiner Text ohne Dekoration (Fail-safe).
  if (!entry) return <>{children ?? term}</>

  const full = t(`glossary.${term}.full`, { defaultValue: entry.full })
  const description = t(`glossary.${term}.description`, { defaultValue: entry.description })

  return (
    <TooltipProvider delayDuration={250}>
      <Tooltip>
        <TooltipTrigger asChild>
          <span
            className={cn(
              'cursor-help underline decoration-dotted decoration-muted-foreground/40 underline-offset-2',
              className,
            )}
          >
            {children ?? term}
          </span>
        </TooltipTrigger>
        <TooltipContent side="top" className="max-w-xs">
          <p className="font-medium text-foreground">{full}</p>
          <p className="mt-0.5 text-xs text-muted-foreground">{description}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}

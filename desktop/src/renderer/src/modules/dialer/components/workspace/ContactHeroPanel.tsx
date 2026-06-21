import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { ExternalLink } from 'lucide-react'
import { cn } from '@/lib/cn'
import type { CampaignContact } from '@/api/dialer-client'

interface ContactHeroPanelProps {
  contact: CampaignContact
  showConnecting?: boolean
  className?: string
  children?: React.ReactNode
}

export default function ContactHeroPanel({
  contact,
  showConnecting,
  className,
  children,
}: ContactHeroPanelProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const initial = contact.contact_name.charAt(0).toUpperCase()

  return (
    <div className={cn('relative flex flex-col items-center text-center', className)}>
      {/* Editorial background initial */}
      <span
        className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 select-none pointer-events-none font-black text-foreground/[0.03]"
        style={{ fontSize: 'clamp(10rem, 25vw, 18rem)', lineHeight: 1 }}
      >
        {initial}
      </span>

      {/* Content */}
      <div className="relative z-10 space-y-3 animate-fade-up">
        {/* Name — the hero */}
        <h2 className="text-5xl font-extrabold tracking-tight text-foreground leading-tight">
          {contact.contact_name}
        </h2>

        {/* Company */}
        {contact.contact_company && (
          <p className="text-xl font-light text-muted-foreground">
            {contact.contact_company}
          </p>
        )}

        {/* Phone */}
        <p className="font-mono text-2xl text-muted-foreground tracking-wide">
          {contact.contact_phone}
        </p>

        {/* CRM deep-link (D-1) */}
        {contact.contact_id && (
          <button
            type="button"
            onClick={() => navigate(`/kontakte?contact=${contact.contact_id}`)}
            className="inline-flex items-center gap-1.5 rounded-full border border-border bg-card px-3 py-1 text-xs font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
          >
            <ExternalLink className="h-3.5 w-3.5" />
            {t('dialer.workspace.openInCrm')}
          </button>
        )}

        {/* Call attempt indicator */}
        {contact.call_count > 0 && (
          <span className="inline-flex items-center gap-1.5 rounded-full bg-warning/10 border border-warning/20 px-3 py-1 text-xs font-medium text-warning">
            {t('dialer.workspace.dialing.callAttempt', { count: contact.call_count + 1 })}
          </span>
        )}

        {/* Connecting indicator */}
        {showConnecting && (
          <div className="flex items-center justify-center gap-2 pt-2">
            <span className="h-2 w-2 rounded-full bg-success animate-glow-pulse" />
            <span className="text-sm font-medium text-muted-foreground">
              {t('dialer.workspace.dialing.connecting')}
            </span>
          </div>
        )}

        {/* Slot for timer or other content */}
        {children}
      </div>
    </div>
  )
}

import { useTranslation } from 'react-i18next'
import { Mail, Phone, Building2, ExternalLink } from 'lucide-react'
import type { Conversation } from '@/types/communication'

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface ContactCardProps {
  conversation: Conversation
}

export function ContactCard({ conversation: conv }: ContactCardProps) {
  const { t } = useTranslation()
  const initials = conv.contactName
    .split(' ')
    .map((n) => n[0])
    .join('')
    .toUpperCase()
    .slice(0, 2)

  return (
    <div className="p-3 border-b border-border">
      {/* Avatar + name */}
      <div className="flex items-center gap-2.5">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-primary-light text-sm font-medium text-primary">
          {initials}
        </div>
        <div className="min-w-0 flex-1">
          <p className="text-sm font-medium text-foreground truncate">{conv.contactName}</p>
          {conv.contactCompany && (
            <div className="flex items-center gap-1 text-xs text-muted-foreground">
              <Building2 className="h-3 w-3 shrink-0" />
              <span className="truncate">{conv.contactCompany}</span>
            </div>
          )}
        </div>
        {/* Link to CRM contact */}
        {conv.contactId && (
          <button
            className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors shrink-0"
            title={t('kommunikation.contact.openInCrm')}
          >
            <ExternalLink className="h-3.5 w-3.5" />
          </button>
        )}
      </div>

      {/* Contact details */}
      <div className="mt-2.5 space-y-1">
        {conv.contactEmail && (
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <Mail className="h-3 w-3 shrink-0" />
            <span className="truncate">{conv.contactEmail}</span>
          </div>
        )}
        {/* Phone placeholder — would come from CRM contact store */}
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <Phone className="h-3 w-3 shrink-0" />
          <span className="text-muted-foreground/50">{t('kommunikation.contact.notProvided')}</span>
        </div>
      </div>
    </div>
  )
}

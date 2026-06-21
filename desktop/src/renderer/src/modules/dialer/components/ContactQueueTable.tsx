import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { formatDate } from '@/lib/format'
import { SkipForward, RotateCcw, Phone, Clock, Building2 } from 'lucide-react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Skeleton } from '@/components/ui/skeleton'
import { ItemActions } from '@/components/shared'
import type { CampaignContact } from '@/api/dialer-client'
import ContactDetailModal from './ContactDetailModal'

export const contactStatusConfig: Record<
  number,
  { key: string; color: string }
> = {
  1: { key: 'dialer.campaign.contacts.status.pending', color: 'var(--text-muted)' },
  2: { key: 'dialer.campaign.contacts.status.inProgress', color: 'var(--info)' },
  3: { key: 'dialer.campaign.contacts.status.completed', color: 'var(--success)' },
  4: { key: 'dialer.campaign.contacts.status.skipped', color: 'var(--text-disabled)' },
  5: { key: 'dialer.campaign.contacts.status.callback', color: 'var(--warning)' },
}

interface ContactQueueTableProps {
  contacts: CampaignContact[]
  isLoading?: boolean
  campaignStatus: number
  onSkip?: (contactId: string) => void
  onRequeue?: (contactId: string) => void
}

export default function ContactQueueTable({
  contacts,
  isLoading,
  campaignStatus,
  onSkip,
  onRequeue,
}: ContactQueueTableProps) {
  const { t } = useTranslation()
  const [detailContact, setDetailContact] = useState<CampaignContact | null>(null)

  if (isLoading) {
    return (
      <div className="space-y-2">
        {Array.from({ length: 6 }).map((_, i) => (
          <Skeleton key={i} className="h-12 w-full" />
        ))}
      </div>
    )
  }

  if (contacts.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center">
        <Phone className="h-10 w-10 text-muted-foreground/40 mb-3" />
        <p className="text-sm font-medium text-muted-foreground">
          {t('dialer.campaigns.empty.title')}
        </p>
      </div>
    )
  }

  return (
    <>
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="w-12">#</TableHead>
          <TableHead>{t('dialer.settings.outcomes.label')}</TableHead>
          <TableHead className="hidden md:table-cell">
            <Building2 className="h-3.5 w-3.5" />
          </TableHead>
          <TableHead>
            <Phone className="h-3.5 w-3.5" />
          </TableHead>
          <TableHead>Status</TableHead>
          <TableHead className="text-center">
            <Phone className="h-3.5 w-3.5 mx-auto" />
          </TableHead>
          {campaignStatus === 2 && <TableHead className="w-10" />}
        </TableRow>
      </TableHeader>
      <TableBody>
        {contacts.map((contact) => {
          const statusCfg = contactStatusConfig[contact.status] ?? contactStatusConfig[1]
          const actions = []

          if (contact.status === 1 && campaignStatus === 2) {
            actions.push({
              label: t('dialer.workspace.onCall.skip'),
              icon: SkipForward,
              onClick: () => onSkip?.(contact.id),
            })
          }
          if ((contact.status === 4 || contact.status === 3) && campaignStatus === 2) {
            actions.push({
              label: 'Wiedereinreihen',
              icon: RotateCcw,
              onClick: () => onRequeue?.(contact.id),
            })
          }

          return (
            <TableRow
              key={contact.id}
              role="button"
              tabIndex={0}
              onClick={() => setDetailContact(contact)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault()
                  setDetailContact(contact)
                }
              }}
              className="cursor-pointer transition-colors hover:bg-accent/50 focus:outline-none focus-visible:bg-accent/50"
            >
              <TableCell className="text-xs text-muted-foreground tabular-nums">
                {contact.position}
              </TableCell>
              <TableCell className="font-medium">{contact.contact_name}</TableCell>
              <TableCell className="hidden md:table-cell text-sm text-muted-foreground">
                {contact.contact_company ?? '—'}
              </TableCell>
              <TableCell className="font-mono text-xs text-muted-foreground">
                {contact.contact_phone}
              </TableCell>
              <TableCell>
                <span
                  className="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium border"
                  style={{
                    backgroundColor: `color-mix(in srgb, ${statusCfg.color} 12%, transparent)`,
                    color: statusCfg.color,
                    borderColor: `color-mix(in srgb, ${statusCfg.color} 30%, transparent)`,
                  }}
                >
                  {t(statusCfg.key)}
                </span>
                {contact.status === 5 && contact.callback_at && (
                  <span className="ml-1.5 inline-flex items-center gap-1 text-xs text-warning">
                    <Clock className="h-3 w-3" />
                    {formatDate(contact.callback_at)}
                  </span>
                )}
              </TableCell>
              <TableCell className="text-center tabular-nums text-xs">
                {contact.call_count}
              </TableCell>
              {campaignStatus === 2 && (
                <TableCell onClick={(e) => e.stopPropagation()}>
                  {actions.length > 0 && <ItemActions items={actions} />}
                </TableCell>
              )}
            </TableRow>
          )
        })}
      </TableBody>
    </Table>

      <ContactDetailModal
        contact={detailContact}
        open={!!detailContact}
        onClose={() => setDetailContact(null)}
      />
    </>
  )
}

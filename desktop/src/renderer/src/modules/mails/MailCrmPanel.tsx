import { useState, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Users, Plus, X, Search, Briefcase, ClipboardList, ExternalLink } from 'lucide-react'
import { toast } from 'sonner'
import {
  useEmailContactLinks,
  useLinkEmailToContact,
  useUnlinkEmailFromContact,
} from '@/api/hooks/useEmail'
import { useContacts } from '@/api/hooks/useContacts'
import { useCreateDeal } from '@/api/hooks/useDeals'
import { usePipelineStages } from '@/api/hooks/usePipelineStages'
import { useCreateActivity } from '@/api/hooks/useActivities'
import { useNavigationStore } from '@/stores/navigation'
import type { EmailMessageInfo } from '@/api/email-types'

interface MailCrmPanelProps {
  message: EmailMessageInfo
}

interface ContactLite {
  id?: string
  firstName?: string
  lastName?: string
  email?: string
}

function contactName(c: ContactLite): string {
  return [c.firstName, c.lastName].filter(Boolean).join(' ') || c.email || c.id || ''
}

/**
 * CRM panel shown in the mail reading pane: link the message to CRM contacts,
 * create a deal from the mail, or log it as an activity. Deal and activity
 * creation write to the shared CRM stores, so they show up in /crm.
 */
export function MailCrmPanel({ message }: MailCrmPanelProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const setIntent = useNavigationStore((s) => s.setIntent)

  const { data: linksData } = useEmailContactLinks(message.id)
  const links = linksData?.links ?? []
  const { data: contactsData } = useContacts()
  const contacts = useMemo<ContactLite[]>(() => contactsData?.contacts ?? [], [contactsData?.contacts])
  const contactById = useMemo(() => Object.fromEntries(contacts.map((c) => [c.id, c])), [contacts])

  const linkContact = useLinkEmailToContact()
  const unlinkContact = useUnlinkEmailFromContact()
  const createDeal = useCreateDeal()
  const createActivity = useCreateActivity()
  const { data: stagesData } = usePipelineStages()
  const firstStageId =
    (stagesData as { stages?: { id: string }[] } | { id: string }[] | undefined) &&
    (Array.isArray(stagesData) ? stagesData[0]?.id : (stagesData as { stages?: { id: string }[] })?.stages?.[0]?.id)

  const [showPicker, setShowPicker] = useState(false)
  const [pickerSearch, setPickerSearch] = useState('')

  const linkedIds = links.map((l) => l.contact_id)
  // The contact that best represents this mail (linked first, else sender match).
  const senderMatch = contacts.find((c) => c.email && c.email.toLowerCase() === message.from.email.toLowerCase())
  const primaryContactId = linkedIds[0] ?? senderMatch?.id

  const pickerResults = contacts
    .filter((c): c is ContactLite & { id: string } => !!c.id && !linkedIds.includes(c.id))
    .filter((c) => !pickerSearch || contactName(c).toLowerCase().includes(pickerSearch.toLowerCase()) || (c.email ?? '').toLowerCase().includes(pickerSearch.toLowerCase()))
    .slice(0, 6)

  const openContact = (id: string) => {
    setIntent({ type: 'open-contact', data: { contactId: id } })
    navigate('/kontakte')
  }

  const handleCreateDeal = () => {
    createDeal.mutate(
      {
        name: message.subject || t('mails.crm.dealDefaultName', { defaultValue: 'Deal aus E-Mail' }),
        value: 0,
        currency: 'EUR',
        stage_id: firstStageId ?? '',
        contact_id: primaryContactId,
        notes: t('mails.crm.dealNote', { defaultValue: 'Aus E-Mail erstellt' }) + `: ${message.subject}`,
      },
      {
        onSuccess: () => {
          toast.success(t('mails.crm.dealCreated', { defaultValue: 'Deal aus E-Mail erstellt' }), {
            action: { label: t('mails.crm.openDeals', { defaultValue: 'Öffnen' }), onClick: () => navigate('/crm/deals') },
          })
        },
        onError: () => toast.error(t('common.error', { defaultValue: 'Fehler' })),
      },
    )
  }

  const handleLogActivity = () => {
    createActivity.mutate(
      {
        activity_type: 'email',
        subject: message.subject || t('mails.crm.activityDefault', { defaultValue: 'E-Mail' }),
        description: message.preview,
        contact_id: primaryContactId,
      },
      {
        onSuccess: () => toast.success(t('mails.crm.activityLogged', { defaultValue: 'Als Aktivität protokolliert' })),
        onError: () => toast.error(t('common.error', { defaultValue: 'Fehler' })),
      },
    )
  }

  return (
    <div className="mb-4 rounded-xl border border-border bg-card/60 p-3">
      <div className="flex items-center justify-between mb-2">
        <span className="flex items-center gap-1.5 text-xs font-medium text-foreground">
          <Users className="h-3.5 w-3.5 text-primary" />
          {t('mails.crm.title', { defaultValue: 'CRM' })}
        </span>
        <div className="flex items-center gap-1">
          <button
            onClick={handleCreateDeal}
            disabled={createDeal.isPending}
            className="flex items-center gap-1 rounded-md border border-border px-2 py-1 text-[11px] text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors disabled:opacity-50"
          >
            <Briefcase className="h-3 w-3" />
            {t('mails.crm.createDeal', { defaultValue: 'Deal aus Mail' })}
          </button>
          <button
            onClick={handleLogActivity}
            disabled={createActivity.isPending}
            className="flex items-center gap-1 rounded-md border border-border px-2 py-1 text-[11px] text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors disabled:opacity-50"
          >
            <ClipboardList className="h-3 w-3" />
            {t('mails.crm.logActivity', { defaultValue: 'Aktivität' })}
          </button>
        </div>
      </div>

      {/* Linked contacts */}
      <div className="flex flex-wrap items-center gap-1.5">
        {links.map((link) => {
          const c = contactById[link.contact_id]
          return (
            <span key={link.id} className="inline-flex items-center gap-1 rounded-full bg-primary/10 pl-2 pr-1 py-0.5 text-xs text-primary">
              <button onClick={() => openContact(link.contact_id)} className="flex items-center gap-1 hover:underline">
                {c ? contactName(c) : link.contact_id}
                <ExternalLink className="h-2.5 w-2.5" />
              </button>
              <button
                onClick={() => unlinkContact.mutate({ messageId: message.id, contactId: link.contact_id })}
                className="rounded-full p-0.5 hover:bg-primary/20"
                aria-label={t('mails.crm.unlink', { defaultValue: 'Verknüpfung entfernen' })}
              >
                <X className="h-2.5 w-2.5" />
              </button>
            </span>
          )
        })}
        <button
          onClick={() => setShowPicker((v) => !v)}
          className="inline-flex items-center gap-1 rounded-full border border-dashed border-border px-2 py-0.5 text-xs text-muted-foreground hover:bg-secondary transition-colors"
        >
          <Plus className="h-3 w-3" />
          {t('mails.crm.link', { defaultValue: 'Kontakt verknüpfen' })}
        </button>
      </div>

      {/* Contact picker */}
      {showPicker && (
        <div className="mt-2 rounded-lg border border-border bg-background p-2">
          <div className="relative mb-1.5">
            <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
            <input
              autoFocus
              value={pickerSearch}
              onChange={(e) => setPickerSearch(e.target.value)}
              placeholder={t('mails.crm.searchContact', { defaultValue: 'Kontakt suchen…' })}
              className="w-full rounded-md border border-border bg-background pl-7 pr-2 py-1 text-xs focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>
          <div className="space-y-0.5 max-h-40 overflow-y-auto">
            {pickerResults.map((c) => (
              <button
                key={c.id}
                onClick={() => {
                  linkContact.mutate({ messageId: message.id, contactId: c.id })
                  setShowPicker(false)
                  setPickerSearch('')
                }}
                className="flex w-full items-center justify-between gap-2 rounded-md px-2 py-1 text-xs text-foreground hover:bg-secondary transition-colors"
              >
                <span className="truncate">{contactName(c)}</span>
                {c.email && <span className="truncate text-[10px] text-muted-foreground">{c.email}</span>}
              </button>
            ))}
            {pickerResults.length === 0 && (
              <p className="px-2 py-1.5 text-xs text-muted-foreground">{t('mails.crm.noContacts', { defaultValue: 'Keine Treffer' })}</p>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

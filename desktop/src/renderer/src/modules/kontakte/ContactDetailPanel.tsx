import { useState } from 'react'
import {
  Phone,
  Mail,
  MessageSquare,
  MapPin,
  Building2,
  Calendar,
  Globe,
  Linkedin,
  Star,
  Briefcase,
  Pencil,
  Trash2,
  Plus,
  X,
  User,
  TrendingUp,
  CheckSquare,
  Shield,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { ContactTimeline } from '@/modules/crm/contacts/ContactTimeline'
import { useTags, useAddContactTags, useRemoveContactTags } from '@/api/hooks/useContactTags'
import { useDeals } from '@/api/hooks/useDeals'
import { useContactEmails } from '@/api/hooks/useEmail'
import { useEntityTasks } from '@/api/hooks/useTasks'
import { ConsentPanel } from '@/modules/kontakte/ConsentPanel'
import type { Contact } from '@/stores/contacts'
import { formatDate } from '@/lib/format'

interface ContactDetailPanelProps {
  contact: Contact
  onClose: () => void
  onEdit: (contact: Contact) => void
  onDelete: (contactId: string) => void
  onEmail: (contact: Contact) => void
  onCall: (contact: Contact) => void
  onMessage: (contact: Contact) => void
  onToggleFavorite: (contactId: string) => void
}

const statusConfig = {
  active: { labelKey: 'kontakte.status.active', variant: 'default' as const, color: 'bg-green-500' },
  prospect: { labelKey: 'kontakte.status.prospect', variant: 'secondary' as const, color: 'bg-amber-500' },
  inactive: { labelKey: 'kontakte.status.inactive', variant: 'outline' as const, color: 'bg-gray-400' },
}

const categoryLabelKeys: Record<string, string> = {
  employee: 'kontakte.category.employee',
  customer: 'kontakte.category.customer',
  partner: 'kontakte.category.partner',
}

// ---------------------------------------------------------------------------
// TagEditor — inline chip list + popover to add/remove tags
// ---------------------------------------------------------------------------

interface TagEditorProps {
  contactId: string
  /** Current tag names on the contact (from the store) */
  tagNames: string[]
}

function TagEditor({ contactId, tagNames }: TagEditorProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const { data: availableTags = [] } = useTags()
  const addTags = useAddContactTags()
  const removeTags = useRemoveContactTags()

  // Find which available tags are already active (match by name, case-insensitive)
  const activeTagNames = new Set(tagNames.map((n) => n.toLowerCase()))

  const handleRemove = (tagName: string) => {
    const matched = availableTags.find((t) => t.name.toLowerCase() === tagName.toLowerCase())
    const tagIds = matched ? [matched.id] : []
    // Optimistically remove; always invalidate even if id not found so store re-fetches
    removeTags.mutate(
      { contactId, tagIds: tagIds.length ? tagIds : ['__unknown__'] },
      {
        onSuccess: () => toast.success(t('kontakte.tag.removed', { name: tagName })),
        onError: () => toast.error(t('kontakte.tag.removeError')),
      },
    )
  }

  const handleAdd = (tag: { id: string; name: string }) => {
    addTags.mutate(
      { contactId, tagIds: [tag.id] },
      {
        onSuccess: () => {
          toast.success(t('kontakte.tag.added', { name: tag.name }))
          setOpen(false)
        },
        onError: () => toast.error(t('kontakte.tag.addError')),
      },
    )
  }

  // Tags not yet on the contact
  const addableTagss = availableTags.filter((t) => !activeTagNames.has(t.name.toLowerCase()))

  return (
    <div className="flex flex-wrap items-center gap-1.5">
      {tagNames.map((tagName) => {
        const meta = availableTags.find((t) => t.name.toLowerCase() === tagName.toLowerCase())
        return (
          <span
            key={tagName}
            className="inline-flex items-center gap-1 rounded-full border border-border bg-secondary px-2 py-0.5 text-xs text-foreground"
          >
            {meta?.color && (
              <span
                className="h-1.5 w-1.5 rounded-full shrink-0"
                style={{ backgroundColor: meta.color }}
              />
            )}
            {tagName}
            <button
              onClick={() => handleRemove(tagName)}
              className="ml-0.5 rounded-full p-0.5 text-muted-foreground hover:text-destructive transition-colors"
              aria-label={t('kontakte.tag.remove', { name: tagName })}
            >
              <X className="h-2.5 w-2.5" />
            </button>
          </span>
        )
      })}

      {/* Add-tag popover */}
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <button
            className="inline-flex items-center gap-1 rounded-full border border-dashed border-border px-2 py-0.5 text-xs text-muted-foreground hover:border-primary hover:text-primary transition-colors"
            aria-label={t('kontakte.tag.add')}
          >
            <Plus className="h-2.5 w-2.5" />
            {t('kontakte.tag.add')}
          </button>
        </PopoverTrigger>
        <PopoverContent align="start" className="w-52 p-2">
          {addableTagss.length === 0 ? (
            <p className="py-2 text-center text-xs text-muted-foreground">{t('kontakte.tag.allAssigned')}</p>
          ) : (
            <div className="space-y-0.5">
              {addableTagss.map((tag) => (
                <button
                  key={tag.id}
                  onClick={() => handleAdd(tag)}
                  className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm text-foreground hover:bg-accent transition-colors"
                >
                  <span
                    className="h-2 w-2 rounded-full shrink-0"
                    style={{ backgroundColor: tag.color }}
                  />
                  {tag.name}
                </button>
              ))}
            </div>
          )}
        </PopoverContent>
      </Popover>
    </div>
  )
}

// ---------------------------------------------------------------------------
// ContactDetailPanel
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Deal stage badge colors (maps stage names to semantic tokens)
// ---------------------------------------------------------------------------

const stageVariantMap: Record<string, 'default' | 'secondary' | 'outline' | 'destructive'> = {
  Lead: 'outline',
  Qualifiziert: 'secondary',
  Angebot: 'secondary',
  Verhandlung: 'default',
  Gewonnen: 'default',
  Verloren: 'destructive',
}

// ---------------------------------------------------------------------------
// Task status badge
// ---------------------------------------------------------------------------

const taskStatusVariantMap: Record<string, 'default' | 'secondary' | 'outline' | 'destructive'> = {
  done: 'default',
  completed: 'default',
  in_progress: 'secondary',
  todo: 'outline',
  cancelled: 'destructive',
}

export function ContactDetailPanel({
  contact,
  onClose,
  onEdit,
  onDelete,
  onEmail,
  onCall,
  onMessage,
  onToggleFavorite,
}: ContactDetailPanelProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const status = statusConfig[contact.status]

  // --- 360° data hooks ---
  const { data: dealsData } = useDeals({ contact_id: contact.id, page_size: 5 })
  const { data: emailsData } = useContactEmails(contact.id, 1, 5)
  const { data: tasksData } = useEntityTasks('contact', contact.id)

  const contactDeals = (dealsData as { deals?: Record<string, unknown>[] } | undefined)?.deals ?? []
  const contactEmails = (emailsData as { messages?: Record<string, unknown>[] } | undefined)?.messages ?? []
  const contactTasks = (tasksData as { tasks?: Record<string, unknown>[] } | undefined)?.tasks ?? []

  return (
    <div className="flex h-full flex-col overflow-hidden min-h-0">
      {/* ------------------------------------------------------------------ */}
      {/* Gradient header                                                      */}
      {/* ------------------------------------------------------------------ */}
      <div className="bg-gradient-to-br from-primary to-primary-dark px-6 py-5 shrink-0">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-1.5 ml-auto">
            <button
              onClick={() => onToggleFavorite(contact.id)}
              className="rounded-md p-1.5 text-primary-foreground/70 hover:text-primary-foreground transition-colors"
              title={contact.isFavorite ? t('kontakte.action.unfavorite') : t('kontakte.action.addToFavorites')}
            >
              <Star className={`h-4 w-4 ${contact.isFavorite ? 'fill-yellow-400 text-yellow-400' : ''}`} />
            </button>
            <button
              onClick={() => onEdit(contact)}
              className="rounded-md p-1.5 text-primary-foreground/70 hover:text-primary-foreground transition-colors"
              title={t('common.edit')}
            >
              <Pencil className="h-4 w-4" />
            </button>
            <button
              onClick={() => onDelete(contact.id)}
              className="rounded-md p-1.5 text-primary-foreground/70 hover:text-red-300 transition-colors"
              title={t('common.delete')}
            >
              <Trash2 className="h-4 w-4" />
            </button>
            <button
              onClick={onClose}
              className="ml-1 rounded-md p-1.5 text-primary-foreground/70 hover:bg-primary-foreground/20 hover:text-primary-foreground transition-colors"
              title={t('common.close')}
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        </div>

        <div className="flex items-center gap-4">
          <div className="flex h-16 w-16 shrink-0 items-center justify-center rounded-full border-2 border-primary-foreground/30 bg-primary-foreground/20 text-xl font-medium text-primary-foreground">
            {contact.initials || <User className="h-7 w-7 opacity-70" />}
          </div>
          <div className="min-w-0">
            <h2 className="text-primary-foreground font-semibold leading-tight">
              {contact.salutation && (
                <span className="text-primary-foreground/70 text-sm mr-1 font-normal">{contact.salutation}</span>
              )}
              {contact.title && (
                <span className="text-primary-foreground/80 text-sm mr-1 font-normal">{contact.title}</span>
              )}
              {contact.firstName} {contact.lastName}
            </h2>
            <p className="text-sm text-primary-foreground/70 truncate">
              {contact.jobTitle}
              {contact.jobTitle && contact.company ? ' · ' : ''}
              {contact.company}
            </p>
            {contact.department && (
              <p className="text-xs text-primary-foreground/50 truncate">{contact.department}</p>
            )}
            <div className="flex flex-wrap items-center gap-2 mt-2">
              <Badge variant={status.variant} className="text-xs">
                <span className={`mr-1.5 inline-block h-1.5 w-1.5 rounded-full ${status.color}`} />
                {t(status.labelKey)}
              </Badge>
              <span className="rounded-full bg-primary-foreground/20 px-2 py-0.5 text-xs text-primary-foreground">
                {t(categoryLabelKeys[contact.category] ?? 'kontakte.category.customer')}
              </span>
            </div>
          </div>
        </div>

        {/* Quick-action buttons */}
        <div className="flex flex-wrap gap-2 mt-4">
          <button
            onClick={() => onCall(contact)}
            className="flex items-center gap-1.5 rounded-lg bg-primary-foreground/20 px-3 py-1.5 text-xs text-primary-foreground hover:bg-primary-foreground/30 transition-colors"
          >
            <Phone className="h-3.5 w-3.5" />
            {t('kontakte.action.call')}
          </button>
          <button
            onClick={() => onEmail(contact)}
            className="flex items-center gap-1.5 rounded-lg bg-primary-foreground/20 px-3 py-1.5 text-xs text-primary-foreground hover:bg-primary-foreground/30 transition-colors"
          >
            <Mail className="h-3.5 w-3.5" />
            {t('kontakte.action.emailShort')}
          </button>
          <button
            onClick={() => onMessage(contact)}
            className="flex items-center gap-1.5 rounded-lg bg-primary-foreground/20 px-3 py-1.5 text-xs text-primary-foreground hover:bg-primary-foreground/30 transition-colors"
          >
            <MessageSquare className="h-3.5 w-3.5" />
            {t('kontakte.action.message')}
          </button>
        </div>
      </div>

      {/* ------------------------------------------------------------------ */}
      {/* Two-column body                                                      */}
      {/* ------------------------------------------------------------------ */}
      <div className="flex flex-1 flex-col overflow-y-auto lg:flex-row lg:overflow-hidden">
        {/* Left column: stammdaten */}
        <div className="p-5 space-y-5 min-w-0 border-b border-border lg:flex-1 lg:max-w-[340px] lg:overflow-y-auto lg:border-b-0 lg:border-r lg:border-border">

          {/* Contact data */}
          <section>
            <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-3">
              {t('kontakte.detail.contactData')}
            </h3>
            <div className="space-y-2.5">
              {contact.email && (
                <div className="flex items-center gap-3 text-sm">
                  <Mail className="h-4 w-4 text-muted-foreground shrink-0" />
                  <a href={`mailto:${contact.email}`} className="text-primary hover:underline truncate">
                    {contact.email}
                  </a>
                </div>
              )}
              {contact.phone && (
                <div className="flex items-center gap-3 text-sm">
                  <Phone className="h-4 w-4 text-muted-foreground shrink-0" />
                  <span className="text-foreground">{contact.phone}</span>
                </div>
              )}
              {contact.mobile && (
                <div className="flex items-center gap-3 text-sm">
                  <Phone className="h-4 w-4 text-muted-foreground shrink-0" />
                  <span className="text-foreground">{contact.mobile}</span>
                  <span className="text-xs text-muted-foreground">({t('kontakte.form.mobile')})</span>
                </div>
              )}
              {contact.company && (
                <div className="flex items-center gap-3 text-sm">
                  <Building2 className="h-4 w-4 text-muted-foreground shrink-0" />
                  <span className="text-foreground">
                    {contact.company}
                    {contact.department ? ` · ${contact.department}` : ''}
                  </span>
                </div>
              )}
              {contact.address.street && (
                <div className="flex items-start gap-3 text-sm">
                  <MapPin className="h-4 w-4 text-muted-foreground shrink-0 mt-0.5" />
                  <span className="text-foreground">
                    {contact.address.street},&nbsp;
                    {contact.address.zip} {contact.address.city}
                    {contact.address.country && contact.address.country !== 'Deutschland'
                      ? `, ${contact.address.country}`
                      : ''}
                  </span>
                </div>
              )}
              {contact.website && (
                <div className="flex items-center gap-3 text-sm">
                  <Globe className="h-4 w-4 text-muted-foreground shrink-0" />
                  <a
                    href={contact.website.startsWith('http') ? contact.website : `https://${contact.website}`}
                    className="text-primary hover:underline truncate"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    {contact.website}
                  </a>
                </div>
              )}
              <div className="flex items-center gap-3 text-sm">
                <Calendar className="h-4 w-4 text-muted-foreground shrink-0" />
                <span className="text-muted-foreground">
                  {t('kontakte.detail.lastContact')}: {formatDate(contact.lastContact)}
                </span>
              </div>
            </div>
          </section>

          {/* Social media */}
          {(contact.socialMedia.linkedin || contact.socialMedia.xing) && (
            <>
              <Separator />
              <section>
                <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-3">
                  {t('kontakte.detail.socialMedia')}
                </h3>
                <div className="flex flex-wrap gap-2">
                  {contact.socialMedia.linkedin && (
                    <a
                      href={`https://linkedin.com/in/${contact.socialMedia.linkedin}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
                    >
                      <Linkedin className="h-3.5 w-3.5 text-[#0077b5]" />
                      LinkedIn
                    </a>
                  )}
                  {contact.socialMedia.xing && (
                    <a
                      href={`https://xing.com/profile/${contact.socialMedia.xing}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
                    >
                      <Globe className="h-3.5 w-3.5 text-[#006567]" />
                      Xing
                    </a>
                  )}
                </div>
              </section>
            </>
          )}

          {/* Tags (always show section — even if empty, for add button) */}
          <Separator />
          <section>
            <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-3">
              {t('kontakte.detail.tags')}
            </h3>
            <TagEditor contactId={contact.id} tagNames={contact.tags} />
          </section>

          {/* Projects */}
          {contact.projects.length > 0 && (
            <>
              <Separator />
              <section>
                <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-3">
                  {t('kontakte.detail.projects')}
                </h3>
                <div className="space-y-1.5">
                  {contact.projects.map((project) => (
                    <div
                      key={project}
                      className="flex items-center gap-2 rounded-md border border-border bg-card px-3 py-2 text-sm text-foreground"
                    >
                      <Briefcase className="h-4 w-4 text-primary shrink-0" />
                      {project}
                    </div>
                  ))}
                </div>
              </section>
            </>
          )}

          {/* Notes */}
          {contact.notes && (
            <>
              <Separator />
              <section>
                <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-3">
                  {t('kontakte.detail.notes')}
                </h3>
                <p className="text-sm text-foreground whitespace-pre-wrap leading-relaxed">
                  {contact.notes}
                </p>
              </section>
            </>
          )}

          {/* ---------------------------------------------------------------- */}
          {/* Deals                                                          */}
          {/* ---------------------------------------------------------------- */}
          <Separator />
          <section>
            <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-3 flex items-center gap-1.5">
              <TrendingUp className="h-3.5 w-3.5" />
              {t('kontakte.detail.deals')}
            </h3>
            {contactDeals.length === 0 ? (
              <p className="text-xs text-muted-foreground">{t('kontakte.detail.dealsEmpty')}</p>
            ) : (
              <div className="space-y-1.5">
                {contactDeals.map((deal) => {
                  const d = deal as {
                    id: string
                    name: string
                    value: number
                    currency: string
                    stageName?: string
                    stageId?: string
                  }
                  const stageName = d.stageName ?? ''
                  const variant = stageVariantMap[stageName] ?? 'outline'
                  const formatted = new Intl.NumberFormat('de-DE', {
                    style: 'currency',
                    currency: d.currency ?? 'EUR',
                    maximumFractionDigits: 0,
                  }).format(d.value ?? 0)
                  return (
                    <button
                      key={d.id}
                      onClick={() => navigate(`/kontakte/pipeline/${d.id}`)}
                      className="flex w-full items-center justify-between gap-2 rounded-md border border-border bg-card px-3 py-2 text-left text-sm hover:bg-accent transition-colors"
                    >
                      <span className="truncate text-foreground">{d.name}</span>
                      <span className="flex shrink-0 items-center gap-1.5">
                        <span className="text-xs font-medium text-muted-foreground">{formatted}</span>
                        {stageName && (
                          <Badge variant={variant} className="text-[10px] px-1.5 py-0">
                            {stageName}
                          </Badge>
                        )}
                      </span>
                    </button>
                  )
                })}
              </div>
            )}
          </section>

          {/* ---------------------------------------------------------------- */}
          {/* E-Mail-Verlauf                                                  */}
          {/* ---------------------------------------------------------------- */}
          <Separator />
          <section>
            <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-3 flex items-center gap-1.5">
              <Mail className="h-3.5 w-3.5" />
              {t('kontakte.detail.emails')}
            </h3>
            {contactEmails.length === 0 ? (
              <p className="text-xs text-muted-foreground">{t('kontakte.detail.emailsEmpty')}</p>
            ) : (
              <div className="space-y-1.5">
                {contactEmails.map((msg) => {
                  const m = msg as { id: string; subject: string; date: string }
                  return (
                    <div
                      key={m.id}
                      className="flex items-center justify-between gap-2 rounded-md border border-border bg-card px-3 py-2 text-sm"
                    >
                      <span className="truncate text-foreground">{m.subject}</span>
                      <span className="shrink-0 text-[10px] text-muted-foreground">
                        {formatDate(m.date)}
                      </span>
                    </div>
                  )
                })}
              </div>
            )}
          </section>

          {/* ---------------------------------------------------------------- */}
          {/* Aufgaben                                                         */}
          {/* ---------------------------------------------------------------- */}
          <Separator />
          <section>
            <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-3 flex items-center gap-1.5">
              <CheckSquare className="h-3.5 w-3.5" />
              {t('kontakte.detail.tasks')}
            </h3>
            {contactTasks.length === 0 ? (
              <p className="text-xs text-muted-foreground">{t('kontakte.detail.tasksEmpty')}</p>
            ) : (
              <div className="space-y-1.5">
                {contactTasks.map((et) => {
                  const e = et as {
                    id: string
                    task_title: string
                    task_id?: string
                    status?: string
                  }
                  const statusVal = e.status ?? 'todo'
                  const variant = taskStatusVariantMap[statusVal] ?? 'outline'
                  return (
                    <div
                      key={e.id}
                      className="flex items-center justify-between gap-2 rounded-md border border-border bg-card px-3 py-2 text-sm"
                    >
                      <span className="truncate text-foreground">{e.task_title}</span>
                      <Badge variant={variant} className="shrink-0 text-[10px] px-1.5 py-0">
                        {statusVal}
                      </Badge>
                    </div>
                  )
                })}
              </div>
            )}
          </section>

          {/* ---------------------------------------------------------------- */}
          {/* Einwilligungen (DSGVO)                                           */}
          {/* ---------------------------------------------------------------- */}
          <Separator />
          <section>
            <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-3 flex items-center gap-1.5">
              <Shield className="h-3.5 w-3.5" />
              {t('kontakte.detail.consent')}
            </h3>
            <ConsentPanel
              contactId={contact.id}
              contactName={`${contact.firstName} ${contact.lastName}`}
            />
          </section>

          {/* Meta */}
          <Separator />
          <section className="pb-4">
            <p className="text-xs text-muted-foreground">
              {t('kontakte.detail.createdAt')} {formatDate(contact.createdAt)}
            </p>
          </section>
        </div>

        {/* Right column: timeline — stacked below on narrow, side-by-side on lg */}
        <div className="flex flex-col p-5 lg:w-0 lg:flex-1 lg:overflow-y-auto">
          <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-4">
            {t('kontakte.detail.timeline')}
          </h3>
          <ContactTimeline contactId={contact.id} />
        </div>
      </div>
    </div>
  )
}

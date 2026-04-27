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
  Video,
  FileText,
  StickyNote,
  ChevronLeft,
  Pencil,
  Trash2,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
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

const activityIcons: Record<string, typeof Mail> = {
  email: Mail,
  call: Phone,
  meeting: Video,
  note: StickyNote,
}

const activityLabelKeys: Record<string, string> = {
  email: 'kontakte.activity.email',
  call: 'kontakte.activity.call',
  meeting: 'kontakte.activity.meeting',
  note: 'kontakte.activity.note',
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
  const status = statusConfig[contact.status]

  return (
    <div className="flex flex-1 flex-col overflow-hidden">
      {/* Gradient Header */}
      <div className="bg-gradient-to-br from-primary to-primary-dark px-6 py-6">
        <div className="flex items-center justify-between mb-4">
          <button
            onClick={onClose}
            className="rounded-md p-1 text-primary-foreground/70 hover:text-primary-foreground md:hidden"
          >
            <ChevronLeft className="h-5 w-5" />
          </button>
          <div className="flex items-center gap-1.5">
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
          </div>
        </div>

        <div className="flex items-center gap-4">
          <div className="flex h-16 w-16 items-center justify-center rounded-full border-3 border-primary-foreground/30 bg-primary-foreground/20 text-xl font-medium text-primary-foreground">
            {contact.initials}
          </div>
          <div>
            <h2 className="text-primary-foreground">
              {contact.salutation && <span className="text-primary-foreground/70 text-sm mr-1">{contact.salutation}</span>}
              {contact.firstName} {contact.lastName}
            </h2>
            <p className="text-sm text-primary-foreground/70">
              {contact.jobTitle}{contact.jobTitle && contact.company ? ' · ' : ''}{contact.company}
            </p>
            <div className="flex items-center gap-2 mt-2">
              <Badge variant={status.variant} className="text-xs">
                <span className={`mr-1.5 inline-block h-1.5 w-1.5 rounded-full ${status.color}`} />
                {t(status.labelKey)}
              </Badge>
              <span className="rounded-full bg-primary-foreground/20 px-2 py-0.5 text-xs text-primary-foreground">
                {t(categoryLabelKeys[contact.category])}
              </span>
            </div>
          </div>
        </div>

        {/* Action buttons */}
        <div className="flex gap-2 mt-4">
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

      {/* Detail content */}
      <div className="flex-1 overflow-y-auto p-6 space-y-6">
        {/* Contact info */}
        <section>
          <h3 className="text-sm font-medium text-foreground mb-3">{t('kontakte.detail.contactData')}</h3>
          <div className="space-y-2.5">
            <div className="flex items-center gap-3 text-sm">
              <Mail className="h-4 w-4 text-muted-foreground shrink-0" />
              <a href={`mailto:${contact.email}`} className="text-primary hover:underline">{contact.email}</a>
            </div>
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
            <div className="flex items-center gap-3 text-sm">
              <Building2 className="h-4 w-4 text-muted-foreground shrink-0" />
              <span className="text-foreground">
                {contact.company}
                {contact.department ? ` · ${contact.department}` : ''}
              </span>
            </div>
            {contact.address.street && (
              <div className="flex items-center gap-3 text-sm">
                <MapPin className="h-4 w-4 text-muted-foreground shrink-0" />
                <span className="text-foreground">
                  {contact.address.street}, {contact.address.zip} {contact.address.city}
                  {contact.address.country !== 'Deutschland' ? `, ${contact.address.country}` : ''}
                </span>
              </div>
            )}
            {contact.website && (
              <div className="flex items-center gap-3 text-sm">
                <Globe className="h-4 w-4 text-muted-foreground shrink-0" />
                <a href={`https://${contact.website}`} className="text-primary hover:underline" target="_blank" rel="noopener noreferrer">
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

        {/* Social Media */}
        {(contact.socialMedia.linkedin || contact.socialMedia.xing) && (
          <>
            <Separator />
            <section>
              <h3 className="text-sm font-medium text-foreground mb-3">{t('kontakte.detail.socialMedia')}</h3>
              <div className="flex gap-2">
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

        {/* Tags */}
        {contact.tags.length > 0 && (
          <>
            <Separator />
            <section>
              <h3 className="text-sm font-medium text-foreground mb-3">{t('kontakte.detail.tags')}</h3>
              <div className="flex flex-wrap gap-1.5">
                {contact.tags.map((tag) => (
                  <Badge key={tag} variant="secondary" className="text-xs">
                    {tag}
                  </Badge>
                ))}
              </div>
            </section>
          </>
        )}

        {/* Projects */}
        {contact.projects.length > 0 && (
          <>
            <Separator />
            <section>
              <h3 className="text-sm font-medium text-foreground mb-3">{t('kontakte.detail.projects')}</h3>
              <div className="space-y-1.5">
                {contact.projects.map((project) => (
                  <div
                    key={project}
                    className="flex items-center gap-2 rounded-md border border-border bg-card px-3 py-2 text-sm text-foreground"
                  >
                    <Briefcase className="h-4 w-4 text-primary" />
                    {project}
                  </div>
                ))}
              </div>
            </section>
          </>
        )}

        {/* Activity Timeline */}
        {contact.activities.length > 0 && (
          <>
            <Separator />
            <section>
              <h3 className="text-sm font-medium text-foreground mb-3">{t('kontakte.detail.recentActivities')}</h3>
              <div className="space-y-3">
                {contact.activities.slice(0, 5).map((activity) => {
                  const Icon = activityIcons[activity.type] || FileText
                  return (
                    <div key={activity.id} className="flex items-start gap-3">
                      <div className="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-secondary">
                        <Icon className="h-3.5 w-3.5 text-muted-foreground" />
                      </div>
                      <div className="min-w-0 flex-1">
                        <p className="text-sm text-foreground">{activity.description}</p>
                        <div className="flex items-center gap-2 mt-0.5">
                          <span className="text-xs text-muted-foreground">
                            {t(activityLabelKeys[activity.type])}
                          </span>
                          <span className="text-xs text-muted-foreground">
                            · {formatDate(activity.date)}
                          </span>
                        </div>
                      </div>
                    </div>
                  )
                })}
              </div>
            </section>
          </>
        )}

        {/* Notes */}
        {contact.notes && (
          <>
            <Separator />
            <section>
              <h3 className="text-sm font-medium text-foreground mb-3">{t('kontakte.detail.notes')}</h3>
              <p className="text-sm text-text-body whitespace-pre-wrap">{contact.notes}</p>
            </section>
          </>
        )}

        {/* Meta info */}
        <Separator />
        <section className="pb-4">
          <p className="text-xs text-muted-foreground">
            {t('kontakte.detail.createdAt')} {formatDate(contact.createdAt)}
          </p>
        </section>
      </div>
    </div>
  )
}

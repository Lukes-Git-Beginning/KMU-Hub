/**
 * Contact detail page showing contact info, custom fields, tags, and activities.
 *
 * Accessed via /crm/contacts/:id. Shows full contact information with
 * linked company, custom fields, tags, and related activities.
 */
import { useParams, useNavigate, Link } from 'react-router-dom'
import {
  ArrowLeft,
  Pencil,
  Trash2,
  Mail,
  Phone,
  Building2,
  Briefcase,
} from 'lucide-react'
import { useContact } from '@/api/hooks/useContacts'
import { useActivities } from '@/api/hooks/useActivities'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Separator } from '@/components/ui/separator'
import { activityTypeLabel, activityTypeIcon } from '../activities/activityUtils'

export default function ContactDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data, isLoading, error, refetch } = useContact(id ?? '')
  const { data: activitiesData } = useActivities({
    contact_id: id,
    page_size: 10,
  })

  const contact = data?.contact
  const activities = activitiesData?.activities ?? []

  function showComingSoon() {
    alert('Kommt bald')
  }

  if (error) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="text-center">
          <p className="text-lg font-semibold text-foreground">
            Fehler beim Laden des Kontakts
          </p>
          <p className="mt-2 text-sm text-muted-foreground">
            {error instanceof Error ? error.message : 'Ein unerwarteter Fehler ist aufgetreten.'}
          </p>
          <Button variant="outline" className="mt-4" onClick={() => refetch()}>
            Erneut versuchen
          </Button>
        </div>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="p-6 space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full" />
        <Skeleton className="h-48 w-full" />
      </div>
    )
  }

  if (!contact) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="text-center">
          <p className="text-lg font-semibold text-foreground">
            Kontakt nicht gefunden
          </p>
          <Button
            variant="outline"
            className="mt-4"
            onClick={() => navigate('/crm/contacts')}
          >
            Zurueck zur Liste
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => navigate('/crm/contacts')}
          >
            <ArrowLeft className="h-4 w-4 mr-1" />
            Zurueck
          </Button>
          <h1 className="text-2xl font-semibold text-foreground">
            {contact.firstName} {contact.lastName}
          </h1>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={showComingSoon}>
            <Pencil className="h-4 w-4 mr-1" />
            Bearbeiten
          </Button>
          <Button
            variant="outline"
            size="sm"
            className="text-destructive"
            onClick={showComingSoon}
          >
            <Trash2 className="h-4 w-4 mr-1" />
            Loeschen
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Contact Info Card */}
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>Kontaktinformationen</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {contact.email && (
              <div className="flex items-center gap-3">
                <Mail className="h-4 w-4 text-muted-foreground" />
                <a
                  href={`mailto:${contact.email}`}
                  className="text-sm text-primary hover:underline"
                >
                  {contact.email}
                </a>
              </div>
            )}
            {contact.phone && (
              <div className="flex items-center gap-3">
                <Phone className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm">{contact.phone}</span>
              </div>
            )}
            {contact.title && (
              <div className="flex items-center gap-3">
                <Briefcase className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm">{contact.title}</span>
              </div>
            )}
            {contact.companyId && (
              <div className="flex items-center gap-3">
                <Building2 className="h-4 w-4 text-muted-foreground" />
                <Link
                  to={`/crm/companies/${contact.companyId}`}
                  className="text-sm text-primary hover:underline"
                >
                  {contact.companyName || 'Unternehmen'}
                </Link>
              </div>
            )}

            {contact.notes && (
              <>
                <Separator />
                <div>
                  <p className="text-sm font-medium text-muted-foreground mb-1">
                    Notizen
                  </p>
                  <p className="text-sm whitespace-pre-wrap">{contact.notes}</p>
                </div>
              </>
            )}

            {/* Custom fields */}
            {contact.customFields &&
              Object.keys(contact.customFields).length > 0 && (
                <>
                  <Separator />
                  <div>
                    <p className="text-sm font-medium text-muted-foreground mb-2">
                      Benutzerdefinierte Felder
                    </p>
                    <div className="grid grid-cols-2 gap-2">
                      {Object.entries(contact.customFields).map(
                        ([key, value]) => (
                          <div key={key}>
                            <p className="text-xs text-muted-foreground">
                              {key}
                            </p>
                            <p className="text-sm">{String(value)}</p>
                          </div>
                        )
                      )}
                    </div>
                  </div>
                </>
              )}
          </CardContent>
        </Card>

        {/* Tags Card */}
        <Card>
          <CardHeader>
            <CardTitle>Tags</CardTitle>
          </CardHeader>
          <CardContent>
            {contact.tags && contact.tags.length > 0 ? (
              <div className="flex flex-wrap gap-2">
                {contact.tags.map((tag) => (
                  <Badge
                    key={tag.id}
                    variant="secondary"
                    style={
                      tag.color
                        ? {
                            backgroundColor: `${tag.color}20`,
                            color: tag.color,
                            borderColor: tag.color,
                          }
                        : undefined
                    }
                  >
                    {tag.name}
                  </Badge>
                ))}
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">
                Keine Tags zugewiesen.
              </p>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Activities section */}
      <Card>
        <CardHeader>
          <CardTitle>Aktivitaeten</CardTitle>
        </CardHeader>
        <CardContent>
          {activities.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              Keine Aktivitaeten fuer diesen Kontakt.
            </p>
          ) : (
            <div className="space-y-3">
              {activities.map((activity) => {
                const Icon = activityTypeIcon(activity.activity_type)
                return (
                  <div
                    key={activity.id}
                    className="flex items-start gap-3 rounded-md border border-border p-3"
                  >
                    <Icon className="mt-0.5 h-4 w-4 text-muted-foreground shrink-0" />
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <p className="text-sm font-medium truncate">
                          {activity.subject}
                        </p>
                        <Badge variant="outline" className="text-xs shrink-0">
                          {activityTypeLabel(activity.activity_type)}
                        </Badge>
                        {activity.is_completed && (
                          <Badge variant="secondary" className="text-xs shrink-0">
                            Erledigt
                          </Badge>
                        )}
                      </div>
                      {activity.description && (
                        <p className="mt-1 text-xs text-muted-foreground line-clamp-2">
                          {activity.description}
                        </p>
                      )}
                      <p className="mt-1 text-xs text-muted-foreground">
                        {activity.created_at
                          ? new Date(activity.created_at).toLocaleDateString(
                              'de-DE',
                              {
                                day: '2-digit',
                                month: '2-digit',
                                year: 'numeric',
                                hour: '2-digit',
                                minute: '2-digit',
                              }
                            )
                          : ''}
                      </p>
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

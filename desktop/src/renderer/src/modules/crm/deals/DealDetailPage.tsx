/**
 * Deal detail page showing deal info, linked entities, and activities.
 *
 * Accessed via /crm/deals/:id. Shows deal value, stage, linked contact
 * and company, custom fields, tags, and related activities.
 */
import { useParams, useNavigate, Link } from 'react-router-dom'
import {
  ArrowLeft,
  Pencil,
  Trash2,
  TrendingUp,
  CalendarDays,
  User,
  Building2,
} from 'lucide-react'
import { useDeal } from '@/api/hooks/useDeals'
import { useActivities } from '@/api/hooks/useActivities'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Separator } from '@/components/ui/separator'
import { activityTypeLabel, activityTypeIcon } from '../activities/activityUtils'

function formatCurrency(value?: number, currency?: string): string {
  if (value == null) return '-'
  return new Intl.NumberFormat('de-DE', {
    style: 'currency',
    currency: currency || 'EUR',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(value)
}

export default function DealDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data, isLoading, error, refetch } = useDeal(id ?? '')
  const { data: activitiesData } = useActivities({
    deal_id: id,
    page_size: 10,
  })

  const deal = data?.deal
  const activities = activitiesData?.activities ?? []

  function showComingSoon() {
    alert('Kommt bald')
  }

  if (error) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="text-center">
          <p className="text-lg font-semibold text-foreground">
            Fehler beim Laden des Deals
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

  if (!deal) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="text-center">
          <p className="text-lg font-semibold text-foreground">
            Deal nicht gefunden
          </p>
          <Button
            variant="outline"
            className="mt-4"
            onClick={() => navigate('/crm/deals')}
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
            onClick={() => navigate('/crm/deals')}
          >
            <ArrowLeft className="h-4 w-4 mr-1" />
            Zurueck
          </Button>
          <div>
            <h1 className="text-2xl font-semibold text-foreground">
              {deal.name}
            </h1>
            <p className="text-lg font-bold text-primary">
              {formatCurrency(deal.value, deal.currency)}
            </p>
          </div>
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
        {/* Deal Info Card */}
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>Deal-Informationen</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center gap-3">
              <TrendingUp className="h-4 w-4 text-muted-foreground" />
              <div>
                <p className="text-xs text-muted-foreground">Phase</p>
                <Badge variant="outline">{deal.stageName || '-'}</Badge>
              </div>
            </div>

            {deal.expectedCloseDate && (
              <div className="flex items-center gap-3">
                <CalendarDays className="h-4 w-4 text-muted-foreground" />
                <div>
                  <p className="text-xs text-muted-foreground">
                    Erwarteter Abschluss
                  </p>
                  <p className="text-sm">
                    {new Date(deal.expectedCloseDate).toLocaleDateString(
                      'de-DE'
                    )}
                  </p>
                </div>
              </div>
            )}

            {deal.contactId && (
              <div className="flex items-center gap-3">
                <User className="h-4 w-4 text-muted-foreground" />
                <div>
                  <p className="text-xs text-muted-foreground">Kontakt</p>
                  <Link
                    to={`/crm/contacts/${deal.contactId}`}
                    className="text-sm text-primary hover:underline"
                  >
                    {deal.contactName || 'Kontakt'}
                  </Link>
                </div>
              </div>
            )}

            {deal.companyId && (
              <div className="flex items-center gap-3">
                <Building2 className="h-4 w-4 text-muted-foreground" />
                <div>
                  <p className="text-xs text-muted-foreground">Unternehmen</p>
                  <Link
                    to={`/crm/companies/${deal.companyId}`}
                    className="text-sm text-primary hover:underline"
                  >
                    {deal.companyName || 'Unternehmen'}
                  </Link>
                </div>
              </div>
            )}

            {deal.closedAt && (
              <>
                <Separator />
                <div>
                  <p className="text-xs text-muted-foreground">
                    Abgeschlossen am
                  </p>
                  <p className="text-sm">
                    {new Date(deal.closedAt).toLocaleDateString('de-DE', {
                      day: '2-digit',
                      month: '2-digit',
                      year: 'numeric',
                      hour: '2-digit',
                      minute: '2-digit',
                    })}
                  </p>
                </div>
              </>
            )}

            {deal.notes && (
              <>
                <Separator />
                <div>
                  <p className="text-sm font-medium text-muted-foreground mb-1">
                    Notizen
                  </p>
                  <p className="text-sm whitespace-pre-wrap">{deal.notes}</p>
                </div>
              </>
            )}

            {deal.customFields &&
              Object.keys(deal.customFields).length > 0 && (
                <>
                  <Separator />
                  <div>
                    <p className="text-sm font-medium text-muted-foreground mb-2">
                      Benutzerdefinierte Felder
                    </p>
                    <div className="grid grid-cols-2 gap-2">
                      {Object.entries(deal.customFields).map(([key, value]) => (
                        <div key={key}>
                          <p className="text-xs text-muted-foreground">{key}</p>
                          <p className="text-sm">{String(value)}</p>
                        </div>
                      ))}
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
            {deal.tags && deal.tags.length > 0 ? (
              <div className="flex flex-wrap gap-2">
                {deal.tags.map((tag) => (
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
              Keine Aktivitaeten fuer diesen Deal.
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
                          <Badge
                            variant="secondary"
                            className="text-xs shrink-0"
                          >
                            Erledigt
                          </Badge>
                        )}
                      </div>
                      <p className="mt-1 text-xs text-muted-foreground">
                        {activity.created_at
                          ? new Date(activity.created_at).toLocaleDateString(
                              'de-DE',
                              {
                                day: '2-digit',
                                month: '2-digit',
                                year: 'numeric',
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

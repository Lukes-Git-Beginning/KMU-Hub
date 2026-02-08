/**
 * Activities list page with type filters, pagination, and complete toggle.
 *
 * Displays all activities across CRM entities. Supports filtering by
 * activity type (call, meeting, note, email, task) and toggling completion.
 */
import { useState, useEffect } from 'react'
import { Plus, ChevronLeft, ChevronRight, Activity } from 'lucide-react'
import {
  useActivities,
  useCompleteActivity,
} from '@/api/hooks/useActivities'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ACTIVITY_TYPES, activityTypeLabel, activityTypeIcon } from './activityUtils'

const PAGE_SIZE = 20

type ActivityTypeFilter = 'all' | 'call' | 'meeting' | 'note' | 'email' | 'task'

export default function ActivitiesListPage() {
  const [typeFilter, setTypeFilter] = useState<ActivityTypeFilter>('all')
  const [page, setPage] = useState(1)
  const completeActivity = useCompleteActivity()

  // Reset page when filter changes
  useEffect(() => {
    setPage(1)
  }, [typeFilter])

  const { data, isLoading, error, refetch } = useActivities({
    page,
    page_size: PAGE_SIZE,
    activity_type:
      typeFilter === 'all'
        ? undefined
        : (typeFilter as 'call' | 'meeting' | 'note' | 'email' | 'task'),
    sort_by: 'created_at',
    sort_desc: true,
  })

  const activities = data?.activities ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  function showComingSoon() {
    alert('Kommt bald')
  }

  if (error) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="text-center">
          <p className="text-lg font-semibold text-foreground">
            Fehler beim Laden der Aktivitaeten
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

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Aktivitaeten</h1>
        <Button onClick={showComingSoon} className="gap-2">
          <Plus className="h-4 w-4" />
          Neue Aktivitaet
        </Button>
      </div>

      {/* Type filter tabs */}
      <Tabs
        value={typeFilter}
        onValueChange={(v) => setTypeFilter(v as ActivityTypeFilter)}
      >
        <TabsList>
          <TabsTrigger value="all">Alle</TabsTrigger>
          {ACTIVITY_TYPES.map((t) => (
            <TabsTrigger key={t.value} value={t.value}>
              {t.label}
            </TabsTrigger>
          ))}
        </TabsList>
      </Tabs>

      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 8 }).map((_, i) => (
            <Skeleton key={i} className="h-20 w-full" />
          ))}
        </div>
      ) : activities.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16">
          <Activity className="h-12 w-12 text-muted-foreground" />
          <p className="mt-4 text-lg font-medium text-foreground">
            Keine Aktivitaeten gefunden
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            {typeFilter !== 'all'
              ? `Keine ${activityTypeLabel(typeFilter)}-Aktivitaeten vorhanden.`
              : 'Erstelle deine erste Aktivitaet, um loszulegen.'}
          </p>
        </div>
      ) : (
        <>
          <div className="space-y-3">
            {activities.map((activity) => {
              const Icon = activityTypeIcon(activity.activity_type)
              return (
                <div
                  key={activity.id}
                  className="flex items-start gap-4 rounded-lg border border-border p-4 transition-colors hover:bg-accent/30"
                >
                  <div
                    className={`mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-full ${
                      activity.is_completed
                        ? 'bg-green-100 text-green-600'
                        : 'bg-muted text-muted-foreground'
                    }`}
                  >
                    <Icon className="h-4 w-4" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <p className="text-sm font-medium">{activity.subject}</p>
                      <Badge variant="outline" className="text-xs">
                        {activityTypeLabel(activity.activity_type)}
                      </Badge>
                      {activity.is_completed && (
                        <Badge
                          variant="secondary"
                          className="text-xs bg-green-100 text-green-700"
                        >
                          Erledigt
                        </Badge>
                      )}
                    </div>
                    {activity.description && (
                      <p className="mt-1 text-sm text-muted-foreground line-clamp-2">
                        {activity.description}
                      </p>
                    )}
                    <div className="mt-2 flex items-center gap-3 text-xs text-muted-foreground">
                      {activity.contact_name && (
                        <span>Kontakt: {activity.contact_name}</span>
                      )}
                      {activity.company_name && (
                        <span>Unternehmen: {activity.company_name}</span>
                      )}
                      {activity.deal_name && (
                        <span>Deal: {activity.deal_name}</span>
                      )}
                      {activity.due_date && (
                        <span>
                          Faellig:{' '}
                          {new Date(activity.due_date).toLocaleDateString(
                            'de-DE'
                          )}
                        </span>
                      )}
                      {activity.created_at && (
                        <span>
                          Erstellt:{' '}
                          {new Date(activity.created_at).toLocaleDateString(
                            'de-DE'
                          )}
                        </span>
                      )}
                    </div>
                  </div>
                  <div className="shrink-0">
                    {!activity.is_completed && (
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={completeActivity.isPending}
                        onClick={() => {
                          if (activity.id) {
                            completeActivity.mutate(activity.id)
                          }
                        }}
                      >
                        Erledigen
                      </Button>
                    )}
                  </div>
                </div>
              )
            })}
          </div>

          <div className="flex items-center justify-between">
            <p className="text-sm text-muted-foreground">
              {total} Aktivitaet{total !== 1 ? 'en' : ''} gesamt
            </p>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={page <= 1}
                onClick={() => setPage((p) => Math.max(1, p - 1))}
              >
                <ChevronLeft className="h-4 w-4" />
              </Button>
              <span className="text-sm text-muted-foreground">
                Seite {page} von {totalPages}
              </span>
              <Button
                variant="outline"
                size="sm"
                disabled={page >= totalPages}
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              >
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </>
      )}
    </div>
  )
}

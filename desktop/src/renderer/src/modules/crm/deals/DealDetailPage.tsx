/**
 * Deal detail page showing deal info, linked entities, activities,
 * and linked tasks (Aufgaben tab).
 *
 * Accessed via /crm/deals/:id. Shows deal value, stage, linked contact
 * and company, custom fields, tags, related activities, and tasks
 * from the Work module. Supports auto-populating task creation from deal.
 */
import { useState } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import {
  ArrowLeft,
  Pencil,
  Trash2,
  TrendingUp,
  CalendarDays,
  User,
  Building2,
  Plus,
  Calendar,
} from 'lucide-react'
import { cn } from '@/lib'
import { useDeal } from '@/api/hooks/useDeals'
import { useActivities } from '@/api/hooks/useActivities'
import { useEntityTasks } from '@/api/hooks/useTasks'
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

const PRIORITY_LABELS: Record<string, string> = {
  urgent: 'Dringend',
  high: 'Hoch',
  normal: 'Normal',
  low: 'Niedrig',
}

const PRIORITY_COLORS: Record<string, string> = {
  urgent: 'bg-red-100 text-red-700 border-red-300',
  high: 'bg-orange-100 text-orange-700 border-orange-300',
  normal: 'bg-blue-100 text-blue-700 border-blue-300',
  low: 'bg-gray-100 text-gray-500 border-gray-300',
}

export default function DealDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [activeSection, setActiveSection] = useState<'activities' | 'tasks'>('activities')

  const { data, isLoading, error, refetch } = useDeal(id ?? '')
  const { data: activitiesData } = useActivities({
    deal_id: id,
    page_size: 10,
  })
  const { data: tasksData } = useEntityTasks('deal', id ?? '')

  const deal = data?.deal
  const activities = activitiesData?.activities ?? []
  const linkedTasks = tasksData?.tasks ?? []

  function showComingSoon() {
    alert('Kommt bald')
  }

  /** Navigate to task creation with deal auto-populate params */
  function handleCreateTaskFromDeal() {
    if (!deal) return
    navigate(`/work/my-tasks?from_deal=${id}`)
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

      {/* Tab navigation: Activities / Tasks */}
      <Card>
        <CardHeader className="pb-0">
          <div className="flex items-center gap-4">
            <button
              type="button"
              className={cn(
                'pb-2 text-sm font-medium border-b-2 transition-colors',
                activeSection === 'activities'
                  ? 'border-primary text-primary'
                  : 'border-transparent text-muted-foreground hover:text-foreground'
              )}
              onClick={() => setActiveSection('activities')}
            >
              Aktivitaeten
              {activities.length > 0 && (
                <span className="ml-1.5 text-xs text-muted-foreground">
                  ({activities.length})
                </span>
              )}
            </button>
            <button
              type="button"
              className={cn(
                'pb-2 text-sm font-medium border-b-2 transition-colors',
                activeSection === 'tasks'
                  ? 'border-primary text-primary'
                  : 'border-transparent text-muted-foreground hover:text-foreground'
              )}
              onClick={() => setActiveSection('tasks')}
            >
              Aufgaben
              {linkedTasks.length > 0 && (
                <span className="ml-1.5 text-xs text-muted-foreground">
                  ({linkedTasks.length})
                </span>
              )}
            </button>
          </div>
        </CardHeader>
        <CardContent className="pt-4">
          {activeSection === 'activities' && (
            <>
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
            </>
          )}

          {activeSection === 'tasks' && (
            <>
              <div className="flex items-center justify-between mb-3">
                <p className="text-sm text-muted-foreground">
                  {linkedTasks.length} verknuepfte Aufgabe{linkedTasks.length !== 1 ? 'n' : ''}
                </p>
                <Button
                  variant="outline"
                  size="sm"
                  className="gap-1 h-7 text-xs"
                  onClick={handleCreateTaskFromDeal}
                >
                  <Plus className="h-3.5 w-3.5" />
                  Aufgabe erstellen
                </Button>
              </div>

              {linkedTasks.length === 0 ? (
                <p className="text-sm text-muted-foreground py-4 text-center">
                  Keine Aufgaben mit diesem Deal verknuepft.
                </p>
              ) : (
                <div className="space-y-1">
                  {linkedTasks.map((task) => {
                    const taskKey =
                      task.project_key && task.task_number
                        ? `${task.project_key}-${task.task_number}`
                        : ''
                    const priorityLabel = PRIORITY_LABELS[task.priority ?? 'normal']
                    const priorityColor = PRIORITY_COLORS[task.priority ?? 'normal']

                    return (
                      <div
                        key={task.id}
                        className="flex items-center gap-3 rounded-md border border-border px-3 py-2 hover:bg-accent/50 cursor-pointer transition-colors"
                        onClick={() => {
                          if (task.project_id && task.id) {
                            navigate(
                              `/work/projects/${task.project_id}/tasks/${task.id}`
                            )
                          }
                        }}
                      >
                        {taskKey && (
                          <span className="text-xs font-mono text-muted-foreground shrink-0">
                            {taskKey}
                          </span>
                        )}

                        <span className="text-sm flex-1 truncate">
                          {task.title}
                        </span>

                        {task.status_name && (
                          <span
                            className="inline-flex items-center rounded-full px-1.5 py-0.5 text-xs border"
                            style={{
                              backgroundColor: `${task.status_color ?? '#6b7280'}15`,
                              color: task.status_color ?? '#6b7280',
                              borderColor: `${task.status_color ?? '#6b7280'}40`,
                            }}
                          >
                            {task.status_name}
                          </span>
                        )}

                        <Badge
                          variant="outline"
                          className={`text-xs shrink-0 ${priorityColor}`}
                        >
                          {priorityLabel}
                        </Badge>

                        {task.assignee_name && (
                          <span className="text-xs text-muted-foreground flex items-center gap-1 shrink-0">
                            <User className="h-3 w-3" />
                            {task.assignee_name}
                          </span>
                        )}

                        {task.due_date && (
                          <span className="text-xs text-muted-foreground flex items-center gap-1 shrink-0">
                            <Calendar className="h-3 w-3" />
                            {new Date(task.due_date).toLocaleDateString('de-DE', {
                              day: '2-digit',
                              month: '2-digit',
                            })}
                          </span>
                        )}
                      </div>
                    )
                  })}
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

/**
 * Activities list page with type filters, pagination, and complete toggle.
 *
 * Displays all activities across CRM entities. Supports filtering by
 * activity type (call, meeting, note, email, task) and toggling completion.
 */
import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, ChevronLeft, ChevronRight, Activity, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { PageHeader } from '@/components/shared'
import {
  useActivities,
  useCreateActivity,
  useUpdateActivity,
  useDeleteActivity,
  useCompleteActivity,
} from '@/api/hooks/useActivities'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ACTIVITY_TYPES, activityTypeLabel, activityTypeIcon } from './activityUtils'
import { ActivityFormDialog, type ActivityFormData } from './ActivityFormDialog'

const PAGE_SIZE = 20

type ActivityTypeFilter = 'all' | 'call' | 'meeting' | 'note' | 'email' | 'task'

export default function ActivitiesListPage() {
  const { t } = useTranslation()
  const [typeFilter, setTypeFilter] = useState<ActivityTypeFilter>('all')
  const [page, setPage] = useState(1)
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [editingActivity, setEditingActivity] = useState<{
    id: string
    data: Partial<ActivityFormData>
  } | null>(null)

  const createActivity = useCreateActivity()
  const updateActivity = useUpdateActivity()
  const deleteActivity = useDeleteActivity()
  const completeActivity = useCompleteActivity()

  // Reset page when filter changes
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- reset state on dependency change
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

  async function handleCreate(form: ActivityFormData) {
    try {
      await createActivity.mutateAsync({
        activity_type: form.activity_type,
        subject: form.subject,
        description: form.description || undefined,
        due_date: form.due_date || undefined,
      })
      toast.success(t('crm.activities.created'))
    } catch {
      toast.error(t('crm.activities.createError'))
    }
  }

  async function handleUpdate(form: ActivityFormData) {
    if (!editingActivity) return
    try {
      await updateActivity.mutateAsync({
        id: editingActivity.id,
        subject: form.subject,
        description: form.description || undefined,
        due_date: form.due_date || undefined,
      })
      toast.success(t('crm.activities.updated'))
    } catch {
      toast.error(t('crm.activities.updateError'))
    }
  }

  async function handleDelete(id: string) {
    try {
      await deleteActivity.mutateAsync(id)
      toast.success(t('crm.activities.deleted'))
    } catch {
      toast.error(t('crm.activities.deleteError'))
    }
  }

  if (error) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="text-center">
          <p className="text-lg font-semibold text-foreground">
            {t('crm.activities.loadError')}
          </p>
          <p className="mt-2 text-sm text-muted-foreground">
            {error instanceof Error ? error.message : t('crm.error.unexpected')}
          </p>
          <Button variant="outline" className="mt-4" onClick={() => refetch()}>
            {t('common.retry')}
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="p-6 space-y-4">
      <PageHeader
        title={t('crm.activities.title')}
        icon={Activity}
        moduleId="contacts"
        actions={
          <Button onClick={() => setShowCreateDialog(true)} className="gap-2">
            <Plus className="h-4 w-4" />
            {t('crm.activities.new')}
          </Button>
        }
      />

      {/* Type filter tabs */}
      <Tabs
        value={typeFilter}
        onValueChange={(v) => setTypeFilter(v as ActivityTypeFilter)}
      >
        <TabsList>
          <TabsTrigger value="all">{t('crm.activities.filterAll')}</TabsTrigger>
          {ACTIVITY_TYPES.map((at) => (
            <TabsTrigger key={at.value} value={at.value}>
              {t(at.labelKey)}
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
            {t('crm.activities.noResults')}
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            {typeFilter !== 'all'
              ? t('crm.activities.noResultsFiltered', { type: t(activityTypeLabel(typeFilter)) })
              : t('crm.activities.emptyHint')}
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
                  className="flex items-start gap-4 rounded-lg border border-border p-4 transition-colors hover:bg-accent/30 cursor-pointer"
                  onClick={() =>
                    activity.id &&
                    setEditingActivity({
                      id: activity.id,
                      data: {
                        activity_type: activity.activity_type as ActivityFormData['activity_type'],
                        subject: activity.subject ?? '',
                        description: activity.description ?? '',
                        due_date: activity.due_date
                          ? activity.due_date.slice(0, 10)
                          : '',
                      },
                    })
                  }
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
                        {t(activityTypeLabel(activity.activity_type))}
                      </Badge>
                      {activity.is_completed && (
                        <Badge
                          variant="secondary"
                          className="text-xs bg-green-100 text-green-700"
                        >
                          {t('crm.activities.completed')}
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
                        <span>{t('crm.field.contact')}: {activity.contact_name}</span>
                      )}
                      {activity.company_name && (
                        <span>{t('crm.field.company')}: {activity.company_name}</span>
                      )}
                      {activity.deal_name && (
                        <span>Deal: {activity.deal_name}</span>
                      )}
                      {activity.due_date && (
                        <span>
                          {t('crm.activities.due')}:{' '}
                          {new Date(activity.due_date).toLocaleDateString(
                            'de-DE'
                          )}
                        </span>
                      )}
                      {activity.created_at && (
                        <span>
                          {t('crm.field.created')}:{' '}
                          {new Date(activity.created_at).toLocaleDateString(
                            'de-DE'
                          )}
                        </span>
                      )}
                    </div>
                  </div>
                  <div className="shrink-0 flex items-center gap-1">
                    {!activity.is_completed && (
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={completeActivity.isPending}
                        onClick={(e) => {
                          e.stopPropagation()
                          if (activity.id) {
                            completeActivity.mutate(activity.id)
                          }
                        }}
                      >
                        {t('crm.activities.markComplete')}
                      </Button>
                    )}
                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-muted-foreground hover:text-destructive"
                      onClick={(e) => {
                        e.stopPropagation()
                        if (activity.id) handleDelete(activity.id)
                      }}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              )
            })}
          </div>

          <div className="flex items-center justify-between">
            <p className="text-sm text-muted-foreground">
              {t('crm.activities.totalCount', { count: total })}
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
                {t('crm.pagination', { page, totalPages })}
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
      <ActivityFormDialog
        open={showCreateDialog}
        onOpenChange={setShowCreateDialog}
        onSubmit={handleCreate}
      />

      <ActivityFormDialog
        open={editingActivity !== null}
        onOpenChange={(open) => {
          if (!open) setEditingActivity(null)
        }}
        initialData={editingActivity?.data}
        onSubmit={handleUpdate}
        isEdit
      />
    </div>
  )
}

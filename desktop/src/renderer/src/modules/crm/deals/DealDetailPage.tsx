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
import { useTranslation } from 'react-i18next'
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
  FileText,
} from 'lucide-react'
import { toast } from 'sonner'
import { cn } from '@/lib'
import { useDeal, useUpdateDeal, useDeleteDeal } from '@/api/hooks/useDeals'
import { usePipelineStages } from '@/api/hooks/usePipelineStages'
import { useCreateQuoteFromDeal } from '@/api/hooks/useFinance'
import { useActivities } from '@/api/hooks/useActivities'
import { useEntityTasks } from '@/api/hooks/useTasks'
import { Button } from '@/components/ui/button'
import { useHasCapability, useScopedCapability } from '@/hooks/useCapability'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Separator } from '@/components/ui/separator'
import { ConfirmDialog } from '@/components/shared'
import { activityTypeLabel, activityTypeIcon } from '../activities/activityUtils'
import { DealFormDialog, type DealFormData } from './DealFormDialog'

function formatCurrency(value?: number, currency?: string): string {
  if (value == null) return '-'
  return new Intl.NumberFormat('de-DE', {
    style: 'currency',
    currency: currency || 'EUR',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(value)
}

const PRIORITY_LABEL_KEYS: Record<string, string> = {
  urgent: 'crm.priority.urgent',
  high: 'crm.priority.high',
  normal: 'crm.priority.normal',
  low: 'crm.priority.low',
}

const PRIORITY_COLORS: Record<string, string> = {
  urgent: 'bg-red-100 text-red-700 border-red-300',
  high: 'bg-orange-100 text-orange-700 border-orange-300',
  normal: 'bg-blue-100 text-blue-700 border-blue-300',
  low: 'bg-gray-100 text-gray-500 border-gray-300',
}

export default function DealDetailPage() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [activeSection, setActiveSection] = useState<'activities' | 'tasks'>('activities')

  const { data, isLoading, error, refetch } = useDeal(id ?? '')
  const { data: activitiesData } = useActivities({
    deal_id: id,
    page_size: 10,
  })
  const { data: tasksData } = useEntityTasks('deal', id ?? '')
  const createQuoteFromDeal = useCreateQuoteFromDeal()
  const updateDeal = useUpdateDeal()
  const deleteDeal = useDeleteDeal()
  const { data: stagesData } = usePipelineStages()
  const stages = stagesData?.stages ?? []
  const [showEditDialog, setShowEditDialog] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)

  const deal = data?.deal
  const activities = activitiesData?.activities ?? []
  const linkedTasks = tasksData?.tasks ?? []

  // RBAC — CRM-Objekte kein owner-Feld → ownerIds leer → scope='own' ergibt deny
  const canEdit = useScopedCapability('crm:deal:edit')
  const canDelete = useScopedCapability('crm:deal:delete')
  const canCreateQuote = useHasCapability('finance:quote:create')
  const canCreateTask = useHasCapability('work:task:create')

  function dealToFormData(): Partial<DealFormData> {
    if (!deal) return {}
    const cf = (deal.customFields ?? {}) as Record<string, unknown>
    return {
      name: deal.name ?? '',
      value: deal.value ?? 0,
      currency: deal.currency ?? 'CHF',
      stage: deal.stageId ?? '',
      priority: (cf._priority as string) ?? 'normal',
      probability: (cf._probability as number) ?? 50,
      contactName: deal.contactName ?? (cf._contactName as string) ?? '',
      companyName: deal.companyName ?? (cf._companyName as string) ?? '',
      expectedCloseDate: deal.expectedCloseDate ?? '',
      notes: deal.notes ?? '',
      tags: (cf._tags as string[]) ?? [],
    }
  }

  async function handleUpdate(form: DealFormData) {
    if (!id) return
    try {
      await updateDeal.mutateAsync({
        id,
        name: form.name,
        value: form.value,
        currency: form.currency,
        expected_close_date: form.expectedCloseDate || undefined,
        notes: form.notes || undefined,
        custom_fields: {
          ...(form.contactName ? { _contactName: form.contactName } : {}),
          ...(form.companyName ? { _companyName: form.companyName } : {}),
          ...(form.priority !== 'normal' ? { _priority: form.priority } : {}),
          ...(form.probability !== 50 ? { _probability: form.probability } : {}),
          ...(form.tags.length > 0 ? { _tags: form.tags } : {}),
        },
      })
      toast.success(t('crm.deals.updated'))
    } catch {
      toast.error(t('crm.deals.updateError'))
    }
  }

  async function handleDelete() {
    if (!id) return
    try {
      await deleteDeal.mutateAsync(id)
      toast.success(t('crm.deals.deleted'))
      navigate('/kontakte/pipeline')
    } catch {
      toast.error(t('crm.deals.deleteError'))
    }
  }

  /** Navigate to task creation with deal auto-populate params */
  function handleCreateTaskFromDeal() {
    if (!deal) return
    navigate(`/work/my-tasks?from_deal=${id}`)
  }

  /** Create a quote pre-populated with deal customer data */
  function handleCreateQuoteFromDeal() {
    if (!id) return
    createQuoteFromDeal.mutate(id, {
      onSuccess: () => {
        navigate('/finanzen')
      },
      onError: () => {
        toast.error(t('crm.deals.quoteError'))
      },
    })
  }

  if (error) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="text-center">
          <p className="text-lg font-semibold text-foreground">
            {t('crm.deals.loadErrorSingle')}
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
            {t('crm.deals.notFound')}
          </p>
          <Button
            variant="outline"
            className="mt-4"
            onClick={() => navigate('/kontakte/pipeline')}
          >
            {t('crm.backToList')}
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
            onClick={() => navigate('/kontakte/pipeline')}
          >
            <ArrowLeft className="h-4 w-4 mr-1" />
            {t('common.back')}
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
          {canCreateQuote && (
            <Button
              variant="outline"
              size="sm"
              onClick={handleCreateQuoteFromDeal}
              disabled={createQuoteFromDeal.isPending}
            >
              <FileText className="h-4 w-4 mr-1" />
              {createQuoteFromDeal.isPending ? t('crm.deals.creatingQuote') : t('crm.deals.createQuote')}
            </Button>
          )}
          {canEdit && (
            <Button variant="outline" size="sm" onClick={() => setShowEditDialog(true)}>
              <Pencil className="h-4 w-4 mr-1" />
              {t('common.edit')}
            </Button>
          )}
          {canDelete && (
            <Button
              variant="outline"
              size="sm"
              className="text-destructive"
              onClick={() => setShowDeleteConfirm(true)}
            >
              <Trash2 className="h-4 w-4 mr-1" />
              {t('common.delete')}
            </Button>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Deal Info Card */}
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>{t('crm.deals.info')}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center gap-3">
              <TrendingUp className="h-4 w-4 text-muted-foreground" />
              <div>
                <p className="text-xs text-muted-foreground">{t('crm.deals.stage')}</p>
                <Badge variant="outline">{deal.stageName || '-'}</Badge>
              </div>
            </div>

            {deal.expectedCloseDate && (
              <div className="flex items-center gap-3">
                <CalendarDays className="h-4 w-4 text-muted-foreground" />
                <div>
                  <p className="text-xs text-muted-foreground">
                    {t('crm.deals.expectedClose')}
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
                  <p className="text-xs text-muted-foreground">{t('crm.field.contact')}</p>
                  <Link
                    to="/kontakte"
                    className="text-sm text-primary hover:underline"
                  >
                    {deal.contactName || t('crm.field.contact')}
                  </Link>
                </div>
              </div>
            )}

            {deal.companyId && (
              <div className="flex items-center gap-3">
                <Building2 className="h-4 w-4 text-muted-foreground" />
                <div>
                  <p className="text-xs text-muted-foreground">{t('crm.field.company')}</p>
                  <Link
                    to={`/kontakte/firmen/${deal.companyId}`}
                    className="text-sm text-primary hover:underline"
                  >
                    {deal.companyName || t('crm.field.company')}
                  </Link>
                </div>
              </div>
            )}

            {deal.closedAt && (
              <>
                <Separator />
                <div>
                  <p className="text-xs text-muted-foreground">
                    {t('crm.deals.closedAt')}
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
                    {t('crm.field.notes')}
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
                      {t('crm.field.customFields')}
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
            <CardTitle>{t('crm.field.tags')}</CardTitle>
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
                {t('crm.tags.none')}
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
              {t('crm.activities.title')}
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
              {t('crm.tasks.title')}
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
                  {t('crm.deals.noActivities')}
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
                              {t(activityTypeLabel(activity.activity_type))}
                            </Badge>
                            {activity.is_completed && (
                              <Badge
                                variant="secondary"
                                className="text-xs shrink-0"
                              >
                                {t('crm.activities.completed')}
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
                  {t('crm.tasks.linkedCount', { count: linkedTasks.length })}
                </p>
                {canCreateTask && (
                  <Button
                    variant="outline"
                    size="sm"
                    className="gap-1 h-7 text-xs"
                    onClick={handleCreateTaskFromDeal}
                  >
                    <Plus className="h-3.5 w-3.5" />
                    {t('crm.tasks.create')}
                  </Button>
                )}
              </div>

              {linkedTasks.length === 0 ? (
                <p className="text-sm text-muted-foreground py-4 text-center">
                  {t('crm.deals.noTasks')}
                </p>
              ) : (
                <div className="space-y-1">
                  {linkedTasks.map((task) => {
                    const taskKey =
                      task.project_key && task.task_number
                        ? `${task.project_key}-${task.task_number}`
                        : ''
                    const priorityLabel = t(PRIORITY_LABEL_KEYS[task.priority ?? 'normal'])
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

      <DealFormDialog
        open={showEditDialog}
        onOpenChange={setShowEditDialog}
        initialData={dealToFormData()}
        onSubmit={handleUpdate}
        isEdit
        stages={stages}
      />

      <ConfirmDialog
        open={showDeleteConfirm}
        onOpenChange={setShowDeleteConfirm}
        title={t('crm.deals.deleteTitle')}
        description={t('crm.deals.deleteConfirm', { name: deal.name })}
        confirmLabel={t('common.delete')}
        variant="destructive"
        onConfirm={handleDelete}
      />
    </div>
  )
}

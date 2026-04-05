/**
 * ActivityFeed widget -- timeline of recent CRM activities.
 *
 * Each entry shows activity type icon, subject, linked entity name,
 * and relative timestamp. Click navigates to the linked entity.
 */
import { memo } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Activity } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { de } from 'date-fns/locale'
import { useActivities } from '@/api/hooks/useActivities'
import { activityTypeIcon, activityTypeLabel } from '@/modules/crm/activities/activityUtils'
import { Skeleton } from '@/components/ui/skeleton'
import { Button } from '@/components/ui/button'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

function ActivityFeed(_props: WidgetProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { data, isLoading, error, refetch } = useActivities({
    page_size: 8,
    sort_by: 'created_at',
    sort_desc: true,
  })

  const activities = data?.activities ?? []

  if (error) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <div className="text-center">
          <p className="text-sm text-muted-foreground">{t('dashboard.error.loading')}</p>
          <Button variant="ghost" size="sm" className="mt-2" onClick={() => refetch()}>
            {t('common.retry')}
          </Button>
        </div>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="space-y-3 p-4">
        {Array.from({ length: 6 }).map((_, i) => (
          <div key={i} className="flex items-center gap-3">
            <Skeleton className="h-7 w-7 rounded" />
            <div className="flex-1 space-y-1">
              <Skeleton className="h-3.5 w-2/3" />
              <Skeleton className="h-3 w-1/3" />
            </div>
          </div>
        ))}
      </div>
    )
  }

  if (activities.length === 0) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <div className="text-center">
          <Activity className="mx-auto h-8 w-8 text-muted-foreground/50" />
          <p className="mt-2 text-sm text-muted-foreground">{t('dashboard.activityFeed.noActivities')}</p>
        </div>
      </div>
    )
  }

  return (
    <div className="divide-y">
      {activities.map((activity) => {
        const Icon = activityTypeIcon(activity.activity_type)
        const typeLabel = activityTypeLabel(activity.activity_type)
        const timeAgo = activity.created_at
          ? formatDistanceToNow(new Date(activity.created_at), { addSuffix: true, locale: de })
          : ''

        // Determine the linked entity for navigation
        const linkedName =
          activity.contact_name || activity.company_name || activity.deal_name
        const linkedPath = activity.contact_id
          ? `/crm/contacts/${activity.contact_id}`
          : activity.company_id
            ? `/crm/companies/${activity.company_id}`
            : activity.deal_id
              ? `/crm/deals/${activity.deal_id}`
              : null

        return (
          <div
            key={activity.id}
            role={linkedPath ? 'button' : undefined}
            tabIndex={linkedPath ? 0 : undefined}
            className={`flex items-start gap-3 px-4 py-2.5 transition-colors ${
              linkedPath ? 'cursor-pointer hover:bg-accent/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50 focus-visible:ring-inset' : ''
            }`}
            onClick={() => linkedPath && navigate(linkedPath)}
            onKeyDown={(e) => { if (linkedPath && (e.key === 'Enter' || e.key === ' ')) { e.preventDefault(); navigate(linkedPath) } }}
          >
            <div className="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded bg-muted">
              <Icon className="h-3.5 w-3.5 text-muted-foreground" />
            </div>
            <div className="min-w-0 flex-1">
              <p className="text-sm">
                <span className="font-medium">{activity.subject}</span>
              </p>
              <div className="mt-0.5 flex items-center gap-2 text-xs text-muted-foreground">
                <span>{typeLabel}</span>
                {linkedName && (
                  <>
                    <span>&middot;</span>
                    <span className="truncate">{linkedName}</span>
                  </>
                )}
                <span>&middot;</span>
                <span className="shrink-0">{timeAgo}</span>
              </div>
            </div>
          </div>
        )
      })}
    </div>
  )
}

export default memo(ActivityFeed)

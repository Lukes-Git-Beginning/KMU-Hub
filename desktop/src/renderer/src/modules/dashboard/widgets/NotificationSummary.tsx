/**
 * NotificationSummary widget -- shows latest 5 notifications.
 *
 * Displays unread count header, notification list with type icons and
 * relative timestamps, and a link to the full notification center.
 * Subscribes to real-time updates via the notification WebSocket hook.
 */
import { memo } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { Bell, BellOff, MessageSquare, TrendingUp, Users, Megaphone, ArrowRight } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { de } from 'date-fns/locale'
import { cn } from '@/lib/cn'
import {
  useNotifications,
  useUnreadNotificationCount,
  useMarkNotificationRead,
  type Notification,
} from '@/api/hooks/useNotifications'
import { Skeleton } from '@/components/ui/skeleton'
import { Button } from '@/components/ui/button'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

function getIcon(moduleId: string | undefined) {
  switch (moduleId) {
    case 'chat':
      return MessageSquare
    case 'crm':
      return TrendingUp
    case 'hr':
      return Users
    default:
      return Megaphone
  }
}

function NotificationSummary(_props: WidgetProps) {
  const navigate = useNavigate()
  const { data: unreadCount = 0 } = useUnreadNotificationCount()
  const { data: notificationsData, isLoading, error, refetch } = useNotifications({
    pageSize: 5,
  })
  const markRead = useMarkNotificationRead()

  const notifications = notificationsData?.notifications ?? []

  const handleClick = (notification: Notification) => {
    if (!notification.is_read && notification.id) {
      markRead.mutate(notification.id)
    }
    if (notification.deep_link) {
      navigate(notification.deep_link)
    }
  }

  if (error) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <div className="text-center">
          <p className="text-sm text-muted-foreground">Fehler beim Laden</p>
          <Button variant="ghost" size="sm" className="mt-2" onClick={() => refetch()}>
            Erneut versuchen
          </Button>
        </div>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="space-y-3 p-4">
        <Skeleton className="h-5 w-24" />
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="flex items-center gap-3">
            <Skeleton className="h-6 w-6 rounded" />
            <div className="flex-1 space-y-1">
              <Skeleton className="h-3.5 w-3/4" />
              <Skeleton className="h-3 w-1/4" />
            </div>
          </div>
        ))}
      </div>
    )
  }

  if (notifications.length === 0) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <div className="text-center">
          <BellOff className="mx-auto h-8 w-8 text-muted-foreground/50" />
          <p className="mt-2 text-sm text-muted-foreground">Keine Benachrichtigungen</p>
        </div>
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col">
      {/* Unread count header */}
      {unreadCount > 0 && (
        <div className="flex items-center gap-2 border-b px-4 py-2">
          <Bell className="h-3.5 w-3.5 text-primary" />
          <span className="text-xs font-medium text-primary">
            {unreadCount} ungelesen
          </span>
        </div>
      )}

      {/* Notification list */}
      <div className="flex-1 divide-y overflow-auto">
        {notifications.map((notification) => {
          const Icon = getIcon(notification.module_id)
          const timeAgo = notification.created_at
            ? formatDistanceToNow(new Date(notification.created_at), {
                addSuffix: true,
                locale: de,
              })
            : ''

          return (
            <div
              key={notification.id}
              className={cn(
                'flex cursor-pointer items-start gap-3 px-4 py-2.5 transition-colors hover:bg-accent/50',
                !notification.is_read && 'bg-primary/5'
              )}
              onClick={() => handleClick(notification)}
            >
              <div className="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded bg-muted">
                <Icon className="h-3 w-3 text-muted-foreground" />
              </div>
              <div className="min-w-0 flex-1">
                <p
                  className={cn(
                    'truncate text-sm',
                    !notification.is_read ? 'font-medium' : ''
                  )}
                >
                  {notification.title}
                </p>
                <span className="text-xs text-muted-foreground">{timeAgo}</span>
              </div>
              {!notification.is_read && (
                <div className="mt-2 h-2 w-2 shrink-0 rounded-full bg-primary" />
              )}
            </div>
          )
        })}
      </div>

      {/* Footer link */}
      <div className="border-t px-4 py-2">
        <Link
          to="/notifications"
          className="flex items-center gap-1 text-xs font-medium text-primary hover:underline"
        >
          Alle anzeigen
          <ArrowRight className="h-3 w-3" />
        </Link>
      </div>
    </div>
  )
}

export default memo(NotificationSummary)

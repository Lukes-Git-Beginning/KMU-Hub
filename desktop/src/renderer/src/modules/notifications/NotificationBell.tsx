/**
 * Notification bell with unread count badge and dropdown popover.
 *
 * Displays in the header bar. Clicking opens a popover with the most
 * recent 10 notifications. Each notification can be marked as read
 * by clicking it. Links to the full notification center at /notifications.
 */
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Bell, MessageSquare, TrendingUp, Users, Megaphone } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { de } from 'date-fns/locale'
import { cn } from '@/lib/cn'
import {
  useNotifications,
  useUnreadNotificationCount,
  useMarkNotificationRead,
  useMarkAllNotificationsRead,
  type Notification,
} from '@/api/hooks/useNotifications'
import { Button } from '@/components/ui/button'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

export function NotificationBell() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { data: unreadCount = 0 } = useUnreadNotificationCount()
  const { data: notificationsData } = useNotifications({ pageSize: 10 })
  const markRead = useMarkNotificationRead()
  const markAllRead = useMarkAllNotificationsRead()

  const notifications = notificationsData?.notifications ?? []

  const handleNotificationClick = (notification: Notification) => {
    // Mark as read
    if (!notification.is_read && notification.id) {
      markRead.mutate(notification.id)
    }

    // Navigate to relevant page via deep_link
    if (notification.deep_link) {
      navigate(notification.deep_link)
    }
  }

  return (
    <Popover>
      <Tooltip>
        <TooltipTrigger asChild>
          <PopoverTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="relative h-9 w-9"
              aria-label={unreadCount > 0 ? t('notifications.bell.ariaLabelUnread', { count: unreadCount }) : t('notifications.bell.ariaLabel')}
            >
              <Bell className="h-4 w-4" aria-hidden="true" />
              {unreadCount > 0 && (
                <span
                  className="absolute -right-0.5 -top-0.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-destructive px-1 text-[10px] font-medium text-destructive-foreground"
                  aria-hidden="true"
                >
                  {unreadCount > 99 ? '99+' : unreadCount}
                </span>
              )}
            </Button>
          </PopoverTrigger>
        </TooltipTrigger>
        <TooltipContent>{t('notifications.bell.tooltip')}</TooltipContent>
      </Tooltip>

      <PopoverContent className="w-80 p-0" align="end">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3">
          <h3 className="text-sm font-semibold text-foreground">{t('notifications.bell.title')}</h3>
          {unreadCount > 0 && (
            <button
              className="text-xs text-primary hover:underline"
              onClick={() => markAllRead.mutate(undefined)}
              disabled={markAllRead.isPending}
            >
              {t('notifications.bell.markAllRead')}
            </button>
          )}
        </div>

        <Separator />

        {/* Notification list */}
        <ScrollArea className="max-h-[360px]">
          {notifications.length === 0 ? (
            <div className="flex items-center justify-center py-8">
              <p className="text-sm text-muted-foreground">{t('notifications.bell.empty')}</p>
            </div>
          ) : (
            <div>
              {notifications.map((notification) => (
                <NotificationItem
                  key={notification.id}
                  notification={notification}
                  onClick={() => handleNotificationClick(notification)}
                />
              ))}
            </div>
          )}
        </ScrollArea>

        <Separator />

        {/* Footer */}
        <div className="px-4 py-2">
          <button
            className="w-full text-center text-xs text-primary hover:underline"
            onClick={() => navigate('/notifications')}
          >
            {t('notifications.bell.showAll')}
          </button>
        </div>
      </PopoverContent>
    </Popover>
  )
}

function NotificationItem({
  notification,
  onClick,
}: {
  notification: Notification
  onClick: () => void
}) {
  const timeAgo = notification.created_at
    ? formatDistanceToNow(new Date(notification.created_at), { addSuffix: true, locale: de })
    : ''

  const Icon = getNotificationIcon(notification.module_id ?? '')

  return (
    <button
      className={cn(
        'flex w-full items-start gap-3 px-4 py-3 text-left transition-colors hover:bg-accent',
        !notification.is_read && 'bg-primary-light/50'
      )}
      onClick={onClick}
    >
      <div className="mt-0.5 shrink-0">
        {/* eslint-disable-next-line react-hooks/static-components -- Icon is a dynamic component variable */}
        <Icon className="h-4 w-4 text-muted-foreground" />
      </div>

      <div className="min-w-0 flex-1">
        <p className={cn(
          'text-sm truncate',
          !notification.is_read ? 'font-medium text-foreground' : 'text-foreground'
        )}>
          {notification.title}
        </p>
        {notification.body && (
          <p className="mt-0.5 text-xs text-muted-foreground line-clamp-2">
            {notification.body}
          </p>
        )}
        <p className="mt-1 text-xs text-muted-foreground">{timeAgo}</p>
      </div>

      {/* Unread dot */}
      {!notification.is_read && (
        <div className="mt-2 h-2 w-2 shrink-0 rounded-full bg-primary" />
      )}
    </button>
  )
}

function getNotificationIcon(moduleId: string) {
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

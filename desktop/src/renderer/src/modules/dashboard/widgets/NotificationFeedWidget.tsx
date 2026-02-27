/**
 * NotificationFeedWidget — compact list of the last 10 notifications
 * from the Zustand notification store.
 *
 * Works offline (no API dependency). Click marks as read and navigates.
 */
import { memo } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import {
  Bell,
  BellOff,
  MessageSquare,
  Mail,
  CheckSquare,
  Clock,
  Video,
  LifeBuoy,
  FileWarning,
  AlertCircle,
  AtSign,
  Download,
  ArrowRight,
} from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { de } from 'date-fns/locale'
import { cn } from '@/lib/cn'
import { useNotificationsStore } from '@/stores/notifications'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

const iconMap: Record<string, React.ElementType> = {
  MessageSquare,
  Mail,
  CheckSquare,
  Clock,
  Video,
  LifeBuoy,
  FileWarning,
  AlertCircle,
  AtSign,
  Download,
  Bell,
}

function NotificationFeedWidget(_props: WidgetProps) {
  const navigate = useNavigate()
  const { notifications, markAsRead } = useNotificationsStore()
  const unreadCount = notifications.filter((n) => !n.isRead).length
  const displayedNotifications = notifications.slice(0, 10)

  const handleClick = (n: (typeof notifications)[0]) => {
    if (!n.isRead) markAsRead(n.id)
    if (n.actionUrl) navigate(n.actionUrl)
  }

  if (displayedNotifications.length === 0) {
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
        <div className="flex items-center gap-2 border-b border-border/60 px-4 py-2">
          <Bell className="h-3.5 w-3.5 text-primary" />
          <span className="text-xs font-medium text-primary">{unreadCount} ungelesen</span>
        </div>
      )}

      {/* List */}
      <div className="flex-1 divide-y divide-border/40 overflow-auto">
        {displayedNotifications.map((n) => {
          const Icon = iconMap[n.icon] || Bell
          let timeAgo = ''
          try {
            timeAgo = formatDistanceToNow(new Date(n.createdAt), { addSuffix: true, locale: de })
          } catch {
            timeAgo = ''
          }

          return (
            <div
              key={n.id}
              onClick={() => handleClick(n)}
              className={cn(
                'flex cursor-pointer items-start gap-3 px-4 py-2.5 transition-colors hover:bg-accent/50',
                !n.isRead && 'bg-primary/5',
              )}
            >
              {n.actorInitials ? (
                <div className="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary/10 text-[10px] font-medium text-primary">
                  {n.actorInitials}
                </div>
              ) : (
                <div className="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded bg-muted">
                  <Icon className="h-3 w-3 text-muted-foreground" />
                </div>
              )}
              <div className="min-w-0 flex-1">
                <p className={cn('truncate text-sm', !n.isRead && 'font-medium')}>
                  {n.title}
                </p>
                <p className="truncate text-xs text-muted-foreground">{n.body}</p>
                <span className="text-[10px] text-muted-foreground/70">{timeAgo}</span>
              </div>
              {!n.isRead && (
                <div className="mt-2 h-2 w-2 shrink-0 rounded-full bg-primary" />
              )}
            </div>
          )
        })}
      </div>

      {/* Footer */}
      <div className="border-t border-border/60 px-4 py-2">
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

export default memo(NotificationFeedWidget)

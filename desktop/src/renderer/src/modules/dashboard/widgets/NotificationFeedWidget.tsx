/**
 * NotificationFeedWidget — compact list of the latest 10 notifications.
 *
 * Reads the MSW/React-Query pipeline (same source as the center + bell), so the
 * widget, toast and badges never disagree. Click marks as read and navigates.
 */
import { memo, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, Link } from 'react-router-dom'
import {
  Bell,
  BellOff,
  MessageSquare,
  CheckSquare,
  TrendingUp,
  Users,
  FileText,
  Receipt,
  FileSignature,
  FolderKanban,
  ShieldAlert,
  PhoneCall,
  Megaphone,
  ArrowRight,
} from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { de } from 'date-fns/locale'
import { cn } from '@/lib/cn'
import {
  useNotifications,
  useUnreadNotificationCount,
  useMarkNotificationRead,
  type Notification,
} from '@/api/hooks/useNotifications'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

type FeedNotification = Notification & { actor_name?: string }

const moduleIcon: Record<string, React.ElementType> = {
  chat: MessageSquare,
  tasks: CheckSquare,
  contacts: TrendingUp,
  team: Users,
  documents: FileText,
  finance: Receipt,
  vertraege: FileSignature,
  projects: FolderKanban,
  settings: ShieldAlert,
  dialer: PhoneCall,
}

function iconFor(moduleId?: string): React.ElementType {
  return (moduleId && moduleIcon[moduleId]) || Megaphone
}

function initialsOf(name?: string): string {
  if (!name) return ''
  return (
    name
      .split(/\s+/)
      .filter(Boolean)
      .slice(0, 2)
      .map((w) => w[0]?.toUpperCase() ?? '')
      .join('') || ''
  )
}

function NotificationFeedWidget(_props: WidgetProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { data } = useNotifications({ pageSize: 10 })
  const { data: unreadCount = 0 } = useUnreadNotificationCount()
  const markRead = useMarkNotificationRead()

  const displayed = useMemo(() => (data?.notifications ?? []) as FeedNotification[], [data])

  const handleClick = (n: FeedNotification) => {
    if (!n.is_read && n.id) markRead.mutate(n.id)
    const target = n.deep_link || (n.module_id ? `/${n.module_id}` : null)
    if (target) navigate(target)
  }

  if (displayed.length === 0) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <div className="text-center">
          <BellOff className="mx-auto h-8 w-8 text-muted-foreground/50" />
          <p className="mt-2 text-sm text-muted-foreground">{t('dashboard.notifications.none')}</p>
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
          <span className="text-xs font-medium text-primary">
            {t('dashboard.notifications.unread', { count: unreadCount })}
          </span>
        </div>
      )}

      {/* List */}
      <div className="flex-1 divide-y divide-border/40 overflow-auto">
        {displayed.map((n) => {
          const Icon = iconFor(n.module_id)
          const initials = n.actor_id ? initialsOf(n.actor_name) : ''
          let timeAgo = ''
          try {
            timeAgo = n.created_at
              ? formatDistanceToNow(new Date(n.created_at), { addSuffix: true, locale: de })
              : ''
          } catch {
            timeAgo = ''
          }

          return (
            <div
              key={n.id}
              onClick={() => handleClick(n)}
              className={cn(
                'flex cursor-pointer items-start gap-3 px-4 py-2.5 transition-colors hover:bg-accent/50',
                !n.is_read && 'bg-primary/5'
              )}
            >
              {initials ? (
                <div className="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary/10 text-[10px] font-medium text-primary">
                  {initials}
                </div>
              ) : (
                <div className="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded bg-muted">
                  {/* eslint-disable-next-line react-hooks/static-components -- Icon is a dynamic component variable */}
                  <Icon className="h-3 w-3 text-muted-foreground" />
                </div>
              )}
              <div className="min-w-0 flex-1">
                <p className={cn('truncate text-sm', !n.is_read && 'font-medium')}>{n.title}</p>
                {n.body && <p className="truncate text-xs text-muted-foreground">{n.body}</p>}
                <span className="text-[10px] text-muted-foreground/70">{timeAgo}</span>
              </div>
              {!n.is_read && <div className="mt-2 h-2 w-2 shrink-0 rounded-full bg-primary" />}
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
          {t('dashboard.notifications.showAll')}
          <ArrowRight className="h-3 w-3" />
        </Link>
      </div>
    </div>
  )
}

export default memo(NotificationFeedWidget)

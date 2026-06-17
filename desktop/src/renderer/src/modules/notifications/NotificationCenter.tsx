/**
 * Full notification center page at /notifications.
 *
 * Displays all notifications with filter tabs (All, Unread),
 * batch mark-all-as-read action, and links to notification preferences.
 */
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { BellOff, Check, MessageSquare, TrendingUp, Users, Megaphone, PhoneCall, Pin, PinOff, EyeOff, ArrowRight } from 'lucide-react'
import { formatDistanceToNow, format } from 'date-fns'
import { de } from 'date-fns/locale'
import { cn } from '@/lib/cn'
import {
  useNotifications,
  useUnreadNotificationCount,
  useMarkNotificationRead,
  useMarkAllNotificationsRead,
  useNotificationPreferences,
  useUpdateNotificationPreference,
  useEventTypes,
  type Notification,
  type NotificationPreference,
} from '@/api/hooks/useNotifications'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'

export default function NotificationCenter() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState<'all' | 'unread'>('all')
  const [page, setPage] = useState(1)
  const [showPreferences, setShowPreferences] = useState(false)

  const { data: unreadCount = 0 } = useUnreadNotificationCount()
  const { data: notificationsData, isLoading } = useNotifications({
    page,
    pageSize: 20,
    isRead: activeTab === 'unread' ? false : undefined,
  })
  const markRead = useMarkNotificationRead()
  const markAllRead = useMarkAllNotificationsRead()

  // Local interaction state: expanded card + pinned/dismissed sets.
  // Demo-stateful within the session (pin/dismiss are client-only until a backend exists).
  const [expandedId, setExpandedId] = useState<string | null>(null)
  const [pinnedIds, setPinnedIds] = useState<Set<string>>(new Set())
  const [dismissedIds, setDismissedIds] = useState<Set<string>>(new Set())

  const rawNotifications = notificationsData?.notifications ?? []
  const total = notificationsData?.total ?? 0
  const hasMore = rawNotifications.length === 20

  // Dismissed hidden; pinned floated to the top.
  const notifications = rawNotifications
    .filter((n) => !dismissedIds.has(n.id ?? ''))
    .sort((a, b) => (pinnedIds.has(b.id ?? '') ? 1 : 0) - (pinnedIds.has(a.id ?? '') ? 1 : 0))

  const togglePin = (id: string) =>
    setPinnedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })

  const dismiss = (id: string) => {
    setDismissedIds((prev) => new Set(prev).add(id))
    setExpandedId((cur) => (cur === id ? null : cur))
  }

  // "Dorthin springen": mark read + navigate to the notification's target.
  const openNotification = (notification: Notification) => {
    if (!notification.is_read && notification.id) markRead.mutate(notification.id)
    const target = notification.deep_link || (notification.module_id ? `/${notification.module_id}` : null)
    if (target) navigate(target)
  }

  return (
    <div className="h-full overflow-auto">
      <div className="mx-auto max-w-3xl px-6 py-6">
        {/* Page header */}
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-2xl font-semibold text-foreground">{t('notifications.center.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">
              {unreadCount > 0
                ? t('notifications.center.unread', { count: unreadCount })
                : t('notifications.center.allRead')}
            </p>
          </div>

          <div className="flex items-center gap-2">
            {unreadCount > 0 && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => markAllRead.mutate(undefined)}
                disabled={markAllRead.isPending}
              >
                <Check className="mr-2 h-4 w-4" />
                {t('notifications.center.markAllRead')}
              </Button>
            )}
            <Button
              variant={showPreferences ? 'secondary' : 'outline'}
              size="sm"
              onClick={() => setShowPreferences(!showPreferences)}
            >
              {t('notifications.center.settings')}
            </Button>
          </div>
        </div>

        {/* Preferences panel */}
        {showPreferences && (
          <div className="mb-6">
            <PreferencesPanel />
          </div>
        )}

        {/* Filter tabs */}
        <Tabs value={activeTab} onValueChange={(v) => { setActiveTab(v as 'all' | 'unread'); setPage(1) }}>
          <TabsList>
            <TabsTrigger value="all">{t('notifications.center.tabAll')}</TabsTrigger>
            <TabsTrigger value="unread">
              {t('notifications.center.tabUnread')}
              {unreadCount > 0 && (
                <Badge variant="destructive" className="ml-2 h-5 min-w-5 px-1.5 text-xs">
                  {unreadCount}
                </Badge>
              )}
            </TabsTrigger>
          </TabsList>

          <TabsContent value={activeTab} className="mt-4">
            {isLoading ? (
              <div className="flex items-center justify-center py-12">
                <div className="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
              </div>
            ) : notifications.length === 0 ? (
              <EmptyState isUnreadFilter={activeTab === 'unread'} />
            ) : (
              <div className="space-y-2">
                {notifications.map((notification) => (
                  <NotificationCard
                    key={notification.id}
                    notification={notification}
                    expanded={expandedId === notification.id}
                    pinned={pinnedIds.has(notification.id ?? '')}
                    onToggleExpand={() =>
                      setExpandedId((cur) => (cur === notification.id ? null : notification.id ?? null))
                    }
                    onOpen={() => openNotification(notification)}
                    onPin={() => notification.id && togglePin(notification.id)}
                    onDismiss={() => notification.id && dismiss(notification.id)}
                    onMarkRead={() => {
                      if (notification.id) markRead.mutate(notification.id)
                    }}
                  />
                ))}

                {/* Pagination */}
                <div className="flex items-center justify-between pt-4">
                  <p className="text-sm text-muted-foreground">
                    {t('notifications.center.page', { page, total })}
                  </p>
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={page <= 1}
                      onClick={() => setPage(page - 1)}
                    >
                      {t('notifications.center.back')}
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={!hasMore}
                      onClick={() => setPage(page + 1)}
                    >
                      {t('notifications.center.forward')}
                    </Button>
                  </div>
                </div>
              </div>
            )}
          </TabsContent>
        </Tabs>
      </div>
    </div>
  )
}

function NotificationCard({
  notification,
  expanded,
  pinned,
  onToggleExpand,
  onOpen,
  onPin,
  onDismiss,
  onMarkRead,
}: {
  notification: Notification
  expanded: boolean
  pinned: boolean
  onToggleExpand: () => void
  onOpen: () => void
  onPin: () => void
  onDismiss: () => void
  onMarkRead: () => void
}) {
  const { t } = useTranslation()
  const timeAgo = notification.created_at
    ? formatDistanceToNow(new Date(notification.created_at), { addSuffix: true, locale: de })
    : ''

  const fullDate = notification.created_at
    ? format(new Date(notification.created_at), 'dd.MM.yyyy HH:mm', { locale: de })
    : ''

  const Icon = getNotificationIcon(notification.module_id ?? '')
  const priorityColor = getPriorityColor(notification.priority)

  return (
    <Card
      className={cn(
        'cursor-pointer transition-colors hover:bg-accent/50',
        !notification.is_read && 'border-l-4 border-l-primary',
        pinned && 'ring-1 ring-primary/40'
      )}
      onClick={onToggleExpand}
    >
      <CardContent className="flex items-start gap-4 p-4">
        <div className={cn('mt-0.5 shrink-0 rounded-md p-2', priorityColor)}>
          {/* eslint-disable-next-line react-hooks/static-components -- Icon is a dynamic component variable */}
          <Icon className="h-4 w-4" />
        </div>

        <div className="min-w-0 flex-1">
          <div className="flex items-start justify-between gap-2">
            <div className="min-w-0">
              <div className="flex items-center gap-1.5">
                {pinned && (
                  <Pin className="h-3 w-3 shrink-0 text-primary" aria-label={t('notifications.actions.pinned')} />
                )}
                <p className={cn(
                  'text-sm',
                  !notification.is_read ? 'font-semibold text-foreground' : 'text-foreground'
                )}>
                  {notification.title}
                </p>
              </div>
              {notification.body && (
                <p className="mt-1 text-sm text-muted-foreground">
                  {notification.body}
                </p>
              )}
            </div>

            {!notification.is_read && (
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7 shrink-0"
                onClick={(e) => {
                  e.stopPropagation()
                  onMarkRead()
                }}
                aria-label={t('notifications.center.markAllRead')}
              >
                <Check className="h-3.5 w-3.5" />
              </Button>
            )}
          </div>

          <div className="mt-2 flex items-center gap-3">
            <span className="text-xs text-muted-foreground" title={fullDate}>
              {timeAgo}
            </span>
            {notification.module_id && (
              <Badge variant="secondary" className="text-xs">
                {notification.module_id}
              </Badge>
            )}
            {notification.group_count && notification.group_count > 1 && (
              <Badge variant="outline" className="text-xs">
                {t('notifications.center.more', { count: notification.group_count - 1 })}
              </Badge>
            )}
          </div>

          {/* Expanded action bar (Darien: open / pin / dismiss) */}
          {expanded && (
            <div
              className="mt-3 flex flex-wrap items-center gap-2 border-t border-border pt-3"
              onClick={(e) => e.stopPropagation()}
            >
              <Button variant="default" size="sm" className="h-8" onClick={onOpen}>
                <ArrowRight className="mr-1.5 h-3.5 w-3.5" />
                {t('notifications.actions.open')}
              </Button>
              <Button variant="outline" size="sm" className="h-8" onClick={onPin}>
                {pinned ? <PinOff className="mr-1.5 h-3.5 w-3.5" /> : <Pin className="mr-1.5 h-3.5 w-3.5" />}
                {pinned ? t('notifications.actions.unpin') : t('notifications.actions.pin')}
              </Button>
              <Button variant="ghost" size="sm" className="h-8 text-muted-foreground" onClick={onDismiss}>
                <EyeOff className="mr-1.5 h-3.5 w-3.5" />
                {t('notifications.actions.dismiss')}
              </Button>
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

function PreferencesPanel() {
  const { t } = useTranslation()
  const { data: eventTypes = [] } = useEventTypes()
  const { data: preferences = [] } = useNotificationPreferences()
  const updatePreference = useUpdateNotificationPreference()

  // Group event types by module
  const moduleGroups = eventTypes.reduce<Record<string, typeof eventTypes>>((acc, et) => {
    const mod = et.module_id ?? 'other'
    if (!acc[mod]) acc[mod] = []
    acc[mod].push(et)
    return acc
  }, {})

  const getPreference = (eventKey: string): NotificationPreference | undefined => {
    return preferences.find((p) => p.event_type_key === eventKey)
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg">{t('notifications.preferences.title')}</CardTitle>
        <CardDescription>
          {t('notifications.preferences.description')}
        </CardDescription>
      </CardHeader>
      <CardContent>
        {Object.keys(moduleGroups).length === 0 ? (
          <p className="text-sm text-muted-foreground">{t('notifications.preferences.noEventTypes')}</p>
        ) : (
          <div className="space-y-6">
            {Object.entries(moduleGroups).map(([moduleId, types]) => (
              <div key={moduleId}>
                <h4 className="text-sm font-semibold text-foreground capitalize mb-3">
                  {moduleId}
                </h4>
                <div className="space-y-3">
                  {types.map((eventType) => {
                    const pref = getPreference(eventType.event_key ?? '')
                    const inApp = pref?.in_app ?? eventType.default_in_app ?? true
                    const desktopPush = pref?.desktop_push ?? eventType.default_desktop_push ?? true

                    return (
                      <div
                        key={eventType.id}
                        className="flex items-center justify-between rounded-md border border-border px-3 py-2"
                      >
                        <div>
                          <p className="text-sm font-medium">{eventType.display_name}</p>
                          {eventType.description && (
                            <p className="text-xs text-muted-foreground">{eventType.description}</p>
                          )}
                        </div>
                        <div className="flex items-center gap-4">
                          <label className="flex items-center gap-2 text-xs">
                            <input
                              type="checkbox"
                              checked={inApp}
                              onChange={(e) => {
                                updatePreference.mutate({
                                  event_type_key: eventType.event_key,
                                  in_app: e.target.checked,
                                  desktop_push: desktopPush,
                                })
                              }}
                              className="h-3.5 w-3.5 rounded border-border"
                            />
                            {t('notifications.preferences.inApp')}
                          </label>
                          <label className="flex items-center gap-2 text-xs">
                            <input
                              type="checkbox"
                              checked={desktopPush}
                              onChange={(e) => {
                                updatePreference.mutate({
                                  event_type_key: eventType.event_key,
                                  in_app: inApp,
                                  desktop_push: e.target.checked,
                                })
                              }}
                              className="h-3.5 w-3.5 rounded border-border"
                            />
                            {t('notifications.preferences.desktop')}
                          </label>
                        </div>
                      </div>
                    )
                  })}
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function EmptyState({ isUnreadFilter }: { isUnreadFilter: boolean }) {
  const { t } = useTranslation()
  return (
    <div className="flex flex-col items-center justify-center py-16">
      <BellOff className="h-12 w-12 text-muted-foreground/50" />
      <h3 className="mt-4 text-lg font-semibold text-foreground">
        {isUnreadFilter ? t('notifications.center.emptyUnread') : t('notifications.center.emptyAll')}
      </h3>
      <p className="mt-1 text-sm text-muted-foreground">
        {isUnreadFilter
          ? t('notifications.center.emptyUnreadDescription')
          : t('notifications.center.emptyAllDescription')}
      </p>
    </div>
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
    case 'dialer':
      return PhoneCall
    default:
      return Megaphone
  }
}

function getPriorityColor(priority: string | undefined): string {
  switch (priority) {
    case 'urgent':
      return 'bg-error-light text-destructive'
    case 'high':
      return 'bg-orange-100 text-orange-600 dark:bg-orange-950/50 dark:text-orange-400'
    case 'normal':
      return 'bg-blue-100 text-blue-600 dark:bg-blue-950/50 dark:text-blue-400'
    case 'low':
      return 'bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-400'
    default:
      return 'bg-muted text-muted-foreground'
  }
}

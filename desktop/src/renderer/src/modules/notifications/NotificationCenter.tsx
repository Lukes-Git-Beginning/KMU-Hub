/**
 * Full notification center page at /notifications.
 *
 * Displays all notifications with filter tabs (All, Unread),
 * batch mark-all-as-read action, and links to notification preferences.
 */
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { BellOff, Check, MessageSquare, TrendingUp, Users, Megaphone } from 'lucide-react'
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

  const notifications = notificationsData?.notifications ?? []
  const total = notificationsData?.total ?? 0
  const hasMore = notifications.length === 20

  const handleNotificationClick = (notification: Notification) => {
    if (!notification.is_read && notification.id) {
      markRead.mutate(notification.id)
    }
    if (notification.deep_link) {
      navigate(notification.deep_link)
    }
  }

  return (
    <div className="h-full overflow-auto">
      <div className="mx-auto max-w-3xl px-6 py-6">
        {/* Page header */}
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-2xl font-semibold text-foreground">Benachrichtigungen</h1>
            <p className="mt-1 text-sm text-muted-foreground">
              {unreadCount > 0
                ? `${unreadCount} ungelesene Benachrichtigung${unreadCount !== 1 ? 'en' : ''}`
                : 'Alle gelesen'}
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
                Alle als gelesen markieren
              </Button>
            )}
            <Button
              variant={showPreferences ? 'secondary' : 'outline'}
              size="sm"
              onClick={() => setShowPreferences(!showPreferences)}
            >
              Einstellungen
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
            <TabsTrigger value="all">Alle</TabsTrigger>
            <TabsTrigger value="unread">
              Ungelesen
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
                    onClick={() => handleNotificationClick(notification)}
                    onMarkRead={() => {
                      if (notification.id) markRead.mutate(notification.id)
                    }}
                  />
                ))}

                {/* Pagination */}
                <div className="flex items-center justify-between pt-4">
                  <p className="text-sm text-muted-foreground">
                    Seite {page} ({total} gesamt)
                  </p>
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={page <= 1}
                      onClick={() => setPage(page - 1)}
                    >
                      Zurueck
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={!hasMore}
                      onClick={() => setPage(page + 1)}
                    >
                      Weiter
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
  onClick,
  onMarkRead,
}: {
  notification: Notification
  onClick: () => void
  onMarkRead: () => void
}) {
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
        !notification.is_read && 'border-l-4 border-l-primary'
      )}
      onClick={onClick}
    >
      <CardContent className="flex items-start gap-4 p-4">
        <div className={cn('mt-0.5 shrink-0 rounded-md p-2', priorityColor)}>
          {/* eslint-disable-next-line react-hooks/static-components -- Icon is a dynamic component variable */}
          <Icon className="h-4 w-4" />
        </div>

        <div className="min-w-0 flex-1">
          <div className="flex items-start justify-between gap-2">
            <div>
              <p className={cn(
                'text-sm',
                !notification.is_read ? 'font-semibold text-foreground' : 'text-foreground'
              )}>
                {notification.title}
              </p>
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
                +{notification.group_count - 1} weitere
              </Badge>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

function PreferencesPanel() {
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
        <CardTitle className="text-lg">Benachrichtigungseinstellungen</CardTitle>
        <CardDescription>
          Konfiguriere, welche Benachrichtigungen du erhalten möchtest.
        </CardDescription>
      </CardHeader>
      <CardContent>
        {Object.keys(moduleGroups).length === 0 ? (
          <p className="text-sm text-muted-foreground">Keine Ereignistypen konfiguriert.</p>
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
                            In-App
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
                            Desktop
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
  return (
    <div className="flex flex-col items-center justify-center py-16">
      <BellOff className="h-12 w-12 text-muted-foreground/50" />
      <h3 className="mt-4 text-lg font-semibold text-foreground">
        {isUnreadFilter ? 'Alle gelesen' : 'Keine Benachrichtigungen'}
      </h3>
      <p className="mt-1 text-sm text-muted-foreground">
        {isUnreadFilter
          ? 'Du hast alle Benachrichtigungen gelesen.'
          : 'Du hast noch keine Benachrichtigungen erhalten.'}
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
    default:
      return Megaphone
  }
}

function getPriorityColor(priority: string | undefined): string {
  switch (priority) {
    case 'urgent':
      return 'bg-red-100 text-red-600 dark:bg-red-950/50 dark:text-red-400'
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

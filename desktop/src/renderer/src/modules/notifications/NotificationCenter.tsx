/**
 * Full notification center page at /notifications.
 *
 * Displays all notifications with filter tabs (All, Unread), a per-module
 * filter + sort control, batch mark-all-as-read action, and links to
 * notification preferences. A row click opens the shared DetailModal with the
 * full notification (actor, module, priority, body) and its actions.
 */
import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import { Bell, BellOff, Check, MessageSquare, TrendingUp, Users, Megaphone, PhoneCall, Pin, PinOff, EyeOff, ArrowRight, AlarmClock } from 'lucide-react'
import { formatDistanceToNow, format } from 'date-fns'
import { de } from 'date-fns/locale'
import { toast } from 'sonner'
import { cn } from '@/lib/cn'
import {
  useNotifications,
  useUnreadNotificationCount,
  useMarkNotificationRead,
  useMarkAllNotificationsRead,
  useNotificationPreferences,
  useUpdateNotificationPreference,
  useEventTypes,
  useToggleNotificationPin,
  useDismissNotification,
  useSnoozeNotification,
  type Notification,
  type NotificationWithPin,
  type NotificationPreference,
} from '@/api/hooks/useNotifications'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { SortMenu, type SortDirection } from '@/components/shared/SortMenu'
import { DetailModal } from '@/components/shared/DetailModal'
import { EmptyState } from '@/components/shared/EmptyState'
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
} from '@/components/ui/dropdown-menu'
import { navItems, moduleHsl } from '@/components/layout/sidebar/nav-items'
import { useNotificationsStore } from '@/stores/notifications'

// Priority ordering for the sort control (highest first when descending).
const PRIORITY_RANK: Record<string, number> = { urgent: 3, high: 2, normal: 1, low: 0 }
// Priority filter order (highest first) + dot colour for the filter chips.
const PRIORITY_ORDER = ['urgent', 'high', 'normal', 'low'] as const
const PRIORITY_DOT: Record<string, string> = {
  urgent: '#dc2626',
  high: '#ea580c',
  normal: '#2563eb',
  low: '#64748b',
}

/** Compute the ISO timestamp for a snooze preset. */
function snoozeUntil(preset: '30min' | '1h' | 'tomorrow'): string {
  const d = new Date()
  if (preset === '30min') d.setMinutes(d.getMinutes() + 30)
  else if (preset === '1h') d.setHours(d.getHours() + 1)
  else {
    d.setDate(d.getDate() + 1)
    d.setHours(9, 0, 0, 0)
  }
  return d.toISOString()
}

// Module ids whose readable label differs from the matching nav item label.
const MODULE_LABEL_OVERRIDE: Record<string, string> = {
  settings: 'notifications.modules.security',
  system: 'notifications.modules.system',
}

// Alias for the extended notification type with server-backed pin/dismiss/actor fields.
type NotificationWithActor = NotificationWithPin

/** Readable label for a notification's module_id, reusing the nav-item labels. */
function moduleLabelOf(t: TFunction, moduleId?: string): string {
  if (!moduleId) return ''
  const override = MODULE_LABEL_OVERRIDE[moduleId]
  if (override) return t(override) as string
  const item = navItems.find((i) => i.id === moduleId)
  return item ? (t(item.label) as string) : moduleId
}

/** Up to two uppercase initials from a display name. */
function getInitials(name: string): string {
  const parts = name.split(/\s+/).filter(Boolean).slice(0, 2)
  return parts.map((w) => w[0]?.toUpperCase() ?? '').join('') || '?'
}

export default function NotificationCenter() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const muteAll = useNotificationsStore((s) => s.muteAll)
  const toggleMuteAll = useNotificationsStore((s) => s.toggleMuteAll)
  const [activeTab, setActiveTab] = useState<'all' | 'unread'>('all')
  const [page, setPage] = useState(1)
  const [showPreferences, setShowPreferences] = useState(false)

  // Module + priority filters + sort (client-side, on the loaded list).
  const [moduleFilter, setModuleFilter] = useState<string | null>(null)
  const [priorityFilter, setPriorityFilter] = useState<string | null>(null)
  const [sortField, setSortField] = useState<'date' | 'priority'>('date')
  const [sortDir, setSortDir] = useState<SortDirection>('desc')

  const { data: unreadCount = 0 } = useUnreadNotificationCount()
  const { data: notificationsData, isLoading } = useNotifications({
    page,
    pageSize: 20,
    isRead: activeTab === 'unread' ? false : undefined,
  })
  const markRead = useMarkNotificationRead()
  const markAllRead = useMarkAllNotificationsRead()
  // Pin + dismiss are persisted via MSW (POST /:id/pin, /:id/dismiss). The list
  // query is invalidated on success, so pinned/dismissed survive a refetch.
  const togglePinMutation = useToggleNotificationPin()
  const dismissMutation = useDismissNotification()
  const snoozeMutation = useSnoozeNotification()

  // Only local interaction state: which row's detail modal is open.
  const [detailId, setDetailId] = useState<string | null>(null)

  // Dismissed rows are already filtered server-side; cast to read is_pinned.
  const visibleBeforeFilter = useMemo(
    () => (notificationsData?.notifications ?? []) as NotificationWithActor[],
    [notificationsData]
  )
  const rawNotifications = visibleBeforeFilter
  const total = notificationsData?.total ?? 0

  // Per-module counts + the modules actually present, ordered like the sidebar.
  const moduleCounts = useMemo(() => {
    const counts: Record<string, number> = {}
    for (const n of visibleBeforeFilter) {
      if (n.module_id) counts[n.module_id] = (counts[n.module_id] ?? 0) + 1
    }
    return counts
  }, [visibleBeforeFilter])

  const availableModules = useMemo(() => {
    const order = navItems.map((i) => i.id)
    return Object.keys(moduleCounts).sort((a, b) => order.indexOf(a) - order.indexOf(b))
  }, [moduleCounts])

  // Per-priority counts + the priorities actually present (highest first).
  const priorityCounts = useMemo(() => {
    const counts: Record<string, number> = {}
    for (const n of visibleBeforeFilter) {
      const p = n.priority ?? 'normal'
      counts[p] = (counts[p] ?? 0) + 1
    }
    return counts
  }, [visibleBeforeFilter])

  const availablePriorities = useMemo(
    () => PRIORITY_ORDER.filter((p) => priorityCounts[p]),
    [priorityCounts],
  )

  // Dismissed hidden; module filter applied; sorted by field; pinned floated to top.
  const notifications = useMemo(() => {
    return visibleBeforeFilter
      .filter((n) => !moduleFilter || n.module_id === moduleFilter)
      .filter((n) => !priorityFilter || (n.priority ?? 'normal') === priorityFilter)
      .slice()
      .sort((a, b) => {
        const cmp =
          sortField === 'priority'
            ? (PRIORITY_RANK[a.priority ?? 'low'] ?? 0) - (PRIORITY_RANK[b.priority ?? 'low'] ?? 0)
            : new Date(a.created_at ?? 0).getTime() - new Date(b.created_at ?? 0).getTime()
        return sortDir === 'asc' ? cmp : -cmp
      })
      .sort((a, b) => (b.is_pinned ? 1 : 0) - (a.is_pinned ? 1 : 0))
  }, [visibleBeforeFilter, moduleFilter, priorityFilter, sortField, sortDir])

  const hasMore = !moduleFilter && !priorityFilter && rawNotifications.length === 20
  const displayTotal = moduleFilter || priorityFilter ? notifications.length : total
  const detailNotification = detailId ? rawNotifications.find((n) => n.id === detailId) ?? null : null

  const sortOptions = useMemo(
    () => [
      { value: 'date', label: t('notifications.sort.date') },
      { value: 'priority', label: t('notifications.sort.priority') },
    ],
    [t]
  )

  const togglePin = (id: string, isPinned: boolean) => togglePinMutation.mutate({ id, isPinned })

  const dismiss = (id: string) => {
    dismissMutation.mutate(id)
    setDetailId((cur) => (cur === id ? null : cur))
  }

  const snooze = (id: string, until: string) => {
    snoozeMutation.mutate({ id, until })
    setDetailId((cur) => (cur === id ? null : cur))
    toast.success(t('notifications.actions.snoozed'))
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
            {/* Quiet now, without a trip into the settings — the state is visible
                on the button itself, so nobody wonders why it stays silent. */}
            <Button
              variant={muteAll ? 'secondary' : 'outline'}
              size="sm"
              onClick={toggleMuteAll}
              aria-pressed={muteAll}
              title={t('settings.notifications.mute.desc')}
            >
              {muteAll ? <BellOff className="mr-2 h-4 w-4" /> : <Bell className="mr-2 h-4 w-4" />}
              {muteAll ? t('settings.notifications.mute.active') : t('settings.notifications.mute.label')}
            </Button>
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
        <Tabs
          value={activeTab}
          onValueChange={(v) => {
            setActiveTab(v as 'all' | 'unread')
            setPage(1)
            setModuleFilter(null)
            setPriorityFilter(null)
          }}
        >
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
            {/* Module filter chips + sort control */}
            {!isLoading && visibleBeforeFilter.length > 0 && (
              <div className="mb-4 space-y-2">
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div className="flex flex-wrap items-center gap-1.5">
                    <FilterChip
                      label={t('notifications.center.allModules')}
                      count={visibleBeforeFilter.length}
                      active={!moduleFilter}
                      onClick={() => setModuleFilter(null)}
                    />
                    {availableModules.map((m) => (
                      <FilterChip
                        key={m}
                        label={moduleLabelOf(t, m)}
                        count={moduleCounts[m]}
                        color={moduleHsl(m)}
                        active={moduleFilter === m}
                        onClick={() => setModuleFilter((cur) => (cur === m ? null : m))}
                      />
                    ))}
                  </div>
                  <SortMenu
                    options={sortOptions}
                    field={sortField}
                    direction={sortDir}
                    onChange={(f, d) => {
                      setSortField(f as 'date' | 'priority')
                      setSortDir(d)
                    }}
                  />
                </div>

                {/* Priority filter chips — only when more than one priority is present. */}
                {availablePriorities.length > 1 && (
                  <div className="flex flex-wrap items-center gap-1.5">
                    <FilterChip
                      label={t('notifications.center.allPriorities')}
                      count={visibleBeforeFilter.length}
                      active={!priorityFilter}
                      onClick={() => setPriorityFilter(null)}
                    />
                    {availablePriorities.map((p) => (
                      <FilterChip
                        key={p}
                        label={t(`notifications.priority.${p}`)}
                        count={priorityCounts[p]}
                        color={PRIORITY_DOT[p]}
                        active={priorityFilter === p}
                        onClick={() => setPriorityFilter((cur) => (cur === p ? null : p))}
                      />
                    ))}
                  </div>
                )}
              </div>
            )}

            {isLoading ? (
              <div className="flex items-center justify-center py-12">
                <div className="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
              </div>
            ) : notifications.length === 0 ? (
              <NotificationsEmpty
                isUnreadFilter={activeTab === 'unread'}
                filteredModule={moduleFilter ? moduleLabelOf(t, moduleFilter) : null}
              />
            ) : (
              <div className="space-y-2">
                {notifications.map((notification) => (
                  <NotificationCard
                    key={notification.id}
                    notification={notification}
                    pinned={Boolean(notification.is_pinned)}
                    onOpenDetail={() => setDetailId(notification.id ?? null)}
                    onMarkRead={() => {
                      if (notification.id) markRead.mutate(notification.id)
                    }}
                  />
                ))}

                {/* Pagination */}
                <div className="flex items-center justify-between pt-4">
                  <p className="text-sm text-muted-foreground">
                    {t('notifications.center.page', { page, total: displayTotal })}
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

      {/* Row detail — centred shared modal with all fields + actions */}
      {detailNotification && (
        <NotificationDetailModal
          notification={detailNotification}
          pinned={Boolean(detailNotification.is_pinned)}
          onClose={() => setDetailId(null)}
          onOpen={() => {
            openNotification(detailNotification)
            setDetailId(null)
          }}
          onPin={() => detailNotification.id && togglePin(detailNotification.id, Boolean(detailNotification.is_pinned))}
          onDismiss={() => detailNotification.id && dismiss(detailNotification.id)}
          onSnooze={(until) => detailNotification.id && snooze(detailNotification.id, until)}
          onMarkRead={() => detailNotification.id && markRead.mutate(detailNotification.id)}
        />
      )}
    </div>
  )
}

function FilterChip({
  label,
  count,
  color,
  active,
  onClick,
}: {
  label: string
  count: number
  color?: string
  active: boolean
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={cn(
        'flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs transition-colors',
        active
          ? 'border-primary bg-primary/10 text-foreground'
          : 'border-border text-muted-foreground hover:bg-secondary hover:text-foreground'
      )}
    >
      {color && <span className="h-2 w-2 shrink-0 rounded-full" style={{ backgroundColor: color }} />}
      <span>{label}</span>
      <span className={cn('rounded-full px-1.5 text-[10px] tabular-nums', active ? 'bg-primary/15' : 'bg-secondary')}>
        {count}
      </span>
    </button>
  )
}

/** Small priority pill (colour + localized label) for cards/modal. */
function PriorityPill({ priority }: { priority?: string }) {
  const { t } = useTranslation()
  if (!priority) return null
  return (
    <span
      className={cn(
        'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
        getPriorityColor(priority)
      )}
    >
      {t(`notifications.priority.${priority}`)}
    </span>
  )
}

function NotificationCard({
  notification,
  pinned,
  onOpenDetail,
  onMarkRead,
}: {
  notification: Notification
  pinned: boolean
  onOpenDetail: () => void
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
      role="button"
      tabIndex={0}
      aria-label={notification.title}
      className={cn(
        'cursor-pointer transition-colors hover:bg-accent/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
        !notification.is_read && 'border-l-4 border-l-primary',
        pinned && 'ring-1 ring-primary/40'
      )}
      onClick={onOpenDetail}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onOpenDetail()
        }
      }}
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
                <p className="mt-1 text-sm text-muted-foreground line-clamp-2">
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
                aria-label={t('notifications.actions.markRead')}
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
                {moduleLabelOf(t, notification.module_id)}
              </Badge>
            )}
            {notification.group_count && notification.group_count > 1 && (
              <Badge variant="outline" className="text-xs">
                {t('notifications.center.more', { count: notification.group_count - 1 })}
              </Badge>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

function NotificationDetailModal({
  notification,
  pinned,
  onClose,
  onOpen,
  onPin,
  onDismiss,
  onSnooze,
  onMarkRead,
}: {
  notification: NotificationWithActor
  pinned: boolean
  onClose: () => void
  onOpen: () => void
  onPin: () => void
  onDismiss: () => void
  onSnooze: (until: string) => void
  onMarkRead: () => void
}) {
  const { t } = useTranslation()
  const isSystem = !notification.actor_id
  const actorName = notification.actor_name || t('notifications.detail.system')
  const timeAgo = notification.created_at
    ? formatDistanceToNow(new Date(notification.created_at), { addSuffix: true, locale: de })
    : ''
  const fullDate = notification.created_at
    ? format(new Date(notification.created_at), 'dd.MM.yyyy HH:mm', { locale: de })
    : '—'
  const moduleName = moduleLabelOf(t, notification.module_id)

  return (
    <DetailModal
      open
      onClose={onClose}
      title={notification.title}
      badge={<PriorityPill priority={notification.priority} />}
      footer={
        <div className="flex flex-wrap items-center gap-2">
          {(notification.deep_link || notification.module_id) && (
            <Button size="sm" onClick={onOpen}>
              <ArrowRight className="mr-1.5 h-3.5 w-3.5" />
              {t('notifications.actions.open')}
            </Button>
          )}
          {!notification.is_read && (
            <Button variant="outline" size="sm" onClick={onMarkRead}>
              <Check className="mr-1.5 h-3.5 w-3.5" />
              {t('notifications.actions.markRead')}
            </Button>
          )}
          <Button variant="outline" size="sm" onClick={onPin}>
            {pinned ? <PinOff className="mr-1.5 h-3.5 w-3.5" /> : <Pin className="mr-1.5 h-3.5 w-3.5" />}
            {pinned ? t('notifications.actions.unpin') : t('notifications.actions.pin')}
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="sm">
                <AlarmClock className="mr-1.5 h-3.5 w-3.5" />
                {t('notifications.actions.snooze')}
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start">
              <DropdownMenuLabel>{t('notifications.actions.snoozeLabel')}</DropdownMenuLabel>
              <DropdownMenuItem onClick={() => onSnooze(snoozeUntil('30min'))}>
                {t('notifications.actions.snooze30min')}
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => onSnooze(snoozeUntil('1h'))}>
                {t('notifications.actions.snooze1h')}
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => onSnooze(snoozeUntil('tomorrow'))}>
                {t('notifications.actions.snoozeTomorrow')}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
          <Button variant="ghost" size="sm" className="text-muted-foreground" onClick={onDismiss}>
            <EyeOff className="mr-1.5 h-3.5 w-3.5" />
            {t('notifications.actions.dismiss')}
          </Button>
        </div>
      }
    >
      {/* Actor */}
      <div className="flex items-center gap-3">
        <Avatar className="h-10 w-10">
          <AvatarFallback className={cn(isSystem && 'bg-primary/10 text-primary')}>
            {isSystem ? <Megaphone className="h-4 w-4" /> : getInitials(actorName)}
          </AvatarFallback>
        </Avatar>
        <div className="min-w-0">
          <p className="text-sm font-medium text-foreground">{actorName}</p>
          <p className="text-xs text-muted-foreground" title={fullDate}>
            {timeAgo}
          </p>
        </div>
      </div>

      {/* Body */}
      {notification.body && (
        <p className="mt-4 text-sm leading-relaxed text-foreground">{notification.body}</p>
      )}

      {/* Meta */}
      <dl className="mt-5 grid grid-cols-2 gap-x-6 gap-y-3 border-t border-border pt-4">
        {notification.module_id && (
          <MetaRow label={t('notifications.detail.module')}>
            <Badge variant="secondary" className="text-xs">{moduleName}</Badge>
          </MetaRow>
        )}
        {notification.priority && (
          <MetaRow label={t('notifications.detail.priority')}>
            <PriorityPill priority={notification.priority} />
          </MetaRow>
        )}
        <MetaRow label={t('notifications.detail.from')}>
          <span className="text-sm text-foreground">{actorName}</span>
        </MetaRow>
        <MetaRow label={t('notifications.detail.received')}>
          <span className="text-sm text-foreground">{fullDate}</span>
        </MetaRow>
      </dl>
    </DetailModal>
  )
}

function MetaRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd>{children}</dd>
    </div>
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
                <h4 className="text-sm font-semibold text-foreground mb-3">
                  {moduleLabelOf(t, moduleId)}
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

function NotificationsEmpty({ isUnreadFilter, filteredModule }: { isUnreadFilter: boolean; filteredModule?: string | null }) {
  const { t } = useTranslation()
  const title = filteredModule
    ? t('notifications.center.emptyFiltered', { module: filteredModule })
    : isUnreadFilter
      ? t('notifications.center.emptyUnread')
      : t('notifications.center.emptyAll')
  const description = filteredModule
    ? t('notifications.center.emptyFilteredDescription')
    : isUnreadFilter
      ? t('notifications.center.emptyUnreadDescription')
      : t('notifications.center.emptyAllDescription')
  return <EmptyState icon={BellOff} title={title} description={description} />
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

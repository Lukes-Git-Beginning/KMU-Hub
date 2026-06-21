/**
 * NotificationToast — surfaces newly arrived notifications as rich Sonner toasts.
 *
 * Single source of truth: the MSW/React-Query notification pipeline (same data
 * the center + bell read). The list is polled so new notifications "arrive"
 * during the session; a stable id-set diff fires one toast per genuinely new,
 * unread entry. DND + quiet hours suppress toasts entirely (N-1).
 *
 * Mounted once in the App shell alongside <Toaster />. It lives OUTSIDE the
 * router, so navigation uses the hash directly rather than useNavigate.
 */
import { useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
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
  Check,
  X,
} from 'lucide-react'
import { cn } from '@/lib/cn'
import {
  useNotifications,
  useMarkNotificationRead,
  useDNDStatus,
  useQuietHours,
  type Notification,
} from '@/api/hooks/useNotifications'
import { useNotificationsStore } from '@/stores/notifications'
import { playNotificationSound, prefersReducedMotion } from '@/lib/notification-sound'
import { shouldSuppressToast } from './notification-gating'

// Seeds carry actor_name beyond the openapi Notification type — read it safely.
type ToastNotification = Notification & { actor_name?: string }

// ---------------------------------------------------------------------------
// Module → icon (mirrors the center/bell so a toast looks like its row)
// ---------------------------------------------------------------------------

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

/** Up to two uppercase initials from a display name. */
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

/** Navigate via the hash router (the toast renders outside <RouterProvider>). */
function navigateHash(to: string) {
  const path = to.startsWith('/') ? to : `/${to}`
  window.location.hash = `#${path}`
}

// ---------------------------------------------------------------------------
// Toast content
// ---------------------------------------------------------------------------

function ToastContent({
  notification,
  toastId,
  onOpen,
  onMarkRead,
}: {
  notification: ToastNotification
  toastId: string | number
  onOpen: () => void
  onMarkRead: () => void
}) {
  const { t } = useTranslation()
  const Icon = iconFor(notification.module_id)
  const initials = notification.actor_id ? initialsOf(notification.actor_name) : ''
  const urgent = notification.priority === 'high' || notification.priority === 'urgent'

  return (
    <div
      className={cn(
        'w-[360px] overflow-hidden rounded-lg border bg-card shadow-lg',
        urgent ? 'border-error/40' : 'border-border/60'
      )}
    >
      <div className="p-3">
        <div className="flex items-start gap-3">
          {initials ? (
            <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary/10 text-xs font-medium text-primary">
              {initials}
            </div>
          ) : (
            <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-muted">
              {/* eslint-disable-next-line react-hooks/static-components -- Icon is a dynamic component variable */}
              <Icon className="h-4 w-4 text-muted-foreground" />
            </div>
          )}

          <div className="min-w-0 flex-1">
            <p className="text-sm font-medium leading-tight text-foreground">{notification.title}</p>
            {notification.body && (
              <p className="mt-0.5 line-clamp-2 text-xs text-muted-foreground">{notification.body}</p>
            )}
          </div>

          <button
            onClick={() => toast.dismiss(toastId)}
            className="shrink-0 rounded p-0.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            aria-label={t('notifications.actions.dismiss')}
          >
            <X className="h-3.5 w-3.5" />
          </button>
        </div>

        <div className="mt-2.5 flex items-center gap-2 pl-11">
          {(notification.deep_link || notification.module_id) && (
            <button
              onClick={() => {
                onOpen()
                toast.dismiss(toastId)
              }}
              className="inline-flex items-center gap-1 rounded-md bg-primary px-2.5 py-1 text-xs font-medium text-primary-foreground transition-colors hover:bg-primary/90"
            >
              <ArrowRight className="h-3.5 w-3.5" />
              {t('notifications.actions.open')}
            </button>
          )}
          <button
            onClick={() => {
              onMarkRead()
              toast.dismiss(toastId)
            }}
            className="inline-flex items-center gap-1 rounded-md bg-muted px-2.5 py-1 text-xs font-medium text-foreground transition-colors hover:bg-muted/80"
          >
            <Check className="h-3.5 w-3.5" />
            {t('notifications.actions.markRead')}
          </button>
        </div>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// NotificationToast (watches the MSW pipeline)
// ---------------------------------------------------------------------------

export default function NotificationToast() {
  // Poll the SAME query the bell + widget read (identical page size, so the
  // shared cache never flip-flops), keeping one source of truth while letting
  // new arrivals surface within a few seconds. Arrivals unshift to the front,
  // so they always land inside the first page.
  const { data } = useNotifications({ pageSize: 10, refetchInterval: 8000 })
  const markRead = useMarkNotificationRead()
  const { data: dnd } = useDNDStatus()
  const { data: quietHours } = useQuietHours()
  const soundEnabled = useNotificationsStore((s) => s.soundEnabled)

  // null until the first list resolves — that first batch seeds the "seen" set
  // silently so a reload never replays the backlog as toasts.
  const seenRef = useRef<Set<string> | null>(null)

  useEffect(() => {
    const notifs = data?.notifications
    if (!notifs) return // not loaded yet — wait for the first real batch to seed
    const list = notifs as ToastNotification[]
    if (seenRef.current === null) {
      // First resolved batch seeds the "seen" set silently (no backlog replay).
      seenRef.current = new Set(list.map((n) => n.id).filter((id): id is string => Boolean(id)))
      return
    }
    const seen = seenRef.current
    const suppressed = shouldSuppressToast({ dnd, quietHours })

    let chimed = false
    for (const n of list) {
      const id = n.id
      if (!id || seen.has(id)) continue
      seen.add(id)
      if (n.is_read) continue
      // Acknowledge-but-stay-silent while the user asked for quiet.
      if (suppressed) continue

      toast.custom(
        (tid) => (
          <ToastContent
            notification={n}
            toastId={tid}
            onOpen={() => {
              if (n.id) markRead.mutate(n.id)
              const target = n.deep_link || (n.module_id ? `/${n.module_id}` : '/')
              navigateHash(target)
            }}
            onMarkRead={() => {
              if (n.id) markRead.mutate(n.id)
            }}
          />
        ),
        {
          id: `notif-${id}`,
          duration: n.priority === 'high' || n.priority === 'urgent' ? 8000 : 5000,
        }
      )

      if (!chimed && soundEnabled && !prefersReducedMotion()) {
        playNotificationSound()
        chimed = true
      }
    }
  }, [data, dnd, quietHours, soundEnabled, markRead])

  return null
}

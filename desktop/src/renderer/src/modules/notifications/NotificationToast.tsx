/**
 * NotificationToast — watches notification store for new entries and
 * renders rich toasts via Sonner's toast.custom().
 *
 * Mounted once in App shell alongside <Toaster />.
 * Supports action buttons, snooze dropdown, and sound toggle.
 */
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
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
  AlarmClock,
  X,
} from 'lucide-react'
import { useNotificationsStore, type Notification } from '../../stores/notifications'

// ---------------------------------------------------------------------------
// Icon mapping
// ---------------------------------------------------------------------------

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

function getIcon(iconName: string) {
  return iconMap[iconName] || Bell
}

// ---------------------------------------------------------------------------
// Snooze options
// ---------------------------------------------------------------------------

const snoozeOptions = [
  { labelKey: 'notifications.toast.snooze30', minutes: 30 },
  { labelKey: 'notifications.toast.snooze1h', minutes: 60 },
  { labelKey: 'notifications.toast.snoozeTomorrow', minutes: 24 * 60 },
]

// ---------------------------------------------------------------------------
// Custom toast content
// ---------------------------------------------------------------------------

function ToastContent({
  notification,
  toastId,
}: {
  notification: Notification
  toastId: string | number
}) {
  const { t } = useTranslation()
  const [showSnooze, setShowSnooze] = useState(false)
  const { snoozeNotification, handleNotificationAction, markAsRead } = useNotificationsStore()

  const Icon = getIcon(notification.icon)

  const priorityBg: Record<string, string> = {
    high: 'border-error/40 bg-error-light/30',
    normal: 'border-border/60 bg-card',
    low: 'border-border/40 bg-card',
  }

  return (
    <div
      className={`w-[360px] rounded-lg border shadow-lg overflow-hidden ${priorityBg[notification.priority] || priorityBg.normal}`}
    >
      <div className="p-3">
        <div className="flex items-start gap-3">
          {/* Actor avatar or icon */}
          {notification.actorInitials ? (
            <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary/10 text-xs font-medium text-primary">
              {notification.actorInitials}
            </div>
          ) : (
            <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-muted">
              {/* eslint-disable-next-line react-hooks/static-components -- Icon is a dynamic component variable */}
              <Icon className="h-4 w-4 text-muted-foreground" />
            </div>
          )}

          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium text-foreground leading-tight">{notification.title}</p>
            <p className="mt-0.5 text-xs text-muted-foreground line-clamp-2">{notification.body}</p>
          </div>

          <button
            onClick={() => toast.dismiss(toastId)}
            className="shrink-0 rounded p-0.5 text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
          >
            <X className="h-3.5 w-3.5" />
          </button>
        </div>

        {/* Action buttons */}
        {notification.actions && notification.actions.length > 0 && (
          <div className="mt-2.5 flex items-center gap-2 pl-11">
            {notification.actions.map((action) => (
              <button
                key={action.key}
                onClick={() => {
                  handleNotificationAction(notification.id, action.key)
                  toast.dismiss(toastId)
                }}
                className={`rounded-md px-2.5 py-1 text-xs font-medium transition-colors ${
                  action.variant === 'primary'
                    ? 'bg-primary text-primary-foreground hover:bg-primary/90'
                    : action.variant === 'destructive'
                      ? 'bg-red-600 text-white hover:bg-red-700'
                      : 'bg-muted text-foreground hover:bg-muted/80'
                }`}
              >
                {action.label}
              </button>
            ))}

            {/* Snooze toggle */}
            <div className="relative ml-auto">
              <button
                onClick={() => setShowSnooze(!showSnooze)}
                className="rounded p-1 text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
                title={t('notifications.toast.snooze')}
              >
                <AlarmClock className="h-3.5 w-3.5" />
              </button>

              {showSnooze && (
                <div className="absolute right-0 bottom-full mb-1 w-28 rounded-lg border border-border bg-card shadow-lg py-1 z-10">
                  {snoozeOptions.map((opt) => (
                    <button
                      key={opt.minutes}
                      onClick={() => {
                        const until = new Date(Date.now() + opt.minutes * 60000).toISOString()
                        snoozeNotification(notification.id, until)
                        toast.dismiss(toastId)
                      }}
                      className="w-full px-3 py-1.5 text-left text-xs text-foreground hover:bg-muted transition-colors"
                    >
                      {t(opt.labelKey)}
                    </button>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}

        {/* No action buttons — show snooze inline */}
        {(!notification.actions || notification.actions.length === 0) && (
          <div className="mt-2 flex items-center gap-2 pl-11">
            <button
              onClick={() => {
                markAsRead(notification.id)
                toast.dismiss(toastId)
              }}
              className="text-xs text-primary hover:underline"
            >
              {t('notifications.toast.markRead')}
            </button>
            <div className="relative ml-auto">
              <button
                onClick={() => setShowSnooze(!showSnooze)}
                className="rounded p-1 text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
                title={t('notifications.toast.snooze')}
              >
                <AlarmClock className="h-3.5 w-3.5" />
              </button>

              {showSnooze && (
                <div className="absolute right-0 bottom-full mb-1 w-28 rounded-lg border border-border bg-card shadow-lg py-1 z-10">
                  {snoozeOptions.map((opt) => (
                    <button
                      key={opt.minutes}
                      onClick={() => {
                        const until = new Date(Date.now() + opt.minutes * 60000).toISOString()
                        snoozeNotification(notification.id, until)
                        toast.dismiss(toastId)
                      }}
                      className="w-full px-3 py-1.5 text-left text-xs text-foreground hover:bg-muted transition-colors"
                    >
                      {t(opt.labelKey)}
                    </button>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// NotificationToast (watches store)
// ---------------------------------------------------------------------------

export default function NotificationToast() {
  const notifications = useNotificationsStore((s) => s.notifications)
  const prevLengthRef = useRef(notifications.length)

  useEffect(() => {
    const currentLen = notifications.length
    if (currentLen > prevLengthRef.current) {
      // New notification(s) added at the front
      const newCount = currentLen - prevLengthRef.current
      for (let i = 0; i < newCount; i++) {
        const n = notifications[i]
        if (n.snoozedUntil && new Date(n.snoozedUntil) > new Date()) continue
        toast.custom((id) => <ToastContent notification={n} toastId={id} />, {
          duration: n.priority === 'high' ? 8000 : 5000,
          id: `notif-${n.id}`,
        })
      }
    }
    prevLengthRef.current = currentLen
  }, [notifications])

  return null
}

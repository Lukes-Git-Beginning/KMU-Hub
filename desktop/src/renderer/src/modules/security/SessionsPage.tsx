/**
 * Active sessions management page.
 *
 * Shows a list of active login sessions with device info, IP address,
 * and location. Users can terminate individual sessions or all others.
 * Admin view toggles to show all users' sessions.
 */
import { useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useFormatDate } from '@/hooks/useFormatters'
import { toast } from 'sonner'
import { Monitor, Smartphone, Tablet, MapPin, Wifi } from 'lucide-react'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { useAuthStore } from '@/stores/auth'
import {
  useMySessions,
  useAllSessions,
  useTerminateSession,
  useTerminateAllSessions,
} from '@/api/hooks/useSessions'
import type { UserSession } from '@/api/security-types'

const deviceConfig: Record<string, { icon: typeof Monitor; colorBg: string; colorText: string }> = {
  desktop: { icon: Monitor, colorBg: 'bg-info-light', colorText: 'text-info' },
  mobile: { icon: Smartphone, colorBg: 'bg-primary-light', colorText: 'text-primary' },
  tablet: { icon: Tablet, colorBg: 'bg-warning-light', colorText: 'text-warning' },
}

function getDeviceConfig(deviceType: string) {
  return deviceConfig[deviceType.toLowerCase()] ?? deviceConfig.desktop
}

function deviceTypeMessageId(deviceType: string): string {
  switch (deviceType.toLowerCase()) {
    case 'desktop': return 'session.device.desktop'
    case 'mobile': return 'session.device.mobile'
    case 'tablet': return 'session.device.tablet'
    default: return 'session.device.unknown'
  }
}

interface SessionCardProps {
  session: UserSession
  onTerminate: (id: string) => void
  isTerminating: boolean
}

function SessionCard({ session, onTerminate, isTerminating }: SessionCardProps) {
  const { t } = useTranslation()
  const formatDate = useFormatDate()
  const device = getDeviceConfig(session.device_type)
  const DeviceIcon = device.icon

  return (
    <div
      className={`rounded-lg border bg-card p-4 transition-shadow hover:shadow-[var(--shadow-card-hover)] glass-surface ${
        session.is_current
          ? 'border-success bg-success-light/10'
          : 'border-border'
      }`}
    >
      <div className="flex items-center gap-4">
        {/* Device icon */}
        <div className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-lg ${device.colorBg}`}>
          <DeviceIcon className={`h-5 w-5 ${device.colorText}`} />
        </div>

        {/* Info */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="font-medium text-sm text-foreground truncate">
              {session.device_name || t(deviceTypeMessageId(session.device_type))}
            </span>
            {session.is_current && (
              <span className="rounded-full bg-success-light px-2 py-0.5 text-[10px] font-medium text-success shrink-0">
                {t('session.current')}
              </span>
            )}
          </div>

          <div className="flex flex-wrap gap-x-4 gap-y-1 mt-1.5">
            <span className="flex items-center gap-1 text-xs text-muted-foreground">
              <Wifi className="h-3 w-3" />
              <span className="font-mono">{session.ip_address}</span>
            </span>
            {session.location && (
              <span className="flex items-center gap-1 text-xs text-muted-foreground">
                <MapPin className="h-3 w-3" />
                {session.location}
              </span>
            )}
            <span className="text-xs text-muted-foreground">
              {t('session.createdAt', {
                time: formatDate(session.created_at, {
                  year: 'numeric',
                  month: '2-digit',
                  day: '2-digit',
                  hour: '2-digit',
                  minute: '2-digit',
                }),
              })}
            </span>
            <span className="text-xs text-muted-foreground">
              {t('session.lastActive', {
                time: formatDate(session.last_active_at, {
                  year: 'numeric',
                  month: '2-digit',
                  day: '2-digit',
                  hour: '2-digit',
                  minute: '2-digit',
                }),
              })}
            </span>
          </div>
        </div>

        {/* Terminate button */}
        {!session.is_current && (
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <button
                disabled={isTerminating}
                className="shrink-0 rounded-lg border border-error/30 px-3 py-1.5 text-xs font-medium text-error hover:bg-error-light transition-colors disabled:opacity-40"
              >
                {t('session.terminate')}
              </button>
            </AlertDialogTrigger>
            <AlertDialogContent className="glass-elevated">
              <AlertDialogHeader>
                <AlertDialogTitle>
                  {t('session.terminate')}
                </AlertDialogTitle>
                <AlertDialogDescription>
                  {t('session.terminateConfirm')}
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>
                  {t('common.cancel')}
                </AlertDialogCancel>
                <AlertDialogAction onClick={() => onTerminate(session.id)}>
                  {t('common.confirm')}
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        )}
      </div>
    </div>
  )
}

export default function SessionsPage() {
  const { t } = useTranslation()
  const user = useAuthStore((s) => s.user)
  const isAdmin = user?.roles.includes('admin') ?? false

  const [showAll, setShowAll] = useState(false)

  const mySessionsQuery = useMySessions()
  const allSessionsQuery = useAllSessions(0, 100)
  const terminateMutation = useTerminateSession()
  const terminateAllMutation = useTerminateAllSessions()

  const sessions: UserSession[] = showAll && isAdmin
    ? (allSessionsQuery.data?.sessions ?? [])
    : (mySessionsQuery.data ?? [])

  const isLoading = showAll ? allSessionsQuery.isLoading : mySessionsQuery.isLoading

  const handleTerminate = useCallback(
    (sessionId: string) => {
      terminateMutation.mutate(sessionId, {
        onSuccess: () => {
          toast.success(t('session.terminated'))
        },
        onError: (err) => {
          toast.error(err instanceof Error ? err.message : t('common.error'))
        },
      })
    },
    [terminateMutation, t],
  )

  const handleTerminateAll = useCallback(() => {
    terminateAllMutation.mutate(undefined, {
      onSuccess: () => {
        toast.success(t('session.terminated'))
      },
      onError: (err) => {
        toast.error(err instanceof Error ? err.message : t('common.error'))
      },
    })
  }, [terminateAllMutation, t])

  // Sort: current session first, then by last_active_at descending
  const sortedSessions = [...sessions].sort((a, b) => {
    if (a.is_current && !b.is_current) return -1
    if (!a.is_current && b.is_current) return 1
    return new Date(b.last_active_at).getTime() - new Date(a.last_active_at).getTime()
  })

  return (
    <div className="flex-1 overflow-y-auto p-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between mb-6 gap-4">
        <div className="flex items-center gap-4">
          <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-info-light">
            <Monitor className="h-6 w-6 text-info" />
          </div>
          <div>
            <h1 className="text-foreground">
              {t('session.title')}
            </h1>
            <div className="flex items-center gap-2 mt-1">
              <span className="rounded-full bg-info-light px-2.5 py-0.5 text-xs font-medium text-info">
                {sessions.length} {showAll ? 'Alle' : 'Meine'}
              </span>
            </div>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {isAdmin && (
            <button
              onClick={() => setShowAll(!showAll)}
              className={`rounded-lg px-3 py-2 text-sm transition-colors ${
                showAll
                  ? 'bg-primary text-primary-foreground'
                  : 'border border-border text-foreground hover:bg-secondary'
              }`}
            >
              {showAll ? 'My Sessions' : 'All Sessions'}
            </button>
          )}
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <button
                disabled={terminateAllMutation.isPending || sessions.length <= 1}
                className="rounded-lg bg-error px-3 py-2 text-sm text-white hover:bg-error/90 transition-colors disabled:opacity-40"
              >
                {t('session.terminateAll')}
              </button>
            </AlertDialogTrigger>
            <AlertDialogContent className="glass-elevated">
              <AlertDialogHeader>
                <AlertDialogTitle>
                  {t('session.terminateAll')}
                </AlertDialogTitle>
                <AlertDialogDescription>
                  {t('session.terminateAllConfirm')}
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>
                  {t('common.cancel')}
                </AlertDialogCancel>
                <AlertDialogAction onClick={handleTerminateAll}>
                  {t('common.confirm')}
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </div>
      </div>

      {/* Session list */}
      {isLoading && (
        <div className="py-12 text-center">
          <Monitor className="mx-auto h-10 w-10 text-muted-foreground/40 mb-2 animate-pulse" />
          <p className="text-sm text-muted-foreground">
            {t('common.loading')}
          </p>
        </div>
      )}

      {!isLoading && sortedSessions.length === 0 && (
        <div className="py-12 text-center">
          <Monitor className="mx-auto h-10 w-10 text-muted-foreground/40 mb-2" />
          <p className="text-sm text-muted-foreground">
            {t('common.noResults')}
          </p>
        </div>
      )}

      <div className="space-y-3">
        {sortedSessions.map((session) => (
          <SessionCard
            key={session.id}
            session={session}
            onTerminate={handleTerminate}
            isTerminating={terminateMutation.isPending}
          />
        ))}
      </div>
    </div>
  )
}

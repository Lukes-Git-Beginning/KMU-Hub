/**
 * Active sessions management page.
 *
 * Shows a list of active login sessions with device info, IP address,
 * and location. Users can terminate individual sessions or all others.
 * Admin view toggles to show all users' sessions.
 */
import { useState, useCallback } from 'react'
import { FormattedMessage, FormattedDate, useIntl } from 'react-intl'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
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
import { Card, CardContent } from '@/components/ui/card'
import { useAuthStore } from '@/stores/auth'
import {
  useMySessions,
  useAllSessions,
  useTerminateSession,
  useTerminateAllSessions,
} from '@/api/hooks/useSessions'
import type { UserSession } from '@/api/security-types'

function DeviceIcon({ deviceType }: { deviceType: string }) {
  // Simple text-based device indicator
  switch (deviceType.toLowerCase()) {
    case 'desktop':
      return <span className="text-lg" aria-hidden="true">&#128187;</span>
    case 'mobile':
      return <span className="text-lg" aria-hidden="true">&#128241;</span>
    case 'tablet':
      return <span className="text-lg" aria-hidden="true">&#128241;</span>
    default:
      return <span className="text-lg" aria-hidden="true">&#128187;</span>
  }
}

function deviceTypeMessageId(deviceType: string): string {
  switch (deviceType.toLowerCase()) {
    case 'desktop':
      return 'session.device.desktop'
    case 'mobile':
      return 'session.device.mobile'
    case 'tablet':
      return 'session.device.tablet'
    default:
      return 'session.device.unknown'
  }
}

interface SessionCardProps {
  session: UserSession
  onTerminate: (id: string) => void
  isTerminating: boolean
}

function SessionCard({ session, onTerminate, isTerminating }: SessionCardProps) {
  const intl = useIntl()

  return (
    <Card className={session.is_current ? 'border-primary' : ''}>
      <CardContent className="flex items-center gap-4 p-4">
        <DeviceIcon deviceType={session.device_type} />

        <div className="flex-1 space-y-1">
          <div className="flex items-center gap-2">
            <span className="font-medium text-sm">
              {session.device_name || intl.formatMessage({ id: deviceTypeMessageId(session.device_type) })}
            </span>
            {session.is_current && (
              <Badge variant="default" className="text-xs">
                <FormattedMessage id="session.current" />
              </Badge>
            )}
          </div>

          <div className="flex flex-wrap gap-x-4 gap-y-0.5 text-xs text-muted-foreground">
            <span className="font-mono">{session.ip_address}</span>
            {session.location && (
              <span>{session.location}</span>
            )}
            <span>
              <FormattedMessage
                id="session.createdAt"
                values={{
                  time: intl.formatDate(session.created_at, {
                    year: 'numeric',
                    month: '2-digit',
                    day: '2-digit',
                    hour: '2-digit',
                    minute: '2-digit',
                  }),
                }}
              />
            </span>
            <span>
              <FormattedMessage
                id="session.lastActive"
                values={{
                  time: intl.formatDate(session.last_active_at, {
                    year: 'numeric',
                    month: '2-digit',
                    day: '2-digit',
                    hour: '2-digit',
                    minute: '2-digit',
                  }),
                }}
              />
            </span>
          </div>
        </div>

        {!session.is_current && (
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button
                variant="destructive"
                size="sm"
                disabled={isTerminating}
              >
                <FormattedMessage id="session.terminate" />
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>
                  <FormattedMessage id="session.terminate" />
                </AlertDialogTitle>
                <AlertDialogDescription>
                  <FormattedMessage id="session.terminateConfirm" />
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>
                  <FormattedMessage id="common.cancel" />
                </AlertDialogCancel>
                <AlertDialogAction onClick={() => onTerminate(session.id)}>
                  <FormattedMessage id="common.confirm" />
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        )}
      </CardContent>
    </Card>
  )
}

export default function SessionsPage() {
  const intl = useIntl()
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
          toast.success(intl.formatMessage({ id: 'session.terminated' }))
        },
        onError: (err) => {
          toast.error(err instanceof Error ? err.message : intl.formatMessage({ id: 'common.error' }))
        },
      })
    },
    [terminateMutation, intl],
  )

  const handleTerminateAll = useCallback(() => {
    terminateAllMutation.mutate(undefined, {
      onSuccess: () => {
        toast.success(intl.formatMessage({ id: 'session.terminated' }))
      },
      onError: (err) => {
        toast.error(err instanceof Error ? err.message : intl.formatMessage({ id: 'common.error' }))
      },
    })
  }, [terminateAllMutation, intl])

  // Sort: current session first, then by last_active_at descending
  const sortedSessions = [...sessions].sort((a, b) => {
    if (a.is_current && !b.is_current) return -1
    if (!a.is_current && b.is_current) return 1
    return new Date(b.last_active_at).getTime() - new Date(a.last_active_at).getTime()
  })

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">
            <FormattedMessage id="session.title" />
          </h2>
          <p className="text-sm text-muted-foreground">
            <FormattedMessage id="session.description" />
          </p>
        </div>
        <div className="flex gap-2">
          {isAdmin && (
            <Button
              variant={showAll ? 'default' : 'outline'}
              size="sm"
              onClick={() => setShowAll(!showAll)}
            >
              {showAll ? 'My Sessions' : 'All Sessions'}
            </Button>
          )}
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button
                variant="destructive"
                size="sm"
                disabled={terminateAllMutation.isPending || sessions.length <= 1}
              >
                <FormattedMessage id="session.terminateAll" />
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>
                  <FormattedMessage id="session.terminateAll" />
                </AlertDialogTitle>
                <AlertDialogDescription>
                  <FormattedMessage id="session.terminateAllConfirm" />
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>
                  <FormattedMessage id="common.cancel" />
                </AlertDialogCancel>
                <AlertDialogAction onClick={handleTerminateAll}>
                  <FormattedMessage id="common.confirm" />
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </div>
      </div>

      {/* Session list */}
      {isLoading && (
        <p className="text-center text-muted-foreground py-8">
          <FormattedMessage id="common.loading" />
        </p>
      )}

      {!isLoading && sortedSessions.length === 0 && (
        <p className="text-center text-muted-foreground py-8">
          <FormattedMessage id="common.noResults" />
        </p>
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

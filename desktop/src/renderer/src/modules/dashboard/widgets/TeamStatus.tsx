/**
 * Team Status widget — shows who is online, away, or offline.
 *
 * Wired to useEmployees() for the team list and useBulkPresence()
 * for real-time presence status.
 */
import { memo, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2 } from 'lucide-react'
import { useEmployees } from '@/api/hooks/hr-hooks'
import { useBulkPresence } from '@/api/hooks/usePresence'
import type { PresenceLevel } from '@/api/video-types'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'
import { UserProfileTrigger } from '@/components/user/UserProfileCard'

type UIStatus = 'online' | 'away' | 'busy' | 'offline'

const STATUS_CONFIG: Record<UIStatus, { color: string; labelKey: string }> = {
  online: { color: 'bg-success', labelKey: 'dashboard.teamStatus.online' },
  away: { color: 'bg-warning', labelKey: 'dashboard.teamStatus.away' },
  busy: { color: 'bg-destructive', labelKey: 'dashboard.teamStatus.busy' },
  offline: { color: 'bg-muted-foreground/50', labelKey: 'dashboard.teamStatus.offline' },
}

const STATUS_ORDER: UIStatus[] = ['online', 'busy', 'away', 'offline']

/** Map backend PresenceLevel to widget UI status. */
function mapPresence(level: PresenceLevel | undefined): UIStatus {
  switch (level) {
    case 'online':
      return 'online'
    case 'away':
      return 'away'
    case 'dnd':
    case 'in_call':
      return 'busy'
    default:
      return 'offline'
  }
}

/** Build initials from a full name string (e.g. "Max Mustermann" -> "MM"). */
function getInitials(name: string): string {
  const parts = name.trim().split(/\s+/)
  if (parts.length >= 2) return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
  return (parts[0]?.[0] ?? '?').toUpperCase()
}

function TeamStatus(_props: WidgetProps) {
  const { t } = useTranslation()
  const { data: employeeData, isLoading: loadingEmployees } = useEmployees()
  const employees = useMemo(() => employeeData?.employees ?? [], [employeeData?.employees])

  const userIds = useMemo(
    () => employees.map((e) => e.userId).filter(Boolean),
    [employees],
  )

  const { data: presenceMap } = useBulkPresence(userIds)

  const team = useMemo(() => {
    return employees.map((e) => ({
      id: e.id,
      userId: e.userId,
      name: e.userName ?? `${e.department ?? t('dashboard.teamStatus.employee')}`,
      role: e.positionTitle ?? e.department ?? '',
      status: mapPresence(presenceMap?.[e.userId]?.status),
      avatar: getInitials(e.userName ?? '?'),
    }))
  }, [employees, presenceMap, t])

  const sorted = useMemo(
    () => [...team].sort((a, b) => STATUS_ORDER.indexOf(a.status) - STATUS_ORDER.indexOf(b.status)),
    [team],
  )

  const counts = useMemo(() => {
    const c: Record<UIStatus, number> = { online: 0, away: 0, busy: 0, offline: 0 }
    for (const m of team) c[m.status]++
    return c
  }, [team])

  if (loadingEmployees) {
    return (
      <div className="flex h-full items-center justify-center">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (team.length === 0) {
    return (
      <div className="flex h-full items-center justify-center px-4">
        <p className="text-xs text-muted-foreground">{t('dashboard.teamStatus.noMembers')}</p>
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col">
      {/* Summary bar */}
      <div className="flex items-center gap-3 px-4 pt-4 pb-2">
        {STATUS_ORDER.map((s) => (
          <div key={s} className="flex items-center gap-1">
            <span className={`h-2 w-2 rounded-full ${STATUS_CONFIG[s].color}`} />
            <span className="text-xs text-muted-foreground">
              {counts[s]} {t(STATUS_CONFIG[s].labelKey)}
            </span>
          </div>
        ))}
      </div>

      {/* Member list */}
      <div className="flex-1 overflow-auto divide-y divide-border">
        {sorted.map((member) => (
          <div
            key={member.id}
            className="flex items-center gap-3 px-4 py-2 hover:bg-accent/50 transition-colors"
          >
            <div className="relative">
              {member.userId ? (
                <UserProfileTrigger userId={member.userId} name={member.name}>
                  <button
                    type="button"
                    className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/10 text-xs font-semibold text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
                  >
                    {member.avatar}
                  </button>
                </UserProfileTrigger>
              ) : (
                <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/10 text-xs font-semibold text-primary">
                  {member.avatar}
                </div>
              )}
              <span
                className={`absolute -bottom-0.5 -right-0.5 h-3 w-3 rounded-full border-2 border-card pointer-events-none ${STATUS_CONFIG[member.status].color}`}
              />
            </div>
            <div className="min-w-0 flex-1">
              <p className="text-sm font-medium text-foreground truncate">{member.name}</p>
              <p className="text-xs text-muted-foreground truncate">{member.role}</p>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

export default memo(TeamStatus)

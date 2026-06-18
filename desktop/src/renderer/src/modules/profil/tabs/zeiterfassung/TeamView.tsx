import { useTranslation } from 'react-i18next'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { cn } from '@/lib/cn'
import { useTeamTime } from '@/api/hooks/hr-hooks'
import { UserProfileTrigger } from '@/components/user/UserProfileCard'
import { formatHoursDecimal } from './time-utils'
import type { TeamTimeRow } from '@/api/hr-types'

function getWeekStartStr(): string {
  const d = new Date()
  const day = d.getDay()
  const mondayOffset = day === 0 ? -6 : 1 - day
  d.setDate(d.getDate() + mondayOffset)
  return d.toISOString().split('T')[0]
}

function getInitials(name: string): string {
  return name
    .split(' ')
    .map((w) => w[0] ?? '')
    .join('')
    .slice(0, 2)
    .toUpperCase()
}

export default function TeamView() {
  const { t } = useTranslation()
  const weekStart = getWeekStartStr()
  const { data: teamRows = [] } = useTeamTime(weekStart)

  type StatusKey = 'clocked_in' | 'clocked_out' | 'absent'
  const STATUS_MAP: Record<StatusKey, { label: string; dotClass: string; animate: boolean }> = {
    clocked_in: { label: t('profil.zeiterfassung.team.active'), dotClass: 'bg-success', animate: true },
    clocked_out: { label: t('profil.zeiterfassung.team.idle'), dotClass: 'bg-gray-400', animate: false },
    absent: { label: t('profil.zeiterfassung.team.absent'), dotClass: 'bg-warning', animate: false },
  }

  const sorted = [...(teamRows as TeamTimeRow[])].sort((a, b) => {
    const order = { clocked_in: 0, clocked_out: 1, absent: 2 }
    const aKey: StatusKey = a.clockedIn ? 'clocked_in' : 'clocked_out'
    const bKey: StatusKey = b.clockedIn ? 'clocked_in' : 'clocked_out'
    return (order[aKey] ?? 1) - (order[bKey] ?? 1)
  })

  const activeCount = (teamRows as TeamTimeRow[]).filter((m) => m.clockedIn).length

  return (
    <div className="p-6 max-w-3xl mx-auto space-y-4">
      <div className="flex items-center justify-between mb-2">
        <h3 className="text-sm font-semibold text-foreground">
          {t('profil.zeiterfassung.team.title', { count: teamRows.length })}
        </h3>
        <div className="flex items-center gap-4 text-xs text-muted-foreground">
          <span className="flex items-center gap-1.5">
            <span className="h-2 w-2 rounded-full bg-success" />
            {t('profil.zeiterfassung.team.active')} ({activeCount})
          </span>
          <span className="flex items-center gap-1.5">
            <span className="h-2 w-2 rounded-full bg-gray-400" />
            {t('profil.zeiterfassung.team.idle')} ({teamRows.length - activeCount})
          </span>
        </div>
      </div>

      <div className="space-y-2">
        {sorted.map((member) => {
          const statusKey: StatusKey = member.clockedIn ? 'clocked_in' : 'clocked_out'
          const status = STATUS_MAP[statusKey]
          const initials = getInitials(member.name)
          const progressPercent = member.targetMinutes > 0
            ? Math.min(100, Math.round((member.weekMinutes / member.targetMinutes) * 100))
            : 0
          return (
            <div
              key={member.employeeId}
              className="flex items-center gap-4 p-4 rounded-xl border border-border bg-card hover:border-primary/30 transition-colors"
            >
              {/* Avatar */}
              <div className="relative">
                <UserProfileTrigger userId={member.employeeId} name={member.name}>
                  <button type="button" className="rounded-full focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50">
                    <Avatar className="h-10 w-10">
                      <AvatarFallback className="text-sm">{initials}</AvatarFallback>
                    </Avatar>
                  </button>
                </UserProfileTrigger>
                <span className={cn(
                  'absolute -bottom-0.5 -right-0.5 h-3 w-3 rounded-full border-2 border-card pointer-events-none',
                  status.dotClass,
                )}>
                  {status.animate && (
                    <span className={cn('absolute inset-0 rounded-full animate-ping opacity-40', status.dotClass)} />
                  )}
                </span>
              </div>

              {/* Info */}
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <span className="font-medium text-sm text-foreground">{member.name}</span>
                  {member.department && (
                    <span className="text-xs text-muted-foreground">({member.department})</span>
                  )}
                  <span className={cn(
                    'px-2 py-0.5 rounded-full text-[10px] font-medium',
                    member.clockedIn
                      ? 'bg-success-light text-success'
                      : 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400',
                  )}>
                    {status.label}
                  </span>
                </div>
                {/* Week progress */}
                <div className="flex items-center gap-2">
                  <div className="flex-1 h-1.5 rounded-full bg-secondary overflow-hidden">
                    <div
                      className={cn('h-full rounded-full transition-all', progressPercent >= 100 ? 'bg-success' : 'bg-primary/60')}
                      style={{ width: `${progressPercent}%` }}
                    />
                  </div>
                  <span className="text-xs text-muted-foreground tabular-nums shrink-0">
                    {formatHoursDecimal(member.weekMinutes)}h / {formatHoursDecimal(member.targetMinutes)}h
                  </span>
                </div>
              </div>

              {/* Week status badge */}
              {member.weekStatus && member.weekStatus !== 'open' && (
                <span className={cn(
                  'px-2 py-0.5 rounded-full text-[10px] font-medium shrink-0',
                  member.weekStatus === 'approved'
                    ? 'bg-success-light text-success'
                    : member.weekStatus === 'submitted'
                      ? 'bg-warning-light text-warning-foreground'
                      : member.weekStatus === 'rejected'
                        ? 'bg-error-light text-destructive'
                        : 'bg-secondary text-muted-foreground',
                )}>
                  {t(`profil.zeiterfassung.approval.status.${member.weekStatus}`)}
                </span>
              )}
            </div>
          )
        })}

        {teamRows.length === 0 && (
          <p className="text-center text-sm text-muted-foreground py-8">
            {t('profil.zeiterfassung.team.noData')}
          </p>
        )}
      </div>
    </div>
  )
}

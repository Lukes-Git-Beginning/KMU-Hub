import { useMemo } from 'react'
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
  const offset = day === 0 ? -6 : 1 - day
  d.setDate(d.getDate() + offset)
  return d.toISOString().split('T')[0]
}

function getInitials(name: string): string {
  return name
    .split(' ')
    .map((w) => w[0])
    .join('')
    .toUpperCase()
    .slice(0, 2)
}

function getStatus(row: TeamTimeRow): 'tracking' | 'idle' | 'absent' {
  return row.clockedIn ? 'tracking' : 'idle'
}

export default function TeamView() {
  const { t } = useTranslation()
  const weekStart = useMemo(getWeekStartStr, [])
  const { data: rows = [], isLoading } = useTeamTime(weekStart)

  const STATUS_MAP = {
    tracking: { label: t('profil.zeiterfassung.team.active'), dotClass: 'bg-success', animate: true },
    idle: { label: t('profil.zeiterfassung.team.idle'), dotClass: 'bg-gray-400', animate: false },
    absent: { label: t('profil.zeiterfassung.team.absent'), dotClass: 'bg-warning', animate: false },
  }

  const sorted = [...rows].sort((a, b) => {
    const order: Record<string, number> = { tracking: 0, idle: 1, absent: 2 }
    return (order[getStatus(a)] ?? 1) - (order[getStatus(b)] ?? 1)
  })

  const trackingCount = rows.filter((r) => r.clockedIn).length
  const idleCount = rows.filter((r) => !r.clockedIn).length

  return (
    <div className="p-6 max-w-3xl mx-auto space-y-4">
      <div className="flex items-center justify-between mb-2">
        <h3 className="text-sm font-semibold text-foreground">
          {t('profil.zeiterfassung.team.title', { count: rows.length })}
        </h3>
        <div className="flex items-center gap-4 text-xs text-muted-foreground">
          <span className="flex items-center gap-1.5">
            <span className="h-2 w-2 rounded-full bg-success" />
            {t('profil.zeiterfassung.team.active')} ({trackingCount})
          </span>
          <span className="flex items-center gap-1.5">
            <span className="h-2 w-2 rounded-full bg-gray-400" />
            {t('profil.zeiterfassung.team.idle')} ({idleCount})
          </span>
        </div>
      </div>

      {isLoading && (
        <div className="space-y-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="h-[72px] rounded-xl border border-border bg-card animate-pulse" />
          ))}
        </div>
      )}

      {!isLoading && (
        <div className="space-y-2">
          {sorted.map((member) => {
            const statusKey = getStatus(member)
            const status = STATUS_MAP[statusKey]
            const initials = getInitials(member.name)
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
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-sm text-foreground">{member.name}</span>
                    <span className={cn(
                      'px-2 py-0.5 rounded-full text-[10px] font-medium',
                      statusKey === 'tracking'
                        ? 'bg-success-light text-success'
                        : statusKey === 'absent'
                          ? 'bg-warning-light text-warning-foreground'
                          : 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400',
                    )}>
                      {status.label}
                    </span>
                    {member.weekStatus && member.weekStatus !== 'open' && (
                      <span className="px-2 py-0.5 rounded-full text-[10px] font-medium bg-secondary text-muted-foreground">
                        {member.weekStatus}
                      </span>
                    )}
                  </div>
                  {member.department && (
                    <p className="text-sm text-muted-foreground mt-0.5 truncate">
                      {member.department}
                    </p>
                  )}
                </div>

                {/* Week hours */}
                <div className="text-right shrink-0">
                  <span className="text-sm font-medium text-foreground tabular-nums">
                    {formatHoursDecimal(member.weekMinutes)}h
                  </span>
                  <span className="text-xs text-muted-foreground block">
                    / {formatHoursDecimal(member.targetMinutes)}h
                  </span>
                </div>
              </div>
            )
          })}

          {rows.length === 0 && (
            <div className="text-center py-12 text-muted-foreground">
              <p className="font-medium">{t('profil.zeiterfassung.team.noData')}</p>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

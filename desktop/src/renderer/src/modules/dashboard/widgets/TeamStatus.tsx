/**
 * Team Status widget — shows who is online, away, or offline.
 */
import { memo } from 'react'
import { EMPLOYEES } from '@/mocks/mock-db'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

type Status = 'online' | 'away' | 'busy' | 'offline'

const STATUS_CONFIG: Record<Status, { color: string; label: string }> = {
  online: { color: 'bg-emerald-500', label: 'Online' },
  away: { color: 'bg-amber-500', label: 'Abwesend' },
  busy: { color: 'bg-red-500', label: 'Beschaeftigt' },
  offline: { color: 'bg-gray-400', label: 'Offline' },
}

const STATUS_ORDER: Status[] = ['online', 'busy', 'away', 'offline']

const TEAM = EMPLOYEES.map((e) => ({
  id: e.id,
  name: `${e.firstName} ${e.lastName}`,
  role: e.jobTitle,
  status: e.status,
  avatar: e.initials,
}))

function TeamStatus(_props: WidgetProps) {
  const sorted = [...TEAM].sort(
    (a, b) => STATUS_ORDER.indexOf(a.status) - STATUS_ORDER.indexOf(b.status)
  )

  const counts = TEAM.reduce<Record<Status, number>>(
    (acc, m) => { acc[m.status]++; return acc },
    { online: 0, away: 0, busy: 0, offline: 0 }
  )

  return (
    <div className="flex h-full flex-col">
      {/* Summary bar */}
      <div className="flex items-center gap-3 px-4 pt-4 pb-2">
        {STATUS_ORDER.map((s) => (
          <div key={s} className="flex items-center gap-1">
            <span className={`h-2 w-2 rounded-full ${STATUS_CONFIG[s].color}`} />
            <span className="text-xs text-muted-foreground">
              {counts[s]} {STATUS_CONFIG[s].label}
            </span>
          </div>
        ))}
      </div>

      {/* Member list */}
      <div className="flex-1 overflow-auto divide-y divide-border">
        {sorted.map((member) => (
          <div
            key={member.id}
            className="flex items-center gap-3 px-4 py-2 hover:bg-accent/50 cursor-pointer transition-colors"
          >
            <div className="relative">
              <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/10 text-xs font-semibold text-primary">
                {member.avatar}
              </div>
              <span
                className={`absolute -bottom-0.5 -right-0.5 h-3 w-3 rounded-full border-2 border-card ${STATUS_CONFIG[member.status].color}`}
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

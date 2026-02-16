import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { cn } from '@/lib/cn'
import { useTimeTrackingStore } from '@/stores/timetracking'

const STATUS_MAP = {
  tracking: { label: 'Aktiv', dotClass: 'bg-emerald-500', animate: true },
  idle: { label: 'Inaktiv', dotClass: 'bg-gray-400', animate: false },
  absent: { label: 'Abwesend', dotClass: 'bg-amber-500', animate: false },
}

export default function TeamView() {
  const teamActivity = useTimeTrackingStore((s) => s.teamActivity)

  const sorted = [...teamActivity].sort((a, b) => {
    const order = { tracking: 0, idle: 1, absent: 2 }
    return order[a.status] - order[b.status]
  })

  return (
    <div className="p-6 max-w-3xl mx-auto space-y-4">
      <div className="flex items-center justify-between mb-2">
        <h3 className="text-sm font-semibold text-foreground">
          Team-Aktivität ({teamActivity.length} Mitglieder)
        </h3>
        <div className="flex items-center gap-4 text-xs text-muted-foreground">
          <span className="flex items-center gap-1.5">
            <span className="h-2 w-2 rounded-full bg-emerald-500" />
            Aktiv ({teamActivity.filter((t) => t.status === 'tracking').length})
          </span>
          <span className="flex items-center gap-1.5">
            <span className="h-2 w-2 rounded-full bg-gray-400" />
            Inaktiv ({teamActivity.filter((t) => t.status === 'idle').length})
          </span>
          <span className="flex items-center gap-1.5">
            <span className="h-2 w-2 rounded-full bg-amber-500" />
            Abwesend ({teamActivity.filter((t) => t.status === 'absent').length})
          </span>
        </div>
      </div>

      <div className="space-y-2">
        {sorted.map((member) => {
          const status = STATUS_MAP[member.status]
          return (
            <div
              key={member.userId}
              className="flex items-center gap-4 p-4 rounded-xl border border-border bg-card hover:border-primary/30 transition-colors"
            >
              {/* Avatar */}
              <div className="relative">
                <Avatar className="h-10 w-10">
                  <AvatarFallback className="text-sm">{member.userInitials}</AvatarFallback>
                </Avatar>
                <span className={cn(
                  'absolute -bottom-0.5 -right-0.5 h-3 w-3 rounded-full border-2 border-card',
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
                  <span className="font-medium text-sm text-foreground">{member.userName}</span>
                  <span className={cn(
                    'px-2 py-0.5 rounded-full text-[10px] font-medium',
                    member.status === 'tracking'
                      ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
                      : member.status === 'absent'
                        ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
                        : 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400',
                  )}>
                    {status.label}
                  </span>
                </div>
                {member.currentDescription && (
                  <p className="text-sm text-muted-foreground mt-0.5 truncate">
                    {member.currentDescription}
                  </p>
                )}
              </div>

              {/* Category + Time */}
              {member.currentCategory && (
                <div className="flex items-center gap-2 shrink-0">
                  <span
                    className="h-2.5 w-2.5 rounded-full"
                    style={{ backgroundColor: member.currentCategoryColor || '#6b7280' }}
                  />
                  <span className="text-sm text-muted-foreground">{member.currentCategory}</span>
                </div>
              )}
              {member.startedAt && (
                <span className="text-xs text-muted-foreground tabular-nums shrink-0">
                  seit {member.startedAt}
                </span>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}

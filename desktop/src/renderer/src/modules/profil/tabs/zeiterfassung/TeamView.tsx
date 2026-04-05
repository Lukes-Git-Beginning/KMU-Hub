import { useTranslation } from 'react-i18next'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { cn } from '@/lib/cn'
import { useTimeTrackingStore } from '@/stores/timetracking'

export default function TeamView() {
  const { t } = useTranslation()
  const teamActivity = useTimeTrackingStore((s) => s.teamActivity)

  const STATUS_MAP = {
    tracking: { label: t('profil.zeiterfassung.team.active'), dotClass: 'bg-success', animate: true },
    idle: { label: t('profil.zeiterfassung.team.idle'), dotClass: 'bg-gray-400', animate: false },
    absent: { label: t('profil.zeiterfassung.team.absent'), dotClass: 'bg-warning', animate: false },
  }

  const sorted = [...teamActivity].sort((a, b) => {
    const order = { tracking: 0, idle: 1, absent: 2 }
    return order[a.status] - order[b.status]
  })

  return (
    <div className="p-6 max-w-3xl mx-auto space-y-4">
      <div className="flex items-center justify-between mb-2">
        <h3 className="text-sm font-semibold text-foreground">
          {t('profil.zeiterfassung.team.title', { count: teamActivity.length })}
        </h3>
        <div className="flex items-center gap-4 text-xs text-muted-foreground">
          <span className="flex items-center gap-1.5">
            <span className="h-2 w-2 rounded-full bg-success" />
            {t('profil.zeiterfassung.team.active')} ({teamActivity.filter((m) => m.status === 'tracking').length})
          </span>
          <span className="flex items-center gap-1.5">
            <span className="h-2 w-2 rounded-full bg-gray-400" />
            {t('profil.zeiterfassung.team.idle')} ({teamActivity.filter((m) => m.status === 'idle').length})
          </span>
          <span className="flex items-center gap-1.5">
            <span className="h-2 w-2 rounded-full bg-warning" />
            {t('profil.zeiterfassung.team.absent')} ({teamActivity.filter((m) => m.status === 'absent').length})
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
                      ? 'bg-success-light text-success'
                      : member.status === 'absent'
                        ? 'bg-warning-light text-warning-foreground'
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
                  {t('profil.zeiterfassung.team.since', { time: member.startedAt })}
                </span>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}

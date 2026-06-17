/**
 * Birthdays widget — upcoming team birthdays.
 *
 * MSW-backed via useBirthdays() (GET /dashboard/birthdays). Backend gap:
 * EmployeeProfile has no `birthday` column yet — until then the handler serves
 * mock data, so the swap to a real endpoint is handler-only.
 */
import { memo } from 'react'
import { useTranslation } from 'react-i18next'
import { Cake, Gift, PartyPopper } from 'lucide-react'
import { useBirthdays } from '@/api/hooks/useBirthdays'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

function Birthdays(_props: WidgetProps) {
  const { t } = useTranslation()
  const { data: birthdays = [], isLoading } = useBirthdays()

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <div role="status" aria-label={t('common.loading')} className="h-5 w-5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center gap-2 px-4 pt-4 pb-2">
        <Cake className="h-4 w-4 text-pink-500" />
        <span className="text-xs font-medium text-muted-foreground">{t('dashboard.birthdays.upcoming')}</span>
      </div>

      {/* List */}
      <div className="flex-1 overflow-auto divide-y divide-border">
        {birthdays.map((bday) => {
          const isToday = bday.daysUntil === 0
          return (
            <div
              key={bday.employeeId}
              className={`flex items-center gap-3 px-4 py-2.5 transition-colors ${
                isToday ? 'bg-pink-500/5' : 'hover:bg-accent/50'
              }`}
            >
              <div className="relative">
                <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/10 text-xs font-semibold text-primary">
                  {bday.initials}
                </div>
                {isToday && (
                  <PartyPopper className="absolute -top-1 -right-1 h-3.5 w-3.5 text-pink-500" />
                )}
              </div>
              <div className="min-w-0 flex-1">
                <p className={`text-sm truncate ${isToday ? 'font-semibold text-foreground' : 'font-medium text-foreground'}`}>
                  {bday.name}
                </p>
                <p className="text-[10px] text-muted-foreground">{bday.department}</p>
              </div>
              <div className="text-right shrink-0">
                <p className={`text-xs font-medium ${isToday ? 'text-pink-500' : 'text-muted-foreground'}`}>
                  {bday.displayDate}
                </p>
                <p className="text-[10px] text-muted-foreground">
                  {isToday ? (
                    <span className="flex items-center gap-0.5 text-pink-500 font-medium">
                      <Gift className="h-2.5 w-2.5" /> {t('dashboard.birthdays.today')}
                    </span>
                  ) : (
                    t('dashboard.birthdays.inDays', { count: bday.daysUntil })
                  )}
                </p>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}

export default memo(Birthdays)

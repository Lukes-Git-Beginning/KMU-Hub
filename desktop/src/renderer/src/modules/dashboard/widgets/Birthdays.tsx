/**
 * Birthdays widget — upcoming team birthdays.
 *
 * TODO: Still uses mock data. EmployeeProfile has no `birthday` field yet.
 * Backend work needed:
 *   1. Add `birthday` (date) column to `employee_profiles` table + migration
 *   2. Expose birthday in EmployeeProfile API response
 *   3. Replace getUpcomingBirthdays() with useEmployees() + birthday filtering
 */
import { memo } from 'react'
import { Cake, Gift, PartyPopper } from 'lucide-react'
import { getUpcomingBirthdays } from '@/mocks/mock-db'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

const birthdays = getUpcomingBirthdays()

function Birthdays(_props: WidgetProps) {
  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center gap-2 px-4 pt-4 pb-2">
        <Cake className="h-4 w-4 text-pink-500" />
        <span className="text-xs font-medium text-muted-foreground">Nächste Geburtstage</span>
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
                      <Gift className="h-2.5 w-2.5" /> Heute!
                    </span>
                  ) : (
                    `in ${bday.daysUntil} Tagen`
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

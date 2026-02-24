/**
 * Birthdays widget — upcoming team birthdays.
 */
import { memo } from 'react'
import { Cake, Gift, PartyPopper } from 'lucide-react'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

interface MockBirthday {
  id: string
  name: string
  avatar: string
  date: string
  daysUntil: number
  department: string
}

const MOCK_BIRTHDAYS: MockBirthday[] = [
  { id: '1', name: 'Anna Schneider', avatar: 'AS', date: '24. Feb', daysUntil: 0, department: 'Projektleitung' },
  { id: '2', name: 'Thomas Mueller', avatar: 'TM', date: '28. Feb', daysUntil: 4, department: 'Vertrieb' },
  { id: '3', name: 'Sarah Braun', avatar: 'SB', date: '05. Mär', daysUntil: 9, department: 'Buchhaltung' },
  { id: '4', name: 'Jonas Hartmann', avatar: 'JH', date: '12. Mär', daysUntil: 16, department: 'Support' },
  { id: '5', name: 'Lena Fischer', avatar: 'LF', date: '20. Mär', daysUntil: 24, department: 'Design' },
]

function Birthdays(_props: WidgetProps) {
  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center gap-2 px-4 pt-4 pb-2">
        <Cake className="h-4 w-4 text-pink-500" />
        <span className="text-xs font-medium text-muted-foreground">Naechste Geburtstage</span>
      </div>

      {/* List */}
      <div className="flex-1 overflow-auto divide-y divide-border">
        {MOCK_BIRTHDAYS.map((bday) => {
          const isToday = bday.daysUntil === 0
          return (
            <div
              key={bday.id}
              className={`flex items-center gap-3 px-4 py-2.5 transition-colors ${
                isToday ? 'bg-pink-500/5' : 'hover:bg-accent/50'
              }`}
            >
              <div className="relative">
                <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/10 text-xs font-semibold text-primary">
                  {bday.avatar}
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
                  {bday.date}
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

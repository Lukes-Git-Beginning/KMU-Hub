/**
 * Absences widget — who is out today (sick, vacation, etc.).
 */
import { memo } from 'react'
import { Palmtree, Thermometer, Baby, GraduationCap, Home } from 'lucide-react'
import { ABSENCES_TODAY, type AbsenceType } from '@/mocks/mock-db'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

const TYPE_CONFIG: Record<AbsenceType, { icon: typeof Palmtree; label: string; color: string; bgColor: string }> = {
  urlaub: { icon: Palmtree, label: 'Urlaub', color: 'text-blue-600', bgColor: 'bg-blue-500/10' },
  krank: { icon: Thermometer, label: 'Krank', color: 'text-red-500', bgColor: 'bg-red-500/10' },
  elternzeit: { icon: Baby, label: 'Elternzeit', color: 'text-violet-600', bgColor: 'bg-violet-500/10' },
  weiterbildung: { icon: GraduationCap, label: 'Weiterbildung', color: 'text-amber-600', bgColor: 'bg-amber-500/10' },
  homeoffice: { icon: Home, label: 'Homeoffice', color: 'text-cyan-600', bgColor: 'bg-cyan-500/10' },
}

function Absences(_props: WidgetProps) {
  return (
    <div className="flex h-full flex-col">
      {/* Summary */}
      <div className="px-4 pt-4 pb-2">
        <p className="text-xs text-muted-foreground">
          <span className="font-semibold text-foreground">{ABSENCES_TODAY.length}</span> Personen heute abwesend
        </p>
      </div>

      {/* List */}
      <div className="flex-1 overflow-auto divide-y divide-border">
        {ABSENCES_TODAY.map((absence) => {
          const config = TYPE_CONFIG[absence.type]
          const Icon = config.icon
          return (
            <div
              key={absence.id}
              className="flex items-center gap-3 px-4 py-2.5 hover:bg-accent/50 cursor-pointer transition-colors"
            >
              <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary/10 text-xs font-semibold text-primary">
                {absence.initials}
              </div>
              <div className="min-w-0 flex-1">
                <p className="text-sm font-medium text-foreground truncate">{absence.name}</p>
                <div className="flex items-center gap-1.5 mt-0.5">
                  <span className={`flex items-center gap-1 text-[10px] rounded-full px-1.5 py-0.5 ${config.bgColor} ${config.color}`}>
                    <Icon className="h-2.5 w-2.5" />
                    {config.label}
                  </span>
                  <span className="text-[10px] text-muted-foreground">bis {absence.until}</span>
                </div>
              </div>
            </div>
          )
        })}
      </div>

      {ABSENCES_TODAY.length === 0 && (
        <div className="flex-1 flex items-center justify-center p-4">
          <p className="text-sm text-muted-foreground">Alle da!</p>
        </div>
      )}
    </div>
  )
}

export default memo(Absences)

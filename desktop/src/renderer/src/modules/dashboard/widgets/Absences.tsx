/**
 * Absences widget — who is out today (sick, vacation, etc.).
 */
import { memo, useMemo } from 'react'
import { Palmtree, Thermometer, Baby, GraduationCap, Home } from 'lucide-react'
import { useAbsenceCalendar } from '@/api/hooks/hr-hooks'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

type AbsenceType = 'urlaub' | 'krank' | 'elternzeit' | 'weiterbildung' | 'homeoffice'

const TYPE_CONFIG: Record<AbsenceType, { icon: typeof Palmtree; label: string; color: string; bgColor: string }> = {
  urlaub: { icon: Palmtree, label: 'Urlaub', color: 'text-blue-600', bgColor: 'bg-blue-500/10' },
  krank: { icon: Thermometer, label: 'Krank', color: 'text-destructive', bgColor: 'bg-destructive/10' },
  elternzeit: { icon: Baby, label: 'Elternzeit', color: 'text-violet-600', bgColor: 'bg-violet-500/10' },
  weiterbildung: { icon: GraduationCap, label: 'Weiterbildung', color: 'text-warning-foreground', bgColor: 'bg-warning/10' },
  homeoffice: { icon: Home, label: 'Homeoffice', color: 'text-cyan-600', bgColor: 'bg-cyan-500/10' },
}

/** Map leave type key from API to local AbsenceType. */
function mapLeaveType(key: string): AbsenceType {
  const lower = key.toLowerCase()
  if (lower.includes('urlaub') || lower.includes('vacation')) return 'urlaub'
  if (lower.includes('krank') || lower.includes('sick')) return 'krank'
  if (lower.includes('eltern') || lower.includes('parent')) return 'elternzeit'
  if (lower.includes('weiterbildung') || lower.includes('training')) return 'weiterbildung'
  if (lower.includes('home') || lower.includes('remote')) return 'homeoffice'
  return 'urlaub' // default fallback
}

/** Extract initials from a full name. */
function getInitials(name: string): string {
  const parts = name.trim().split(/\s+/)
  if (parts.length >= 2) {
    return `${parts[0][0]}${parts[parts.length - 1][0]}`.toUpperCase()
  }
  return name.substring(0, 2).toUpperCase()
}

/** Format a date string to dd.MM. */
function formatDate(dateStr: string): string {
  if (!dateStr) return ''
  const parts = dateStr.split('-')
  if (parts.length === 3) {
    return `${parseInt(parts[2], 10)}.${parseInt(parts[1], 10)}.`
  }
  return dateStr
}

function Absences(_props: WidgetProps) {
   
  const todayStr = useMemo(() => {
    const today = new Date()
    return `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}-${String(today.getDate()).padStart(2, '0')}`
  }, [])

  const { data: entries, isLoading } = useAbsenceCalendar({
    start_date: todayStr,
    end_date: todayStr,
  })

  const absences = useMemo(() => {
    if (!entries || !Array.isArray(entries)) return []
    return entries.map((entry) => ({
      id: entry.employeeId,
      name: entry.employeeName,
      initials: getInitials(entry.employeeName),
      type: mapLeaveType(entry.leaveTypeKey),
      until: formatDate(entry.endDate),
    }))
  }, [entries])

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <div role="status" aria-label="Laden" className="h-5 w-5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col">
      {/* Summary */}
      <div className="px-4 pt-4 pb-2">
        <p className="text-xs text-muted-foreground">
          <span className="font-semibold text-foreground">{absences.length}</span> Personen heute abwesend
        </p>
      </div>

      {/* List */}
      <div className="flex-1 overflow-auto divide-y divide-border">
        {absences.map((absence) => {
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
                  {absence.until && (
                    <span className="text-[10px] text-muted-foreground">bis {absence.until}</span>
                  )}
                </div>
              </div>
            </div>
          )
        })}
      </div>

      {absences.length === 0 && (
        <div className="flex-1 flex items-center justify-center p-4">
          <p className="text-sm text-muted-foreground">Alle da!</p>
        </div>
      )}
    </div>
  )
}

export default memo(Absences)

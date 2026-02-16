import { useState, useMemo } from 'react'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import type { TeamMember, HRRequest } from '@/stores/team'

const DAYS_SHORT = ['Mo', 'Di', 'Mi', 'Do', 'Fr']

const TYPE_COLORS: Record<string, { bg: string; text: string }> = {
  vacation: { bg: '#3d8abf', text: '#ffffff' },
  sick: { bg: '#bf3d3d', text: '#ffffff' },
  overtime: { bg: '#bf8a3d', text: '#ffffff' },
  doctor: { bg: '#1e7e74', text: '#ffffff' },
  homeoffice: { bg: '#4a7c6a', text: '#ffffff' },
  education: { bg: '#7c5a8a', text: '#ffffff' },
}

const TYPE_LABELS: Record<string, string> = {
  vacation: 'Urlaub',
  sick: 'Krank',
  overtime: 'UE',
  doctor: 'Arzt',
  homeoffice: 'HO',
  education: 'WB',
}

interface AbsenceCalendarProps {
  members: TeamMember[]
  requests: HRRequest[]
}

function getWeekDays(date: Date): Date[] {
  const d = new Date(date)
  const day = d.getDay()
  const diff = d.getDate() - day + (day === 0 ? -6 : 1)
  const monday = new Date(d.setDate(diff))
  return Array.from({ length: 5 }, (_, i) => {
    const dd = new Date(monday)
    dd.setDate(monday.getDate() + i)
    return dd
  })
}

function formatDateKey(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

function isInRange(date: string, start: string, end: string): boolean {
  return date >= start && date <= end
}

export function AbsenceCalendar({ members, requests }: AbsenceCalendarProps) {
  const [currentDate, setCurrentDate] = useState(new Date(2026, 1, 9))
  const [weeksToShow, setWeeksToShow] = useState(2)

  const weeks = useMemo(() => {
    const result: Date[][] = []
    const d = new Date(currentDate)
    for (let w = 0; w < weeksToShow; w++) {
      result.push(getWeekDays(d))
      d.setDate(d.getDate() + 7)
    }
    return result
  }, [currentDate, weeksToShow])

  const allDays = weeks.flat()

  const approvedRequests = requests.filter((r) => r.status === 'approved' || r.status === 'pending')

  const navigate = (dir: -1 | 1) => {
    const d = new Date(currentDate)
    d.setDate(d.getDate() + dir * 7)
    setCurrentDate(d)
  }

  const activeMembers = members.filter((m) => m.isActive)

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <button onClick={() => navigate(-1)} className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary transition-colors">
            <ChevronLeft className="h-4 w-4" />
          </button>
          <span className="text-sm font-medium text-foreground min-w-[200px] text-center">
            {allDays[0].toLocaleDateString('de-CH')} – {allDays[allDays.length - 1].toLocaleDateString('de-CH')}
          </span>
          <button onClick={() => navigate(1)} className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary transition-colors">
            <ChevronRight className="h-4 w-4" />
          </button>
        </div>
        <div className="flex items-center gap-2">
          {[2, 4].map((w) => (
            <button
              key={w}
              onClick={() => setWeeksToShow(w)}
              className={`rounded-md px-2.5 py-1 text-xs transition-colors ${
                weeksToShow === w ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:bg-secondary'
              }`}
            >
              {w} Wochen
            </button>
          ))}
        </div>
      </div>

      {/* Grid */}
      <div className="overflow-x-auto rounded-lg border border-border">
        <table className="w-full min-w-[700px] border-collapse">
          <thead>
            <tr className="bg-secondary/50">
              <th className="w-40 border-r border-border px-3 py-2 text-left text-[10px] uppercase tracking-wider text-muted-foreground">
                Mitarbeiter
              </th>
              {allDays.map((d, i) => {
                const today = formatDateKey(d) === formatDateKey(new Date())
                const isWeekStart = i % 5 === 0 && i > 0
                return (
                  <th
                    key={i}
                    className={`px-1 py-2 text-center ${isWeekStart ? 'border-l-2 border-border' : 'border-l border-border-muted'}`}
                  >
                    <p className="text-[9px] uppercase text-muted-foreground">{DAYS_SHORT[i % 5]}</p>
                    <p className={`text-[10px] mt-0.5 ${
                      today ? 'inline-flex h-5 w-5 items-center justify-center rounded-full bg-primary text-primary-foreground font-medium' : 'text-foreground'
                    }`}>
                      {d.getDate()}
                    </p>
                  </th>
                )
              })}
            </tr>
          </thead>
          <tbody>
            {activeMembers.map((member) => {
              const memberRequests = approvedRequests.filter((r) => r.memberId === member.id)

              return (
                <tr key={member.id} className="hover:bg-secondary/20 transition-colors">
                  <td className="border-r border-border border-t px-3 py-2">
                    <div className="flex items-center gap-2">
                      <div className="flex h-6 w-6 items-center justify-center rounded-full bg-primary-light text-[9px] font-medium text-primary">
                        {member.initials}
                      </div>
                      <div className="min-w-0">
                        <p className="text-xs font-medium text-foreground truncate">{member.firstName} {member.lastName}</p>
                        <p className="text-[9px] text-muted-foreground truncate">{member.department}</p>
                      </div>
                    </div>
                  </td>
                  {allDays.map((d, i) => {
                    const dateKey = formatDateKey(d)
                    const matchingReq = memberRequests.find((r) => isInRange(dateKey, r.startDate, r.endDate))
                    const isWeekStart = i % 5 === 0 && i > 0
                    const colors = matchingReq ? TYPE_COLORS[matchingReq.type] : null

                    return (
                      <td
                        key={i}
                        className={`border-t ${isWeekStart ? 'border-l-2 border-border' : 'border-l border-border-muted'} px-0.5 py-1 text-center`}
                      >
                        {matchingReq && colors && (
                          <div
                            className="mx-auto rounded px-1 py-0.5 text-[8px] font-medium"
                            style={{ backgroundColor: `${colors.bg}25`, color: colors.bg }}
                            title={`${TYPE_LABELS[matchingReq.type]}: ${matchingReq.reason}${matchingReq.status === 'pending' ? ' (ausstehend)' : ''}`}
                          >
                            {TYPE_LABELS[matchingReq.type]}
                            {matchingReq.status === 'pending' && '?'}
                          </div>
                        )}
                      </td>
                    )
                  })}
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>

      {/* Legend */}
      <div className="flex flex-wrap items-center gap-3">
        {Object.entries(TYPE_LABELS).map(([key, label]) => {
          const colors = TYPE_COLORS[key]
          return (
            <div key={key} className="flex items-center gap-1.5">
              <span className="h-3 w-3 rounded" style={{ backgroundColor: colors.bg }} />
              <span className="text-[10px] text-muted-foreground">{label === 'UE' ? 'Überstunden' : label === 'HO' ? 'Homeoffice' : label === 'WB' ? 'Weiterbildung' : label}</span>
            </div>
          )
        })}
        <div className="flex items-center gap-1.5 ml-2">
          <span className="text-[10px] text-muted-foreground italic">? = ausstehend</span>
        </div>
      </div>
    </div>
  )
}

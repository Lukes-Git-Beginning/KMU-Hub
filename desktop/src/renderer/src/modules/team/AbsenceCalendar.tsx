import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { ChevronLeft, ChevronRight, Loader2 } from 'lucide-react'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useAbsenceCalendar, useEmployees, useHRSettings } from '@/api/hooks/hr-hooks'
import type { AbsenceEntry } from '@/api/hr-types'

const DAYS_SHORT = ['Mo', 'Di', 'Mi', 'Do', 'Fr']

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

export function AbsenceCalendar() {
  const { t } = useTranslation()
  const [currentDate, setCurrentDate] = useState(() => new Date())
  const [weeksToShow, setWeeksToShow] = useState(2)
  const [departmentFilter, setDepartmentFilter] = useState<string>('all')

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
  const startDate = formatDateKey(allDays[0])
  const endDate = formatDateKey(allDays[allDays.length - 1])

  // Fetch absence calendar data from API
  const { data: absenceEntries, isLoading } = useAbsenceCalendar({
    start_date: startDate,
    end_date: endDate,
    department: departmentFilter !== 'all' ? departmentFilter : undefined,
  })

  // Fetch employees for department filter options
  const { data: employeesData } = useEmployees()

  // Fetch settings for reason visibility
  const { data: settings } = useHRSettings()

  // Extract unique departments from employees
  const departments = useMemo(() => {
    const employees = employeesData?.employees ?? []
    const depts = [...new Set(employees.map((e) => e.department).filter(Boolean))] as string[]
    return depts.sort()
  }, [employeesData])

  // Group absence entries by employee
  const employeeAbsences = useMemo(() => {
    const entries = absenceEntries ?? []
    const map = new Map<string, { name: string; department: string; entries: AbsenceEntry[] }>()
    for (const entry of entries) {
      if (!map.has(entry.employeeId)) {
        map.set(entry.employeeId, {
          name: entry.employeeName,
          department: entry.department,
          entries: [],
        })
      }
      map.get(entry.employeeId)!.entries.push(entry)
    }
    return [...map.entries()].sort(([, a], [, b]) => a.name.localeCompare(b.name))
  }, [absenceEntries])

  const navigate = (dir: -1 | 1) => {
    const d = new Date(currentDate)
    d.setDate(d.getDate() + dir * 7)
    setCurrentDate(d)
  }

  const showReasons = settings?.showAbsenceReason !== false

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <button onClick={() => navigate(-1)} className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary transition-colors">
            <ChevronLeft className="h-4 w-4" />
          </button>
          <span className="text-sm font-medium text-foreground min-w-[200px] text-center">
            {allDays[0].toLocaleDateString('de-DE')} – {allDays[allDays.length - 1].toLocaleDateString('de-DE')}
          </span>
          <button onClick={() => navigate(1)} className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary transition-colors">
            <ChevronRight className="h-4 w-4" />
          </button>
        </div>
        <div className="flex items-center gap-2">
          {/* Department filter */}
          <Select value={departmentFilter} onValueChange={setDepartmentFilter}>
            <SelectTrigger className="w-40 h-8 text-xs">
              <SelectValue placeholder={t('team.absence.departmentPlaceholder')} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{t('team.absence.allDepartments')}</SelectItem>
              {departments.map((dept) => (
                <SelectItem key={dept} value={dept}>{dept}</SelectItem>
              ))}
            </SelectContent>
          </Select>
          {/* Week toggle */}
          {[2, 4].map((w) => (
            <button
              key={w}
              onClick={() => setWeeksToShow(w)}
              className={`rounded-md px-2.5 py-1 text-xs transition-colors ${
                weeksToShow === w ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:bg-secondary'
              }`}
            >
              {t('team.absence.weeks', { count: w })}
            </button>
          ))}
        </div>
      </div>

      {/* Loading indicator */}
      {isLoading && (
        <div className="flex items-center justify-center py-8">
          <Loader2 className="h-6 w-6 animate-spin text-primary" />
        </div>
      )}

      {/* Grid */}
      {!isLoading && (
        <div className="overflow-x-auto rounded-lg border border-border">
          <table className="w-full min-w-[700px] border-collapse">
            <thead>
              <tr className="bg-secondary/50">
                <th className="w-40 border-r border-border px-3 py-2 text-left text-[10px] uppercase tracking-wider text-muted-foreground">
                  {t('team.absence.employee')}
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
              {employeeAbsences.length === 0 && (
                <tr>
                  <td colSpan={allDays.length + 1} className="py-8 text-center text-sm text-muted-foreground">
                    {t('team.absence.noAbsences')}
                  </td>
                </tr>
              )}
              {employeeAbsences.map(([employeeId, data]) => (
                <tr key={employeeId} className="hover:bg-secondary/20 transition-colors">
                  <td className="border-r border-border border-t px-3 py-2">
                    <div className="flex items-center gap-2">
                      <div className="flex h-6 w-6 items-center justify-center rounded-full bg-primary-light text-[9px] font-medium text-primary">
                        {data.name
                          .split(' ')
                          .map((n) => n[0])
                          .join('')
                          .slice(0, 2)
                          .toUpperCase()}
                      </div>
                      <div className="min-w-0">
                        <p className="text-xs font-medium text-foreground truncate">{data.name}</p>
                        <p className="text-[9px] text-muted-foreground truncate">{data.department}</p>
                      </div>
                    </div>
                  </td>
                  {allDays.map((d, i) => {
                    const dateKey = formatDateKey(d)
                    const matchingEntry = data.entries.find((e) => isInRange(dateKey, e.startDate, e.endDate))
                    const isWeekStart = i % 5 === 0 && i > 0

                    // When showAbsenceReason is disabled, show neutral gray
                    const barColor = matchingEntry
                      ? showReasons
                        ? matchingEntry.color
                        : '#9ca3af'
                      : null
                    const barLabel = matchingEntry
                      ? showReasons
                        ? matchingEntry.leaveTypeName
                        : t('team.absence.absent')
                      : null

                    return (
                      <td
                        key={i}
                        className={`border-t ${isWeekStart ? 'border-l-2 border-border' : 'border-l border-border-muted'} px-0.5 py-1 text-center`}
                      >
                        {matchingEntry && barColor && (
                          <div
                            className="mx-auto rounded px-1 py-0.5 text-[8px] font-medium"
                            style={{ backgroundColor: `${barColor}25`, color: barColor }}
                            title={showReasons ? matchingEntry.leaveTypeName : t('team.absence.absent')}
                          >
                            {barLabel}
                          </div>
                        )}
                      </td>
                    )
                  })}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Legend */}
      {showReasons && (
        <div className="flex flex-wrap items-center gap-3">
          {(() => {
            // Deduplicate leave type colors from entries
            const seen = new Set<string>()
            const legends: { key: string; label: string; color: string }[] = []
            for (const entry of absenceEntries ?? []) {
              if (!seen.has(entry.leaveTypeKey)) {
                seen.add(entry.leaveTypeKey)
                legends.push({
                  key: entry.leaveTypeKey,
                  label: entry.leaveTypeName,
                  color: entry.color,
                })
              }
            }
            return legends.map((l) => (
              <div key={l.key} className="flex items-center gap-1.5">
                <span className="h-3 w-3 rounded" style={{ backgroundColor: l.color }} />
                <span className="text-[10px] text-muted-foreground">{l.label}</span>
              </div>
            ))
          })()}
        </div>
      )}
    </div>
  )
}

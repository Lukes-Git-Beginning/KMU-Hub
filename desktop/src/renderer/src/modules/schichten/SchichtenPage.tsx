import { useState, useMemo, useCallback } from 'react'
import {
  Search,
  Plus,
  CalendarDays,
  Clock,
  Coffee,
  ArrowLeftRight,
  Palette,
  ChevronLeft,
  ChevronRight,
  Users,
  AlertTriangle,
  BarChart3,
  Timer,
  X,
  StickyNote,
  Check,
  XCircle,
  Sun,
  Sunset,
  Moon,
  FileDown,
  GripVertical,
  ShieldAlert,
  PartyPopper,
  Percent,
  UserCheck,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  useSchichtenStore,
  type ShiftTemplate,
  type ShiftAssignment,
  type ShiftSwapRequest,
} from '@/stores/schichten'
import { ItemActions, ConfirmDialog, EmptyState } from '@/components/shared'

// ============================================================
// Types
// ============================================================

type TabKey = 'wochenplan' | 'vorlagen' | 'anfragen' | 'verfuegbarkeit'

interface Employee {
  id: string
  name: string
  initials: string
  availability: 'available' | 'limited' | 'unavailable'
  role: string
}

interface AssignDialogState {
  open: boolean
  employeeId: string
  date: string
}

interface TemplateDialogState {
  open: boolean
}

interface DragState {
  active: boolean
  assignmentKey: string
  sourceEmpId: string
  sourceDate: string
  templateId: string
  templateName: string
}

interface ArbZGViolation {
  employeeId: string
  employeeName: string
  type: 'max_hours' | 'rest_period' | 'break_missing' | 'consecutive_days'
  message: string
  severity: 'warning' | 'error'
}

// ============================================================
// Constants & Mock Data
// ============================================================

const WEEKDAYS = ['Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa', 'So']

const swapStatusLabels: Record<string, string> = {
  pending: 'Ausstehend',
  approved: 'Genehmigt',
  rejected: 'Abgelehnt',
}

const swapStatusColors: Record<string, string> = {
  pending: 'bg-warning-light text-warning',
  approved: 'bg-success-light text-success',
  rejected: 'bg-error-light text-error',
}

const SHIFT_STYLE_MAP: Record<string, { bg: string; text: string; border: string; icon: typeof Sun }> = {
  'tpl-1': { bg: 'bg-info-light', text: 'text-info', border: 'border-info/20', icon: Sun },
  'tpl-2': { bg: 'bg-warning-light', text: 'text-warning', border: 'border-warning/20', icon: Sunset },
  'tpl-3': { bg: 'bg-primary-light', text: 'text-primary', border: 'border-primary/20', icon: Moon },
}

// 7.7: Surcharge rules (Zuschlaege)
const SURCHARGE_RULES: Record<string, { label: string; rate: string }> = {
  'tpl-3': { label: 'Nacht', rate: '+25%' },
}

const WEEKEND_SURCHARGE = { label: 'WE', rate: '+50%' }
const HOLIDAY_SURCHARGE = { label: 'Feiertag', rate: '+100%' }

// 7.10: German holidays (DE-first, configurable)
const GERMAN_HOLIDAYS_2026: Record<string, string> = {
  '2026-01-01': 'Neujahr',
  '2026-04-03': 'Karfreitag',
  '2026-04-06': 'Ostermontag',
  '2026-05-01': 'Tag der Arbeit',
  '2026-05-14': 'Christi Himmelfahrt',
  '2026-05-25': 'Pfingstmontag',
  '2026-10-03': 'Tag der Deutschen Einheit',
  '2026-12-25': '1. Weihnachtstag',
  '2026-12-26': '2. Weihnachtstag',
}

const EMPLOYEES: Employee[] = [
  { id: 'u-1', name: 'Thomas Keller', initials: 'TK', availability: 'available', role: 'Produktion' },
  { id: 'u-2', name: 'Lukas Brunner', initials: 'LB', availability: 'available', role: 'Produktion' },
  { id: 'u-3', name: 'Sandra Mueller', initials: 'SM', availability: 'limited', role: 'Logistik' },
  { id: 'u-4', name: 'Reto Aeschlimann', initials: 'RA', availability: 'available', role: 'Produktion' },
  { id: 'u-5', name: 'Daniel Frei', initials: 'DF', availability: 'available', role: 'Lager' },
  { id: 'u-6', name: 'Nicole Berger', initials: 'NB', availability: 'unavailable', role: 'Logistik' },
  { id: 'u-7', name: 'Marco Hartmann', initials: 'MH', availability: 'available', role: 'Produktion' },
  { id: 'u-8', name: 'Irene Graf', initials: 'IG', availability: 'limited', role: 'Verwaltung' },
]

// 7.12: Self-service availability grid mock
type AvailabilityLevel = 'green' | 'yellow' | 'red'
const AVAILABILITY_MOCK: Record<string, Record<string, AvailabilityLevel>> = {
  'u-1': {}, // Thomas — all available (green default)
  'u-3': { '1': 'yellow', '2': 'yellow', '3': 'red' }, // Sandra — limited some days
  'u-6': { '0': 'red', '1': 'red', '2': 'red', '3': 'red', '4': 'red', '5': 'red', '6': 'red' }, // Nicole — unavailable
  'u-8': { '3': 'yellow', '4': 'yellow', '5': 'red', '6': 'red' }, // Irene — limited/off weekends
}

const availabilityDot: Record<string, string> = {
  available: 'bg-success',
  limited: 'bg-warning',
  unavailable: 'bg-error',
}

const availabilityLabel: Record<string, string> = {
  available: 'Verfuegbar',
  limited: 'Eingeschraenkt',
  unavailable: 'Nicht verfuegbar',
}

const availabilityLevelColors: Record<AvailabilityLevel, { bg: string; text: string; label: string }> = {
  green: { bg: 'bg-success/20', text: 'text-success', label: 'Verfuegbar' },
  yellow: { bg: 'bg-warning/20', text: 'text-warning', label: 'Eingeschraenkt' },
  red: { bg: 'bg-error/20', text: 'text-error', label: 'Nicht verfuegbar' },
}

function buildMockAssignments(weekDates: string[]): ShiftAssignment[] {
  const assignments: ShiftAssignment[] = []
  let idCounter = 1

  const grid: (string | null)[][] = [
    ['tpl-1', 'tpl-1', 'tpl-1', null, 'tpl-1', 'tpl-1', null],
    ['tpl-2', 'tpl-2', null, 'tpl-2', 'tpl-2', null, null],
    [null, 'tpl-1', 'tpl-1', 'tpl-1', null, null, null],
    ['tpl-1', null, 'tpl-2', 'tpl-1', 'tpl-1', null, null],
    ['tpl-3', 'tpl-3', null, 'tpl-3', 'tpl-3', null, 'tpl-3'],
    [null, null, null, null, null, null, null],
    [null, 'tpl-2', 'tpl-2', 'tpl-2', null, 'tpl-2', null],
    ['tpl-1', null, null, 'tpl-1', 'tpl-1', null, null],
  ]

  const templateNames: Record<string, string> = {
    'tpl-1': 'Fruehschicht',
    'tpl-2': 'Spaetschicht',
    'tpl-3': 'Nachtschicht',
  }

  grid.forEach((row, empIdx) => {
    const emp = EMPLOYEES[empIdx]
    row.forEach((tplId, dayIdx) => {
      if (tplId && dayIdx < weekDates.length) {
        assignments.push({
          id: `mock-sa-${idCounter++}`,
          userId: emp.id,
          userName: emp.name,
          templateId: tplId,
          templateName: templateNames[tplId],
          date: weekDates[dayIdx],
          status: dayIdx < 2 ? 'confirmed' : 'scheduled',
        })
      }
    })
  })

  return assignments
}

const MOCK_SWAP_REQUESTS: ShiftSwapRequest[] = [
  {
    id: 'swap-1', assignmentId: 'mock-sa-3', requestedBy: 'Thomas Keller', swapWithUser: 'Reto Aeschlimann',
    status: 'pending', reason: 'Arzttermin am Mittwochmorgen, brauche Fruehschicht-Tausch', createdAt: '2026-02-14T10:30:00',
  },
  {
    id: 'swap-2', assignmentId: 'mock-sa-8', requestedBy: 'Lukas Brunner', swapWithUser: 'Marco Hartmann',
    status: 'pending', reason: 'Elternabend in der Schule, brauche freien Abend', createdAt: '2026-02-13T16:15:00',
  },
  {
    id: 'swap-3', assignmentId: 'mock-sa-12', requestedBy: 'Daniel Frei', swapWithUser: 'Lukas Brunner',
    status: 'approved', reason: 'Familienfeier am Wochenende', createdAt: '2026-02-12T09:00:00',
  },
]

// ============================================================
// Helpers
// ============================================================

function getWeekDates(offset: number): string[] {
  const today = new Date()
  const dayOfWeek = today.getDay()
  const monday = new Date(today)
  monday.setDate(today.getDate() - ((dayOfWeek + 6) % 7) + offset * 7)
  const dates: string[] = []
  for (let i = 0; i < 7; i++) {
    const d = new Date(monday)
    d.setDate(monday.getDate() + i)
    dates.push(d.toISOString().split('T')[0])
  }
  return dates
}

function getKW(dateStr: string): number {
  const date = new Date(dateStr + 'T00:00:00')
  const d = new Date(Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()))
  const dayNum = d.getUTCDay() || 7
  d.setUTCDate(d.getUTCDate() + 4 - dayNum)
  const yearStart = new Date(Date.UTC(d.getUTCFullYear(), 0, 1))
  return Math.ceil(((d.getTime() - yearStart.getTime()) / 86400000 + 1) / 7)
}

function formatDateRange(dates: string[]): string {
  if (dates.length === 0) return ''
  const first = new Date(dates[0] + 'T00:00:00')
  const last = new Date(dates[dates.length - 1] + 'T00:00:00')
  const opts: Intl.DateTimeFormatOptions = { day: '2-digit', month: 'short' }
  return `${first.toLocaleDateString('de-DE', opts)} – ${last.toLocaleDateString('de-DE', opts)} ${last.getFullYear()}`
}

function isToday(dateStr: string): boolean {
  return dateStr === new Date().toISOString().split('T')[0]
}

function isWeekend(dateStr: string): boolean {
  const d = new Date(dateStr + 'T00:00:00')
  const day = d.getDay()
  return day === 0 || day === 6
}

function isHoliday(dateStr: string): string | null {
  return GERMAN_HOLIDAYS_2026[dateStr] ?? null
}

function computeShiftHours(templateId: string): number {
  const hours: Record<string, number> = { 'tpl-1': 7.5, 'tpl-2': 7.5, 'tpl-3': 7.25 }
  return hours[templateId] ?? 8
}

// 7.7: Compute surcharge for a shift on a given date
function getSurchargeLabel(templateId: string, dateStr: string): { label: string; rate: string } | null {
  const holiday = isHoliday(dateStr)
  if (holiday) return HOLIDAY_SURCHARGE
  if (isWeekend(dateStr)) return WEEKEND_SURCHARGE
  return SURCHARGE_RULES[templateId] ?? null
}

// 7.8+7.9: ArbZG validation
function computeViolations(employees: Employee[], assignments: ShiftAssignment[], weekDates: string[]): ArbZGViolation[] {
  const violations: ArbZGViolation[] = []

  for (const emp of employees) {
    const empAssignments = assignments.filter((a) => a.userId === emp.id)
    const weeklyHours = empAssignments.reduce((s, a) => s + computeShiftHours(a.templateId), 0)

    // Max 48h/week (ArbZG §3)
    if (weeklyHours > 48) {
      violations.push({
        employeeId: emp.id, employeeName: emp.name, type: 'max_hours', severity: 'error',
        message: `${emp.name}: ${weeklyHours.toFixed(1)}h/Woche ueberschreitet 48h-Grenze (ArbZG §3)`,
      })
    } else if (weeklyHours > 40) {
      violations.push({
        employeeId: emp.id, employeeName: emp.name, type: 'max_hours', severity: 'warning',
        message: `${emp.name}: ${weeklyHours.toFixed(1)}h/Woche — ueber 40h Regelarbeitszeit`,
      })
    }

    // Rest period: 11h between shifts (ArbZG §5)
    const sortedByDate = [...empAssignments].sort((a, b) => a.date.localeCompare(b.date))
    for (let i = 1; i < sortedByDate.length; i++) {
      const prev = sortedByDate[i - 1]
      const curr = sortedByDate[i]
      // Spaetschicht (22:00) followed by Fruehschicht (06:00) next day = 8h rest
      if (prev.templateId === 'tpl-2' && curr.templateId === 'tpl-1') {
        const prevDate = new Date(prev.date + 'T00:00:00')
        const currDate = new Date(curr.date + 'T00:00:00')
        const dayDiff = (currDate.getTime() - prevDate.getTime()) / 86400000
        if (dayDiff === 1) {
          violations.push({
            employeeId: emp.id, employeeName: emp.name, type: 'rest_period', severity: 'error',
            message: `${emp.name}: Nur 8h Ruhezeit zwischen Spaet- und Fruehschicht am ${curr.date} (min. 11h, ArbZG §5)`,
          })
        }
      }
      // Nachtschicht (06:00) followed by anything same day = conflict
      if (prev.templateId === 'tpl-3' && prev.date === curr.date) {
        violations.push({
          employeeId: emp.id, employeeName: emp.name, type: 'rest_period', severity: 'error',
          message: `${emp.name}: Doppelschicht am ${curr.date} (Nacht + weitere Schicht)`,
        })
      }
    }

    // 7.9: Consecutive days > 6
    const workedDays = new Set(empAssignments.map((a) => a.date))
    let consecutive = 0
    for (const date of weekDates) {
      if (workedDays.has(date)) {
        consecutive++
        if (consecutive > 6) {
          violations.push({
            employeeId: emp.id, employeeName: emp.name, type: 'consecutive_days', severity: 'error',
            message: `${emp.name}: 7+ aufeinanderfolgende Arbeitstage (ArbZG §9)`,
          })
          break
        }
      } else {
        consecutive = 0
      }
    }

    // 7.9: Double booking (same person, same date, different shift)
    const dateMap = new Map<string, number>()
    for (const a of empAssignments) {
      dateMap.set(a.date, (dateMap.get(a.date) ?? 0) + 1)
    }
    for (const [date, count] of dateMap) {
      if (count > 1) {
        violations.push({
          employeeId: emp.id, employeeName: emp.name, type: 'rest_period', severity: 'error',
          message: `${emp.name}: Doppelbuchung am ${new Date(date + 'T00:00:00').toLocaleDateString('de-DE')}`,
        })
      }
    }
  }

  return violations
}

// ============================================================
// Component
// ============================================================

export default function SchichtenPage() {
  const { templates } = useSchichtenStore()

  // State
  const [tab, setTab] = useState<TabKey>('wochenplan')
  const [search, setSearch] = useState('')
  const [weekOffset, setWeekOffset] = useState(0)
  const [confirmDelete, setConfirmDelete] = useState<ShiftTemplate | null>(null)
  const [assignDialog, setAssignDialog] = useState<AssignDialogState>({ open: false, employeeId: '', date: '' })
  const [templateDialog, setTemplateDialog] = useState<TemplateDialogState>({ open: false })
  const [hoveredCell, setHoveredCell] = useState<string | null>(null)

  // 7.11: Drag state
  const [dragState, setDragState] = useState<DragState | null>(null)
  const [dropTarget, setDropTarget] = useState<string | null>(null)

  // Assign dialog form
  const [assignEmployee, setAssignEmployee] = useState('')
  const [assignTemplate, setAssignTemplate] = useState('')
  const [assignDate, setAssignDate] = useState('')
  const [assignNotes, setAssignNotes] = useState('')

  // Template dialog form
  const [newTplName, setNewTplName] = useState('')
  const [newTplStart, setNewTplStart] = useState('06:00')
  const [newTplEnd, setNewTplEnd] = useState('14:00')
  const [newTplBreak, setNewTplBreak] = useState('30')
  const [newTplColor, setNewTplColor] = useState('#3b82f6')

  // 7.12: Availability editing
  const [editingAvailability, setEditingAvailability] = useState(false)
  const [availabilityData, setAvailabilityData] = useState(AVAILABILITY_MOCK)

  const localSwapRequests = MOCK_SWAP_REQUESTS

  // Derived data
  const weekDates = useMemo(() => getWeekDates(weekOffset), [weekOffset])
  const kw = useMemo(() => getKW(weekDates[0]), [weekDates])
  const dateRange = useMemo(() => formatDateRange(weekDates), [weekDates])
  const localAssignments = useMemo(() => buildMockAssignments(weekDates), [weekDates])

  const filteredEmployees = useMemo(() => {
    if (!search) return EMPLOYEES
    const q = search.toLowerCase()
    return EMPLOYEES.filter((e) => e.name.toLowerCase().includes(q) || e.role.toLowerCase().includes(q))
  }, [search])

  const assignmentMap = useMemo(() => {
    const map = new Map<string, ShiftAssignment>()
    localAssignments.forEach((a) => map.set(`${a.userId}-${a.date}`, a))
    return map
  }, [localAssignments])

  const templateMap = useMemo(() => {
    const map = new Map<string, ShiftTemplate>()
    templates.forEach((t) => map.set(t.id, t))
    return map
  }, [templates])

  const pendingSwapCount = localSwapRequests.filter((r) => r.status === 'pending').length

  // 7.10: Holiday detection for week
  const weekHolidays = useMemo(() => {
    const map = new Map<string, string>()
    for (const date of weekDates) {
      const h = isHoliday(date)
      if (h) map.set(date, h)
    }
    return map
  }, [weekDates])

  // 7.8+7.9: ArbZG violations
  const violations = useMemo(() => computeViolations(EMPLOYEES, localAssignments, weekDates), [localAssignments, weekDates])
  const violationsByEmployee = useMemo(() => {
    const map = new Map<string, ArbZGViolation[]>()
    for (const v of violations) {
      if (!map.has(v.employeeId)) map.set(v.employeeId, [])
      map.get(v.employeeId)!.push(v)
    }
    return map
  }, [violations])
  const errorCount = violations.filter((v) => v.severity === 'error').length
  const warningCount = violations.filter((v) => v.severity === 'warning').length

  // KPI calculations
  const totalSlots = EMPLOYEES.length * 5
  const filledSlots = localAssignments.filter((a) => {
    const dayIdx = weekDates.indexOf(a.date)
    return dayIdx >= 0 && dayIdx < 5
  }).length
  const occupancyRate = totalSlots > 0 ? Math.round((filledSlots / totalSlots) * 100) : 0
  const openShifts = totalSlots - filledSlots
  const totalHoursThisWeek = localAssignments.reduce((sum, a) => sum + computeShiftHours(a.templateId), 0)

  // Handlers
  const handleCellClick = useCallback((employeeId: string, date: string) => {
    if (dragState) return
    const key = `${employeeId}-${date}`
    const existing = assignmentMap.get(key)
    if (existing) {
      const tpl = templateMap.get(existing.templateId)
      const surcharge = getSurchargeLabel(existing.templateId, date)
      toast.info(`${existing.templateName}: ${tpl?.startTime} – ${tpl?.endTime}${surcharge ? ` (${surcharge.rate} ${surcharge.label})` : ''}`, {
        description: `${existing.userName} am ${new Date(existing.date + 'T00:00:00').toLocaleDateString('de-DE', { weekday: 'long', day: '2-digit', month: 'long' })}`,
      })
    } else {
      setAssignEmployee(employeeId)
      setAssignTemplate('')
      setAssignDate(date)
      setAssignNotes('')
      setAssignDialog({ open: true, employeeId, date })
    }
  }, [assignmentMap, templateMap, dragState])

  // 7.11: Drag handlers
  const handleDragStart = useCallback((empId: string, date: string, assignment: ShiftAssignment) => {
    setDragState({
      active: true,
      assignmentKey: `${empId}-${date}`,
      sourceEmpId: empId,
      sourceDate: date,
      templateId: assignment.templateId,
      templateName: assignment.templateName,
    })
  }, [])

  const handleDragOver = useCallback((empId: string, date: string) => {
    if (!dragState) return
    const key = `${empId}-${date}`
    if (key !== dragState.assignmentKey) {
      setDropTarget(key)
    }
  }, [dragState])

  const handleDrop = useCallback((empId: string, date: string) => {
    if (!dragState) return
    const emp = EMPLOYEES.find((e) => e.id === empId)
    toast.success('Schicht verschoben', {
      description: `${dragState.templateName} → ${emp?.name} am ${new Date(date + 'T00:00:00').toLocaleDateString('de-DE')}`,
    })
    setDragState(null)
    setDropTarget(null)
  }, [dragState])

  const handleDragEnd = useCallback(() => {
    setDragState(null)
    setDropTarget(null)
  }, [])

  const handleAssignSubmit = () => {
    if (!assignEmployee || !assignTemplate || !assignDate) {
      toast.error('Bitte alle Pflichtfelder ausfuellen')
      return
    }
    const emp = EMPLOYEES.find((e) => e.id === assignEmployee)
    const tpl = templates.find((t) => t.id === assignTemplate)
    toast.success('Schicht zugewiesen', {
      description: `${tpl?.name} an ${emp?.name} am ${new Date(assignDate + 'T00:00:00').toLocaleDateString('de-DE')}`,
    })
    setAssignDialog({ open: false, employeeId: '', date: '' })
  }

  const handleTemplateSubmit = () => {
    if (!newTplName || !newTplStart || !newTplEnd) {
      toast.error('Bitte Name und Zeiten angeben')
      return
    }
    toast.success(`Vorlage "${newTplName}" erstellt`, {
      description: `${newTplStart} – ${newTplEnd}, ${newTplBreak} Min. Pause`,
    })
    setTemplateDialog({ open: false })
    setNewTplName(''); setNewTplStart('06:00'); setNewTplEnd('14:00'); setNewTplBreak('30'); setNewTplColor('#3b82f6')
  }

  const handleApproveSwap = (swap: ShiftSwapRequest) => {
    toast.success('Tausch genehmigt', { description: `${swap.requestedBy} <> ${swap.swapWithUser}` })
  }

  const handleRejectSwap = (swap: ShiftSwapRequest) => {
    toast.info('Tausch abgelehnt', { description: `Anfrage von ${swap.requestedBy}` })
  }

  const handleDeleteTemplate = (template: ShiftTemplate) => {
    setConfirmDelete(null)
    toast.success(`Vorlage "${template.name}" wurde geloescht`)
  }

  const handleOpenAssignDialog = () => {
    setAssignEmployee(''); setAssignTemplate(''); setAssignDate(weekDates[0]); setAssignNotes('')
    setAssignDialog({ open: true, employeeId: '', date: '' })
  }

  // 7.13: PDF export
  const handlePDFExport = () => {
    toast.success('PDF-Export gestartet', {
      description: `Wochenplan KW ${kw} wird generiert...`,
    })
  }

  // 7.12: Toggle availability
  const cycleAvailability = (empId: string, dayIdx: string) => {
    setAvailabilityData((prev) => {
      const empData = { ...(prev[empId] ?? {}) }
      const current = empData[dayIdx] ?? 'green'
      const next: AvailabilityLevel = current === 'green' ? 'yellow' : current === 'yellow' ? 'red' : 'green'
      if (next === 'green') {
        delete empData[dayIdx]
      } else {
        empData[dayIdx] = next
      }
      return { ...prev, [empId]: empData }
    })
  }

  const getTemplateActions = (template: ShiftTemplate) => [
    { label: 'Bearbeiten', onClick: () => toast.info(`Vorlage "${template.name}" bearbeiten`) },
    { separator: true as const, label: '', onClick: () => {} },
    { label: 'Loeschen', variant: 'destructive' as const, onClick: () => setConfirmDelete(template) },
  ]

  return (
    <div className="flex-1 overflow-y-auto p-6">
      {/* ── Header ── */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between mb-6 gap-4">
        <div>
          <h1 className="text-foreground">Schichtplanung</h1>
          <p className="text-sm text-muted-foreground">
            {EMPLOYEES.length} Mitarbeiter &middot; KW {kw} &middot;{' '}
            {pendingSwapCount > 0
              ? `${pendingSwapCount} offene Tausch-Anfragen`
              : 'Keine offenen Anfragen'}
          </p>
        </div>
        <div className="flex items-center gap-2">
          {/* 7.13: PDF Export */}
          <button
            onClick={handlePDFExport}
            className="flex items-center gap-2 rounded-lg border border-border px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
          >
            <FileDown className="h-4 w-4" />
            PDF-Export
          </button>
          <button
            onClick={handleOpenAssignDialog}
            className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            <Plus className="h-4 w-4" />
            Schicht zuweisen
          </button>
        </div>
      </div>

      {/* ── KPI Row ── */}
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-4 mb-6">
        {[
          {
            label: 'Besetzungsgrad', value: `${occupancyRate}%`, icon: BarChart3,
            color: occupancyRate >= 80 ? 'text-success' : occupancyRate >= 60 ? 'text-warning' : 'text-error',
            bg: occupancyRate >= 80 ? 'bg-success-light' : occupancyRate >= 60 ? 'bg-warning-light' : 'bg-error-light',
          },
          {
            label: 'Offene Schichten', value: `${openShifts}`, icon: AlertTriangle,
            color: openShifts <= 3 ? 'text-success' : 'text-warning',
            bg: openShifts <= 3 ? 'bg-success-light' : 'bg-warning-light',
          },
          {
            label: 'Tausch-Anfragen', value: `${pendingSwapCount}`, icon: ArrowLeftRight,
            color: pendingSwapCount === 0 ? 'text-muted-foreground' : 'text-info',
            bg: pendingSwapCount === 0 ? 'bg-secondary' : 'bg-info-light',
          },
          {
            label: 'Stunden / Woche', value: `${totalHoursThisWeek.toFixed(1)}h`, icon: Timer,
            color: 'text-primary', bg: 'bg-primary-light',
          },
          {
            label: 'ArbZG-Warnungen', value: `${errorCount + warningCount}`, icon: ShieldAlert,
            color: errorCount > 0 ? 'text-error' : warningCount > 0 ? 'text-warning' : 'text-success',
            bg: errorCount > 0 ? 'bg-error-light' : warningCount > 0 ? 'bg-warning-light' : 'bg-success-light',
          },
        ].map((stat) => {
          const Icon = stat.icon
          return (
            <div key={stat.label} className="rounded-lg border border-border bg-card p-4">
              <div className="flex items-center justify-between mb-2">
                <p className="text-xs text-muted-foreground">{stat.label}</p>
                <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${stat.bg}`}>
                  <Icon className={`h-4 w-4 ${stat.color}`} />
                </div>
              </div>
              <p className={`text-xl font-semibold ${stat.color}`}>{stat.value}</p>
            </div>
          )
        })}
      </div>

      {/* 7.8: ArbZG Violation Banner */}
      {violations.length > 0 && tab === 'wochenplan' && (
        <div className={`rounded-lg border p-3 mb-4 ${errorCount > 0 ? 'border-error/30 bg-error-light/30' : 'border-warning/30 bg-warning-light/30'}`}>
          <div className="flex items-start gap-2">
            <ShieldAlert className={`h-4 w-4 mt-0.5 flex-shrink-0 ${errorCount > 0 ? 'text-error' : 'text-warning'}`} />
            <div className="flex-1 min-w-0">
              <p className={`text-sm font-medium ${errorCount > 0 ? 'text-error' : 'text-warning'}`}>
                {errorCount > 0 ? `${errorCount} Verstoesse` : ''}{errorCount > 0 && warningCount > 0 ? ' + ' : ''}{warningCount > 0 ? `${warningCount} Warnungen` : ''} — Arbeitszeitgesetz
              </p>
              <div className="mt-1 space-y-0.5">
                {violations.slice(0, 3).map((v, i) => (
                  <p key={i} className="text-xs text-muted-foreground flex items-center gap-1">
                    {v.severity === 'error' ? <XCircle className="h-3 w-3 text-error" /> : <AlertTriangle className="h-3 w-3 text-warning" />}
                    {v.message}
                  </p>
                ))}
                {violations.length > 3 && (
                  <p className="text-xs text-muted-foreground">+{violations.length - 3} weitere</p>
                )}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* ── Tabs ── */}
      <div className="flex items-center gap-4 border-b border-border mb-6">
        {([
          { key: 'wochenplan' as const, label: 'Wochenplan', icon: CalendarDays },
          { key: 'vorlagen' as const, label: `Vorlagen (${templates.length})`, icon: Palette },
          { key: 'anfragen' as const, label: `Tausch-Anfragen (${pendingSwapCount})`, icon: ArrowLeftRight },
          { key: 'verfuegbarkeit' as const, label: 'Verfuegbarkeit', icon: UserCheck },
        ]).map((t) => {
          const Icon = t.icon
          return (
            <button
              key={t.key}
              onClick={() => setTab(t.key)}
              className={`flex items-center gap-1.5 border-b-2 px-1 pb-2 text-sm transition-colors ${
                tab === t.key
                  ? 'border-primary text-primary font-medium'
                  : 'border-transparent text-muted-foreground hover:text-foreground'
              }`}
            >
              <Icon className="h-3.5 w-3.5" />
              {t.label}
            </button>
          )
        })}
      </div>

      {/* ============================================================ */}
      {/* WOCHENPLAN TAB                                                */}
      {/* ============================================================ */}
      {tab === 'wochenplan' && (
        <>
          {/* Week Navigation */}
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-3">
              <button
                onClick={() => setWeekOffset((o) => o - 1)}
                className="flex h-8 w-8 items-center justify-center rounded-lg border border-border hover:bg-secondary transition-colors"
              >
                <ChevronLeft className="h-4 w-4 text-foreground" />
              </button>
              <div className="text-center min-w-[200px]">
                <span className="text-sm font-semibold text-foreground">KW {kw}</span>
                <span className="mx-2 text-muted-foreground">|</span>
                <span className="text-sm text-muted-foreground">{dateRange}</span>
              </div>
              <button
                onClick={() => setWeekOffset((o) => o + 1)}
                className="flex h-8 w-8 items-center justify-center rounded-lg border border-border hover:bg-secondary transition-colors"
              >
                <ChevronRight className="h-4 w-4 text-foreground" />
              </button>
              {weekOffset !== 0 && (
                <button
                  onClick={() => setWeekOffset(0)}
                  className="ml-1 rounded-lg border border-border px-3 py-1 text-xs text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
                >
                  Heute
                </button>
              )}
            </div>
            <div className="relative max-w-[240px]">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <input
                type="text"
                placeholder="Mitarbeiter suchen..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-1.5 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
          </div>

          {/* Shift Grid */}
          {filteredEmployees.length === 0 ? (
            <EmptyState
              icon={CalendarDays}
              title="Keine Mitarbeiter gefunden"
              description={search ? 'Passe deine Suche an' : 'Weise Schichten zu, um den Wochenplan zu fuellen'}
            />
          ) : (
            <div className="rounded-lg border border-border overflow-hidden">
              {/* Grid Header */}
              <div className="grid border-b border-border bg-card" style={{ gridTemplateColumns: '220px repeat(7, 1fr)' }}>
                <div className="flex items-center gap-2 px-4 py-3 border-r border-border">
                  <Users className="h-4 w-4 text-muted-foreground" />
                  <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Mitarbeiter</span>
                </div>
                {weekDates.map((date, i) => {
                  const today = isToday(date)
                  const holiday = weekHolidays.get(date)
                  const weekend = isWeekend(date)
                  const dateObj = new Date(date + 'T00:00:00')
                  return (
                    <div
                      key={date}
                      className={`flex flex-col items-center justify-center py-2 ${i < 6 ? 'border-r border-border' : ''} ${
                        holiday ? 'bg-error/5' : today ? 'bg-primary/5' : weekend ? 'bg-secondary/30' : ''
                      }`}
                    >
                      <span className={`text-xs font-semibold ${
                        holiday ? 'text-error' : today ? 'text-primary' : weekend ? 'text-muted-foreground' : 'text-foreground'
                      }`}>
                        {WEEKDAYS[i]}
                      </span>
                      <span className={`text-[11px] ${today ? 'text-primary font-medium' : 'text-muted-foreground'}`}>
                        {dateObj.toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit' })}
                      </span>
                      {today && <div className="mt-0.5 h-0.5 w-4 rounded-full bg-primary" />}
                      {/* 7.10: Holiday indicator */}
                      {holiday && (
                        <span className="mt-0.5 flex items-center gap-0.5 text-[9px] text-error font-medium">
                          <PartyPopper className="h-2.5 w-2.5" />
                          {holiday.length > 12 ? holiday.slice(0, 10) + '...' : holiday}
                        </span>
                      )}
                    </div>
                  )
                })}
              </div>

              {/* Grid Body — Employee Rows */}
              {filteredEmployees.map((emp, empIdx) => {
                const empViolations = violationsByEmployee.get(emp.id)
                const hasError = empViolations?.some((v) => v.severity === 'error')

                return (
                  <div
                    key={emp.id}
                    className={`grid ${empIdx < filteredEmployees.length - 1 ? 'border-b border-border-muted' : ''} ${
                      hasError ? 'bg-error/[0.02]' : ''
                    }`}
                    style={{ gridTemplateColumns: '220px repeat(7, 1fr)' }}
                  >
                    {/* Employee Name Cell */}
                    <div className="flex items-center gap-3 px-4 py-3 border-r border-border bg-card/50">
                      <div className="relative flex-shrink-0">
                        <div className="flex h-8 w-8 items-center justify-center rounded-full bg-secondary text-xs font-medium text-foreground">
                          {emp.initials}
                        </div>
                        <div
                          className={`absolute -bottom-0.5 -right-0.5 h-2.5 w-2.5 rounded-full border-2 border-card ${availabilityDot[emp.availability]}`}
                          title={availabilityLabel[emp.availability]}
                        />
                      </div>
                      <div className="min-w-0">
                        <div className="flex items-center gap-1.5">
                          <p className="text-sm font-medium text-foreground truncate">{emp.name}</p>
                          {hasError && <AlertTriangle className="h-3 w-3 text-error flex-shrink-0" />}
                        </div>
                        <p className="text-[11px] text-muted-foreground">{emp.role}</p>
                      </div>
                    </div>

                    {/* Day Cells */}
                    {weekDates.map((date, dayIdx) => {
                      const key = `${emp.id}-${date}`
                      const assignment = assignmentMap.get(key)
                      const template = assignment ? templateMap.get(assignment.templateId) : null
                      const style = assignment ? SHIFT_STYLE_MAP[assignment.templateId] : null
                      const today = isToday(date)
                      const holiday = weekHolidays.get(date)
                      const weekend = isWeekend(date)
                      const isHovered = hoveredCell === key
                      const ShiftIcon = style?.icon ?? Sun
                      const isDragSource = dragState?.assignmentKey === key
                      const isDropTarget = dropTarget === key
                      const surcharge = assignment ? getSurchargeLabel(assignment.templateId, date) : null

                      return (
                        <div
                          key={date}
                          className={`relative flex items-center justify-center px-1 py-2 cursor-pointer transition-colors ${
                            dayIdx < 6 ? 'border-r border-border-muted' : ''
                          } ${holiday ? 'bg-error/[0.03]' : today ? 'bg-primary/[0.02]' : weekend ? 'bg-secondary/10' : ''} ${
                            !assignment && isHovered ? 'bg-secondary/60' : ''
                          } ${isDragSource ? 'opacity-30' : ''} ${isDropTarget ? 'ring-2 ring-primary ring-inset bg-primary/10' : ''}`}
                          onClick={() => handleCellClick(emp.id, date)}
                          onMouseEnter={() => { setHoveredCell(key); if (dragState) handleDragOver(emp.id, date) }}
                          onMouseLeave={() => setHoveredCell(null)}
                          onMouseUp={() => { if (dragState) handleDrop(emp.id, date) }}
                        >
                          {assignment && template && style ? (
                            <div
                              className={`w-full rounded-md border px-2 py-1.5 transition-shadow ${style.bg} ${style.border} ${
                                isHovered ? 'shadow-sm ring-1 ring-border' : ''
                              }`}
                              onMouseDown={(e) => { e.preventDefault(); handleDragStart(emp.id, date, assignment) }}
                              onMouseUp={() => handleDragEnd()}
                            >
                              <div className={`flex items-center gap-1 ${style.text}`}>
                                {/* 7.11: Drag handle */}
                                <GripVertical className="h-3 w-3 flex-shrink-0 opacity-30 cursor-grab" />
                                <ShiftIcon className="h-3 w-3 flex-shrink-0" />
                                <span className="text-xs font-semibold truncate">{assignment.templateName}</span>
                              </div>
                              <div className={`text-[10px] mt-0.5 ${style.text} opacity-75`}>
                                {template.startTime} – {template.endTime}
                              </div>
                              {/* 7.7: Surcharge badge */}
                              {surcharge && (
                                <div className="mt-0.5 flex items-center gap-0.5">
                                  <Percent className="h-2.5 w-2.5 text-warning" />
                                  <span className="text-[9px] font-medium text-warning">{surcharge.rate} {surcharge.label}</span>
                                </div>
                              )}
                              {assignment.status === 'confirmed' && (
                                <div className="absolute top-1 right-1.5">
                                  <Check className={`h-3 w-3 ${style.text} opacity-50`} />
                                </div>
                              )}
                            </div>
                          ) : (
                            <div
                              className={`w-full h-10 rounded-md border border-dashed transition-colors ${
                                isDropTarget ? 'border-primary bg-primary/10' : isHovered
                                  ? 'border-primary/40 bg-primary/5'
                                  : 'border-border-muted'
                              }`}
                            >
                              {isHovered && !dragState && (
                                <div className="flex items-center justify-center h-full">
                                  <Plus className="h-3.5 w-3.5 text-primary/40" />
                                </div>
                              )}
                            </div>
                          )}
                        </div>
                      )
                    })}
                  </div>
                )
              })}

              {/* Legend */}
              <div className="flex items-center gap-5 px-4 py-2.5 border-t border-border bg-card/30 flex-wrap">
                <span className="text-[11px] text-muted-foreground font-medium mr-1">Legende:</span>
                {templates.map((tpl) => {
                  const style = SHIFT_STYLE_MAP[tpl.id]
                  if (!style) return null
                  const Icon = style.icon
                  return (
                    <div key={tpl.id} className="flex items-center gap-1.5">
                      <div className={`flex h-5 w-5 items-center justify-center rounded ${style.bg}`}>
                        <Icon className={`h-3 w-3 ${style.text}`} />
                      </div>
                      <span className="text-[11px] text-muted-foreground">
                        {tpl.name} ({tpl.startTime}–{tpl.endTime})
                      </span>
                    </div>
                  )
                })}
                <div className="flex items-center gap-1.5">
                  <Percent className="h-3 w-3 text-warning" />
                  <span className="text-[11px] text-muted-foreground">Zuschlag</span>
                </div>
                <div className="flex items-center gap-1.5">
                  <PartyPopper className="h-3 w-3 text-error" />
                  <span className="text-[11px] text-muted-foreground">Feiertag</span>
                </div>
                <div className="flex items-center gap-1.5 ml-auto">
                  <div className="h-2.5 w-2.5 rounded-full bg-success" />
                  <span className="text-[11px] text-muted-foreground">Verfuegbar</span>
                  <div className="h-2.5 w-2.5 rounded-full bg-warning ml-2" />
                  <span className="text-[11px] text-muted-foreground">Eingeschraenkt</span>
                  <div className="h-2.5 w-2.5 rounded-full bg-error ml-2" />
                  <span className="text-[11px] text-muted-foreground">Abwesend</span>
                </div>
              </div>
            </div>
          )}
        </>
      )}

      {/* ============================================================ */}
      {/* VORLAGEN TAB                                                  */}
      {/* ============================================================ */}
      {tab === 'vorlagen' && (
        <>
          <div className="flex items-center justify-between gap-3 mb-4">
            <div className="relative flex-1 max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <input
                type="text"
                placeholder="Vorlage suchen..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <button
              onClick={() => setTemplateDialog({ open: true })}
              className="flex items-center gap-2 rounded-lg border border-border px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
            >
              <Plus className="h-4 w-4" />
              Neue Vorlage
            </button>
          </div>

          {templates.length === 0 ? (
            <EmptyState
              icon={Palette}
              title="Keine Vorlagen"
              description="Erstelle Schichtvorlagen, um Schichten schneller zuzuweisen"
            />
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {templates
                .filter((t) => !search || t.name.toLowerCase().includes(search.toLowerCase()))
                .map((template) => {
                  const style = SHIFT_STYLE_MAP[template.id]
                  const Icon = style?.icon ?? Clock
                  const startH = parseInt(template.startTime.split(':')[0])
                  const startM = parseInt(template.startTime.split(':')[1])
                  const endH = parseInt(template.endTime.split(':')[0])
                  const endM = parseInt(template.endTime.split(':')[1])
                  let durationMin = (endH * 60 + endM) - (startH * 60 + startM)
                  if (durationMin < 0) durationMin += 24 * 60
                  const durationHours = Math.floor((durationMin - template.breakMinutes) / 60)
                  const durationRemainder = (durationMin - template.breakMinutes) % 60
                  const surcharge = SURCHARGE_RULES[template.id]

                  return (
                    <div
                      key={template.id}
                      className="rounded-lg border border-border bg-card p-4 transition-shadow hover:shadow-[var(--shadow-card-hover)]"
                    >
                      <div className="flex items-start justify-between mb-3">
                        <div className="flex items-center gap-3">
                          <div className={`flex h-11 w-11 items-center justify-center rounded-lg ${style?.bg ?? 'bg-secondary'}`}>
                            <Icon className={`h-5 w-5 ${style?.text ?? 'text-foreground'}`} />
                          </div>
                          <div>
                            <h4 className="text-sm font-semibold text-foreground">{template.name}</h4>
                            <div className="flex items-center gap-1.5 text-xs text-muted-foreground mt-0.5">
                              <Clock className="h-3 w-3" />
                              <span>{template.startTime} – {template.endTime}</span>
                            </div>
                          </div>
                        </div>
                        <ItemActions items={getTemplateActions(template)} />
                      </div>

                      <div className="flex items-center gap-4 border-t border-border-muted pt-3 text-xs text-muted-foreground">
                        <div className="flex items-center gap-1.5">
                          <Timer className="h-3 w-3" />
                          <span>{durationHours}h{durationRemainder > 0 ? ` ${durationRemainder}min` : ''} netto</span>
                        </div>
                        <div className="flex items-center gap-1.5">
                          <Coffee className="h-3 w-3" />
                          <span>{template.breakMinutes} Min. Pause</span>
                        </div>
                        {surcharge && (
                          <div className="flex items-center gap-1.5 text-warning">
                            <Percent className="h-3 w-3" />
                            <span>{surcharge.rate} {surcharge.label}</span>
                          </div>
                        )}
                        <div className="flex items-center gap-1.5 ml-auto">
                          <div className="h-3 w-3 rounded-full" style={{ backgroundColor: template.color }} />
                          <span>Farbe</span>
                        </div>
                      </div>
                    </div>
                  )
                })}
            </div>
          )}
        </>
      )}

      {/* ============================================================ */}
      {/* TAUSCH-ANFRAGEN TAB                                           */}
      {/* ============================================================ */}
      {tab === 'anfragen' && (
        <>
          <div className="flex items-center gap-3 mb-4">
            <div className="relative flex-1 max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <input
                type="text"
                placeholder="Anfrage suchen..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
          </div>

          {localSwapRequests.length === 0 ? (
            <EmptyState
              icon={ArrowLeftRight}
              title="Keine Anfragen"
              description="Es gibt aktuell keine Tauschanfragen"
            />
          ) : (
            <div className="space-y-3">
              {localSwapRequests
                .filter((r) => {
                  if (!search) return true
                  const q = search.toLowerCase()
                  return r.requestedBy.toLowerCase().includes(q) || r.swapWithUser.toLowerCase().includes(q)
                })
                .map((swap) => {
                  const reqEmp = EMPLOYEES.find((e) => e.name === swap.requestedBy)
                  const swapEmp = EMPLOYEES.find((e) => e.name === swap.swapWithUser)
                  return (
                    <div key={swap.id} className="rounded-lg border border-border bg-card p-4 transition-shadow hover:shadow-[var(--shadow-card-hover)]">
                      <div className="flex items-start justify-between gap-4">
                        <div className="flex-1">
                          <div className="flex items-center gap-3 mb-2">
                            <div className="flex items-center gap-2">
                              <div className="flex h-7 w-7 items-center justify-center rounded-full bg-secondary text-[11px] font-medium text-foreground">
                                {reqEmp?.initials ?? swap.requestedBy.split(' ').map((n) => n[0]).join('')}
                              </div>
                              <span className="text-sm font-medium text-foreground">{swap.requestedBy}</span>
                            </div>
                            <ArrowLeftRight className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                            <div className="flex items-center gap-2">
                              <div className="flex h-7 w-7 items-center justify-center rounded-full bg-secondary text-[11px] font-medium text-foreground">
                                {swapEmp?.initials ?? swap.swapWithUser.split(' ').map((n) => n[0]).join('')}
                              </div>
                              <span className="text-sm font-medium text-foreground">{swap.swapWithUser}</span>
                            </div>
                          </div>

                          <div className="flex items-center gap-2 mb-2">
                            <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${swapStatusColors[swap.status]}`}>
                              {swapStatusLabels[swap.status]}
                            </span>
                            <span className="text-[11px] text-muted-foreground">
                              Erstellt am {new Date(swap.createdAt).toLocaleDateString('de-DE', { day: '2-digit', month: 'long', year: 'numeric' })}
                            </span>
                          </div>

                          <p className="text-sm text-muted-foreground">
                            <StickyNote className="inline h-3.5 w-3.5 mr-1 -mt-0.5" />
                            {swap.reason}
                          </p>
                        </div>

                        {swap.status === 'pending' && (
                          <div className="flex flex-col gap-2 flex-shrink-0">
                            <button
                              onClick={() => handleApproveSwap(swap)}
                              className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
                            >
                              <Check className="h-3.5 w-3.5" />
                              Genehmigen
                            </button>
                            <button
                              onClick={() => handleRejectSwap(swap)}
                              className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
                            >
                              <XCircle className="h-3.5 w-3.5" />
                              Ablehnen
                            </button>
                          </div>
                        )}
                      </div>
                    </div>
                  )
                })}
            </div>
          )}
        </>
      )}

      {/* ============================================================ */}
      {/* VERFUEGBARKEIT TAB (7.12)                                     */}
      {/* ============================================================ */}
      {tab === 'verfuegbarkeit' && (
        <>
          <div className="flex items-center justify-between mb-4">
            <div>
              <p className="text-sm text-muted-foreground">
                Mitarbeiter koennen ihre Verfuegbarkeit pro Wochentag angeben.
                Klicke auf eine Zelle, um den Status zu aendern.
              </p>
            </div>
            <button
              onClick={() => {
                setEditingAvailability(!editingAvailability)
                if (editingAvailability) toast.success('Verfuegbarkeit gespeichert')
              }}
              className={`flex items-center gap-2 rounded-lg px-4 py-2 text-sm transition-colors ${
                editingAvailability
                  ? 'bg-success text-white hover:bg-success/90'
                  : 'bg-primary text-primary-foreground hover:bg-button-primary-hover'
              }`}
            >
              {editingAvailability ? <Check className="h-4 w-4" /> : <UserCheck className="h-4 w-4" />}
              {editingAvailability ? 'Speichern' : 'Bearbeiten'}
            </button>
          </div>

          <div className="rounded-lg border border-border overflow-hidden">
            {/* Header */}
            <div className="grid border-b border-border bg-card" style={{ gridTemplateColumns: '220px repeat(7, 1fr)' }}>
              <div className="flex items-center gap-2 px-4 py-3 border-r border-border">
                <Users className="h-4 w-4 text-muted-foreground" />
                <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Mitarbeiter</span>
              </div>
              {WEEKDAYS.map((day, i) => (
                <div key={i} className={`flex items-center justify-center py-3 text-xs font-semibold text-foreground ${i < 6 ? 'border-r border-border' : ''}`}>
                  {day}
                </div>
              ))}
            </div>

            {/* Body */}
            {EMPLOYEES.map((emp, empIdx) => (
              <div
                key={emp.id}
                className={`grid ${empIdx < EMPLOYEES.length - 1 ? 'border-b border-border-muted' : ''}`}
                style={{ gridTemplateColumns: '220px repeat(7, 1fr)' }}
              >
                <div className="flex items-center gap-3 px-4 py-3 border-r border-border bg-card/50">
                  <div className="flex h-8 w-8 items-center justify-center rounded-full bg-secondary text-xs font-medium text-foreground">
                    {emp.initials}
                  </div>
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-foreground truncate">{emp.name}</p>
                    <p className="text-[11px] text-muted-foreground">{emp.role}</p>
                  </div>
                </div>

                {WEEKDAYS.map((_, dayIdx) => {
                  const empAvail = availabilityData[emp.id] ?? {}
                  const level: AvailabilityLevel = empAvail[String(dayIdx)] ?? 'green'
                  const config = availabilityLevelColors[level]

                  return (
                    <div
                      key={dayIdx}
                      className={`flex items-center justify-center py-3 ${dayIdx < 6 ? 'border-r border-border-muted' : ''} ${
                        editingAvailability ? 'cursor-pointer hover:bg-secondary/30' : ''
                      } ${config.bg}`}
                      onClick={() => { if (editingAvailability) cycleAvailability(emp.id, String(dayIdx)) }}
                    >
                      <div className={`flex items-center gap-1.5 ${config.text}`}>
                        <div className={`h-3 w-3 rounded-full ${
                          level === 'green' ? 'bg-success' : level === 'yellow' ? 'bg-warning' : 'bg-error'
                        }`} />
                        <span className="text-[11px] font-medium">{config.label}</span>
                      </div>
                    </div>
                  )
                })}
              </div>
            ))}

            {/* Legend */}
            <div className="flex items-center gap-5 px-4 py-2.5 border-t border-border bg-card/30">
              <span className="text-[11px] text-muted-foreground font-medium">Legende:</span>
              {Object.entries(availabilityLevelColors).map(([level, config]) => (
                <div key={level} className="flex items-center gap-1.5">
                  <div className={`h-3 w-3 rounded-full ${
                    level === 'green' ? 'bg-success' : level === 'yellow' ? 'bg-warning' : 'bg-error'
                  }`} />
                  <span className="text-[11px] text-muted-foreground">{config.label}</span>
                </div>
              ))}
              {editingAvailability && (
                <span className="text-[11px] text-primary font-medium ml-auto">Klicke auf Zellen zum Aendern</span>
              )}
            </div>
          </div>
        </>
      )}

      {/* ============================================================ */}
      {/* DIALOG: Schicht zuweisen                                      */}
      {/* ============================================================ */}
      {assignDialog.open && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 bg-black/40" onClick={() => setAssignDialog({ open: false, employeeId: '', date: '' })} />
          <div className="relative z-10 w-full max-w-md rounded-lg border border-border bg-card p-6 shadow-xl">
            <div className="flex items-center justify-between mb-5">
              <h3 className="text-base font-semibold text-foreground">Schicht zuweisen</h3>
              <button onClick={() => setAssignDialog({ open: false, employeeId: '', date: '' })} className="flex h-7 w-7 items-center justify-center rounded-md hover:bg-secondary transition-colors">
                <X className="h-4 w-4 text-muted-foreground" />
              </button>
            </div>

            <div className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5">Mitarbeiter *</label>
                <select value={assignEmployee} onChange={(e) => setAssignEmployee(e.target.value)} className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring">
                  <option value="">Mitarbeiter waehlen...</option>
                  {EMPLOYEES.map((e) => <option key={e.id} value={e.id}>{e.name} ({e.role})</option>)}
                </select>
              </div>
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5">Schichtvorlage *</label>
                <select value={assignTemplate} onChange={(e) => setAssignTemplate(e.target.value)} className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring">
                  <option value="">Vorlage waehlen...</option>
                  {templates.map((t) => <option key={t.id} value={t.id}>{t.name} ({t.startTime} – {t.endTime})</option>)}
                </select>
              </div>
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5">Datum *</label>
                <input type="date" value={assignDate} onChange={(e) => setAssignDate(e.target.value)} className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring" />
              </div>
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5">Notizen</label>
                <textarea value={assignNotes} onChange={(e) => setAssignNotes(e.target.value)} placeholder="Optionale Anmerkungen..." rows={2} className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring resize-none" />
              </div>
            </div>

            <div className="flex items-center justify-end gap-2 mt-6">
              <button onClick={() => setAssignDialog({ open: false, employeeId: '', date: '' })} className="rounded-lg border border-border px-4 py-2 text-sm text-foreground hover:bg-secondary transition-colors">
                Abbrechen
              </button>
              <button onClick={handleAssignSubmit} className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors">
                Zuweisen
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ============================================================ */}
      {/* DIALOG: Vorlage erstellen                                     */}
      {/* ============================================================ */}
      {templateDialog.open && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 bg-black/40" onClick={() => setTemplateDialog({ open: false })} />
          <div className="relative z-10 w-full max-w-md rounded-lg border border-border bg-card p-6 shadow-xl">
            <div className="flex items-center justify-between mb-5">
              <h3 className="text-base font-semibold text-foreground">Neue Schichtvorlage</h3>
              <button onClick={() => setTemplateDialog({ open: false })} className="flex h-7 w-7 items-center justify-center rounded-md hover:bg-secondary transition-colors">
                <X className="h-4 w-4 text-muted-foreground" />
              </button>
            </div>

            <div className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5">Name *</label>
                <input type="text" value={newTplName} onChange={(e) => setNewTplName(e.target.value)} placeholder="z.B. Mittelschicht" className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring" />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1.5">Startzeit *</label>
                  <input type="time" value={newTplStart} onChange={(e) => setNewTplStart(e.target.value)} className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring" />
                </div>
                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1.5">Endzeit *</label>
                  <input type="time" value={newTplEnd} onChange={(e) => setNewTplEnd(e.target.value)} className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring" />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1.5">Pause (Minuten)</label>
                  <input type="number" min="0" max="120" value={newTplBreak} onChange={(e) => setNewTplBreak(e.target.value)} className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring" />
                </div>
                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1.5">Farbe</label>
                  <div className="flex items-center gap-2">
                    <input type="color" value={newTplColor} onChange={(e) => setNewTplColor(e.target.value)} className="h-9 w-9 rounded-lg border border-border cursor-pointer" />
                    <div className="flex gap-1.5">
                      {['#3b82f6', '#f59e0b', '#8b5cf6', '#10b981', '#ef4444', '#ec4899'].map((c) => (
                        <button
                          key={c}
                          onClick={() => setNewTplColor(c)}
                          className={`h-6 w-6 rounded-full border-2 transition-transform hover:scale-110 ${newTplColor === c ? 'border-foreground scale-110' : 'border-transparent'}`}
                          style={{ backgroundColor: c }}
                        />
                      ))}
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div className="flex items-center justify-end gap-2 mt-6">
              <button onClick={() => setTemplateDialog({ open: false })} className="rounded-lg border border-border px-4 py-2 text-sm text-foreground hover:bg-secondary transition-colors">
                Abbrechen
              </button>
              <button onClick={handleTemplateSubmit} className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors">
                Erstellen
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Confirm Delete Template */}
      <ConfirmDialog
        open={!!confirmDelete}
        onOpenChange={() => setConfirmDelete(null)}
        title="Vorlage loeschen?"
        description={`Die Vorlage "${confirmDelete?.name}" wird geloescht. Bestehende Zuweisungen bleiben erhalten.`}
        confirmLabel="Loeschen"
        variant="destructive"
        onConfirm={() => confirmDelete && handleDeleteTemplate(confirmDelete)}
      />
    </div>
  )
}

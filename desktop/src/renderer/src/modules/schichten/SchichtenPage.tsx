import { useState, useMemo, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { formatDate } from '@/lib/format'
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
  StickyNote,
  Check,
  XCircle,
  Sun,
  FileDown,
  FileSpreadsheet,
  GripVertical,
  ShieldAlert,
  PartyPopper,
  Percent,
  UserCheck,
  CalendarClock,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  type ShiftTemplate,
  type ShiftAssignment,
} from '@/stores/schichten'
import {
  useTemplatesList,
  useCreateTemplate,
  useUpdateTemplate,
  useDeleteTemplate,
  useApplyTemplate,
  useAssignEmployee,
  useUnassignEmployee,
  useCreateShift,
  usePublishShifts,
  useShiftsWithAssignments,
  useSwapRequests,
  useCreateSwapRequest,
  useApproveSwapRequest,
  useRejectSwapRequest,
  type SwapRequest,
} from '@/api/hooks/useSchichten'
import { useEmployees } from '@/api/hooks/hr-hooks'
import type { ShiftTemplate as ApiShiftTemplate } from '@/api/schichten-types'
import { ItemActions, ConfirmDialog, EmptyState, PageHeader, SortMenu, type SortDirection } from '@/components/shared'
import { useSchichtenPrefsStore } from '@/stores/schichtenPrefs'
import { useSchichtenTenantStore } from '@/stores/schichtenTenant'
import { ShiftDetailModal } from './ShiftDetailModal'
import {
  buildWeekCsv,
  buildWeekPlanPdf,
  csvDateStamp,
  downloadBlob,
} from './schichten-export'
import {
  WEEKDAYS,
  SHIFT_STYLE_MAP,
  SURCHARGE_RULES,
  isWeekend,
  isHoliday,
  computeShiftHours,
  getSurchargeLabel,
  getInitials,
  type ArbZGViolation,
  type SchichtenEmployee as Employee,
} from './schichten-shared'

// ============================================================
// Types
// ============================================================

type TabKey = 'wochenplan' | 'vorlagen' | 'anfragen' | 'verfügbarkeit'

interface AssignDialogState {
  open: boolean
  employeeId: string
  date: string
}

interface TemplateDialogState {
  open: boolean
  edit: ShiftTemplate | null
}

interface DragState {
  active: boolean
  assignmentKey: string
  sourceEmpId: string
  sourceDate: string
  templateId: string
  templateName: string
}

// ============================================================
// Constants & Mock Data
// ============================================================

const swapStatusLabelKeys: Record<string, string> = {
  pending: 'schichten.anfragen.status.pending',
  approved: 'schichten.anfragen.status.approved',
  rejected: 'schichten.anfragen.status.rejected',
}

const swapStatusColors: Record<string, string> = {
  pending: 'bg-warning-light text-warning',
  approved: 'bg-success-light text-success',
  rejected: 'bg-error-light text-error',
}

// Adapter: API template (start_hour/minute/duration) → UI template (startTime/endTime/breakMinutes/color)
// The API does not store color or breakMinutes — break falls back to the tenant default until backend adds them.
function adaptApiTemplate(t: ApiShiftTemplate, defaultBreakMinutes: number): ShiftTemplate {
  const endTotalMin = t.start_hour * 60 + t.start_minute + t.duration_minutes
  const endH = Math.floor(endTotalMin / 60) % 24
  const endM = endTotalMin % 60
  const pad = (n: number) => String(n).padStart(2, '0')
  return {
    id: t.id,
    name: t.name,
    startTime: `${pad(t.start_hour)}:${pad(t.start_minute)}`,
    endTime: `${pad(endH)}:${pad(endM)}`,
    color: '#3b82f6', // default — backend does not yet expose color
    breakMinutes: defaultBreakMinutes, // tenant default — backend does not yet expose breakMinutes
  }
}

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

const availabilityLabelKeys: Record<string, string> = {
  available: 'schichten.availability.available',
  limited: 'schichten.availability.limited',
  unavailable: 'schichten.availability.unavailable',
}

const availabilityLevelColorKeys: Record<AvailabilityLevel, { bg: string; text: string; labelKey: string }> = {
  green: { bg: 'bg-success/20', text: 'text-success', labelKey: 'schichten.availability.available' },
  yellow: { bg: 'bg-warning/20', text: 'text-warning', labelKey: 'schichten.availability.limited' },
  red: { bg: 'bg-error/20', text: 'text-error', labelKey: 'schichten.availability.unavailable' },
}

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

// Match shift RFC3339 start_time to the best template by start_hour, or return null
function matchTemplateByStartHour(startTime: string, templates: ShiftTemplate[]): ShiftTemplate | null {
  const hour = new Date(startTime).getUTCHours()
  // Find template whose startTime hour matches the shift's start hour
  const match = templates.find((t) => {
    const tHour = parseInt(t.startTime.split(':')[0], 10)
    return tHour === hour
  })
  return match ?? templates[0] ?? null
}

function computeViolations(
  employees: Employee[],
  assignments: ShiftAssignment[],
  weekDates: string[],
  maxWeeklyHours: number,
): ArbZGViolation[] {
  const violations: ArbZGViolation[] = []

  for (const emp of employees) {
    const empAssignments = assignments.filter((a) => a.userId === emp.id)
    const weeklyHours = empAssignments.reduce((s, a) => s + computeShiftHours(a.templateId), 0)

    // Weekly ceiling (tenant setting, legal max 48h — ArbZG §3)
    if (weeklyHours > maxWeeklyHours) {
      violations.push({
        employeeId: emp.id, employeeName: emp.name, type: 'max_hours', severity: 'error',
        message: `${emp.name}: ${weeklyHours.toFixed(1)}h/Woche überschreitet ${maxWeeklyHours}h-Grenze (ArbZG §3)`,
      })
    } else if (weeklyHours > Math.min(40, maxWeeklyHours)) {
      violations.push({
        employeeId: emp.id, employeeName: emp.name, type: 'max_hours', severity: 'warning',
        message: `${emp.name}: ${weeklyHours.toFixed(1)}h/Woche — über 40h Regelarbeitszeit`,
      })
    }

    // Rest period: 11h between shifts (ArbZG §5)
    const sortedByDate = [...empAssignments].sort((a, b) => a.date.localeCompare(b.date))
    for (let i = 1; i < sortedByDate.length; i++) {
      const prev = sortedByDate[i - 1]
      const curr = sortedByDate[i]
      // Spätschicht (22:00) followed by Frühschicht (06:00) next day = 8h rest
      if (prev.templateId === 'tpl-2' && curr.templateId === 'tpl-1') {
        const prevDate = new Date(prev.date + 'T00:00:00')
        const currDate = new Date(curr.date + 'T00:00:00')
        const dayDiff = (currDate.getTime() - prevDate.getTime()) / 86400000
        if (dayDiff === 1) {
          violations.push({
            employeeId: emp.id, employeeName: emp.name, type: 'rest_period', severity: 'error',
            message: `${emp.name}: Nur 8h Ruhezeit zwischen Spät- und Frühschicht am ${curr.date} (min. 11h, ArbZG §5)`,
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
          message: `${emp.name}: Doppelbuchung am ${formatDate(date + 'T00:00:00')}`,
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
  const { t } = useTranslation()

  // ── Module settings (personal prefs + tenant policy) ──
  const defaultTab = useSchichtenPrefsStore((s) => s.defaultTab)
  const showSurcharges = useSchichtenPrefsStore((s) => s.showSurcharges)
  const swapEnabled = useSchichtenTenantStore((s) => s.swapEnabled)
  const maxWeeklyHours = useSchichtenTenantStore((s) => s.maxWeeklyHours)
  const tenantBreakMinutes = useSchichtenTenantStore((s) => s.defaultBreakMinutes)

  // ── Queries ──
  const templatesQuery = useTemplatesList()
  const swapRequestsQuery = useSwapRequests()
  // Adapt API template shape (start_hour/minute/duration) to the UI ShiftTemplate type (startTime/endTime/color/breakMinutes)
  const templates: ShiftTemplate[] = useMemo(
    () => (templatesQuery.data?.templates ?? []).map((tpl) => adaptApiTemplate(tpl, tenantBreakMinutes)),
    [templatesQuery.data, tenantBreakMinutes],
  )
  const swapRequests: SwapRequest[] = swapRequestsQuery.data ?? []

  // ── Mutations ──
  const assignEmployeeMutation = useAssignEmployee()
  const unassignMutation = useUnassignEmployee()
  const createTemplateMutation = useCreateTemplate()
  const updateTemplateMutation = useUpdateTemplate()
  const deleteTemplateMutation = useDeleteTemplate()
  const applyTemplateMutation = useApplyTemplate()
  const approveSwapMutation = useApproveSwapRequest()
  const rejectSwapMutation = useRejectSwapRequest()
  const createSwapMutation = useCreateSwapRequest()
  const createShiftMutation = useCreateShift()
  const publishShiftsMutation = usePublishShifts()

  const swapStatusLabels = useMemo(
    () => Object.fromEntries(Object.entries(swapStatusLabelKeys).map(([k, v]) => [k, t(v)])),
    [t],
  )

  const availabilityLabel = useMemo(
    () => Object.fromEntries(Object.entries(availabilityLabelKeys).map(([k, v]) => [k, t(v)])),
    [t],
  )

  const availabilityLevelColors = useMemo(
    () => Object.fromEntries(
      Object.entries(availabilityLevelColorKeys).map(([k, v]) => [k, { bg: v.bg, text: v.text, label: t(v.labelKey) }])
    ) as Record<AvailabilityLevel, { bg: string; text: string; label: string }>,
    [t],
  )

  // State
  const [tab, setTab] = useState<TabKey>(defaultTab)
  const [search, setSearch] = useState('')
  const [weekOffset, setWeekOffset] = useState(0)
  const [confirmDelete, setConfirmDelete] = useState<ShiftTemplate | null>(null)
  const [assignDialog, setAssignDialog] = useState<AssignDialogState>({ open: false, employeeId: '', date: '' })
  const [templateDialog, setTemplateDialog] = useState<TemplateDialogState>({ open: false, edit: null })
  const [hoveredCell, setHoveredCell] = useState<string | null>(null)
  // Detail modal for an occupied grid cell (key: `${employeeId}-${date}`)
  const [detailKey, setDetailKey] = useState<string | null>(null)
  // Apply-template dialog (target week for the previously unused apply endpoint)
  const [applyTarget, setApplyTarget] = useState<ShiftTemplate | null>(null)
  const [applyWeek, setApplyWeek] = useState<'current' | 'next'>('current')
  // Week-grid employee sorting (field + direction, shared SortMenu)
  const [sortField, setSortField] = useState('name')
  const [sortDirection, setSortDirection] = useState<SortDirection>('asc')

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

  // Derived data — weekDates must be computed before hooks that depend on it
  const weekDates = useMemo(() => getWeekDates(weekOffset), [weekOffset])
  const kw = useMemo(() => getKW(weekDates[0]), [weekDates])
  const dateRange = useMemo(() => formatDateRange(weekDates), [weekDates])

  // ── Real data queries (hooks must be at top level, called after weekDates) ──
  const employeesQuery = useEmployees()
  const shiftsWithAssignments = useShiftsWithAssignments(
    weekDates[0] + 'T00:00:00Z',
    weekDates[6] + 'T23:59:59Z',
  )

  // Adapt API employees to the local Employee shape.
  // useEmployees() runs adaptEmployee → camelCase EmployeeProfile (userName/positionTitle).
  const employees: Employee[] = useMemo(() => {
    const raw = employeesQuery.data?.employees ?? []
    return raw.map((e) => ({
      id: e.id,
      name: e.userName ?? e.id,
      initials: getInitials(e.userName ?? e.id),
      availability: 'available' as const,
      role: e.positionTitle ?? e.department ?? 'Mitarbeiter',
    }))
  }, [employeesQuery.data])

  // Build ShiftAssignment[] (local store shape) from real API data
  const localAssignments: ShiftAssignment[] = useMemo(() => {
    return shiftsWithAssignments.data.map((swa) => {
      const date = swa.shiftStartTime.split('T')[0]
      const matchedTemplate = matchTemplateByStartHour(swa.shiftStartTime, templates)
      const templateId = matchedTemplate?.id ?? swa.shiftId
      const templateName = matchedTemplate?.name ?? swa.shiftTitle
      const empName = employees.find((e) => e.id === swa.employeeId)?.name ?? swa.employeeId
      return {
        id: swa.assignmentId,
        userId: swa.employeeId,
        userName: empName,
        templateId,
        templateName,
        date,
        status: swa.shiftStatus === 'published' ? 'confirmed' as const : 'scheduled' as const,
      }
    })
  }, [shiftsWithAssignments.data, templates, employees])

  // Map from employeeId-date → assignment for O(1) grid lookup
  // When multiple shifts fall on the same date for the same employee (edge case),
  // prefer 'confirmed' (published) over 'scheduled' (draft).
  const assignmentMap = useMemo(() => {
    const map = new Map<string, ShiftAssignment>()
    for (const a of localAssignments) {
      const key = `${a.userId}-${a.date}`
      const existing = map.get(key)
      if (!existing || a.status === 'confirmed') {
        map.set(key, a)
      }
    }
    return map
  }, [localAssignments])

  // Map from shiftId → { id, startTime, employeeIds } for resolving shiftId in assign dialog
  const shiftsByDateAndTemplate = useMemo(() => {
    // key: `date-templateId` → shiftId
    const map = new Map<string, string>()
    for (const swa of shiftsWithAssignments.shifts) {
      const date = swa.start_time.split('T')[0]
      const matchedTemplate = matchTemplateByStartHour(swa.start_time, templates)
      const templateId = matchedTemplate?.id ?? swa.id
      map.set(`${date}-${templateId}`, swa.id)
    }
    return map
  }, [shiftsWithAssignments.shifts, templates])

  // Weekly hours per employee (for the hours sort field)
  const employeeHours = useMemo(() => {
    const map = new Map<string, number>()
    for (const a of localAssignments) {
      map.set(a.userId, (map.get(a.userId) ?? 0) + computeShiftHours(a.templateId))
    }
    return map
  }, [localAssignments])

  const filteredEmployees = useMemo(() => {
    const q = search.toLowerCase()
    const filtered = search
      ? employees.filter((e) => e.name.toLowerCase().includes(q) || e.role.toLowerCase().includes(q))
      : employees
    const dir = sortDirection === 'asc' ? 1 : -1
    return [...filtered].sort((a, b) => {
      if (sortField === 'role') return dir * a.role.localeCompare(b.role, 'de')
      if (sortField === 'hours') return dir * ((employeeHours.get(a.id) ?? 0) - (employeeHours.get(b.id) ?? 0))
      return dir * a.name.localeCompare(b.name, 'de')
    })
  }, [search, employees, sortField, sortDirection, employeeHours])

  const templateMap = useMemo(() => {
    const map = new Map<string, ShiftTemplate>()
    templates.forEach((t) => map.set(t.id, t))
    return map
  }, [templates])

  const pendingSwapCount = swapRequests.filter((r) => r.status === 'pending').length

  // 7.10: Holiday detection for week
  const weekHolidays = useMemo(() => {
    const map = new Map<string, string>()
    for (const date of weekDates) {
      const h = isHoliday(date)
      if (h) map.set(date, h)
    }
    return map
  }, [weekDates])

  // 7.8+7.9: ArbZG violations (weekly ceiling from tenant settings)
  const violations = useMemo(
    () => computeViolations(employees, localAssignments, weekDates, maxWeeklyHours),
    [employees, localAssignments, weekDates, maxWeeklyHours],
  )
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
  const totalSlots = employees.length * 5
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
      // Occupied cell → full detail modal (shared/DetailModal) instead of an info toast
      setDetailKey(key)
    } else {
      setAssignEmployee(employeeId)
      setAssignTemplate('')
      setAssignDate(date)
      setAssignNotes('')
      setAssignDialog({ open: true, employeeId, date })
    }
  }, [assignmentMap, dragState])

  // ── Detail modal derived state ──
  const detailAssignment = detailKey ? (assignmentMap.get(detailKey) ?? null) : null
  const detailTemplate = detailAssignment ? (templateMap.get(detailAssignment.templateId) ?? null) : null
  const detailEmployee = detailAssignment ? employees.find((e) => e.id === detailAssignment.userId) : undefined
  const detailShiftId = detailAssignment
    ? shiftsByDateAndTemplate.get(`${detailAssignment.date}-${detailAssignment.templateId}`)
    : undefined
  const detailViolations = detailAssignment ? (violationsByEmployee.get(detailAssignment.userId) ?? []) : []

  const handleUnassignFromDetail = () => {
    if (!detailAssignment || !detailShiftId) return
    unassignMutation.mutate(
      { shiftId: detailShiftId, employeeId: detailAssignment.userId },
      {
        onSuccess: () => {
          toast.success(t('schichten.detail.entferntToast'), {
            description: `${detailAssignment.templateName} · ${detailAssignment.userName}`,
          })
          setDetailKey(null)
        },
        onError: () => {
          toast.error(t('schichten.detail.entferntFehler'))
        },
      },
    )
  }

  const handleCreateSwapFromDetail = (swapWithEmployeeId: string, reason: string) => {
    if (!detailAssignment || !detailShiftId) return
    createSwapMutation.mutate(
      {
        assignmentId: detailAssignment.id,
        shiftId: detailShiftId,
        requestedByEmployeeId: detailAssignment.userId,
        swapWithEmployeeId,
        reason: reason || undefined,
        idempotencyKey: crypto.randomUUID(),
      },
      {
        onSuccess: () => {
          toast.success(t('schichten.detail.tauschErstelltToast'), {
            description: t('schichten.detail.tauschErstelltHint'),
          })
          setDetailKey(null)
        },
        onError: () => {
          toast.error(t('schichten.detail.tauschFehler'))
        },
      },
    )
  }

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
    // Dropping on the source cell is a plain click, not a move — without this
    // guard every click on an occupied cell fired a pointless unassign→assign
    // API round-trip plus a misleading "moved" toast.
    if (`${empId}-${date}` === dragState.assignmentKey) {
      setDragState(null)
      setDropTarget(null)
      return
    }
    const emp = employees.find((e) => e.id === empId)
    // Resolve the real shiftId for the dragged template on the target date
    const shiftId = shiftsByDateAndTemplate.get(`${date}-${dragState.templateId}`)
    if (!shiftId) {
      // No shift exists for this slot — show optimistic toast only
      toast.success(t('schichten.toast.verschoben'), {
        description: `${dragState.templateName} → ${emp?.name} am ${formatDate(date + 'T00:00:00')}`,
      })
      setDragState(null)
      setDropTarget(null)
      return
    }
    // Unassign from source, assign to target
    unassignMutation.mutate(
      { shiftId, employeeId: dragState.sourceEmpId },
      {
        onSettled: () => {
          assignEmployeeMutation.mutate(
            { shiftId, employee_id: empId },
            {
              onSuccess: () => {
                toast.success(t('schichten.toast.verschoben'), {
                  description: `${dragState.templateName} → ${emp?.name} am ${formatDate(date + 'T00:00:00')}`,
                })
              },
              onError: () => {
                toast.error(t('schichten.toast.verschobenFehler', { defaultValue: 'Zuweisung fehlgeschlagen' }))
              },
            },
          )
        },
      },
    )
    setDragState(null)
    setDropTarget(null)
  }, [dragState, employees, shiftsByDateAndTemplate, unassignMutation, assignEmployeeMutation, t])

  const handleDragEnd = useCallback(() => {
    setDragState(null)
    setDropTarget(null)
  }, [])

  const handleAssignSubmit = () => {
    if (!assignEmployee || !assignTemplate || !assignDate) {
      toast.error(t('schichten.dialog.assign.errorRequired'))
      return
    }
    const emp = employees.find((e) => e.id === assignEmployee)
    const tpl = templates.find((tplItem) => tplItem.id === assignTemplate)

    // Try to resolve an existing shift for this date + template combination.
    // If none exists, create the shift first, then assign.
    const existingShiftId = shiftsByDateAndTemplate.get(`${assignDate}-${assignTemplate}`)

    if (existingShiftId) {
      assignEmployeeMutation.mutate(
        { shiftId: existingShiftId, employee_id: assignEmployee },
        {
          onSuccess: () => {
            toast.success(t('schichten.dialog.assign.successToast'), {
              description: `${tpl?.name} an ${emp?.name} am ${formatDate(assignDate + 'T00:00:00')}`,
            })
            setAssignDialog({ open: false, employeeId: '', date: '' })
          },
          onError: () => {
            toast.error(t('schichten.dialog.assign.errorRequired'))
          },
        },
      )
    } else {
      // No shift yet for this date+template — create one first, then assign
      if (!tpl) {
        toast.error(t('schichten.dialog.assign.errorRequired'))
        return
      }
      const [startH, startM] = tpl.startTime.split(':').map(Number)
      const [endH, endM] = tpl.endTime.split(':').map(Number)
      const pad = (n: number) => String(n).padStart(2, '0')
      const startTime = `${assignDate}T${pad(startH)}:${pad(startM)}:00Z`
      const endTime = `${assignDate}T${pad(endH)}:${pad(endM)}:00Z`
      createShiftMutation.mutate(
        { title: tpl.name, start_time: startTime, end_time: endTime },
        {
          onSuccess: (data) => {
            const newShiftId = data.shift.id
            assignEmployeeMutation.mutate(
              { shiftId: newShiftId, employee_id: assignEmployee },
              {
                onSuccess: () => {
                  toast.success(t('schichten.dialog.assign.successToast'), {
                    description: `${tpl.name} an ${emp?.name} am ${formatDate(assignDate + 'T00:00:00')}`,
                  })
                  setAssignDialog({ open: false, employeeId: '', date: '' })
                },
                onError: () => {
                  toast.error(t('schichten.dialog.assign.errorRequired'))
                },
              },
            )
          },
          onError: () => {
            toast.error(t('schichten.dialog.assign.errorRequired'))
          },
        },
      )
    }
  }

  // Open the template dialog in create or edit mode — seeding the form on open
  // avoids the stale-state bug when switching between templates.
  const openTemplateDialog = (edit: ShiftTemplate | null) => {
    setNewTplName(edit?.name ?? '')
    setNewTplStart(edit?.startTime ?? '06:00')
    setNewTplEnd(edit?.endTime ?? '14:00')
    setNewTplBreak(String(edit?.breakMinutes ?? tenantBreakMinutes))
    setNewTplColor(edit?.color ?? '#3b82f6')
    setTemplateDialog({ open: true, edit })
  }

  const handleTemplateSubmit = () => {
    if (!newTplName || !newTplStart || !newTplEnd) {
      toast.error(t('schichten.dialog.template.errorRequired'))
      return
    }
    const [startH, startM] = newTplStart.split(':').map(Number)
    const [endH, endM] = newTplEnd.split(':').map(Number)
    let durationMin = (endH * 60 + endM) - (startH * 60 + startM)
    if (durationMin < 0) durationMin += 24 * 60

    const closeAndReset = () => {
      setTemplateDialog({ open: false, edit: null })
      setNewTplName(''); setNewTplStart('06:00'); setNewTplEnd('14:00'); setNewTplBreak('30'); setNewTplColor('#3b82f6')
    }

    if (templateDialog.edit) {
      // Edit mode — wires the previously unused updateTemplate mutation
      updateTemplateMutation.mutate(
        {
          id: templateDialog.edit.id,
          name: newTplName,
          start_hour: startH,
          start_minute: startM,
          duration_minutes: durationMin,
        },
        {
          onSuccess: () => {
            toast.success(t('schichten.dialog.template.updatedToast', { name: newTplName }))
            closeAndReset()
          },
          onError: () => {
            toast.error(t('schichten.dialog.template.errorRequired'))
          },
        },
      )
      return
    }

    createTemplateMutation.mutate(
      {
        name: newTplName,
        day_of_week: 0, // Not enforced per template in UI — defaults to 0 (Sunday/any)
        start_hour: startH,
        start_minute: startM,
        duration_minutes: durationMin,
      },
      {
        onSuccess: () => {
          toast.success(t('schichten.dialog.template.successToast', { name: newTplName }), {
            description: t('schichten.dialog.template.successPause', { start: newTplStart, end: newTplEnd, break: newTplBreak }),
          })
          closeAndReset()
        },
        onError: () => {
          toast.error(t('schichten.dialog.template.errorRequired'))
        },
      },
    )
  }

  // Apply a template to a week — wires the previously unused applyTemplate endpoint
  const handleApplyTemplate = () => {
    if (!applyTarget) return
    const offset = applyWeek === 'next' ? weekOffset + 1 : weekOffset
    const targetDates = getWeekDates(offset)
    applyTemplateMutation.mutate(
      {
        id: applyTarget.id,
        range_start: targetDates[0] + 'T00:00:00Z',
        range_end: targetDates[6] + 'T23:59:59Z',
      },
      {
        onSuccess: (data) => {
          toast.success(t('schichten.vorlagen.applyToast', { name: applyTarget.name }), {
            description: t('schichten.vorlagen.applyToastHint', { count: data.created_count }),
          })
          setApplyTarget(null)
        },
        onError: () => {
          toast.error(t('schichten.error.loadFailed'))
        },
      },
    )
  }

  const handleApproveSwap = (swap: SwapRequest) => {
    approveSwapMutation.mutate(swap.id, {
      onSuccess: () => {
        toast.success(t('schichten.toast.tauschGenehmigt'), {
          description: `${swap.requestedByEmployeeId} <> ${swap.swapWithEmployeeId}`,
        })
      },
      onError: () => {
        toast.error(t('schichten.swap.error.approveFailed'))
      },
    })
  }

  const handleRejectSwap = (swap: SwapRequest) => {
    rejectSwapMutation.mutate(swap.id, {
      onSuccess: () => {
        toast.info(t('schichten.toast.tauschAbgelehnt'), {
          description: `Anfrage von ${swap.requestedByEmployeeId}`,
        })
      },
      onError: () => {
        toast.error(t('schichten.swap.error.rejectFailed'))
      },
    })
  }

  const handleDeleteTemplate = (template: ShiftTemplate) => {
    setConfirmDelete(null)
    deleteTemplateMutation.mutate(template.id, {
      onSuccess: () => {
        toast.success(t('schichten.delete.successToast', { name: template.name }))
      },
      onError: () => {
        toast.error(t('schichten.delete.errorToast', { name: template.name }))
      },
    })
  }

  const handleOpenAssignDialog = () => {
    setAssignEmployee(''); setAssignTemplate(''); setAssignDate(weekDates[0]); setAssignNotes('')
    setAssignDialog({ open: true, employeeId: '', date: '' })
  }

  // Publish all draft shifts for the current week
  const handlePublishWeek = () => {
    publishShiftsMutation.mutate(
      { from: weekDates[0] + 'T00:00:00Z', to: weekDates[6] + 'T23:59:59Z' },
      {
        onSuccess: () => {
          toast.success(t('schichten.toast.veroeffentlicht'), {
            description: t('schichten.toast.veroeffentlichtBeschreibung', { kw }),
          })
        },
        onError: () => {
          toast.error(t('schichten.error.loadFailed'))
        },
      },
    )
  }

  // 7.13: PDF export — real download (market layout: header, per-employee shifts, totals)
  const handlePDFExport = () => {
    const blob = buildWeekPlanPdf(
      kw,
      dateRange,
      filteredEmployees,
      localAssignments,
      templateMap,
      { occupancyRate, totalHours: totalHoursThisWeek },
      {
        title: t('schichten.export.pdfTitle'),
        employees: t('schichten.wochenplan.mitarbeiter'),
        hours: t('schichten.kpi.stundenWoche'),
        occupancy: t('schichten.kpi.besetzungsgrad'),
        noShifts: t('schichten.export.keineSchichten'),
        draft: t('schichten.detail.status.entwurf'),
        published: t('schichten.detail.status.veroeffentlicht'),
      },
    )
    downloadBlob(blob, `dienstplan-kw${kw}-${csvDateStamp()}.pdf`)
    toast.success(t('schichten.toast.pdfExport'), {
      description: t('schichten.toast.pdfGeneriert', { kw }),
    })
  }

  // CSV export — one row per assignment (market standard)
  const handleCsvExport = () => {
    const csv = buildWeekCsv(localAssignments, employees, templateMap)
    // BOM prefix keeps DACH-Excel from mangling umlauts
    downloadBlob(
      new Blob(['﻿' + csv], { type: 'text/csv;charset=utf-8' }),
      `schichten-kw${kw}-${csvDateStamp()}.csv`,
    )
    toast.success(t('schichten.export.csvToast', { kw }))
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
    { label: t('schichten.vorlagen.actions.bearbeiten'), onClick: () => openTemplateDialog(template) },
    { label: t('schichten.vorlagen.actions.anwenden'), onClick: () => { setApplyWeek('current'); setApplyTarget(template) } },
    { separator: true as const, label: '', onClick: () => {} },
    { label: t('schichten.vorlagen.actions.loeschen'), variant: 'destructive' as const, onClick: () => setConfirmDelete(template) },
  ]

  return (
    <div className="flex-1 overflow-y-auto p-6">
      <PageHeader
        title={t('schichten.page.title')}
        description={t('schichten.page.description', {
          employees: employees.length,
          kw,
          swapInfo: pendingSwapCount > 0 ? t('schichten.page.offeneAnfragen', { count: pendingSwapCount }) : t('schichten.page.keineAnfragen'),
        })}
        icon={CalendarClock}
        moduleId="schichten"
        className="mb-6"
        actions={
          <div className="flex items-center gap-2">
            <button
              onClick={handleCsvExport}
              className="flex items-center gap-2 rounded-xl border border-border px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
            >
              <FileSpreadsheet className="h-4 w-4" />
              {t('schichten.actions.csvExport')}
            </button>
            <button
              onClick={handlePDFExport}
              className="flex items-center gap-2 rounded-xl border border-border px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
            >
              <FileDown className="h-4 w-4" />
              {t('schichten.actions.pdfExport')}
            </button>
            <button
              onClick={handlePublishWeek}
              disabled={publishShiftsMutation.isPending}
              className="flex items-center gap-2 rounded-xl border border-border px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors disabled:opacity-50"
            >
              <CalendarClock className="h-4 w-4" />
              {t('schichten.actions.wochePublizieren')}
            </button>
            <button
              onClick={handleOpenAssignDialog}
              className="flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
            >
              <Plus className="h-4 w-4" />
              {t('schichten.actions.schichtZuweisen')}
            </button>
          </div>
        }
      />

      {/* ── KPI Row ── */}
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-4 mb-6">
        {[
          {
            label: t('schichten.kpi.besetzungsgrad'), value: `${occupancyRate}%`, icon: BarChart3,
            color: occupancyRate >= 80 ? 'text-success' : occupancyRate >= 60 ? 'text-warning' : 'text-error',
            bg: occupancyRate >= 80 ? 'bg-success-light' : occupancyRate >= 60 ? 'bg-warning-light' : 'bg-error-light',
          },
          {
            label: t('schichten.kpi.offeneSchichten'), value: `${openShifts}`, icon: AlertTriangle,
            color: openShifts <= 3 ? 'text-success' : 'text-warning',
            bg: openShifts <= 3 ? 'bg-success-light' : 'bg-warning-light',
          },
          {
            label: t('schichten.kpi.tauschAnfragen'), value: `${pendingSwapCount}`, icon: ArrowLeftRight,
            color: pendingSwapCount === 0 ? 'text-muted-foreground' : 'text-info',
            bg: pendingSwapCount === 0 ? 'bg-secondary' : 'bg-info-light',
          },
          {
            label: t('schichten.kpi.stundenWoche'), value: `${totalHoursThisWeek.toFixed(1)}h`, icon: Timer,
            color: 'text-primary', bg: 'bg-primary-light',
          },
          {
            label: t('schichten.kpi.arbzgWarnungen'), value: `${errorCount + warningCount}`, icon: ShieldAlert,
            color: errorCount > 0 ? 'text-error' : warningCount > 0 ? 'text-warning' : 'text-success',
            bg: errorCount > 0 ? 'bg-error-light' : warningCount > 0 ? 'bg-warning-light' : 'bg-success-light',
          },
        ].map((stat) => {
          const Icon = stat.icon
          return (
            <div key={stat.label} className="rounded-xl border border-border bg-card p-4 hover:shadow-md hover:-translate-y-0.5 transition-all duration-200">
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
                {errorCount > 0 ? t('schichten.arbzg.verstoss', { count: errorCount }) : ''}{errorCount > 0 && warningCount > 0 ? ' + ' : ''}{warningCount > 0 ? t('schichten.arbzg.warnungen', { count: warningCount }) : ''} — {t('schichten.arbzg.label')}
              </p>
              <div className="mt-1 space-y-0.5">
                {violations.slice(0, 3).map((v, i) => (
                  <p key={i} className="text-xs text-muted-foreground flex items-center gap-1">
                    {v.severity === 'error' ? <XCircle className="h-3 w-3 text-error" /> : <AlertTriangle className="h-3 w-3 text-warning" />}
                    {v.message}
                  </p>
                ))}
                {violations.length > 3 && (
                  <p className="text-xs text-muted-foreground">{t('schichten.arbzg.weitere', { count: violations.length - 3 })}</p>
                )}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* ── Tabs ── */}
      <div className="flex items-center gap-4 border-b border-border mb-6">
        {([
          { key: 'wochenplan' as const, label: t('schichten.tabs.wochenplan'), icon: CalendarDays },
          { key: 'vorlagen' as const, label: t('schichten.tabs.vorlagen', { count: templates.length }), icon: Palette },
          { key: 'anfragen' as const, label: t('schichten.tabs.anfragen', { count: pendingSwapCount }), icon: ArrowLeftRight },
          { key: 'verfügbarkeit' as const, label: t('schichten.tabs.verfuegbarkeit'), icon: UserCheck },
        ]).map((tabItem) => {
          const Icon = tabItem.icon
          return (
            <button
              key={tabItem.key}
              onClick={() => setTab(tabItem.key)}
              className={`flex items-center gap-1.5 border-b-2 px-1 pb-2 text-sm transition-colors ${
                tab === tabItem.key
                  ? 'border-primary text-primary font-medium tab-accent-active'
                  : 'border-transparent text-muted-foreground hover:text-foreground'
              }`}
            >
              <Icon className="h-3.5 w-3.5" />
              {tabItem.label}
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
                  {t('schichten.wochenplan.heute')}
                </button>
              )}
            </div>
            <div className="flex items-center gap-2">
              <div className="relative max-w-[240px]">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <input
                  type="text"
                  placeholder={t('schichten.wochenplan.searchPlaceholder')}
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-1.5 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                />
              </div>
              <SortMenu
                options={[
                  { value: 'name', label: t('schichten.sort.name') },
                  { value: 'role', label: t('schichten.sort.rolle') },
                  { value: 'hours', label: t('schichten.sort.stunden') },
                ]}
                field={sortField}
                direction={sortDirection}
                onChange={(f, d) => { setSortField(f); setSortDirection(d) }}
              />
            </div>
          </div>

          {/* Shift Grid */}
          {shiftsWithAssignments.isLoading || employeesQuery.isLoading ? (
            <div className="rounded-lg border border-border overflow-hidden">
              {[1, 2, 3, 4].map((i) => (
                <div key={i} className="h-14 border-b border-border bg-card animate-pulse bg-secondary/20" />
              ))}
            </div>
          ) : shiftsWithAssignments.isError || employeesQuery.isError ? (
            <div className="rounded-lg border border-error/30 bg-error-light/30 p-4 text-sm text-error">
              {t('schichten.error.loadFailed')}
            </div>
          ) : filteredEmployees.length === 0 ? (
            <EmptyState
              icon={CalendarDays}
              title={t('schichten.wochenplan.empty.title')}
              description={search ? t('schichten.wochenplan.empty.hint') : t('schichten.wochenplan.empty.planHint')}
            />
          ) : (
            <div className="rounded-lg border border-border overflow-hidden">
              {/* Grid Header */}
              <div className="grid border-b border-border bg-card" style={{ gridTemplateColumns: '220px repeat(7, 1fr)' }}>
                <div className="flex items-center gap-2 px-4 py-3 border-r border-border">
                  <Users className="h-4 w-4 text-muted-foreground" />
                  <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('schichten.wochenplan.mitarbeiter')}</span>
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
                          role="button"
                          tabIndex={0}
                          aria-label={
                            assignment
                              ? `${assignment.templateName} — ${emp.name}, ${date}`
                              : t('schichten.wochenplan.zelleFrei', { name: emp.name, date })
                          }
                          className={`relative flex items-center justify-center px-1 py-2 cursor-pointer transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring focus-visible:ring-inset ${
                            dayIdx < 6 ? 'border-r border-border-muted' : ''
                          } ${holiday ? 'bg-error/[0.03]' : today ? 'bg-primary/[0.02]' : weekend ? 'bg-secondary/10' : ''} ${
                            !assignment && isHovered ? 'bg-secondary/60' : ''
                          } ${isDragSource ? 'opacity-30' : ''} ${isDropTarget ? 'ring-2 ring-primary ring-inset bg-primary/10' : ''}`}
                          onClick={() => handleCellClick(emp.id, date)}
                          onKeyDown={(e) => {
                            if (e.key === 'Enter' || e.key === ' ') {
                              e.preventDefault()
                              handleCellClick(emp.id, date)
                            }
                          }}
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
                              {/* 7.7: Surcharge badge (toggleable via personal prefs) */}
                              {showSurcharges && surcharge && (
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
                <span className="text-[11px] text-muted-foreground font-medium mr-1">{t('schichten.legend.legende')}</span>
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
                  <span className="text-[11px] text-muted-foreground">{t('schichten.legend.zuschlag')}</span>
                </div>
                <div className="flex items-center gap-1.5">
                  <PartyPopper className="h-3 w-3 text-error" />
                  <span className="text-[11px] text-muted-foreground">{t('schichten.legend.feiertag')}</span>
                </div>
                <div className="flex items-center gap-1.5 ml-auto">
                  <div className="h-2.5 w-2.5 rounded-full bg-success" />
                  <span className="text-[11px] text-muted-foreground">{t('schichten.legend.verfuegbar')}</span>
                  <div className="h-2.5 w-2.5 rounded-full bg-warning ml-2" />
                  <span className="text-[11px] text-muted-foreground">{t('schichten.legend.eingeschraenkt')}</span>
                  <div className="h-2.5 w-2.5 rounded-full bg-error ml-2" />
                  <span className="text-[11px] text-muted-foreground">{t('schichten.legend.abwesend')}</span>
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
                placeholder={t('schichten.vorlagen.suchenPlaceholder')}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <button
              onClick={() => openTemplateDialog(null)}
              className="flex items-center gap-2 rounded-lg border border-border px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
            >
              <Plus className="h-4 w-4" />
              {t('schichten.vorlagen.neue')}
            </button>
          </div>

          {templatesQuery.isLoading ? (
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {[1, 2, 3].map((i) => (
                <div key={i} className="rounded-lg border border-border bg-card p-4 h-32 animate-pulse bg-secondary/20" />
              ))}
            </div>
          ) : templatesQuery.isError ? (
            <div className="rounded-lg border border-error/30 bg-error-light/30 p-4 text-sm text-error">
              {t('schichten.error.loadFailed')}
            </div>
          ) : templates.length === 0 ? (
            <EmptyState
              icon={Palette}
              title={t('schichten.vorlagen.empty.title')}
              description={t('schichten.vorlagen.empty.hint')}
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
                          <span>{t('schichten.vorlagen.netto', { h: durationHours, min: durationRemainder > 0 ? ` ${durationRemainder}min` : '' })}</span>
                        </div>
                        <div className="flex items-center gap-1.5">
                          <Coffee className="h-3 w-3" />
                          <span>{t('schichten.vorlagen.pause', { min: template.breakMinutes })}</span>
                        </div>
                        {surcharge && (
                          <div className="flex items-center gap-1.5 text-warning">
                            <Percent className="h-3 w-3" />
                            <span>{surcharge.rate} {surcharge.label}</span>
                          </div>
                        )}
                        <div className="flex items-center gap-1.5 ml-auto">
                          <div className="h-3 w-3 rounded-full" style={{ backgroundColor: template.color }} />
                          <span>{t('schichten.vorlagen.farbe')}</span>
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
                placeholder={t('schichten.anfragen.suchenPlaceholder')}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
          </div>

          {swapRequestsQuery.isLoading ? (
            <div className="space-y-3">
              {[1, 2, 3].map((i) => (
                <div key={i} className="rounded-lg border border-border bg-card p-4 h-24 animate-pulse bg-secondary/20" />
              ))}
            </div>
          ) : swapRequestsQuery.isError ? (
            <div className="rounded-lg border border-error/30 bg-error-light/30 p-4 text-sm text-error">
              {t('schichten.error.loadFailed')}
            </div>
          ) : swapRequests.length === 0 ? (
            <EmptyState
              icon={ArrowLeftRight}
              title={t('schichten.anfragen.empty.title')}
              description={t('schichten.anfragen.empty.hint')}
            />
          ) : (
            <div className="space-y-3">
              {swapRequests
                .filter((r) => {
                  if (!search) return true
                  const q = search.toLowerCase()
                  return (
                    r.requestedByEmployeeId.toLowerCase().includes(q) ||
                    r.swapWithEmployeeId.toLowerCase().includes(q) ||
                    (r.reason ?? '').toLowerCase().includes(q)
                  )
                })
                .map((swap) => {
                  // Employee IDs are UUIDs from the API; initials derived from first 2 chars as fallback
                  const reqInitials = swap.requestedByEmployeeId.slice(0, 2).toUpperCase()
                  const swapInitials = swap.swapWithEmployeeId.slice(0, 2).toUpperCase()
                  return (
                    <div key={swap.id} className="rounded-lg border border-border bg-card p-4 transition-shadow hover:shadow-[var(--shadow-card-hover)]">
                      <div className="flex items-start justify-between gap-4">
                        <div className="flex-1">
                          <div className="flex items-center gap-3 mb-2">
                            <div className="flex items-center gap-2">
                              <div className="flex h-7 w-7 items-center justify-center rounded-full bg-secondary text-[11px] font-medium text-foreground">
                                {reqInitials}
                              </div>
                              <span className="text-sm font-medium text-foreground font-mono text-xs">{swap.requestedByEmployeeId}</span>
                            </div>
                            <ArrowLeftRight className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                            <div className="flex items-center gap-2">
                              <div className="flex h-7 w-7 items-center justify-center rounded-full bg-secondary text-[11px] font-medium text-foreground">
                                {swapInitials}
                              </div>
                              <span className="text-sm font-medium text-foreground font-mono text-xs">{swap.swapWithEmployeeId}</span>
                            </div>
                          </div>

                          <div className="flex items-center gap-2 mb-2">
                            <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${swapStatusColors[swap.status]}`}>
                              {swapStatusLabels[swap.status]}
                            </span>
                            <span className="text-[11px] text-muted-foreground">
                              {t('schichten.anfragen.erstellt', { date: new Date(swap.createdAt).toLocaleDateString('de-DE', { day: '2-digit', month: 'long', year: 'numeric' }) })}
                            </span>
                          </div>

                          {swap.reason && (
                            <p className="text-sm text-muted-foreground">
                              <StickyNote className="inline h-3.5 w-3.5 mr-1 -mt-0.5" />
                              {swap.reason}
                            </p>
                          )}
                        </div>

                        {swap.status === 'pending' && (
                          <div className="flex flex-col gap-2 flex-shrink-0">
                            <button
                              onClick={() => handleApproveSwap(swap)}
                              disabled={approveSwapMutation.isPending}
                              className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
                            >
                              <Check className="h-3.5 w-3.5" />
                              {t('schichten.anfragen.genehmigen')}
                            </button>
                            <button
                              onClick={() => handleRejectSwap(swap)}
                              disabled={rejectSwapMutation.isPending}
                              className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors disabled:opacity-50"
                            >
                              <XCircle className="h-3.5 w-3.5" />
                              {t('schichten.anfragen.ablehnen')}
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
      {tab === 'verfügbarkeit' && (
        <>
          <div className="flex items-center justify-between mb-4">
            <div>
              <p className="text-sm text-muted-foreground">
                {t('schichten.verfuegbarkeit.hinweis')}
              </p>
            </div>
            <button
              onClick={() => {
                setEditingAvailability(!editingAvailability)
                if (editingAvailability) toast.success(t('schichten.verfuegbarkeit.gespeichert'))
              }}
              className={`flex items-center gap-2 rounded-lg px-4 py-2 text-sm transition-colors ${
                editingAvailability
                  ? 'bg-success text-white hover:bg-success/90'
                  : 'bg-primary text-primary-foreground hover:bg-button-primary-hover'
              }`}
            >
              {editingAvailability ? <Check className="h-4 w-4" /> : <UserCheck className="h-4 w-4" />}
              {editingAvailability ? t('schichten.verfuegbarkeit.speichern') : t('schichten.verfuegbarkeit.bearbeiten')}
            </button>
          </div>

          <div className="rounded-lg border border-border overflow-hidden">
            {/* Header */}
            <div className="grid border-b border-border bg-card" style={{ gridTemplateColumns: '220px repeat(7, 1fr)' }}>
              <div className="flex items-center gap-2 px-4 py-3 border-r border-border">
                <Users className="h-4 w-4 text-muted-foreground" />
                <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('schichten.wochenplan.mitarbeiter')}</span>
              </div>
              {WEEKDAYS.map((day, i) => (
                <div key={i} className={`flex items-center justify-center py-3 text-xs font-semibold text-foreground ${i < 6 ? 'border-r border-border' : ''}`}>
                  {day}
                </div>
              ))}
            </div>

            {/* Body */}
            {employees.map((emp, empIdx) => (
              <div
                key={emp.id}
                className={`grid ${empIdx < employees.length - 1 ? 'border-b border-border-muted' : ''}`}
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
              <span className="text-[11px] text-muted-foreground font-medium">{t('schichten.legend.legende')}</span>
              {Object.entries(availabilityLevelColors).map(([level, config]) => (
                <div key={level} className="flex items-center gap-1.5">
                  <div className={`h-3 w-3 rounded-full ${
                    level === 'green' ? 'bg-success' : level === 'yellow' ? 'bg-warning' : 'bg-error'
                  }`} />
                  <span className="text-[11px] text-muted-foreground">{config.label}</span>
                </div>
              ))}
              {editingAvailability && (
                <span className="text-[11px] text-primary font-medium ml-auto">{t('schichten.verfuegbarkeit.aendernHint')}</span>
              )}
            </div>
          </div>
        </>
      )}

      {/* ============================================================ */}
      {/* DIALOG: Schicht zuweisen                                      */}
      {/* ============================================================ */}
      <Dialog open={assignDialog.open} onOpenChange={(o) => { if (!o) setAssignDialog({ open: false, employeeId: '', date: '' }) }}>
        <DialogContent className="gap-0 p-0 max-w-md">
          <div className="p-6">
            <DialogHeader className="mb-5">
              <DialogTitle className="text-base font-semibold text-foreground">{t('schichten.dialog.assign.title')}</DialogTitle>
              <DialogDescription className="sr-only">{t('schichten.dialog.assign.title')}</DialogDescription>
            </DialogHeader>

            <div className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5">{t('schichten.dialog.assign.mitarbeiter')}</label>
                <select value={assignEmployee} onChange={(e) => setAssignEmployee(e.target.value)} className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring">
                  <option value="">{t('schichten.dialog.assign.mitarbeiterSelect')}</option>
                  {employees.map((e) => <option key={e.id} value={e.id}>{e.name} ({e.role})</option>)}
                </select>
              </div>
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5">{t('schichten.dialog.assign.vorlage')}</label>
                <select value={assignTemplate} onChange={(e) => setAssignTemplate(e.target.value)} className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring">
                  <option value="">{t('schichten.dialog.assign.vorlageSelect')}</option>
                  {templates.map((tplItem) => <option key={tplItem.id} value={tplItem.id}>{tplItem.name} ({tplItem.startTime} – {tplItem.endTime})</option>)}
                </select>
              </div>
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5">{t('schichten.dialog.assign.datum')}</label>
                <input type="date" value={assignDate} onChange={(e) => setAssignDate(e.target.value)} className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring" />
              </div>
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5">{t('schichten.dialog.assign.notizen')}</label>
                <textarea value={assignNotes} onChange={(e) => setAssignNotes(e.target.value)} placeholder={t('schichten.dialog.assign.notizenPlaceholder')} rows={2} className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring resize-none" />
              </div>
            </div>

            <DialogFooter className="mt-6">
              <button onClick={() => setAssignDialog({ open: false, employeeId: '', date: '' })} className="rounded-lg border border-border px-4 py-2 text-sm text-foreground hover:bg-secondary transition-colors">
                {t('common.cancel')}
              </button>
              <button onClick={handleAssignSubmit} className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors">
                {t('schichten.dialog.assign.zuweisen')}
              </button>
            </DialogFooter>
          </div>
        </DialogContent>
      </Dialog>

      {/* ============================================================ */}
      {/* DIALOG: Vorlage erstellen / bearbeiten                        */}
      {/* ============================================================ */}
      <Dialog open={templateDialog.open} onOpenChange={(o) => { if (!o) setTemplateDialog({ open: false, edit: null }) }}>
        <DialogContent className="gap-0 p-0 max-w-md" key={templateDialog.open ? (templateDialog.edit?.id ?? 'new') : 'closed'}>
          <div className="p-6">
            <DialogHeader className="mb-5">
              <DialogTitle className="text-base font-semibold text-foreground">
                {templateDialog.edit ? t('schichten.dialog.template.editTitle') : t('schichten.dialog.template.title')}
              </DialogTitle>
              <DialogDescription className="sr-only">
                {templateDialog.edit ? t('schichten.dialog.template.editTitle') : t('schichten.dialog.template.title')}
              </DialogDescription>
            </DialogHeader>

            <div className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5">{t('schichten.dialog.template.name')}</label>
                <input type="text" value={newTplName} onChange={(e) => setNewTplName(e.target.value)} placeholder={t('schichten.dialog.template.namePlaceholder')} className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring" />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1.5">{t('schichten.dialog.template.startzeit')}</label>
                  <input type="time" value={newTplStart} onChange={(e) => setNewTplStart(e.target.value)} className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring" />
                </div>
                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1.5">{t('schichten.dialog.template.endzeit')}</label>
                  <input type="time" value={newTplEnd} onChange={(e) => setNewTplEnd(e.target.value)} className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring" />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1.5">{t('schichten.dialog.template.pause')}</label>
                  <input type="number" min="0" max="120" value={newTplBreak} onChange={(e) => setNewTplBreak(e.target.value)} className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring" />
                </div>
                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1.5">{t('schichten.dialog.template.farbe')}</label>
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

            <DialogFooter className="mt-6">
              <button onClick={() => setTemplateDialog({ open: false, edit: null })} className="rounded-lg border border-border px-4 py-2 text-sm text-foreground hover:bg-secondary transition-colors">
                {t('common.cancel')}
              </button>
              <button onClick={handleTemplateSubmit} className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors">
                {templateDialog.edit ? t('schichten.dialog.template.speichern') : t('schichten.dialog.template.erstellen')}
              </button>
            </DialogFooter>
          </div>
        </DialogContent>
      </Dialog>

      {/* Confirm Delete Template */}
      <ConfirmDialog
        open={!!confirmDelete}
        onOpenChange={() => setConfirmDelete(null)}
        title={t('schichten.delete.title')}
        description={t('schichten.delete.description', { name: confirmDelete?.name ?? '' })}
        confirmLabel={t('schichten.vorlagen.actions.loeschen')}
        variant="destructive"
        onConfirm={() => confirmDelete && handleDeleteTemplate(confirmDelete)}
      />

      {/* ============================================================ */}
      {/* MODAL: Schicht-Detail (belegte Zelle)                         */}
      {/* ============================================================ */}
      <ShiftDetailModal
        open={!!detailAssignment}
        onClose={() => setDetailKey(null)}
        assignment={detailAssignment}
        template={detailTemplate}
        employeeRole={detailEmployee?.role ?? ''}
        shiftId={detailShiftId}
        violations={detailViolations}
        swapCandidates={swapEnabled ? employees.filter((e) => e.id !== detailAssignment?.userId) : []}
        swapEnabled={swapEnabled}
        onUnassign={handleUnassignFromDetail}
        isUnassigning={unassignMutation.isPending}
        onCreateSwap={handleCreateSwapFromDetail}
        isCreatingSwap={createSwapMutation.isPending}
      />

      {/* ============================================================ */}
      {/* DIALOG: Vorlage auf Woche anwenden                            */}
      {/* ============================================================ */}
      <Dialog open={!!applyTarget} onOpenChange={(o) => { if (!o) setApplyTarget(null) }}>
        <DialogContent className="gap-0 p-0 max-w-sm">
          <div className="p-6">
            <DialogHeader className="mb-5">
              <DialogTitle className="text-base font-semibold text-foreground">
                {t('schichten.vorlagen.applyTitle', { name: applyTarget?.name ?? '' })}
              </DialogTitle>
              <DialogDescription className="text-xs text-muted-foreground">
                {t('schichten.vorlagen.applyDescription')}
              </DialogDescription>
            </DialogHeader>

            <div className="space-y-2">
              {([
                { key: 'current' as const, label: t('schichten.vorlagen.applyCurrentWeek', { kw }) },
                { key: 'next' as const, label: t('schichten.vorlagen.applyNextWeek', { kw: getKW(getWeekDates(weekOffset + 1)[0]) }) },
              ]).map((opt) => (
                <button
                  key={opt.key}
                  onClick={() => setApplyWeek(opt.key)}
                  className={`flex w-full items-center gap-2 rounded-lg border px-3 py-2 text-sm transition-colors ${
                    applyWeek === opt.key
                      ? 'border-primary bg-primary/5 text-foreground'
                      : 'border-border text-muted-foreground hover:bg-secondary'
                  }`}
                >
                  <CalendarDays className="h-4 w-4" />
                  {opt.label}
                  {applyWeek === opt.key && <Check className="ml-auto h-4 w-4 text-primary" />}
                </button>
              ))}
            </div>

            <DialogFooter className="mt-6">
              <button onClick={() => setApplyTarget(null)} className="rounded-lg border border-border px-4 py-2 text-sm text-foreground hover:bg-secondary transition-colors">
                {t('common.cancel')}
              </button>
              <button
                onClick={handleApplyTemplate}
                disabled={applyTemplateMutation.isPending}
                className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
              >
                {t('schichten.vorlagen.applyConfirm')}
              </button>
            </DialogFooter>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}

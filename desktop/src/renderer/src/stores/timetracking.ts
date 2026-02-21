import { create } from 'zustand'
import { persist } from 'zustand/middleware'

// ── Types ──────────────────────────────────────────────────────

export type TimerStatus = 'idle' | 'running' | 'paused'

export interface TimeCategory {
  id: string
  name: string
  color: string
  icon: string
  isDefault: boolean
}

export interface TimeTemplate {
  id: string
  name: string
  categoryId: string
  description: string
  estimatedMinutes: number
}

export interface TimeEntry {
  id: string
  categoryId: string
  description: string
  date: string // YYYY-MM-DD
  startTime: string // HH:mm
  endTime: string | null // HH:mm, null if running
  durationMinutes: number
  isManual: boolean
  userId: string
  userName: string
  userInitials: string
  projectId: string | null
  taskId: string | null
  location: { lat: number; lng: number; address: string } | null
}

export interface WorkTarget {
  dailyHours: number
  weeklyHours: number
  monthlyHours: number
}

export interface AbsenceRequest {
  id: string
  type: 'vacation' | 'sick' | 'homeoffice' | 'education' | 'other'
  startDate: string
  endDate: string
  days: number
  status: 'pending' | 'approved' | 'rejected'
  reason: string
  comment?: string
  createdAt: string
}

export interface TeamActivityEntry {
  userId: string
  userName: string
  userInitials: string
  status: 'tracking' | 'idle' | 'absent'
  currentCategory?: string
  currentCategoryColor?: string
  currentDescription?: string
  startedAt?: string
}

interface ActiveTimer {
  status: TimerStatus
  categoryId: string | null
  description: string
  startedAt: number | null
  pausedAt: number | null
  accumulatedMs: number
  projectId: string | null
  taskId: string | null
}

// ── Week Approval ────────────────────────────────────────────

export interface WeekApproval {
  id: string
  weekStart: string // YYYY-MM-DD (Monday)
  status: 'draft' | 'pending' | 'approved' | 'rejected'
  submittedAt?: string
  reviewedBy?: string
  reviewedAt?: string
  comment?: string
}

// ── Mock Projects & Tasks ────────────────────────────────────

export interface MockProject {
  id: string
  name: string
  key: string
}

export interface MockTask {
  id: string
  projectId: string
  title: string
  key: string
}

export const MOCK_PROJECTS: MockProject[] = [
  { id: 'proj-1', name: 'KMU Hub CRM', key: 'KMU' },
  { id: 'proj-2', name: 'Website Redesign', key: 'WEB' },
  { id: 'proj-3', name: 'Mobile App', key: 'MOB' },
]

export const MOCK_TASKS: MockTask[] = [
  { id: 'task-1', projectId: 'proj-1', title: 'API Endpunkte', key: 'KMU-42' },
  { id: 'task-2', projectId: 'proj-1', title: 'Dashboard UI', key: 'KMU-58' },
  { id: 'task-3', projectId: 'proj-2', title: 'Header Component', key: 'WEB-12' },
  { id: 'task-4', projectId: 'proj-2', title: 'Responsive Layout', key: 'WEB-15' },
  { id: 'task-5', projectId: 'proj-3', title: 'Push Notifications', key: 'MOB-8' },
]

// ── Mock GPS Locations ───────────────────────────────────────

const MOCK_LOCATIONS = [
  { lat: 47.3769, lng: 8.5417, address: 'Bahnhofstrasse 1, 8001 Zürich' },
  { lat: 52.5200, lng: 13.4050, address: 'Friedrichstrasse 43, 10117 Berlin' },
  { lat: 48.1351, lng: 11.5820, address: 'Marienplatz 1, 80331 München' },
]

// ── Store Interface ────────────────────────────────────────────

interface TimeTrackingState {
  activeTimer: ActiveTimer
  entries: TimeEntry[]
  categories: TimeCategory[]
  templates: TimeTemplate[]
  targets: WorkTarget
  absences: AbsenceRequest[]
  teamActivity: TeamActivityEntry[]
  weekApprovals: WeekApproval[]
  gpsEnabled: boolean

  // Timer actions
  startTimer: (categoryId: string, description: string, projectId?: string | null, taskId?: string | null) => void
  pauseTimer: () => void
  resumeTimer: () => void
  stopTimer: () => void
  discardTimer: () => void

  // Entry CRUD
  addManualEntry: (entry: Omit<TimeEntry, 'id' | 'isManual' | 'userId' | 'userName' | 'userInitials'>) => void
  updateEntry: (id: string, updates: Partial<TimeEntry>) => void
  deleteEntry: (id: string) => void

  // Category CRUD
  addCategory: (category: Omit<TimeCategory, 'id'>) => void
  updateCategory: (id: string, updates: Partial<TimeCategory>) => void
  deleteCategory: (id: string) => void

  // Template CRUD
  addTemplate: (template: Omit<TimeTemplate, 'id'>) => void
  deleteTemplate: (id: string) => void

  // Targets
  updateTargets: (targets: Partial<WorkTarget>) => void

  // Absence CRUD
  addAbsence: (absence: Omit<AbsenceRequest, 'id' | 'createdAt'>) => void
  updateAbsenceStatus: (id: string, status: 'approved' | 'rejected', comment?: string) => void
  deleteAbsence: (id: string) => void

  // Week Approval actions
  submitWeekReport: (weekStart: string) => void
  approveWeekReport: (weekStart: string, reviewer: string) => void
  rejectWeekReport: (weekStart: string, reviewer: string, comment: string) => void

  // GPS
  setGpsEnabled: (enabled: boolean) => void
}

// ── Mock Data ──────────────────────────────────────────────────

const INITIAL_CATEGORIES: TimeCategory[] = [
  { id: 'cat-dev', name: 'Entwicklung', color: '#3b82f6', icon: 'Code', isDefault: true },
  { id: 'cat-meeting', name: 'Meeting', color: '#8b5cf6', icon: 'Users', isDefault: true },
  { id: 'cat-client', name: 'Kundengespraech', color: '#f59e0b', icon: 'Phone', isDefault: true },
  { id: 'cat-admin', name: 'Admin', color: '#6b7280', icon: 'FileText', isDefault: true },
  { id: 'cat-break', name: 'Pause', color: '#94a3b8', icon: 'Coffee', isDefault: true },
  { id: 'cat-design', name: 'Design', color: '#ec4899', icon: 'Palette', isDefault: true },
]

const INITIAL_TEMPLATES: TimeTemplate[] = [
  { id: 'tpl-1', name: 'Sprint Planning', categoryId: 'cat-meeting', description: 'Sprint Planning Session', estimatedMinutes: 60 },
  { id: 'tpl-2', name: 'Code Review', categoryId: 'cat-dev', description: 'Pull Request Review', estimatedMinutes: 30 },
  { id: 'tpl-3', name: 'Daily Standup', categoryId: 'cat-meeting', description: 'Tägliches Team-Meeting', estimatedMinutes: 15 },
  { id: 'tpl-4', name: 'Kundendemo', categoryId: 'cat-client', description: 'Produkt-Demo für Kunden', estimatedMinutes: 45 },
]

function todayStr(): string {
  return new Date().toISOString().split('T')[0]
}

function daysAgo(n: number): string {
  const d = new Date()
  d.setDate(d.getDate() - n)
  return d.toISOString().split('T')[0]
}

const INITIAL_ENTRIES: TimeEntry[] = [
  // Today
  { id: 'te-1', categoryId: 'cat-dev', description: 'API Endpunkte implementieren', date: todayStr(), startTime: '08:30', endTime: '10:00', durationMinutes: 90, isManual: false, userId: 'self', userName: 'Markus Weber', userInitials: 'MW', projectId: 'proj-1', taskId: 'task-1', location: { lat: 47.3769, lng: 8.5417, address: 'Bahnhofstrasse 1, 8001 Zürich' } },
  { id: 'te-2', categoryId: 'cat-meeting', description: 'Daily Standup', date: todayStr(), startTime: '10:00', endTime: '10:15', durationMinutes: 15, isManual: false, userId: 'self', userName: 'Markus Weber', userInitials: 'MW', projectId: null, taskId: null, location: null },
  { id: 'te-3', categoryId: 'cat-dev', description: 'Bug Fix: Datenbankverbindung', date: todayStr(), startTime: '10:30', endTime: '12:00', durationMinutes: 90, isManual: false, userId: 'self', userName: 'Markus Weber', userInitials: 'MW', projectId: 'proj-1', taskId: 'task-2', location: { lat: 47.3769, lng: 8.5417, address: 'Bahnhofstrasse 1, 8001 Zürich' } },
  { id: 'te-4', categoryId: 'cat-client', description: 'Kundenmeeting Meier AG', date: todayStr(), startTime: '13:00', endTime: '14:00', durationMinutes: 60, isManual: false, userId: 'self', userName: 'Markus Weber', userInitials: 'MW', projectId: null, taskId: null, location: null },
  // Yesterday
  { id: 'te-5', categoryId: 'cat-dev', description: 'Frontend Refactoring', date: daysAgo(1), startTime: '08:00', endTime: '11:30', durationMinutes: 210, isManual: false, userId: 'self', userName: 'Markus Weber', userInitials: 'MW', projectId: 'proj-2', taskId: 'task-3', location: null },
  { id: 'te-6', categoryId: 'cat-meeting', description: 'Sprint Review', date: daysAgo(1), startTime: '13:00', endTime: '14:30', durationMinutes: 90, isManual: false, userId: 'self', userName: 'Markus Weber', userInitials: 'MW', projectId: null, taskId: null, location: null },
  { id: 'te-7', categoryId: 'cat-admin', description: 'E-Mails bearbeiten', date: daysAgo(1), startTime: '14:30', endTime: '15:30', durationMinutes: 60, isManual: true, userId: 'self', userName: 'Markus Weber', userInitials: 'MW', projectId: null, taskId: null, location: null },
  { id: 'te-8', categoryId: 'cat-dev', description: 'Tests schreiben', date: daysAgo(1), startTime: '15:30', endTime: '17:00', durationMinutes: 90, isManual: false, userId: 'self', userName: 'Markus Weber', userInitials: 'MW', projectId: 'proj-1', taskId: null, location: { lat: 52.5200, lng: 13.4050, address: 'Friedrichstrasse 43, 10117 Berlin' } },
  // 2 days ago
  { id: 'te-9', categoryId: 'cat-design', description: 'UI Mockups erstellen', date: daysAgo(2), startTime: '08:30', endTime: '12:00', durationMinutes: 210, isManual: false, userId: 'self', userName: 'Markus Weber', userInitials: 'MW', projectId: 'proj-2', taskId: 'task-4', location: null },
  { id: 'te-10', categoryId: 'cat-client', description: 'Support-Call Fischer GmbH', date: daysAgo(2), startTime: '13:00', endTime: '13:45', durationMinutes: 45, isManual: false, userId: 'self', userName: 'Markus Weber', userInitials: 'MW', projectId: null, taskId: null, location: null },
  { id: 'te-11', categoryId: 'cat-dev', description: 'Deployment Pipeline', date: daysAgo(2), startTime: '14:00', endTime: '17:00', durationMinutes: 180, isManual: false, userId: 'self', userName: 'Markus Weber', userInitials: 'MW', projectId: 'proj-3', taskId: 'task-5', location: { lat: 48.1351, lng: 11.5820, address: 'Marienplatz 1, 80331 München' } },
  // Last week
  { id: 'te-12', categoryId: 'cat-dev', description: 'Authentifizierung', date: daysAgo(5), startTime: '08:00', endTime: '12:00', durationMinutes: 240, isManual: false, userId: 'self', userName: 'Markus Weber', userInitials: 'MW', projectId: 'proj-1', taskId: null, location: null },
  { id: 'te-13', categoryId: 'cat-meeting', description: 'Quartals-Review', date: daysAgo(5), startTime: '13:00', endTime: '15:00', durationMinutes: 120, isManual: false, userId: 'self', userName: 'Markus Weber', userInitials: 'MW', projectId: null, taskId: null, location: null },
  { id: 'te-14', categoryId: 'cat-admin', description: 'Dokumentation aktualisieren', date: daysAgo(6), startTime: '09:00', endTime: '11:00', durationMinutes: 120, isManual: true, userId: 'self', userName: 'Markus Weber', userInitials: 'MW', projectId: null, taskId: null, location: null },
  { id: 'te-15', categoryId: 'cat-dev', description: 'Performance Optimierung', date: daysAgo(7), startTime: '08:00', endTime: '16:30', durationMinutes: 510, isManual: false, userId: 'self', userName: 'Markus Weber', userInitials: 'MW', projectId: 'proj-1', taskId: 'task-1', location: null },
]

const INITIAL_TEAM_ACTIVITY: TeamActivityEntry[] = [
  { userId: 'm1', userName: 'Anna Mueller', userInitials: 'AM', status: 'tracking', currentCategory: 'Meeting', currentCategoryColor: '#8b5cf6', currentDescription: 'Sprint Planning vorbereiten', startedAt: '09:30' },
  { userId: 'm2', userName: 'Michael Berg', userInitials: 'MB', status: 'tracking', currentCategory: 'Entwicklung', currentCategoryColor: '#3b82f6', currentDescription: 'API Integration testen', startedAt: '08:15' },
  { userId: 'm3', userName: 'Sarah Klein', userInitials: 'SK', status: 'absent', currentCategory: undefined, currentDescription: 'Urlaub' },
  { userId: 'm4', userName: 'Jonas Diaz', userInitials: 'JD', status: 'tracking', currentCategory: 'Entwicklung', currentCategoryColor: '#3b82f6', currentDescription: 'Bug Fix: iOS Login', startedAt: '10:00' },
  { userId: 'm7', userName: 'Eva Brunner', userInitials: 'EB', status: 'idle' },
  { userId: 'm6', userName: 'Peter Koch', userInitials: 'PK', status: 'tracking', currentCategory: 'Admin', currentCategoryColor: '#6b7280', currentDescription: 'Penetration Testing', startedAt: '07:45' },
]

const INITIAL_ABSENCES: AbsenceRequest[] = [
  { id: 'abs-1', type: 'vacation', startDate: '2026-02-17', endDate: '2026-02-21', days: 5, status: 'pending', reason: 'Skiurlaub in den Alpen', createdAt: '2026-02-03' },
  { id: 'abs-2', type: 'homeoffice', startDate: '2026-02-10', endDate: '2026-02-14', days: 5, status: 'approved', reason: 'Konzentrationswoche', createdAt: '2026-02-05' },
  { id: 'abs-3', type: 'sick', startDate: '2026-01-20', endDate: '2026-01-21', days: 2, status: 'approved', reason: 'Erkaeltung', createdAt: '2026-01-20' },
  { id: 'abs-4', type: 'education', startDate: '2026-03-10', endDate: '2026-03-11', days: 2, status: 'pending', reason: 'React Advanced Workshop', createdAt: '2026-02-08' },
]

// Helper to get the Monday of the current week
function getWeekStartStr(offset = 0): string {
  const d = new Date()
  const day = d.getDay()
  const mondayOffset = day === 0 ? -6 : 1 - day
  d.setDate(d.getDate() + mondayOffset + offset * 7)
  return d.toISOString().split('T')[0]
}

const INITIAL_WEEK_APPROVALS: WeekApproval[] = [
  { id: 'wa-1', weekStart: getWeekStartStr(0), status: 'draft' },
  { id: 'wa-2', weekStart: getWeekStartStr(-1), status: 'approved', submittedAt: '2026-02-14T10:00:00Z', reviewedBy: 'Anna Mueller', reviewedAt: '2026-02-14T14:30:00Z' },
  { id: 'wa-3', weekStart: getWeekStartStr(-2), status: 'rejected', submittedAt: '2026-02-07T09:00:00Z', reviewedBy: 'Anna Mueller', reviewedAt: '2026-02-07T16:00:00Z', comment: 'Bitte Freitag nachtragen' },
]

// ── Store ──────────────────────────────────────────────────────

export const useTimeTrackingStore = create<TimeTrackingState>()(
  persist(
    (set, get) => ({
      activeTimer: {
        status: 'idle' as TimerStatus,
        categoryId: null,
        description: '',
        startedAt: null,
        pausedAt: null,
        accumulatedMs: 0,
        projectId: null,
        taskId: null,
      },
      entries: INITIAL_ENTRIES,
      categories: INITIAL_CATEGORIES,
      templates: INITIAL_TEMPLATES,
      targets: { dailyHours: 8.4, weeklyHours: 42, monthlyHours: 176 },
      absences: INITIAL_ABSENCES,
      teamActivity: INITIAL_TEAM_ACTIVITY,
      weekApprovals: INITIAL_WEEK_APPROVALS,
      gpsEnabled: false,

      // ── Timer ──────────────────────────────────────────────

      startTimer: (categoryId, description, projectId, taskId) =>
        set({
          activeTimer: {
            status: 'running',
            categoryId,
            description,
            startedAt: Date.now(),
            pausedAt: null,
            accumulatedMs: 0,
            projectId: projectId ?? null,
            taskId: taskId ?? null,
          },
        }),

      pauseTimer: () =>
        set((state) => ({
          activeTimer: {
            ...state.activeTimer,
            status: 'paused',
            pausedAt: Date.now(),
          },
        })),

      resumeTimer: () =>
        set((state) => {
          const { activeTimer } = state
          if (!activeTimer.pausedAt || !activeTimer.startedAt) return state
          const pausedDuration = Date.now() - activeTimer.pausedAt
          return {
            activeTimer: {
              ...activeTimer,
              status: 'running',
              startedAt: activeTimer.startedAt + pausedDuration,
              pausedAt: null,
            },
          }
        }),

      stopTimer: () => {
        const { activeTimer, entries, gpsEnabled } = get()
        if (!activeTimer.startedAt || !activeTimer.categoryId) return

        const startDate = new Date(activeTimer.startedAt)
        const now = new Date()
        let totalMs = activeTimer.accumulatedMs
        if (activeTimer.status === 'running') {
          totalMs += now.getTime() - activeTimer.startedAt
        } else if (activeTimer.pausedAt) {
          totalMs += activeTimer.pausedAt - activeTimer.startedAt
        }

        const durationMinutes = Math.round(totalMs / 60000)
        const pad = (n: number) => String(n).padStart(2, '0')

        // Pick a random mock location when GPS is enabled
        const location = gpsEnabled
          ? MOCK_LOCATIONS[Math.floor(Math.random() * MOCK_LOCATIONS.length)]
          : null

        const entry: TimeEntry = {
          id: `te-${Date.now()}`,
          categoryId: activeTimer.categoryId,
          description: activeTimer.description,
          date: startDate.toISOString().split('T')[0],
          startTime: `${pad(startDate.getHours())}:${pad(startDate.getMinutes())}`,
          endTime: `${pad(now.getHours())}:${pad(now.getMinutes())}`,
          durationMinutes,
          isManual: false,
          userId: 'self',
          userName: 'Markus Weber',
          userInitials: 'MW',
          projectId: activeTimer.projectId,
          taskId: activeTimer.taskId,
          location,
        }

        set({
          entries: [...entries, entry],
          activeTimer: {
            status: 'idle',
            categoryId: null,
            description: '',
            startedAt: null,
            pausedAt: null,
            accumulatedMs: 0,
            projectId: null,
            taskId: null,
          },
        })
      },

      discardTimer: () =>
        set({
          activeTimer: {
            status: 'idle',
            categoryId: null,
            description: '',
            startedAt: null,
            pausedAt: null,
            accumulatedMs: 0,
            projectId: null,
            taskId: null,
          },
        }),

      // ── Entries ────────────────────────────────────────────

      addManualEntry: (entry) =>
        set((state) => ({
          entries: [
            ...state.entries,
            {
              ...entry,
              id: `te-${Date.now()}`,
              isManual: true,
              userId: 'self',
              userName: 'Markus Weber',
              userInitials: 'MW',
            },
          ],
        })),

      updateEntry: (id, updates) =>
        set((state) => ({
          entries: state.entries.map((e) =>
            e.id === id ? { ...e, ...updates } : e,
          ),
        })),

      deleteEntry: (id) =>
        set((state) => ({
          entries: state.entries.filter((e) => e.id !== id),
        })),

      // ── Categories ─────────────────────────────────────────

      addCategory: (category) =>
        set((state) => ({
          categories: [
            ...state.categories,
            { ...category, id: `cat-${Date.now()}` },
          ],
        })),

      updateCategory: (id, updates) =>
        set((state) => ({
          categories: state.categories.map((c) =>
            c.id === id ? { ...c, ...updates } : c,
          ),
        })),

      deleteCategory: (id) =>
        set((state) => ({
          categories: state.categories.filter((c) => c.id !== id),
        })),

      // ── Templates ──────────────────────────────────────────

      addTemplate: (template) =>
        set((state) => ({
          templates: [
            ...state.templates,
            { ...template, id: `tpl-${Date.now()}` },
          ],
        })),

      deleteTemplate: (id) =>
        set((state) => ({
          templates: state.templates.filter((t) => t.id !== id),
        })),

      // ── Targets ────────────────────────────────────────────

      updateTargets: (targets) =>
        set((state) => ({
          targets: { ...state.targets, ...targets },
        })),

      // ── Absences ───────────────────────────────────────────

      addAbsence: (absence) =>
        set((state) => ({
          absences: [
            ...state.absences,
            {
              ...absence,
              id: `abs-${Date.now()}`,
              createdAt: new Date().toISOString().split('T')[0],
            },
          ],
        })),

      updateAbsenceStatus: (id, status, comment) =>
        set((state) => ({
          absences: state.absences.map((a) =>
            a.id === id ? { ...a, status, comment } : a,
          ),
        })),

      deleteAbsence: (id) =>
        set((state) => ({
          absences: state.absences.filter((a) => a.id !== id),
        })),

      // ── Week Approvals ────────────────────────────────────

      submitWeekReport: (weekStart) =>
        set((state) => {
          const existing = state.weekApprovals.find((a) => a.weekStart === weekStart)
          if (existing) {
            return {
              weekApprovals: state.weekApprovals.map((a) =>
                a.weekStart === weekStart
                  ? { ...a, status: 'pending' as const, submittedAt: new Date().toISOString(), comment: undefined }
                  : a,
              ),
            }
          }
          return {
            weekApprovals: [
              ...state.weekApprovals,
              {
                id: `wa-${Date.now()}`,
                weekStart,
                status: 'pending' as const,
                submittedAt: new Date().toISOString(),
              },
            ],
          }
        }),

      approveWeekReport: (weekStart, reviewer) =>
        set((state) => ({
          weekApprovals: state.weekApprovals.map((a) =>
            a.weekStart === weekStart
              ? { ...a, status: 'approved' as const, reviewedBy: reviewer, reviewedAt: new Date().toISOString() }
              : a,
          ),
        })),

      rejectWeekReport: (weekStart, reviewer, comment) =>
        set((state) => ({
          weekApprovals: state.weekApprovals.map((a) =>
            a.weekStart === weekStart
              ? { ...a, status: 'rejected' as const, reviewedBy: reviewer, reviewedAt: new Date().toISOString(), comment }
              : a,
          ),
        })),

      // ── GPS ───────────────────────────────────────────────

      setGpsEnabled: (enabled) => set({ gpsEnabled: enabled }),
    }),
    { name: 'kmuhub-timetracking' },
  ),
)

// ── Overtime Helper Functions (6.8) ──────────────────────────

/** Get overtime minutes for a specific day (positive = over, negative = under) */
export function getOvertimeForDay(entries: TimeEntry[], date: string, dailyTargetMinutes: number): number {
  const dayMinutes = entries
    .filter((e) => e.date === date)
    .reduce((sum, e) => sum + e.durationMinutes, 0)
  return dayMinutes - dailyTargetMinutes
}

/** Get overtime minutes for a week (positive = over, negative = under) */
export function getOvertimeForWeek(entries: TimeEntry[], weekStartDate: string, weeklyTargetMinutes: number): number {
  const start = new Date(weekStartDate)
  const dateStrings: string[] = []
  for (let i = 0; i < 7; i++) {
    const d = new Date(start)
    d.setDate(start.getDate() + i)
    dateStrings.push(d.toISOString().split('T')[0])
  }
  const weekMinutes = entries
    .filter((e) => dateStrings.includes(e.date))
    .reduce((sum, e) => sum + e.durationMinutes, 0)
  return weekMinutes - weeklyTargetMinutes
}

/** Get overtime minutes for a month (positive = over, negative = under) */
export function getOvertimeForMonth(entries: TimeEntry[], year: number, month: number, monthlyTargetMinutes: number): number {
  const prefix = `${year}-${String(month).padStart(2, '0')}`
  const monthMinutes = entries
    .filter((e) => e.date.startsWith(prefix))
    .reduce((sum, e) => sum + e.durationMinutes, 0)
  return monthMinutes - monthlyTargetMinutes
}

/** Get cumulative overtime saldo (all entries vs target * workdays) */
export function getOvertimeSaldo(entries: TimeEntry[], targets: WorkTarget): number {
  const uniqueDates = [...new Set(entries.map((e) => e.date))]
  const workingDays = uniqueDates.filter((ds) => {
    const d = new Date(ds)
    return d.getDay() >= 1 && d.getDay() <= 5
  }).length
  const totalMinutes = entries.reduce((sum, e) => sum + e.durationMinutes, 0)
  const totalTarget = workingDays * targets.dailyHours * 60
  return totalMinutes - totalTarget
}

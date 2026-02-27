import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { toast } from 'sonner'
import { getAllTeamMembers, DEPARTMENTS } from '@/mocks/mock-db'

export interface TeamMember {
  id: string
  firstName: string
  lastName: string
  initials: string
  email: string
  phone: string
  mobile?: string
  role: string
  department: string
  status: 'online' | 'away' | 'offline' | 'dnd'
  contractType: 'Vollzeit' | 'Teilzeit' | 'Praktikum' | 'Freelance'
  workload: number // percentage, e.g. 100, 80, 60
  joinDate: string
  manager?: string
  location: string
  currentTask?: string
  projects: string[]
  skills: string[]
  notes?: string
  isActive: boolean
}

export interface HRRequest {
  id: string
  type: 'vacation' | 'sick' | 'overtime' | 'doctor' | 'homeoffice' | 'education'
  memberId: string
  memberName: string
  memberInitials: string
  startDate: string
  endDate: string
  days: number
  status: 'pending' | 'approved' | 'rejected'
  reason: string
  comment?: string
  createdAt: string
}

export interface Department {
  id: string
  name: string
  color: string
}

export interface PayrollEntry {
  id: string
  memberId: string
  memberName: string
  department: string
  employmentType: 'fulltime' | 'parttime' | 'hourly'
  grossSalary: number
  deductions: { ahv: number; pension: number; tax: number; other: number }
  netSalary: number
  month: string // '2026-01' format
  status: 'draft' | 'approved' | 'paid'
}

export interface Training {
  id: string
  name: string
  type: 'safety' | 'technical' | 'soft_skills' | 'compliance' | 'certification'
  duration: string // e.g. '2 Tage', '4 Stunden'
  mandatory: boolean
  provider: string
  validityMonths: number // 0 = no expiry
}

export interface TrainingParticipation {
  id: string
  trainingId: string
  memberId: string
  memberName: string
  status: 'completed' | 'pending' | 'expired' | 'scheduled'
  completedAt?: string
  expiresAt?: string
  certificateId?: string
}

interface TeamStore {
  members: TeamMember[]
  requests: HRRequest[]
  departments: Department[]
  payroll: PayrollEntry[]
  trainings: Training[]
  trainingParticipations: TrainingParticipation[]

  addMember: (member: Omit<TeamMember, 'id' | 'initials' | 'isActive'>) => void
  updateMember: (id: string, updates: Partial<TeamMember>) => void
  deactivateMember: (id: string) => void
  deleteMember: (id: string) => void

  addRequest: (request: Omit<HRRequest, 'id' | 'createdAt'>) => void
  approveRequest: (id: string, comment?: string) => void
  rejectRequest: (id: string, comment?: string) => void
  deleteRequest: (id: string) => void

  startPayrollRun: () => void
  addTraining: (training: Omit<Training, 'id'>) => void
  recordParticipation: (participation: Omit<TrainingParticipation, 'id'>) => void
}

function getInitials(first: string, last: string): string {
  return `${first[0] || ''}${last[0] || ''}`.toUpperCase()
}

// ---------------------------------------------------------------------------
// Initial data from central mock-db (TechVision GmbH, 18 employees)
// ---------------------------------------------------------------------------

const INITIAL_DEPARTMENTS: Department[] = DEPARTMENTS.map((d) => ({
  id: d.id,
  name: d.name,
  color: d.color,
}))

const INITIAL_MEMBERS: TeamMember[] = getAllTeamMembers()

const INITIAL_REQUESTS: HRRequest[] = [
  { id: 'hr1', type: 'vacation', memberId: 'e10', memberName: 'Sabine Fischer', memberInitials: 'SF', startDate: '2026-02-17', endDate: '2026-02-28', days: 8, status: 'approved', reason: 'Winterurlaub Oesterreich', createdAt: '2026-02-03' },
  { id: 'hr2', type: 'sick', memberId: 'e13', memberName: 'Petra Zimmermann', memberInitials: 'PZ', startDate: '2026-02-21', endDate: '2026-02-24', days: 2, status: 'approved', reason: 'Grippe', comment: 'Gute Besserung!', createdAt: '2026-02-21' },
  { id: 'hr3', type: 'overtime', memberId: 'e2', memberName: 'Markus Weber', memberInitials: 'MW', startDate: '2026-02-20', endDate: '2026-02-20', days: 1, status: 'approved', reason: 'Release-Vorbereitung API v2, 4h Ueberstunden', createdAt: '2026-02-20' },
  { id: 'hr4', type: 'doctor', memberId: 'e6', memberName: 'Tim Hartmann', memberInitials: 'TH', startDate: '2026-02-26', endDate: '2026-02-26', days: 0.5, status: 'pending', reason: 'Zahnarzttermin nachmittags', createdAt: '2026-02-22' },
  { id: 'hr5', type: 'vacation', memberId: 'e3', memberName: 'Thomas Meier', memberInitials: 'TM', startDate: '2026-03-10', endDate: '2026-03-14', days: 5, status: 'pending', reason: 'Familienurlaub Mallorca', createdAt: '2026-02-15' },
  { id: 'hr6', type: 'homeoffice', memberId: 'e8', memberName: 'Sophie Lang', memberInitials: 'SL', startDate: '2026-02-24', endDate: '2026-02-28', days: 5, status: 'approved', reason: 'Homeoffice-Woche (Usability Tests remote)', createdAt: '2026-02-20' },
  { id: 'hr7', type: 'education', memberId: 'e7', memberName: 'Lena Braun', memberInitials: 'LB', startDate: '2026-02-24', endDate: '2026-02-25', days: 2, status: 'approved', reason: 'React Advanced Workshop, TU Muenchen', createdAt: '2026-02-18' },
  { id: 'hr8', type: 'vacation', memberId: 'e12', memberName: 'Julia Hofmann', memberInitials: 'JH', startDate: '2026-03-03', endDate: '2026-03-07', days: 5, status: 'pending', reason: 'Staedtetrip Barcelona', createdAt: '2026-02-22' },
]

const INITIAL_PAYROLL: PayrollEntry[] = [
  { id: 'pay1', memberId: 'e1', memberName: 'Stefan Vogel', department: 'Geschaeftsfuehrung', employmentType: 'fulltime', grossSalary: 12500, month: '2026-01', status: 'paid', deductions: { ahv: 1188, pension: 938, tax: 2875, other: 62 }, netSalary: 7437 },
  { id: 'pay2', memberId: 'e2', memberName: 'Markus Weber', department: 'Entwicklung', employmentType: 'fulltime', grossSalary: 9800, month: '2026-01', status: 'paid', deductions: { ahv: 931, pension: 735, tax: 2058, other: 49 }, netSalary: 6027 },
  { id: 'pay3', memberId: 'e3', memberName: 'Thomas Meier', department: 'Vertrieb', employmentType: 'fulltime', grossSalary: 8500, month: '2026-01', status: 'paid', deductions: { ahv: 808, pension: 638, tax: 1700, other: 43 }, netSalary: 5311 },
  { id: 'pay4', memberId: 'e4', memberName: 'Laura Neumann', department: 'Entwicklung', employmentType: 'fulltime', grossSalary: 7200, month: '2026-01', status: 'paid', deductions: { ahv: 684, pension: 540, tax: 1368, other: 36 }, netSalary: 4572 },
  { id: 'pay5', memberId: 'e5', memberName: 'Felix Krause', department: 'Entwicklung', employmentType: 'fulltime', grossSalary: 6800, month: '2026-01', status: 'paid', deductions: { ahv: 646, pension: 510, tax: 1258, other: 34 }, netSalary: 4352 },
  { id: 'pay6', memberId: 'e6', memberName: 'Tim Hartmann', department: 'Entwicklung', employmentType: 'fulltime', grossSalary: 6200, month: '2026-01', status: 'paid', deductions: { ahv: 589, pension: 465, tax: 1116, other: 31 }, netSalary: 3999 },
  { id: 'pay7', memberId: 'e9', memberName: 'Nina Richter', department: 'Design', employmentType: 'fulltime', grossSalary: 7500, month: '2026-01', status: 'paid', deductions: { ahv: 713, pension: 563, tax: 1463, other: 38 }, netSalary: 4723 },
  { id: 'pay8', memberId: 'e8', memberName: 'Sophie Lang', department: 'Design', employmentType: 'parttime', grossSalary: 4800, month: '2026-01', status: 'draft', deductions: { ahv: 456, pension: 360, tax: 816, other: 24 }, netSalary: 3144 },
  { id: 'pay9', memberId: 'e10', memberName: 'Sabine Fischer', department: 'Vertrieb', employmentType: 'fulltime', grossSalary: 5800, month: '2026-01', status: 'draft', deductions: { ahv: 551, pension: 435, tax: 1044, other: 29 }, netSalary: 3741 },
  { id: 'pay10', memberId: 'e11', memberName: 'Kevin Baumann', department: 'Vertrieb', employmentType: 'fulltime', grossSalary: 4800, month: '2026-01', status: 'draft', deductions: { ahv: 456, pension: 360, tax: 816, other: 24 }, netSalary: 3144 },
  { id: 'pay11', memberId: 'e12', memberName: 'Julia Hofmann', department: 'Marketing', employmentType: 'fulltime', grossSalary: 5500, month: '2026-01', status: 'draft', deductions: { ahv: 523, pension: 413, tax: 963, other: 28 }, netSalary: 3573 },
  { id: 'pay12', memberId: 'e13', memberName: 'Petra Zimmermann', department: 'Buchhaltung', employmentType: 'fulltime', grossSalary: 5200, month: '2026-01', status: 'draft', deductions: { ahv: 494, pension: 390, tax: 884, other: 26 }, netSalary: 3406 },
]

const INITIAL_TRAININGS: Training[] = [
  { id: 'tr1', name: 'Erste Hilfe', type: 'safety', duration: '1 Tag', mandatory: true, provider: 'Deutsches Rotes Kreuz', validityMonths: 24 },
  { id: 'tr2', name: 'Arbeitssicherheit', type: 'safety', duration: '4 Stunden', mandatory: true, provider: 'BG ETEM', validityMonths: 12 },
  { id: 'tr3', name: 'React Advanced', type: 'technical', duration: '2 Tage', mandatory: false, provider: 'TU Muenchen Weiterbildung', validityMonths: 0 },
  { id: 'tr4', name: 'Datenschutz DSGVO', type: 'compliance', duration: '3 Stunden', mandatory: true, provider: 'IHK Muenchen', validityMonths: 12 },
  { id: 'tr5', name: 'ITIL Foundation', type: 'certification', duration: '3 Tage', mandatory: false, provider: 'SERVIEW GmbH', validityMonths: 0 },
  { id: 'tr6', name: 'Fuehrungskompetenz', type: 'soft_skills', duration: '2 Tage', mandatory: false, provider: 'WHU Executive Education', validityMonths: 0 },
]

const INITIAL_PARTICIPATIONS: TrainingParticipation[] = [
  { id: 'tp1', trainingId: 'tr1', memberId: 'e1', memberName: 'Stefan Vogel', status: 'completed', completedAt: '2025-06-15', expiresAt: '2027-06-15', certificateId: 'CERT-EH-2025-001' },
  { id: 'tp2', trainingId: 'tr4', memberId: 'e2', memberName: 'Markus Weber', status: 'completed', completedAt: '2025-11-20', expiresAt: '2026-11-20', certificateId: 'CERT-DS-2025-088' },
  { id: 'tp3', trainingId: 'tr6', memberId: 'e1', memberName: 'Stefan Vogel', status: 'completed', completedAt: '2025-10-05', certificateId: 'CERT-FK-2025-012' },
  { id: 'tp4', trainingId: 'tr1', memberId: 'e4', memberName: 'Laura Neumann', status: 'expired', completedAt: '2024-01-10', expiresAt: '2026-01-10' },
  { id: 'tp5', trainingId: 'tr3', memberId: 'e6', memberName: 'Tim Hartmann', status: 'completed', completedAt: '2025-08-22', certificateId: 'CERT-RA-2025-045' },
  { id: 'tp6', trainingId: 'tr2', memberId: 'e5', memberName: 'Felix Krause', status: 'expired', completedAt: '2024-12-01', expiresAt: '2025-12-01' },
  { id: 'tp7', trainingId: 'tr5', memberId: 'e16', memberName: 'Martin Wolf', status: 'completed', completedAt: '2025-04-18', certificateId: 'CERT-IT-2025-007' },
  { id: 'tp8', trainingId: 'tr4', memberId: 'e12', memberName: 'Julia Hofmann', status: 'pending' },
  { id: 'tp9', trainingId: 'tr1', memberId: 'e7', memberName: 'Lena Braun', status: 'scheduled' },
  { id: 'tp10', trainingId: 'tr2', memberId: 'e6', memberName: 'Tim Hartmann', status: 'scheduled' },
  { id: 'tp11', trainingId: 'tr6', memberId: 'e3', memberName: 'Thomas Meier', status: 'pending' },
  { id: 'tp12', trainingId: 'tr4', memberId: 'e15', memberName: 'Elena Schuster', status: 'completed', completedAt: '2025-09-15', expiresAt: '2026-09-15', certificateId: 'CERT-DS-2025-102' },
]

export const useTeamStore = create<TeamStore>()(
  persist(
    (set) => ({
      members: INITIAL_MEMBERS,
      requests: INITIAL_REQUESTS,
      departments: INITIAL_DEPARTMENTS,
      payroll: INITIAL_PAYROLL,
      trainings: INITIAL_TRAININGS,
      trainingParticipations: INITIAL_PARTICIPATIONS,

      addMember: (member) =>
        set((state) => ({
          members: [
            ...state.members,
            {
              ...member,
              id: `e${Date.now()}`,
              initials: getInitials(member.firstName, member.lastName),
              isActive: true,
            },
          ],
        })),

      updateMember: (id, updates) =>
        set((state) => ({
          members: state.members.map((m) =>
            m.id === id
              ? {
                  ...m,
                  ...updates,
                  initials: updates.firstName || updates.lastName
                    ? getInitials(updates.firstName ?? m.firstName, updates.lastName ?? m.lastName)
                    : m.initials,
                }
              : m,
          ),
        })),

      deactivateMember: (id) =>
        set((state) => ({
          members: state.members.map((m) =>
            m.id === id ? { ...m, isActive: false, status: 'offline' as const } : m,
          ),
        })),

      deleteMember: (id) =>
        set((state) => ({
          members: state.members.filter((m) => m.id !== id),
        })),

      addRequest: (request) =>
        set((state) => ({
          requests: [
            ...state.requests,
            { ...request, id: `hr${Date.now()}`, createdAt: new Date().toISOString().split('T')[0] },
          ],
        })),

      approveRequest: (id, comment) =>
        set((state) => ({
          requests: state.requests.map((r) =>
            r.id === id ? { ...r, status: 'approved' as const, comment } : r,
          ),
        })),

      rejectRequest: (id, comment) =>
        set((state) => ({
          requests: state.requests.map((r) =>
            r.id === id ? { ...r, status: 'rejected' as const, comment } : r,
          ),
        })),

      deleteRequest: (id) =>
        set((state) => ({
          requests: state.requests.filter((r) => r.id !== id),
        })),

      startPayrollRun: () =>
        set((state) => {
          const drafts = state.payroll.filter((p) => p.status === 'draft')
          if (drafts.length === 0) {
            toast.info('Keine Entwuerfe vorhanden')
            return state
          }
          toast.success(`${drafts.length} Lohnabrechnungen freigegeben`)
          return {
            payroll: state.payroll.map((p) =>
              p.status === 'draft' ? { ...p, status: 'approved' as const } : p,
            ),
          }
        }),

      addTraining: (training) =>
        set((state) => ({
          trainings: [
            ...state.trainings,
            { ...training, id: `tr${Date.now()}` },
          ],
        })),

      recordParticipation: (participation) =>
        set((state) => ({
          trainingParticipations: [
            ...state.trainingParticipations,
            { ...participation, id: `tp${Date.now()}` },
          ],
        })),
    }),
    { name: 'kmuhub-team' },
  ),
)

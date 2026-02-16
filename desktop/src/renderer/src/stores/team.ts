import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { toast } from 'sonner'

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

const INITIAL_DEPARTMENTS: Department[] = [
  { id: 'dev', name: 'Entwicklung', color: 'var(--primary)' },
  { id: 'design', name: 'Design', color: 'var(--info)' },
  { id: 'marketing', name: 'Marketing', color: 'var(--warning)' },
  { id: 'management', name: 'Management', color: 'var(--success)' },
  { id: 'hr', name: 'Human Resources', color: '#7c5a8a' },
  { id: 'sales', name: 'Vertrieb', color: '#c4873a' },
]

const INITIAL_MEMBERS: TeamMember[] = [
  {
    id: 'm1', firstName: 'Anna', lastName: 'Mueller', initials: 'AM',
    role: 'Projektleiterin', department: 'Management',
    email: 'anna.mueller@kmuhub.ch', phone: '+41 79 123 45 67',
    status: 'online', contractType: 'Vollzeit', workload: 100,
    joinDate: '2024-03-15', location: 'Zürich',
    currentTask: 'Sprint Planning vorbereiten',
    projects: ['Website Relaunch', 'Mobile App'],
    skills: ['Scrum', 'Jira', 'Confluence'], isActive: true,
  },
  {
    id: 'm2', firstName: 'Michael', lastName: 'Berg', initials: 'MB',
    role: 'Senior Developer', department: 'Entwicklung',
    email: 'michael.berg@kmuhub.ch', phone: '+41 79 234 56 78', mobile: '+41 76 234 56 78',
    status: 'online', contractType: 'Vollzeit', workload: 100,
    joinDate: '2024-01-10', location: 'Zürich',
    currentTask: 'API Integration testen',
    projects: ['CRM Integration', 'Security Audit'],
    skills: ['Go', 'TypeScript', 'PostgreSQL', 'Docker'], isActive: true,
  },
  {
    id: 'm3', firstName: 'Sarah', lastName: 'Klein', initials: 'SK',
    role: 'UI/UX Designer', department: 'Design',
    email: 'sarah.klein@kmuhub.ch', phone: '+41 79 345 67 89',
    status: 'away', contractType: 'Vollzeit', workload: 80,
    joinDate: '2024-06-01', location: 'Bern',
    currentTask: 'Landing Page Mockup',
    projects: ['Website Relaunch'],
    skills: ['Figma', 'Sketch', 'Tailwind CSS'], isActive: true,
  },
  {
    id: 'm4', firstName: 'Jonas', lastName: 'Diaz', initials: 'JD',
    role: 'Full-Stack Developer', department: 'Entwicklung',
    email: 'jonas.diaz@kmuhub.ch', phone: '+41 79 456 78 90',
    status: 'online', contractType: 'Vollzeit', workload: 100,
    joinDate: '2024-09-15', location: 'Zürich',
    currentTask: 'Bug Fix: iOS Login',
    projects: ['Mobile App', 'Security Audit'],
    skills: ['React', 'Node.js', 'React Native'], isActive: true,
  },
  {
    id: 'm5', firstName: 'Lisa', lastName: 'Schmidt', initials: 'LS',
    role: 'Marketing Manager', department: 'Marketing',
    email: 'lisa.schmidt@kmuhub.ch', phone: '+41 79 567 89 01',
    status: 'offline', contractType: 'Vollzeit', workload: 100,
    joinDate: '2024-04-20', location: 'Basel',
    projects: ['Marketing Kampagne'],
    skills: ['SEO', 'Google Analytics', 'Content Marketing'], isActive: true,
  },
  {
    id: 'm6', firstName: 'Peter', lastName: 'Koch', initials: 'PK',
    role: 'Security Engineer', department: 'Entwicklung',
    email: 'peter.koch@kmuhub.ch', phone: '+41 79 678 90 12',
    status: 'online', contractType: 'Vollzeit', workload: 100,
    joinDate: '2024-02-01', location: 'Zürich',
    currentTask: 'Penetration Testing',
    projects: ['Security Audit', 'Datenmigration'],
    skills: ['Go', 'Penetration Testing', 'OWASP'], isActive: true,
  },
  {
    id: 'm7', firstName: 'Eva', lastName: 'Brunner', initials: 'EB',
    role: 'Content Creator', department: 'Marketing',
    email: 'eva.brunner@kmuhub.ch', phone: '+41 79 789 01 23',
    status: 'online', contractType: 'Teilzeit', workload: 60,
    joinDate: '2025-01-15', location: 'Luzern',
    currentTask: 'Blog-Artikel: KMU Digitalisierung',
    projects: ['Marketing Kampagne', 'Website Relaunch'],
    skills: ['Copywriting', 'Social Media', 'WordPress'], isActive: true,
  },
  {
    id: 'm8', firstName: 'Tom', lastName: 'Brunner', initials: 'TB',
    role: 'Junior Developer', department: 'Entwicklung',
    email: 'tom.brunner@kmuhub.ch', phone: '+41 79 890 12 34',
    status: 'away', contractType: 'Vollzeit', workload: 100,
    joinDate: '2025-06-01', manager: 'Michael Berg', location: 'Zürich',
    currentTask: 'Unit Tests schreiben',
    projects: ['CRM Integration'],
    skills: ['TypeScript', 'React', 'Jest'], isActive: true,
  },
  {
    id: 'm9', firstName: 'Claudia', lastName: 'Frei', initials: 'CF',
    role: 'Grafikdesignerin', department: 'Design',
    email: 'claudia.frei@kmuhub.ch', phone: '+41 79 901 23 45',
    status: 'online', contractType: 'Teilzeit', workload: 80,
    joinDate: '2025-03-01', location: 'Bern',
    currentTask: 'Icon-Set für App',
    projects: ['Mobile App'],
    skills: ['Illustrator', 'Photoshop', 'Figma'], isActive: true,
  },
  {
    id: 'm10', firstName: 'Markus', lastName: 'Steiner', initials: 'MS',
    role: 'Sales Manager', department: 'Vertrieb',
    email: 'markus.steiner@kmuhub.ch', phone: '+41 79 012 34 56',
    status: 'online', contractType: 'Vollzeit', workload: 100,
    joinDate: '2024-11-01', location: 'Zürich',
    currentTask: 'Demo für Meier AG vorbereiten',
    projects: ['CRM Integration'],
    skills: ['Verhandlung', 'CRM', 'Präsentation'], isActive: true,
  },
  {
    id: 'm11', firstName: 'Laura', lastName: 'Weber', initials: 'LW',
    role: 'HR Managerin', department: 'Human Resources',
    email: 'laura.weber@kmuhub.ch', phone: '+41 79 123 99 88',
    status: 'online', contractType: 'Vollzeit', workload: 100,
    joinDate: '2024-05-15', location: 'Zürich',
    currentTask: 'Quartal-Review vorbereiten',
    projects: [],
    skills: ['Recruiting', 'Arbeitsrecht', 'SAP HR'], isActive: true,
  },
  {
    id: 'm12', firstName: 'Nils', lastName: 'Hofer', initials: 'NH',
    role: 'DevOps Engineer', department: 'Entwicklung',
    email: 'nils.hofer@kmuhub.ch', phone: '+41 79 456 00 11',
    status: 'dnd', contractType: 'Vollzeit', workload: 100,
    joinDate: '2025-02-01', location: 'Zürich',
    currentTask: 'Kubernetes Cluster Upgrade',
    projects: ['Security Audit', 'Datenmigration'],
    skills: ['Kubernetes', 'Terraform', 'AWS', 'Linux'], isActive: true,
  },
]

const INITIAL_REQUESTS: HRRequest[] = [
  { id: 'hr1', type: 'vacation', memberId: 'm3', memberName: 'Sarah Klein', memberInitials: 'SK', startDate: '2026-02-17', endDate: '2026-02-21', days: 5, status: 'pending', reason: 'Skiurlaub', createdAt: '2026-02-03' },
  { id: 'hr2', type: 'sick', memberId: 'm5', memberName: 'Lisa Schmidt', memberInitials: 'LS', startDate: '2026-02-08', endDate: '2026-02-08', days: 1, status: 'approved', reason: 'Krankheit', comment: 'Gute Besserung!', createdAt: '2026-02-08' },
  { id: 'hr3', type: 'overtime', memberId: 'm2', memberName: 'Michael Berg', memberInitials: 'MB', startDate: '2026-02-07', endDate: '2026-02-07', days: 1, status: 'approved', reason: 'Release-Vorbereitung, 3h Überstunden', createdAt: '2026-02-07' },
  { id: 'hr4', type: 'doctor', memberId: 'm4', memberName: 'Jonas Diaz', memberInitials: 'JD', startDate: '2026-02-12', endDate: '2026-02-12', days: 0.5, status: 'pending', reason: 'Arzttermin nachmittags', createdAt: '2026-02-10' },
  { id: 'hr5', type: 'vacation', memberId: 'm10', memberName: 'Markus Steiner', memberInitials: 'MS', startDate: '2026-03-02', endDate: '2026-03-06', days: 5, status: 'pending', reason: 'Familienurlaub Mallorca', createdAt: '2026-02-05' },
  { id: 'hr6', type: 'homeoffice', memberId: 'm7', memberName: 'Eva Brunner', memberInitials: 'EB', startDate: '2026-02-10', endDate: '2026-02-14', days: 5, status: 'approved', reason: 'Homeoffice-Woche (Kind krank)', createdAt: '2026-02-09' },
  { id: 'hr7', type: 'education', memberId: 'm8', memberName: 'Tom Brunner', memberInitials: 'TB', startDate: '2026-02-24', endDate: '2026-02-25', days: 2, status: 'pending', reason: 'React Advanced Kurs', createdAt: '2026-02-06' },
]

const INITIAL_PAYROLL: PayrollEntry[] = [
  {
    id: 'pay1', memberId: 'm1', memberName: 'Anna Mueller', department: 'Management',
    employmentType: 'fulltime', grossSalary: 8500, month: '2026-01', status: 'paid',
    deductions: { ahv: 1105, pension: 595, tax: 1020, other: 45 },
    netSalary: 5735,
  },
  {
    id: 'pay2', memberId: 'm2', memberName: 'Michael Berg', department: 'Entwicklung',
    employmentType: 'fulltime', grossSalary: 8200, month: '2026-01', status: 'paid',
    deductions: { ahv: 1066, pension: 574, tax: 984, other: 45 },
    netSalary: 5531,
  },
  {
    id: 'pay3', memberId: 'm3', memberName: 'Sarah Klein', department: 'Design',
    employmentType: 'parttime', grossSalary: 5600, month: '2026-01', status: 'approved',
    deductions: { ahv: 728, pension: 392, tax: 672, other: 35 },
    netSalary: 3773,
  },
  {
    id: 'pay4', memberId: 'm4', memberName: 'Jonas Diaz', department: 'Entwicklung',
    employmentType: 'fulltime', grossSalary: 7200, month: '2026-01', status: 'approved',
    deductions: { ahv: 936, pension: 504, tax: 864, other: 40 },
    netSalary: 4856,
  },
  {
    id: 'pay5', memberId: 'm5', memberName: 'Lisa Schmidt', department: 'Marketing',
    employmentType: 'fulltime', grossSalary: 7000, month: '2026-01', status: 'draft',
    deductions: { ahv: 910, pension: 490, tax: 840, other: 40 },
    netSalary: 4720,
  },
  {
    id: 'pay6', memberId: 'm6', memberName: 'Peter Koch', department: 'Entwicklung',
    employmentType: 'fulltime', grossSalary: 7800, month: '2026-01', status: 'draft',
    deductions: { ahv: 1014, pension: 546, tax: 936, other: 42 },
    netSalary: 5262,
  },
  {
    id: 'pay7', memberId: 'm7', memberName: 'Eva Brunner', department: 'Marketing',
    employmentType: 'parttime', grossSalary: 4500, month: '2026-01', status: 'draft',
    deductions: { ahv: 585, pension: 315, tax: 540, other: 30 },
    netSalary: 3030,
  },
  {
    id: 'pay8', memberId: 'm8', memberName: 'Tom Brunner', department: 'Entwicklung',
    employmentType: 'fulltime', grossSalary: 5800, month: '2026-01', status: 'draft',
    deductions: { ahv: 754, pension: 406, tax: 696, other: 35 },
    netSalary: 3909,
  },
]

const INITIAL_TRAININGS: Training[] = [
  { id: 'tr1', name: 'Erste Hilfe', type: 'safety', duration: '1 Tag', mandatory: true, provider: 'Schweizerisches Rotes Kreuz', validityMonths: 24 },
  { id: 'tr2', name: 'Arbeitssicherheit', type: 'safety', duration: '4 Stunden', mandatory: true, provider: 'SUVA', validityMonths: 12 },
  { id: 'tr3', name: 'Excel Advanced', type: 'technical', duration: '2 Tage', mandatory: false, provider: 'Digicomp', validityMonths: 0 },
  { id: 'tr4', name: 'Datenschutz DSGVO', type: 'compliance', duration: '3 Stunden', mandatory: true, provider: 'SwissLegal Academy', validityMonths: 12 },
  { id: 'tr5', name: 'Gabelstapler-Schein', type: 'certification', duration: '3 Tage', mandatory: false, provider: 'TÜV Schweiz', validityMonths: 60 },
  { id: 'tr6', name: 'Führungskompetenz', type: 'soft_skills', duration: '2 Tage', mandatory: false, provider: 'HSG Executive Education', validityMonths: 0 },
]

const INITIAL_PARTICIPATIONS: TrainingParticipation[] = [
  { id: 'tp1', trainingId: 'tr1', memberId: 'm1', memberName: 'Anna Mueller', status: 'completed', completedAt: '2025-06-15', expiresAt: '2027-06-15', certificateId: 'CERT-EH-2025-001' },
  { id: 'tp2', trainingId: 'tr2', memberId: 'm1', memberName: 'Anna Mueller', status: 'completed', completedAt: '2025-09-10', expiresAt: '2026-09-10', certificateId: 'CERT-AS-2025-014' },
  { id: 'tp3', trainingId: 'tr4', memberId: 'm2', memberName: 'Michael Berg', status: 'completed', completedAt: '2025-11-20', expiresAt: '2026-11-20', certificateId: 'CERT-DS-2025-088' },
  { id: 'tp4', trainingId: 'tr1', memberId: 'm3', memberName: 'Sarah Klein', status: 'expired', completedAt: '2024-01-10', expiresAt: '2026-01-10' },
  { id: 'tp5', trainingId: 'tr3', memberId: 'm4', memberName: 'Jonas Diaz', status: 'completed', completedAt: '2025-08-22', certificateId: 'CERT-EX-2025-045' },
  { id: 'tp6', trainingId: 'tr6', memberId: 'm1', memberName: 'Anna Mueller', status: 'completed', completedAt: '2025-10-05', certificateId: 'CERT-FK-2025-012' },
  { id: 'tp7', trainingId: 'tr2', memberId: 'm6', memberName: 'Peter Koch', status: 'expired', completedAt: '2024-12-01', expiresAt: '2025-12-01' },
  { id: 'tp8', trainingId: 'tr5', memberId: 'm10', memberName: 'Markus Steiner', status: 'completed', completedAt: '2025-04-18', expiresAt: '2030-04-18', certificateId: 'CERT-GS-2025-007' },
  { id: 'tp9', trainingId: 'tr4', memberId: 'm5', memberName: 'Lisa Schmidt', status: 'pending', },
  { id: 'tp10', trainingId: 'tr1', memberId: 'm8', memberName: 'Tom Brunner', status: 'scheduled', },
  { id: 'tp11', trainingId: 'tr2', memberId: 'm4', memberName: 'Jonas Diaz', status: 'scheduled', },
  { id: 'tp12', trainingId: 'tr6', memberId: 'm11', memberName: 'Laura Weber', status: 'pending', },
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
              id: `m${Date.now()}`,
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
            toast.info('Keine Entwürfe vorhanden')
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

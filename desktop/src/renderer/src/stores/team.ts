import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { toast } from 'sonner'
import { DEMO_MODE } from '@/mocks/demo-mode'

// Members, requests, departments → migrated to API (useEmployees, useLeaveRequests)
// Only training & payroll remain as Zustand mocks (no backend API yet)

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

export interface TrainingMaterial {
  id: string
  name: string
  type: 'pdf' | 'video' | 'link' | 'slides'
}

export interface Training {
  id: string
  name: string
  type: 'safety' | 'technical' | 'soft_skills' | 'compliance' | 'certification'
  duration: string // e.g. '2 Tage', '4 Stunden'
  mandatory: boolean
  provider: string
  validityMonths: number // 0 = no expiry
  description?: string
  objectives?: string[] // Lernziele
  materials?: TrainingMaterial[] // Unterlagen / Begleitmaterial
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
  payroll: PayrollEntry[]
  trainings: Training[]
  trainingParticipations: TrainingParticipation[]

  startPayrollRun: () => void
  addTraining: (training: Omit<Training, 'id'>) => void
  recordParticipation: (participation: Omit<TrainingParticipation, 'id'>) => void
}

const INITIAL_PAYROLL: PayrollEntry[] = [
  { id: 'pay1', memberId: 'e1', memberName: 'Stefan Vogel', department: 'Geschäftsführung', employmentType: 'fulltime', grossSalary: 12500, month: '2026-01', status: 'paid', deductions: { ahv: 1188, pension: 938, tax: 2875, other: 62 }, netSalary: 7437 },
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
  {
    id: 'tr1', name: 'Erste Hilfe', type: 'safety', duration: '1 Tag', mandatory: true, provider: 'Deutsches Rotes Kreuz', validityMonths: 24,
    description: 'Grundausbildung in lebensrettenden Sofortmaßnahmen am Arbeitsplatz. Praxisnahe Übungen zu Wundversorgung, stabiler Seitenlage und Reanimation.',
    objectives: ['Lebensrettende Sofortmaßnahmen durchführen', 'Wunden fachgerecht versorgen', 'Notruf korrekt absetzen und Unfallstelle sichern'],
    materials: [
      { id: 'm-tr1-1', name: 'Erste-Hilfe-Leitfaden.pdf', type: 'pdf' },
      { id: 'm-tr1-2', name: 'Reanimation – Schulungsvideo', type: 'video' },
      { id: 'm-tr1-3', name: 'Notfall-Checkliste.pdf', type: 'pdf' },
    ],
  },
  {
    id: 'tr2', name: 'Arbeitssicherheit', type: 'safety', duration: '4 Stunden', mandatory: true, provider: 'BG ETEM', validityMonths: 12,
    description: 'Jährliche Unterweisung zu Arbeitsschutz und Gefährdungsbeurteilung gemäß ArbSchG. Pflichtschulung für alle Mitarbeitenden.',
    objectives: ['Gefährdungen am Arbeitsplatz erkennen', 'Schutzmaßnahmen korrekt anwenden', 'Meldewege bei Unfällen kennen'],
    materials: [
      { id: 'm-tr2-1', name: 'Unterweisung-Arbeitssicherheit.pdf', type: 'pdf' },
      { id: 'm-tr2-2', name: 'Präsentation-Grundlagen.pptx', type: 'slides' },
    ],
  },
  {
    id: 'tr3', name: 'React Advanced', type: 'technical', duration: '2 Tage', mandatory: false, provider: 'TU München Weiterbildung', validityMonths: 0,
    description: 'Vertiefungskurs zu modernen React-Patterns: Performance-Optimierung, State-Management und Server Components.',
    objectives: ['Komplexe Komponenten performant strukturieren', 'State-Management-Strategien bewerten', 'Render-Performance messen und optimieren'],
    materials: [
      { id: 'm-tr3-1', name: 'Kursunterlagen-React-Advanced.pdf', type: 'pdf' },
      { id: 'm-tr3-2', name: 'Code-Beispiele (Repository)', type: 'link' },
    ],
  },
  {
    id: 'tr4', name: 'Datenschutz DSGVO', type: 'compliance', duration: '3 Stunden', mandatory: true, provider: 'IHK München', validityMonths: 12,
    description: 'Pflichtschulung zur Datenschutz-Grundverordnung: Umgang mit personenbezogenen Daten, Betroffenenrechte und Meldepflichten.',
    objectives: ['Grundsätze der DSGVO anwenden', 'Betroffenenrechte korrekt umsetzen', 'Datenpannen fristgerecht melden'],
    materials: [
      { id: 'm-tr4-1', name: 'DSGVO-Handbuch.pdf', type: 'pdf' },
      { id: 'm-tr4-2', name: 'Fallbeispiele-Datenschutz.pdf', type: 'pdf' },
    ],
  },
  {
    id: 'tr5', name: 'ITIL Foundation', type: 'certification', duration: '3 Tage', mandatory: false, provider: 'SERVIEW GmbH', validityMonths: 0,
    description: 'Zertifizierungskurs zum ITIL-4-Foundation-Framework für IT-Service-Management. Schließt mit offizieller Prüfung ab.',
    objectives: ['ITIL-4-Grundkonzepte verstehen', 'Service-Wertschöpfungskette anwenden', 'Auf die Foundation-Prüfung vorbereiten'],
    materials: [
      { id: 'm-tr5-1', name: 'ITIL-Foundation-Skript.pdf', type: 'pdf' },
      { id: 'm-tr5-2', name: 'Übungsprüfung.pdf', type: 'pdf' },
      { id: 'm-tr5-3', name: 'Webinar-Aufzeichnung', type: 'video' },
    ],
  },
  {
    id: 'tr6', name: 'Führungskompetenz', type: 'soft_skills', duration: '2 Tage', mandatory: false, provider: 'WHU Executive Education', validityMonths: 0,
    description: 'Entwicklung von Führungsfähigkeiten für Team- und Projektleitende: Mitarbeitergespräche, Feedback-Kultur und Konfliktlösung.',
    objectives: ['Mitarbeitergespräche wirksam führen', 'Konstruktives Feedback geben', 'Konflikte im Team moderieren'],
    materials: [
      { id: 'm-tr6-1', name: 'Leadership-Workbook.pdf', type: 'pdf' },
      { id: 'm-tr6-2', name: 'Gesprächsleitfaden.pdf', type: 'pdf' },
    ],
  },
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
      // Sample data seeds only in demo mode. Production must start empty:
      // these are invented salary records with real-looking names and gross
      // amounts, and persisting them would show a user fabricated payroll
      // for colleagues that do not exist.
      payroll: DEMO_MODE ? INITIAL_PAYROLL : [],
      trainings: DEMO_MODE ? INITIAL_TRAININGS : [],
      trainingParticipations: DEMO_MODE ? INITIAL_PARTICIPATIONS : [],

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
    {
      name: 'cosmi-team',
      version: 1,
      // v1 drops the seeded payroll/training records that older builds wrote
      // to localStorage. User-created records carry `<prefix><timestamp>` ids
      // and are preserved; the seed ids are derived from the constants above
      // so this filter cannot drift away from them.
      migrate: (persisted) => {
        const state = (persisted ?? {}) as Partial<TeamStore>
        if (DEMO_MODE) return state as TeamStore
        const seeded = <T extends { id: string }>(rows: T[]): Set<string> =>
          new Set(rows.map((r) => r.id))
        const mockPayroll = seeded(INITIAL_PAYROLL)
        const mockTrainings = seeded(INITIAL_TRAININGS)
        const mockParticipations = seeded(INITIAL_PARTICIPATIONS)
        return {
          ...state,
          payroll: (state.payroll ?? []).filter((p) => !mockPayroll.has(p.id)),
          trainings: (state.trainings ?? []).filter((t) => !mockTrainings.has(t.id)),
          trainingParticipations: (state.trainingParticipations ?? []).filter(
            (p) => !mockParticipations.has(p.id),
          ),
        } as TeamStore
      },
    },
  ),
)

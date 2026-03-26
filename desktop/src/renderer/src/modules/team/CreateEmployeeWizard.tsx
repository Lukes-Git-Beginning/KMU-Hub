/**
 * 5-step wizard for creating a new employee (Admin/IT).
 *
 * Steps: 1. Account, 2. Employment, 3. Personal, 4. Documents, 5. Summary
 * Fullscreen overlay with left sidebar step navigation.
 */
import { useState, useCallback, useMemo } from 'react'
import {
  X,
  ChevronLeft,
  ChevronRight,
  User,
  Briefcase,
  Home,
  FileText,
  CheckCircle,
  Check,
  Eye,
  EyeOff,
  Upload,
  Trash2,
  Loader2,
  Maximize2,
  UserPlus,
  Shield,
  MapPin,
  Phone,
  Calendar,
  Building2,
  AlertTriangle,
  Mail,
} from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Button } from '@/components/ui/button'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { useCreateEmployee, useEmployees } from '@/api/hooks/hr-hooks'
import type { ContractType } from '@/api/hr-types'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface EmployeeFormData {
  // Step 1: Account
  firstName: string
  lastName: string
  email: string
  phone: string
  temporaryPassword: string
  roles: string[]

  // Step 2: Employment
  department: string
  positionTitle: string
  contractType: ContractType
  workloadPercent: number
  workDaysPerWeek: number
  annualLeaveDays: number
  startDate: string
  managerUserId: string
  location: string

  // Step 3: Personal
  addressStreet: string
  addressCity: string
  addressPostalCode: string
  addressCountry: string
  emergencyContactName: string
  emergencyContactPhone: string

  // Step 4: Documents (local state only)
  documents: DocumentEntry[]

  // Step 5: Options
  sendInviteEmail: boolean
}

interface DocumentEntry {
  id: string
  category: string
  fileName: string
  notes: string
}

const INITIAL_FORM: EmployeeFormData = {
  firstName: '',
  lastName: '',
  email: '',
  phone: '',
  temporaryPassword: '',
  roles: ['member'],
  department: '',
  positionTitle: '',
  contractType: 'full_time',
  workloadPercent: 100,
  workDaysPerWeek: 5,
  annualLeaveDays: 25,
  startDate: new Date().toISOString().split('T')[0],
  managerUserId: '',
  location: '',
  addressStreet: '',
  addressCity: '',
  addressPostalCode: '',
  addressCountry: 'CH',
  emergencyContactName: '',
  emergencyContactPhone: '',
  documents: [],
  sendInviteEmail: true,
}

// ---------------------------------------------------------------------------
// Steps config
// ---------------------------------------------------------------------------

const STEPS = [
  { label: 'Account', icon: User, desc: 'Login & Systemrollen' },
  { label: 'Arbeit', icon: Briefcase, desc: 'Vertrag & Abteilung' },
  { label: 'Persoenlich', icon: Home, desc: 'Adresse & Notfall' },
  { label: 'Dokumente', icon: FileText, desc: 'Vertraege & Unterlagen' },
  { label: 'Uebersicht', icon: CheckCircle, desc: 'Pruefen & Erstellen' },
]

const ROLE_OPTIONS = [
  { id: 'admin', label: 'Admin', desc: 'Voller Systemzugriff', icon: Shield },
  { id: 'manager', label: 'Projektleiter', desc: 'Projekte & Team', icon: Briefcase },
  { id: 'member', label: 'Mitarbeiter', desc: 'Standard-Zugriff', icon: User },
  { id: 'hr', label: 'HR-Manager', desc: 'Team & HR', icon: UserPlus },
  { id: 'it_support', label: 'IT-Support', desc: 'Infrastruktur', icon: Building2 },
]

const CONTRACT_LABELS: Record<string, string> = {
  full_time: 'Vollzeit',
  part_time: 'Teilzeit',
  praktikum: 'Praktikum',
  freelance: 'Freelance',
}

const DOC_CATEGORIES = [
  'Arbeitsvertrag',
  'Personalausweis',
  'Zeugnis',
  'Zertifikat',
  'Sozialversicherung',
  'Steuer',
  'Sonstiges',
]

// ---------------------------------------------------------------------------
// Main wizard
// ---------------------------------------------------------------------------

interface WizardProps {
  onClose: () => void
  /** True when rendered inside a pop-out Electron BrowserWindow */
  isWindow?: boolean
  /** Pre-filled form data (used by pop-out window to restore state) */
  initialData?: EmployeeFormData
}

export function CreateEmployeeWizard({ onClose, isWindow, initialData }: WizardProps) {
  const [step, setStep] = useState(0)
  const [form, setForm] = useState<EmployeeFormData>(initialData ?? { ...INITIAL_FORM })
  const createEmployee = useCreateEmployee()
  const { data: employeesData } = useEmployees()
  const departments = useMemo(() => {
    const employees = employeesData?.employees ?? []
    const deptNames = [...new Set(employees.map((e) => e.department).filter(Boolean))] as string[]
    return deptNames.map((name) => ({ id: name, name }))
  }, [employeesData])

  const update = useCallback(
    <K extends keyof EmployeeFormData>(key: K, value: EmployeeFormData[K]) => {
      setForm((prev) => ({ ...prev, [key]: value }))
    },
    [],
  )

  const canAdvance = useCallback(() => {
    if (step === 0) {
      return (
        !!form.firstName.trim() &&
        !!form.lastName.trim() &&
        !!form.email.trim() &&
        !!form.temporaryPassword.trim() &&
        form.roles.length > 0
      )
    }
    if (step === 1) {
      return !!form.startDate
    }
    return true
  }, [step, form])

  const stepComplete = useCallback(
    (idx: number) => {
      if (idx === 0) {
        return (
          !!form.firstName.trim() &&
          !!form.lastName.trim() &&
          !!form.email.trim() &&
          !!form.temporaryPassword.trim() &&
          form.roles.length > 0
        )
      }
      if (idx === 1) return !!form.startDate
      if (idx === 2) return true // optional
      if (idx === 3) return true // optional
      return false
    },
    [form],
  )

  const initials = useMemo(() => {
    const f = form.firstName.trim().charAt(0).toUpperCase()
    const l = form.lastName.trim().charAt(0).toUpperCase()
    return f || l ? `${f}${l}` : ''
  }, [form.firstName, form.lastName])

  const handleSubmit = () => {
    createEmployee.mutate(
      {
        firstName: form.firstName.trim(),
        lastName: form.lastName.trim(),
        email: form.email.trim(),
        phone: form.phone.trim() || undefined,
        temporaryPassword: form.temporaryPassword,
        roles: form.roles,
        department: form.department || undefined,
        positionTitle: form.positionTitle || undefined,
        contractType: form.contractType,
        workDaysPerWeek: form.workDaysPerWeek,
        annualLeaveDays: form.annualLeaveDays,
        workloadPercent: form.workloadPercent,
        managerUserId: form.managerUserId || undefined,
        startDate: form.startDate,
        location: form.location || undefined,
        addressStreet: form.addressStreet || undefined,
        addressCity: form.addressCity || undefined,
        addressPostalCode: form.addressPostalCode || undefined,
        addressCountry: form.addressCountry,
        emergencyContactName: form.emergencyContactName || undefined,
        emergencyContactPhone: form.emergencyContactPhone || undefined,
        sendInviteEmail: form.sendInviteEmail,
      },
      {
        onSuccess: () => {
          if (isWindow) {
            setTimeout(() => window.electronAPI?.window.close(), 300)
          } else {
            onClose()
          }
        },
      },
    )
  }

  const handlePopOut = () => {
    try {
      localStorage.setItem('kmuhub-employee-wizard-draft', JSON.stringify(form))
    } catch { /* ignore */ }
    window.electronAPI?.employeeWizard?.openWindow()
    onClose()
  }

  return (
    <Dialog open onOpenChange={(open) => { if (!open) { if (isWindow) { window.electronAPI?.window.close() } else { onClose() } } }}>
      <DialogContent className="gap-0 p-0 max-w-full w-screen h-screen rounded-none border-0 [&>button:last-child]:hidden">
    <div className="flex flex-col h-full">
      {/* ── Header ── */}
      <div
        className="flex items-center justify-between border-b border-border px-5 py-2.5"
        style={isWindow ? ({ WebkitAppRegion: 'drag' } as React.CSSProperties) : undefined}
      >
        <div className="flex items-center gap-3">
          {/* Avatar preview */}
          <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/10 text-primary text-xs font-bold">
            {initials || <User className="h-4 w-4" />}
          </div>
          <div>
            <DialogTitle className="text-sm font-semibold text-foreground leading-tight">
              Neuen Mitarbeiter erstellen
            </DialogTitle>
            <DialogDescription className="sr-only">Wizard zum Erstellen eines neuen Mitarbeiters</DialogDescription>
            {(form.firstName || form.lastName) && (
              <p className="text-[11px] text-muted-foreground leading-tight">
                {form.firstName} {form.lastName}
                {form.email && <span className="ml-1.5 opacity-70">&lt;{form.email}&gt;</span>}
              </p>
            )}
          </div>
        </div>

        <div className="flex items-center gap-1" style={isWindow ? ({ WebkitAppRegion: 'no-drag' } as React.CSSProperties) : undefined}>
          {!isWindow && (
            <button
              onClick={handlePopOut}
              className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
              title="Als Fenster oeffnen"
            >
              <Maximize2 className="h-4 w-4" />
            </button>
          )}
          <button
            onClick={isWindow ? () => window.electronAPI?.window.close() : onClose}
            className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
          >
            <X className="h-5 w-5" />
          </button>
        </div>
      </div>

      {/* ── Body: sidebar + content ── */}
      <div className="flex flex-1 overflow-hidden">
        {/* Step sidebar */}
        <div className="w-56 shrink-0 border-r border-border bg-card/50 px-3 py-5 flex flex-col gap-1">
          {STEPS.map((s, idx) => {
            const Icon = s.icon
            const isActive = idx === step
            const isDone = idx < step || (idx < 4 && stepComplete(idx) && idx < step)
            const isFuture = idx > step

            return (
              <button
                key={idx}
                onClick={() => idx <= step && setStep(idx)}
                disabled={idx > step}
                className={`group flex items-start gap-3 rounded-lg px-3 py-2.5 text-left transition-all ${
                  isActive
                    ? 'bg-primary/10 text-primary'
                    : isDone
                      ? 'text-foreground hover:bg-secondary/60 cursor-pointer'
                      : 'text-muted-foreground/60 cursor-not-allowed'
                }`}
              >
                <div
                  className={`mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-full transition-colors ${
                    isActive
                      ? 'bg-primary text-primary-foreground'
                      : isDone
                        ? 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-400'
                        : 'bg-secondary/70 text-muted-foreground'
                  }`}
                >
                  {isDone && !isActive ? (
                    <Check className="h-3.5 w-3.5" />
                  ) : (
                    <Icon className="h-3.5 w-3.5" />
                  )}
                </div>
                <div className="min-w-0">
                  <p
                    className={`text-xs font-semibold leading-tight ${
                      isActive ? 'text-primary' : isFuture ? 'text-muted-foreground/60' : 'text-foreground'
                    }`}
                  >
                    {s.label}
                  </p>
                  <p
                    className={`text-[11px] leading-tight mt-0.5 ${
                      isActive ? 'text-primary/70' : 'text-muted-foreground'
                    }`}
                  >
                    {s.desc}
                  </p>
                </div>
              </button>
            )
          })}

          {/* Progress indicator */}
          <div className="mt-auto pt-4 px-3">
            <div className="h-1.5 w-full rounded-full bg-secondary overflow-hidden">
              <div
                className="h-full rounded-full bg-primary transition-all duration-300"
                style={{ width: `${((step + 1) / STEPS.length) * 100}%` }}
              />
            </div>
            <p className="text-[10px] text-muted-foreground mt-1.5 text-center">
              Schritt {step + 1} von {STEPS.length}
            </p>
          </div>
        </div>

        {/* Main content area */}
        <div className="flex-1 overflow-y-auto">
          <div className="mx-auto max-w-3xl px-8 py-6">
            {step === 0 && <AccountStep form={form} update={update} />}
            {step === 1 && (
              <EmploymentStep form={form} update={update} departments={departments} />
            )}
            {step === 2 && <PersonalStep form={form} update={update} />}
            {step === 3 && <DocumentsStep form={form} update={update} />}
            {step === 4 && <SummaryStep form={form} update={update} />}
          </div>
        </div>
      </div>

      {/* ── Footer ── */}
      <div className="flex items-center justify-between border-t border-border px-6 py-3">
        <button
          onClick={() => setStep((s) => Math.max(0, s - 1))}
          disabled={step === 0}
          className="flex items-center gap-1.5 rounded-lg border border-border px-4 py-2 text-xs font-medium text-foreground hover:bg-secondary transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        >
          <ChevronLeft className="h-3.5 w-3.5" />
          Zurueck
        </button>

        {step < 4 ? (
          <button
            onClick={() => setStep((s) => Math.min(4, s + 1))}
            disabled={!canAdvance()}
            className="flex items-center gap-1.5 rounded-lg bg-primary px-5 py-2 text-xs font-medium text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            Weiter
            <ChevronRight className="h-3.5 w-3.5" />
          </button>
        ) : (
          <button
            onClick={handleSubmit}
            disabled={createEmployee.isPending}
            className="flex items-center gap-1.5 rounded-lg bg-primary px-5 py-2 text-xs font-medium text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {createEmployee.isPending ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <CheckCircle className="h-3.5 w-3.5" />
            )}
            {createEmployee.isPending ? 'Erstelle...' : 'Mitarbeiter erstellen'}
          </button>
        )}
      </div>
    </div>
      </DialogContent>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

type UpdateFn = <K extends keyof EmployeeFormData>(key: K, value: EmployeeFormData[K]) => void

function SectionCard({
  title,
  icon: Icon,
  children,
}: {
  title: string
  icon: React.ComponentType<{ className?: string }>
  children: React.ReactNode
}) {
  return (
    <div className="rounded-xl border border-border bg-card/40 p-5">
      <div className="flex items-center gap-2 mb-4">
        <Icon className="h-4 w-4 text-primary" />
        <h3 className="text-sm font-semibold text-foreground">{title}</h3>
      </div>
      {children}
    </div>
  )
}

function FormField({
  label,
  required,
  hint,
  children,
}: {
  label: string
  required?: boolean
  hint?: string
  children: React.ReactNode
}) {
  return (
    <div className="space-y-1.5">
      <Label className="text-xs">
        {label}
        {required && <span className="text-destructive ml-0.5">*</span>}
      </Label>
      {children}
      {hint && <p className="text-[11px] text-muted-foreground">{hint}</p>}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Step 1: Account
// ---------------------------------------------------------------------------

function AccountStep({ form, update }: { form: EmployeeFormData; update: UpdateFn }) {
  const [showPw, setShowPw] = useState(false)

  const toggleRole = (roleId: string) => {
    if (form.roles.includes(roleId)) {
      if (form.roles.length === 1) return
      update(
        'roles',
        form.roles.filter((r) => r !== roleId),
      )
    } else {
      update('roles', [...form.roles, roleId])
    }
  }

  return (
    <div className="space-y-5">
      <SectionCard title="Kontodaten" icon={Mail}>
        <div className="grid grid-cols-2 gap-4">
          <FormField label="Vorname" required>
            <Input
              autoFocus
              placeholder="Max"
              value={form.firstName}
              onChange={(e) => update('firstName', e.target.value)}
            />
          </FormField>
          <FormField label="Nachname" required>
            <Input
              placeholder="Muster"
              value={form.lastName}
              onChange={(e) => update('lastName', e.target.value)}
            />
          </FormField>
        </div>

        <div className="grid grid-cols-2 gap-4 mt-4">
          <FormField label="E-Mail" required>
            <Input
              type="email"
              placeholder="max.muster@firma.ch"
              value={form.email}
              onChange={(e) => update('email', e.target.value)}
            />
          </FormField>
          <FormField label="Telefon">
            <Input
              placeholder="+41 79 123 45 67"
              value={form.phone}
              onChange={(e) => update('phone', e.target.value)}
            />
          </FormField>
        </div>

        <div className="mt-4">
          <FormField
            label="Temporaeres Passwort"
            required
            hint="Der Mitarbeiter wird aufgefordert, das Passwort beim ersten Login zu aendern."
          >
            <div className="relative">
              <Input
                type={showPw ? 'text' : 'password'}
                placeholder="Min. 8 Zeichen"
                value={form.temporaryPassword}
                onChange={(e) => update('temporaryPassword', e.target.value)}
              />
              <button
                type="button"
                onClick={() => setShowPw(!showPw)}
                className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              >
                {showPw ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
          </FormField>
        </div>
      </SectionCard>

      <SectionCard title="Systemrollen" icon={Shield}>
        <p className="text-xs text-muted-foreground mb-3">
          Mindestens eine Rolle auswaehlen. Mehrfachauswahl moeglich.
        </p>
        <div className="grid grid-cols-3 gap-2">
          {ROLE_OPTIONS.map((role) => {
            const RIcon = role.icon
            const selected = form.roles.includes(role.id)
            return (
              <button
                key={role.id}
                onClick={() => toggleRole(role.id)}
                className={`flex items-center gap-2.5 rounded-lg border px-3 py-2.5 text-left transition-all ${
                  selected
                    ? 'border-primary bg-primary/5 ring-1 ring-primary/20'
                    : 'border-border hover:bg-secondary/50 hover:border-border'
                }`}
              >
                <div
                  className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-md transition-colors ${
                    selected
                      ? 'bg-primary/15 text-primary'
                      : 'bg-secondary text-muted-foreground'
                  }`}
                >
                  <RIcon className="h-3.5 w-3.5" />
                </div>
                <div className="min-w-0">
                  <p className="text-xs font-medium text-foreground leading-tight truncate">{role.label}</p>
                  <p className="text-[10px] text-muted-foreground leading-tight truncate">{role.desc}</p>
                </div>
                {selected && (
                  <Check className="ml-auto h-3.5 w-3.5 shrink-0 text-primary" />
                )}
              </button>
            )
          })}
        </div>
      </SectionCard>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Step 2: Employment
// ---------------------------------------------------------------------------

function EmploymentStep({
  form,
  update,
  departments,
}: {
  form: EmployeeFormData
  update: UpdateFn
  departments: { id: string; name: string }[]
}) {
  return (
    <div className="space-y-5">
      <SectionCard title="Arbeitsverhaeltnis" icon={Briefcase}>
        <div className="grid grid-cols-2 gap-4">
          <FormField label="Abteilung">
            <Select value={form.department} onValueChange={(v) => update('department', v)}>
              <SelectTrigger>
                <SelectValue placeholder="Waehlen..." />
              </SelectTrigger>
              <SelectContent>
                {departments.map((d) => (
                  <SelectItem key={d.id} value={d.name}>
                    {d.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FormField>
          <FormField label="Position">
            <Input
              placeholder="z.B. Software Engineer"
              value={form.positionTitle}
              onChange={(e) => update('positionTitle', e.target.value)}
            />
          </FormField>
        </div>

        <div className="grid grid-cols-2 gap-4 mt-4">
          <FormField label="Vertragsart">
            <Select
              value={form.contractType}
              onValueChange={(v) => update('contractType', v as ContractType)}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {Object.entries(CONTRACT_LABELS).map(([k, v]) => (
                  <SelectItem key={k} value={k}>
                    {v}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FormField>
          <FormField label="Pensum (%)">
            <Input
              type="number"
              min={10}
              max={100}
              step={10}
              value={form.workloadPercent}
              onChange={(e) => update('workloadPercent', Number(e.target.value))}
            />
          </FormField>
        </div>
      </SectionCard>

      <SectionCard title="Zeitrahmen" icon={Calendar}>
        <div className="grid grid-cols-3 gap-4">
          <FormField label="Arbeitstage / Woche">
            <Input
              type="number"
              min={1}
              max={7}
              value={form.workDaysPerWeek}
              onChange={(e) => update('workDaysPerWeek', Number(e.target.value))}
            />
          </FormField>
          <FormField label="Jahresurlaub (Tage)">
            <Input
              type="number"
              min={0}
              max={60}
              value={form.annualLeaveDays}
              onChange={(e) => update('annualLeaveDays', Number(e.target.value))}
            />
          </FormField>
          <FormField label="Startdatum" required>
            <Input
              type="date"
              value={form.startDate}
              onChange={(e) => update('startDate', e.target.value)}
            />
          </FormField>
        </div>
      </SectionCard>

      <SectionCard title="Standort & Vorgesetzte/r" icon={MapPin}>
        <div className="grid grid-cols-2 gap-4">
          <FormField label="Standort">
            <Input
              placeholder="z.B. Zuerich"
              value={form.location}
              onChange={(e) => update('location', e.target.value)}
            />
          </FormField>
          <FormField label="Vorgesetzte/r (User-ID)">
            <Input
              placeholder="Optional"
              value={form.managerUserId}
              onChange={(e) => update('managerUserId', e.target.value)}
            />
          </FormField>
        </div>
      </SectionCard>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Step 3: Personal
// ---------------------------------------------------------------------------

function PersonalStep({ form, update }: { form: EmployeeFormData; update: UpdateFn }) {
  return (
    <div className="space-y-5">
      <SectionCard title="Adresse" icon={Home}>
        <div className="space-y-4">
          <FormField label="Strasse">
            <Input
              placeholder="Musterstrasse 1"
              value={form.addressStreet}
              onChange={(e) => update('addressStreet', e.target.value)}
            />
          </FormField>
          <div className="grid grid-cols-3 gap-3">
            <FormField label="PLZ">
              <Input
                placeholder="8000"
                value={form.addressPostalCode}
                onChange={(e) => update('addressPostalCode', e.target.value)}
              />
            </FormField>
            <div className="col-span-2">
              <FormField label="Ort">
                <Input
                  placeholder="Zuerich"
                  value={form.addressCity}
                  onChange={(e) => update('addressCity', e.target.value)}
                />
              </FormField>
            </div>
          </div>
          <FormField label="Land">
            <Select value={form.addressCountry} onValueChange={(v) => update('addressCountry', v)}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="CH">Schweiz</SelectItem>
                <SelectItem value="DE">Deutschland</SelectItem>
                <SelectItem value="AT">Oesterreich</SelectItem>
                <SelectItem value="LI">Liechtenstein</SelectItem>
              </SelectContent>
            </Select>
          </FormField>
        </div>
      </SectionCard>

      <SectionCard title="Notfallkontakt" icon={Phone}>
        <div className="grid grid-cols-2 gap-4">
          <FormField label="Name">
            <Input
              placeholder="Anna Muster"
              value={form.emergencyContactName}
              onChange={(e) => update('emergencyContactName', e.target.value)}
            />
          </FormField>
          <FormField label="Telefon">
            <Input
              placeholder="+41 79 ..."
              value={form.emergencyContactPhone}
              onChange={(e) => update('emergencyContactPhone', e.target.value)}
            />
          </FormField>
        </div>
      </SectionCard>

      <div className="rounded-xl border border-dashed border-border bg-secondary/20 p-4 flex items-start gap-3">
        <AlertTriangle className="h-4 w-4 text-muted-foreground mt-0.5 shrink-0" />
        <p className="text-xs text-muted-foreground">
          Persoenliche Daten sind optional und koennen auch spaeter vom Mitarbeiter
          selbst ergaenzt werden.
        </p>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Step 4: Documents
// ---------------------------------------------------------------------------

function DocumentsStep({ form, update }: { form: EmployeeFormData; update: UpdateFn }) {
  const [category, setCategory] = useState('Arbeitsvertrag')
  const [fileName, setFileName] = useState('')
  const [notes, setNotes] = useState('')

  const addDocument = () => {
    if (!fileName.trim()) {
      toast.error('Bitte waehlen Sie eine Datei')
      return
    }
    const entry: DocumentEntry = {
      id: crypto.randomUUID(),
      category,
      fileName: fileName.trim(),
      notes: notes.trim(),
    }
    update('documents', [...form.documents, entry])
    setFileName('')
    setNotes('')
    toast.success(`"${entry.fileName}" hinzugefuegt`)
  }

  const removeDocument = (id: string) => {
    update(
      'documents',
      form.documents.filter((d) => d.id !== id),
    )
  }

  return (
    <div className="space-y-5">
      {/* Uploaded list */}
      {form.documents.length > 0 && (
        <SectionCard title={`Hinzugefuegt (${form.documents.length})`} icon={FileText}>
          <div className="space-y-2">
            {form.documents.map((doc) => (
              <div
                key={doc.id}
                className="flex items-center gap-3 rounded-lg border border-border bg-background/60 px-3.5 py-2.5"
              >
                <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary">
                  <FileText className="h-4 w-4" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-foreground truncate">{doc.fileName}</p>
                  <p className="text-[11px] text-muted-foreground">
                    {doc.category}
                    {doc.notes ? ` — ${doc.notes}` : ''}
                  </p>
                </div>
                <button
                  onClick={() => removeDocument(doc.id)}
                  className="rounded-md p-1.5 text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
            ))}
          </div>
        </SectionCard>
      )}

      {/* Upload form */}
      <SectionCard title="Dokument hinzufuegen" icon={Upload}>
        <div className="grid grid-cols-2 gap-4">
          <FormField label="Kategorie">
            <Select value={category} onValueChange={setCategory}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {DOC_CATEGORIES.map((cat) => (
                  <SelectItem key={cat} value={cat}>
                    {cat}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FormField>
          <FormField label="Datei" required>
            <Input
              type="file"
              onChange={(e) => {
                const file = e.target.files?.[0]
                if (file) setFileName(file.name)
              }}
              className="text-xs"
            />
          </FormField>
        </div>
        <div className="mt-4">
          <FormField label="Notizen">
            <Input
              placeholder="Optional — z.B. Befristung, Gueltigkeitsdatum"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
            />
          </FormField>
        </div>
        <div className="mt-4">
          <Button variant="outline" size="sm" onClick={addDocument} disabled={!fileName.trim()}>
            <Upload className="h-3.5 w-3.5 mr-1.5" />
            Hinzufuegen
          </Button>
        </div>
      </SectionCard>

      <div className="rounded-xl border border-dashed border-border bg-secondary/20 p-4 flex items-start gap-3">
        <FileText className="h-4 w-4 text-muted-foreground mt-0.5 shrink-0" />
        <p className="text-xs text-muted-foreground">
          Dieser Schritt ist optional. Dokumente koennen auch spaeter im Mitarbeiterprofil
          hochgeladen werden.
        </p>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Step 5: Summary
// ---------------------------------------------------------------------------

function SummaryStep({ form, update }: { form: EmployeeFormData; update: UpdateFn }) {
  const roleLabels = form.roles
    .map((r) => ROLE_OPTIONS.find((o) => o.id === r)?.label ?? r)
    .join(', ')

  return (
    <div className="space-y-5">
      {/* Account */}
      <SummaryCard title="Account" icon={User}>
        <SummaryRow label="Name" value={`${form.firstName} ${form.lastName}`} />
        <SummaryRow label="E-Mail" value={form.email} />
        {form.phone && <SummaryRow label="Telefon" value={form.phone} />}
        <SummaryRow label="Rolle(n)" value={roleLabels} />
      </SummaryCard>

      {/* Employment */}
      <SummaryCard title="Arbeitsverhaeltnis" icon={Briefcase}>
        {form.department && <SummaryRow label="Abteilung" value={form.department} />}
        {form.positionTitle && <SummaryRow label="Position" value={form.positionTitle} />}
        <SummaryRow
          label="Vertrag"
          value={CONTRACT_LABELS[form.contractType] ?? form.contractType}
        />
        <SummaryRow label="Pensum" value={`${form.workloadPercent}%`} />
        <SummaryRow label="Arbeitstage" value={`${form.workDaysPerWeek} Tage/Woche`} />
        <SummaryRow label="Urlaub" value={`${form.annualLeaveDays} Tage/Jahr`} />
        <SummaryRow
          label="Startdatum"
          value={new Date(form.startDate).toLocaleDateString('de-DE')}
        />
        {form.location && <SummaryRow label="Standort" value={form.location} />}
      </SummaryCard>

      {/* Personal */}
      {(form.addressStreet || form.emergencyContactName) && (
        <SummaryCard title="Persoenlich" icon={Home}>
          {form.addressStreet && (
            <SummaryRow
              label="Adresse"
              value={`${form.addressStreet}, ${form.addressPostalCode} ${form.addressCity}, ${form.addressCountry}`}
            />
          )}
          {form.emergencyContactName && (
            <SummaryRow
              label="Notfallkontakt"
              value={`${form.emergencyContactName}${form.emergencyContactPhone ? ` (${form.emergencyContactPhone})` : ''}`}
            />
          )}
        </SummaryCard>
      )}

      {/* Documents */}
      {form.documents.length > 0 && (
        <SummaryCard title={`Dokumente (${form.documents.length})`} icon={FileText}>
          {form.documents.map((doc) => (
            <SummaryRow key={doc.id} label={doc.category} value={doc.fileName} />
          ))}
        </SummaryCard>
      )}

      {/* Options */}
      <div className="rounded-xl border border-border bg-card/40 p-5">
        <label className="flex items-center gap-3 cursor-pointer">
          <Switch
            checked={form.sendInviteEmail}
            onCheckedChange={(v) => update('sendInviteEmail', v)}
          />
          <div>
            <p className="text-sm font-medium text-foreground">Einladungs-Email senden</p>
            <p className="text-xs text-muted-foreground">
              Der Mitarbeiter erhaelt eine E-Mail mit Login-Link und temporaerem Passwort
            </p>
          </div>
        </label>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Summary helpers
// ---------------------------------------------------------------------------

function SummaryCard({
  title,
  icon: Icon,
  children,
}: {
  title: string
  icon: React.ComponentType<{ className?: string }>
  children: React.ReactNode
}) {
  return (
    <div className="rounded-xl border border-border bg-card/40 p-5">
      <div className="flex items-center gap-2 mb-3">
        <Icon className="h-4 w-4 text-primary" />
        <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
          {title}
        </h4>
      </div>
      <div className="space-y-2">{children}</div>
    </div>
  )
}

function SummaryRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between text-sm">
      <span className="text-muted-foreground">{label}</span>
      <span className="text-foreground font-medium">{value}</span>
    </div>
  )
}

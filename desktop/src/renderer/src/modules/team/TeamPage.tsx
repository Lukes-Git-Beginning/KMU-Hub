import { useState, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Search,
  Plus,
  Grid3X3,
  List,
  Mail,
  Phone,
  MessageSquare,
  Clock,
  Calendar,
  Users,
  Briefcase,
  Banknote,
  GraduationCap,
  ChevronLeft,
  ChevronRight,
  AlertTriangle,
  Shield,
  Wrench,
  Heart,
  Scale,
  Award,
  X,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  useTeamStore,
  type TeamMember,
  type HRRequest,
  type PayrollEntry,
  type Training,
  type TrainingParticipation,
} from '@/stores/team'
import { useMeetingsStore } from '@/stores/meetings'
import { useNavigationStore } from '@/stores/navigation'
import { ItemActions, ConfirmDialog, EmptyState } from '@/components/shared'
import { MemberDetailPanel } from './MemberDetailPanel'
import { InviteMemberDialog } from './InviteMemberDialog'
import { EditMemberDialog } from './EditMemberDialog'
import { HRApprovalDialog } from './HRApprovalDialog'
import { AbsenceCalendar } from './AbsenceCalendar'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Checkbox } from '@/components/ui/checkbox'

type TabKey = 'members' | 'requests' | 'absences' | 'lohn' | 'schulungen'

const formatCHF = (amount: number) =>
  new Intl.NumberFormat('de-CH', { style: 'currency', currency: 'CHF', minimumFractionDigits: 0, maximumFractionDigits: 0 }).format(amount)

const statusColors: Record<string, string> = {
  online: 'bg-success',
  away: 'bg-warning',
  offline: 'bg-text-disabled',
  dnd: 'bg-error',
}

const requestTypeLabels: Record<string, string> = {
  vacation: 'Urlaub',
  sick: 'Krankheit',
  overtime: 'Überstunden',
  doctor: 'Arzttermin',
  homeoffice: 'Homeoffice',
  education: 'Weiterbildung',
}

const requestTypeColors: Record<string, string> = {
  vacation: 'bg-info-light text-info',
  sick: 'bg-error-light text-error',
  overtime: 'bg-warning-light text-warning',
  doctor: 'bg-primary-light text-primary',
  homeoffice: 'bg-success-light text-success',
  education: 'bg-primary-light text-primary',
}

const requestStatusColors: Record<string, string> = {
  pending: 'bg-warning-light text-warning',
  approved: 'bg-success-light text-success',
  rejected: 'bg-error-light text-error',
}

const requestStatusLabels: Record<string, string> = {
  pending: 'Ausstehend',
  approved: 'Genehmigt',
  rejected: 'Abgelehnt',
}

// Payroll helpers
const payrollStatusColors: Record<string, string> = {
  draft: 'bg-secondary text-muted-foreground',
  approved: 'bg-warning-light text-warning',
  paid: 'bg-success-light text-success',
}

const payrollStatusLabels: Record<string, string> = {
  draft: 'Entwurf',
  approved: 'Freigegeben',
  paid: 'Bezahlt',
}

const employmentTypeLabels: Record<string, string> = {
  fulltime: 'Festangestellt',
  parttime: 'Teilzeit',
  hourly: 'Stundenlohn',
}

const employmentTypeColors: Record<string, string> = {
  fulltime: 'bg-primary-light text-primary',
  parttime: 'bg-info-light text-info',
  hourly: 'bg-warning-light text-warning',
}

// Training helpers
const trainingTypeLabels: Record<string, string> = {
  safety: 'Sicherheit',
  technical: 'Technisch',
  soft_skills: 'Soft Skills',
  compliance: 'Compliance',
  certification: 'Zertifizierung',
}

const trainingTypeColors: Record<string, string> = {
  safety: 'bg-error-light text-error',
  technical: 'bg-info-light text-info',
  soft_skills: 'bg-primary-light text-primary',
  compliance: 'bg-warning-light text-warning',
  certification: 'bg-success-light text-success',
}

const trainingTypeIcons: Record<string, typeof Shield> = {
  safety: Shield,
  technical: Wrench,
  soft_skills: Heart,
  compliance: Scale,
  certification: Award,
}

const participationStatusColors: Record<string, string> = {
  completed: 'bg-success-light text-success',
  pending: 'bg-warning-light text-warning',
  expired: 'bg-error-light text-error',
  scheduled: 'bg-info-light text-info',
}

const participationStatusLabels: Record<string, string> = {
  completed: 'Abgeschlossen',
  pending: 'Ausstehend',
  expired: 'Abgelaufen',
  scheduled: 'Geplant',
}

export default function TeamPage() {
  const navigate = useNavigate()
  const {
    members, requests, departments, approveRequest, rejectRequest, deactivateMember,
    payroll, trainings, trainingParticipations, startPayrollRun, addTraining, recordParticipation,
  } = useTeamStore()
  const { startCall } = useMeetingsStore()
  const { setIntent } = useNavigationStore()

  const [tab, setTab] = useState<TabKey>('members')
  const [search, setSearch] = useState('')
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid')
  const [selectedMember, setSelectedMember] = useState<TeamMember | null>(null)
  const [showInvite, setShowInvite] = useState(false)
  const [editMember, setEditMember] = useState<TeamMember | null>(null)
  const [showEditDialog, setShowEditDialog] = useState(false)
  const [approvalRequest, setApprovalRequest] = useState<HRRequest | null>(null)
  const [confirmDeactivate, setConfirmDeactivate] = useState<TeamMember | null>(null)

  // Lohn tab state
  const [payrollMonth, setPayrollMonth] = useState('2026-01')
  const [selectedPayroll, setSelectedPayroll] = useState<PayrollEntry | null>(null)

  // Schulungen tab state
  const [showAddTraining, setShowAddTraining] = useState(false)
  const [showRecordParticipation, setShowRecordParticipation] = useState(false)

  const activeMembers = members.filter((m) => m.isActive)
  const filteredMembers = activeMembers.filter((m) => {
    if (!search) return true
    const q = search.toLowerCase()
    const fullName = `${m.firstName} ${m.lastName}`.toLowerCase()
    return fullName.includes(q) || m.role.toLowerCase().includes(q) || m.department.toLowerCase().includes(q)
  })

  const pendingCount = requests.filter((r) => r.status === 'pending').length

  // Department counts from actual data
  const deptCounts = departments.map((dept) => ({
    ...dept,
    count: activeMembers.filter((m) => m.department === dept.name).length,
  })).filter((d) => d.count > 0)

  // Payroll computed values
  const monthPayroll = useMemo(() => payroll.filter((p) => p.month === payrollMonth), [payroll, payrollMonth])
  const totalGross = useMemo(() => monthPayroll.reduce((s, p) => s + p.grossSalary, 0), [monthPayroll])
  const avgNet = useMemo(() => monthPayroll.length ? Math.round(monthPayroll.reduce((s, p) => s + p.netSalary, 0) / monthPayroll.length) : 0, [monthPayroll])
  const draftCount = useMemo(() => monthPayroll.filter((p) => p.status === 'draft').length, [monthPayroll])

  // Training computed values
  const mandatoryCount = useMemo(() => trainings.filter((t) => t.mandatory).length, [trainings])
  const expiredCount = useMemo(() => trainingParticipations.filter((p) => p.status === 'expired').length, [trainingParticipations])

  // Month navigation helpers
  const navigateMonth = (dir: -1 | 1) => {
    const [y, m] = payrollMonth.split('-').map(Number)
    const d = new Date(y, m - 1 + dir, 1)
    setPayrollMonth(`${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`)
    setSelectedPayroll(null)
  }

  const monthLabel = useMemo(() => {
    const [y, m] = payrollMonth.split('-').map(Number)
    return new Date(y, m - 1).toLocaleDateString('de-CH', { month: 'long', year: 'numeric' })
  }, [payrollMonth])

  // Cross-module actions
  const handleEmail = (member: TeamMember) => {
    setIntent({ type: 'compose-email', data: { to: member.email, name: `${member.firstName} ${member.lastName}` } })
    navigate('/mails')
  }

  const handleCall = (member: TeamMember) => {
    startCall(`${member.firstName} ${member.lastName}`, member.initials)
  }

  const handleMessage = (member: TeamMember) => {
    setIntent({ type: 'send-message', data: { name: `${member.firstName} ${member.lastName}` } })
    navigate('/chat')
  }

  const handleDeactivate = (member: TeamMember) => {
    deactivateMember(member.id)
    setConfirmDeactivate(null)
    toast.success(`${member.firstName} ${member.lastName} deaktiviert`)
  }

  const handleApprove = (id: string, comment?: string) => {
    approveRequest(id, comment)
    toast.success('Antrag genehmigt')
  }

  const handleReject = (id: string, comment?: string) => {
    rejectRequest(id, comment)
    toast.success('Antrag abgelehnt')
  }

  const getMemberActions = (member: TeamMember) => [
    { label: 'Profil ansehen', onClick: () => setSelectedMember(member) },
    { label: 'E-Mail senden', icon: Mail, onClick: () => handleEmail(member) },
    { label: 'Anrufen', icon: Phone, onClick: () => handleCall(member) },
    { label: 'Nachricht senden', icon: MessageSquare, onClick: () => handleMessage(member) },
    { separator: true as const, label: '', onClick: () => {} },
    { label: 'Bearbeiten', onClick: () => { setEditMember(member); setShowEditDialog(true) } },
    { label: 'Deaktivieren', variant: 'destructive' as const, onClick: () => setConfirmDeactivate(member) },
  ]

  return (
    <div className="flex-1 overflow-y-auto p-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between mb-6 gap-4">
        <div>
          <h1 className="text-foreground">Team</h1>
          <p className="text-sm text-muted-foreground">
            {activeMembers.length} Mitglieder · {pendingCount} offene Anfragen
          </p>
        </div>
        <button
          onClick={() => setShowInvite(true)}
          className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
        >
          <Plus className="h-4 w-4" />
          Mitglied einladen
        </button>
      </div>

      {/* Department cards */}
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3 mb-6">
        {deptCounts.map((dept) => (
          <div key={dept.id} className="flex items-center gap-3 rounded-lg border border-border bg-card p-3">
            <div
              className="flex h-10 w-10 items-center justify-center rounded-lg"
              style={{ backgroundColor: `color-mix(in srgb, ${dept.color} 15%, transparent)` }}
            >
              <Users className="h-5 w-5" style={{ color: dept.color }} />
            </div>
            <div>
              <p className="text-sm font-medium text-foreground">{dept.name}</p>
              <p className="text-xs text-muted-foreground">{dept.count} Mitglieder</p>
            </div>
          </div>
        ))}
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-4 border-b border-border mb-6">
        {([
          { key: 'members' as const, label: `Mitglieder (${activeMembers.length})`, icon: undefined },
          { key: 'requests' as const, label: `Anfragen (${pendingCount} offen)`, icon: undefined },
          { key: 'absences' as const, label: 'Abwesenheiten', icon: undefined },
          { key: 'lohn' as const, label: 'Lohn', icon: Banknote },
          { key: 'schulungen' as const, label: 'Schulungen', icon: GraduationCap },
        ]).map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`flex items-center gap-1.5 border-b-2 px-1 pb-2 text-sm transition-colors ${
              tab === t.key ? 'border-primary text-primary font-medium' : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            {t.icon && <t.icon className="h-3.5 w-3.5" />}
            {t.label}
          </button>
        ))}
      </div>

      {/* Members Tab */}
      {tab === 'members' && (
        <>
          <div className="flex items-center gap-3 mb-4">
            <div className="relative flex-1 max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <input
                type="text"
                placeholder="Mitglied suchen..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <div className="flex gap-1">
              <button
                onClick={() => setViewMode('grid')}
                className={`rounded-md p-1.5 transition-colors ${viewMode === 'grid' ? 'bg-secondary text-foreground' : 'text-muted-foreground hover:bg-secondary'}`}
              >
                <Grid3X3 className="h-4 w-4" />
              </button>
              <button
                onClick={() => setViewMode('list')}
                className={`rounded-md p-1.5 transition-colors ${viewMode === 'list' ? 'bg-secondary text-foreground' : 'text-muted-foreground hover:bg-secondary'}`}
              >
                <List className="h-4 w-4" />
              </button>
            </div>
          </div>

          {filteredMembers.length === 0 ? (
            <EmptyState
              icon={Users}
              title="Keine Mitglieder gefunden"
              description={search ? 'Passe deine Suche an' : 'Lade Mitglieder ein, um loszulegen'}
              actionLabel={search ? undefined : 'Mitglied einladen'}
              onAction={search ? undefined : () => setShowInvite(true)}
            />
          ) : viewMode === 'grid' ? (
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {filteredMembers.map((member) => (
                <MemberCard
                  key={member.id}
                  member={member}
                  actions={getMemberActions(member)}
                  onEmail={() => handleEmail(member)}
                  onMessage={() => handleMessage(member)}
                  onCall={() => handleCall(member)}
                  onClick={() => setSelectedMember(member)}
                />
              ))}
            </div>
          ) : (
            <div className="space-y-2">
              {filteredMembers.map((member) => (
                <MemberRow
                  key={member.id}
                  member={member}
                  actions={getMemberActions(member)}
                  onEmail={() => handleEmail(member)}
                  onMessage={() => handleMessage(member)}
                  onClick={() => setSelectedMember(member)}
                />
              ))}
            </div>
          )}
        </>
      )}

      {/* Requests Tab */}
      {tab === 'requests' && (
        <div className="space-y-3">
          {requests.length === 0 ? (
            <EmptyState
              icon={Calendar}
              title="Keine Anfragen"
              description="Es gibt aktuell keine HR-Anfragen"
            />
          ) : (
            requests.map((request) => (
              <RequestCard
                key={request.id}
                request={request}
                onApprove={() => setApprovalRequest(request)}
              />
            ))
          )}
        </div>
      )}

      {/* Absences Tab */}
      {tab === 'absences' && (
        <AbsenceCalendar members={members} requests={requests} />
      )}

      {/* Lohn Tab */}
      {tab === 'lohn' && (
        <div>
          {/* Stats Row */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6">
            <div className="rounded-lg border border-border bg-card p-4">
              <p className="text-xs text-muted-foreground mb-1">Gesamte Lohnkosten</p>
              <p className="text-lg font-semibold text-foreground">{formatCHF(totalGross)}</p>
            </div>
            <div className="rounded-lg border border-border bg-card p-4">
              <p className="text-xs text-muted-foreground mb-1">Durchschnitt Nettolohn</p>
              <p className="text-lg font-semibold text-foreground">{formatCHF(avgNet)}</p>
            </div>
            <div className="rounded-lg border border-border bg-card p-4">
              <p className="text-xs text-muted-foreground mb-1">Mitarbeiter</p>
              <p className="text-lg font-semibold text-foreground">{monthPayroll.length}</p>
            </div>
            <div className="rounded-lg border border-border bg-card p-4">
              <p className="text-xs text-muted-foreground mb-1">Naechster Lohnlauf</p>
              <p className="text-lg font-semibold text-foreground">
                {draftCount > 0 ? `${draftCount} Entwürfe` : 'Keine'}
              </p>
            </div>
          </div>

          {/* Month selector + action */}
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-2">
              <button
                onClick={() => navigateMonth(-1)}
                className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary transition-colors"
              >
                <ChevronLeft className="h-4 w-4" />
              </button>
              <span className="text-sm font-medium text-foreground min-w-[140px] text-center capitalize">
                {monthLabel}
              </span>
              <button
                onClick={() => navigateMonth(1)}
                className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary transition-colors"
              >
                <ChevronRight className="h-4 w-4" />
              </button>
            </div>
            {draftCount > 0 && (
              <button
                onClick={startPayrollRun}
                className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                <Banknote className="h-4 w-4" />
                Lohnlauf starten
              </button>
            )}
          </div>

          {/* Payroll Table */}
          {monthPayroll.length === 0 ? (
            <EmptyState
              icon={Banknote}
              title="Keine Lohndaten"
              description={`Für ${monthLabel} sind keine Lohnabrechnungen vorhanden`}
            />
          ) : (
            <div className="rounded-lg border border-border overflow-hidden">
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-border bg-secondary/50">
                      <th className="px-4 py-3 text-left font-medium text-muted-foreground">Mitarbeiter</th>
                      <th className="px-4 py-3 text-left font-medium text-muted-foreground">Abteilung</th>
                      <th className="px-4 py-3 text-left font-medium text-muted-foreground">Anstellung</th>
                      <th className="px-4 py-3 text-right font-medium text-muted-foreground">Brutto</th>
                      <th className="px-4 py-3 text-right font-medium text-muted-foreground">Abzuege</th>
                      <th className="px-4 py-3 text-right font-medium text-muted-foreground">Netto</th>
                      <th className="px-4 py-3 text-left font-medium text-muted-foreground">Status</th>
                    </tr>
                  </thead>
                  <tbody>
                    {monthPayroll.map((entry) => {
                      const totalDeductions = entry.deductions.ahv + entry.deductions.pension + entry.deductions.tax + entry.deductions.other
                      const isSelected = selectedPayroll?.id === entry.id
                      return (
                        <tr
                          key={entry.id}
                          onClick={() => setSelectedPayroll(isSelected ? null : entry)}
                          className={`border-b border-border-muted cursor-pointer transition-colors ${
                            isSelected ? 'bg-primary-light/50' : 'hover:bg-secondary/30'
                          }`}
                        >
                          <td className="px-4 py-3 font-medium text-foreground">{entry.memberName}</td>
                          <td className="px-4 py-3 text-muted-foreground">{entry.department}</td>
                          <td className="px-4 py-3">
                            <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${employmentTypeColors[entry.employmentType]}`}>
                              {employmentTypeLabels[entry.employmentType]}
                            </span>
                          </td>
                          <td className="px-4 py-3 text-right text-foreground tabular-nums">{formatCHF(entry.grossSalary)}</td>
                          <td className="px-4 py-3 text-right text-error tabular-nums">-{formatCHF(totalDeductions)}</td>
                          <td className="px-4 py-3 text-right font-medium text-foreground tabular-nums">{formatCHF(entry.netSalary)}</td>
                          <td className="px-4 py-3">
                            <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${payrollStatusColors[entry.status]}`}>
                              {payrollStatusLabels[entry.status]}
                            </span>
                          </td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* Salary breakdown detail */}
          {selectedPayroll && (
            <div className="mt-4 rounded-lg border border-border bg-card p-4">
              <div className="flex items-center justify-between mb-3">
                <h3 className="text-sm font-medium text-foreground">
                  Lohnabrechnung: {selectedPayroll.memberName}
                </h3>
                <button
                  onClick={() => setSelectedPayroll(null)}
                  className="rounded-md p-1 text-muted-foreground hover:bg-secondary transition-colors"
                >
                  <X className="h-4 w-4" />
                </button>
              </div>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <div className="space-y-1">
                  <p className="text-xs text-muted-foreground">AHV / IV / EO</p>
                  <p className="text-sm font-medium text-foreground">{formatCHF(selectedPayroll.deductions.ahv)}</p>
                  <p className="text-[10px] text-muted-foreground">
                    {((selectedPayroll.deductions.ahv / selectedPayroll.grossSalary) * 100).toFixed(1)}% vom Brutto
                  </p>
                </div>
                <div className="space-y-1">
                  <p className="text-xs text-muted-foreground">Pensionskasse (BVG)</p>
                  <p className="text-sm font-medium text-foreground">{formatCHF(selectedPayroll.deductions.pension)}</p>
                  <p className="text-[10px] text-muted-foreground">
                    {((selectedPayroll.deductions.pension / selectedPayroll.grossSalary) * 100).toFixed(1)}% vom Brutto
                  </p>
                </div>
                <div className="space-y-1">
                  <p className="text-xs text-muted-foreground">Quellensteuer</p>
                  <p className="text-sm font-medium text-foreground">{formatCHF(selectedPayroll.deductions.tax)}</p>
                  <p className="text-[10px] text-muted-foreground">
                    {((selectedPayroll.deductions.tax / selectedPayroll.grossSalary) * 100).toFixed(1)}% vom Brutto
                  </p>
                </div>
                <div className="space-y-1">
                  <p className="text-xs text-muted-foreground">Sonstige Abzuege</p>
                  <p className="text-sm font-medium text-foreground">{formatCHF(selectedPayroll.deductions.other)}</p>
                  <p className="text-[10px] text-muted-foreground">
                    {((selectedPayroll.deductions.other / selectedPayroll.grossSalary) * 100).toFixed(1)}% vom Brutto
                  </p>
                </div>
              </div>
              <div className="mt-4 flex items-center justify-between border-t border-border-muted pt-3">
                <span className="text-sm text-muted-foreground">Bruttolohn</span>
                <span className="text-sm font-medium text-foreground">{formatCHF(selectedPayroll.grossSalary)}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground">Total Abzuege</span>
                <span className="text-sm font-medium text-error">
                  -{formatCHF(selectedPayroll.deductions.ahv + selectedPayroll.deductions.pension + selectedPayroll.deductions.tax + selectedPayroll.deductions.other)}
                </span>
              </div>
              <div className="flex items-center justify-between border-t border-border pt-2 mt-2">
                <span className="text-sm font-medium text-foreground">Nettolohn</span>
                <span className="text-base font-semibold text-foreground">{formatCHF(selectedPayroll.netSalary)}</span>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Schulungen Tab */}
      {tab === 'schulungen' && (
        <div>
          {/* Stats Row */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6">
            <div className="rounded-lg border border-border bg-card p-4">
              <p className="text-xs text-muted-foreground mb-1">Schulungen gesamt</p>
              <p className="text-lg font-semibold text-foreground">{trainings.length}</p>
            </div>
            <div className="rounded-lg border border-border bg-card p-4">
              <p className="text-xs text-muted-foreground mb-1">Pflichtschulungen</p>
              <p className="text-lg font-semibold text-foreground">{mandatoryCount}</p>
            </div>
            <div className="rounded-lg border border-border bg-card p-4">
              <p className="text-xs text-muted-foreground mb-1">Abgelaufen</p>
              <div className="flex items-center gap-2">
                <p className="text-lg font-semibold text-foreground">{expiredCount}</p>
                {expiredCount > 0 && (
                  <span className="rounded-full bg-error-light px-2 py-0.5 text-[10px] font-medium text-error flex items-center gap-1">
                    <AlertTriangle className="h-3 w-3" />
                    Achtung
                  </span>
                )}
              </div>
            </div>
            <div className="rounded-lg border border-border bg-card p-4">
              <p className="text-xs text-muted-foreground mb-1">Teilnahmen</p>
              <p className="text-lg font-semibold text-foreground">{trainingParticipations.length}</p>
            </div>
          </div>

          {/* Action buttons */}
          <div className="flex items-center gap-2 mb-4">
            <button
              onClick={() => setShowAddTraining(true)}
              className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
            >
              <Plus className="h-4 w-4" />
              Schulung anlegen
            </button>
            <button
              onClick={() => setShowRecordParticipation(true)}
              className="flex items-center gap-2 rounded-lg border border-border bg-card px-4 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
            >
              <Plus className="h-4 w-4" />
              Teilnahme erfassen
            </button>
          </div>

          {/* Two-column layout */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Left: Schulungs-Katalog */}
            <div>
              <h3 className="text-sm font-medium text-foreground mb-3">Schulungs-Katalog</h3>
              <div className="space-y-3">
                {trainings.map((training) => {
                  const TypeIcon = trainingTypeIcons[training.type] || GraduationCap
                  return (
                    <div key={training.id} className="rounded-lg border border-border bg-card p-4">
                      <div className="flex items-start justify-between mb-2">
                        <div className="flex items-center gap-2">
                          <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${trainingTypeColors[training.type]}`}>
                            <TypeIcon className="h-4 w-4" />
                          </div>
                          <div>
                            <h4 className="text-sm font-medium text-foreground">{training.name}</h4>
                            <div className="flex items-center gap-2 mt-0.5">
                              <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${trainingTypeColors[training.type]}`}>
                                {trainingTypeLabels[training.type]}
                              </span>
                              <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
                                training.mandatory ? 'bg-error-light text-error' : 'bg-secondary text-muted-foreground'
                              }`}>
                                {training.mandatory ? 'Pflicht' : 'Optional'}
                              </span>
                            </div>
                          </div>
                        </div>
                      </div>
                      <div className="space-y-1 text-xs text-muted-foreground">
                        <p>Dauer: {training.duration}</p>
                        <p>Anbieter: {training.provider}</p>
                        {training.validityMonths > 0 && (
                          <p>Gueltig: {training.validityMonths} Monate</p>
                        )}
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>

            {/* Right: Teilnahme-Uebersicht */}
            <div>
              <h3 className="text-sm font-medium text-foreground mb-3">Teilnahme-Uebersicht</h3>
              <div className="rounded-lg border border-border overflow-hidden">
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-border bg-secondary/50">
                        <th className="px-3 py-2.5 text-left font-medium text-muted-foreground">Mitarbeiter</th>
                        <th className="px-3 py-2.5 text-left font-medium text-muted-foreground">Schulung</th>
                        <th className="px-3 py-2.5 text-left font-medium text-muted-foreground">Status</th>
                        <th className="px-3 py-2.5 text-left font-medium text-muted-foreground">Abgeschlossen</th>
                        <th className="px-3 py-2.5 text-left font-medium text-muted-foreground">Ablauf</th>
                      </tr>
                    </thead>
                    <tbody>
                      {trainingParticipations.map((p) => {
                        const training = trainings.find((t) => t.id === p.trainingId)
                        return (
                          <tr
                            key={p.id}
                            className={`border-b border-border-muted ${
                              p.status === 'expired' ? 'bg-error-light/20' : ''
                            }`}
                          >
                            <td className="px-3 py-2.5 text-foreground">{p.memberName}</td>
                            <td className="px-3 py-2.5 text-muted-foreground">{training?.name ?? '-'}</td>
                            <td className="px-3 py-2.5">
                              <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${participationStatusColors[p.status]}`}>
                                {p.status === 'expired' && <AlertTriangle className="h-3 w-3" />}
                                {participationStatusLabels[p.status]}
                              </span>
                            </td>
                            <td className="px-3 py-2.5 text-muted-foreground tabular-nums">
                              {p.completedAt ? new Date(p.completedAt).toLocaleDateString('de-CH') : '-'}
                            </td>
                            <td className="px-3 py-2.5 text-muted-foreground tabular-nums">
                              {p.expiresAt ? (
                                <span className={p.status === 'expired' ? 'text-error font-medium' : ''}>
                                  {new Date(p.expiresAt).toLocaleDateString('de-CH')}
                                </span>
                              ) : '-'}
                            </td>
                          </tr>
                        )
                      })}
                    </tbody>
                  </table>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Member Detail Panel */}
      {selectedMember && (
        <MemberDetailPanel
          member={selectedMember}
          onClose={() => setSelectedMember(null)}
          onEmail={() => { handleEmail(selectedMember); setSelectedMember(null) }}
          onCall={() => { handleCall(selectedMember); setSelectedMember(null) }}
          onMessage={() => { handleMessage(selectedMember); setSelectedMember(null) }}
          onEdit={() => { setEditMember(selectedMember); setShowEditDialog(true); setSelectedMember(null) }}
        />
      )}

      {/* Invite Dialog */}
      <InviteMemberDialog open={showInvite} onOpenChange={setShowInvite} />

      {/* Edit Dialog */}
      <EditMemberDialog open={showEditDialog} onOpenChange={setShowEditDialog} member={editMember} />

      {/* HR Approval Dialog */}
      <HRApprovalDialog
        open={!!approvalRequest}
        onOpenChange={() => setApprovalRequest(null)}
        request={approvalRequest}
        onApprove={handleApprove}
        onReject={handleReject}
      />

      {/* Confirm Deactivate */}
      <ConfirmDialog
        open={!!confirmDeactivate}
        onOpenChange={() => setConfirmDeactivate(null)}
        title="Mitglied deaktivieren?"
        description={`${confirmDeactivate?.firstName} ${confirmDeactivate?.lastName} wird deaktiviert. Das Konto kann später reaktiviert werden.`}
        confirmLabel="Deaktivieren"
        variant="destructive"
        onConfirm={() => confirmDeactivate && handleDeactivate(confirmDeactivate)}
      />

      {/* Add Training Dialog */}
      <AddTrainingDialog
        open={showAddTraining}
        onOpenChange={setShowAddTraining}
        onAdd={addTraining}
      />

      {/* Record Participation Dialog */}
      <RecordParticipationDialog
        open={showRecordParticipation}
        onOpenChange={setShowRecordParticipation}
        trainings={trainings}
        members={activeMembers}
        onRecord={recordParticipation}
      />
    </div>
  )
}

// ============================================================
// Sub-Components
// ============================================================

interface MemberCardProps {
  member: TeamMember
  actions: { label: string; icon?: any; onClick: () => void; variant?: string; separator?: boolean }[]
  onEmail: () => void
  onMessage: () => void
  onCall: () => void
  onClick: () => void
}

function MemberCard({ member, actions, onEmail, onMessage, onCall, onClick }: MemberCardProps) {
  const fullName = `${member.firstName} ${member.lastName}`

  return (
    <div className="rounded-lg border border-border bg-card p-4 transition-shadow hover:shadow-[var(--shadow-card-hover)]">
      <div className="flex items-start justify-between mb-3">
        <button onClick={onClick} className="flex items-center gap-3 text-left">
          <div className="relative">
            <div className="flex h-11 w-11 items-center justify-center rounded-full bg-primary-light text-sm font-medium text-primary">
              {member.initials}
            </div>
            <span
              className={`absolute -bottom-0.5 -right-0.5 h-3.5 w-3.5 rounded-full border-2 border-card ${statusColors[member.status]}`}
            />
          </div>
          <div>
            <h4 className="text-sm font-medium text-foreground">{fullName}</h4>
            <p className="text-xs text-muted-foreground">{member.role}</p>
          </div>
        </button>
        <ItemActions items={actions} />
      </div>

      <div className="space-y-1.5 text-xs text-muted-foreground mb-3">
        <div className="flex items-center gap-2">
          <Briefcase className="h-3 w-3" />
          <span>{member.department}</span>
        </div>
        {member.currentTask && (
          <div className="flex items-center gap-2">
            <Clock className="h-3 w-3" />
            <span className="truncate">{member.currentTask}</span>
          </div>
        )}
      </div>

      {member.projects.length > 0 && (
        <div className="flex flex-wrap gap-1 mb-3">
          {member.projects.map((p) => (
            <span key={p} className="rounded-full bg-secondary px-2 py-0.5 text-[10px] text-muted-foreground">
              {p}
            </span>
          ))}
        </div>
      )}

      <div className="flex gap-1.5 border-t border-border-muted pt-3">
        <button
          onClick={onEmail}
          className="flex-1 flex items-center justify-center gap-1 rounded-md border border-border py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
        >
          <Mail className="h-3 w-3" />
          E-Mail
        </button>
        <button
          onClick={onMessage}
          className="flex-1 flex items-center justify-center gap-1 rounded-md border border-border py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
        >
          <MessageSquare className="h-3 w-3" />
          Chat
        </button>
        <button
          onClick={onCall}
          className="flex-1 flex items-center justify-center gap-1 rounded-md border border-border py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
        >
          <Phone className="h-3 w-3" />
          Anrufen
        </button>
      </div>
    </div>
  )
}

interface MemberRowProps {
  member: TeamMember
  actions: { label: string; icon?: any; onClick: () => void; variant?: string; separator?: boolean }[]
  onEmail: () => void
  onMessage: () => void
  onClick: () => void
}

function MemberRow({ member, actions, onEmail, onMessage, onClick }: MemberRowProps) {
  const fullName = `${member.firstName} ${member.lastName}`

  return (
    <div className="flex items-center gap-4 rounded-lg border border-border bg-card px-4 py-3 hover:shadow-[var(--shadow-card)] transition-shadow">
      <button onClick={onClick} className="flex items-center gap-3 flex-1 min-w-0 text-left">
        <div className="relative">
          <div className="flex h-9 w-9 items-center justify-center rounded-full bg-primary-light text-xs font-medium text-primary">
            {member.initials}
          </div>
          <span
            className={`absolute -bottom-0.5 -right-0.5 h-3 w-3 rounded-full border-2 border-card ${statusColors[member.status]}`}
          />
        </div>
        <div className="min-w-0">
          <span className="text-sm font-medium text-foreground">{fullName}</span>
          <p className="text-xs text-muted-foreground truncate">{member.role} &middot; {member.department}</p>
        </div>
      </button>
      {member.currentTask && (
        <div className="hidden lg:flex items-center gap-1.5 text-xs text-muted-foreground max-w-[200px]">
          <Clock className="h-3 w-3 shrink-0" />
          <span className="truncate">{member.currentTask}</span>
        </div>
      )}
      <div className="flex gap-1">
        <button onClick={onEmail} className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary" title="E-Mail">
          <Mail className="h-4 w-4" />
        </button>
        <button onClick={onMessage} className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary" title="Nachricht">
          <MessageSquare className="h-4 w-4" />
        </button>
        <ItemActions items={actions} />
      </div>
    </div>
  )
}

function RequestCard({ request, onApprove }: { request: HRRequest; onApprove: () => void }) {
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary-light text-sm font-medium text-primary">
            {request.memberInitials}
          </div>
          <div>
            <h4 className="text-sm font-medium text-foreground">{request.memberName}</h4>
            <div className="flex items-center gap-2 mt-0.5">
              <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${requestTypeColors[request.type] ?? 'bg-secondary text-muted-foreground'}`}>
                {requestTypeLabels[request.type] ?? request.type}
              </span>
              <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${requestStatusColors[request.status]}`}>
                {requestStatusLabels[request.status]}
              </span>
            </div>
          </div>
        </div>
      </div>

      <div className="space-y-1.5 text-sm text-muted-foreground mb-3">
        <div className="flex items-center gap-2">
          <Calendar className="h-3.5 w-3.5" />
          <span>
            {new Date(request.startDate).toLocaleDateString('de-CH')}
            {request.startDate !== request.endDate && ` – ${new Date(request.endDate).toLocaleDateString('de-CH')}`}
            {' '}({request.days} {request.days === 1 ? 'Tag' : 'Tage'})
          </span>
        </div>
        <p className="text-sm text-text-body">{request.reason}</p>
        {request.comment && (
          <p className="text-xs text-muted-foreground italic">Kommentar: {request.comment}</p>
        )}
      </div>

      {request.status === 'pending' && (
        <button
          onClick={onApprove}
          className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
        >
          Bearbeiten
        </button>
      )}
    </div>
  )
}

// ============================================================
// Add Training Dialog
// ============================================================

function AddTrainingDialog({ open, onOpenChange, onAdd }: {
  open: boolean
  onOpenChange: (v: boolean) => void
  onAdd: (t: Omit<Training, 'id'>) => void
}) {
  const [name, setName] = useState('')
  const [type, setType] = useState<Training['type']>('safety')
  const [duration, setDuration] = useState('')
  const [mandatory, setMandatory] = useState(false)
  const [provider, setProvider] = useState('')
  const [validityMonths, setValidityMonths] = useState(0)

  const reset = () => {
    setName(''); setType('safety'); setDuration(''); setMandatory(false); setProvider(''); setValidityMonths(0)
  }

  const handleAdd = () => {
    if (!name.trim() || !duration.trim() || !provider.trim()) return
    onAdd({ name: name.trim(), type, duration: duration.trim(), mandatory, provider: provider.trim(), validityMonths })
    toast.success(`Schulung "${name.trim()}" angelegt`)
    reset()
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) reset(); onOpenChange(v) }}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Schulung anlegen</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div className="space-y-1.5">
            <Label>Name *</Label>
            <Input autoFocus placeholder="z.B. Erste Hilfe Kurs" value={name} onChange={(e) => setName(e.target.value)} />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Typ</Label>
              <Select value={type} onValueChange={(v) => setType(v as Training['type'])}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="safety">Sicherheit</SelectItem>
                  <SelectItem value="technical">Technisch</SelectItem>
                  <SelectItem value="soft_skills">Soft Skills</SelectItem>
                  <SelectItem value="compliance">Compliance</SelectItem>
                  <SelectItem value="certification">Zertifizierung</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>Dauer *</Label>
              <Input placeholder="z.B. 2 Tage" value={duration} onChange={(e) => setDuration(e.target.value)} />
            </div>
          </div>

          <div className="space-y-1.5">
            <Label>Anbieter *</Label>
            <Input placeholder="z.B. Schweizerisches Rotes Kreuz" value={provider} onChange={(e) => setProvider(e.target.value)} />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Gueltigkeit (Monate)</Label>
              <Input
                type="number"
                min={0}
                placeholder="0 = unbegrenzt"
                value={validityMonths}
                onChange={(e) => setValidityMonths(Number(e.target.value))}
              />
            </div>
            <div className="flex items-center gap-2 pt-6">
              <Checkbox
                id="mandatory"
                checked={mandatory}
                onCheckedChange={(v) => setMandatory(v === true)}
              />
              <Label htmlFor="mandatory" className="text-sm font-normal cursor-pointer">Pflichtschulung</Label>
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => { reset(); onOpenChange(false) }}>
            Abbrechen
          </Button>
          <Button onClick={handleAdd} disabled={!name.trim() || !duration.trim() || !provider.trim()}>
            Anlegen
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ============================================================
// Record Participation Dialog
// ============================================================

function RecordParticipationDialog({ open, onOpenChange, trainings, members, onRecord }: {
  open: boolean
  onOpenChange: (v: boolean) => void
  trainings: Training[]
  members: TeamMember[]
  onRecord: (p: Omit<TrainingParticipation, 'id'>) => void
}) {
  const [memberId, setMemberId] = useState('')
  const [trainingId, setTrainingId] = useState('')
  const [status, setStatus] = useState<TrainingParticipation['status']>('completed')
  const [completedAt, setCompletedAt] = useState('')

  const reset = () => {
    setMemberId(''); setTrainingId(''); setStatus('completed'); setCompletedAt('')
  }

  const handleRecord = () => {
    if (!memberId || !trainingId) return
    const member = members.find((m) => m.id === memberId)
    if (!member) return

    const training = trainings.find((t) => t.id === trainingId)
    const memberName = `${member.firstName} ${member.lastName}`

    const participation: Omit<TrainingParticipation, 'id'> = {
      trainingId,
      memberId,
      memberName,
      status,
      ...(completedAt ? { completedAt } : {}),
    }

    // Calculate expiry if completed and training has validity
    if (status === 'completed' && completedAt && training && training.validityMonths > 0) {
      const date = new Date(completedAt)
      date.setMonth(date.getMonth() + training.validityMonths)
      participation.expiresAt = date.toISOString().split('T')[0]
    }

    onRecord(participation)
    toast.success(`Teilnahme für ${memberName} erfasst`)
    reset()
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) reset(); onOpenChange(v) }}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Teilnahme erfassen</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div className="space-y-1.5">
            <Label>Mitarbeiter *</Label>
            <Select value={memberId} onValueChange={setMemberId}>
              <SelectTrigger>
                <SelectValue placeholder="Mitarbeiter waehlen..." />
              </SelectTrigger>
              <SelectContent>
                {members.map((m) => (
                  <SelectItem key={m.id} value={m.id}>{m.firstName} {m.lastName}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-1.5">
            <Label>Schulung *</Label>
            <Select value={trainingId} onValueChange={setTrainingId}>
              <SelectTrigger>
                <SelectValue placeholder="Schulung waehlen..." />
              </SelectTrigger>
              <SelectContent>
                {trainings.map((t) => (
                  <SelectItem key={t.id} value={t.id}>{t.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Status</Label>
              <Select value={status} onValueChange={(v) => setStatus(v as TrainingParticipation['status'])}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="completed">Abgeschlossen</SelectItem>
                  <SelectItem value="pending">Ausstehend</SelectItem>
                  <SelectItem value="scheduled">Geplant</SelectItem>
                  <SelectItem value="expired">Abgelaufen</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>Datum</Label>
              <Input type="date" value={completedAt} onChange={(e) => setCompletedAt(e.target.value)} />
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => { reset(); onOpenChange(false) }}>
            Abbrechen
          </Button>
          <Button onClick={handleRecord} disabled={!memberId || !trainingId}>
            Erfassen
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

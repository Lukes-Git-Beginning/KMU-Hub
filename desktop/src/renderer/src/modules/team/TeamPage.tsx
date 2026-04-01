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
  GraduationCap,
  AlertTriangle,
  Shield,
  Wrench,
  Heart,
  Scale,
  Award,
  Loader2,
  Link2,
  FolderOpen,
  ListChecks,
  Network,
  UserCircle,
  Settings,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  useTeamStore,
  type Training,
  type TrainingParticipation,
} from '@/stores/team'
import { useTimeTrackingStore, type TeamActivityEntry } from '@/stores/timetracking'
import { useMeetingsStore } from '@/stores/meetings'
import { useNavigationStore } from '@/stores/navigation'
import { useEmployees, useLeaveRequests, useUpdateEmployee } from '@/api/hooks/hr-hooks'
import type { EmployeeProfile, LeaveRequest } from '@/api/hr-types'
import { ItemActions, ConfirmDialog, EmptyState, PageHeader, type ActionItem } from '@/components/shared'
import { MemberDetailPanel } from './MemberDetailPanel'
import { InviteMemberDialog } from './InviteMemberDialog'
import { CreateEmployeeWizard } from './CreateEmployeeWizard'
import { EditMemberDialog } from './EditMemberDialog'
import { HRApprovalDialog } from './HRApprovalDialog'
import { AbsenceCalendar } from './AbsenceCalendar'
import { TimeCorrectionPanel } from './TimeCorrectionPanel'
import { HRIntegrationPanel } from './HRIntegrationPanel'
import { PersonnelDocuments } from './PersonnelDocuments'
import { OnboardingChecklist } from './OnboardingChecklist'
import { OrgChart } from './OrgChart'
import { SelfServiceView } from './SelfServiceView'
import { TeamSettingsTab } from '@/modules/settings/tabs/TeamSettingsTab'
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

type TabKey = 'members' | 'requests' | 'absences' | 'korrekturen' | 'personalakte' | 'onboarding' | 'orgchart' | 'integrationen' | 'schulungen' | 'selfservice' | 'einstellungen'

const contractTypeLabels: Record<string, string> = {
  full_time: 'Vollzeit',
  part_time: 'Teilzeit',
  praktikum: 'Praktikum',
  freelance: 'Freelancer',
}

const leaveStatusColors: Record<string, string> = {
  pending: 'bg-warning-light text-warning',
  approved: 'bg-success-light text-success',
  rejected: 'bg-error-light text-error',
  cancelled: 'bg-secondary text-muted-foreground',
}

const leaveStatusLabels: Record<string, string> = {
  pending: 'Ausstehend',
  approved: 'Genehmigt',
  rejected: 'Abgelehnt',
  cancelled: 'Storniert',
}

// Training helpers (still Zustand mock)
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
  // Keep Zustand only for training/payroll mocks (no backend API yet)
  const {
    trainings, trainingParticipations, addTraining, recordParticipation,
  } = useTeamStore()
  const updateEmployeeMutation = useUpdateEmployee()
  const { startCall } = useMeetingsStore()
  const { setIntent } = useNavigationStore()

  // Team activity from time tracking store
  const teamActivity = useTimeTrackingStore((s) => s.teamActivity)
  const activityByName = useMemo(() => {
    const map = new Map<string, TeamActivityEntry>()
    for (const a of teamActivity) map.set(a.userName, a)
    return map
  }, [teamActivity])

  // Use TanStack Query for HR data
  const { data: employeesData, isLoading: employeesLoading } = useEmployees()
  const { data: pendingRequestsData } = useLeaveRequests({ status: 'pending' })

  const [tab, setTab] = useState<TabKey>('members')
  const [search, setSearch] = useState('')
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid')
  const [selectedMemberId, setSelectedMemberId] = useState<string | null>(null)
  const [selectedMemberName, setSelectedMemberName] = useState<string>('')
  const [selectedMemberInitials, setSelectedMemberInitials] = useState<string>('')
  const [showInvite, setShowInvite] = useState(false)
  const [showCreateWizard, setShowCreateWizard] = useState(false)
  const [editMember, _setEditMember] = useState<EmployeeProfile | null>(null)
  const [showEditDialog, setShowEditDialog] = useState(false)
  const [approvalRequest, setApprovalRequest] = useState<LeaveRequest | null>(null)
  const [confirmDeactivate, setConfirmDeactivate] = useState<EmployeeProfile | null>(null)

  // Schulungen tab state
  const [showAddTraining, setShowAddTraining] = useState(false)
  const [showRecordParticipation, setShowRecordParticipation] = useState(false)

  // Employees from API
  const apiEmployees = useMemo(() => employeesData?.employees ?? [], [employeesData?.employees])
  const pendingRequests = pendingRequestsData?.requests ?? []
  const pendingCount = pendingRequests.length

  // Filter employees
  const filteredEmployees = apiEmployees.filter((emp) => {
    if (!search) return true
    const q = search.toLowerCase()
    const name = (emp.userName ?? '').toLowerCase()
    return name.includes(q) || (emp.department ?? '').toLowerCase().includes(q) || (emp.positionTitle ?? '').toLowerCase().includes(q)
  })

  // Department counts from API data
  const deptCounts = useMemo(() => {
    const counts = new Map<string, number>()
    for (const emp of apiEmployees) {
      const dept = emp.department ?? 'Sonstige'
      counts.set(dept, (counts.get(dept) ?? 0) + 1)
    }
    return [...counts.entries()]
      .map(([name, count]) => ({ name, count }))
      .filter((d) => d.count > 0)
      .sort((a, b) => a.name.localeCompare(b.name))
  }, [apiEmployees])

  // Training computed values (still from Zustand mock)
   
  const mandatoryCount = useMemo(() => trainings.filter((t) => t.mandatory).length, [trainings])
  const expiredCount = useMemo(() => trainingParticipations.filter((p) => p.status === 'expired').length, [trainingParticipations])

  // Cross-module actions
  const handleEmail = (name: string, email?: string) => {
    setIntent({ type: 'compose-email', data: { to: email ?? '', name } })
    navigate('/mails')
  }

  const handleCall = (name: string, initials: string) => {
    startCall(name, initials)
  }

  const handleMessage = (name: string) => {
    setIntent({ type: 'send-message', data: { name } })
    navigate('/chat')
  }

  const handleDeactivate = (emp: EmployeeProfile) => {
    // TODO: Backend needs a dedicated deactivation endpoint; for now update via employee API
    updateEmployeeMutation.mutate({ id: emp.id, data: {} })
    setConfirmDeactivate(null)
    toast.success(`${emp.userName ?? 'Mitarbeiter'} deaktiviert`)
  }

  const getInitials = (name: string) => {
    return name
      .split(' ')
      .map((n) => n[0])
      .join('')
      .slice(0, 2)
      .toUpperCase()
  }

  const getEmployeeActions = (emp: EmployeeProfile): ActionItem[] => {
    const name = emp.userName ?? 'Unbekannt'
    const initials = getInitials(name)
    return [
      { label: 'Profil ansehen', onClick: () => { setSelectedMemberId(emp.id); setSelectedMemberName(name); setSelectedMemberInitials(initials) } },
      { label: 'E-Mail senden', icon: Mail, onClick: () => handleEmail(name, emp.userEmail) },
      { label: 'Anrufen', icon: Phone, onClick: () => handleCall(name, initials) },
      { label: 'Nachricht senden', icon: MessageSquare, onClick: () => handleMessage(name) },
    ]
  }

  return (
    <div className="flex-1 overflow-y-auto p-6">
      {/* Header */}
      <PageHeader
        title="Team"
        description={`${apiEmployees.length} Mitglieder · ${pendingCount} offene Anfragen`}
        icon={Users}
        moduleId="team"
        actions={
          <button
            onClick={() => setShowCreateWizard(true)}
            className="flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            <Plus className="h-4 w-4" />
            Mitarbeiter erstellen
          </button>
        }
        className="mb-6"
      />

      {/* Department cards */}
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3 mb-6 animate-fade-up" style={{ animationDelay: '50ms' }}>
        {deptCounts.map((dept) => (
          <div key={dept.name} className="flex items-center gap-3 rounded-xl border border-border bg-card p-3 transition-all duration-200 hover:shadow-md hover:-translate-y-0.5">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-light">
              <Users className="h-5 w-5 text-primary" />
            </div>
            <div>
              <p className="text-sm font-medium text-foreground">{dept.name}</p>
              <p className="text-xs text-muted-foreground">{dept.count} Mitglieder</p>
            </div>
          </div>
        ))}
      </div>

      {/* Tabs — scrollable with fade mask */}
      <div className="relative mb-6">
        <div
          className="flex items-center gap-4 border-b border-border overflow-x-auto scrollbar-hide pr-8"
          style={{ maskImage: 'linear-gradient(to right, black calc(100% - 40px), transparent)', WebkitMaskImage: 'linear-gradient(to right, black calc(100% - 40px), transparent)' }}
        >
          {([
            { key: 'members' as const, label: `Mitglieder (${apiEmployees.length})`, icon: undefined },
            { key: 'requests' as const, label: `Anfragen (${pendingCount} offen)`, icon: undefined },
            { key: 'absences' as const, label: 'Abwesenheiten', icon: undefined },
            { key: 'korrekturen' as const, label: 'Korrekturen', icon: Clock },
            { key: 'personalakte' as const, label: 'Personalakte', icon: FolderOpen },
            { key: 'onboarding' as const, label: 'Onboarding', icon: ListChecks },
            { key: 'orgchart' as const, label: 'Organigramm', icon: Network },
            { key: 'integrationen' as const, label: 'Integrationen', icon: Link2 },
            { key: 'schulungen' as const, label: 'Schulungen', icon: GraduationCap },
            { key: 'selfservice' as const, label: 'Self-Service', icon: UserCircle },
            { key: 'einstellungen' as const, label: 'Einstellungen', icon: Settings },
          ]).map((t) => (
            <button
              key={t.key}
              onClick={() => setTab(t.key)}
              className={`flex items-center gap-1.5 border-b-2 px-1 pb-2 text-sm transition-colors whitespace-nowrap ${
                tab === t.key ? 'border-primary text-primary font-medium tab-accent-active' : 'border-transparent text-muted-foreground hover:text-foreground'
              }`}
            >
              {t.icon && <t.icon className="h-3.5 w-3.5" />}
              {t.label}
            </button>
          ))}
        </div>
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

          {employeesLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-6 w-6 animate-spin text-primary" />
            </div>
          ) : filteredEmployees.length === 0 ? (
            <EmptyState
              icon={Users}
              title="Keine Mitglieder gefunden"
              description={search ? 'Passe deine Suche an' : 'Lade Mitglieder ein, um loszulegen'}
              action={search ? undefined : { label: 'Mitarbeiter erstellen', onClick: () => setShowCreateWizard(true) }}
            />
          ) : viewMode === 'grid' ? (
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {filteredEmployees.map((emp) => {
                const name = emp.userName ?? 'Unbekannt'
                const initials = getInitials(name)
                return (
                  <EmployeeCard
                    key={emp.id}
                    employee={emp}
                    name={name}
                    initials={initials}
                    actions={getEmployeeActions(emp)}
                    activity={activityByName.get(name)}
                    onEmail={() => handleEmail(name, emp.userEmail)}
                    onMessage={() => handleMessage(name)}
                    onCall={() => handleCall(name, initials)}
                    onClick={() => { setSelectedMemberId(emp.id); setSelectedMemberName(name); setSelectedMemberInitials(initials) }}
                  />
                )
              })}
            </div>
          ) : (
            <div className="space-y-2">
              {filteredEmployees.map((emp) => {
                const name = emp.userName ?? 'Unbekannt'
                const initials = getInitials(name)
                return (
                  <EmployeeRow
                    key={emp.id}
                    employee={emp}
                    name={name}
                    initials={initials}
                    actions={getEmployeeActions(emp)}
                    activity={activityByName.get(name)}
                    onEmail={() => handleEmail(name, emp.userEmail)}
                    onMessage={() => handleMessage(name)}
                    onClick={() => { setSelectedMemberId(emp.id); setSelectedMemberName(name); setSelectedMemberInitials(initials) }}
                  />
                )
              })}
            </div>
          )}
        </>
      )}

      {/* Requests Tab -- uses TanStack Query */}
      {tab === 'requests' && (
        <div className="space-y-3">
          {pendingRequests.length === 0 ? (
            <EmptyState
              icon={Calendar}
              title="Keine Anfragen"
              description="Es gibt aktuell keine offenen HR-Anfragen"
            />
          ) : (
            pendingRequests.map((req) => (
              <LeaveRequestCard
                key={req.id}
                request={req}
                onApprove={() => setApprovalRequest(req)}
              />
            ))
          )}
        </div>
      )}

      {/* Absences Tab */}
      {tab === 'absences' && (
        <AbsenceCalendar />
      )}

      {/* Korrekturen Tab */}
      {tab === 'korrekturen' && (
        <TimeCorrectionPanel />
      )}

      {/* Personalakte Tab (7.3) */}
      {tab === 'personalakte' && <PersonnelDocuments />}

      {/* Onboarding Tab (7.4) */}
      {tab === 'onboarding' && <OnboardingChecklist />}

      {/* Organigramm Tab (7.5) */}
      {tab === 'orgchart' && <OrgChart />}

      {/* Integrationen Tab (7.1 + 7.2 — replaces old Lohn tab) */}
      {tab === 'integrationen' && <HRIntegrationPanel />}

      {/* Schulungen Tab (still Zustand mock) */}
      {tab === 'schulungen' && (
        <div>
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

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
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
                          <p>Gültig: {training.validityMonths} Monate</p>
                        )}
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>

            <div>
              <h3 className="text-sm font-medium text-foreground mb-3">Teilnahme-Übersicht</h3>
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
                              {p.completedAt ? new Date(p.completedAt).toLocaleDateString('de-DE') : '-'}
                            </td>
                            <td className="px-3 py-2.5 text-muted-foreground tabular-nums">
                              {p.expiresAt ? (
                                <span className={p.status === 'expired' ? 'text-error font-medium' : ''}>
                                  {new Date(p.expiresAt).toLocaleDateString('de-DE')}
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

      {/* Self-Service Tab (7.6) */}
      {tab === 'selfservice' && <SelfServiceView />}

      {/* Einstellungen Tab (HR settings moved from global Settings) */}
      {tab === 'einstellungen' && <TeamSettingsTab />}

      {/* Member Detail Panel */}
      {selectedMemberId && (
        <MemberDetailPanel
          memberId={selectedMemberId}
          memberName={selectedMemberName}
          memberInitials={selectedMemberInitials}
          onClose={() => setSelectedMemberId(null)}
          onEmail={() => { handleEmail(selectedMemberName); setSelectedMemberId(null) }}
          onCall={() => { handleCall(selectedMemberName, selectedMemberInitials); setSelectedMemberId(null) }}
          onMessage={() => { handleMessage(selectedMemberName); setSelectedMemberId(null) }}
          onEdit={() => { setSelectedMemberId(null) }}
        />
      )}

      {/* Create Employee Wizard */}
      {showCreateWizard && (
        <CreateEmployeeWizard onClose={() => setShowCreateWizard(false)} />
      )}

      {/* Invite Dialog (legacy, kept for quick invite) */}
      <InviteMemberDialog open={showInvite} onOpenChange={setShowInvite} />

      {/* Edit Dialog (kept for Zustand members if needed) */}
      <EditMemberDialog open={showEditDialog} onOpenChange={setShowEditDialog} member={editMember} />

      {/* HR Approval Dialog -- now uses TanStack Query */}
      <HRApprovalDialog
        open={!!approvalRequest}
        onOpenChange={() => setApprovalRequest(null)}
        request={approvalRequest}
      />

      {/* Confirm Deactivate */}
      <ConfirmDialog
        open={!!confirmDeactivate}
        onOpenChange={() => setConfirmDeactivate(null)}
        title="Mitglied deaktivieren?"
        description={`${confirmDeactivate?.userName ?? 'Mitarbeiter'} wird deaktiviert. Das Konto kann später reaktiviert werden.`}
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
        members={apiEmployees}
        onRecord={recordParticipation}
      />
    </div>
  )
}

// ============================================================
// Sub-Components
// ============================================================

interface EmployeeCardProps {
  employee: EmployeeProfile
  name: string
  initials: string
  actions: ActionItem[]
  activity?: TeamActivityEntry
  onEmail: () => void
  onMessage: () => void
  onCall: () => void
  onClick: () => void
}

function EmployeeCard({ employee, name, initials, actions, activity, onEmail, onMessage, onCall, onClick }: EmployeeCardProps) {
  const statusDot = activity?.status === 'tracking'
    ? 'bg-success'
    : activity?.status === 'absent'
      ? 'bg-warning'
      : 'bg-gray-400'

  return (
    <div className="rounded-lg border border-border bg-card p-4 transition-shadow hover:shadow-[var(--shadow-card-hover)]">
      <div className="flex items-start justify-between mb-3">
        <button onClick={onClick} className="flex items-center gap-3 text-left">
          <div className="relative">
            <div className="flex h-11 w-11 items-center justify-center rounded-full bg-primary-light text-sm font-medium text-primary">
              {initials}
            </div>
            {activity && (
              <span className={`absolute -bottom-0.5 -right-0.5 h-3.5 w-3.5 rounded-full border-2 border-card ${statusDot}`} />
            )}
          </div>
          <div>
            <h4 className="text-sm font-medium text-foreground">{name}</h4>
            <p className="text-xs text-muted-foreground">{employee.positionTitle ?? ''}</p>
          </div>
        </button>
        <ItemActions items={actions} />
      </div>

      {/* Current activity */}
      {activity && activity.status === 'tracking' && activity.currentDescription && (
        <div className="flex items-center gap-2 mb-3 rounded-md bg-secondary/60 px-2.5 py-1.5">
          {activity.currentCategoryColor && (
            <span className="h-2 w-2 rounded-full shrink-0" style={{ backgroundColor: activity.currentCategoryColor }} />
          )}
          <span className="text-[11px] text-foreground/80 truncate">{activity.currentDescription}</span>
          {activity.startedAt && (
            <span className="text-[10px] text-muted-foreground tabular-nums shrink-0 ml-auto">seit {activity.startedAt}</span>
          )}
        </div>
      )}
      {activity && activity.status === 'absent' && activity.currentDescription && (
        <div className="flex items-center gap-2 mb-3 rounded-md bg-warning/10 px-2.5 py-1.5">
          <span className="text-[11px] text-warning-foreground">{activity.currentDescription}</span>
        </div>
      )}

      <div className="space-y-1.5 text-xs text-muted-foreground mb-3">
        <div className="flex items-center gap-2">
          <Briefcase className="h-3 w-3" />
          <span>{employee.department ?? 'Keine Abteilung'}</span>
        </div>
        <div className="flex items-center gap-2">
          <Clock className="h-3 w-3" />
          <span>{contractTypeLabels[employee.contractType] ?? employee.contractType}</span>
        </div>
      </div>

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

interface EmployeeRowProps {
  employee: EmployeeProfile
  name: string
  initials: string
  actions: ActionItem[]
  activity?: TeamActivityEntry
  onEmail: () => void
  onMessage: () => void
  onClick: () => void
}

function EmployeeRow({ employee, name, initials, actions, activity, onEmail, onMessage, onClick }: EmployeeRowProps) {
  const statusDot = activity?.status === 'tracking'
    ? 'bg-success'
    : activity?.status === 'absent'
      ? 'bg-warning'
      : 'bg-gray-400'

  return (
    <div className="flex items-center gap-4 rounded-lg border border-border bg-card px-4 py-3 hover:shadow-[var(--shadow-card)] transition-shadow">
      <button onClick={onClick} className="flex items-center gap-3 flex-1 min-w-0 text-left">
        <div className="relative">
          <div className="flex h-9 w-9 items-center justify-center rounded-full bg-primary-light text-xs font-medium text-primary">
            {initials}
          </div>
          {activity && (
            <span className={`absolute -bottom-0.5 -right-0.5 h-3 w-3 rounded-full border-2 border-card ${statusDot}`} />
          )}
        </div>
        <div className="min-w-0 flex-1">
          <span className="text-sm font-medium text-foreground">{name}</span>
          <p className="text-xs text-muted-foreground truncate">
            {employee.positionTitle ?? ''} &middot; {employee.department ?? ''}
          </p>
        </div>
        {activity?.status === 'tracking' && activity.currentDescription && (
          <div className="hidden lg:flex items-center gap-1.5 shrink-0">
            {activity.currentCategoryColor && (
              <span className="h-2 w-2 rounded-full" style={{ backgroundColor: activity.currentCategoryColor }} />
            )}
            <span className="text-[11px] text-foreground/70 max-w-[200px] truncate">{activity.currentDescription}</span>
          </div>
        )}
        {activity?.status === 'absent' && activity.currentDescription && (
          <span className="hidden lg:block text-[11px] text-warning-foreground shrink-0">{activity.currentDescription}</span>
        )}
      </button>
      <div className="flex gap-1">
        <button onClick={onEmail} className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary" title="E-Mail" aria-label="E-Mail">
          <Mail className="h-4 w-4" />
        </button>
        <button onClick={onMessage} className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary" title="Nachricht" aria-label="Nachricht">
          <MessageSquare className="h-4 w-4" />
        </button>
        <ItemActions items={actions} />
      </div>
    </div>
  )
}

function LeaveRequestCard({ request, onApprove }: { request: LeaveRequest; onApprove: () => void }) {
  const initials = request.employeeName
    ?.split(' ')
    .map((n) => n[0])
    .join('')
    .slice(0, 2)
    .toUpperCase() ?? '??'

  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary-light text-sm font-medium text-primary">
            {initials}
          </div>
          <div>
            <h4 className="text-sm font-medium text-foreground">{request.employeeName ?? 'Unbekannt'}</h4>
            <div className="flex items-center gap-2 mt-0.5">
              <span className="rounded-full px-2 py-0.5 text-[10px] font-medium bg-info-light text-info">
                {request.leaveType?.name ?? 'Abwesenheit'}
              </span>
              <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${leaveStatusColors[request.status]}`}>
                {leaveStatusLabels[request.status]}
              </span>
            </div>
          </div>
        </div>
      </div>

      <div className="space-y-1.5 text-sm text-muted-foreground mb-3">
        <div className="flex items-center gap-2">
          <Calendar className="h-3.5 w-3.5" />
          <span>
            {new Date(request.startDate).toLocaleDateString('de-DE')}
            {request.startDate !== request.endDate && ` – ${new Date(request.endDate).toLocaleDateString('de-DE')}`}
            {' '}({request.totalDays} {request.totalDays === 1 ? 'Tag' : 'Tage'})
          </span>
        </div>
        {request.reason && <p className="text-sm text-text-body">{request.reason}</p>}
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
// Add Training Dialog (kept from Zustand mock)
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
                <SelectTrigger><SelectValue /></SelectTrigger>
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
            <Input placeholder="z.B. DRK" value={provider} onChange={(e) => setProvider(e.target.value)} />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Gültigkeit (Monate)</Label>
              <Input type="number" min={0} placeholder="0 = unbegrenzt" value={validityMonths} onChange={(e) => setValidityMonths(Number(e.target.value))} />
            </div>
            <div className="flex items-center gap-2 pt-6">
              <Checkbox id="mandatory" checked={mandatory} onCheckedChange={(v) => setMandatory(v === true)} />
              <Label htmlFor="mandatory" className="text-sm font-normal cursor-pointer">Pflichtschulung</Label>
            </div>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => { reset(); onOpenChange(false) }}>Abbrechen</Button>
          <Button onClick={handleAdd} disabled={!name.trim() || !duration.trim() || !provider.trim()}>Anlegen</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ============================================================
// Record Participation Dialog (kept from Zustand mock)
// ============================================================

function RecordParticipationDialog({ open, onOpenChange, trainings, members, onRecord }: {
  open: boolean
  onOpenChange: (v: boolean) => void
  trainings: Training[]
  members: EmployeeProfile[]
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
    const emp = members.find((m) => m.id === memberId)
    if (!emp) return
    const training = trainings.find((t) => t.id === trainingId)
    const memberName = emp.userName ?? 'Unbekannt'
    const participation: Omit<TrainingParticipation, 'id'> = {
      trainingId, memberId, memberName, status,
      ...(completedAt ? { completedAt } : {}),
    }
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
              <SelectTrigger><SelectValue placeholder="Mitarbeiter wählen..." /></SelectTrigger>
              <SelectContent>
                {members.map((m) => (
                  <SelectItem key={m.id} value={m.id}>{m.userName ?? 'Unbekannt'}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1.5">
            <Label>Schulung *</Label>
            <Select value={trainingId} onValueChange={setTrainingId}>
              <SelectTrigger><SelectValue placeholder="Schulung wählen..." /></SelectTrigger>
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
                <SelectTrigger><SelectValue /></SelectTrigger>
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
          <Button variant="outline" onClick={() => { reset(); onOpenChange(false) }}>Abbrechen</Button>
          <Button onClick={handleRecord} disabled={!memberId || !trainingId}>Erfassen</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

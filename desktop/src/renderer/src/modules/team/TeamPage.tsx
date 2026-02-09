import { useState } from 'react'
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
  CheckCircle2,
  XCircle,
  MoreVertical,
} from 'lucide-react'
import { toast } from 'sonner'
import { useTeamStore, type TeamMember, type HRRequest } from '@/stores/team'
import { useMeetingsStore } from '@/stores/meetings'
import { useNavigationStore } from '@/stores/navigation'
import { ItemActions, ConfirmDialog, EmptyState } from '@/components/shared'
import { MemberDetailPanel } from './MemberDetailPanel'
import { InviteMemberDialog } from './InviteMemberDialog'
import { EditMemberDialog } from './EditMemberDialog'
import { HRApprovalDialog } from './HRApprovalDialog'
import { AbsenceCalendar } from './AbsenceCalendar'

type TabKey = 'members' | 'requests' | 'absences'

const statusColors: Record<string, string> = {
  online: 'bg-success',
  away: 'bg-warning',
  offline: 'bg-text-disabled',
  dnd: 'bg-error',
}

const requestTypeLabels: Record<string, string> = {
  vacation: 'Urlaub',
  sick: 'Krankheit',
  overtime: 'Ueberstunden',
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

export default function TeamPage() {
  const navigate = useNavigate()
  const { members, requests, departments, approveRequest, rejectRequest, deactivateMember } = useTeamStore()
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

  // Cross-module actions
  const handleEmail = (member: TeamMember) => {
    setIntent({ type: 'compose-email', data: { email: member.email, name: `${member.firstName} ${member.lastName}` } })
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
          { key: 'members' as const, label: `Mitglieder (${activeMembers.length})` },
          { key: 'requests' as const, label: `Anfragen (${pendingCount} offen)` },
          { key: 'absences' as const, label: 'Abwesenheiten' },
        ]).map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`border-b-2 px-1 pb-2 text-sm transition-colors ${
              tab === t.key ? 'border-primary text-primary font-medium' : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
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
        description={`${confirmDeactivate?.firstName} ${confirmDeactivate?.lastName} wird deaktiviert. Das Konto kann spaeter reaktiviert werden.`}
        confirmLabel="Deaktivieren"
        variant="destructive"
        onConfirm={() => confirmDeactivate && handleDeactivate(confirmDeactivate)}
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
        <ItemActions actions={actions} />
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
        <ItemActions actions={actions} />
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

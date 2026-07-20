/**
 * Shared, chrome-less member profile body — modelled on the Kontakte
 * ContactDetailPanel: a fixed-height column (gradient header + tabs) that never
 * resizes with content. Sections are always visible (no collapsibles in the
 * overview), text is readable (text-sm), and the dense/interactive areas
 * (payroll, documents) live in their own tabs. Rendered inside the in-app
 * profile window (MemberProfileDialog) and the deep-link route (MemberDetailPage).
 *
 * R-4: isSelf-Bypass removed; own-scope grants cover the member's own file.
 * Per-drawer edit buttons (data_personal:edit / data_job:edit, scoped).
 * Absence drawer (absence_data:view, scoped). Offboard via header actions menu.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import {
  Mail,
  Phone,
  MessageSquare,
  Briefcase,
  MapPin,
  Calendar,
  Shield,
  User,
  X,
  Loader2,
  MoreVertical,
  Pencil,
  LogOut,
} from 'lucide-react'
import { Separator } from '@/components/ui/separator'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from '@/components/ui/dropdown-menu'
import {
  useEmployee,
  useEmployeeLeaveBalance,
} from '@/api/hooks/hr-hooks'
import { useUserModules } from '@/api/hooks/useModuleAssignments'
import { useHasCapability } from '@/hooks/useCapability'
import { useMeetingsStore } from '@/stores/meetings'
import { useNavigationStore } from '@/stores/navigation'
import { LEADABLE_MODULES } from '@/lib/module-settings'
import { formatDate } from '@/lib/format'
import { MODULE_DISPLAY_NAMES } from './ModuleAssignmentTab'
import { EmployeePayrollData } from './EmployeePayrollData'
import { DocumentsSection, EmployeeModuleLeadSection } from './ProfileSections'
import { UserRolesSection } from '@/components/shared/rbac/UserRolesSection'
import { useAdminUsers } from '@/api/hooks/useAdminUsers'
import { useHrScopedCapability } from './useTeamPermissions'
import { EditMemberDialog } from './EditMemberDialog'
import { OffboardEmployeeDialog } from './OffboardEmployeeDialog'
import { useOffboardEmployee } from '@/api/hooks/hr-hooks'

function getTenure(joinDate: string): string {
  const join = new Date(joinDate)
  const now = new Date()
  const months =
    (now.getFullYear() - join.getFullYear()) * 12 + (now.getMonth() - join.getMonth())
  if (months < 1) return 'Neu'
  if (months < 12) return `${months}M`
  const years = Math.floor(months / 12)
  const rem = months % 12
  return rem > 0 ? `${years}J ${rem}M` : `${years}J`
}

interface MemberProfileContentProps {
  memberId: string
  /** Called right before navigating away (e.g. to close the host window). */
  onNavigateAway?: () => void
  /** When provided, renders a close control in the header (window hosts). */
  onClose?: () => void
}

export function MemberProfileContent({ memberId, onNavigateAway, onClose }: MemberProfileContentProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { startCall } = useMeetingsStore()
  const { setIntent } = useNavigationStore()

  const { data: employee, isLoading } = useEmployee(memberId)
  const { data: balance } = useEmployeeLeaveBalance(employee?.userId ?? '')
  const { data: userGrants = [] } = useUserModules(employee?.userId ?? '')

  // R-4: scope-aware drawer gates (isSelf-Bypass entfernt — own-scope Grants decken das)
  const targetUserId = employee?.userId ?? null
  const canViewPersonal = useHrScopedCapability('team:data_personal:view', targetUserId)
  const canEditPersonal = useHrScopedCapability('team:data_personal:edit', targetUserId)
  const canViewJob = useHrScopedCapability('team:data_job:view', targetUserId)
  const canEditJob = useHrScopedCapability('team:data_job:edit', targetUserId)
  const canViewAbsenceData = useHrScopedCapability('team:absence_data:view', targetUserId)
  const canViewPayroll = useHasCapability('team:payroll:view')
  const canOffboard = useHasCapability('team:employee:offboard')

  // Roles section
  const canAssignHr = useHasCapability('team:role:assign')
  const canAssignIt = useHasCapability('admin:role:assign')
  const showRoles = canAssignHr || canAssignIt
  const { data: adminUsers = [] } = useAdminUsers()
  const memberAccount = employee?.userId
    ? adminUsers.find((u) => u.id === employee.userId)
    : undefined

  const canManageLeads = useHasCapability('admin:modules:manage')
  const hasLeadable = userGrants.some((g) => LEADABLE_MODULES.includes(g.moduleId))
  const showModuleLead = canManageLeads && hasLeadable && !!employee?.userId

  // Edit dialogs
  const [editDialogOpen, setEditDialogOpen] = useState(false)
  const [offboardOpen, setOffboardOpen] = useState(false)

  // Offboard mutation (calls MSW handler POST /api/v1/hr/employees/:id/offboard)
  const offboardMutation = useOffboardEmployee()

  const fullName = employee
    ? (employee.userName ?? employee.department ?? t('team.member.employee'))
    : t('common.loading')

  const initials = fullName
    .split(' ')
    .map((n: string) => n[0])
    .join('')
    .slice(0, 2)
    .toUpperCase()

  const CONTRACT_TYPE_KEYS: Record<string, string> = {
    full_time: 'team.contractType.fullTime',
    part_time: 'team.contractType.partTime',
    mini_job:  'team.contractType.miniJob',
    intern:    'team.contractType.internship',
    temporary: 'team.contractType.temporary',
  }
  const contractLabel = employee?.contractType
    ? t(CONTRACT_TYPE_KEYS[employee.contractType] ?? 'team.contractType.fullTime')
    : ''
  const tenure = employee ? getTenure(employee.startDate) : ''

  function goTo(path: string) {
    onNavigateAway?.()
    navigate(path)
  }
  function handleEmail() {
    setIntent({ type: 'compose-email', data: { to: employee?.userEmail ?? '', name: fullName } })
    goTo('/mails')
  }
  function handleCall() {
    startCall(fullName, initials)
  }
  function handleMessage() {
    setIntent({ type: 'send-message', data: { name: fullName } })
    goTo('/chat')
  }

  function handleOffboardConfirm(data: {
    lastWorkDay: string
    exitDate: string
    exitType: string
    reason: string
    backfill: boolean
    successorUserId?: string
  }) {
    if (!employee) return
    offboardMutation.mutate(
      { id: employee.id, data },
      { onSuccess: () => setOffboardOpen(false) },
    )
  }

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center py-16">
        <Loader2 className="h-6 w-6 animate-spin text-primary" />
      </div>
    )
  }
  if (!employee) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-3 py-16">
        <p className="text-sm text-muted-foreground">{t('team.member.notFound')}</p>
      </div>
    )
  }

  const bothSectionsHidden = !canViewPersonal && !canViewJob

  return (
    <div className="flex h-full min-h-0 flex-col overflow-hidden">
      {/* Gradient header */}
      <div className="relative shrink-0 bg-gradient-to-br from-primary to-primary-dark px-6 py-5">
        {onClose && (
          <button
            type="button"
            onClick={onClose}
            aria-label={t('shared.detailPanel.close')}
            className="absolute right-4 top-4 rounded-md p-1.5 text-primary-foreground/70 transition-colors hover:bg-primary-foreground/20 hover:text-primary-foreground"
          >
            <X className="h-4 w-4" />
          </button>
        )}

        {/* Header actions menu (Austritt + ggf. weitere) */}
        <div className="absolute right-12 top-4">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                type="button"
                aria-label={t('shared.actions')}
                className="rounded-md p-1.5 text-primary-foreground/70 transition-colors hover:bg-primary-foreground/20 hover:text-primary-foreground"
              >
                <MoreVertical className="h-4 w-4" />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-52">
              {/* Austritt — nur mit team:employee:offboard, nur im Profil */}
              {canOffboard && (
                <>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    className="text-destructive focus:bg-destructive/10 focus:text-destructive gap-2"
                    onClick={() => setOffboardOpen(true)}
                  >
                    <LogOut className="h-3.5 w-3.5" />
                    {t('team.offboard.initiate')}
                  </DropdownMenuItem>
                </>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        <div className="flex items-center gap-4">
          <div className="flex h-16 w-16 shrink-0 items-center justify-center rounded-full border-2 border-primary-foreground/30 bg-primary-foreground/20 text-xl font-semibold text-primary-foreground">
            {initials || <User className="h-7 w-7 opacity-70" />}
          </div>
          <div className="min-w-0">
            <h2 className="font-semibold leading-tight text-primary-foreground">{fullName}</h2>
            {employee.positionTitle && (
              <p className="truncate text-sm text-primary-foreground/70">{employee.positionTitle}</p>
            )}
            <div className="mt-2 flex flex-wrap items-center gap-1.5">
              {contractLabel && (
                <span className="rounded-full bg-primary-foreground/20 px-2 py-0.5 text-xs text-primary-foreground">
                  {contractLabel}
                </span>
              )}
              {employee.workDaysPerWeek ? (
                <span className="rounded-full bg-primary-foreground/20 px-2 py-0.5 text-xs text-primary-foreground">
                  {employee.workDaysPerWeek} {t('team.detail.daysPerWeek')}
                </span>
              ) : null}
              {employee.department && employee.department !== fullName && (
                <span className="rounded-full bg-primary-foreground/20 px-2 py-0.5 text-xs text-primary-foreground">
                  {employee.department}
                </span>
              )}
            </div>
          </div>
        </div>

        {/* Quick actions */}
        <div className="mt-4 flex flex-wrap gap-2">
          <button
            onClick={handleEmail}
            className="flex items-center gap-1.5 rounded-lg bg-primary-foreground/20 px-3 py-1.5 text-xs text-primary-foreground transition-colors hover:bg-primary-foreground/30"
          >
            <Mail className="h-3.5 w-3.5" />
            E-Mail
          </button>
          <button
            onClick={handleCall}
            className="flex items-center gap-1.5 rounded-lg bg-primary-foreground/20 px-3 py-1.5 text-xs text-primary-foreground transition-colors hover:bg-primary-foreground/30"
          >
            <Phone className="h-3.5 w-3.5" />
            {t('team.detail.call')}
          </button>
          <button
            onClick={handleMessage}
            className="flex items-center gap-1.5 rounded-lg bg-primary-foreground/20 px-3 py-1.5 text-xs text-primary-foreground transition-colors hover:bg-primary-foreground/30"
          >
            <MessageSquare className="h-3.5 w-3.5" />
            Chat
          </button>
        </div>
      </div>

      {/* Tabs */}
      <Tabs defaultValue="overview" className="flex min-h-0 flex-1 flex-col overflow-hidden">
        <TabsList className="shrink-0 gap-1 px-6 pt-2">
          <TabsTrigger value="overview">{t('team.detail.tabOverview')}</TabsTrigger>
          {canViewPayroll && (
            <TabsTrigger value="payroll">{t('team.payroll.masterData.title')}</TabsTrigger>
          )}
        </TabsList>

        {/* Overview */}
        <TabsContent value="overview" className="mt-0 flex-1 space-y-5 overflow-y-auto p-6">

          {/* Kontakt — data_personal:view (scoped) */}
          {canViewPersonal && (
            <section>
              <SectionHeader
                title={t('team.detail.contact')}
                editLabel={canEditPersonal ? t('common.edit') : undefined}
                onEdit={canEditPersonal ? () => setEditDialogOpen(true) : undefined}
              />
              <div className="space-y-2.5">
                {employee.userEmail ? (
                  <InfoRow icon={Mail}>
                    <a href={`mailto:${employee.userEmail}`} className="text-primary hover:underline">
                      {employee.userEmail}
                    </a>
                  </InfoRow>
                ) : null}
                {employee.emergencyContactPhone && (
                  <InfoRow icon={Phone}>
                    {employee.emergencyContactPhone} ({t('team.detail.emergencyContact')}:{' '}
                    {employee.emergencyContactName})
                  </InfoRow>
                )}
                {employee.addressCity && (
                  <InfoRow icon={MapPin}>
                    {[employee.addressStreet, employee.addressPostalCode, employee.addressCity]
                      .filter(Boolean)
                      .join(', ')}
                  </InfoRow>
                )}
                {!employee.userEmail && !employee.emergencyContactPhone && !employee.addressCity && (
                  <p className="text-sm text-muted-foreground">{t('team.detail.noContactData')}</p>
                )}
              </div>
            </section>
          )}

          {canViewPersonal && canViewJob && <Separator />}

          {/* Beide Schubladen verborgen — Hinweis */}
          {bothSectionsHidden && (
            <p className="text-sm text-muted-foreground">{t('rbac.gate.profileRestricted')}</p>
          )}

          {/* Beschäftigung — data_job:view (scoped) */}
          {canViewJob && (
            <section>
              <SectionHeader
                title={t('team.detail.employment')}
                editLabel={canEditJob ? t('common.edit') : undefined}
                onEdit={canEditJob ? () => setEditDialogOpen(true) : undefined}
              />
              <div className="space-y-2.5">
                {employee.department && <InfoRow icon={Briefcase}>{employee.department}</InfoRow>}
                {contractLabel && <InfoRow icon={Shield}>{contractLabel}</InfoRow>}
                {employee.startDate && (
                  <InfoRow icon={Calendar}>
                    {t('team.detail.since')} {formatDate(employee.startDate)} ({tenure})
                  </InfoRow>
                )}
                {employee.managerName && (
                  <InfoRow icon={Briefcase}>
                    {t('team.detail.reportsTo')}: {employee.managerName}
                  </InfoRow>
                )}
              </div>
            </section>
          )}

          {/* Abwesenheiten-Schublade (Salden/Resturlaub/Historie) — absence_data:view (scoped) */}
          {canViewAbsenceData && balance && (
            <>
              <Separator />
              <section>
                <SectionTitle>{t('team.detail.absenceData')}</SectionTitle>
                <div className="rounded-lg border border-border bg-card p-4">
                  <div className="flex items-end gap-2">
                    <span className="text-3xl font-bold text-primary">{balance.remaining}</span>
                    <span className="mb-1 text-sm text-muted-foreground">
                      / {balance.totalEntitlement} {t('team.detail.days')}
                    </span>
                  </div>
                  <div className="mt-2 h-2 w-full overflow-hidden rounded-full bg-secondary">
                    <div
                      className="h-full rounded-full bg-primary transition-all"
                      style={{
                        width: `${Math.min(100, Math.round((balance.used / balance.totalEntitlement) * 100))}%`,
                      }}
                    />
                  </div>
                  <div className="mt-2 flex justify-between text-xs text-muted-foreground">
                    <span>{balance.used} {t('team.detail.taken')}</span>
                    {balance.carriedOver > 0 && (
                      <span className={balance.carryoverExpired ? 'text-error' : ''}>
                        {balance.carriedOver} {t('team.detail.carryover')}
                        {balance.carryoverExpired ? ` (${t('team.detail.expired')})` : ''}
                      </span>
                    )}
                  </div>
                </div>
              </section>
            </>
          )}

          {/* Dokumente (scoped via DocumentsSection) */}
          {(canViewJob || canViewPersonal) && (
            <>
              <Separator />
              <DocumentsSection
                memberId={memberId}
                targetUserId={targetUserId}
                embedded
              />
            </>
          )}

          {/* Roles & access */}
          {showRoles && memberAccount && (
            <>
              <Separator />
              <section>
                <SectionTitle>{t('team.member.roles.title')}</SectionTitle>
                <UserRolesSection
                  userId={memberAccount.id}
                  roleIds={memberAccount.roles}
                  editable={canAssignHr || canAssignIt}
                  displayName={fullName}
                />
              </section>
            </>
          )}

          {/* Module access */}
          {userGrants.length > 0 && (
            <>
              <Separator />
              <section>
                <SectionTitle>{t('team.member.modules.title')} ({userGrants.length})</SectionTitle>
                <div className="space-y-1.5">
                  {userGrants.map((g) => (
                    <div
                      key={g.moduleId}
                      className="flex items-center justify-between gap-2 rounded-md border border-border bg-card px-3 py-2 text-sm"
                    >
                      <span className="truncate font-medium text-foreground">
                        {MODULE_DISPLAY_NAMES[g.moduleId] ?? g.moduleId}
                      </span>
                    </div>
                  ))}
                </div>
              </section>
            </>
          )}

          {/* Module-Lead delegation (admin-only) */}
          {showModuleLead && employee.userId && (
            <>
              <Separator />
              <EmployeeModuleLeadSection userId={employee.userId} />
            </>
          )}
        </TabsContent>

        {/* Payroll — tab only shown with team:payroll:view */}
        {canViewPayroll && (
          <TabsContent value="payroll" className="mt-0 flex-1 overflow-y-auto p-6">
            <EmployeePayrollData
              employeeId={memberId}
              targetUserId={targetUserId}
              embedded
            />
          </TabsContent>
        )}
      </Tabs>

      {/* Edit-Dialog (PersonalDaten + Beschäftigung) */}
      <EditMemberDialog
        open={editDialogOpen}
        onOpenChange={setEditDialogOpen}
        member={employee}
      />

      {/* Offboard-Dialog */}
      {canOffboard && (
        <OffboardEmployeeDialog
          open={offboardOpen}
          onOpenChange={setOffboardOpen}
          employee={employee}
          onConfirm={handleOffboardConfirm}
          isPending={offboardMutation?.isPending ?? false}
        />
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Section helpers
// ---------------------------------------------------------------------------

function SectionTitle({ children }: { children: React.ReactNode }) {
  return (
    <h3 className="mb-3 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
      {children}
    </h3>
  )
}

/** Section header with optional edit-pencil icon */
function SectionHeader({
  title,
  editLabel,
  onEdit,
}: {
  title: string
  editLabel?: string
  onEdit?: () => void
}) {
  return (
    <div className="mb-3 flex items-center justify-between gap-2">
      <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
        {title}
      </h3>
      {onEdit && editLabel && (
        <button
          type="button"
          onClick={onEdit}
          aria-label={editLabel}
          className="flex items-center gap-1 rounded-md p-1 text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
        >
          <Pencil className="h-3 w-3" />
          <span className="text-[10px]">{editLabel}</span>
        </button>
      )}
    </div>
  )
}

function InfoRow({
  icon: Icon,
  children,
}: {
  icon: React.ComponentType<{ className?: string }>
  children: React.ReactNode
}) {
  return (
    <div className="flex items-center gap-3 text-sm text-foreground">
      <Icon className="h-4 w-4 shrink-0 text-muted-foreground" />
      <span className="min-w-0">{children}</span>
    </div>
  )
}

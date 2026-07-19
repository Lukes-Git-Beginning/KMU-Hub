import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  Mail,
  Phone,
  MessageSquare,
  Briefcase,
  MapPin,
  Calendar,
  Shield,
  PhoneCall,
  Loader2,
  FileText,
  Upload,
  ChevronDown,
  ChevronUp,
  AlertTriangle,
  ArrowRight,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { DetailPanel } from '@/components/shared'
import { toast } from 'sonner'
import { useUIStore } from '@/stores/ui'
import {
  useEmployee,
  useEmployeeLeaveBalance,
  useEmployeeDocuments,
  useDocumentCategories,
  useUploadEmployeeDocument,
} from '@/api/hooks/hr-hooks'
import { useUserModules } from '@/api/hooks/useModuleAssignments'
import { useInsightSettings } from '@/api/hooks/useBilling'
import { useNavigationStore } from '@/stores/navigation'
import { useHasCapability, useScopedCapability } from '@/hooks/useCapability'
import { useModuleLeadsStore } from '@/stores/moduleLeads'
import { LEADABLE_MODULES } from '@/lib/module-settings'
import { DEFAULT_INSIGHT_SETTINGS } from '@/lib/pricing'
import { MODULE_DISPLAY_NAMES } from './ModuleAssignmentTab'
import { EmployeePayrollData } from './EmployeePayrollData'
import { formatRelativeTime, formatDate } from '@/lib/format'

const contractTypeKeys: Record<string, string> = {
  full_time: 'team.contractType.fullTime',
  part_time: 'team.contractType.partTime',
  mini_job:  'team.contractType.miniJob',
  intern:    'team.contractType.internship',
  temporary: 'team.contractType.temporary',
  // legacy aliases — kept for any cached data still in transit
  praktikum: 'team.contractType.internship',
  freelance: 'team.contractType.temporary',
}

interface MemberDetailPanelProps {
  memberId: string
  memberName?: string
  memberInitials?: string
  onClose: () => void
  onEmail: () => void
  onCall: () => void
  onMessage: () => void
  onEdit: () => void
}

export function MemberDetailPanel({
  memberId,
  memberName,
  memberInitials,
  onClose,
  onEmail,
  onCall,
  onMessage,
  onEdit,
}: MemberDetailPanelProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { data: employee, isLoading } = useEmployee(memberId)
  const { data: balance } = useEmployeeLeaveBalance(employee?.userId ?? '')
  const { data: documents } = useEmployeeDocuments(memberId)

  // RBAC: Edit-Button only when caller has team:employee:edit on this employee.
  // scope=own → only the employee themselves may trigger; scope=team≈all (documented gap, R3-BRIEFING §1.4).
  const canEditEmployee = useScopedCapability('team:employee:edit', employee?.userId)

  const fullName = employee
    ? (employee.userName ?? `${employee.department ?? t('team.member.employee')}`)
    : (memberName ?? t('common.loading'))

  const initials = memberInitials ?? fullName
    .split(' ')
    .map((n) => n[0])
    .join('')
    .slice(0, 2)
    .toUpperCase()

  const tenure = employee ? getTenure(employee.startDate) : ''

  return (
    <DetailPanel
      open
      title={t('team.detail.title')}
      onClose={onClose}
      onExpand={() => {
        navigate(`/team/member/${memberId}`)
        onClose()
      }}
    >
      {/* Header with gradient */}
      <div className="relative -mx-5 -mt-1 mb-4">
        <div
          className="h-20 rounded-t-lg"
          style={{
            background: `linear-gradient(135deg, var(--primary) 0%, color-mix(in srgb, var(--primary) 60%, #000) 100%)`,
          }}
        />
        <div className="absolute -bottom-8 left-5 flex items-end gap-3">
          <div className="relative">
            <div className="flex h-16 w-16 items-center justify-center rounded-full border-4 border-card bg-primary-light text-lg font-bold text-primary">
              {initials}
            </div>
          </div>
        </div>
      </div>

      {isLoading ? (
        <div className="mt-6 flex items-center justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-primary" />
        </div>
      ) : (
        <div className="mt-6 space-y-4">
          {/* Name & Role */}
          <div>
            <h3 className="text-base font-medium text-foreground">{fullName}</h3>
            <p className="text-xs text-muted-foreground">{employee?.positionTitle ?? ''}</p>
            <div className="flex items-center gap-2 mt-1">
              <span className="text-[10px] text-muted-foreground">
                {t(contractTypeKeys[employee?.contractType ?? ''] ?? employee?.contractType ?? '')}
                {employee?.workDaysPerWeek ? ` · ${employee.workDaysPerWeek} ${t('team.detail.daysPerWeek')}` : ''}
              </span>
            </div>
          </div>

          {/* Quick Actions */}
          <div className="flex gap-2">
            <button
              onClick={onEmail}
              className="flex-1 flex items-center justify-center gap-1.5 rounded-lg border border-border py-2 text-xs text-foreground hover:bg-secondary transition-colors"
            >
              <Mail className="h-3.5 w-3.5" />
              E-Mail
            </button>
            <button
              onClick={onCall}
              className="flex-1 flex items-center justify-center gap-1.5 rounded-lg border border-border py-2 text-xs text-foreground hover:bg-secondary transition-colors"
            >
              <PhoneCall className="h-3.5 w-3.5" />
              {t('team.detail.call')}
            </button>
            <button
              onClick={onMessage}
              className="flex-1 flex items-center justify-center gap-1.5 rounded-lg border border-border py-2 text-xs text-foreground hover:bg-secondary transition-colors"
            >
              <MessageSquare className="h-3.5 w-3.5" />
              Chat
            </button>
          </div>

          {/* Contact Info */}
          {employee?.userEmail && (
            <section className="space-y-2">
              <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">{t('team.detail.contact')}</h4>
              <div className="space-y-1.5 text-xs">
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Mail className="h-3.5 w-3.5 shrink-0" />
                  <a href={`mailto:${employee.userEmail}`} className="text-primary hover:underline">{employee.userEmail}</a>
                </div>
                {employee.emergencyContactPhone && (
                  <div className="flex items-center gap-2 text-muted-foreground">
                    <Phone className="h-3.5 w-3.5 shrink-0" />
                    <span>{employee.emergencyContactPhone} ({t('team.detail.emergencyContact')}: {employee.emergencyContactName})</span>
                  </div>
                )}
                {employee.addressCity && (
                  <div className="flex items-center gap-2 text-muted-foreground">
                    <MapPin className="h-3.5 w-3.5 shrink-0" />
                    <span>
                      {[employee.addressStreet, employee.addressPostalCode, employee.addressCity].filter(Boolean).join(', ')}
                    </span>
                  </div>
                )}
              </div>
            </section>
          )}

          {/* Employment */}
          <section className="space-y-2">
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">{t('team.detail.employment')}</h4>
            <div className="space-y-1.5 text-xs">
              {employee?.department && (
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Briefcase className="h-3.5 w-3.5 shrink-0" />
                  <span>{employee.department}</span>
                </div>
              )}
              <div className="flex items-center gap-2 text-muted-foreground">
                <Shield className="h-3.5 w-3.5 shrink-0" />
                <span>{t(contractTypeKeys[employee?.contractType ?? ''] ?? employee?.contractType ?? '')}</span>
              </div>
              {employee?.startDate && (
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Calendar className="h-3.5 w-3.5 shrink-0" />
                  <span>{t('team.detail.since')} {formatDate(employee.startDate)} ({tenure})</span>
                </div>
              )}
              {employee?.managerName && (
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Briefcase className="h-3.5 w-3.5 shrink-0" />
                  <span>{t('team.detail.reportsTo')}: {employee.managerName}</span>
                </div>
              )}
            </div>
          </section>

          {/* Leave Balance */}
          {balance && (
            <section className="space-y-2">
              <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">{t('team.detail.leaveEntitlement')}</h4>
              <div className="rounded-lg border border-border bg-secondary/30 p-3">
                <div className="flex items-end gap-2 mb-2">
                  <span className="text-2xl font-bold text-primary">{balance.remaining}</span>
                  <span className="text-sm text-muted-foreground mb-0.5">/ {balance.totalEntitlement} {t('team.detail.days')}</span>
                </div>
                <div className="w-full h-2 rounded-full bg-secondary overflow-hidden mb-2">
                  <div
                    className="h-full rounded-full bg-primary transition-all"
                    style={{ width: `${Math.min(100, Math.round((balance.used / balance.totalEntitlement) * 100))}%` }}
                  />
                </div>
                <div className="flex justify-between text-[10px] text-muted-foreground">
                  <span>{balance.used} {t('team.detail.taken')}</span>
                  {balance.carriedOver > 0 && (
                    <span className={balance.carryoverExpired ? 'text-error' : ''}>
                      {balance.carriedOver} {t('team.detail.carryover')}{balance.carryoverExpired ? ` (${t('team.detail.expired')})` : ''}
                    </span>
                  )}
                </div>
              </div>
            </section>
          )}

          {/* Module-Zuteilung */}
          {employee?.userId && (
            <EmployeeModulesSection
              userId={employee.userId}
              onManage={() => onEdit()}
            />
          )}

          {/* Erweiterte Moduleinstellungen (Modul-Leiter) — admin/it only */}
          {employee?.userId && <EmployeeModuleLeadSection userId={employee.userId} />}

          {/* Lohn-Stammdaten (hr_only) */}
          <EmployeePayrollData employeeId={memberId} />

          {/* Documents */}
          <DocumentsSection memberId={memberId} documents={documents ?? []} />

          {/* Edit button — only with team:employee:edit (scoped: own or team≈all) */}
          {canEditEmployee && (
            <button
              onClick={onEdit}
              className="w-full rounded-lg border border-border py-2 text-xs text-foreground hover:bg-secondary transition-colors"
            >
              {t('team.detail.editProfile')}
            </button>
          )}
        </div>
      )}
    </DetailPanel>
  )
}

// ============================================================
// Documents Section with upload
// ============================================================
export function DocumentsSection({
  memberId,
  documents,
  embedded = false,
}: {
  memberId: string
  documents: { id: string; fileName?: string; categoryName?: string; createdAt: string }[]
  /** When true, render content always-open without the collapse toggle. */
  embedded?: boolean
}) {
  const { t } = useTranslation()
  const { data: categories } = useDocumentCategories(memberId)
  const uploadMutation = useUploadEmployeeDocument()
  const canUploadDoc = useHasCapability('team:documents:edit')

  const [expanded, setExpanded] = useState(false)
  const showContent = embedded || expanded
  const [showUpload, setShowUpload] = useState(false)
  const [categoryId, setCategoryId] = useState('')
  const [notes, setNotes] = useState('')
  const [_fileName, setFileName] = useState('')

  const handleUpload = () => {
    if (!categoryId) {
      toast.error(t('team.documents.selectCategory'))
      return
    }
    // In real app, fileId comes from a file upload service. Here we simulate.
    const mockFileId = `file-${Date.now()}`
    uploadMutation.mutate(
      {
        employeeId: memberId,
        data: {
          categoryId,
          fileId: mockFileId,
          notes: notes.trim() || undefined,
        },
      },
      {
        onSuccess: () => {
          setShowUpload(false)
          setCategoryId('')
          setNotes('')
          setFileName('')
        },
      },
    )
  }

  const Chevron = expanded ? ChevronUp : ChevronDown

  return (
    <section className="space-y-2">
      <button
        onClick={() => { if (!embedded) setExpanded(!expanded) }}
        className={`flex items-center gap-1 w-full text-left ${embedded ? 'cursor-default' : ''}`}
      >
        <h4 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          {t('team.documents.title')} ({documents.length})
        </h4>
        {!embedded && <Chevron className="h-3 w-3 text-muted-foreground ml-auto" />}
      </button>

      {showContent && (
        <div className="space-y-2">
          {/* Document list */}
          {documents.length > 0 ? (
            <div className="space-y-1">
              {documents.map((doc) => (
                <div key={doc.id} className="flex items-center gap-2 text-sm text-muted-foreground">
                  <FileText className="h-3.5 w-3.5 shrink-0" />
                  <span className="truncate">{doc.fileName ?? doc.categoryName ?? t('team.documents.document')}</span>
                  <span className="text-[10px] ml-auto shrink-0">
                    {formatDate(doc.createdAt)}
                  </span>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground italic">{t('team.documents.noDocuments')}</p>
          )}

          {/* Upload area — only with team:documents:edit */}
          {!showUpload && canUploadDoc ? (
            <Button
              variant="outline"
              size="sm"
              className="w-full text-xs"
              onClick={() => setShowUpload(true)}
            >
              <Upload className="mr-1.5 h-3.5 w-3.5" />
              {t('team.documents.uploadDocument')}
            </Button>
          ) : showUpload && canUploadDoc ? (
            <div className="rounded-lg border border-border bg-secondary/30 p-3 space-y-2">
              <div className="space-y-1">
                <Label className="text-xs">{t('team.documents.category')}</Label>
                <select
                  value={categoryId}
                  onChange={(e) => setCategoryId(e.target.value)}
                  className="w-full rounded border border-border bg-input-background px-2 py-1.5 text-xs text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                >
                  <option value="">{t('team.member.selectPlaceholder')}</option>
                  {(categories ?? []).map((cat) => (
                    <option key={cat.id} value={cat.id}>{cat.name}</option>
                  ))}
                  {/* Fallback categories if API not loaded */}
                  {!categories && (
                    <>
                      <option value="contract">Vertrag</option>
                      <option value="certificate">Zeugnis</option>
                      <option value="id">Ausweis</option>
                      <option value="cert">Zertifikat</option>
                      <option value="other">Sonstiges</option>
                    </>
                  )}
                </select>
              </div>
              <div className="space-y-1">
                <Label className="text-xs">{t('team.documents.file')}</Label>
                <Input
                  type="file"
                  onChange={(e) => setFileName(e.target.files?.[0]?.name ?? '')}
                  className="text-xs h-8"
                />
              </div>
              <div className="space-y-1">
                <Label className="text-xs">{t('team.documents.notesOptional')}</Label>
                <Input
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                  placeholder={t('team.documents.notesPlaceholder')}
                  className="text-xs h-8"
                />
              </div>
              <div className="flex gap-2">
                <Button
                  size="sm"
                  className="flex-1 text-xs h-7"
                  onClick={handleUpload}
                  disabled={uploadMutation.isPending}
                >
                  {uploadMutation.isPending ? (
                    <Loader2 className="mr-1 h-3 w-3 animate-spin" />
                  ) : (
                    <Upload className="mr-1 h-3 w-3" />
                  )}
                  {t('common.upload')}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  className="text-xs h-7"
                  onClick={() => { setShowUpload(false); setCategoryId(''); setNotes(''); setFileName('') }}
                >
                  {t('common.cancel')}
                </Button>
              </div>
            </div>
          ) : null}
        </div>
      )}
    </section>
  )
}

// ============================================================
// Employee Modules Section
// ============================================================
export function EmployeeModulesSection({
  userId,
  onManage,
}: {
  userId: string
  onManage: () => void
}) {
  const { t, i18n } = useTranslation()
  const { setIntent } = useNavigationStore()
  const openSettingsOverlay = useUIStore((s) => s.openSettingsOverlay)
  const { data: userGrants = [] } = useUserModules(userId)
  const { data: insightSettings } = useInsightSettings()
  const minInactivityDays = insightSettings?.minInactivityDays ?? DEFAULT_INSIGHT_SETTINGS.minInactivityDays

  const [expanded, setExpanded] = useState(true)

  function isInactive(lastActiveAt: string | null): boolean {
    if (!lastActiveAt) return true
    const diffDays = (Date.now() - new Date(lastActiveAt).getTime()) / 86_400_000
    return diffDays > minInactivityDays
  }

  const handleManage = () => {
    setIntent({
      type: 'open-team-modulzuteilung',
      data: { userIds: [userId] },
    })
    onManage()
    openSettingsOverlay('modulzuteilung')
  }

  const Chevron = expanded ? ChevronUp : ChevronDown

  if (userGrants.length === 0) return null

  return (
    <section className="space-y-2">
      <button
        type="button"
        onClick={() => setExpanded((v) => !v)}
        className="flex items-center gap-1 w-full text-left"
        aria-expanded={expanded}
      >
        <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
          {t('team.member.modules.title')} ({userGrants.length})
        </h4>
        <Chevron className="h-3 w-3 text-muted-foreground ml-auto" aria-hidden="true" />
      </button>

      {expanded && (
        <div className="space-y-1">
          {userGrants.map((grant) => {
            const moduleName = MODULE_DISPLAY_NAMES[grant.moduleId] ?? grant.moduleId
            const inactive = isInactive(grant.lastActiveAt)
            const relTime = formatRelativeTime(grant.lastActiveAt, i18n.language)

            return (
              <div
                key={grant.moduleId}
                className={`flex items-center justify-between gap-2 rounded-md px-2.5 py-1.5 text-xs ${
                  inactive ? 'bg-warning/5' : 'bg-secondary/30'
                }`}
              >
                <span className="font-medium text-foreground truncate">{moduleName}</span>
                <span className={`flex items-center gap-1 shrink-0 ${inactive ? 'text-warning-foreground' : 'text-muted-foreground'}`}>
                  {inactive && <AlertTriangle className="h-3 w-3" aria-label="inaktiv" />}
                  {grant.lastActiveAt
                    ? t('team.member.modules.lastUsed', { time: relTime })
                    : t('team.member.modules.neverUsed')}
                </span>
              </div>
            )
          })}

          <button
            type="button"
            onClick={handleManage}
            className="flex w-full items-center justify-end gap-1.5 pt-1 text-xs text-primary hover:underline transition-colors"
          >
            {t('team.member.modules.manage')}
            <ArrowRight className="h-3 w-3" aria-hidden="true" />
          </button>
        </div>
      )}
    </section>
  )
}

// ============================================================
// Employee Module-Lead Section ("erweiterte Moduleinstellungen")
// Admin/IT delegates tenant-wide settings rights per module.
// ============================================================
export function EmployeeModuleLeadSection({ userId }: { userId: string }) {
  const { t } = useTranslation()
  const { data: userGrants = [] } = useUserModules(userId)
  const leadModules = useModuleLeadsStore((s) => s.leads[userId] ?? [])
  const toggleLead = useModuleLeadsStore((s) => s.toggleLead)
  const [expanded, setExpanded] = useState(false)

  // Delegating Modul-Leiter rights is module administration (admin/it_admin).
  const canManage = useHasCapability('admin:modules:manage')

  // Modules the employee has access to AND that expose tenant-wide settings.
  const leadable = userGrants
    .map((g) => g.moduleId)
    .filter((m) => LEADABLE_MODULES.includes(m))

  if (!canManage || leadable.length === 0) return null

  const Chevron = expanded ? ChevronUp : ChevronDown
  const activeCount = leadable.filter((m) => leadModules.includes(m)).length

  return (
    <section className="space-y-2">
      <button
        type="button"
        onClick={() => setExpanded((v) => !v)}
        className="flex items-center gap-1 w-full text-left"
        aria-expanded={expanded}
      >
        <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
          {t('team.member.moduleLead.title')}{activeCount > 0 ? ` (${activeCount})` : ''}
        </h4>
        <Chevron className="h-3 w-3 text-muted-foreground ml-auto" aria-hidden="true" />
      </button>

      {expanded && (
        <div className="space-y-1.5">
          <p className="text-[10px] leading-relaxed text-muted-foreground">
            {t('team.member.moduleLead.hint')}
          </p>
          {leadable.map((m) => {
            const isLead = leadModules.includes(m)
            const moduleName = MODULE_DISPLAY_NAMES[m] ?? m
            return (
              <button
                key={m}
                type="button"
                onClick={() => toggleLead(userId, m)}
                aria-pressed={isLead}
                className={`flex w-full items-center justify-between gap-2 rounded-md px-2.5 py-1.5 text-xs transition-colors ${
                  isLead ? 'bg-primary-light text-primary' : 'bg-secondary/30 text-foreground hover:bg-secondary'
                }`}
              >
                <span className="font-medium truncate">{moduleName}</span>
                <span
                  className={`flex h-4 w-7 shrink-0 items-center rounded-full px-0.5 transition-colors ${
                    isLead ? 'justify-end bg-primary' : 'justify-start bg-border'
                  }`}
                  aria-hidden="true"
                >
                  <span className="h-3 w-3 rounded-full bg-white shadow-sm" />
                </span>
              </button>
            )
          })}
        </div>
      )}
    </section>
  )
}

function getTenure(joinDate: string): string {
  const join = new Date(joinDate)
  const now = new Date()
  const months = (now.getFullYear() - join.getFullYear()) * 12 + (now.getMonth() - join.getMonth())
  if (months < 1) return 'Neu'
  if (months < 12) return `${months}M`
  const years = Math.floor(months / 12)
  const rem = months % 12
  return rem > 0 ? `${years}J ${rem}M` : `${years}J`
}

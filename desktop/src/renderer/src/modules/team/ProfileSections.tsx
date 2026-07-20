/**
 * Shared profile section components extracted from MemberDetailPanel (R-4 cleanup).
 * DocumentsSection and EmployeeModuleLeadSection are rendered by MemberProfileContent.
 * DocumentsSection is now scope-gated via useHrScopedCapability (R-4 §3.4).
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  ChevronDown,
  ChevronUp,
  FileText,
  Upload,
  Loader2,
  AlertTriangle,
  ArrowRight,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { toast } from 'sonner'
import { useUIStore } from '@/stores/ui'
import {
  useEmployeeDocuments,
  useDocumentCategories,
  useUploadEmployeeDocument,
} from '@/api/hooks/hr-hooks'
import { useUserModules } from '@/api/hooks/useModuleAssignments'
import { useInsightSettings } from '@/api/hooks/useBilling'
import { useNavigationStore } from '@/stores/navigation'
import { useHasCapability } from '@/hooks/useCapability'
import { useHrScopedCapability } from './useTeamPermissions'
import { useModuleLeadsStore } from '@/stores/moduleLeads'
import { LEADABLE_MODULES } from '@/lib/module-settings'
import { DEFAULT_INSIGHT_SETTINGS } from '@/lib/pricing'
import { MODULE_DISPLAY_NAMES } from './ModuleAssignmentTab'
import { formatRelativeTime, formatDate } from '@/lib/format'

// ============================================================
// Documents Section with upload
// ============================================================
export function DocumentsSection({
  memberId,
  targetUserId,
  embedded = false,
}: {
  memberId: string
  /** Auth userId of the employee — used for scope-aware gate. */
  targetUserId: string | null | undefined
  documents?: { id: string; fileName?: string; categoryName?: string; createdAt: string }[]
  /** When true, render content always-open without the collapse toggle. */
  embedded?: boolean
}) {
  const { t } = useTranslation()
  const { data: documents = [] } = useEmployeeDocuments(memberId)
  const { data: categories } = useDocumentCategories(memberId)
  const uploadMutation = useUploadEmployeeDocument()

  // R-4: scope-aware gate — only show documents drawer if the viewer has access
  const canView = useHrScopedCapability('team:documents:view', targetUserId)
  const canEdit = useHrScopedCapability('team:documents:edit', targetUserId)
  // Payslip documents are salary data: a manager with documents:view (team)
  // but no salary grant must never see them (R4-BRIEFING §0 — never salary).
  const canViewSalaryDocs = useHrScopedCapability('team:salary:view', targetUserId)
  const visibleDocuments = documents.filter((doc) => {
    if (canViewSalaryDocs) return true
    const cat = (doc as { categoryId?: string; categoryName?: string })
    return cat.categoryId !== 'hrcat-payroll' && cat.categoryName !== 'Gehaltsabrechnungen'
  })

  const [expanded, setExpanded] = useState(false)
  const showContent = embedded || expanded
  const [showUpload, setShowUpload] = useState(false)
  const [categoryId, setCategoryId] = useState('')
  const [notes, setNotes] = useState('')
  const [_fileName, setFileName] = useState('')

  // Gate: hide entire section if no view access
  if (!canView) return null

  const handleUpload = () => {
    if (!categoryId) {
      toast.error(t('team.documents.selectCategory'))
      return
    }
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
          {t('team.documents.title')} ({visibleDocuments.length})
        </h4>
        {!embedded && <Chevron className="h-3 w-3 text-muted-foreground ml-auto" />}
      </button>

      {showContent && (
        <div className="space-y-2">
          {visibleDocuments.length > 0 ? (
            <div className="space-y-1">
              {visibleDocuments.map((doc) => (
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

          {/* Upload area — only with team:documents:edit (scoped) */}
          {!showUpload && canEdit ? (
            <Button
              variant="outline"
              size="sm"
              className="w-full text-xs"
              onClick={() => setShowUpload(true)}
            >
              <Upload className="mr-1.5 h-3.5 w-3.5" />
              {t('team.documents.uploadDocument')}
            </Button>
          ) : showUpload && canEdit ? (
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
// Employee Module-Lead Section
// ============================================================
export function EmployeeModuleLeadSection({ userId }: { userId: string }) {
  const { t } = useTranslation()
  const { data: userGrants = [] } = useUserModules(userId)
  const leadModules = useModuleLeadsStore((s) => s.leads[userId] ?? [])
  const toggleLead = useModuleLeadsStore((s) => s.toggleLead)
  const [expanded, setExpanded] = useState(false)

  const canManage = useHasCapability('admin:modules:manage')

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

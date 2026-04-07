import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Separator } from '@/components/ui/separator'
import { Save, Plus, Trash2, Users, ShieldCheck, Clock, Palmtree, Stethoscope, Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import { ConfirmDialog } from '@/components/shared'
import { useSettingsStore, type LeaveType } from '@/stores/settings'
import { useHRSettings, useUpdateHRSettings } from '@/api/hooks/hr-hooks'

export function TeamSettingsTab() {
  const { t } = useTranslation()
  const { teamAdmin, updateTeamAdmin } = useSettingsStore()

  const [departments, setDepartments] = useState(teamAdmin.departments)
  const [newDept, setNewDept] = useState('')
  const [roles, setRoles] = useState(teamAdmin.roles)
  const [newRole, setNewRole] = useState('')
  const [leaveTypes, setLeaveTypes] = useState<LeaveType[]>(teamAdmin.leaveTypes)
  const [workHours, setWorkHours] = useState(teamAdmin.workHoursPerWeek)
  const [overtimeEnabled, setOvertimeEnabled] = useState(teamAdmin.overtimeEnabled)
  const [deleteTarget, setDeleteTarget] = useState<{ type: 'dept' | 'role' | 'leave'; index: number } | null>(null)

  const addDepartment = () => {
    const trimmed = newDept.trim()
    if (!trimmed || departments.includes(trimmed)) return
    setDepartments([...departments, trimmed])
    setNewDept('')
  }

  const removeDepartment = (idx: number) => {
    setDepartments(departments.filter((_, i) => i !== idx))
    setDeleteTarget(null)
  }

  const addRole = () => {
    const trimmed = newRole.trim()
    if (!trimmed || roles.includes(trimmed)) return
    setRoles([...roles, trimmed])
    setNewRole('')
  }

  const removeRole = (idx: number) => {
    setRoles(roles.filter((_, i) => i !== idx))
    setDeleteTarget(null)
  }

  const addLeaveType = () => {
    setLeaveTypes([...leaveTypes, { name: t('settings.team.newLeaveTypeName'), days: 0, color: '#6b7280' }])
  }

  const updateLeave = (idx: number, data: Partial<LeaveType>) => {
    setLeaveTypes(leaveTypes.map((lt, i) => (i === idx ? { ...lt, ...data } : lt)))
  }

  const removeLeave = (idx: number) => {
    setLeaveTypes(leaveTypes.filter((_, i) => i !== idx))
    setDeleteTarget(null)
  }

  const handleSaveDepartments = () => {
    updateTeamAdmin({ departments })
    toast.success(t('settings.team.savedDepartments'))
  }

  const handleSaveRoles = () => {
    updateTeamAdmin({ roles })
    toast.success(t('settings.team.savedRoles'))
  }

  const handleSaveLeave = () => {
    updateTeamAdmin({ leaveTypes })
    toast.success(t('settings.team.savedLeave'))
  }

  const handleSaveWorktime = () => {
    updateTeamAdmin({ workHoursPerWeek: workHours, overtimeEnabled })
    toast.success(t('settings.team.savedWorktime'))
  }

  return (
    <div className="max-w-2xl">
      <h2 className="text-foreground mb-1">{t('settings.team.title')}</h2>
      <p className="text-sm text-muted-foreground mb-6">{t('settings.team.subtitle')}</p>

      {/* Departments */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Users className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">{t('settings.team.sectionDepartments')}</h3>
        </div>

        <div className="space-y-1.5 mb-3">
          {departments.map((dept, idx) => (
            <div key={idx} className="flex items-center justify-between rounded-md border border-border px-3 py-2 text-sm text-foreground">
              {dept}
              <button
                onClick={() => setDeleteTarget({ type: 'dept', index: idx })}
                className="text-muted-foreground hover:text-error transition-colors"
              >
                <Trash2 className="h-3.5 w-3.5" />
              </button>
            </div>
          ))}
        </div>

        <div className="flex gap-2">
          <Input
            placeholder={t('settings.team.newDepartment')}
            value={newDept}
            onChange={(e) => setNewDept(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && addDepartment()}
            className="flex-1"
          />
          <Button variant="outline" size="sm" onClick={addDepartment} disabled={!newDept.trim()}>
            <Plus className="h-4 w-4" />
          </Button>
        </div>

        <Button onClick={handleSaveDepartments} className="mt-3" size="sm">
          <Save className="mr-1.5 h-4 w-4" />
          {t('settings.team.saveDepartments')}
        </Button>
      </section>

      <Separator className="mb-8" />

      {/* Roles */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <ShieldCheck className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">{t('settings.team.sectionRoles')}</h3>
        </div>

        <div className="space-y-1.5 mb-3">
          {roles.map((role, idx) => (
            <div key={idx} className="flex items-center justify-between rounded-md border border-border px-3 py-2 text-sm text-foreground">
              {role}
              {idx > 0 && (
                <button
                  onClick={() => setDeleteTarget({ type: 'role', index: idx })}
                  className="text-muted-foreground hover:text-error transition-colors"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              )}
            </div>
          ))}
        </div>

        <div className="flex gap-2">
          <Input
            placeholder={t('settings.team.newRole')}
            value={newRole}
            onChange={(e) => setNewRole(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && addRole()}
            className="flex-1"
          />
          <Button variant="outline" size="sm" onClick={addRole} disabled={!newRole.trim()}>
            <Plus className="h-4 w-4" />
          </Button>
        </div>

        <Button onClick={handleSaveRoles} className="mt-3" size="sm">
          <Save className="mr-1.5 h-4 w-4" />
          {t('settings.team.saveRoles')}
        </Button>
      </section>

      <Separator className="mb-8" />

      {/* Leave types */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Palmtree className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">{t('settings.team.sectionLeave')}</h3>
        </div>

        <div className="space-y-2 mb-3">
          {leaveTypes.map((lt, idx) => (
            <div key={idx} className="flex items-center gap-2 rounded-md border border-border px-3 py-2">
              <input
                type="color"
                value={lt.color}
                onChange={(e) => updateLeave(idx, { color: e.target.value })}
                className="h-6 w-6 rounded border-none cursor-pointer"
              />
              <Input
                value={lt.name}
                onChange={(e) => updateLeave(idx, { name: e.target.value })}
                className="flex-1 h-8 text-sm"
              />
              <div className="flex items-center gap-1">
                <Input
                  type="number"
                  min={0}
                  value={lt.days}
                  onChange={(e) => updateLeave(idx, { days: Number(e.target.value) })}
                  className="w-16 h-8 text-sm text-center"
                />
                <span className="text-[10px] text-muted-foreground">{t('settings.team.daysPerYear')}</span>
              </div>
              <button
                onClick={() => setDeleteTarget({ type: 'leave', index: idx })}
                className="text-muted-foreground hover:text-error transition-colors"
              >
                <Trash2 className="h-3.5 w-3.5" />
              </button>
            </div>
          ))}
        </div>

        <Button variant="outline" size="sm" onClick={addLeaveType}>
          <Plus className="mr-1.5 h-4 w-4" />
          {t('settings.team.addLeaveType')}
        </Button>

        <Button onClick={handleSaveLeave} className="mt-3 ml-2" size="sm">
          <Save className="mr-1.5 h-4 w-4" />
          Speichern
        </Button>
      </section>

      <Separator className="mb-8" />

      {/* Work time */}
      <section>
        <div className="flex items-center gap-2 mb-4">
          <Clock className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">{t('settings.team.sectionWorktime')}</h3>
        </div>

        <div className="space-y-4">
          <div className="space-y-1.5">
            <Label>{t('settings.team.weeklyHours')}</Label>
            <Input
              type="number"
              min={1}
              max={60}
              value={workHours}
              onChange={(e) => setWorkHours(Number(e.target.value))}
              className="w-32"
            />
          </div>

          <div className="flex items-center justify-between rounded-lg border border-border bg-card p-4">
            <div>
              <p className="text-sm text-foreground">{t('settings.team.overtimeTracking')}</p>
              <p className="text-xs text-muted-foreground">{t('settings.team.overtimeTrackingDesc')}</p>
            </div>
            <Switch checked={overtimeEnabled} onCheckedChange={setOvertimeEnabled} />
          </div>
        </div>

        <Button onClick={handleSaveWorktime} className="mt-4" size="sm">
          <Save className="mr-1.5 h-4 w-4" />
          {t('settings.team.saveWorktime')}
        </Button>
      </section>

      <Separator className="mb-8" />

      {/* HR Settings (from API) */}
      <HRSettingsSection />

      {/* Confirm dialog for deletions */}
      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => { if (!open) setDeleteTarget(null) }}
        title={t('settings.team.deleteTitle')}
        description={t('settings.team.deleteDesc')}
        confirmLabel={t('common.delete')}
        variant="destructive"
        onConfirm={() => {
          if (!deleteTarget) return
          if (deleteTarget.type === 'dept') removeDepartment(deleteTarget.index)
          else if (deleteTarget.type === 'role') removeRole(deleteTarget.index)
          else removeLeave(deleteTarget.index)
        }}
      />
    </div>
  )
}

// ============================================================
// HR Settings Section (API-backed)
// ============================================================
function HRSettingsSection() {
  const { t } = useTranslation()
  const { data: hrSettings, isLoading } = useHRSettings()
  const updateMutation = useUpdateHRSettings()

  const [auThreshold, setAuThreshold] = useState(3)
  const [defaultLeaveDays, setDefaultLeaveDays] = useState(25)
  const [showAbsenceReason, setShowAbsenceReason] = useState(false)
  const [timezone, setTimezone] = useState('Europe/Zurich')

   
  useEffect(() => {
    if (!hrSettings) return
    // eslint-disable-next-line react-hooks/set-state-in-effect -- sync form fields from prop/API data
    setAuThreshold(hrSettings.auThresholdDays ?? 3)
    setDefaultLeaveDays(hrSettings.defaultAnnualLeaveDays ?? 25)
    setShowAbsenceReason(hrSettings.showAbsenceReason ?? false)
    setTimezone(hrSettings.timezone ?? 'Europe/Zurich')
  }, [hrSettings])

  const handleSave = () => {
    updateMutation.mutate({
      auThresholdDays: auThreshold,
      defaultAnnualLeaveDays: defaultLeaveDays,
      showAbsenceReason,
      timezone,
    })
  }

  if (isLoading) {
    return (
      <section>
        <div className="flex items-center gap-2 text-muted-foreground py-4">
          <Loader2 className="h-4 w-4 animate-spin" />
          <span className="text-sm">{t('settings.team.hrLoading')}</span>
        </div>
      </section>
    )
  }

  return (
    <section>
      <div className="flex items-center gap-2 mb-4">
        <Stethoscope className="h-4 w-4 text-muted-foreground" />
        <h3 className="text-sm font-medium text-foreground">{t('settings.team.sectionHR')}</h3>
      </div>

      <div className="space-y-4">
        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1.5">
            <Label>{t('settings.team.auThreshold')}</Label>
            <Input
              type="number"
              min={1}
              max={30}
              value={auThreshold}
              onChange={(e) => setAuThreshold(Number(e.target.value))}
              className="w-full"
            />
            <p className="text-[10px] text-muted-foreground">{t('settings.team.auThresholdDesc')}</p>
          </div>
          <div className="space-y-1.5">
            <Label>{t('settings.team.defaultLeaveDays')}</Label>
            <Input
              type="number"
              min={0}
              max={60}
              value={defaultLeaveDays}
              onChange={(e) => setDefaultLeaveDays(Number(e.target.value))}
              className="w-full"
            />
            <p className="text-[10px] text-muted-foreground">{t('settings.team.defaultLeaveDaysDesc')}</p>
          </div>
        </div>

        <div className="flex items-center justify-between rounded-lg border border-border bg-card p-3">
          <div>
            <p className="text-sm text-foreground">{t('settings.team.showAbsenceReason')}</p>
            <p className="text-xs text-muted-foreground">{t('settings.team.showAbsenceReasonDesc')}</p>
          </div>
          <Switch checked={showAbsenceReason} onCheckedChange={setShowAbsenceReason} />
        </div>

        <div className="space-y-1.5">
          <Label>{t('settings.team.timezone')}</Label>
          <select
            value={timezone}
            onChange={(e) => setTimezone(e.target.value)}
            className="w-full rounded-lg border border-border bg-input-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
          >
            <option value="Europe/Zurich">Europe/Zurich (GMT+1)</option>
            <option value="Europe/Berlin">Europe/Berlin (GMT+1)</option>
            <option value="Europe/Vienna">Europe/Vienna (GMT+1)</option>
          </select>
        </div>
      </div>

      <Button onClick={handleSave} className="mt-4" size="sm" disabled={updateMutation.isPending}>
        <Save className="mr-1.5 h-4 w-4" />
        {t('settings.team.saveHR')}
      </Button>
    </section>
  )
}

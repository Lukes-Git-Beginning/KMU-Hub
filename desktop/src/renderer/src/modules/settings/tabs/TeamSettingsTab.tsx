import { useState } from 'react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Separator } from '@/components/ui/separator'
import { Save, Plus, Trash2, Users, ShieldCheck, Clock, Palmtree } from 'lucide-react'
import { toast } from 'sonner'
import { ConfirmDialog } from '@/components/shared'
import { useSettingsStore, type LeaveType } from '@/stores/settings'

export function TeamSettingsTab() {
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
    setLeaveTypes([...leaveTypes, { name: 'Neu', days: 0, color: '#6b7280' }])
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
    toast.success('Abteilungen gespeichert')
  }

  const handleSaveRoles = () => {
    updateTeamAdmin({ roles })
    toast.success('Rollen gespeichert')
  }

  const handleSaveLeave = () => {
    updateTeamAdmin({ leaveTypes })
    toast.success('Abwesenheitsarten gespeichert')
  }

  const handleSaveWorktime = () => {
    updateTeamAdmin({ workHoursPerWeek: workHours, overtimeEnabled })
    toast.success('Arbeitszeitregelung gespeichert')
  }

  return (
    <div className="max-w-2xl">
      <h2 className="text-foreground mb-1">Team & HR</h2>
      <p className="text-sm text-muted-foreground mb-6">Abteilungen, Rollen, Abwesenheiten und Arbeitszeit verwalten</p>

      {/* Departments */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Users className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">Abteilungen</h3>
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
            placeholder="Neue Abteilung"
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
          Abteilungen speichern
        </Button>
      </section>

      <Separator className="mb-8" />

      {/* Roles */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <ShieldCheck className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">Rollen</h3>
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
            placeholder="Neue Rolle"
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
          Rollen speichern
        </Button>
      </section>

      <Separator className="mb-8" />

      {/* Leave types */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Palmtree className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">Abwesenheitsarten</h3>
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
                <span className="text-[10px] text-muted-foreground">Tage/Jahr</span>
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
          Abwesenheitsart hinzufuegen
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
          <h3 className="text-sm font-medium text-foreground">Arbeitszeitregelung</h3>
        </div>

        <div className="space-y-4">
          <div className="space-y-1.5">
            <Label>Wochenstunden</Label>
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
              <p className="text-sm text-foreground">Ueberstunden-Erfassung</p>
              <p className="text-xs text-muted-foreground">Mitarbeiter koennen Ueberstunden rapportieren</p>
            </div>
            <Switch checked={overtimeEnabled} onCheckedChange={setOvertimeEnabled} />
          </div>
        </div>

        <Button onClick={handleSaveWorktime} className="mt-4" size="sm">
          <Save className="mr-1.5 h-4 w-4" />
          Arbeitszeit speichern
        </Button>
      </section>

      {/* Confirm dialog for deletions */}
      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => { if (!open) setDeleteTarget(null) }}
        title="Eintrag loeschen?"
        description="Dieser Eintrag wird entfernt. Die Aenderung wird erst beim Speichern wirksam."
        confirmLabel="Loeschen"
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

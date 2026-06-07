/**
 * DATEV-Lohn / payroll-export configuration (tenant scope → "Für alle").
 * Connection (Berater-/Mandanten-Nr, target system, transfer) + wage-type and
 * absence-key mappings + payroll groups. Native inputs so the shell's
 * fieldset[disabled] locks them for non-leads.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Trash2 } from 'lucide-react'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  usePayrollSettingsStore,
  type PayrollTarget,
  type PayrollTransfer,
} from '@/stores/payrollSettings'

export function PayrollSettings() {
  const { t } = useTranslation()
  const s = usePayrollSettingsStore()
  const [newGroup, setNewGroup] = useState('')

  return (
    <div className="space-y-6">
      {/* Connection */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="space-y-1.5">
          <Label className="text-sm font-medium text-foreground">{t('team.payroll.beraterNr')}</Label>
          <Input value={s.beraterNr} onChange={(e) => s.setConnection({ beraterNr: e.target.value })} placeholder="1234567" />
          <p className="text-xs text-muted-foreground">{t('team.payroll.orderHint')}</p>
        </div>
        <div className="space-y-1.5">
          <Label className="text-sm font-medium text-foreground">{t('team.payroll.mandantNr')}</Label>
          <Input value={s.mandantNr} onChange={(e) => s.setConnection({ mandantNr: e.target.value })} placeholder="10001" />
        </div>
        <div className="space-y-1.5">
          <Label className="text-sm font-medium text-foreground">{t('team.payroll.target')}</Label>
          <Select value={s.target} onValueChange={(v) => s.setConnection({ target: v as PayrollTarget })}>
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="lug">{t('team.payroll.targetLug')}</SelectItem>
              <SelectItem value="lodas">{t('team.payroll.targetLodas')}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1.5">
          <Label className="text-sm font-medium text-foreground">{t('team.payroll.transfer')}</Label>
          <Select value={s.transfer} onValueChange={(v) => s.setConnection({ transfer: v as PayrollTransfer })}>
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="file">{t('team.payroll.transferFile')}</SelectItem>
              <SelectItem value="service">{t('team.payroll.transferService')}</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      {/* Wage-type mapping */}
      <div className="space-y-2">
        <div>
          <h4 className="text-sm font-medium text-foreground">{t('team.payroll.wageTypes')}</h4>
          <p className="text-xs text-muted-foreground">{t('team.payroll.wageTypesHint')}</p>
        </div>
        <div className="rounded-xl border border-border divide-y divide-border-muted">
          {s.wageTypes.map((w) => (
            <div key={w.id} className="flex items-center gap-3 px-3 py-2">
              <span className="flex-1 text-sm text-foreground">{w.label}</span>
              <Input
                value={w.datevNr}
                onChange={(e) => s.setWageType(w.id, e.target.value)}
                className="h-8 w-28"
                placeholder="Lohnart-Nr"
              />
            </div>
          ))}
        </div>
      </div>

      {/* Absence mapping */}
      <div className="space-y-2">
        <div>
          <h4 className="text-sm font-medium text-foreground">{t('team.payroll.absences')}</h4>
          <p className="text-xs text-muted-foreground">{t('team.payroll.absencesHint')}</p>
        </div>
        <div className="rounded-xl border border-border divide-y divide-border-muted">
          {s.absences.map((a) => (
            <div key={a.id} className="flex items-center gap-3 px-3 py-2">
              <div className="flex flex-1 items-center gap-2.5">
                <Checkbox
                  id={`abs-${a.id}`}
                  checked={a.exported}
                  onCheckedChange={(v) => s.setAbsence(a.id, { exported: v === true })}
                />
                <Label htmlFor={`abs-${a.id}`} className="text-sm text-foreground">{a.label}</Label>
              </div>
              <Input
                value={a.datevKey}
                onChange={(e) => s.setAbsence(a.id, { datevKey: e.target.value })}
                className="h-8 w-28"
                placeholder={t('team.payroll.absenceKey')}
                disabled={!a.exported}
              />
            </div>
          ))}
        </div>
      </div>

      {/* Payroll groups */}
      <div className="space-y-2">
        <div>
          <h4 className="text-sm font-medium text-foreground">{t('team.payroll.groups')}</h4>
          <p className="text-xs text-muted-foreground">{t('team.payroll.groupsHint')}</p>
        </div>
        <div className="flex flex-wrap gap-2">
          {s.groups.map((g) => (
            <span key={g.id} className="inline-flex items-center gap-1.5 rounded-full border border-border bg-secondary px-2.5 py-1 text-sm text-foreground">
              {g.name}
              <button
                onClick={() => s.removeGroup(g.id)}
                className="rounded-full p-0.5 text-muted-foreground transition-colors hover:text-destructive"
                aria-label={t('common.delete')}
              >
                <Trash2 className="h-3 w-3" />
              </button>
            </span>
          ))}
        </div>
        <div className="flex items-center gap-2">
          <Input
            value={newGroup}
            onChange={(e) => setNewGroup(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && newGroup.trim()) {
                s.addGroup(newGroup.trim())
                setNewGroup('')
              }
            }}
            className="h-9 max-w-xs"
            placeholder={t('team.payroll.newGroup')}
          />
          <button
            onClick={() => { if (newGroup.trim()) { s.addGroup(newGroup.trim()); setNewGroup('') } }}
            className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm text-foreground transition-colors hover:bg-secondary"
          >
            <Plus className="h-4 w-4" />
            {t('common.add')}
          </button>
        </div>
      </div>
    </div>
  )
}

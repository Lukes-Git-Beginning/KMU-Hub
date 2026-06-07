/**
 * Lohnvorbereitung — monthly payroll run (working surface).
 *
 * Cosmi prepares master + movement data and hands it to DATEV (vorbereitende
 * Lohnabrechnung). Flow: pick period + group → review per-employee change list
 * (master-data changes + movement data) → lock/approve → generate export →
 * history. MOCK-FIRST; real DATEV file generation = backend (see backend-gaps.md).
 */
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { CheckCircle2, Lock, Download, UserPlus, TrendingUp, Loader2, FileText } from 'lucide-react'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { EmptyState } from '@/components/shared'
import { useEmployees } from '@/api/hooks/hr-hooks'
import { usePayrollSettingsStore } from '@/stores/payrollSettings'
import { usePayrollRunsStore } from '@/stores/payrollRuns'
import { PayrollDeductionPreview } from './PayrollDeductionPreview'
import { formatDate } from '@/lib/format'

// Deterministic mock movement data per employee (stable across renders/QA).
function hashStr(s: string): number {
  let h = 0
  for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) | 0
  return Math.abs(h)
}

type StammChange = 'new' | 'salary' | 'leaver' | null

interface ChangeRow {
  id: string
  name: string
  department: string
  groupId: string
  stamm: StammChange
  hours: number
  overtime: number
  absenceDays: number
  oneTime: number
}

function buildRow(emp: { id: string; userName?: string; department?: string }, groupIds: string[]): ChangeRow {
  const h = hashStr(emp.id)
  const stamm: StammChange = h % 13 === 0 ? 'new' : h % 7 === 0 ? 'salary' : h % 17 === 0 ? 'leaver' : null
  return {
    id: emp.id,
    name: emp.userName ?? 'Unbekannt',
    department: emp.department ?? '',
    groupId: groupIds.length ? groupIds[h % groupIds.length] : 'all',
    stamm,
    hours: 150 + (h % 31),
    overtime: h % 8,
    absenceDays: (h >> 3) % 6,
    oneTime: h % 4 === 0 ? 200 + (h % 800) : 0,
  }
}

function lastMonths(count: number): { value: string; label: string }[] {
  const now = new Date()
  const out: { value: string; label: string }[] = []
  for (let i = 0; i < count; i++) {
    const d = new Date(now.getFullYear(), now.getMonth() - i, 1)
    const value = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`
    const label = d.toLocaleDateString('de-DE', { month: 'long', year: 'numeric' })
    out.push({ value, label })
  }
  return out
}

export function PayrollPrepPanel() {
  const { t } = useTranslation()
  const { data: employeesData, isLoading } = useEmployees()
  const groups = usePayrollSettingsStore((s) => s.groups)
  const target = usePayrollSettingsStore((s) => s.target)
  const runs = usePayrollRunsStore((s) => s.runs)
  const lock = usePayrollRunsStore((s) => s.lock)
  const unlock = usePayrollRunsStore((s) => s.unlock)
  const recordExport = usePayrollRunsStore((s) => s.recordExport)

  const months = useMemo(() => lastMonths(6), [])
  const [period, setPeriod] = useState(months[0].value)
  const [groupId, setGroupId] = useState(groups[0]?.id ?? 'all')
  const [exporting, setExporting] = useState(false)

  // Reactive boolean selector — subscribing to s.locked so the Export button
  // appears immediately after approval (selecting the isLocked fn would not).
  const locked = usePayrollRunsStore((s) => s.locked.includes(`${period}|${groupId}`))

  const groupIds = useMemo(() => groups.map((g) => g.id), [groups])
  const employees = useMemo(() => employeesData?.employees ?? [], [employeesData])

  const rows = useMemo(
    () => employees.map((e) => buildRow(e, groupIds)).filter((r) => r.groupId === groupId),
    [employees, groupIds, groupId],
  )

  const changeCount = rows.filter((r) => r.stamm !== null).length
  const groupName = groups.find((g) => g.id === groupId)?.name ?? ''
  const periodLabel = months.find((m) => m.value === period)?.label ?? period

  const stammBadge = (s: StammChange) => {
    if (s === 'new') return <Badge variant="secondary" className="gap-1 text-[10px]"><UserPlus className="h-2.5 w-2.5" />{t('team.payroll.run.new')}</Badge>
    if (s === 'salary') return <Badge variant="secondary" className="gap-1 text-[10px]"><TrendingUp className="h-2.5 w-2.5" />{t('team.payroll.run.salaryChange')}</Badge>
    if (s === 'leaver') return <Badge variant="outline" className="text-[10px]">{t('team.payroll.run.leaver')}</Badge>
    return <span className="text-xs text-muted-foreground">—</span>
  }

  const handleExport = () => {
    setExporting(true)
    setTimeout(() => {
      setExporting(false)
      recordExport({ period, group: groupName, employeeCount: rows.length, target })
      toast.success(t('team.payroll.run.exported', { count: rows.length }))
    }, 1200)
  }

  return (
    <div className="space-y-6">
      {/* Controls */}
      <div className="flex flex-wrap items-end gap-3">
        <div className="space-y-1.5">
          <label className="text-xs font-medium text-muted-foreground">{t('team.payroll.run.period')}</label>
          <Select value={period} onValueChange={setPeriod}>
            <SelectTrigger className="w-44"><SelectValue /></SelectTrigger>
            <SelectContent>
              {months.map((m) => <SelectItem key={m.value} value={m.value}>{m.label}</SelectItem>)}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1.5">
          <label className="text-xs font-medium text-muted-foreground">{t('team.payroll.run.group')}</label>
          <Select value={groupId} onValueChange={setGroupId}>
            <SelectTrigger className="w-44"><SelectValue /></SelectTrigger>
            <SelectContent>
              {groups.map((g) => <SelectItem key={g.id} value={g.id}>{g.name}</SelectItem>)}
            </SelectContent>
          </Select>
        </div>
        <div className="ml-auto flex items-center gap-2">
          {changeCount > 0 && (
            <Badge variant="outline" className="text-xs">{t('team.payroll.run.changes', { count: changeCount })}</Badge>
          )}
          {locked ? (
            <>
              <Badge variant="secondary" className="gap-1"><Lock className="h-3 w-3" />{t('team.payroll.run.locked')}</Badge>
              <Button variant="outline" size="sm" onClick={() => unlock(period, groupId)}>{t('team.payroll.run.unlock')}</Button>
              <Button size="sm" onClick={handleExport} disabled={exporting}>
                {exporting ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Download className="mr-2 h-4 w-4" />}
                {t('team.payroll.run.export', { target: target.toUpperCase() })}
              </Button>
            </>
          ) : (
            <Button size="sm" onClick={() => lock(period, groupId)} disabled={rows.length === 0}>
              <CheckCircle2 className="mr-2 h-4 w-4" />
              {t('team.payroll.run.approve')}
            </Button>
          )}
        </div>
      </div>

      {/* Change list */}
      {isLoading ? (
        <div className="flex items-center justify-center py-12"><Loader2 className="h-6 w-6 animate-spin text-primary" /></div>
      ) : rows.length === 0 ? (
        <EmptyState icon={FileText} title={t('team.payroll.run.emptyTitle')} description={t('team.payroll.run.emptyDesc')} />
      ) : (
        <div className="overflow-hidden rounded-xl border border-border bg-card">
          <div className="grid grid-cols-[1fr_140px_1fr_120px] gap-3 border-b border-border bg-secondary/40 px-4 py-2.5 text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
            <span>{t('team.payroll.run.employee')}</span>
            <span>{t('team.payroll.run.masterData')}</span>
            <span>{t('team.payroll.run.movementData')}</span>
            <span className="text-right">{t('team.payroll.run.oneTime')}</span>
          </div>
          {rows.map((r) => (
            <div key={r.id} className="grid grid-cols-[1fr_140px_1fr_120px] items-center gap-3 border-b border-border-muted px-4 py-3 last:border-b-0">
              <div className="min-w-0">
                <p className="truncate text-sm text-foreground">{r.name}</p>
                {r.department && <p className="text-[11px] text-muted-foreground">{r.department}</p>}
              </div>
              <div>{stammBadge(r.stamm)}</div>
              <div className="flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs text-muted-foreground">
                <span className="text-foreground">{t('team.payroll.run.hours', { count: r.hours })}</span>
                {r.overtime > 0 && <span className="text-warning">+{t('team.payroll.run.overtime', { count: r.overtime })}</span>}
                {r.absenceDays > 0 && <span>{t('team.payroll.run.absenceDays', { count: r.absenceDays })}</span>}
              </div>
              <div className="text-right text-sm tabular-nums text-foreground">
                {r.oneTime > 0 ? new Intl.NumberFormat('de-DE', { style: 'currency', currency: 'EUR', maximumFractionDigits: 0 }).format(r.oneTime) : '—'}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* History */}
      <div>
        <h3 className="mb-3 text-sm font-medium text-foreground">{t('team.payroll.run.history')}</h3>
        {runs.length === 0 ? (
          <p className="rounded-xl border border-dashed border-border py-6 text-center text-xs text-muted-foreground">
            {t('team.payroll.run.historyEmpty')}
          </p>
        ) : (
          <ul className="space-y-2">
            {runs.slice(0, 8).map((run) => (
              <li key={run.id} className="flex items-center gap-3 rounded-xl border border-border bg-card px-4 py-2.5 text-sm">
                <Download className="h-4 w-4 shrink-0 text-muted-foreground" />
                <span className="text-foreground">{months.find((m) => m.value === run.period)?.label ?? run.period}</span>
                <span className="text-muted-foreground">· {run.group}</span>
                <span className="text-muted-foreground">· {t('team.payroll.run.employees', { count: run.employeeCount })}</span>
                <Badge variant="outline" className="ml-auto text-[10px]">{run.target.toUpperCase()}</Badge>
                <span className="shrink-0 text-[11px] text-muted-foreground">{formatDate(run.exportedAt)}</span>
              </li>
            ))}
          </ul>
        )}
      </div>

      {/* Deduction preview (plausibility helper) */}
      <PayrollDeductionPreview />
    </div>
  )
}

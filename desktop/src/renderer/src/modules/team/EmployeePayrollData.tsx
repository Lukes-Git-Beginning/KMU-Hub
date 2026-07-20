import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ChevronDown, ChevronUp, CheckCircle2, AlertTriangle, Pencil, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { usePayrollMasterDataStore, isPayrollComplete } from '@/stores/payrollMasterData'
import { useHrScopedCapability } from './useTeamPermissions'
import type { PayrollMasterData } from '@/stores/payrollMasterData'
import { usePayrollSettingsStore } from '@/stores/payrollSettings'
import {
  GENDER_OPTIONS,
  MARITAL_STATUS_OPTIONS,
  TAX_CLASS_OPTIONS,
  CONFESSION_OPTIONS,
  SV_STATUS_OPTIONS,
  EMPLOYMENT_TYPE_OPTIONS,
  PAY_TYPE_OPTIONS,
} from '@/lib/payroll-enums'
import type { EnumOption } from '@/lib/payroll-enums'

/**
 * Lohn-Stammdaten (payroll master data) section for the employee detail panel.
 *
 * R-4: scope-aware gate via useHrScopedCapability (own = eigenes Gehalt sehen,
 * team/all = fremde Gehaltsdaten). Prop targetUserId is the auth userId of the
 * employee whose record is shown (null while loading → pessimistic deny).
 *
 * hr_only: visible to admin/hr roles only (besondere Personaldaten, DSGVO).
 * Collapsible, view + inline-edit, grouped by DATEV topic areas. Feeds the
 * Lohnvorbereitung. MOCK-FIRST overlay (payrollMasterData store) — see spec.
 */
export function EmployeePayrollData({
  employeeId,
  targetUserId = null,
  embedded = false,
}: {
  employeeId: string
  /**
   * Auth userId of the employee being viewed.
   * Used by useHrScopedCapability to resolve own/team/all scope.
   * Pass null while the employee record is still loading.
   */
  targetUserId?: string | null
  /** When true, render content always-open without the collapse toggle. */
  embedded?: boolean
}) {
  const { t } = useTranslation()
  const stored = usePayrollMasterDataStore((s) => s.data[employeeId])
  const save = usePayrollMasterDataStore((s) => s.set)
  const groups = usePayrollSettingsStore((s) => s.groups)

  // R-4: scope-aware salary gate — own = eigenes Gehalt (member-Standard),
  // team/all = fremde Gehaltsdaten (nur hr_admin/admin).
  const canView = useHrScopedCapability('team:salary:view', targetUserId)
  const canEdit = useHrScopedCapability('team:salary:edit', targetUserId)

  const [expanded, setExpanded] = useState(false)
  const showContent = embedded || expanded
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState<PayrollMasterData>({})

  if (!canView) return null

  const complete = isPayrollComplete(stored)

  const startEdit = () => {
    setDraft(stored ?? {})
    setEditing(true)
  }
  const cancelEdit = () => {
    setEditing(false)
    setDraft({})
  }
  const commit = () => {
    save(employeeId, draft)
    setEditing(false)
  }

  const patch = (p: Partial<PayrollMasterData>) => setDraft((d) => ({ ...d, ...p }))

  const Chevron = expanded ? ChevronUp : ChevronDown

  return (
    <section className="space-y-2">
      <button
        type="button"
        onClick={() => { if (!embedded) setExpanded((v) => !v) }}
        className={`flex w-full items-center gap-2 text-left ${embedded ? 'cursor-default' : ''}`}
        aria-expanded={showContent}
      >
        <h4 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          {t('team.payroll.masterData.title')}
        </h4>
        {complete ? (
          <CheckCircle2 className="h-3.5 w-3.5 text-success" aria-label={t('team.payroll.masterData.complete')} />
        ) : (
          <span className="flex items-center gap-1 rounded-full bg-warning/10 px-1.5 py-0.5 text-[10px] font-medium text-warning-foreground">
            <AlertTriangle className="h-2.5 w-2.5" aria-hidden="true" />
            {t('team.payroll.masterData.incomplete')}
          </span>
        )}
        {!embedded && <Chevron className="ml-auto h-3 w-3 text-muted-foreground" aria-hidden="true" />}
      </button>

      {showContent && (
        <div className="space-y-3">
          {/* Edit / Save toolbar — only with team:salary:edit */}
          {canEdit && (
            <div className="flex items-center justify-end gap-2">
              {editing ? (
                <>
                  <Button variant="outline" size="sm" className="h-7 text-xs" onClick={cancelEdit}>
                    <X className="mr-1 h-3 w-3" />
                    {t('common.cancel')}
                  </Button>
                  <Button size="sm" className="h-7 text-xs" onClick={commit}>
                    {t('common.save')}
                  </Button>
                </>
              ) : (
                <Button variant="outline" size="sm" className="h-7 text-xs" onClick={startEdit}>
                  <Pencil className="mr-1 h-3 w-3" />
                  {t('common.edit')}
                </Button>
              )}
            </div>
          )}

          {/* DSGVO note */}
          <p className="rounded-md bg-secondary/40 px-2.5 py-1.5 text-[10px] leading-relaxed text-muted-foreground">
            {t('team.payroll.masterData.dsgvoHint')}
          </p>

          {/* 1. Persönlich */}
          <Group label={t('team.payroll.masterData.groups.personal')}>
            <Field label={t('team.payroll.masterData.fields.birthDate')} required>
              {editing ? (
                <DateInput value={draft.birthDate} onChange={(v) => patch({ birthDate: v })} />
              ) : (
                <View value={stored?.birthDate} />
              )}
            </Field>
            <Field label={t('team.payroll.masterData.fields.birthPlace')}>
              {editing ? (
                <TextInput value={draft.birthPlace} onChange={(v) => patch({ birthPlace: v })} />
              ) : (
                <View value={stored?.birthPlace} />
              )}
            </Field>
            <Field label={t('team.payroll.masterData.fields.gender')} required>
              {editing ? (
                <SelectInput value={draft.gender} options={GENDER_OPTIONS} onChange={(v) => patch({ gender: v })} />
              ) : (
                <ViewEnum value={stored?.gender} options={GENDER_OPTIONS} />
              )}
            </Field>
            <Field label={t('team.payroll.masterData.fields.nationality')} required>
              {editing ? (
                <TextInput value={draft.nationality} onChange={(v) => patch({ nationality: v })} />
              ) : (
                <View value={stored?.nationality} />
              )}
            </Field>
            <Field label={t('team.payroll.masterData.fields.maritalStatus')} required>
              {editing ? (
                <SelectInput value={draft.maritalStatus} options={MARITAL_STATUS_OPTIONS} onChange={(v) => patch({ maritalStatus: v })} />
              ) : (
                <ViewEnum value={stored?.maritalStatus} options={MARITAL_STATUS_OPTIONS} />
              )}
            </Field>
          </Group>

          {/* 2. Steuer */}
          <Group label={t('team.payroll.masterData.groups.tax')}>
            <Field label={t('team.payroll.masterData.fields.taxId')} required>
              {editing ? (
                <TextInput value={draft.taxId} onChange={(v) => patch({ taxId: v })} placeholder="00 000 000 000" />
              ) : (
                <View value={stored?.taxId} />
              )}
            </Field>
            <Field label={t('team.payroll.masterData.fields.taxClass')}>
              {editing ? (
                <SelectInput value={draft.taxClass} options={TAX_CLASS_OPTIONS} onChange={(v) => patch({ taxClass: v })} />
              ) : (
                <ViewEnum value={stored?.taxClass} options={TAX_CLASS_OPTIONS} />
              )}
            </Field>
            <Field label={t('team.payroll.masterData.fields.childAllowances')}>
              {editing ? (
                <NumberInput value={draft.childAllowances} step={0.5} onChange={(v) => patch({ childAllowances: v })} />
              ) : (
                <View value={stored?.childAllowances?.toString()} />
              )}
            </Field>
            <Field label={t('team.payroll.masterData.fields.confession')}>
              {editing ? (
                <SelectInput value={draft.confession} options={CONFESSION_OPTIONS} onChange={(v) => patch({ confession: v })} />
              ) : (
                <ViewEnum value={stored?.confession} options={CONFESSION_OPTIONS} />
              )}
            </Field>
          </Group>

          {/* 3. Sozialversicherung */}
          <Group label={t('team.payroll.masterData.groups.socialSecurity')}>
            <Field label={t('team.payroll.masterData.fields.svNumber')} required>
              {editing ? (
                <TextInput value={draft.svNumber} onChange={(v) => patch({ svNumber: v })} />
              ) : (
                <View value={stored?.svNumber} />
              )}
            </Field>
            <Field label={t('team.payroll.masterData.fields.healthInsurance')} required>
              {editing ? (
                <TextInput value={draft.healthInsurance} onChange={(v) => patch({ healthInsurance: v })} />
              ) : (
                <View value={stored?.healthInsurance} />
              )}
            </Field>
            <Field label={t('team.payroll.masterData.fields.healthInsuranceNr')}>
              {editing ? (
                <TextInput value={draft.healthInsuranceNr} onChange={(v) => patch({ healthInsuranceNr: v })} />
              ) : (
                <View value={stored?.healthInsuranceNr} />
              )}
            </Field>
            <Field label={t('team.payroll.masterData.fields.svStatus')}>
              {editing ? (
                <SelectInput value={draft.svStatus} options={SV_STATUS_OPTIONS} onChange={(v) => patch({ svStatus: v })} />
              ) : (
                <ViewEnum value={stored?.svStatus} options={SV_STATUS_OPTIONS} />
              )}
            </Field>
            <Field label={t('team.payroll.masterData.fields.parentProperty')}>
              {editing ? (
                <BoolToggle value={draft.parentProperty} onChange={(v) => patch({ parentProperty: v })} />
              ) : (
                <View value={stored?.parentProperty === undefined ? undefined : t(stored.parentProperty ? 'common.yes' : 'common.no')} />
              )}
            </Field>
          </Group>

          {/* 4. Beschäftigung */}
          <Group label={t('team.payroll.masterData.groups.employment')}>
            <Field label={t('team.payroll.masterData.fields.weeklyHours')} required>
              {editing ? (
                <NumberInput value={draft.weeklyHours} step={0.5} onChange={(v) => patch({ weeklyHours: v })} />
              ) : (
                <View value={stored?.weeklyHours?.toString()} />
              )}
            </Field>
            <Field label={t('team.payroll.masterData.fields.employmentType')} required>
              {editing ? (
                <SelectInput value={draft.employmentType} options={EMPLOYMENT_TYPE_OPTIONS} onChange={(v) => patch({ employmentType: v })} />
              ) : (
                <ViewEnum value={stored?.employmentType} options={EMPLOYMENT_TYPE_OPTIONS} />
              )}
            </Field>
            <Field label={t('team.payroll.masterData.fields.jobKey')} required>
              {editing ? (
                <TextInput value={draft.jobKey} onChange={(v) => patch({ jobKey: v })} placeholder="000000000" />
              ) : (
                <View value={stored?.jobKey} />
              )}
            </Field>
            <Field label={t('team.payroll.masterData.fields.endDate')}>
              {editing ? (
                <DateInput value={draft.endDate} onChange={(v) => patch({ endDate: v })} />
              ) : (
                <View value={stored?.endDate} />
              )}
            </Field>
          </Group>

          {/* 5. Bezüge & Bank */}
          <Group label={t('team.payroll.masterData.groups.compensation')}>
            <Field label={t('team.payroll.masterData.fields.payType')} required>
              {editing ? (
                <SelectInput value={draft.payType} options={PAY_TYPE_OPTIONS} onChange={(v) => patch({ payType: v })} />
              ) : (
                <ViewEnum value={stored?.payType} options={PAY_TYPE_OPTIONS} />
              )}
            </Field>
            {(editing ? draft.payType : stored?.payType) === 'hourly' ? (
              <Field label={t('team.payroll.masterData.fields.hourlyWage')} required>
                {editing ? (
                  <NumberInput value={draft.hourlyWage} step={0.01} onChange={(v) => patch({ hourlyWage: v })} suffix="€/h" />
                ) : (
                  <View value={stored?.hourlyWage !== undefined ? formatEur(stored.hourlyWage) : undefined} />
                )}
              </Field>
            ) : (
              <Field label={t('team.payroll.masterData.fields.monthlySalary')} required>
                {editing ? (
                  <NumberInput value={draft.monthlySalary} step={1} onChange={(v) => patch({ monthlySalary: v })} suffix="€" />
                ) : (
                  <View value={stored?.monthlySalary !== undefined ? formatEur(stored.monthlySalary) : undefined} />
                )}
              </Field>
            )}
            <Field label={t('team.payroll.masterData.fields.specialPayments')}>
              {editing ? (
                <TextInput value={draft.specialPayments} onChange={(v) => patch({ specialPayments: v })} />
              ) : (
                <View value={stored?.specialPayments} />
              )}
            </Field>
            <Field label={t('team.payroll.masterData.fields.payrollGroup')}>
              {editing ? (
                <select
                  value={draft.payrollGroupId ?? ''}
                  onChange={(e) => patch({ payrollGroupId: e.target.value || undefined })}
                  className={selectClass}
                >
                  <option value="">{t('team.member.selectPlaceholder')}</option>
                  {groups.map((g) => (
                    <option key={g.id} value={g.id}>{g.name}</option>
                  ))}
                </select>
              ) : (
                <View value={groups.find((g) => g.id === stored?.payrollGroupId)?.name} />
              )}
            </Field>
            <Field label={t('team.payroll.masterData.fields.iban')} required>
              {editing ? (
                <TextInput value={draft.iban} onChange={(v) => patch({ iban: v })} placeholder="DE00 0000 0000 0000 0000 00" />
              ) : (
                <View value={stored?.iban} />
              )}
            </Field>
            <Field label={t('team.payroll.masterData.fields.bic')}>
              {editing ? (
                <TextInput value={draft.bic} onChange={(v) => patch({ bic: v })} />
              ) : (
                <View value={stored?.bic} />
              )}
            </Field>
            <Field label={t('team.payroll.masterData.fields.accountHolder')}>
              {editing ? (
                <TextInput value={draft.accountHolder} onChange={(v) => patch({ accountHolder: v })} />
              ) : (
                <View value={stored?.accountHolder} />
              )}
            </Field>
          </Group>
        </div>
      )}
    </section>
  )
}

// ---------------------------------------------------------------------------
// Layout + field primitives
// ---------------------------------------------------------------------------

const selectClass =
  'w-full rounded border border-border bg-input-background px-2 py-1.5 text-xs text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring'

function formatEur(n: number): string {
  return new Intl.NumberFormat('de-DE', { style: 'currency', currency: 'EUR' }).format(n)
}

function Group({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="space-y-1.5 rounded-lg border border-border-muted bg-secondary/20 p-2.5">
      <p className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">{label}</p>
      <div className="space-y-1.5">{children}</div>
    </div>
  )
}

function Field({ label, required, children }: { label: string; required?: boolean; children: React.ReactNode }) {
  return (
    <div className="grid grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)] items-center gap-2">
      <Label className="text-xs text-muted-foreground">
        {label}
        {required && <span className="ml-0.5 text-warning-foreground">*</span>}
      </Label>
      <div className="min-w-0">{children}</div>
    </div>
  )
}

function View({ value }: { value?: string }) {
  return <span className="block truncate text-sm text-foreground">{value && value.length > 0 ? value : '—'}</span>
}

function ViewEnum<T extends string>({ value, options }: { value?: T; options: EnumOption<T>[] }) {
  const { t } = useTranslation()
  const opt = options.find((o) => o.value === value)
  return <View value={opt ? t(opt.labelKey) : undefined} />
}

function TextInput({
  value,
  onChange,
  placeholder,
}: {
  value?: string
  onChange: (v: string) => void
  placeholder?: string
}) {
  return (
    <Input
      value={value ?? ''}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder}
      className="h-7 text-xs"
    />
  )
}

function NumberInput({
  value,
  onChange,
  step,
  suffix,
}: {
  value?: number
  onChange: (v: number | undefined) => void
  step?: number
  suffix?: string
}) {
  return (
    <div className="flex items-center gap-1">
      <Input
        type="number"
        step={step}
        value={value ?? ''}
        onChange={(e) => onChange(e.target.value === '' ? undefined : Number(e.target.value))}
        className="h-7 text-xs"
      />
      {suffix && <span className="shrink-0 text-[10px] text-muted-foreground">{suffix}</span>}
    </div>
  )
}

function DateInput({ value, onChange }: { value?: string; onChange: (v: string) => void }) {
  return (
    <Input
      type="date"
      value={value ?? ''}
      onChange={(e) => onChange(e.target.value)}
      className="h-7 text-xs"
    />
  )
}

function SelectInput<T extends string>({
  value,
  options,
  onChange,
}: {
  value?: T
  options: EnumOption<T>[]
  onChange: (v: T) => void
}) {
  const { t } = useTranslation()
  return (
    <select value={value ?? ''} onChange={(e) => onChange(e.target.value as T)} className={selectClass}>
      <option value="">{t('team.member.selectPlaceholder')}</option>
      {options.map((o) => (
        <option key={o.value} value={o.value}>
          {t(o.labelKey)}
        </option>
      ))}
    </select>
  )
}

function BoolToggle({ value, onChange }: { value?: boolean; onChange: (v: boolean) => void }) {
  const { t } = useTranslation()
  return (
    <select
      value={value === undefined ? '' : value ? 'yes' : 'no'}
      onChange={(e) => onChange(e.target.value === 'yes')}
      className={selectClass}
    >
      <option value="">{t('team.member.selectPlaceholder')}</option>
      <option value="yes">{t('common.yes')}</option>
      <option value="no">{t('common.no')}</option>
    </select>
  )
}

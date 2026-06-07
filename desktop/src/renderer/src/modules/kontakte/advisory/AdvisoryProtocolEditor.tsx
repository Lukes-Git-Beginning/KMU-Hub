/**
 * Beratungsprotokoll — full-screen editor (8 sections).
 *
 * Reached via /kontakte/protokoll/:contactId/:protocolId. The protocol draft is
 * created by the caller (history tab) before navigation, so this view always
 * loads an existing record. A finalized protocol renders read-only (immutable
 * legal documentation).
 */
import { useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  ArrowLeft,
  Save,
  Lock,
  CheckCircle2,
  Plus,
  Trash2,
  ShieldCheck,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Checkbox } from '@/components/ui/checkbox'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { ConfirmDialog } from '@/components/shared'
import { useContacts } from '@/api/hooks/useContacts'
import { useAdvisoryProtocolsStore } from '@/stores/advisoryProtocols'
import {
  ADVISORY_LOCATIONS,
  ADVISORY_OCCASIONS,
  ADVISORY_WARNINGS,
  ASSET_CLASSES,
  CUSTOMER_CATEGORIES,
  DELIVERY_FORMS,
  INVESTMENT_HORIZONS,
  INVESTMENT_PURPOSES,
  SRI_CLASSES,
  protocolDurationMin,
  sriColor,
  type AdvisoryProduct,
  type AdvisoryProtocol,
  type EnumOption,
  type SriClass,
} from '@/lib/advisory'
import { cn } from '@/lib/cn'

const SECTIONS = [
  { id: 's1', labelKey: 'advisory.section.head' },
  { id: 's2', labelKey: 'advisory.section.customer' },
  { id: 's3', labelKey: 'advisory.section.knowledge' },
  { id: 's4', labelKey: 'advisory.section.financial' },
  { id: 's5', labelKey: 'advisory.section.goals' },
  { id: 's6', labelKey: 'advisory.section.products' },
  { id: 's7', labelKey: 'advisory.section.recommendation' },
  { id: 's8', labelKey: 'advisory.section.compliance' },
] as const

// ---------------------------------------------------------------------------
// Small field helpers (kept local — only this editor needs them)
// ---------------------------------------------------------------------------

function Field({
  label,
  required,
  hint,
  children,
}: {
  label: string
  required?: boolean
  hint?: string
  children: React.ReactNode
}) {
  return (
    <div className="space-y-1.5">
      <Label className="text-sm font-medium text-foreground">
        {label}
        {required && <span className="ml-0.5 text-destructive">*</span>}
      </Label>
      {children}
      {hint && <p className="text-xs text-muted-foreground">{hint}</p>}
    </div>
  )
}

function SelectField<T extends string>({
  value,
  onChange,
  options,
  placeholder,
  disabled,
}: {
  value: T
  onChange: (v: T) => void
  options: EnumOption<T>[]
  placeholder?: string
  disabled?: boolean
}) {
  const { t } = useTranslation()
  return (
    <Select value={value || undefined} onValueChange={(v) => onChange(v as T)} disabled={disabled}>
      <SelectTrigger>
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent>
        {options.map((o) => (
          <SelectItem key={o.value} value={o.value}>
            {t(o.labelKey)}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

function ChipMulti<T extends string>({
  selected,
  onToggle,
  options,
  disabled,
}: {
  selected: T[]
  onToggle: (v: T) => void
  options: EnumOption<T>[]
  disabled?: boolean
}) {
  const { t } = useTranslation()
  return (
    <div className="flex flex-wrap gap-2">
      {options.map((o) => {
        const active = selected.includes(o.value)
        return (
          <button
            key={o.value}
            type="button"
            disabled={disabled}
            onClick={() => onToggle(o.value)}
            className={cn(
              'rounded-full border px-3 py-1 text-xs font-medium transition-colors disabled:opacity-50',
              active
                ? 'border-primary bg-primary/10 text-primary'
                : 'border-border text-muted-foreground hover:border-primary/40 hover:text-foreground',
            )}
          >
            {t(o.labelKey)}
          </button>
        )
      })}
    </div>
  )
}

function SectionCard({
  id,
  index,
  title,
  children,
}: {
  id: string
  index: number
  title: string
  children: React.ReactNode
}) {
  return (
    <section id={id} className="scroll-mt-24 rounded-2xl border border-border bg-card p-6">
      <h3 className="mb-5 flex items-center gap-2.5 text-sm font-semibold text-foreground">
        <span className="flex h-6 w-6 items-center justify-center rounded-full bg-primary/10 text-xs font-semibold text-primary">
          {index}
        </span>
        {title}
      </h3>
      {children}
    </section>
  )
}

// ---------------------------------------------------------------------------
// Editor
// ---------------------------------------------------------------------------

export default function AdvisoryProtocolEditor() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { contactId = '', protocolId = '' } = useParams()

  const stored = useAdvisoryProtocolsStore((s) => s.get(protocolId))
  const updateStore = useAdvisoryProtocolsStore((s) => s.update)
  const finalizeStore = useAdvisoryProtocolsStore((s) => s.finalize)

  const { data: contactsData } = useContacts({ page_size: 200 })
  const contactName = useMemo(() => {
    const c = (contactsData?.contacts ?? []).find((x) => x.id === contactId)
    return c ? `${c.firstName ?? ''} ${c.lastName ?? ''}`.trim() : ''
  }, [contactsData, contactId])

  const [form, setForm] = useState<AdvisoryProtocol | null>(stored ?? null)
  const [activeSection, setActiveSection] = useState('s1')
  const [confirmFinalize, setConfirmFinalize] = useState(false)

  const readOnly = form?.status === 'finalized'

  const duration = useMemo(
    () => (form ? protocolDurationMin(form.timeFrom, form.timeTo) : null),
    [form],
  )

  const back = () => navigate(`/kontakte?contact=${contactId}`)

  if (!form) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-4 p-8 text-center">
        <p className="text-sm text-muted-foreground">{t('advisory.notFound')}</p>
        <Button variant="outline" onClick={() => navigate('/kontakte')}>
          <ArrowLeft className="mr-2 h-4 w-4" />
          {t('common.back')}
        </Button>
      </div>
    )
  }

  const set = <K extends keyof AdvisoryProtocol>(key: K, value: AdvisoryProtocol[K]) =>
    setForm((f) => (f ? { ...f, [key]: value } : f))

  const num = (v: string): number | null => (v === '' ? null : Number(v))

  const handleSave = () => {
    updateStore(form.id, form)
    toast.success(t('advisory.saved'))
  }

  const handleFinalize = () => {
    updateStore(form.id, form)
    finalizeStore(form.id)
    setForm((f) => (f ? { ...f, status: 'finalized' } : f))
    setConfirmFinalize(false)
    toast.success(t('advisory.finalized'))
  }

  const scrollTo = (id: string) => {
    setActiveSection(id)
    document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }

  // Product line helpers
  const addProduct = () =>
    set('products', [
      ...form.products,
      { id: crypto.randomUUID(), name: '', riskClass: 3, risks: '', recommended: false },
    ])
  const updateProduct = (id: string, patch: Partial<AdvisoryProduct>) =>
    set(
      'products',
      form.products.map((p) => (p.id === id ? { ...p, ...patch } : p)),
    )
  const removeProduct = (id: string) =>
    set(
      'products',
      form.products.filter((p) => p.id !== id),
    )

  return (
    <div className="flex h-full flex-col bg-background animate-fade-in">
      {/* Sticky header */}
      <header className="sticky top-0 z-10 flex items-center gap-3 border-b border-border bg-card/95 px-6 py-3 backdrop-blur">
        <button
          onClick={back}
          className="flex items-center gap-1.5 rounded-md px-2 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
        >
          <ArrowLeft className="h-4 w-4" />
          {t('common.back')}
        </button>
        <div className="ml-1 min-w-0">
          <h2 className="truncate text-base font-semibold leading-tight text-foreground">
            {t('advisory.title')}
          </h2>
          {contactName && (
            <p className="truncate text-xs text-muted-foreground">{contactName}</p>
          )}
        </div>
        {readOnly ? (
          <Badge variant="secondary" className="ml-2 gap-1">
            <Lock className="h-3 w-3" />
            {t('advisory.status.finalized')}
          </Badge>
        ) : (
          <Badge variant="outline" className="ml-2">
            {t('advisory.status.draft')}
          </Badge>
        )}
        <div className="ml-auto flex items-center gap-2">
          {!readOnly && (
            <>
              <Button variant="outline" size="sm" onClick={handleSave}>
                <Save className="mr-2 h-4 w-4" />
                {t('common.save')}
              </Button>
              <Button size="sm" onClick={() => setConfirmFinalize(true)}>
                <CheckCircle2 className="mr-2 h-4 w-4" />
                {t('advisory.finalize')}
              </Button>
            </>
          )}
        </div>
      </header>

      <div className="flex flex-1 overflow-hidden">
        {/* Section nav */}
        <nav className="hidden w-56 shrink-0 overflow-y-auto border-r border-border p-4 lg:block">
          <ul className="space-y-1">
            {SECTIONS.map((s, i) => (
              <li key={s.id}>
                <button
                  onClick={() => scrollTo(s.id)}
                  className={cn(
                    'flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-left text-sm transition-colors',
                    activeSection === s.id
                      ? 'bg-primary/10 font-medium text-primary'
                      : 'text-muted-foreground hover:bg-accent hover:text-foreground',
                  )}
                >
                  <span className="text-xs tabular-nums opacity-60">{i + 1}</span>
                  <span className="truncate">{t(s.labelKey)}</span>
                </button>
              </li>
            ))}
          </ul>
          {readOnly && (
            <p className="mt-4 rounded-lg bg-secondary px-3 py-2 text-xs text-muted-foreground">
              {t('advisory.readOnlyHint')}
            </p>
          )}
        </nav>

        {/* Form body */}
        <div
          className="flex-1 space-y-5 overflow-y-auto p-6"
          onScroll={(e) => {
            // Lightweight scroll-spy: pick the section nearest the top.
            const container = e.currentTarget
            let current: string = SECTIONS[0].id
            for (const s of SECTIONS) {
              const el = document.getElementById(s.id)
              if (el && el.offsetTop - container.scrollTop <= 120) current = s.id
            }
            setActiveSection(current)
          }}
        >
          {/* Section 1 — Gesprächskopf */}
          <SectionCard id="s1" index={1} title={t('advisory.section.head')}>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
              <Field label={t('advisory.field.date')} required>
                <Input type="date" value={form.date} disabled={readOnly} onChange={(e) => set('date', e.target.value)} />
              </Field>
              <Field label={t('advisory.field.timeFrom')} required>
                <Input type="time" value={form.timeFrom} disabled={readOnly} onChange={(e) => set('timeFrom', e.target.value)} />
              </Field>
              <Field label={t('advisory.field.timeTo')} required>
                <Input type="time" value={form.timeTo} disabled={readOnly} onChange={(e) => set('timeTo', e.target.value)} />
              </Field>
              <Field label={t('advisory.field.duration')} hint={t('advisory.field.durationHint')}>
                <Input value={duration != null ? t('advisory.minutes', { count: duration }) : '—'} disabled readOnly />
              </Field>
              <Field label={t('advisory.field.location')} required>
                <SelectField value={form.location} onChange={(v) => set('location', v)} options={ADVISORY_LOCATIONS} disabled={readOnly} />
              </Field>
              <Field label={t('advisory.field.advisor')} required>
                <Input value={form.advisor} disabled={readOnly} onChange={(e) => set('advisor', e.target.value)} />
              </Field>
              <Field label={t('advisory.field.occasion')} required>
                <SelectField value={form.occasion} onChange={(v) => set('occasion', v)} options={ADVISORY_OCCASIONS} disabled={readOnly} />
              </Field>
              <Field label={t('advisory.field.occasionNote')}>
                <Input value={form.occasionNote} disabled={readOnly} onChange={(e) => set('occasionNote', e.target.value)} />
              </Field>
              <Field label={t('advisory.field.customerCategory')} required>
                <SelectField value={form.customerCategory} onChange={(v) => set('customerCategory', v)} options={CUSTOMER_CATEGORIES} disabled={readOnly} />
              </Field>
            </div>
          </SectionCard>

          {/* Section 2 — Kunde & Profil */}
          <SectionCard id="s2" index={2} title={t('advisory.section.customer')}>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <Field label={t('advisory.field.birthDate')}>
                <Input type="date" value={form.birthDate} disabled={readOnly} onChange={(e) => set('birthDate', e.target.value)} />
              </Field>
              <Field label={t('advisory.field.maritalStatus')}>
                <Input value={form.maritalStatus} disabled={readOnly} onChange={(e) => set('maritalStatus', e.target.value)} />
              </Field>
              <Field label={t('advisory.field.taxStatus')}>
                <Input value={form.taxStatus} disabled={readOnly} onChange={(e) => set('taxStatus', e.target.value)} />
              </Field>
            </div>
          </SectionCard>

          {/* Section 3 — Kenntnisse & Erfahrungen */}
          <SectionCard id="s3" index={3} title={t('advisory.section.knowledge')}>
            <div className="space-y-4">
              <Field label={t('advisory.field.knownAssetClasses')} required>
                <ChipMulti
                  selected={form.knownAssetClasses}
                  onToggle={(v) =>
                    set(
                      'knownAssetClasses',
                      form.knownAssetClasses.includes(v)
                        ? form.knownAssetClasses.filter((x) => x !== v)
                        : [...form.knownAssetClasses, v],
                    )
                  }
                  options={ASSET_CLASSES}
                  disabled={readOnly}
                />
              </Field>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <Field label={t('advisory.field.pastTransactions')} required hint={t('advisory.field.pastTransactionsHint')}>
                  <Textarea rows={2} value={form.pastTransactions} disabled={readOnly} onChange={(e) => set('pastTransactions', e.target.value)} />
                </Field>
                <Field label={t('advisory.field.financialEducation')} required>
                  <Textarea rows={2} value={form.financialEducation} disabled={readOnly} onChange={(e) => set('financialEducation', e.target.value)} />
                </Field>
                <Field label={t('advisory.field.professionalExperience')} required>
                  <Textarea rows={2} value={form.professionalExperience} disabled={readOnly} onChange={(e) => set('professionalExperience', e.target.value)} />
                </Field>
                <Field label={t('advisory.field.selfAssessment')} hint={t('advisory.field.selfAssessmentHint')}>
                  <Input
                    type="number"
                    min={1}
                    max={5}
                    value={form.selfAssessment ?? ''}
                    disabled={readOnly}
                    onChange={(e) => set('selfAssessment', num(e.target.value))}
                  />
                </Field>
              </div>
            </div>
          </SectionCard>

          {/* Section 4 — Finanzielle Situation */}
          <SectionCard id="s4" index={4} title={t('advisory.section.financial')}>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
              <Field label={t('advisory.field.monthlyNetIncome')} required>
                <Input type="number" value={form.monthlyNetIncome ?? ''} disabled={readOnly} onChange={(e) => set('monthlyNetIncome', num(e.target.value))} />
              </Field>
              <Field label={t('advisory.field.recurringLiabilities')} required>
                <Input type="number" value={form.recurringLiabilities ?? ''} disabled={readOnly} onChange={(e) => set('recurringLiabilities', num(e.target.value))} />
              </Field>
              <Field label={t('advisory.field.liquidAssets')} required>
                <Input type="number" value={form.liquidAssets ?? ''} disabled={readOnly} onChange={(e) => set('liquidAssets', num(e.target.value))} />
              </Field>
              <Field label={t('advisory.field.currentInvestments')} required>
                <Input type="number" value={form.currentInvestments ?? ''} disabled={readOnly} onChange={(e) => set('currentInvestments', num(e.target.value))} />
              </Field>
              <Field label={t('advisory.field.realEstate')}>
                <Input value={form.realEstate} disabled={readOnly} onChange={(e) => set('realEstate', e.target.value)} />
              </Field>
              <Field label={t('advisory.field.existingInsurance')}>
                <Input value={form.existingInsurance} disabled={readOnly} onChange={(e) => set('existingInsurance', e.target.value)} />
              </Field>
              <Field label={t('advisory.field.maxLossAbs')} required>
                <Input type="number" value={form.maxLossCapacityAbs ?? ''} disabled={readOnly} onChange={(e) => set('maxLossCapacityAbs', num(e.target.value))} />
              </Field>
              <Field label={t('advisory.field.maxLossPct')} required>
                <Input type="number" value={form.maxLossCapacityPct ?? ''} disabled={readOnly} onChange={(e) => set('maxLossCapacityPct', num(e.target.value))} />
              </Field>
            </div>
          </SectionCard>

          {/* Section 5 — Anlageziele & Risikoprofil */}
          <SectionCard id="s5" index={5} title={t('advisory.section.goals')}>
            <div className="space-y-4">
              <Field label={t('advisory.field.investmentPurpose')} required>
                <ChipMulti
                  selected={form.investmentPurpose}
                  onToggle={(v) =>
                    set(
                      'investmentPurpose',
                      form.investmentPurpose.includes(v)
                        ? form.investmentPurpose.filter((x) => x !== v)
                        : [...form.investmentPurpose, v],
                    )
                  }
                  options={INVESTMENT_PURPOSES}
                  disabled={readOnly}
                />
              </Field>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
                <Field label={t('advisory.field.horizon')} required>
                  <SelectField value={form.horizon} onChange={(v) => set('horizon', v)} options={INVESTMENT_HORIZONS} disabled={readOnly} placeholder={t('advisory.field.horizonPlaceholder')} />
                </Field>
                <Field label={t('advisory.field.riskClass')} required>
                  <Select value={String(form.riskClass)} onValueChange={(v) => set('riskClass', Number(v) as SriClass)} disabled={readOnly}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {SRI_CLASSES.map((c) => (
                        <SelectItem key={c.value} value={String(c.value)}>
                          <span className="flex items-center gap-2">
                            <span className="h-2 w-2 rounded-full" style={{ backgroundColor: c.color }} />
                            {c.value} — {t(c.labelKey)}
                          </span>
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </Field>
                <Field label={t('advisory.field.riskTolerance')} required>
                  <Input value={form.riskTolerance} disabled={readOnly} onChange={(e) => set('riskTolerance', e.target.value)} />
                </Field>
                <Field label={t('advisory.field.riskCapacity')} required>
                  <Input value={form.riskCapacity} disabled={readOnly} onChange={(e) => set('riskCapacity', e.target.value)} />
                </Field>
                <Field label={t('advisory.field.oneTimeAmount')}>
                  <Input type="number" value={form.oneTimeAmount ?? ''} disabled={readOnly} onChange={(e) => set('oneTimeAmount', num(e.target.value))} />
                </Field>
                <Field label={t('advisory.field.monthlySavings')}>
                  <Input type="number" value={form.monthlySavings ?? ''} disabled={readOnly} onChange={(e) => set('monthlySavings', num(e.target.value))} />
                </Field>
              </div>
              <div className="flex items-start gap-3 rounded-xl border border-border bg-secondary/40 p-4">
                <Switch checked={form.esgPreference} disabled={readOnly} onCheckedChange={(v) => set('esgPreference', v)} />
                <div className="flex-1 space-y-2">
                  <Label className="text-sm font-medium text-foreground">{t('advisory.field.esgPreference')}</Label>
                  <p className="text-xs text-muted-foreground">{t('advisory.field.esgHint')}</p>
                  {form.esgPreference && (
                    <Input
                      className="mt-1"
                      placeholder={t('advisory.field.esgDetailsPlaceholder')}
                      value={form.esgDetails}
                      disabled={readOnly}
                      onChange={(e) => set('esgDetails', e.target.value)}
                    />
                  )}
                </div>
              </div>
            </div>
          </SectionCard>

          {/* Section 6 — Besprochene Produkte */}
          <SectionCard id="s6" index={6} title={t('advisory.section.products')}>
            {form.products.length === 0 ? (
              <p className="mb-4 text-sm text-muted-foreground">{t('advisory.products.empty')}</p>
            ) : (
              <div className="mb-4 space-y-4">
                {form.products.map((p, i) => (
                  <div key={p.id} className="rounded-xl border border-border p-4">
                    <div className="mb-3 flex items-center justify-between">
                      <span className="text-xs font-medium text-muted-foreground">
                        {t('advisory.products.item', { index: i + 1 })}
                      </span>
                      {!readOnly && (
                        <button onClick={() => removeProduct(p.id)} className="rounded p-1 text-muted-foreground transition-colors hover:text-destructive">
                          <Trash2 className="h-4 w-4" />
                        </button>
                      )}
                    </div>
                    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
                      <Field label={t('advisory.products.name')} required>
                        <Input value={p.name} disabled={readOnly} onChange={(e) => updateProduct(p.id, { name: e.target.value })} />
                      </Field>
                      <Field label={t('advisory.products.isin')}>
                        <Input value={p.isin ?? ''} disabled={readOnly} onChange={(e) => updateProduct(p.id, { isin: e.target.value })} />
                      </Field>
                      <Field label={t('advisory.products.category')}>
                        <Input value={p.category ?? ''} disabled={readOnly} onChange={(e) => updateProduct(p.id, { category: e.target.value })} />
                      </Field>
                      <Field label={t('advisory.field.riskClass')} required>
                        <Select value={String(p.riskClass)} onValueChange={(v) => updateProduct(p.id, { riskClass: Number(v) as SriClass })} disabled={readOnly}>
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            {SRI_CLASSES.map((c) => (
                              <SelectItem key={c.value} value={String(c.value)}>
                                {c.value} — {t(c.labelKey)}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </Field>
                      <Field label={t('advisory.products.costsOneTime')} required>
                        <Input type="number" value={p.costsOneTime ?? ''} disabled={readOnly} onChange={(e) => updateProduct(p.id, { costsOneTime: num(e.target.value) ?? undefined })} />
                      </Field>
                      <Field label={t('advisory.products.costsRunning')} required>
                        <Input type="number" value={p.costsRunning ?? ''} disabled={readOnly} onChange={(e) => updateProduct(p.id, { costsRunning: num(e.target.value) ?? undefined })} />
                      </Field>
                      <Field label={t('advisory.products.opportunities')}>
                        <Textarea rows={2} value={p.opportunities ?? ''} disabled={readOnly} onChange={(e) => updateProduct(p.id, { opportunities: e.target.value })} />
                      </Field>
                      <Field label={t('advisory.products.risks')} required>
                        <Textarea rows={2} value={p.risks} disabled={readOnly} onChange={(e) => updateProduct(p.id, { risks: e.target.value })} />
                      </Field>
                      <div className="flex items-center gap-2 self-end pb-2">
                        <Checkbox id={`rec-${p.id}`} checked={p.recommended} disabled={readOnly} onCheckedChange={(v) => updateProduct(p.id, { recommended: v === true })} />
                        <Label htmlFor={`rec-${p.id}`} className="text-sm text-foreground">{t('advisory.products.recommended')}</Label>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
            {!readOnly && (
              <Button variant="outline" size="sm" onClick={addProduct}>
                <Plus className="mr-2 h-4 w-4" />
                {t('advisory.products.add')}
              </Button>
            )}
          </SectionCard>

          {/* Section 7 — Empfehlung & Geeignetheit */}
          <SectionCard id="s7" index={7} title={t('advisory.section.recommendation')}>
            <div className="space-y-4">
              <Field label={t('advisory.field.recommendationSummary')} required>
                <Textarea rows={3} value={form.recommendationSummary} disabled={readOnly} onChange={(e) => set('recommendationSummary', e.target.value)} />
              </Field>
              <Field label={t('advisory.field.suitabilityReasoning')} required hint={t('advisory.field.suitabilityHint')}>
                <Textarea rows={3} value={form.suitabilityReasoning} disabled={readOnly} onChange={(e) => set('suitabilityReasoning', e.target.value)} />
              </Field>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <Field label={t('advisory.field.goalReference')} required>
                  <Textarea rows={2} value={form.goalReference} disabled={readOnly} onChange={(e) => set('goalReference', e.target.value)} />
                </Field>
                <Field label={t('advisory.field.riskClassReference')} hint={t('advisory.field.riskClassRefHint')}>
                  <div className="flex h-10 items-center gap-2 rounded-lg border border-input bg-secondary/40 px-3 text-sm">
                    <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: sriColor(form.riskClass) }} />
                    {t('advisory.field.riskClassRefValue', { sri: form.riskClass })}
                  </div>
                </Field>
                <Field label={t('advisory.field.alternatives')}>
                  <Textarea rows={2} value={form.alternatives} disabled={readOnly} onChange={(e) => set('alternatives', e.target.value)} />
                </Field>
                <Field label={t('advisory.field.notRecommended')}>
                  <Textarea rows={2} value={form.notRecommended} disabled={readOnly} onChange={(e) => set('notRecommended', e.target.value)} />
                </Field>
              </div>
            </div>
          </SectionCard>

          {/* Section 8 — Abschluss & Compliance */}
          <SectionCard id="s8" index={8} title={t('advisory.section.compliance')}>
            <div className="space-y-4">
              <Field label={t('advisory.field.mainConcerns')} required hint={t('advisory.field.mainConcernsHint')}>
                <Textarea rows={2} value={form.mainConcerns} disabled={readOnly} onChange={(e) => set('mainConcerns', e.target.value)} />
              </Field>
              <Field label={t('advisory.field.warningsGiven')} required>
                <div className="space-y-2 rounded-xl border border-border p-4">
                  {ADVISORY_WARNINGS.map((w) => (
                    <div key={w.value} className="flex items-center gap-2.5">
                      <Checkbox
                        id={`warn-${w.value}`}
                        checked={form.warningsGiven.includes(w.value)}
                        disabled={readOnly}
                        onCheckedChange={(v) =>
                          set(
                            'warningsGiven',
                            v === true
                              ? [...form.warningsGiven, w.value]
                              : form.warningsGiven.filter((x) => x !== w.value),
                          )
                        }
                      />
                      <Label htmlFor={`warn-${w.value}`} className="text-sm text-foreground">{t(w.labelKey)}</Label>
                    </div>
                  ))}
                </div>
              </Field>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
                <Field label={t('advisory.field.deliveryForm')} required>
                  <SelectField value={form.deliveryForm} onChange={(v) => set('deliveryForm', v)} options={DELIVERY_FORMS} disabled={readOnly} />
                </Field>
                <Field label={t('advisory.field.documentDeliveredDate')} required>
                  <Input type="date" value={form.documentDeliveredDate} disabled={readOnly} onChange={(e) => set('documentDeliveredDate', e.target.value)} />
                </Field>
                <Field label={t('advisory.field.advisorSignature')} required hint={t('advisory.field.signatureHint')}>
                  <Input value={form.advisorSignature} disabled={readOnly} onChange={(e) => set('advisorSignature', e.target.value)} />
                </Field>
                <Field label={t('advisory.field.followupDate')}>
                  <Input type="date" value={form.followupDate} disabled={readOnly} onChange={(e) => set('followupDate', e.target.value)} />
                </Field>
              </div>
              <div className="space-y-2.5 rounded-xl border border-border p-4">
                <div className="flex items-center gap-2.5">
                  <Checkbox id="doc-delivered" checked={form.documentDelivered} disabled={readOnly} onCheckedChange={(v) => set('documentDelivered', v === true)} />
                  <Label htmlFor="doc-delivered" className="text-sm text-foreground">{t('advisory.field.documentDelivered')}</Label>
                </div>
                <div className="flex items-center gap-2.5">
                  <Checkbox id="cust-confirm" checked={form.customerConfirmation} disabled={readOnly} onCheckedChange={(v) => set('customerConfirmation', v === true)} />
                  <Label htmlFor="cust-confirm" className="text-sm text-foreground">{t('advisory.field.customerConfirmation')}</Label>
                </div>
                <div className="flex items-center gap-2.5">
                  <Checkbox id="doc-waiver" checked={form.documentWaiver} disabled={readOnly} onCheckedChange={(v) => set('documentWaiver', v === true)} />
                  <Label htmlFor="doc-waiver" className="text-sm text-foreground">{t('advisory.field.documentWaiver')}</Label>
                </div>
              </div>
              <Field label={t('advisory.field.internalNotes')} hint={t('advisory.field.internalNotesHint')}>
                <Textarea rows={2} value={form.internalNotes} disabled={readOnly} onChange={(e) => set('internalNotes', e.target.value)} />
              </Field>
            </div>
          </SectionCard>

          {/* Retention notice */}
          <div className="flex items-start gap-2.5 rounded-xl border border-border bg-secondary/30 p-4 text-xs text-muted-foreground">
            <ShieldCheck className="mt-0.5 h-4 w-4 shrink-0 text-primary" />
            <p>{t('advisory.retentionNotice')}</p>
          </div>
        </div>
      </div>

      <ConfirmDialog
        open={confirmFinalize}
        onOpenChange={setConfirmFinalize}
        title={t('advisory.confirmFinalize.title')}
        description={t('advisory.confirmFinalize.body')}
        confirmLabel={t('advisory.finalize')}
        onConfirm={handleFinalize}
      />
    </div>
  )
}

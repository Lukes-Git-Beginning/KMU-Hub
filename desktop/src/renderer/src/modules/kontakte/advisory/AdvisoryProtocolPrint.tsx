/**
 * Print/PDF view of the advisory protocol ("Geeignetheitserklärung").
 *
 * Renders the finalized protocol as a clean, black-on-white document that the
 * advisor hands to the client on a durable medium (legal requirement under
 * MiFID II / §64 WpHG / FinVermV — must be provided before/with the contract).
 *
 * No PDF dependency: a scoped `@media print` block isolates `#advisory-print`
 * so the browser/Electron "Save as PDF" produces the document. The real backend
 * (server-side PDF + 10y immutable storage) is tracked in backend-gaps.md.
 */
import { useTranslation } from 'react-i18next'
import { createPortal } from 'react-dom'
import { Printer, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  ADVISORY_LOCATIONS,
  ADVISORY_OCCASIONS,
  ADVISORY_WARNINGS,
  ASSET_CLASSES,
  CUSTOMER_CATEGORIES,
  DELIVERY_FORMS,
  INVESTMENT_HORIZONS,
  INVESTMENT_PURPOSES,
  protocolDurationMin,
  type AdvisoryProtocol,
  type EnumOption,
} from '@/lib/advisory'

interface Props {
  protocol: AdvisoryProtocol
  contactName: string
  company?: string
  onClose: () => void
}

const PRINT_CSS = `
@media print {
  body * { visibility: hidden !important; }
  #advisory-print, #advisory-print * { visibility: visible !important; }
  #advisory-print { position: absolute; inset: 0; margin: 0; padding: 0; box-shadow: none; }
  #advisory-print-toolbar { display: none !important; }
}
`

export function AdvisoryProtocolPrint({ protocol: p, contactName, company, onClose }: Props) {
  const { t } = useTranslation()

  const optLabel = <T extends string>(opts: EnumOption<T>[], v: T | ''): string => {
    const o = opts.find((x) => x.value === v)
    return o ? t(o.labelKey) : ''
  }
  const multiLabel = <T extends string>(opts: EnumOption<T>[], vals: T[]): string =>
    vals.map((v) => optLabel(opts, v)).filter(Boolean).join(', ')

  const money = (v: number | null): string =>
    v == null ? '' : v.toLocaleString('de-DE', { style: 'currency', currency: 'EUR' })

  const duration = protocolDurationMin(p.timeFrom, p.timeTo)

  return createPortal(
    <div className="fixed inset-0 z-[80] overflow-auto bg-black/50">
      <style>{PRINT_CSS}</style>

      {/* Toolbar (hidden in print) */}
      <div
        id="advisory-print-toolbar"
        className="sticky top-0 z-10 flex items-center justify-between border-b border-border bg-card px-4 py-2.5"
      >
        <span className="text-sm font-medium text-foreground">{t('advisory.print.toolbarTitle')}</span>
        <div className="flex items-center gap-2">
          <Button size="sm" onClick={() => window.print()}>
            <Printer className="mr-2 h-4 w-4" />
            {t('advisory.print.action')}
          </Button>
          <Button variant="outline" size="sm" onClick={onClose}>
            <X className="mr-2 h-4 w-4" />
            {t('common.close')}
          </Button>
        </div>
      </div>

      {/* Document */}
      <div className="mx-auto my-6 max-w-[820px] px-4">
        <div
          id="advisory-print"
          className="mx-auto bg-white px-10 py-10 text-[13px] leading-relaxed text-gray-900 shadow-lg"
          style={{ fontFamily: 'Plus Jakarta Sans, sans-serif' }}
        >
          {/* Letterhead */}
          <div className="mb-6 flex items-start justify-between border-b border-gray-300 pb-4">
            <div>
              <h1 className="text-xl font-bold text-gray-900">{t('advisory.print.docTitle')}</h1>
              <p className="mt-1 text-xs text-gray-500">{t('advisory.print.docSubtitle')}</p>
            </div>
            {company && <p className="text-right text-sm font-semibold text-gray-700">{company}</p>}
          </div>

          {/* Parties */}
          <Grid>
            <Row label={t('advisory.print.client')} value={contactName} />
            <Row label={t('advisory.field.advisor')} value={p.advisor} />
            <Row label={t('advisory.field.date')} value={p.date} />
            <Row
              label={t('advisory.field.time')}
              value={
                p.timeFrom && p.timeTo
                  ? `${p.timeFrom}–${p.timeTo}${duration != null ? ` (${duration} min)` : ''}`
                  : ''
              }
            />
            <Row label={t('advisory.field.location')} value={optLabel(ADVISORY_LOCATIONS, p.location)} />
            <Row label={t('advisory.field.occasion')} value={optLabel(ADVISORY_OCCASIONS, p.occasion)} />
            <Row label={t('advisory.field.customerCategory')} value={optLabel(CUSTOMER_CATEGORIES, p.customerCategory)} />
          </Grid>

          {/* 2 — Kunde & Profil */}
          <Section title={t('advisory.section.customer')}>
            <Grid>
              <Row label={t('advisory.field.birthDate')} value={p.birthDate} />
              <Row label={t('advisory.field.maritalStatus')} value={p.maritalStatus} />
              <Row label={t('advisory.field.taxStatus')} value={p.taxStatus} />
            </Grid>
          </Section>

          {/* 3 — Kenntnisse & Erfahrungen */}
          <Section title={t('advisory.section.knowledge')}>
            <Row label={t('advisory.field.knownAssetClasses')} value={multiLabel(ASSET_CLASSES, p.knownAssetClasses)} block />
            <Row label={t('advisory.field.pastTransactions')} value={p.pastTransactions} block />
            <Row label={t('advisory.field.professionalExperience')} value={p.professionalExperience} block />
            <Row label={t('advisory.field.selfAssessment')} value={p.selfAssessment != null ? `${p.selfAssessment}/5` : ''} />
          </Section>

          {/* 4 — Finanzielle Situation */}
          <Section title={t('advisory.section.financial')}>
            <Grid>
              <Row label={t('advisory.field.monthlyNetIncome')} value={money(p.monthlyNetIncome)} />
              <Row label={t('advisory.field.recurringLiabilities')} value={money(p.recurringLiabilities)} />
              <Row label={t('advisory.field.liquidAssets')} value={money(p.liquidAssets)} />
              <Row label={t('advisory.field.currentInvestments')} value={money(p.currentInvestments)} />
              <Row
                label={t('advisory.field.maxLossCapacity')}
                value={[money(p.maxLossCapacityAbs), p.maxLossCapacityPct != null ? `${p.maxLossCapacityPct} %` : '']
                  .filter(Boolean)
                  .join(' · ')}
              />
            </Grid>
            <Row label={t('advisory.field.realEstate')} value={p.realEstate} block />
            <Row label={t('advisory.field.existingInsurance')} value={p.existingInsurance} block />
          </Section>

          {/* 5 — Anlageziele & Risikoprofil */}
          <Section title={t('advisory.section.goals')}>
            <Row label={t('advisory.field.investmentPurpose')} value={multiLabel(INVESTMENT_PURPOSES, p.investmentPurpose)} block />
            <Grid>
              <Row label={t('advisory.field.horizon')} value={optLabel(INVESTMENT_HORIZONS, p.horizon)} />
              <Row label={t('advisory.field.riskClass')} value={t('advisory.print.sriValue', { n: p.riskClass })} />
              <Row label={t('advisory.field.oneTimeAmount')} value={money(p.oneTimeAmount)} />
              <Row label={t('advisory.field.monthlySavings')} value={money(p.monthlySavings)} />
              <Row label={t('advisory.field.esg')} value={p.esgPreference ? t('common.yes') : t('common.no')} />
            </Grid>
            <Row label={t('advisory.field.riskTolerance')} value={p.riskTolerance} block />
            <Row label={t('advisory.field.riskCapacity')} value={p.riskCapacity} block />
            {p.esgPreference && <Row label={t('advisory.field.esgDetails')} value={p.esgDetails} block />}
          </Section>

          {/* 6 — Besprochene Produkte */}
          {p.products.length > 0 && (
            <Section title={t('advisory.section.products')}>
              <table className="w-full border-collapse text-[12px]">
                <thead>
                  <tr className="border-b border-gray-400 text-left">
                    <th className="py-1 pr-2 font-semibold">{t('advisory.product.name')}</th>
                    <th className="py-1 pr-2 font-semibold">ISIN</th>
                    <th className="py-1 pr-2 font-semibold">SRI</th>
                    <th className="py-1 pr-2 font-semibold">{t('advisory.product.costs')}</th>
                    <th className="py-1 font-semibold">{t('advisory.product.recommended')}</th>
                  </tr>
                </thead>
                <tbody>
                  {p.products.map((prod) => (
                    <tr key={prod.id} className="border-b border-gray-200 align-top">
                      <td className="py-1 pr-2">{prod.name || '—'}</td>
                      <td className="py-1 pr-2">{prod.isin || '—'}</td>
                      <td className="py-1 pr-2">{prod.riskClass}</td>
                      <td className="py-1 pr-2">
                        {[prod.costsOneTime != null ? money(prod.costsOneTime) : '', prod.costsRunning != null ? `${money(prod.costsRunning)} p.a.` : '']
                          .filter(Boolean)
                          .join(' · ') || '—'}
                      </td>
                      <td className="py-1">{prod.recommended ? t('common.yes') : t('common.no')}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </Section>
          )}

          {/* 7 — Empfehlung & Geeignetheit (Kern) */}
          <Section title={t('advisory.section.recommendation')}>
            <Row label={t('advisory.field.recommendationSummary')} value={p.recommendationSummary} block />
            <Row label={t('advisory.field.suitabilityReasoning')} value={p.suitabilityReasoning} block />
            <Row label={t('advisory.field.goalReference')} value={p.goalReference} block />
            <Row label={t('advisory.field.alternatives')} value={p.alternatives} block />
            <Row label={t('advisory.field.notRecommended')} value={p.notRecommended} block />
          </Section>

          {/* 8 — Abschluss & Compliance */}
          <Section title={t('advisory.section.compliance')}>
            <Row label={t('advisory.field.mainConcerns')} value={p.mainConcerns} block />
            <Row label={t('advisory.field.warningsGiven')} value={multiLabel(ADVISORY_WARNINGS, p.warningsGiven)} block />
            <Grid>
              <Row label={t('advisory.field.deliveryForm')} value={optLabel(DELIVERY_FORMS, p.deliveryForm)} />
              <Row label={t('advisory.field.documentDeliveredDate')} value={p.documentDeliveredDate} />
              <Row label={t('advisory.field.customerConfirmation')} value={p.customerConfirmation ? t('common.yes') : t('common.no')} />
              <Row label={t('advisory.field.followupDate')} value={p.followupDate} />
            </Grid>
          </Section>

          {/* Signatures */}
          <div className="mt-10 grid grid-cols-2 gap-10">
            <SignatureLine label={t('advisory.print.signatureAdvisor')} />
            <SignatureLine label={t('advisory.print.signatureClient')} />
          </div>

          <p className="mt-8 border-t border-gray-200 pt-3 text-[10px] text-gray-400">
            {t('advisory.print.legalNote')}
          </p>
        </div>
      </div>
    </div>,
    document.body,
  )
}

// --- presentational helpers (print-safe, theme-independent) -----------------

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="mt-6">
      <h2 className="mb-2 border-b border-gray-200 pb-1 text-sm font-bold uppercase tracking-wide text-gray-700">
        {title}
      </h2>
      <div className="space-y-1.5">{children}</div>
    </section>
  )
}

function Grid({ children }: { children: React.ReactNode }) {
  return <div className="grid grid-cols-2 gap-x-8 gap-y-1.5">{children}</div>
}

function Row({ label, value, block }: { label: string; value?: string; block?: boolean }) {
  if (!value) return null
  if (block) {
    return (
      <div className="col-span-2">
        <p className="text-[11px] font-medium uppercase tracking-wide text-gray-500">{label}</p>
        <p className="whitespace-pre-wrap text-gray-900">{value}</p>
      </div>
    )
  }
  return (
    <div className="flex justify-between gap-3">
      <span className="text-gray-500">{label}</span>
      <span className="text-right font-medium text-gray-900">{value}</span>
    </div>
  )
}

function SignatureLine({ label }: { label: string }) {
  return (
    <div>
      <div className="h-10 border-b border-gray-400" />
      <p className="mt-1 text-[11px] text-gray-500">{label}</p>
    </div>
  )
}

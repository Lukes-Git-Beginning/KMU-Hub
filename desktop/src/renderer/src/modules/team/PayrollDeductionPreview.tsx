/**
 * Multi-country gross→net deduction preview (DE/CH/AT) — a plausibility helper
 * inside the Lohnvorbereitung. Illustrative rates only (real calc = DATEV).
 * Extracted from the former HRIntegrationPanel.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Globe, Banknote, FileText } from 'lucide-react'

type Country = 'DE' | 'CH' | 'AT'

interface CountryDeduction {
  label: string
  rate: string
  amount: number
  type: 'employee' | 'employer' | 'both'
}

const COUNTRY_DEDUCTIONS: Record<Country, { label: string; currency: string; grossExample: number; deductions: CountryDeduction[] }> = {
  DE: {
    label: 'Deutschland',
    currency: 'EUR',
    grossExample: 4200,
    deductions: [
      { label: 'Lohnsteuer', rate: '~18,5%', amount: 777, type: 'employee' },
      { label: 'Solidaritätszuschlag', rate: '5,5% d. LSt', amount: 42, type: 'employee' },
      { label: 'Kirchensteuer', rate: '8% d. LSt', amount: 62, type: 'employee' },
      { label: 'Rentenversicherung', rate: '9,3%', amount: 391, type: 'both' },
      { label: 'Krankenversicherung', rate: '7,3% + Zusatz', amount: 340, type: 'both' },
      { label: 'Pflegeversicherung', rate: '1,7%', amount: 71, type: 'both' },
      { label: 'Arbeitslosenversicherung', rate: '1,3%', amount: 55, type: 'both' },
    ],
  },
  CH: {
    label: 'Schweiz',
    currency: 'CHF',
    grossExample: 6500,
    deductions: [
      { label: 'AHV / IV / EO', rate: '5,3%', amount: 345, type: 'both' },
      { label: 'BVG (Pensionskasse)', rate: '~7%', amount: 455, type: 'both' },
      { label: 'Quellensteuer', rate: 'Tarif A', amount: 520, type: 'employee' },
      { label: 'NBU (Nichtberufsunfall)', rate: '~1,4%', amount: 91, type: 'employee' },
      { label: 'ALV', rate: '1,1%', amount: 72, type: 'both' },
    ],
  },
  AT: {
    label: 'Österreich',
    currency: 'EUR',
    grossExample: 3800,
    deductions: [
      { label: 'Lohnsteuer', rate: '~20%', amount: 760, type: 'employee' },
      { label: 'Sozialversicherung (SV)', rate: '18,12%', amount: 689, type: 'employee' },
      { label: 'Krankenversicherung', rate: '3,87%', amount: 147, type: 'both' },
      { label: 'Pensionsversicherung', rate: '10,25%', amount: 390, type: 'both' },
      { label: 'Arbeitslosenversicherung', rate: '3%', amount: 114, type: 'both' },
    ],
  },
}

const fmt = (amount: number, currency: string) =>
  new Intl.NumberFormat('de-DE', { style: 'currency', currency, minimumFractionDigits: 0, maximumFractionDigits: 0 }).format(amount)

export function PayrollDeductionPreview() {
  const { t } = useTranslation()
  const [country, setCountry] = useState<Country>('DE')
  const data = COUNTRY_DEDUCTIONS[country]
  const total = data.deductions.reduce((s, d) => s + d.amount, 0)
  const net = data.grossExample - total

  return (
    <div>
      <div className="mb-3 flex items-center justify-between">
        <h3 className="flex items-center gap-2 text-sm font-medium text-foreground">
          <Globe className="h-4 w-4 text-muted-foreground" />
          {t('team.integration.salaryPreview')}
        </h3>
        <div className="flex items-center gap-1 rounded-lg border border-border bg-card p-0.5">
          {(['DE', 'CH', 'AT'] as Country[]).map((c) => (
            <button
              key={c}
              onClick={() => setCountry(c)}
              className={`rounded-md px-3 py-1 text-xs font-medium transition-colors ${
                country === c ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              {c}
            </button>
          ))}
        </div>
      </div>

      <div className="overflow-hidden rounded-lg border border-border bg-card">
        <div className="flex items-center justify-between border-b border-border bg-secondary/30 px-4 py-3">
          <div className="flex items-center gap-2">
            <Banknote className="h-4 w-4 text-primary" />
            <span className="text-sm font-medium text-foreground">{data.label}</span>
          </div>
          <span className="text-xs text-muted-foreground">
            {t('team.integration.example')}: {fmt(data.grossExample, data.currency)} {t('team.integration.gross')}
          </span>
        </div>

        <div className="divide-y divide-border-muted">
          {data.deductions.map((ded, i) => (
            <div key={i} className="flex items-center justify-between px-4 py-2.5">
              <div className="flex items-center gap-2">
                <span className="text-sm text-foreground">{ded.label}</span>
                <span className={`rounded-full px-1.5 py-0.5 text-[9px] font-medium ${
                  ded.type === 'employee' ? 'bg-warning-light text-warning' : ded.type === 'employer' ? 'bg-info-light text-info' : 'bg-secondary text-muted-foreground'
                }`}>
                  {ded.type === 'employee' ? 'AN' : ded.type === 'employer' ? 'AG' : 'AN+AG'}
                </span>
              </div>
              <div className="flex items-center gap-3">
                <span className="text-xs text-muted-foreground tabular-nums">{ded.rate}</span>
                <span className="w-20 text-right text-sm font-medium text-error tabular-nums">−{fmt(ded.amount, data.currency)}</span>
              </div>
            </div>
          ))}
        </div>

        <div className="border-t border-border bg-secondary/20 px-4 py-3">
          <div className="mb-1 flex items-center justify-between">
            <span className="text-sm text-muted-foreground">{t('team.integration.grossSalary')}</span>
            <span className="text-sm font-medium text-foreground tabular-nums">{fmt(data.grossExample, data.currency)}</span>
          </div>
          <div className="mb-1 flex items-center justify-between">
            <span className="text-sm text-muted-foreground">{t('team.integration.totalDeductions')}</span>
            <span className="text-sm font-medium text-error tabular-nums">−{fmt(total, data.currency)}</span>
          </div>
          <div className="mt-2 flex items-center justify-between border-t border-border pt-2">
            <span className="text-sm font-semibold text-foreground">{t('team.integration.netSalary')}</span>
            <span className="text-base font-bold text-foreground tabular-nums">{fmt(net, data.currency)}</span>
          </div>
        </div>
      </div>

      <p className="mt-2 flex items-center gap-1 text-[11px] text-muted-foreground">
        <FileText className="h-3 w-3" />
        {t('team.integration.disclaimer')}
      </p>
    </div>
  )
}

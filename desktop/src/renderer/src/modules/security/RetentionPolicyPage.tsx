/**
 * Retention Policy Page — displays legally required data retention periods.
 *
 * Shows a table of document types with retention period, legal basis,
 * and status. Supports DE/AT/CH tabs (Germany default).
 */
import { useState, useCallback } from 'react'
import { Navigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Clock, Info } from 'lucide-react'
import { Switch } from '@/components/ui/switch'
import { useAuthStore } from '@/stores/auth'

type Country = 'de' | 'at' | 'ch'

interface RetentionEntry {
  type: string
  period: string
  basis: string
  status: 'aktiv' | 'auslaufend' | 'abgelaufen'
}

const COUNTRY_LABELS: Record<Country, string> = {
  de: 'Deutschland',
  at: 'Österreich',
  ch: 'Schweiz',
}

const RETENTION_DATA: Record<Country, RetentionEntry[]> = {
  de: [
    { type: 'Rechnungen (eingehend/ausgehend)', period: '10 Jahre', basis: '\u00A7 147 AO, \u00A7 14b UStG', status: 'aktiv' },
    { type: 'Jahresabschluesse', period: '10 Jahre', basis: '\u00A7 257 HGB', status: 'aktiv' },
    { type: 'Buchungsbelege', period: '10 Jahre', basis: '\u00A7 147 AO', status: 'aktiv' },
    { type: 'Geschäftsbriefe (empfangen/gesendet)', period: '6 Jahre', basis: '\u00A7 257 HGB', status: 'aktiv' },
    { type: 'Verträge', period: '10 Jahre nach Ende', basis: '\u00A7 147 AO', status: 'aktiv' },
    { type: 'E-Mails (geschäftlich)', period: '6\u201310 Jahre', basis: 'GoBD', status: 'aktiv' },
    { type: 'Personalakten', period: '3 Jahre nach Austritt', basis: '\u00A7 195 BGB', status: 'aktiv' },
    { type: 'Bewerbungsunterlagen (abgelehnt)', period: '6 Monate', basis: 'AGG', status: 'aktiv' },
    { type: 'Arbeitszeitnachweise', period: '2 Jahre', basis: '\u00A7 16 ArbZG', status: 'aktiv' },
    { type: 'Lohnabrechnungen', period: '6 Jahre', basis: '\u00A7 257 HGB', status: 'aktiv' },
    { type: 'DSGVO-Einwilligungen', period: 'Bis Widerruf + 3 Jahre', basis: 'Art. 7 DSGVO', status: 'aktiv' },
    { type: 'Steuerunterlagen', period: '10 Jahre', basis: '\u00A7 147 AO', status: 'aktiv' },
  ],
  at: [
    { type: 'Rechnungen / Buchungsbelege', period: '7 Jahre', basis: '\u00A7 132 BAO', status: 'aktiv' },
    { type: 'Jahresabschluesse', period: '7 Jahre', basis: '\u00A7 212 UGB', status: 'aktiv' },
    { type: 'Geschäftsbriefe', period: '7 Jahre', basis: '\u00A7 212 UGB', status: 'aktiv' },
    { type: 'Verträge', period: '7 Jahre nach Ende', basis: '\u00A7 132 BAO', status: 'aktiv' },
    { type: 'E-Mails (geschäftlich)', period: '7 Jahre', basis: 'BAO', status: 'aktiv' },
    { type: 'Personalakten', period: '3 Jahre nach Austritt', basis: 'ABGB', status: 'aktiv' },
    { type: 'Arbeitszeitnachweise', period: '2 Jahre', basis: 'AZG', status: 'aktiv' },
    { type: 'DSGVO-Einwilligungen', period: 'Bis Widerruf + 3 Jahre', basis: 'Art. 7 DSGVO', status: 'aktiv' },
  ],
  ch: [
    { type: 'Rechnungen / Buchungsbelege', period: '10 Jahre', basis: 'Art. 958f OR', status: 'aktiv' },
    { type: 'Jahresabschluesse', period: '10 Jahre', basis: 'Art. 958f OR', status: 'aktiv' },
    { type: 'Geschäftsbriefe', period: '10 Jahre', basis: 'Art. 962 OR', status: 'aktiv' },
    { type: 'Verträge', period: '10 Jahre nach Ende', basis: 'Art. 127 OR', status: 'aktiv' },
    { type: 'E-Mails (geschäftlich)', period: '10 Jahre', basis: 'GeBüV', status: 'aktiv' },
    { type: 'Personalakten', period: '5 Jahre nach Austritt', basis: 'Art. 128a OR', status: 'aktiv' },
    { type: 'Arbeitszeitnachweise', period: '5 Jahre', basis: 'ArG', status: 'aktiv' },
    { type: 'AHV/BVG-Unterlagen', period: '10 Jahre', basis: 'AHVG/BVG', status: 'aktiv' },
    { type: 'DSG-Einwilligungen', period: 'Bis Widerruf + 3 Jahre', basis: 'nDSG', status: 'aktiv' },
  ],
}

export default function RetentionPolicyPage() {
  const { t } = useTranslation()
  const user = useAuthStore((s) => s.user)
  const isAdmin = user?.roles.includes('admin')
  const [country, setCountry] = useState<Country>('de')
  // Which categories have their automatic deletion rule paused (demo state).
  const [paused, setPaused] = useState<Set<string>>(new Set())

  const entries = RETENTION_DATA[country]

  const togglePause = useCallback((key: string) => {
    setPaused((prev) => {
      const next = new Set(prev)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })
  }, [])

  if (!isAdmin) {
    return <Navigate to="/" replace />
  }

  return (
    <div className="flex-1 overflow-y-auto p-6">
      <div className="mx-auto max-w-3xl">
        {/* Header */}
        <div className="flex items-center gap-4 mb-6">
          <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-warning-light">
            <Clock className="h-6 w-6 text-warning" />
          </div>
          <div>
            <h1 className="text-foreground">{t('security.retention.title')}</h1>
            <p className="text-sm text-muted-foreground mt-0.5">
              {t('security.retention.subtitle')}
            </p>
          </div>
        </div>

        {/* Info */}
        <div className="flex items-start gap-2 rounded-lg bg-info-light/50 px-4 py-3 mb-6">
          <Info className="h-4 w-4 text-info mt-0.5 shrink-0" />
          <p className="text-xs text-info">
            {t('security.retention.infoNote')}
          </p>
        </div>

        {/* Country tabs */}
        <div className="flex items-center gap-1 rounded-lg border border-border p-1 mb-6 w-fit">
          {(Object.keys(COUNTRY_LABELS) as Country[]).map((c) => (
            <button
              key={c}
              onClick={() => setCountry(c)}
              className={`rounded-md px-4 py-1.5 text-xs font-medium transition-colors ${
                country === c
                  ? 'bg-primary text-primary-foreground'
                  : 'text-muted-foreground hover:bg-secondary'
              }`}
            >
              {COUNTRY_LABELS[c]}
            </button>
          ))}
        </div>

        {/* Table */}
        <div className="rounded-lg border border-border bg-card overflow-hidden glass-surface">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-secondary/30">
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('security.retention.col.documentType')}</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('security.retention.col.retentionPeriod')}</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('security.retention.col.legalBasis')}</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">{t('security.retention.col.autoDeletion')}</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((entry, i) => (
                <tr key={i} className="border-b border-border-muted last:border-0 hover:bg-secondary/30 transition-colors">
                  <td className="px-4 py-3 text-sm text-foreground">{entry.type}</td>
                  <td className="px-4 py-3 text-sm text-foreground font-medium">{entry.period}</td>
                  <td className="px-4 py-3 text-xs text-muted-foreground font-mono">{entry.basis}</td>
                  <td className="px-4 py-3">
                    {(() => {
                      const key = `${country}-${entry.type}`
                      const isPaused = paused.has(key)
                      return (
                        <div className="flex items-center gap-2.5">
                          <Switch
                            checked={!isPaused}
                            onCheckedChange={() => togglePause(key)}
                            aria-label={t('security.retention.col.autoDeletion')}
                          />
                          <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${isPaused ? 'bg-secondary text-muted-foreground' : 'bg-success-light text-success'}`}>
                            {isPaused ? t('security.retention.status.paused') : t('security.retention.status.active')}
                          </span>
                        </div>
                      )
                    })()}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Summary */}
        <div className="mt-4 flex items-center gap-4 text-xs text-muted-foreground">
          <span>{t('security.retention.summary.documentTypes', { count: entries.length })}</span>
          <span>&middot;</span>
          <span>{t('security.retention.summary.country', { country: COUNTRY_LABELS[country] })}</span>
          <span>&middot;</span>
          <span>{t('security.retention.summary.asOf')}</span>
        </div>
      </div>
    </div>
  )
}

/**
 * Lead-scoring rule editor (CRM settings → tenant scope).
 *
 * Lets a CRM lead configure the auto-score weights (source base + per-field
 * points) and the hot/warm thresholds — replacing the formerly hardcoded
 * logic in useLeads. Market-standard pattern (HubSpot/Pipedrive). Mock-first
 * via useLeadScoringStore; backend tracked in backend-gaps.md.
 */
import { useTranslation } from 'react-i18next'
import { RotateCcw } from 'lucide-react'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { useLeadScoringStore } from '@/stores/leadScoring'
import { computeLeadScore, scoreToTemperature } from '@/api/hooks/useLeads'

const TEMP_COLOR: Record<string, string> = { hot: '#dc2626', warm: '#d97706', cold: '#2563eb' }

function NumField({ id, label, value, onChange }: { id: string; label: string; value: number; onChange: (v: number) => void }) {
  return (
    <div className="space-y-1.5">
      <Label htmlFor={id} className="text-sm font-medium text-foreground">{label}</Label>
      <Input id={id} type="number" min={0} max={100} step={1} value={value} onChange={(e) => onChange(Number(e.target.value) || 0)} />
    </div>
  )
}

export function LeadScoringSettings() {
  const { t } = useTranslation()
  const config = useLeadScoringStore((s) => s.config)
  const setConfig = useLeadScoringStore((s) => s.setConfig)
  const reset = useLeadScoringStore((s) => s.reset)

  // Live preview against a representative fully-filled dialer lead.
  const sample = { source: 'dialer' as const, email: 'a@b.de', phone: '+49', company: 'X GmbH', notes: 'Interesse' }
  const sampleScore = computeLeadScore(sample)
  const sampleTemp = scoreToTemperature(sampleScore)

  return (
    <div className="space-y-6">
      {/* Source base points */}
      <div>
        <h4 className="mb-2 text-sm font-semibold text-foreground">{t('crm.leadScoring.sourceTitle')}</h4>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
          <NumField id="ls-dialer" label={t('crm.leadScoring.source.dialer')} value={config.sourceBase.dialer} onChange={(v) => setConfig({ sourceBase: { ...config.sourceBase, dialer: v } })} />
          <NumField id="ls-manual" label={t('crm.leadScoring.source.manual')} value={config.sourceBase.manual} onChange={(v) => setConfig({ sourceBase: { ...config.sourceBase, manual: v } })} />
          <NumField id="ls-csv" label={t('crm.leadScoring.source.csv')} value={config.sourceBase.csv} onChange={(v) => setConfig({ sourceBase: { ...config.sourceBase, csv: v } })} />
        </div>
      </div>

      {/* Per-field points */}
      <div>
        <h4 className="mb-2 text-sm font-semibold text-foreground">{t('crm.leadScoring.fieldTitle')}</h4>
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <NumField id="ls-email" label={t('crm.leadScoring.field.email')} value={config.fieldPoints.email} onChange={(v) => setConfig({ fieldPoints: { ...config.fieldPoints, email: v } })} />
          <NumField id="ls-phone" label={t('crm.leadScoring.field.phone')} value={config.fieldPoints.phone} onChange={(v) => setConfig({ fieldPoints: { ...config.fieldPoints, phone: v } })} />
          <NumField id="ls-company" label={t('crm.leadScoring.field.company')} value={config.fieldPoints.company} onChange={(v) => setConfig({ fieldPoints: { ...config.fieldPoints, company: v } })} />
          <NumField id="ls-notes" label={t('crm.leadScoring.field.notes')} value={config.fieldPoints.notes} onChange={(v) => setConfig({ fieldPoints: { ...config.fieldPoints, notes: v } })} />
        </div>
      </div>

      {/* Thresholds */}
      <div>
        <h4 className="mb-2 text-sm font-semibold text-foreground">{t('crm.leadScoring.thresholdTitle')}</h4>
        <div className="grid grid-cols-2 gap-3">
          <NumField id="ls-hot" label={t('crm.leadScoring.hotFrom')} value={config.thresholds.hot} onChange={(v) => setConfig({ thresholds: { ...config.thresholds, hot: v } })} />
          <NumField id="ls-warm" label={t('crm.leadScoring.warmFrom')} value={config.thresholds.warm} onChange={(v) => setConfig({ thresholds: { ...config.thresholds, warm: v } })} />
        </div>
      </div>

      {/* Live preview */}
      <div className="flex items-center justify-between rounded-xl border border-border bg-secondary/40 px-4 py-3">
        <div className="flex items-center gap-2.5 text-sm">
          <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: TEMP_COLOR[sampleTemp] }} />
          <span className="text-muted-foreground">{t('crm.leadScoring.preview')}</span>
          <span className="font-medium text-foreground">{sampleScore} · {t(`crm.leadScoring.temp.${sampleTemp}`)}</span>
        </div>
        <Button variant="outline" size="sm" onClick={reset}>
          <RotateCcw className="mr-2 h-3.5 w-3.5" />
          {t('crm.leadScoring.reset')}
        </Button>
      </div>
    </div>
  )
}

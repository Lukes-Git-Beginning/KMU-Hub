/**
 * Mandanten-Segmente configuration (CRM settings → "Für alle").
 * Two thresholds drive the rule-based A/B/C classification by revenue potential.
 */
import { useTranslation } from 'react-i18next'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { useSegmentSettingsStore } from '@/stores/segmentSettings'
import { SEGMENT_META } from '@/lib/segments'

function fmtEur(v: number): string {
  return new Intl.NumberFormat('de-DE', { style: 'currency', currency: 'EUR', maximumFractionDigits: 0 }).format(v)
}

export function SegmentSettings() {
  const { t } = useTranslation()
  const thresholds = useSegmentSettingsStore((s) => s.thresholds)
  const setThreshold = useSegmentSettingsStore((s) => s.setThreshold)

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="space-y-1.5">
          <Label htmlFor="seg-a" className="text-sm font-medium text-foreground">
            {t('crm.segment.thresholdA')}
          </Label>
          <Input
            id="seg-a"
            type="number"
            min={0}
            step={1000}
            value={thresholds.a}
            onChange={(e) => setThreshold('a', Number(e.target.value) || 0)}
          />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="seg-b" className="text-sm font-medium text-foreground">
            {t('crm.segment.thresholdB')}
          </Label>
          <Input
            id="seg-b"
            type="number"
            min={0}
            step={1000}
            value={thresholds.b}
            onChange={(e) => setThreshold('b', Number(e.target.value) || 0)}
          />
        </div>
      </div>

      {/* Live legend */}
      <ul className="space-y-1.5 rounded-xl border border-border bg-secondary/40 p-4">
        <li className="flex items-center gap-2.5 text-sm">
          <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: SEGMENT_META.A.color }} />
          <span className="font-medium text-foreground">{t('crm.segment.a')}</span>
          <span className="text-muted-foreground">{t('crm.segment.ruleFrom', { value: fmtEur(thresholds.a) })}</span>
        </li>
        <li className="flex items-center gap-2.5 text-sm">
          <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: SEGMENT_META.B.color }} />
          <span className="font-medium text-foreground">{t('crm.segment.b')}</span>
          <span className="text-muted-foreground">{t('crm.segment.ruleRange', { from: fmtEur(thresholds.b), to: fmtEur(thresholds.a) })}</span>
        </li>
        <li className="flex items-center gap-2.5 text-sm">
          <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: SEGMENT_META.C.color }} />
          <span className="font-medium text-foreground">{t('crm.segment.c')}</span>
          <span className="text-muted-foreground">{t('crm.segment.ruleUnder', { value: fmtEur(thresholds.b) })}</span>
        </li>
      </ul>
    </div>
  )
}

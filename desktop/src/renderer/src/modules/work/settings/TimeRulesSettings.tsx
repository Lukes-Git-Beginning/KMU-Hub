/**
 * TimeRulesSettings — tenant-wide time-tracking rules.
 *
 * Embedded in the work settings panel. Billable-by-default toggle + default
 * hourly rate used for hours→invoice. Mock-first (stores/workSettings).
 */
import { useTranslation } from 'react-i18next'
import { useWorkSettingsStore } from '@/stores/workSettings'

export function TimeRulesSettings() {
  const { t } = useTranslation()
  const billableByDefault = useWorkSettingsStore((s) => s.billableByDefault)
  const defaultHourlyRate = useWorkSettingsStore((s) => s.defaultHourlyRate)
  const setBillableByDefault = useWorkSettingsStore((s) => s.setBillableByDefault)
  const setDefaultHourlyRate = useWorkSettingsStore((s) => s.setDefaultHourlyRate)

  return (
    <div className="space-y-4">
      {/* Billable by default */}
      <label className="flex cursor-pointer items-center justify-between gap-3 rounded-lg border border-border px-3 py-2.5">
        <span>
          <span className="block text-sm text-foreground">{t('work.settings.time.billableDefault')}</span>
          <span className="block text-xs text-muted-foreground">{t('work.settings.time.billableDefaultHint')}</span>
        </span>
        <button
          type="button"
          role="switch"
          aria-checked={billableByDefault}
          onClick={() => setBillableByDefault(!billableByDefault)}
          className={`flex h-5 w-9 shrink-0 items-center rounded-full px-0.5 transition-colors ${
            billableByDefault ? 'justify-end bg-primary' : 'justify-start bg-border'
          }`}
        >
          <span className="h-4 w-4 rounded-full bg-white shadow-sm" />
        </button>
      </label>

      {/* Default hourly rate */}
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('work.settings.time.hourlyRate')}</label>
        <div className="flex items-center gap-2">
          <input
            type="number"
            min={0}
            step={5}
            value={defaultHourlyRate}
            onChange={(e) => setDefaultHourlyRate(Math.max(0, Number(e.target.value) || 0))}
            className="h-9 w-32 rounded-lg border border-border bg-transparent px-2 text-sm text-foreground outline-none focus:border-primary"
          />
          <span className="text-sm text-muted-foreground">{t('work.settings.time.hourlyRateUnit')}</span>
        </div>
        <p className="text-xs text-muted-foreground">{t('work.settings.time.hourlyRateHint')}</p>
      </div>
    </div>
  )
}

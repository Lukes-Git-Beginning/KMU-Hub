/**
 * Kontierungs-Einstellungen (tenant-Scope / "Für alle", ModuleSettingsShell).
 *
 * Steuert den Kontenrahmen (SKR03/SKR04), der die Sachkonto-Vorschläge bei der
 * Ausgaben-Kontierung bestimmt und später den DATEV-EXTF-Export (P3) parametriert.
 */
import { useTranslation } from 'react-i18next'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useFinanceTenantStore } from '@/stores/financeTenant'
import type { ChartFramework } from './lib/skr-accounts'

const FRAMEWORKS: { value: ChartFramework; descKey: string }[] = [
  { value: 'SKR03', descKey: 'finanzen.settings.kontierung.skr03Desc' },
  { value: 'SKR04', descKey: 'finanzen.settings.kontierung.skr04Desc' },
]

export function KontierungSettings() {
  const { t } = useTranslation()
  const framework = useFinanceTenantStore((s) => s.chartFramework)
  const setFramework = useFinanceTenantStore((s) => s.setChartFramework)

  return (
    <div className="space-y-1.5">
      <Label className="text-sm font-medium text-foreground">
        {t('finanzen.settings.kontierung.frameworkLabel')}
      </Label>
      <Select value={framework} onValueChange={(v) => setFramework(v as ChartFramework)}>
        <SelectTrigger className="max-w-xs">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {FRAMEWORKS.map((f) => (
            <SelectItem key={f.value} value={f.value}>
              {f.value}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <p className="max-w-md text-xs text-muted-foreground">
        {t(FRAMEWORKS.find((f) => f.value === framework)!.descKey)}
      </p>
    </div>
  )
}

/**
 * Personal Buchhaltung preferences (personal scope in the module-settings shell).
 * Currently one wired comfort setting: the start tab.
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
import { useFinancePrefsStore, type FinanceStartTab } from '@/stores/financePrefs'

const START_TAB_OPTIONS: { value: FinanceStartTab; labelKey: string }[] = [
  { value: 'last', labelKey: 'finanzen.prefs.startTab.last' },
  { value: 'dashboard', labelKey: 'finanzen.prefs.startTab.dashboard' },
  { value: 'invoices', labelKey: 'finanzen.prefs.startTab.invoices' },
  { value: 'quotes', labelKey: 'finanzen.prefs.startTab.quotes' },
  { value: 'expenses', labelKey: 'finanzen.prefs.startTab.expenses' },
  { value: 'dunning', labelKey: 'finanzen.prefs.startTab.dunning' },
]

export function FinancePersonalPrefs() {
  const { t } = useTranslation()
  const startTab = useFinancePrefsStore((s) => s.startTab)
  const setStartTab = useFinancePrefsStore((s) => s.setStartTab)

  return (
    <div className="space-y-1.5">
      <Label className="text-sm font-medium text-foreground">{t('finanzen.prefs.startTab.label')}</Label>
      <Select value={startTab} onValueChange={(v) => setStartTab(v as FinanceStartTab)}>
        <SelectTrigger className="max-w-xs">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {START_TAB_OPTIONS.map((o) => (
            <SelectItem key={o.value} value={o.value}>
              {t(o.labelKey)}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <p className="text-xs text-muted-foreground">{t('finanzen.prefs.startTab.hint')}</p>
    </div>
  )
}

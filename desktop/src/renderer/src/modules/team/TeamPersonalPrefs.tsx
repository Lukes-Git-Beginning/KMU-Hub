/**
 * Personal team preferences (personal scope in the module-settings shell):
 * start tab + default member view. Both wired into TeamPage.
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
import { useTeamPrefsStore, type TeamStartTab, type TeamView } from '@/stores/teamPrefs'

const START_TAB_OPTIONS: { value: TeamStartTab; labelKey: string }[] = [
  { value: 'last', labelKey: 'team.prefs.startTab.last' },
  { value: 'members', labelKey: 'team.prefs.startTab.members' },
  { value: 'requests', labelKey: 'team.prefs.startTab.requests' },
  { value: 'absences', labelKey: 'team.prefs.startTab.absences' },
  { value: 'lohnvorbereitung', labelKey: 'team.prefs.startTab.payroll' },
]

const VIEW_OPTIONS: { value: TeamView; labelKey: string }[] = [
  { value: 'grid', labelKey: 'team.prefs.view.grid' },
  { value: 'list', labelKey: 'team.prefs.view.list' },
]

export function TeamPersonalPrefs() {
  const { t } = useTranslation()
  const { startTab, defaultView, setStartTab, setDefaultView } = useTeamPrefsStore()

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
      <div className="space-y-1.5">
        <Label className="text-sm font-medium text-foreground">{t('team.prefs.startTab.label')}</Label>
        <Select value={startTab} onValueChange={(v) => setStartTab(v as TeamStartTab)}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {START_TAB_OPTIONS.map((o) => (
              <SelectItem key={o.value} value={o.value}>{t(o.labelKey)}</SelectItem>
            ))}
          </SelectContent>
        </Select>
        <p className="text-xs text-muted-foreground">{t('team.prefs.startTab.hint')}</p>
      </div>
      <div className="space-y-1.5">
        <Label className="text-sm font-medium text-foreground">{t('team.prefs.view.label')}</Label>
        <Select value={defaultView} onValueChange={(v) => setDefaultView(v as TeamView)}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {VIEW_OPTIONS.map((o) => (
              <SelectItem key={o.value} value={o.value}>{t(o.labelKey)}</SelectItem>
            ))}
          </SelectContent>
        </Select>
        <p className="text-xs text-muted-foreground">{t('team.prefs.view.hint')}</p>
      </div>
    </div>
  )
}

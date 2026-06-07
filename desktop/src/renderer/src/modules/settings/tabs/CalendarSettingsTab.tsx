import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Save, Calendar, Eye, Clock } from 'lucide-react'
import { toast } from 'sonner'
import { useSettingsStore } from '@/stores/settings'
import { ModuleSettingsShell, type ModuleSettingsSection } from '@/components/shared'

const HOLIDAY_REGIONS = [
  // Deutschland — alle 16 Bundesländer
  { id: 'DE-BW', label: 'Baden-Württemberg' },
  { id: 'DE-BY', label: 'Bayern' },
  { id: 'DE-BE', label: 'Berlin' },
  { id: 'DE-BB', label: 'Brandenburg' },
  { id: 'DE-HB', label: 'Bremen' },
  { id: 'DE-HH', label: 'Hamburg' },
  { id: 'DE-HE', label: 'Hessen' },
  { id: 'DE-MV', label: 'Mecklenburg-Vorpommern' },
  { id: 'DE-NI', label: 'Niedersachsen' },
  { id: 'DE-NW', label: 'Nordrhein-Westfalen' },
  { id: 'DE-RP', label: 'Rheinland-Pfalz' },
  { id: 'DE-SL', label: 'Saarland' },
  { id: 'DE-SN', label: 'Sachsen' },
  { id: 'DE-ST', label: 'Sachsen-Anhalt' },
  { id: 'DE-SH', label: 'Schleswig-Holstein' },
  { id: 'DE-TH', label: 'Thüringen' },
  // Österreich
  { id: 'AT-B', label: 'Burgenland' },
  { id: 'AT-K', label: 'Kärnten' },
  { id: 'AT-NÖ', label: 'Niederösterreich' },
  { id: 'AT-OÖ', label: 'Oberösterreich' },
  { id: 'AT-S', label: 'Salzburg' },
  { id: 'AT-ST', label: 'Steiermark' },
  { id: 'AT-T', label: 'Tirol' },
  { id: 'AT-V', label: 'Vorarlberg' },
  { id: 'AT-W', label: 'Wien' },
  // Schweiz
  { id: 'CH-ZH', label: 'Zürich' },
  { id: 'CH-BE', label: 'Bern' },
  { id: 'CH-LU', label: 'Luzern' },
  { id: 'CH-SG', label: 'St. Gallen' },
  { id: 'CH-AG', label: 'Aargau' },
  { id: 'CH-BS', label: 'Basel-Stadt' },
  { id: 'CH-GE', label: 'Genf' },
]

const REMINDER_KEYS = [
  { value: 0, key: 'settings.calendar.reminder.none' },
  { value: 5, key: 'settings.calendar.reminder.5min' },
  { value: 10, key: 'settings.calendar.reminder.10min' },
  { value: 15, key: 'settings.calendar.reminder.15min' },
  { value: 30, key: 'settings.calendar.reminder.30min' },
  { value: 60, key: 'settings.calendar.reminder.1hour' },
  { value: 1440, key: 'settings.calendar.reminder.1day' },
]

export function CalendarSettingsTab() {
  const { t } = useTranslation()
  const { calendar, updateCalendar } = useSettingsStore()

  const [defaultView, setDefaultView] = useState(calendar.defaultView)
  const [workStart, setWorkStart] = useState(calendar.workStartHour)
  const [workEnd, setWorkEnd] = useState(calendar.workEndHour)
  const [reminder, setReminder] = useState(calendar.defaultReminder)
  const [region, setRegion] = useState(calendar.holidayRegion)
  const [weekStart, setWeekStart] = useState(calendar.weekStartsOn)

  const handleSave = () => {
    updateCalendar({
      defaultView,
      workStartHour: workStart,
      workEndHour: workEnd,
      defaultReminder: reminder,
      holidayRegion: region,
      weekStartsOn: weekStart,
    })
    toast.success(t('settings.calendar.saved'))
  }

  const sections: ModuleSettingsSection[] = [
    {
      id: 'personal',
      titleKey: 'settings.calendar.section.personalTitle',
      descriptionKey: 'settings.calendar.section.personalDesc',
      scope: 'personal',
      icon: Eye,
      children: (
        <div className="space-y-6">
          {/* Default view */}
          <div className="space-y-1.5">
            <Label>{t('settings.calendar.defaultView')}</Label>
            <div className="flex gap-2">
              {(['week', 'day', 'month'] as const).map((v) => {
                const labels = { week: t('settings.calendar.view.week'), day: t('settings.calendar.view.day'), month: t('settings.calendar.view.month') }
                return (
                  <button
                    key={v}
                    onClick={() => setDefaultView(v)}
                    className={`flex-1 rounded-lg border py-2 text-sm transition-colors ${
                      defaultView === v
                        ? 'border-primary bg-primary/5 text-primary font-medium'
                        : 'border-border text-foreground hover:bg-secondary'
                    }`}
                  >
                    {labels[v]}
                  </button>
                )
              })}
            </div>
          </div>

          {/* Week start */}
          <div className="space-y-1.5">
            <Label>{t('settings.calendar.weekStartsOn')}</Label>
            <div className="flex gap-2">
              {(['monday', 'sunday'] as const).map((d) => (
                <button
                  key={d}
                  onClick={() => setWeekStart(d)}
                  className={`flex-1 rounded-lg border py-2 text-sm transition-colors ${
                    weekStart === d
                      ? 'border-primary bg-primary/5 text-primary font-medium'
                      : 'border-border text-foreground hover:bg-secondary'
                  }`}
                >
                  {d === 'monday' ? t('settings.calendar.weekDay.monday') : t('settings.calendar.weekDay.sunday')}
                </button>
              ))}
            </div>
          </div>

          {/* Default reminder */}
          <div className="space-y-1.5">
            <Label>{t('settings.calendar.defaultReminder')}</Label>
            <Select value={String(reminder)} onValueChange={(v) => setReminder(Number(v))}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                {REMINDER_KEYS.map((r) => (
                  <SelectItem key={r.value} value={String(r.value)}>{t(r.key)}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
      ),
    },
    {
      id: 'tenant',
      titleKey: 'settings.calendar.section.tenantTitle',
      descriptionKey: 'settings.calendar.section.tenantDesc',
      scope: 'tenant',
      icon: Clock,
      children: (
        <div className="space-y-6">
          {/* Work hours */}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label>{t('settings.calendar.workStart')}</Label>
              <Select value={String(workStart)} onValueChange={(v) => setWorkStart(Number(v))}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {Array.from({ length: 12 }, (_, i) => i + 5).map((h) => (
                    <SelectItem key={h} value={String(h)}>{`${String(h).padStart(2, '0')}:00`}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>{t('settings.calendar.workEnd')}</Label>
              <Select value={String(workEnd)} onValueChange={(v) => setWorkEnd(Number(v))}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {Array.from({ length: 12 }, (_, i) => i + 12).map((h) => (
                    <SelectItem key={h} value={String(h)}>{`${String(h).padStart(2, '0')}:00`}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Holiday region */}
          <div className="space-y-1.5">
            <div className="flex items-center gap-2">
              <Calendar className="h-4 w-4 text-muted-foreground" />
              <Label>{t('settings.calendar.holidayRegion')}</Label>
            </div>
            <Select value={region} onValueChange={setRegion}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                {HOLIDAY_REGIONS.map((r) => (
                  <SelectItem key={r.id} value={r.id}>{r.label}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className="text-[10px] text-muted-foreground">{t('settings.calendar.holidayRegionHint')}</p>
          </div>
        </div>
      ),
    },
  ]

  return (
    <ModuleSettingsShell
      moduleId="calendar"
      titleKey="settings.calendar.title"
      descriptionKey="settings.calendar.subtitle"
      sections={sections}
      footer={
        <Button onClick={handleSave} size="sm">
          <Save className="mr-1.5 h-4 w-4" />
          {t('settings.calendar.saveButton')}
        </Button>
      }
    />
  )
}

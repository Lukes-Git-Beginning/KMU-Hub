import { useState } from 'react'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Save, Calendar } from 'lucide-react'
import { toast } from 'sonner'
import { useSettingsStore } from '@/stores/settings'

const HOLIDAY_REGIONS = [
  { id: 'CH-ZH', label: 'Zuerich' },
  { id: 'CH-BE', label: 'Bern' },
  { id: 'CH-LU', label: 'Luzern' },
  { id: 'CH-SG', label: 'St. Gallen' },
  { id: 'CH-AG', label: 'Aargau' },
  { id: 'CH-BS', label: 'Basel-Stadt' },
  { id: 'CH-GE', label: 'Genf' },
  { id: 'DE-BY', label: 'Bayern' },
  { id: 'DE-BW', label: 'Baden-Wuerttemberg' },
  { id: 'DE-NW', label: 'Nordrhein-Westfalen' },
  { id: 'AT-W', label: 'Wien' },
  { id: 'AT-T', label: 'Tirol' },
]

const REMINDERS = [
  { value: 0, label: 'Keine' },
  { value: 5, label: '5 Minuten vorher' },
  { value: 10, label: '10 Minuten vorher' },
  { value: 15, label: '15 Minuten vorher' },
  { value: 30, label: '30 Minuten vorher' },
  { value: 60, label: '1 Stunde vorher' },
  { value: 1440, label: '1 Tag vorher' },
]

export function CalendarSettingsTab() {
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
    toast.success('Kalender-Einstellungen gespeichert')
  }

  return (
    <div className="max-w-2xl">
      <h2 className="text-foreground mb-1">Kalender</h2>
      <p className="text-sm text-muted-foreground mb-6">Standard-Ansicht, Arbeitszeiten und Feiertage konfigurieren</p>

      <div className="space-y-6">
        {/* Default view */}
        <div className="space-y-1.5">
          <Label>Standard-Ansicht</Label>
          <div className="flex gap-2">
            {(['week', 'day', 'month'] as const).map((v) => {
              const labels = { week: 'Woche', day: 'Tag', month: 'Monat' }
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
          <Label>Woche beginnt am</Label>
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
                {d === 'monday' ? 'Montag' : 'Sonntag'}
              </button>
            ))}
          </div>
        </div>

        {/* Work hours */}
        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-1.5">
            <Label>Arbeitsbeginn</Label>
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
            <Label>Arbeitsende</Label>
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

        {/* Default reminder */}
        <div className="space-y-1.5">
          <Label>Standard-Erinnerung</Label>
          <Select value={String(reminder)} onValueChange={(v) => setReminder(Number(v))}>
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent>
              {REMINDERS.map((r) => (
                <SelectItem key={r.value} value={String(r.value)}>{r.label}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {/* Holiday region */}
        <div className="space-y-1.5">
          <div className="flex items-center gap-2">
            <Calendar className="h-4 w-4 text-muted-foreground" />
            <Label>Feiertags-Region</Label>
          </div>
          <Select value={region} onValueChange={setRegion}>
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent>
              {HOLIDAY_REGIONS.map((r) => (
                <SelectItem key={r.id} value={r.id}>{r.label}</SelectItem>
              ))}
            </SelectContent>
          </Select>
          <p className="text-[10px] text-muted-foreground">Bestimmt welche Feiertage im Kalender angezeigt werden</p>
        </div>
      </div>

      <Button onClick={handleSave} className="mt-6" size="sm">
        <Save className="mr-1.5 h-4 w-4" />
        Kalender-Einstellungen speichern
      </Button>
    </div>
  )
}

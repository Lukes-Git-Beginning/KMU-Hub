/**
 * 5.8 — Business Hours Configuration
 *
 * Modal dialog for configuring support operating hours,
 * holidays, and timezone.
 */
import { useState } from 'react'
import {
  Clock,
  Plus,
  Trash2,
  ChevronDown,
  Calendar,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  useHelpdeskStore,
  type BusinessDay,
  type Holiday,
  MOCK_BUSINESS_HOURS,
  MOCK_HOLIDAYS,
} from '@/stores/helpdesk'

// ---------------------------------------------------------------------------
// Timezone options (DACH)
// ---------------------------------------------------------------------------

const TIMEZONE_OPTIONS = [
  'Europe/Zurich',
  'Europe/Berlin',
  'Europe/Vienna',
  'Europe/Luxembourg',
  'Europe/Amsterdam',
  'Europe/London',
  'UTC',
] as const

interface BusinessHoursDialogProps {
  open: boolean
  onClose: () => void
}

export function BusinessHoursDialog({ open, onClose }: BusinessHoursDialogProps) {
  const { businessHours, holidays } = useHelpdeskStore()

  // Local editable state (clone from store/mock)
  const [hours, setHours] = useState<BusinessDay[]>(() =>
    (businessHours.length > 0 ? businessHours : MOCK_BUSINESS_HOURS).map((d) => ({ ...d }))
  )
  const [holidayList, setHolidayList] = useState<Holiday[]>(() =>
    (holidays.length > 0 ? holidays : MOCK_HOLIDAYS).map((h) => ({ ...h }))
  )
  const [timezone, setTimezone] = useState('Europe/Zurich')

  // New holiday form
  const [newHolidayDate, setNewHolidayDate] = useState('')
  const [newHolidayName, setNewHolidayName] = useState('')

  const toggleDay = (idx: number) => {
    setHours((prev) =>
      prev.map((d, i) => (i === idx ? { ...d, active: !d.active } : d))
    )
  }

  const updateTime = (idx: number, field: 'start' | 'end', value: string) => {
    setHours((prev) =>
      prev.map((d, i) => (i === idx ? { ...d, [field]: value } : d))
    )
  }

  const addHoliday = () => {
    if (!newHolidayDate || !newHolidayName.trim()) {
      toast.error('Bitte Datum und Name eingeben')
      return
    }
    const id = `h-${Date.now()}`
    setHolidayList((prev) => [...prev, { id, date: newHolidayDate, name: newHolidayName.trim() }])
    setNewHolidayDate('')
    setNewHolidayName('')
  }

  const removeHoliday = (id: string) => {
    setHolidayList((prev) => prev.filter((h) => h.id !== id))
  }

  const handleSave = () => {
    toast.success('Geschaeftszeiten gespeichert')
    onClose()
  }

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) onClose() }}>
      <DialogContent className="gap-0 p-0 max-w-2xl max-h-[85vh] flex flex-col">
        {/* Header */}
        <DialogHeader className="border-b border-border px-6 py-4 shrink-0">
          <div className="flex items-center gap-2">
            <Clock className="h-4 w-4 text-primary" />
            <DialogTitle className="text-base font-semibold text-foreground">Geschaeftszeiten konfigurieren</DialogTitle>
          </div>
          <DialogDescription className="sr-only">Oeffnungszeiten und Feiertage konfigurieren</DialogDescription>
        </DialogHeader>

        {/* Body */}
        <div className="flex-1 overflow-y-auto px-6 py-5 space-y-6">
          {/* Timezone */}
          <div>
            <label className="mb-1.5 block text-xs font-medium text-foreground">Zeitzone</label>
            <div className="relative w-64">
              <select
                value={timezone}
                onChange={(e) => setTimezone(e.target.value)}
                className="w-full appearance-none rounded-lg border border-border bg-card px-3 pr-8 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
              >
                {TIMEZONE_OPTIONS.map((tz) => (
                  <option key={tz} value={tz}>{tz}</option>
                ))}
              </select>
              <ChevronDown className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            </div>
          </div>

          {/* Business hours table */}
          <div>
            <h3 className="text-[10px] uppercase tracking-wider text-muted-foreground mb-2">Oeffnungszeiten</h3>
            <div className="rounded-lg border border-border overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-secondary/30">
                    <th className="px-4 py-2 text-left text-xs font-medium text-muted-foreground w-36">Tag</th>
                    <th className="px-4 py-2 text-center text-xs font-medium text-muted-foreground w-20">Aktiv</th>
                    <th className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">Von</th>
                    <th className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">Bis</th>
                  </tr>
                </thead>
                <tbody>
                  {hours.map((day, idx) => (
                    <tr key={day.day} className="border-b border-border-muted last:border-0">
                      <td className="px-4 py-2.5 text-foreground font-medium">{day.day}</td>
                      <td className="px-4 py-2.5 text-center">
                        <button
                          onClick={() => toggleDay(idx)}
                          className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
                            day.active ? 'bg-primary' : 'bg-secondary'
                          }`}
                        >
                          <span
                            className={`inline-block h-3.5 w-3.5 rounded-full bg-white transition-transform ${
                              day.active ? 'translate-x-[18px]' : 'translate-x-1'
                            }`}
                          />
                        </button>
                      </td>
                      <td className="px-4 py-2.5">
                        <input
                          type="time"
                          value={day.start}
                          onChange={(e) => updateTime(idx, 'start', e.target.value)}
                          disabled={!day.active}
                          className="rounded-md border border-border bg-card px-2 py-1 text-xs text-foreground disabled:opacity-40 focus:outline-none focus:ring-1 focus:ring-focus-ring"
                        />
                      </td>
                      <td className="px-4 py-2.5">
                        <input
                          type="time"
                          value={day.end}
                          onChange={(e) => updateTime(idx, 'end', e.target.value)}
                          disabled={!day.active}
                          className="rounded-md border border-border bg-card px-2 py-1 text-xs text-foreground disabled:opacity-40 focus:outline-none focus:ring-1 focus:ring-focus-ring"
                        />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          {/* Holidays section */}
          <div>
            <h3 className="text-[10px] uppercase tracking-wider text-muted-foreground mb-2">Feiertage</h3>
            <div className="space-y-2 mb-3">
              {holidayList.length === 0 && (
                <p className="text-xs text-muted-foreground">Keine Feiertage konfiguriert.</p>
              )}
              {holidayList
                .sort((a, b) => a.date.localeCompare(b.date))
                .map((h) => (
                  <div
                    key={h.id}
                    className="flex items-center justify-between rounded-lg border border-border bg-card px-3 py-2"
                  >
                    <div className="flex items-center gap-3">
                      <Calendar className="h-3.5 w-3.5 text-muted-foreground" />
                      <span className="text-xs font-mono text-muted-foreground">
                        {new Date(h.date + 'T00:00:00').toLocaleDateString('de-CH')}
                      </span>
                      <span className="text-sm text-foreground">{h.name}</span>
                    </div>
                    <button
                      onClick={() => removeHoliday(h.id)}
                      className="rounded p-1 text-muted-foreground hover:bg-error-light hover:text-error transition-colors"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </div>
                ))}
            </div>

            {/* Add holiday */}
            <div className="flex items-end gap-2">
              <div className="flex-1">
                <label className="mb-1 block text-[10px] text-muted-foreground">Datum</label>
                <input
                  type="date"
                  value={newHolidayDate}
                  onChange={(e) => setNewHolidayDate(e.target.value)}
                  className="w-full rounded-lg border border-border bg-card px-3 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-focus-ring"
                />
              </div>
              <div className="flex-1">
                <label className="mb-1 block text-[10px] text-muted-foreground">Bezeichnung</label>
                <input
                  type="text"
                  value={newHolidayName}
                  onChange={(e) => setNewHolidayName(e.target.value)}
                  placeholder="z.B. Auffahrt"
                  className="w-full rounded-lg border border-border bg-card px-3 py-1.5 text-xs text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-1 focus:ring-focus-ring"
                />
              </div>
              <button
                onClick={addHoliday}
                className="flex items-center gap-1 rounded-lg border border-border px-3 py-1.5 text-xs text-muted-foreground hover:bg-secondary transition-colors shrink-0"
              >
                <Plus className="h-3.5 w-3.5" />
                Hinzufuegen
              </button>
            </div>
          </div>
        </div>

        {/* Footer */}
        <DialogFooter className="border-t border-border px-6 py-4 shrink-0">
          <button
            onClick={onClose}
            className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
          >
            Abbrechen
          </button>
          <button
            onClick={handleSave}
            className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            Speichern
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

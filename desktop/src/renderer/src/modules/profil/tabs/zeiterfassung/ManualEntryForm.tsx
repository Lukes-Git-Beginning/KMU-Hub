import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog'
import { useTimeTrackingStore } from '@/stores/timetracking'
import { minutesBetween, todayStr } from './time-utils'
import { toast } from 'sonner'

interface ManualEntryFormProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export default function ManualEntryForm({ open, onOpenChange }: ManualEntryFormProps) {
  const categories = useTimeTrackingStore((s) => s.categories)
  const addManualEntry = useTimeTrackingStore((s) => s.addManualEntry)

  const [date, setDate] = useState(todayStr())
  const [startTime, setStartTime] = useState('09:00')
  const [endTime, setEndTime] = useState('10:00')
  const [categoryId, setCategoryId] = useState(categories[0]?.id || '')
  const [description, setDescription] = useState('')

  const duration = minutesBetween(startTime, endTime)
  const isValid = date && startTime && endTime && categoryId && duration > 0

  const handleSubmit = () => {
    if (!isValid) {
      toast.error('Bitte alle Felder korrekt ausfuellen')
      return
    }
    if (date > todayStr()) {
      toast.error('Datum darf nicht in der Zukunft liegen')
      return
    }

    addManualEntry({
      categoryId,
      description,
      date,
      startTime,
      endTime,
      durationMinutes: duration,
    })

    toast.success('Eintrag hinzugefuegt')
    onOpenChange(false)
    setDescription('')
    setStartTime('09:00')
    setEndTime('10:00')
    setDate(todayStr())
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Manuelle Zeiterfassung</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">Datum</label>
            <Input
              type="date"
              value={date}
              max={todayStr()}
              onChange={(e) => setDate(e.target.value)}
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">Start</label>
              <Input
                type="time"
                value={startTime}
                onChange={(e) => setStartTime(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">Ende</label>
              <Input
                type="time"
                value={endTime}
                onChange={(e) => setEndTime(e.target.value)}
              />
            </div>
          </div>

          {duration > 0 && (
            <p className="text-sm text-muted-foreground">
              Dauer: <span className="font-medium text-foreground">{Math.floor(duration / 60)}h {duration % 60}m</span>
            </p>
          )}
          {duration <= 0 && startTime && endTime && (
            <p className="text-sm text-destructive">
              Endzeit muss nach der Startzeit liegen
            </p>
          )}

          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">Kategorie</label>
            <Select value={categoryId} onValueChange={setCategoryId}>
              <SelectTrigger>
                <SelectValue placeholder="Kategorie waehlen..." />
              </SelectTrigger>
              <SelectContent>
                {categories.map((cat) => (
                  <SelectItem key={cat.id} value={cat.id}>
                    <span className="flex items-center gap-2">
                      <span
                        className="h-2.5 w-2.5 rounded-full shrink-0"
                        style={{ backgroundColor: cat.color }}
                      />
                      {cat.name}
                    </span>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">Beschreibung</label>
            <Input
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Was hast du gemacht?"
              onKeyDown={(e) => e.key === 'Enter' && isValid && handleSubmit()}
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Abbrechen</Button>
          <Button onClick={handleSubmit} disabled={!isValid}>Eintrag erstellen</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

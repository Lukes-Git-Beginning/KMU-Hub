import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog'
import { useCreateTimeEntry, useTimeCategories } from '@/api/hooks/hr-hooks'
import { minutesBetween, todayStr } from './time-utils'

interface ManualEntryFormProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export default function ManualEntryForm({ open, onOpenChange }: ManualEntryFormProps) {
  const { t } = useTranslation()
  const { data: categories = [] } = useTimeCategories()
  const createEntry = useCreateTimeEntry()

  const [date, setDate] = useState(todayStr())
  const [startTime, setStartTime] = useState('09:00')
  const [endTime, setEndTime] = useState('10:00')
  const [categoryId, setCategoryId] = useState('')
  const [description, setDescription] = useState('')

  const duration = minutesBetween(startTime, endTime)
  const isValid = date && startTime && endTime && categoryId && duration > 0

  const handleSubmit = () => {
    if (!isValid) return
    if (date > todayStr()) {
      return
    }

    // Wire-Shape: clockIn/clockOut as ISO timestamps
    const clockIn = `${date}T${startTime}:00Z`
    const clockOut = `${date}T${endTime}:00Z`

    createEntry.mutate(
      {
        clockIn,
        clockOut,
        breakMinutes: 0,
        activity: description || undefined,
        note: description || undefined,
      },
      {
        onSuccess: () => {
          onOpenChange(false)
          setDescription('')
          setStartTime('09:00')
          setEndTime('10:00')
          setDate(todayStr())
        },
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('profil.zeiterfassung.manual.title')}</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('profil.zeiterfassung.manual.date')}</label>
            <Input
              type="date"
              value={date}
              max={todayStr()}
              onChange={(e) => setDate(e.target.value)}
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">{t('profil.zeiterfassung.manual.start')}</label>
              <Input
                type="time"
                value={startTime}
                onChange={(e) => setStartTime(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">{t('profil.zeiterfassung.manual.end')}</label>
              <Input
                type="time"
                value={endTime}
                onChange={(e) => setEndTime(e.target.value)}
              />
            </div>
          </div>

          {duration > 0 && (
            <p className="text-sm text-muted-foreground">
              {t('profil.zeiterfassung.manual.duration')}:{' '}
              <span className="font-medium text-foreground">
                {Math.floor(duration / 60)}h {duration % 60}m
              </span>
            </p>
          )}
          {duration <= 0 && startTime && endTime && (
            <p className="text-sm text-destructive">
              {t('profil.zeiterfassung.manual.endAfterStart')}
            </p>
          )}

          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('profil.zeiterfassung.manual.category')}</label>
            <Select
              value={categoryId}
              onValueChange={setCategoryId}
            >
              <SelectTrigger>
                <SelectValue placeholder={t('profil.zeiterfassung.manual.selectCategory')} />
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
            <label className="text-sm font-medium text-foreground">{t('profil.zeiterfassung.manual.description')}</label>
            <Input
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder={t('profil.zeiterfassung.manual.descriptionPlaceholder')}
              onKeyDown={(e) => e.key === 'Enter' && isValid && handleSubmit()}
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>{t('common.cancel')}</Button>
          <Button
            onClick={handleSubmit}
            disabled={!isValid || createEntry.isPending}
          >
            {t('profil.zeiterfassung.manual.createEntry')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

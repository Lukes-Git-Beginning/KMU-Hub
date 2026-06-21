import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { cn } from '@/lib/cn'
import type { CallOutcome } from '@/api/dialer-client'

const presetColors = [
  '#22c55e', '#ef4444', '#f59e0b', '#3b82f6',
  '#8b5cf6', '#ec4899', '#14b8a6', '#f97316',
]

interface OutcomeFormDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSubmit: (data: {
    label: string
    color: string
    is_positive: boolean
    is_callback: boolean
    is_appointment: boolean
    sort_order: number
  }) => void
  outcome?: CallOutcome | null
  isLoading?: boolean
  nextSortOrder: number
}

export default function OutcomeFormDialog({
  open,
  onOpenChange,
  onSubmit,
  outcome,
  isLoading,
  nextSortOrder,
}: OutcomeFormDialogProps) {
  const { t } = useTranslation()
  const [label, setLabel] = useState('')
  const [color, setColor] = useState('#22c55e')
  const [isPositive, setIsPositive] = useState(false)
  const [isCallback, setIsCallback] = useState(false)
  const [isAppointment, setIsAppointment] = useState(false)

  const isEdit = !!outcome

  useEffect(() => {
    if (open && outcome) {
      setLabel(outcome.label)
      setColor(outcome.color)
      setIsPositive(outcome.is_positive)
      setIsCallback(outcome.is_callback)
      setIsAppointment(outcome.is_appointment)
    } else if (open) {
      setLabel('')
      setColor('#22c55e')
      setIsPositive(false)
      setIsCallback(false)
      setIsAppointment(false)
    }
  }, [open, outcome])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    onSubmit({
      label: label.trim(),
      color,
      is_positive: isPositive,
      is_callback: isCallback,
      is_appointment: isAppointment,
      sort_order: outcome?.sort_order ?? nextSortOrder,
    })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>
            {isEdit ? t('dialer.campaign.actions.edit') : t('dialer.settings.outcomes.new')}
          </DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-5 pt-2">
          {/* Label */}
          <div className="space-y-1.5">
            <Label>{t('dialer.settings.outcomes.label')}</Label>
            <Input
              value={label}
              onChange={(e) => setLabel(e.target.value)}
              placeholder={t('dialer.settings.outcomes.labelPlaceholder')}
              autoFocus
            />
          </div>

          {/* Color */}
          <div className="space-y-1.5">
            <Label>{t('dialer.settings.outcomes.color')}</Label>
            <div className="flex items-center gap-2">
              {presetColors.map((c) => (
                <button
                  key={c}
                  type="button"
                  onClick={() => setColor(c)}
                  className={cn(
                    'h-8 w-8 rounded-lg transition-all',
                    color === c ? 'ring-2 ring-offset-2 ring-primary scale-110' : 'hover:scale-105',
                  )}
                  style={{ backgroundColor: c }}
                />
              ))}
              <label className="relative h-8 w-8 rounded-lg border-2 border-dashed border-border cursor-pointer hover:border-primary transition-colors overflow-hidden">
                <input
                  type="color"
                  value={color}
                  onChange={(e) => setColor(e.target.value)}
                  className="absolute inset-0 opacity-0 cursor-pointer"
                />
                <span className="absolute inset-0 flex items-center justify-center text-xs text-muted-foreground">
                  +
                </span>
              </label>
            </div>
          </div>

          {/* Flags */}
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <Label>{t('dialer.settings.outcomes.isPositive')}</Label>
              <Switch checked={isPositive} onCheckedChange={setIsPositive} />
            </div>
            <div className="flex items-center justify-between">
              <Label>{t('dialer.settings.outcomes.isCallback')}</Label>
              <Switch checked={isCallback} onCheckedChange={setIsCallback} />
            </div>
            <div className="flex items-center justify-between">
              <Label>{t('dialer.settings.outcomes.isAppointment')}</Label>
              <Switch checked={isAppointment} onCheckedChange={setIsAppointment} />
            </div>
          </div>

          {/* Preview */}
          <div className="flex items-center gap-3 rounded-lg border p-3">
            <div className="h-4 w-4 rounded-md" style={{ backgroundColor: color }} />
            <span className="text-sm font-medium">{label || '...'}</span>
          </div>

          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>
              {t('common.cancel')}
            </Button>
            <Button type="submit" disabled={!label.trim() || isLoading} loading={isLoading}>
              {isEdit ? t('common.save') : t('dialer.settings.outcomes.new')}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}

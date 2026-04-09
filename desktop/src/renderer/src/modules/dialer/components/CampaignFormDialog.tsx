import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Eye, Zap, Brain } from 'lucide-react'
import { cn } from '@/lib/cn'
import type { Campaign } from '@/api/dialer-client'

interface CampaignFormDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSubmit: (data: { name: string; description?: string; mode: number }) => void
  campaign?: Campaign | null
  isLoading?: boolean
}

const modes = [
  { value: 1, icon: Eye, labelKey: 'dialer.campaign.mode.preview', color: 'var(--info)' },
  { value: 2, icon: Zap, labelKey: 'dialer.campaign.mode.power', color: 'var(--warning)', disabled: true },
  { value: 3, icon: Brain, labelKey: 'dialer.campaign.mode.predictive', color: 'var(--accent-1)', disabled: true },
]

export default function CampaignFormDialog({
  open,
  onOpenChange,
  onSubmit,
  campaign,
  isLoading,
}: CampaignFormDialogProps) {
  const { t } = useTranslation()
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [mode, setMode] = useState(1)

  const isEdit = !!campaign

  useEffect(() => {
    if (open && campaign) {
      setName(campaign.name)
      setDescription(campaign.description ?? '')
      setMode(campaign.mode)
    } else if (open) {
      setName('')
      setDescription('')
      setMode(1)
    }
  }, [open, campaign])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    onSubmit({ name: name.trim(), description: description.trim() || undefined, mode })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>
            {isEdit ? t('dialer.campaign.actions.edit') : t('dialer.campaigns.new')}
          </DialogTitle>
          <DialogDescription>
            {t('dialer.campaigns.description')}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4 pt-2">
          <div className="space-y-1.5">
            <Label htmlFor="campaign-name">{t('dialer.settings.outcomes.label')}</Label>
            <Input
              id="campaign-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="z.B. Kalt-Akquise Q2/2026"
              autoFocus
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="campaign-description">Beschreibung</Label>
            <Textarea
              id="campaign-description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Kurze Beschreibung der Kampagne..."
              rows={3}
            />
          </div>

          {/* Mode selector */}
          <div className="space-y-1.5">
            <Label>Modus</Label>
            <div className="grid grid-cols-3 gap-2">
              {modes.map((m) => {
                const Icon = m.icon
                const isSelected = mode === m.value
                return (
                  <button
                    key={m.value}
                    type="button"
                    disabled={m.disabled}
                    className={cn(
                      'relative flex flex-col items-center gap-1.5 rounded-lg border p-3 text-xs font-medium transition-all',
                      isSelected
                        ? 'border-2 shadow-sm'
                        : 'border-border hover:border-border/80',
                      m.disabled && 'opacity-40 cursor-not-allowed',
                    )}
                    style={
                      isSelected
                        ? {
                            borderColor: m.color,
                            backgroundColor: `color-mix(in srgb, ${m.color} 8%, transparent)`,
                            color: m.color,
                          }
                        : undefined
                    }
                    onClick={() => !m.disabled && setMode(m.value)}
                  >
                    <Icon className="h-5 w-5" />
                    <span>{t(m.labelKey)}</span>
                    {m.disabled && (
                      <span className="absolute -top-1.5 -right-1.5 rounded-full bg-muted px-1.5 py-0.5 text-[9px] font-bold text-muted-foreground">
                        Soon
                      </span>
                    )}
                  </button>
                )
              })}
            </div>
          </div>

          <div className="flex justify-end gap-2 pt-2">
            <Button
              type="button"
              variant="ghost"
              onClick={() => onOpenChange(false)}
            >
              Abbrechen
            </Button>
            <Button type="submit" disabled={!name.trim() || isLoading} loading={isLoading}>
              {isEdit ? t('dialer.campaign.actions.edit') : t('dialer.campaigns.new')}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}

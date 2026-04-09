import { useTranslation } from 'react-i18next'
import { CheckCircle2, RotateCcw, CalendarCheck, Pencil, Trash2 } from 'lucide-react'
import { Switch } from '@/components/ui/switch'
import { ItemActions } from '@/components/shared'
import type { CallOutcome } from '@/api/dialer-client'

interface OutcomeCardProps {
  outcome: CallOutcome
  onEdit: (outcome: CallOutcome) => void
  onDelete: (id: string) => void
  onToggleActive: (id: string, isActive: boolean) => void
}

export default function OutcomeCard({
  outcome,
  onEdit,
  onDelete,
  onToggleActive,
}: OutcomeCardProps) {
  const { t } = useTranslation()

  const flags = [
    { active: outcome.is_positive, icon: CheckCircle2, label: t('dialer.settings.outcomes.isPositive') },
    { active: outcome.is_callback, icon: RotateCcw, label: t('dialer.settings.outcomes.isCallback') },
    { active: outcome.is_appointment, icon: CalendarCheck, label: t('dialer.settings.outcomes.isAppointment') },
  ]

  return (
    <div className="flex items-center gap-4 rounded-xl border bg-card p-4 hover-lift transition-all">
      {/* Color swatch */}
      <div
        className="h-5 w-5 rounded-md shrink-0 shadow-sm"
        style={{ backgroundColor: outcome.color }}
      />

      {/* Label */}
      <span className="text-sm font-semibold flex-1 min-w-0 truncate">
        {outcome.label}
      </span>

      {/* Flags */}
      <div className="flex items-center gap-2">
        {flags.map(({ active, icon: Icon, label }) => (
          <span
            key={label}
            title={label}
            className="flex h-7 w-7 items-center justify-center rounded-md"
            style={
              active
                ? {
                    backgroundColor: `color-mix(in srgb, ${outcome.color} 15%, transparent)`,
                    color: outcome.color,
                  }
                : { color: 'var(--text-disabled)' }
            }
          >
            <Icon className="h-3.5 w-3.5" />
          </span>
        ))}
      </div>

      {/* Active toggle */}
      <Switch
        checked={outcome.is_active}
        onCheckedChange={(checked) => onToggleActive(outcome.id, checked)}
      />

      {/* Actions */}
      <ItemActions
        items={[
          {
            label: t('dialer.campaign.actions.edit'),
            icon: Pencil,
            onClick: () => onEdit(outcome),
          },
          {
            label: t('dialer.settings.outcomes.delete'),
            icon: Trash2,
            onClick: () => onDelete(outcome.id),
            variant: 'destructive' as const,
          },
        ]}
      />
    </div>
  )
}

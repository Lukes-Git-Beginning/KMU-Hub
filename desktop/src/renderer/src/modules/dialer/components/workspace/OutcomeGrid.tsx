import { useTranslation } from 'react-i18next'
import { CheckCircle2, XCircle, RotateCcw, CalendarCheck } from 'lucide-react'
import { cn } from '@/lib/cn'
import type { CallOutcome } from '@/api/dialer-client'

function getOutcomeIcon(outcome: CallOutcome) {
  if (outcome.is_appointment) return CalendarCheck
  if (outcome.is_callback) return RotateCcw
  if (outcome.is_positive) return CheckCircle2
  return XCircle
}

interface OutcomeGridProps {
  outcomes: CallOutcome[]
  selectedId: string | null
  onSelect: (id: string) => void
  className?: string
}

export default function OutcomeGrid({
  outcomes,
  selectedId,
  onSelect,
  className,
}: OutcomeGridProps) {
  const { t } = useTranslation()

  return (
    <div className={cn('space-y-3', className)}>
      <h3 className="text-xs font-semibold tracking-widest uppercase text-muted-foreground">
        {t('dialer.workspace.wrapUp.selectOutcome')}
      </h3>
      <div className="grid grid-cols-2 gap-3">
        {outcomes.map((outcome, idx) => {
          const Icon = getOutcomeIcon(outcome)
          const isSelected = selectedId === outcome.id

          return (
            <button
              key={outcome.id}
              onClick={() => onSelect(outcome.id)}
              className={cn(
                `relative flex items-center gap-3 rounded-xl border-2 p-4 text-left transition-all animate-fade-up stagger-${Math.min(idx + 1, 4)}`,
                isSelected
                  ? 'shadow-md ring-2'
                  : 'hover:shadow-sm',
              )}
              style={
                isSelected
                  ? {
                      backgroundColor: outcome.color,
                      borderColor: outcome.color,
                      color: '#fff',
                      // ring color
                      ['--tw-ring-color' as string]: `color-mix(in srgb, ${outcome.color} 50%, transparent)`,
                    }
                  : {
                      borderColor: `color-mix(in srgb, ${outcome.color} 30%, transparent)`,
                      backgroundColor: `color-mix(in srgb, ${outcome.color} 6%, transparent)`,
                    }
              }
            >
              <div
                className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg"
                style={
                  isSelected
                    ? { backgroundColor: 'rgba(255,255,255,0.2)' }
                    : { backgroundColor: `color-mix(in srgb, ${outcome.color} 15%, transparent)` }
                }
              >
                <Icon
                  className="h-5 w-5"
                  style={isSelected ? { color: '#fff' } : { color: outcome.color }}
                />
              </div>
              <div>
                <p className={cn('text-sm font-semibold', !isSelected && 'text-foreground')}>
                  {outcome.label}
                </p>
                {outcome.is_callback && (
                  <p
                    className={cn(
                      'text-xs mt-0.5',
                      isSelected ? 'text-white/70' : 'text-muted-foreground',
                    )}
                  >
                    Callback
                  </p>
                )}
                {outcome.is_appointment && (
                  <p
                    className={cn(
                      'text-xs mt-0.5',
                      isSelected ? 'text-white/70' : 'text-muted-foreground',
                    )}
                  >
                    Termin
                  </p>
                )}
              </div>
            </button>
          )
        })}
      </div>
    </div>
  )
}

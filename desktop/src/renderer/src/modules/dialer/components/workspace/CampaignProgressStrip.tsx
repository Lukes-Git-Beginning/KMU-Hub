import { useTranslation } from 'react-i18next'
import { PhoneCall, TrendingUp, Clock } from 'lucide-react'
import type { AgentDashboard } from '@/api/dialer-client'

interface CampaignProgressStripProps {
  dashboard: AgentDashboard | undefined
  className?: string
}

export default function CampaignProgressStrip({
  dashboard,
  className,
}: CampaignProgressStripProps) {
  const { t } = useTranslation()

  if (!dashboard) return null

  const avgMin = Math.floor(dashboard.average_duration_seconds / 60)
  const avgSec = Math.round(dashboard.average_duration_seconds % 60)

  return (
    <div className={`space-y-3 ${className ?? ''}`}>
      <h4 className="text-xs font-semibold tracking-widest uppercase text-muted-foreground">
        {t('dialer.workspace.idle.todayStats')}
      </h4>

      <div className="space-y-2">
        <div className="flex items-center gap-3 rounded-lg bg-card border border-border p-3">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10">
            <PhoneCall className="h-4 w-4 text-primary" />
          </div>
          <div>
            <p className="text-xl font-bold tabular-nums">{dashboard.total_calls_today}</p>
            <p className="text-xs text-muted-foreground">{t('dialer.dashboard.callsToday')}</p>
          </div>
        </div>

        <div className="flex items-center gap-3 rounded-lg bg-card border border-border p-3">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-success/10">
            <TrendingUp className="h-4 w-4 text-success" />
          </div>
          <div>
            <p className="text-xl font-bold tabular-nums">
              {dashboard.calls_per_hour.toFixed(1)}
            </p>
            <p className="text-xs text-muted-foreground">{t('dialer.dashboard.callsPerHour')}</p>
          </div>
        </div>

        <div className="flex items-center gap-3 rounded-lg bg-card border border-border p-3">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-warning/10">
            <Clock className="h-4 w-4 text-warning" />
          </div>
          <div>
            <p className="text-xl font-bold tabular-nums">
              {avgMin}:{String(avgSec).padStart(2, '0')}
            </p>
            <p className="text-xs text-muted-foreground">{t('dialer.dashboard.avgDuration')}</p>
          </div>
        </div>
      </div>

      {/* Recent outcomes */}
      {dashboard.outcome_breakdown.length > 0 && (
        <div className="space-y-1.5 pt-1">
          {dashboard.outcome_breakdown.map((entry) => (
            <div key={entry.outcome_id} className="flex items-center justify-between text-sm">
              <div className="flex items-center gap-2">
                <span
                  className="h-2 w-2 rounded-full shrink-0"
                  style={{ backgroundColor: entry.outcome_color }}
                />
                <span className="text-muted-foreground">{entry.outcome_label}</span>
              </div>
              <span className="font-medium tabular-nums">{entry.count}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

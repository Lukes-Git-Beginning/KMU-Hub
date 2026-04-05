import { useTranslation } from 'react-i18next'
import { TrendingUp } from 'lucide-react'

interface ProgressStat {
  labelKey: string
  current: number
  total: number
  percent: number
}

const stats: ProgressStat[] = [
  { labelKey: 'dashboard.stats.tasksCompleted', current: 12, total: 28, percent: 43 },
  { labelKey: 'dashboard.stats.projectProgress', current: 75, total: 100, percent: 75 },
]

export function QuickStatsSection() {
  const { t } = useTranslation()
  return (
    <div className="space-y-6">
      {/* Progress Card */}
      <div className="rounded-lg border border-border bg-card p-6">
        <div className="mb-4 flex items-center gap-3">
          <TrendingUp className="h-5 w-5 text-primary" />
          <h3 className="text-sm font-semibold text-foreground">
            {t('dashboard.stats.todayProgress')}
          </h3>
        </div>
        <div className="space-y-4">
          {stats.map((stat) => (
            <div key={stat.labelKey}>
              <div className="mb-2 flex items-center justify-between">
                <span className="text-xs text-muted-foreground">
                  {t(stat.labelKey)}
                </span>
                <span className="text-xs font-medium text-foreground">
                  {stat.current}/{stat.total}
                </span>
              </div>
              <div className="h-2 overflow-hidden rounded-full bg-accent">
                <div
                  className="h-full rounded-full bg-primary transition-all"
                  style={{ width: `${stat.percent}%` }}
                />
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Support CTA */}
      <div className="rounded-lg bg-gradient-to-br from-success to-success/80 p-6 text-success-foreground">
        <h3 className="mb-2 text-lg font-semibold">{t('dashboard.support.title')}</h3>
        <p className="mb-4 text-sm opacity-80">
          {t('dashboard.support.description')}
        </p>
        <button className="w-full rounded-lg bg-card px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-card/90">
          {t('dashboard.support.contact')}
        </button>
      </div>
    </div>
  )
}

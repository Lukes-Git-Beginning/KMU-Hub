import { TrendingUp } from 'lucide-react'

interface ProgressStat {
  label: string
  current: number
  total: number
  percent: number
}

const stats: ProgressStat[] = [
  { label: 'Aufgaben erledigt', current: 12, total: 28, percent: 43 },
  { label: 'Projekt-Fortschritt', current: 75, total: 100, percent: 75 },
]

export function QuickStatsSection() {
  return (
    <div className="space-y-6">
      {/* Progress Card */}
      <div className="rounded-lg border border-border bg-card p-6">
        <div className="mb-4 flex items-center gap-3">
          <TrendingUp className="h-5 w-5 text-primary" />
          <h3 className="text-sm font-semibold text-foreground">
            Fortschritt heute
          </h3>
        </div>
        <div className="space-y-4">
          {stats.map((stat) => (
            <div key={stat.label}>
              <div className="mb-2 flex items-center justify-between">
                <span className="text-xs text-muted-foreground">
                  {stat.label}
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
      <div className="rounded-lg bg-gradient-to-br from-emerald-500 to-emerald-600 p-6 text-white">
        <h3 className="mb-2 text-lg font-semibold">Benötigen Sie Hilfe?</h3>
        <p className="mb-4 text-sm text-emerald-100">
          Unser Support-Team steht Ihnen zur Verfuegung
        </p>
        <button className="w-full rounded-lg bg-white px-4 py-2 text-sm font-medium text-emerald-600 transition-colors hover:bg-emerald-50">
          Support kontaktieren
        </button>
      </div>
    </div>
  )
}

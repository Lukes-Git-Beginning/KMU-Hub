import { useTranslation } from 'react-i18next'
import { AlertCircle } from 'lucide-react'
import { Link } from 'react-router-dom'

interface Alert {
  id: number
  titleKey: string
  type: 'warning' | 'info'
  actionKey: string
  path: string
}

const alerts: Alert[] = [
  {
    id: 1,
    titleKey: 'dashboard.alerts.projectDeadline',
    type: 'warning',
    actionKey: 'dashboard.alerts.showProjects',
    path: '/work/projects',
  },
  {
    id: 2,
    titleKey: 'dashboard.alerts.financeIntegration',
    type: 'info',
    actionKey: 'dashboard.alerts.connectNow',
    path: '/finanzen',
  },
]

export function AlertsSection() {
  const { t } = useTranslation()
  if (alerts.length === 0) return null

  return (
    <div className="mb-8 grid grid-cols-1 gap-4 md:grid-cols-2">
      {alerts.map((alert, i) => (
        <div
          key={alert.id}
          className={`flex items-center justify-between rounded-xl border p-4 animate-fade-up stagger-${i + 1} ${
            alert.type === 'warning'
              ? 'border-warning bg-warning-light'
              : 'border-info bg-info-light'
          }`}
        >
          <div className="flex items-center gap-3">
            <AlertCircle
              className={`h-5 w-5 ${
                alert.type === 'warning'
                  ? 'text-warning-foreground'
                  : 'text-info-foreground'
              }`}
            />
            <span className="text-sm text-foreground">{t(alert.titleKey)}</span>
          </div>
          <Link
            to={alert.path}
            className={`shrink-0 text-sm font-medium hover:underline ${
              alert.type === 'warning'
                ? 'text-warning-foreground'
                : 'text-info-foreground'
            }`}
          >
            {t(alert.actionKey)}
          </Link>
        </div>
      ))}
    </div>
  )
}

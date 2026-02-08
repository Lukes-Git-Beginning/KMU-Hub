import { AlertCircle } from 'lucide-react'
import { Link } from 'react-router-dom'

interface Alert {
  id: number
  title: string
  type: 'warning' | 'info'
  action: string
  path: string
}

const alerts: Alert[] = [
  {
    id: 1,
    title: '3 Projekte mit Deadline diese Woche',
    type: 'warning',
    action: 'Projekte anzeigen',
    path: '/work/projects',
  },
  {
    id: 2,
    title: 'Buchhaltungs-Integration verfuegbar',
    type: 'info',
    action: 'Jetzt verbinden',
    path: '/finance',
  },
]

export function AlertsSection() {
  if (alerts.length === 0) return null

  return (
    <div className="mb-8 grid grid-cols-1 gap-4 md:grid-cols-2">
      {alerts.map((alert) => (
        <div
          key={alert.id}
          className={`flex items-center justify-between rounded-lg border p-4 ${
            alert.type === 'warning'
              ? 'border-yellow-200 bg-yellow-50 dark:border-yellow-800 dark:bg-yellow-900/20'
              : 'border-blue-200 bg-blue-50 dark:border-blue-800 dark:bg-blue-900/20'
          }`}
        >
          <div className="flex items-center gap-3">
            <AlertCircle
              className={`h-5 w-5 ${
                alert.type === 'warning'
                  ? 'text-yellow-600 dark:text-yellow-400'
                  : 'text-blue-600 dark:text-blue-400'
              }`}
            />
            <span className="text-sm text-foreground">{alert.title}</span>
          </div>
          <Link
            to={alert.path}
            className={`shrink-0 text-sm font-medium hover:underline ${
              alert.type === 'warning'
                ? 'text-yellow-700 dark:text-yellow-400'
                : 'text-blue-700 dark:text-blue-400'
            }`}
          >
            {alert.action}
          </Link>
        </div>
      ))}
    </div>
  )
}

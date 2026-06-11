import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { AlertCircle, AlertTriangle, FileWarning, Receipt, Info } from 'lucide-react'
import { useAlerts } from '@/hooks/useAlerts'
import type { AlertSeverity } from '@/hooks/useAlerts'

const SEVERITY_STYLES: Record<AlertSeverity, { wrapper: string; icon: string; link: string }> = {
  warning: {
    wrapper: 'border-warning bg-warning-light',
    icon: 'text-warning-foreground',
    link: 'text-warning-foreground',
  },
  critical: {
    wrapper: 'border-destructive/40 bg-destructive/10',
    icon: 'text-destructive',
    link: 'text-destructive',
  },
  info: {
    wrapper: 'border-info bg-info-light',
    icon: 'text-info-foreground',
    link: 'text-info-foreground',
  },
}

const ICON_MAP = {
  AlertCircle,
  AlertTriangle,
  FileWarning,
  Receipt,
  Info,
}

export function AlertsSection() {
  const { t } = useTranslation()
  const alerts = useAlerts()

  if (alerts.length === 0) return null

  return (
    <div className="mb-8 grid grid-cols-1 gap-4 md:grid-cols-2">
      {alerts.map((alert, i) => {
        const styles = SEVERITY_STYLES[alert.severity]
        const Icon = ICON_MAP[alert.iconName]
        return (
          <div
            key={alert.id}
            className={`flex items-center justify-between rounded-xl border p-4 animate-fade-up stagger-${i + 1} ${styles.wrapper}`}
          >
            <div className="flex items-center gap-3">
              <Icon className={`h-5 w-5 shrink-0 ${styles.icon}`} />
              <span className="text-sm text-foreground">
                {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
                {(t as any)(alert.titleKey, alert.titleVars ?? {})}
              </span>
            </div>
            <Link
              to={alert.path}
              className={`shrink-0 text-sm font-medium hover:underline ${styles.link}`}
            >
              {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
              {(t as any)(alert.actionKey)}
            </Link>
          </div>
        )
      })}
    </div>
  )
}

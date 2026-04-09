import { useTranslation } from 'react-i18next'
import { Eye, Zap, Brain } from 'lucide-react'
import { cn } from '@/lib/cn'

const modeConfig: Record<
  number,
  { key: string; icon: typeof Eye; color: string }
> = {
  1: { key: 'dialer.campaign.mode.preview', icon: Eye, color: 'var(--info)' },
  2: { key: 'dialer.campaign.mode.power', icon: Zap, color: 'var(--warning)' },
  3: { key: 'dialer.campaign.mode.predictive', icon: Brain, color: 'var(--accent-1)' },
}

interface CampaignModeBadgeProps {
  mode: number
  className?: string
}

export default function CampaignModeBadge({ mode, className }: CampaignModeBadgeProps) {
  const { t } = useTranslation()
  const config = modeConfig[mode] ?? modeConfig[1]
  const Icon = config.icon

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium border',
        className,
      )}
      style={{
        backgroundColor: `color-mix(in srgb, ${config.color} 12%, transparent)`,
        color: config.color,
        borderColor: `color-mix(in srgb, ${config.color} 30%, transparent)`,
      }}
    >
      <Icon className="h-3 w-3" />
      {t(config.key)}
    </span>
  )
}

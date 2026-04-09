import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/cn'

const statusConfig: Record<
  number,
  { key: string; className: string }
> = {
  1: { key: 'dialer.campaign.status.draft', className: 'bg-muted text-muted-foreground border-border' },
  2: { key: 'dialer.campaign.status.active', className: 'bg-success/15 text-success border-success/30' },
  3: { key: 'dialer.campaign.status.paused', className: 'bg-warning/15 text-warning border-warning/30' },
  4: { key: 'dialer.campaign.status.completed', className: 'bg-info/15 text-info border-info/30' },
  5: { key: 'dialer.campaign.status.archived', className: 'bg-muted/50 text-muted-foreground/60 border-border/50' },
}

interface CampaignStatusBadgeProps {
  status: number
  className?: string
}

export default function CampaignStatusBadge({ status, className }: CampaignStatusBadgeProps) {
  const { t } = useTranslation()
  const config = statusConfig[status] ?? statusConfig[1]

  return (
    <Badge
      variant="outline"
      className={cn('text-xs font-medium', config.className, className)}
    >
      {t(config.key)}
    </Badge>
  )
}

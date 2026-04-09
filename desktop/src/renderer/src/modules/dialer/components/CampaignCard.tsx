import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Play, Pause, MoreVertical, Archive, Pencil, Users } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import CampaignStatusBadge from './CampaignStatusBadge'
import CampaignModeBadge from './CampaignModeBadge'
import CampaignProgressBar from './CampaignProgressBar'
import type { Campaign } from '@/api/dialer-client'

interface CampaignCardProps {
  campaign: Campaign
  onStart?: (id: string) => void
  onPause?: (id: string) => void
  onEdit?: (campaign: Campaign) => void
  onArchive?: (id: string) => void
  className?: string
}

const modeBorderColor: Record<number, string> = {
  1: 'var(--info)',
  2: 'var(--warning)',
  3: 'var(--accent-1)',
}

export default function CampaignCard({
  campaign,
  onStart,
  onPause,
  onEdit,
  onArchive,
  className,
}: CampaignCardProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const borderColor = modeBorderColor[campaign.mode] ?? 'var(--info)'
  const isActive = campaign.status === 2
  const isPaused = campaign.status === 3
  const isDraft = campaign.status === 1

  return (
    <div
      className={`group relative rounded-xl border bg-card p-5 shadow-sm hover-lift cursor-pointer transition-all ${className ?? ''}`}
      style={{ borderLeftWidth: '3px', borderLeftColor: borderColor }}
      onClick={() => navigate(`/dialer/campaigns/${campaign.id}`)}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          navigate(`/dialer/campaigns/${campaign.id}`)
        }
      }}
      tabIndex={0}
      role="button"
    >
      <div className="flex gap-4">
        {/* Left — 70% */}
        <div className="flex-1 min-w-0 space-y-3">
          <div className="flex items-start gap-2">
            <h3 className="text-lg font-bold text-foreground truncate">{campaign.name}</h3>
            <CampaignStatusBadge status={campaign.status} className="shrink-0" />
          </div>

          {campaign.description && (
            <p className="text-sm text-muted-foreground line-clamp-2">
              {campaign.description}
            </p>
          )}

          <CampaignProgressBar
            completed={campaign.completed_count}
            callback={0}
            skipped={0}
            total={campaign.contact_count}
          />

          {/* Agent avatars */}
          {campaign.assigned_agent_ids.length > 0 && (
            <div className="flex items-center gap-1">
              <Users className="h-3.5 w-3.5 text-muted-foreground mr-1" />
              <div className="flex -space-x-1.5">
                {campaign.assigned_agent_ids.slice(0, 4).map((agentId) => (
                  <Avatar key={agentId} className="h-6 w-6 border-2 border-card">
                    <AvatarFallback className="text-[10px] font-medium bg-primary/10 text-primary">
                      {agentId.slice(-2).toUpperCase()}
                    </AvatarFallback>
                  </Avatar>
                ))}
                {campaign.assigned_agent_ids.length > 4 && (
                  <span className="flex h-6 w-6 items-center justify-center rounded-full border-2 border-card bg-muted text-[10px] font-medium text-muted-foreground">
                    +{campaign.assigned_agent_ids.length - 4}
                  </span>
                )}
              </div>
            </div>
          )}
        </div>

        {/* Right — action column */}
        <div
          className="flex flex-col items-end gap-2 shrink-0"
          onClick={(e) => e.stopPropagation()}
        >
          <CampaignModeBadge mode={campaign.mode} />

          <div className="flex items-center gap-1 mt-auto">
            {(isDraft || isPaused) && (
              <Button
                variant="default"
                size="sm"
                className="gap-1.5"
                onClick={() => onStart?.(campaign.id)}
                disabled={isDraft && campaign.contact_count === 0}
              >
                <Play className="h-3.5 w-3.5" />
                {t('dialer.campaign.actions.start')}
              </Button>
            )}
            {isActive && (
              <Button
                variant="outline"
                size="sm"
                className="gap-1.5"
                onClick={() => onPause?.(campaign.id)}
              >
                <Pause className="h-3.5 w-3.5" />
                {t('dialer.campaign.actions.pause')}
              </Button>
            )}

            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon" className="h-8 w-8">
                  <MoreVertical className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="menu-stagger">
                <DropdownMenuItem onClick={() => onEdit?.(campaign)}>
                  <Pencil className="mr-2 h-4 w-4" />
                  {t('dialer.campaign.actions.edit')}
                </DropdownMenuItem>
                {(campaign.status === 3 || campaign.status === 4) && (
                  <>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      className="text-destructive focus:text-destructive"
                      onClick={() => onArchive?.(campaign.id)}
                    >
                      <Archive className="mr-2 h-4 w-4" />
                      {t('dialer.campaign.actions.archive')}
                    </DropdownMenuItem>
                  </>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </div>
    </div>
  )
}

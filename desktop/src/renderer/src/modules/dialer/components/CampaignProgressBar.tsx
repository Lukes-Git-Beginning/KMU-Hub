import { cn } from '@/lib/cn'

interface CampaignProgressBarProps {
  completed: number
  callback: number
  skipped: number
  total: number
  className?: string
  showLabel?: boolean
}

export default function CampaignProgressBar({
  completed,
  callback,
  skipped,
  total,
  className,
  showLabel = true,
}: CampaignProgressBarProps) {
  if (total === 0) {
    return (
      <div className={cn('flex items-center gap-3', className)}>
        <div className="h-1.5 flex-1 rounded-full bg-border" />
        {showLabel && (
          <span className="text-xs text-muted-foreground whitespace-nowrap">0 / 0</span>
        )}
      </div>
    )
  }

  const pctCompleted = (completed / total) * 100
  const pctCallback = (callback / total) * 100
  const pctSkipped = (skipped / total) * 100
  const pctDone = Math.round(((completed + callback + skipped) / total) * 100)

  return (
    <div className={cn('flex items-center gap-3', className)}>
      <div className="relative h-1.5 flex-1 rounded-full bg-border overflow-hidden">
        {/* Completed (green) */}
        <div
          className="absolute inset-y-0 left-0 rounded-full bg-success transition-all duration-500"
          style={{ width: `${pctCompleted}%` }}
        />
        {/* Callback (amber) */}
        <div
          className="absolute inset-y-0 rounded-full bg-warning transition-all duration-500"
          style={{ left: `${pctCompleted}%`, width: `${pctCallback}%` }}
        />
        {/* Skipped (muted) */}
        <div
          className="absolute inset-y-0 rounded-full bg-muted-foreground/30 transition-all duration-500"
          style={{ left: `${pctCompleted + pctCallback}%`, width: `${pctSkipped}%` }}
        />
      </div>
      {showLabel && (
        <span className="text-xs font-medium text-muted-foreground whitespace-nowrap tabular-nums">
          {completed} / {total}
          <span className="ml-1 text-muted-foreground/60">({pctDone}%)</span>
        </span>
      )}
    </div>
  )
}

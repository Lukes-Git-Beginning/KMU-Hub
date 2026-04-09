import { useTranslation } from 'react-i18next'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { cn } from '@/lib/cn'

type AgentStatus = 1 | 2 | 3 | 4 | 5

const statusConfig: Record<AgentStatus, { key: string; color: string; dot: string }> = {
  1: { key: 'dialer.agentStatus.available', color: 'var(--success)', dot: 'bg-success' },
  2: { key: 'dialer.agentStatus.onCall', color: 'var(--info)', dot: 'bg-info' },
  3: { key: 'dialer.agentStatus.wrapUp', color: 'var(--warning)', dot: 'bg-warning' },
  4: { key: 'dialer.agentStatus.break', color: 'var(--accent-1)', dot: 'bg-accent-1' },
  5: { key: 'dialer.agentStatus.offline', color: 'var(--text-muted)', dot: 'bg-muted-foreground' },
}

// Statuses the user can manually switch to
const switchable: AgentStatus[] = [1, 4, 5]

interface AgentStatusPillProps {
  status: number
  onChange?: (newStatus: number) => void
  className?: string
}

export default function AgentStatusPill({ status, onChange, className }: AgentStatusPillProps) {
  const { t } = useTranslation()
  const cfg = statusConfig[(status as AgentStatus)] ?? statusConfig[5]

  const pill = (
    <button
      className={cn(
        'inline-flex items-center gap-2 rounded-full border px-4 py-2 text-sm font-medium transition-all hover:shadow-sm',
        className,
      )}
      style={{
        borderColor: `color-mix(in srgb, ${cfg.color} 40%, transparent)`,
        backgroundColor: `color-mix(in srgb, ${cfg.color} 8%, transparent)`,
        color: cfg.color,
      }}
    >
      <span
        className={cn('h-2.5 w-2.5 rounded-full', cfg.dot)}
        style={status === 2 ? { animation: 'pulse 2s infinite' } : undefined}
      />
      {t(cfg.key)}
    </button>
  )

  if (!onChange) return pill

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>{pill}</DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="menu-stagger">
        {switchable.map((s) => {
          const sCfg = statusConfig[s]
          return (
            <DropdownMenuItem
              key={s}
              onClick={() => onChange(s)}
              className="gap-2"
            >
              <span className={cn('h-2 w-2 rounded-full', sCfg.dot)} />
              {t(sCfg.key)}
            </DropdownMenuItem>
          )
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

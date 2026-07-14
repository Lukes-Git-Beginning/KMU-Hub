import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { PhoneCall, Clock, Target, Radio, ExternalLink, Phone } from 'lucide-react'
import { DetailModal } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/cn'
import { formatRelativeTime } from '@/lib/format'
import { agentStatusConfig, type AgentStatus } from './workspace/AgentStatusPill'
import type { SupervisorAgent, RecentCallEntry } from '@/api/dialer-client'

function formatDuration(totalSeconds: number): string {
  const mm = Math.floor(totalSeconds / 60)
  const ss = String(totalSeconds % 60).padStart(2, '0')
  return `${mm}:${ss}`
}

interface AgentDetailModalProps {
  agent: SupervisorAgent | null
  recentCalls: RecentCallEntry[]
  open: boolean
  onClose: () => void
}

export default function AgentDetailModal({
  agent,
  recentCalls,
  open,
  onClose,
}: AgentDetailModalProps) {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()

  if (!agent) return null

  const fullName = `${agent.first_name} ${agent.last_name}`
  const cfg = agentStatusConfig[(agent.status as AgentStatus)] ?? agentStatusConfig[5]
  const agentCalls = recentCalls.filter((c) => c.agent_name === fullName).slice(0, 5)

  const statusBadge = (
    <span
      className="inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium"
      style={{
        borderColor: `color-mix(in srgb, ${cfg.color} 40%, transparent)`,
        backgroundColor: `color-mix(in srgb, ${cfg.color} 8%, transparent)`,
        color: cfg.color,
      }}
    >
      <span
        className={cn('h-2 w-2 rounded-full', cfg.dot)}
        style={agent.status === 2 ? { animation: 'pulse 2s infinite' } : undefined}
      />
      {t(cfg.key)}
    </span>
  )

  return (
    <DetailModal
      open={open}
      onClose={onClose}
      title={fullName}
      subtitle={agent.active_campaign_name ?? t('dialer.workspace.idle.noActiveCampaign')}
      badge={statusBadge}
      footer={
        agent.campaign_id ? (
          <div className="flex justify-end">
            <Button
              variant="outline"
              size="sm"
              className="gap-2"
              onClick={() => navigate(`/dialer/campaigns/${agent.campaign_id}`)}
            >
              <ExternalLink className="h-4 w-4" />
              {t('dialer.supervisor.agentDetail.openCampaign')}
            </Button>
          </div>
        ) : undefined
      }
    >
      <div className="space-y-6">
        {/* Agent facts */}
        <div className="grid grid-cols-2 gap-3">
          <Fact
            icon={PhoneCall}
            label={t('dialer.dashboard.callsToday')}
            value={String(agent.calls_today)}
          />
          <Fact
            icon={Clock}
            label={t('dialer.dashboard.avgDuration')}
            value={formatDuration(agent.avg_duration_seconds)}
            mono
          />
          <Fact
            icon={Target}
            label={t('dialer.supervisor.agentDetail.activeCampaign')}
            value={agent.active_campaign_name ?? '—'}
          />
          <Fact
            icon={Radio}
            label={t('dialer.supervisor.agentDetail.statusSince')}
            value={formatRelativeTime(agent.since, i18n.language)}
          />
        </div>

        {/* Recent calls */}
        <section className="space-y-2">
          <h4 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
            {t('dialer.dashboard.recentCalls')}
          </h4>
          {agentCalls.length === 0 ? (
            <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-border py-8 text-center">
              <Phone className="mb-2 h-7 w-7 text-muted-foreground/40" />
              <p className="text-sm text-muted-foreground">
                {t('dialer.supervisor.agentDetail.noCalls')}
              </p>
            </div>
          ) : (
            <ul className="space-y-2">
              {agentCalls.map((call) => (
                <li
                  key={call.id}
                  className="flex items-start gap-3 rounded-lg border border-border bg-card px-3 py-2.5"
                >
                  <span
                    className="mt-1 h-2.5 w-2.5 shrink-0 rounded-full"
                    style={{ backgroundColor: call.outcome_color }}
                  />
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center justify-between gap-2">
                      <span className="truncate text-sm font-medium text-foreground">
                        {call.contact_name}
                      </span>
                      <span className="shrink-0 font-mono text-xs tabular-nums text-muted-foreground">
                        {formatDuration(call.duration_seconds)}
                      </span>
                    </div>
                    <p className="truncate text-xs text-muted-foreground">
                      {call.outcome_label ?? t('dialer.workspace.wrapUp.completeWithoutOutcome')}
                      {call.contact_company ? ` · ${call.contact_company}` : ''}
                      {' · '}
                      {formatRelativeTime(call.created_at, i18n.language)}
                    </p>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>
    </DetailModal>
  )
}

function Fact({
  icon: Icon,
  label,
  value,
  mono,
}: {
  icon: typeof PhoneCall
  label: string
  value: string
  mono?: boolean
}) {
  return (
    <div className="flex items-start gap-2 rounded-lg border border-border bg-card px-3 py-2">
      <Icon className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" />
      <div className="min-w-0">
        <p className="text-[11px] uppercase tracking-wide text-muted-foreground">{label}</p>
        <p className={`truncate text-sm font-medium text-foreground ${mono ? 'font-mono tabular-nums' : ''}`}>
          {value}
        </p>
      </div>
    </div>
  )
}

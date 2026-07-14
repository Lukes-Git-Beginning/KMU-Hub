import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Headset, Users, PhoneCall, CalendarCheck, Phone, ChevronRight } from 'lucide-react'
import { PageHeader, StatCard, EmptyState } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/cn'
import { formatRelativeTime } from '@/lib/format'
import { useSupervisorOverview } from '@/api/hooks/useDialer'
import { agentStatusConfig, type AgentStatus } from './components/workspace/AgentStatusPill'
import AgentDetailModal from './components/AgentDetailModal'
import type { SupervisorAgent, RecentCallEntry } from '@/api/dialer-client'

function formatDuration(totalSeconds: number): string {
  const mm = Math.floor(totalSeconds / 60)
  const ss = String(totalSeconds % 60).padStart(2, '0')
  return `${mm}:${ss}`
}

function initials(name: string): string {
  return name
    .split(' ')
    .map((p) => p.charAt(0))
    .join('')
    .slice(0, 2)
    .toUpperCase()
}

export default function SupervisorDashboardPage() {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const { data, isLoading } = useSupervisorOverview()
  const [selectedAgent, setSelectedAgent] = useState<SupervisorAgent | null>(null)

  if (isLoading) {
    return (
      <div className="p-6 space-y-6">
        <Skeleton className="h-12 w-48" />
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-28" />
          ))}
        </div>
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <Skeleton className="h-72" />
          <Skeleton className="h-72" />
        </div>
      </div>
    )
  }

  if (!data) return null

  const { agents, recent_calls, totals } = data

  return (
    <div className="p-6 space-y-6">
      <PageHeader
        title={t('dialer.supervisor.title')}
        description={t('dialer.supervisor.description')}
        icon={Headset}
        moduleId="dialer"
        actions={
          <Button variant="outline" className="gap-2" onClick={() => navigate('/dialer/workspace')}>
            <PhoneCall className="h-4 w-4" />
            {t('dialer.dashboard.openWorkspace')}
          </Button>
        }
      />

      {/* KPI row */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          label={t('dialer.supervisor.activeAgents')}
          value={totals.active_agents}
          icon={Users}
          hero
          className="animate-fade-up stagger-1"
        />
        <StatCard
          label={t('dialer.agentStatus.onCall')}
          value={totals.on_call}
          icon={Headset}
          hero
          className="animate-fade-up stagger-2"
        />
        <StatCard
          label={t('dialer.dashboard.callsToday')}
          value={totals.calls_today}
          icon={PhoneCall}
          hero
          className="animate-fade-up stagger-3"
        />
        <StatCard
          label={t('dialer.supervisor.appointmentsToday')}
          value={totals.appointments_today}
          icon={CalendarCheck}
          hero
          className="animate-fade-up stagger-4"
        />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Agents */}
        <Card className="animate-fade-up stagger-5">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              {t('dialer.supervisor.team')}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            {agents.length === 0 ? (
              <p className="py-8 text-center text-sm text-muted-foreground">
                {t('dialer.supervisor.noAgents')}
              </p>
            ) : (
              agents.map((agent) => (
                <AgentRow
                  key={agent.user_id}
                  agent={agent}
                  onOpen={() => setSelectedAgent(agent)}
                />
              ))
            )}
          </CardContent>
        </Card>

        {/* Recent calls */}
        <Card className="animate-fade-up stagger-6">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              {t('dialer.dashboard.recentCalls')}
            </CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            {recent_calls.length === 0 ? (
              <EmptyState icon={Phone} title={t('dialer.supervisor.noCalls')} />
            ) : (
              <ul className="divide-y divide-border">
                {recent_calls.map((call) => (
                  <RecentCallRow key={call.id} call={call} locale={i18n.language} />
                ))}
              </ul>
            )}
          </CardContent>
        </Card>
      </div>

      <AgentDetailModal
        agent={selectedAgent}
        recentCalls={recent_calls}
        open={selectedAgent !== null}
        onClose={() => setSelectedAgent(null)}
      />
    </div>
  )
}

function AgentRow({ agent, onOpen }: { agent: SupervisorAgent; onOpen: () => void }) {
  const { t } = useTranslation()
  const cfg = agentStatusConfig[(agent.status as AgentStatus)] ?? agentStatusConfig[5]

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onOpen}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onOpen()
        }
      }}
      aria-label={t('dialer.supervisor.agentDetail.ariaOpen', {
        name: `${agent.first_name} ${agent.last_name}`,
      })}
      className="group flex cursor-pointer items-center gap-3 rounded-lg border border-border bg-card px-3 py-2.5 transition-colors hover:bg-secondary/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
    >
      <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-secondary text-xs font-bold text-foreground">
        {initials(`${agent.first_name} ${agent.last_name}`)}
      </span>
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium text-foreground">
          {agent.first_name} {agent.last_name}
        </p>
        <p className="truncate text-xs text-muted-foreground">
          {agent.active_campaign_name ?? t('dialer.workspace.idle.noActiveCampaign')}
        </p>
      </div>
      <div className="hidden sm:block text-right">
        <p className="text-sm font-semibold tabular-nums text-foreground">{agent.calls_today}</p>
        <p className="text-[11px] text-muted-foreground">{t('dialer.dashboard.callsToday')}</p>
      </div>
      <span
        className="inline-flex shrink-0 items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs font-medium"
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
      <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground/40 transition-colors group-hover:text-muted-foreground" />
    </div>
  )
}

function RecentCallRow({ call, locale }: { call: RecentCallEntry; locale: string }) {
  return (
    <li className="flex items-center gap-3 px-4 py-3">
      <span
        className="h-9 w-1 shrink-0 rounded-full"
        style={{ backgroundColor: call.outcome_color }}
        aria-hidden="true"
      />
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <p className="truncate text-sm font-medium text-foreground">{call.contact_name}</p>
          {call.outcome_label && (
            <span
              className="inline-flex shrink-0 items-center rounded-full px-2 py-0.5 text-[11px] font-medium"
              style={{
                backgroundColor: `color-mix(in srgb, ${call.outcome_color} 14%, transparent)`,
                color: call.outcome_color,
              }}
            >
              {call.outcome_label}
            </span>
          )}
        </div>
        <p className="truncate text-xs text-muted-foreground">
          {call.agent_name}
          {call.contact_company ? ` · ${call.contact_company}` : ''}
        </p>
      </div>
      <div className="shrink-0 text-right">
        <p className="font-mono text-xs tabular-nums text-foreground">
          {formatDuration(call.duration_seconds)}
        </p>
        <p className="text-[11px] text-muted-foreground">
          {formatRelativeTime(call.created_at, locale)}
        </p>
      </div>
    </li>
  )
}

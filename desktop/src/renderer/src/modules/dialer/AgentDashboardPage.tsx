import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { LayoutDashboard, PhoneCall, Timer, TrendingUp, Percent } from 'lucide-react'
import { PageHeader, StatCard } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { useAgentDashboard } from '@/api/hooks/useDialer'
import OutcomeDonutChart from './components/dashboard/OutcomeDonutChart'

export default function AgentDashboardPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { data: dashboard, isLoading } = useAgentDashboard()

  if (isLoading) {
    return (
      <div className="p-6 space-y-4">
        <Skeleton className="h-12 w-48" />
        <div className="grid grid-cols-4 gap-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-32" />
          ))}
        </div>
        <Skeleton className="h-64" />
      </div>
    )
  }

  if (!dashboard) return null

  const avgMin = Math.floor(dashboard.average_duration_seconds / 60)
  const avgSec = Math.round(dashboard.average_duration_seconds % 60)
  const positiveCount = dashboard.outcome_breakdown
    .filter((o) => o.outcome_color === '#22c55e' || o.outcome_color === '#3b82f6')
    .reduce((acc, o) => acc + o.count, 0)
  const positiveRate = dashboard.total_calls_today > 0
    ? Math.round((positiveCount / dashboard.total_calls_today) * 100)
    : 0

  return (
    <div className="p-6 space-y-6">
      <PageHeader
        title={t('dialer.dashboard.title')}
        description={
          dashboard.active_campaign_name
            ? `${dashboard.active_campaign_name} — ${new Date().toLocaleDateString('de-DE', { weekday: 'long', day: 'numeric', month: 'long' })}`
            : t('dialer.dashboard.description')
        }
        icon={LayoutDashboard}
        moduleId="dialer"
        actions={
          <Button
            variant="outline"
            className="gap-2"
            onClick={() => navigate('/dialer/workspace')}
          >
            <PhoneCall className="h-4 w-4" />
            {t('dialer.dashboard.openWorkspace')}
          </Button>
        }
      />

      {/* KPI Row — hero stat cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          label={t('dialer.dashboard.callsToday')}
          value={dashboard.total_calls_today}
          icon={PhoneCall}
          hero
          className="animate-fade-up stagger-1"
        />
        <StatCard
          label={t('dialer.dashboard.avgDuration')}
          value={`${avgMin}:${String(avgSec).padStart(2, '0')}`}
          icon={Timer}
          hero
          className="animate-fade-up stagger-2"
        />
        <StatCard
          label={t('dialer.dashboard.callsPerHour')}
          value={dashboard.calls_per_hour.toFixed(1)}
          icon={TrendingUp}
          hero
          className="animate-fade-up stagger-3"
        />
        <StatCard
          label={t('dialer.dashboard.positiveRate')}
          value={positiveRate}
          suffix="%"
          icon={Percent}
          hero
          className="animate-fade-up stagger-4"
        />
      </div>

      {/* Two-column layout */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Outcome donut */}
        <Card className="animate-fade-up stagger-5">
          <CardHeader>
            <CardTitle className="text-sm font-medium text-muted-foreground">
              {t('dialer.dashboard.outcomeBreakdown')}
            </CardTitle>
          </CardHeader>
          <CardContent className="flex items-center justify-center py-4">
            {dashboard.outcome_breakdown.length > 0 ? (
              <OutcomeDonutChart
                data={dashboard.outcome_breakdown}
                total={dashboard.total_calls_today}
              />
            ) : (
              <p className="text-sm text-muted-foreground py-8">
                {t('dialer.dashboard.noCallsToday')}
              </p>
            )}
          </CardContent>
        </Card>

        {/* Active campaign card */}
        {dashboard.active_campaign_name && (
          <Card className="animate-fade-up stagger-6">
            <CardHeader>
              <CardTitle className="text-sm font-medium text-muted-foreground">
                {t('dialer.workspace.idle.selectCampaign')}
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <h3 className="text-xl font-bold">{dashboard.active_campaign_name}</h3>
              </div>
              <Button
                className="w-full gap-2"
                onClick={() => navigate('/dialer/workspace')}
              >
                <PhoneCall className="h-4 w-4" />
                {t('dialer.dashboard.openWorkspace')}
              </Button>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  )
}

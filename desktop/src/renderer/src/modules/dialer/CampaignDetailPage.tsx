import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  PhoneCall,
  Play,
  Pause,
  UserPlus,
  ArrowLeft,
  Users,
  CheckCircle2,
  Clock,
} from 'lucide-react'
import { toast } from 'sonner'
import { StatCard } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import {
  useCampaign,
  useCampaignContacts,
  useCampaignDashboard,
  useStartCampaign,
  usePauseCampaign,
  useAddContacts,
  useSkipContact,
  useRequeueContact,
} from '@/api/hooks/useDialer'
import { useDialerStore } from '@/stores/dialer'
import ContactQueueTable from './components/ContactQueueTable'
import AddContactsDialog from './components/AddContactsDialog'
import CampaignStatusBadge from './components/CampaignStatusBadge'
import CampaignModeBadge from './components/CampaignModeBadge'
import CampaignProgressBar from './components/CampaignProgressBar'
import OutcomeDonutChart from './components/dashboard/OutcomeDonutChart'

export default function CampaignDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const [addContactsOpen, setAddContactsOpen] = useState(false)
  const [contactStatusFilter, setContactStatusFilter] = useState<number | undefined>(undefined)

  const { data: campaign, isLoading: campaignLoading } = useCampaign(id!)
  const { data: contactsData, isLoading: contactsLoading } = useCampaignContacts(id!, {
    status_filter: contactStatusFilter,
  })
  const { data: dashboard } = useCampaignDashboard(id!)

  const startMutation = useStartCampaign()
  const pauseMutation = usePauseCampaign()
  const addContactsMutation = useAddContacts()
  const skipMutation = useSkipContact()
  const requeueMutation = useRequeueContact()
  const setActiveCampaign = useDialerStore((s) => s.setActiveCampaign)

  if (campaignLoading) {
    return (
      <div className="p-6 space-y-4">
        <Skeleton className="h-12 w-64" />
        <div className="grid grid-cols-4 gap-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-24" />
          ))}
        </div>
        <Skeleton className="h-96" />
      </div>
    )
  }

  if (!campaign) {
    return (
      <div className="flex items-center justify-center h-64 text-muted-foreground">
        {t('dialer.campaign.notFound')}
      </div>
    )
  }

  const contacts = contactsData?.contacts ?? []
  const isDraft = campaign.status === 1
  const isActive = campaign.status === 2
  const isPaused = campaign.status === 3

  const handleStartCall = () => {
    setActiveCampaign(campaign.id)
    navigate('/dialer/workspace')
  }

  const contactStatusFilters = [
    { value: undefined, label: t('dialer.campaigns.filterAll') },
    { value: 1, label: t('dialer.campaign.contacts.status.pending') },
    { value: 3, label: t('dialer.campaign.contacts.status.completed') },
    { value: 4, label: t('dialer.campaign.contacts.status.skipped') },
    { value: 5, label: t('dialer.campaign.contacts.status.callback') },
  ]

  return (
    <div className="p-6 space-y-5">
      {/* Header */}
      <div className="flex items-start justify-between gap-4">
        <div className="space-y-2">
          <button
            onClick={() => navigate('/dialer/campaigns')}
            className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors mb-1"
          >
            <ArrowLeft className="h-3.5 w-3.5" />
            {t('dialer.campaigns.title')}
          </button>
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold text-foreground">{campaign.name}</h1>
            <CampaignStatusBadge status={campaign.status} />
            <CampaignModeBadge mode={campaign.mode} />
          </div>
          {campaign.description && (
            <p className="text-sm text-muted-foreground max-w-xl">
              {campaign.description}
            </p>
          )}
        </div>

        <div className="flex items-center gap-2 shrink-0">
          {isDraft && (
            <Button
              variant="outline"
              size="sm"
              className="gap-1.5"
              onClick={() => setAddContactsOpen(true)}
            >
              <UserPlus className="h-4 w-4" />
              {t('dialer.campaign.contacts.add')}
            </Button>
          )}
          {(isDraft || isPaused) && (
            <Button
              size="sm"
              className="gap-1.5"
              onClick={() =>
                startMutation.mutate(campaign.id, {
                  onSuccess: () => toast.success(t('dialer.campaign.status.active')),
                })
              }
              disabled={isDraft && campaign.contact_count === 0}
            >
              <Play className="h-4 w-4" />
              {t('dialer.campaign.actions.start')}
            </Button>
          )}
          {isActive && (
            <>
              <Button
                variant="outline"
                size="sm"
                className="gap-1.5"
                onClick={() =>
                  pauseMutation.mutate(campaign.id, {
                    onSuccess: () => toast.success(t('dialer.campaign.status.paused')),
                  })
                }
              >
                <Pause className="h-4 w-4" />
                {t('dialer.campaign.actions.pause')}
              </Button>
              <Button size="sm" className="gap-1.5" onClick={handleStartCall}>
                <PhoneCall className="h-4 w-4" />
                {t('dialer.campaign.launchCTA')}
              </Button>
            </>
          )}
        </div>
      </div>

      {/* Layout: Left (queue) + Right (stats) */}
      <div className="grid grid-cols-1 xl:grid-cols-[1fr_380px] gap-6">
        {/* Left — Contact Queue */}
        <div className="space-y-4">
          {/* Stats strip */}
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            <StatCard
              label={t('dialer.campaign.stats.total')}
              value={dashboard?.total_contacts ?? campaign.contact_count}
              icon={Users}
              className="animate-fade-up stagger-1"
            />
            <StatCard
              label={t('dialer.campaign.contacts.status.completed')}
              value={dashboard?.completed_contacts ?? campaign.completed_count}
              icon={CheckCircle2}
              className="animate-fade-up stagger-2"
            />
            <StatCard
              label={t('dialer.campaign.contacts.status.pending')}
              value={dashboard?.pending_contacts ?? 0}
              icon={Clock}
              className="animate-fade-up stagger-3"
            />
            <StatCard
              label={t('dialer.campaign.contacts.status.callback')}
              value={dashboard?.callback_contacts ?? 0}
              icon={PhoneCall}
              className="animate-fade-up stagger-4"
            />
          </div>

          {/* Progress bar */}
          <CampaignProgressBar
            completed={dashboard?.completed_contacts ?? campaign.completed_count}
            callback={dashboard?.callback_contacts ?? 0}
            skipped={dashboard?.skipped_contacts ?? 0}
            total={dashboard?.total_contacts ?? campaign.contact_count}
          />

          {/* Contact filter pills */}
          <div className="flex items-center gap-1.5">
            {contactStatusFilters.map((f) => (
              <button
                key={f.label}
                onClick={() => setContactStatusFilter(f.value)}
                className={`rounded-full px-3 py-1 text-xs font-medium transition-all ${
                  contactStatusFilter === f.value
                    ? 'bg-primary text-primary-foreground shadow-sm'
                    : 'bg-card border border-border text-muted-foreground hover:bg-accent'
                }`}
              >
                {f.label}
              </button>
            ))}
          </div>

          {/* Table */}
          <Card>
            <CardContent className="p-0">
              <ContactQueueTable
                contacts={contacts}
                isLoading={contactsLoading}
                campaignStatus={campaign.status}
                isFiltered={contactStatusFilter !== undefined}
                onSkip={(contactId) =>
                  skipMutation.mutate(
                    { campaignId: campaign.id, contactId },
                    { onSuccess: () => toast.success(t('dialer.campaign.contacts.status.skipped')) },
                  )
                }
                onRequeue={(contactId) =>
                  requeueMutation.mutate(
                    { campaignId: campaign.id, contactId },
                    { onSuccess: () => toast.success(t('dialer.campaign.contacts.status.pending')) },
                  )
                }
              />
            </CardContent>
          </Card>
        </div>

        {/* Right — Dashboard Panel */}
        <div className="space-y-4">
          {dashboard && (
            <>
              <Card className="animate-fade-up stagger-2">
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm font-medium text-muted-foreground">
                    {t('dialer.campaign.stats.performance')}
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid grid-cols-2 gap-3">
                    <div className="rounded-lg bg-success/10 p-3 text-center">
                      <p className="text-2xl font-bold text-success tabular-nums">
                        {dashboard.positive_calls}
                      </p>
                      <p className="text-xs text-muted-foreground">{t('dialer.campaign.stats.positive')}</p>
                    </div>
                    <div className="rounded-lg bg-primary/10 p-3 text-center">
                      <p className="text-2xl font-bold text-primary tabular-nums">
                        {dashboard.calls_per_hour.toFixed(1)}
                      </p>
                      <p className="text-xs text-muted-foreground">{t('dialer.campaign.stats.callsPerHour')}</p>
                    </div>
                  </div>
                  <div className="rounded-lg bg-accent/50 p-3 text-center">
                    <p className="text-lg font-bold tabular-nums">
                      {Math.floor(dashboard.average_duration_seconds / 60)}:
                      {String(Math.round(dashboard.average_duration_seconds % 60)).padStart(2, '0')}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {t('dialer.dashboard.avgDuration')}
                    </p>
                  </div>
                </CardContent>
              </Card>

              {dashboard.outcome_breakdown.length > 0 && (
                <Card className="animate-fade-up stagger-3">
                  <CardHeader className="pb-2">
                    <CardTitle className="text-sm font-medium text-muted-foreground">
                      {t('dialer.dashboard.outcomeBreakdown')}
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <OutcomeDonutChart
                      data={dashboard.outcome_breakdown}
                      total={dashboard.total_calls}
                      size={140}
                    />
                  </CardContent>
                </Card>
              )}
            </>
          )}
        </div>
      </div>

      {/* Add contacts dialog */}
      <AddContactsDialog
        open={addContactsOpen}
        onOpenChange={setAddContactsOpen}
        isLoading={addContactsMutation.isPending}
        onSubmit={(payload) =>
          addContactsMutation.mutate(
            { campaignId: campaign.id, ...payload },
            {
              onSuccess: (data) => {
                setAddContactsOpen(false)
                toast.success(t('dialer.campaign.contacts.added', { count: data.added_count }))
              },
            },
          )
        }
      />
    </div>
  )
}

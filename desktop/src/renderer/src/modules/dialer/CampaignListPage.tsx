import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { PhoneCall, Plus } from 'lucide-react'
import { toast } from 'sonner'
import { PageHeader, EmptyState, ConfirmDialog } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import {
  useCampaigns,
  useCreateCampaign,
  useStartCampaign,
  usePauseCampaign,
  useArchiveCampaign,
} from '@/api/hooks/useDialer'
import CampaignCard from './components/CampaignCard'
import CampaignFormDialog from './components/CampaignFormDialog'
import type { Campaign } from '@/api/dialer-client'

const statusFilters = [
  { value: undefined, labelKey: 'dialer.campaigns.filterAll' },
  { value: 1, labelKey: 'dialer.campaign.status.draft' },
  { value: 2, labelKey: 'dialer.campaign.status.active' },
  { value: 3, labelKey: 'dialer.campaign.status.paused' },
  { value: 4, labelKey: 'dialer.campaign.status.completed' },
] as const

export default function CampaignListPage() {
  const { t } = useTranslation()
  const [statusFilter, setStatusFilter] = useState<number | undefined>(undefined)
  const [formOpen, setFormOpen] = useState(false)
  const [editCampaign, setEditCampaign] = useState<Campaign | null>(null)
  const [archiveId, setArchiveId] = useState<string | null>(null)

  const { data, isLoading } = useCampaigns({ status_filter: statusFilter })
  const createMutation = useCreateCampaign()
  const startMutation = useStartCampaign()
  const pauseMutation = usePauseCampaign()
  const archiveMutation = useArchiveCampaign()

  const campaigns = data?.campaigns ?? []

  const handleCreate = (formData: { name: string; description?: string; mode: number }) => {
    createMutation.mutate(formData, {
      onSuccess: () => {
        setFormOpen(false)
        toast.success(t('dialer.settings.outcomes.created'))
      },
    })
  }

  const handleStart = (id: string) => {
    startMutation.mutate(id, {
      onSuccess: () => toast.success(t('dialer.campaign.status.active')),
    })
  }

  const handlePause = (id: string) => {
    pauseMutation.mutate(id, {
      onSuccess: () => toast.success(t('dialer.campaign.status.paused')),
    })
  }

  const handleArchive = () => {
    if (!archiveId) return
    archiveMutation.mutate(archiveId, {
      onSuccess: () => {
        setArchiveId(null)
        toast.success(t('dialer.campaign.status.archived'))
      },
    })
  }

  return (
    <div className="p-6 space-y-5">
      <PageHeader
        title={t('dialer.campaigns.title')}
        description={t('dialer.campaigns.description')}
        icon={PhoneCall}
        moduleId="dialer"
        actions={
          <Button className="gap-2" onClick={() => { setEditCampaign(null); setFormOpen(true) }}>
            <Plus className="h-4 w-4" />
            {t('dialer.campaigns.new')}
          </Button>
        }
      />

      {/* Status filter pills */}
      <div className="flex items-center gap-1.5">
        {statusFilters.map((filter) => {
          const isActive = statusFilter === filter.value
          return (
            <button
              key={filter.labelKey}
              onClick={() => setStatusFilter(filter.value)}
              className={`rounded-full px-3.5 py-1.5 text-xs font-medium transition-all ${
                isActive
                  ? 'bg-primary text-primary-foreground shadow-sm'
                  : 'bg-card border border-border text-muted-foreground hover:bg-accent hover:text-accent-foreground'
              }`}
            >
              {t(filter.labelKey)}
            </button>
          )
        })}
      </div>

      {/* Campaign grid */}
      {isLoading ? (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-44 w-full rounded-xl" />
          ))}
        </div>
      ) : campaigns.length === 0 ? (
        <EmptyState
          icon={PhoneCall}
          title={t('dialer.campaigns.empty.title')}
          description={t('dialer.campaigns.empty.description')}
          action={{
            label: t('dialer.campaigns.new'),
            onClick: () => { setEditCampaign(null); setFormOpen(true) },
          }}
        />
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {campaigns.map((campaign, idx) => (
            <div
              key={campaign.id}
              className={`animate-fade-up stagger-${Math.min(idx + 1, 8)}`}
            >
              <CampaignCard
                campaign={campaign}
                onStart={handleStart}
                onPause={handlePause}
                onEdit={(c) => { setEditCampaign(c); setFormOpen(true) }}
                onArchive={setArchiveId}
              />
            </div>
          ))}
        </div>
      )}

      {/* Form dialog */}
      <CampaignFormDialog
        open={formOpen}
        onOpenChange={setFormOpen}
        onSubmit={handleCreate}
        campaign={editCampaign}
        isLoading={createMutation.isPending}
      />

      {/* Archive confirmation */}
      <ConfirmDialog
        open={!!archiveId}
        onOpenChange={(open) => !open && setArchiveId(null)}
        title={t('dialer.campaign.actions.archive')}
        description={t('dialer.campaign.actions.archiveConfirm')}
        onConfirm={handleArchive}
        variant="destructive"
      />
    </div>
  )
}

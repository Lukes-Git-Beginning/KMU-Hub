import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Settings, Plus } from 'lucide-react'
import { toast } from 'sonner'
import { PageHeader, ConfirmDialog } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import {
  useOutcomes,
  useCreateOutcome,
  useUpdateOutcome,
  useDeleteOutcome,
} from '@/api/hooks/useDialer'
import OutcomeCard from './components/settings/OutcomeCard'
import OutcomeFormDialog from './components/settings/OutcomeFormDialog'
import type { CallOutcome } from '@/api/dialer-client'

export default function DialerSettingsPage() {
  const { t } = useTranslation()
  const [formOpen, setFormOpen] = useState(false)
  const [editOutcome, setEditOutcome] = useState<CallOutcome | null>(null)
  const [deleteId, setDeleteId] = useState<string | null>(null)

  const { data, isLoading } = useOutcomes(true)
  const createMutation = useCreateOutcome()
  const updateMutation = useUpdateOutcome()
  const deleteMutation = useDeleteOutcome()

  const outcomes = data?.outcomes ?? []

  const handleSubmit = (formData: {
    label: string
    color: string
    is_positive: boolean
    is_callback: boolean
    is_appointment: boolean
    sort_order: number
  }) => {
    if (editOutcome) {
      updateMutation.mutate(
        { id: editOutcome.id, ...formData },
        {
          onSuccess: () => {
            setFormOpen(false)
            setEditOutcome(null)
            toast.success(t('dialer.settings.outcomes.updated'))
          },
        },
      )
    } else {
      createMutation.mutate(formData, {
        onSuccess: () => {
          setFormOpen(false)
          toast.success(t('dialer.settings.outcomes.created'))
        },
      })
    }
  }

  const handleToggleActive = (id: string, isActive: boolean) => {
    updateMutation.mutate({ id, is_active: isActive })
  }

  const handleDelete = () => {
    if (!deleteId) return
    deleteMutation.mutate(deleteId, {
      onSuccess: () => {
        setDeleteId(null)
        toast.success(t('dialer.settings.outcomes.deleted'))
      },
    })
  }

  return (
    <div className="p-6 space-y-5">
      <PageHeader
        title={t('dialer.settings.title')}
        description={t('dialer.settings.description')}
        icon={Settings}
        moduleId="dialer"
        actions={
          <Button
            className="gap-2"
            onClick={() => {
              setEditOutcome(null)
              setFormOpen(true)
            }}
          >
            <Plus className="h-4 w-4" />
            {t('dialer.settings.outcomes.new')}
          </Button>
        }
      />

      {/* Outcomes list */}
      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-16 w-full rounded-xl" />
          ))}
        </div>
      ) : outcomes.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <p className="text-sm text-muted-foreground">
            {t('dialer.settings.outcomes.empty')}
          </p>
        </div>
      ) : (
        <div className="space-y-2">
          {outcomes.map((outcome, idx) => (
            <div
              key={outcome.id}
              className={`animate-fade-up stagger-${Math.min(idx + 1, 8)}`}
            >
              <OutcomeCard
                outcome={outcome}
                onEdit={(o) => {
                  setEditOutcome(o)
                  setFormOpen(true)
                }}
                onDelete={setDeleteId}
                onToggleActive={handleToggleActive}
              />
            </div>
          ))}
        </div>
      )}

      {/* Form dialog */}
      <OutcomeFormDialog
        open={formOpen}
        onOpenChange={setFormOpen}
        onSubmit={handleSubmit}
        outcome={editOutcome}
        isLoading={createMutation.isPending || updateMutation.isPending}
        nextSortOrder={outcomes.length}
      />

      {/* Delete confirmation */}
      <ConfirmDialog
        open={!!deleteId}
        onOpenChange={(open) => !open && setDeleteId(null)}
        title={t('dialer.settings.outcomes.delete')}
        description={t('dialer.settings.outcomes.deleteConfirm')}
        onConfirm={handleDelete}
        variant="destructive"
      />
    </div>
  )
}

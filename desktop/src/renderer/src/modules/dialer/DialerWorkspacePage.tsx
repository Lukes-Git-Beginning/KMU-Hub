import { useState, useCallback, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { PhoneCall, ChevronRight } from 'lucide-react'
import { toast } from 'sonner'
import { EmptyState } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { useDialerStore } from '@/stores/dialer'
import {
  useCampaign,
  useCampaigns,
  useAgentStatus,
  useAgentDashboard,
  useOutcomes,
  useGetNextContact,
  useInitiateCall,
  useLogOutcome,
  useSaveNotes,
  useCompleteWrapUp,
  useSetAgentStatus,
  useSkipContact,
} from '@/api/hooks/useDialer'
import type { CampaignContact } from '@/api/dialer-client'

import AgentStatusPill from './components/workspace/AgentStatusPill'
import CallTimer from './components/workspace/CallTimer'
import ContactHeroPanel from './components/workspace/ContactHeroPanel'
import CallControlBar from './components/workspace/CallControlBar'
import OutcomeGrid from './components/workspace/OutcomeGrid'
import CallNotesEditor from './components/workspace/CallNotesEditor'
import WrapUpPanel from './components/workspace/WrapUpPanel'
import CampaignProgressStrip from './components/workspace/CampaignProgressStrip'
import CampaignProgressBar from './components/CampaignProgressBar'
import CampaignModeBadge from './components/CampaignModeBadge'

export default function DialerWorkspacePage() {
  const { t } = useTranslation()
  const [notesOpen, setNotesOpen] = useState(false)
  const [activeContact, setActiveContact] = useState<CampaignContact | null>(null)

  // Store
  const {
    callPhase,
    activeCampaignId,
    activeSessionId,
    callStartedAt,
    selectedOutcomeId,
    callbackAt,
    draftNotes,
    isWrapUpSubmitting,
    startDialing,
    onCallConnected,
    onHangUp,
    selectOutcome,
    setCallbackAt,
    setDraftNotes,
    completeWrapUp: resetStore,
    setActiveCampaign,
    setWrapUpSubmitting,
  } = useDialerStore()

  // Queries
  const { data: agentStatus } = useAgentStatus()
  const { data: agentDashboard } = useAgentDashboard()
  const { data: activeCampaign } = useCampaign(activeCampaignId ?? '')
  const { data: campaignsData } = useCampaigns()
  const { data: outcomesData } = useOutcomes()

  // Mutations
  const getNextMutation = useGetNextContact()
  const initCallMutation = useInitiateCall()
  const logOutcomeMutation = useLogOutcome()
  const saveNotesMutation = useSaveNotes()
  const completeWrapUpMutation = useCompleteWrapUp()
  const setStatusMutation = useSetAgentStatus()
  const skipMutation = useSkipContact()

  const outcomes = outcomesData?.outcomes ?? []
  const selectedOutcome = outcomes.find((o) => o.id === selectedOutcomeId) ?? null
  const campaigns = campaignsData?.campaigns?.filter((c) => c.status === 2) ?? []

  // The call phase lives in the (session-persistent) store, but the active
  // contact is local component state. Leaving the workspace and returning
  // remounts the page — dropping the contact while a non-idle phase lingers,
  // which left the call phases with nothing to render (blank screen). Recover to
  // idle whenever the phase has no contact to back it.
  useEffect(() => {
    if (callPhase !== 'idle' && !activeContact) {
      resetStore()
      setActiveContact(null)
    }
  }, [callPhase, activeContact, resetStore])

  // ── Actions ──

  const handleLoadNext = useCallback(() => {
    if (!activeCampaignId) return
    getNextMutation.mutate(activeCampaignId, {
      onSuccess: (contact) => {
        setActiveContact(contact)
        startDialing(contact.id, activeCampaignId)
      },
      onError: () => {
        toast.info(t('dialer.workspace.idle.noMoreContacts'))
      },
    })
  }, [activeCampaignId, getNextMutation, startDialing, t])

  const handleDial = useCallback(() => {
    if (!activeContact) return
    initCallMutation.mutate(
      { campaign_contact_id: activeContact.id },
      {
        onSuccess: (session) => {
          onCallConnected(session.id)
        },
      },
    )
  }, [activeContact, initCallMutation, onCallConnected])

  const handleHangUp = useCallback(() => {
    onHangUp()
  }, [onHangUp])

  const handleCancel = useCallback(() => {
    resetStore()
    setActiveContact(null)
  }, [resetStore])

  const handleSkip = useCallback(() => {
    if (!activeContact || !activeCampaignId) return
    skipMutation.mutate(
      { campaignId: activeCampaignId, contactId: activeContact.id },
      {
        onSuccess: () => {
          resetStore()
          setActiveContact(null)
          toast.info(t('dialer.campaign.contacts.status.skipped'))
        },
      },
    )
  }, [activeContact, activeCampaignId, skipMutation, resetStore, t])

  const handleComplete = useCallback(() => {
    if (!activeSessionId || !selectedOutcomeId) return
    setWrapUpSubmitting(true)

    logOutcomeMutation.mutate(
      {
        callId: activeSessionId,
        outcome_id: selectedOutcomeId,
        notes: draftNotes || undefined,
        callback_at: callbackAt || undefined,
      },
      {
        onSuccess: () => {
          completeWrapUpMutation.mutate(
            { callId: activeSessionId },
            {
              onSuccess: () => {
                resetStore()
                setActiveContact(null)
                toast.success(t('dialer.workspace.wrapUp.complete'))
              },
              onError: () => {
                setWrapUpSubmitting(false)
                toast.error(t('dialer.workspace.wrapUp.completeFailed'))
              },
            },
          )
        },
        onError: () => setWrapUpSubmitting(false),
      },
    )
  }, [
    activeSessionId,
    selectedOutcomeId,
    draftNotes,
    callbackAt,
    logOutcomeMutation,
    completeWrapUpMutation,
    resetStore,
    setWrapUpSubmitting,
    t,
  ])

  const handleSkipOutcome = useCallback(() => {
    if (!activeSessionId || !activeContact || !activeCampaignId) return
    setWrapUpSubmitting(true)
    // First mark the contact as skipped so it doesn't stay in_progress forever.
    skipMutation.mutate(
      { campaignId: activeCampaignId, contactId: activeContact.id },
      {
        onSuccess: () => {
          completeWrapUpMutation.mutate(
            { callId: activeSessionId },
            {
              onSuccess: () => {
                resetStore()
                setActiveContact(null)
              },
              onError: () => setWrapUpSubmitting(false),
            },
          )
        },
        onError: () => setWrapUpSubmitting(false),
      },
    )
  }, [activeSessionId, activeContact, activeCampaignId, skipMutation, completeWrapUpMutation, resetStore, setWrapUpSubmitting])

  const handleAutoSave = useCallback(
    (text: string) => {
      if (activeSessionId) {
        saveNotesMutation.mutate({ callId: activeSessionId, notes: text })
      }
    },
    [activeSessionId, saveNotesMutation],
  )

  const handleStatusChange = useCallback(
    (newStatus: number) => {
      setStatusMutation.mutate({ status: newStatus })
    },
    [setStatusMutation],
  )

  // ── Render phases ──

  return (
    <div className="flex h-full flex-col">
      {/* Phase content with animated transitions */}
      <div key={callPhase} className="flex-1 overflow-auto animate-scale-in">
        {callPhase === 'idle' && (
          <IdlePhase
            activeCampaign={activeCampaign ?? null}
            campaigns={campaigns}
            agentStatus={agentStatus?.status ?? 5}
            agentDashboard={agentDashboard ?? undefined}
            onSelectCampaign={setActiveCampaign}
            onLoadNext={handleLoadNext}
            onStatusChange={handleStatusChange}
            isLoading={getNextMutation.isPending}
          />
        )}

        {callPhase === 'dialing' && activeContact && (
          <div className="flex flex-col items-center justify-center min-h-[70vh] px-6">
            <ContactHeroPanel contact={activeContact} showConnecting />
            <div className="mt-10">
              <CallControlBar
                phase="dialing"
                onDial={handleDial}
                onCancel={handleCancel}
              />
            </div>
          </div>
        )}

        {callPhase === 'on_call' && activeContact && (
          <div className="flex flex-col items-center justify-center min-h-[70vh] px-6">
            <ContactHeroPanel contact={activeContact}>
              <div className="pt-4">
                <CallTimer startedAt={callStartedAt} />
              </div>
            </ContactHeroPanel>
            <div className="mt-10">
              <CallControlBar
                phase="on_call"
                onHangUp={handleHangUp}
                onSkip={handleSkip}
                onToggleNotes={() => setNotesOpen((prev) => !prev)}
                notesOpen={notesOpen}
              />
            </div>
            {notesOpen && (
              <div className="w-full max-w-xl mt-6 animate-fade-up">
                <CallNotesEditor
                  value={draftNotes}
                  onChange={setDraftNotes}
                  onAutoSave={handleAutoSave}
                />
              </div>
            )}
          </div>
        )}

        {callPhase === 'wrap_up' && activeContact && (
          <div className="flex gap-8 p-6 max-w-5xl mx-auto">
            {/* Left — Outcome selection */}
            <div className="flex-[55] min-w-0">
              <h2 className="text-2xl font-bold mb-6">{t('dialer.workspace.wrapUp.title')}</h2>
              <OutcomeGrid
                outcomes={outcomes}
                selectedId={selectedOutcomeId}
                onSelect={selectOutcome}
              />
            </div>
            {/* Right — Summary + Complete */}
            <div className="flex-[45] min-w-0">
              <WrapUpPanel
                contact={activeContact}
                callDurationStart={callStartedAt}
                selectedOutcome={selectedOutcome}
                notes={draftNotes}
                onNotesChange={setDraftNotes}
                callbackAt={callbackAt}
                onCallbackAtChange={setCallbackAt}
                onComplete={handleComplete}
                onSkipOutcome={handleSkipOutcome}
                isSubmitting={isWrapUpSubmitting}
              />
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

// ── Idle Phase Sub-Component ──

function IdlePhase({
  activeCampaign,
  campaigns,
  agentStatus,
  agentDashboard,
  onSelectCampaign,
  onLoadNext,
  onStatusChange,
  isLoading,
}: {
  activeCampaign: { id: string; name: string; mode: number; contact_count: number; completed_count: number } | null
  campaigns: Array<{ id: string; name: string; mode: number; contact_count: number; completed_count: number }>
  agentStatus: number
  agentDashboard: import('@/api/dialer-client').AgentDashboard | undefined
  onSelectCampaign: (id: string | null) => void
  onLoadNext: () => void
  onStatusChange: (status: number) => void
  isLoading: boolean
}) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [showCampaignPicker, setShowCampaignPicker] = useState(false)

  return (
    <div className="flex gap-8 p-6 max-w-5xl mx-auto">
      {/* Left 60% */}
      <div className="flex-[60] space-y-6 min-w-0">
        {/* Agent status */}
        <div className="flex items-center justify-between">
          <AgentStatusPill status={agentStatus} onChange={onStatusChange} />
        </div>

        {/* Active campaign card */}
        <div className="space-y-2">
          <h3 className="text-xs font-semibold tracking-widest uppercase text-muted-foreground">
            {t('dialer.workspace.idle.selectCampaign')}
          </h3>

          {activeCampaign ? (
            <Card className="animate-fade-up">
              <CardContent className="p-4 space-y-3">
                <div className="flex items-center justify-between">
                  <h4 className="text-lg font-bold">{activeCampaign.name}</h4>
                  <CampaignModeBadge mode={activeCampaign.mode} />
                </div>
                <CampaignProgressBar
                  completed={activeCampaign.completed_count}
                  callback={0}
                  skipped={0}
                  total={activeCampaign.contact_count}
                />
                <button
                  onClick={() => setShowCampaignPicker(!showCampaignPicker)}
                  className="text-xs text-primary hover:text-primary/80 transition-colors"
                >
                  {t('dialer.workspace.idle.changeCampaign')}
                </button>
              </CardContent>
            </Card>
          ) : campaigns.length === 0 ? (
            <Card className="animate-fade-up border-dashed">
              <CardContent className="p-2">
                <EmptyState
                  icon={PhoneCall}
                  title={t('dialer.workspace.idle.emptyTitle')}
                  description={t('dialer.workspace.idle.emptyDescription')}
                  action={{
                    label: t('dialer.workspace.idle.goToCampaigns'),
                    onClick: () => navigate('/dialer/campaigns'),
                  }}
                />
              </CardContent>
            </Card>
          ) : (
            <Card className="animate-fade-up border-dashed">
              <CardContent className="p-2">
                <EmptyState
                  icon={PhoneCall}
                  title={t('dialer.workspace.idle.chooseCampaign')}
                  description={t('dialer.workspace.idle.chooseCampaignHint')}
                />
              </CardContent>
            </Card>
          )}

          {/* Campaign picker (expandable) */}
          {(showCampaignPicker || !activeCampaign) && campaigns.length > 0 && (
            <div className="space-y-1.5 animate-fade-up">
              {campaigns
                .filter((c) => c.id !== activeCampaign?.id)
                .map((campaign) => (
                  <button
                    key={campaign.id}
                    onClick={() => {
                      onSelectCampaign(campaign.id)
                      setShowCampaignPicker(false)
                    }}
                    className="flex w-full items-center justify-between rounded-lg border border-border bg-card px-4 py-3 text-left hover:bg-accent transition-colors"
                  >
                    <div>
                      <p className="text-sm font-medium">{campaign.name}</p>
                      <p className="text-xs text-muted-foreground">
                        {campaign.completed_count} / {campaign.contact_count}
                      </p>
                    </div>
                    <ChevronRight className="h-4 w-4 text-muted-foreground" />
                  </button>
                ))}
            </div>
          )}
        </div>

        {/* Load next CTA */}
        <Button
          size="lg"
          className="w-full h-14 text-lg gap-3"
          onClick={onLoadNext}
          disabled={!activeCampaign || agentStatus !== 1 || isLoading}
          loading={isLoading}
        >
          <PhoneCall className="h-5 w-5" />
          {t('dialer.workspace.idle.loadNext')}
        </Button>

        {agentStatus !== 1 && activeCampaign && (
          <p className="text-xs text-center text-muted-foreground">
            {t('dialer.workspace.idle.setAvailableHint', {
              status: t('dialer.agentStatus.available'),
            })}
          </p>
        )}
      </div>

      {/* Right 40% — Today stats */}
      <div className="flex-[40] min-w-0">
        <CampaignProgressStrip dashboard={agentDashboard} />
      </div>
    </div>
  )
}

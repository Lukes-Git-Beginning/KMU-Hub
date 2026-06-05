/**
 * MeetingLobby -- Pre-meeting lobby with camera/mic preview and meeting context.
 *
 * Split layout: left side has camera preview (PreJoinScreen from video feature),
 * right side shows meeting details, agenda, attendees, and "Letzte Notizen"
 * for recurring meetings.
 *
 * Join-with-Consent (R2-P0.4): if the meeting is already being recorded when the
 * user tries to join, the join button is replaced by an explicit Accept / Decline
 * banner. Accept = consent submitted + join; Decline = consent declined + join
 * (user is anonymized in the recording per GDPR).
 */
import { useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Calendar,
  Clock,
  Users,
  FileText,
  ChevronDown,
  ChevronUp,
  Phone,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { PreJoinScreen } from '@/features/video/PreJoinScreen'
import {
  useMeeting,
  usePreviousMeetingNotes,
  useStartMeeting,
} from '@/api/hooks/useMeetings'
import { useActiveMeetingRecording, useSetRecordingConsent } from '@/api/hooks/useVideo'
import { useVideoStore } from '@/stores/video'
import type { RsvpStatus } from '@/api/video-types'
import { formatDate as libFormatDate, formatTime as libFormatTime } from '@/lib/format'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface MeetingLobbyProps {
  meetingId: string
  onJoinMeeting?: () => void
  onCancel?: () => void
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const rsvpLabelKeys: Record<RsvpStatus, string> = {
  accepted: 'meetings.rsvp.accepted',
  declined: 'meetings.rsvp.declined',
  tentative: 'meetings.rsvp.tentative',
  pending: 'meetings.rsvp.pending',
}

const rsvpColors: Record<RsvpStatus, string> = {
  accepted: 'bg-green-100 text-green-700',
  declined: 'bg-red-100 text-red-700',
  tentative: 'bg-amber-100 text-amber-700',
  pending: 'bg-gray-100 text-gray-600',
}

function formatTime(isoDate: string): string {
  return libFormatTime(isoDate, { hour: '2-digit', minute: '2-digit' })
}

function formatDate(isoDate: string): string {
  return libFormatDate(isoDate, { weekday: 'long', day: 'numeric', month: 'long' })
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function MeetingLobby({ meetingId, onJoinMeeting, onCancel }: MeetingLobbyProps) {
  const { t } = useTranslation()
  const { data: meeting, isLoading } = useMeeting(meetingId)
  const { data: previousNotes } = usePreviousMeetingNotes(meetingId)
  const startMutation = useStartMeeting()
  const setActiveCall = useVideoStore((s) => s.setActiveCall)
  const setConsent = useSetRecordingConsent()

  const [previousNotesExpanded, setPreviousNotesExpanded] = useState(true)
  // Tracks whether the user has acknowledged the recording notice in the lobby.
  // null = no active recording / not checked yet
  // false = declined (join with anonymization)
  // true  = accepted (join with full recording)
  const [recordingConsentGiven, setRecordingConsentGiven] = useState<boolean | null>(null)

  // Check for an active recording so the lobby can show the consent notice.
  const { data: activeRecording } = useActiveMeetingRecording(meetingId)
  const activeRecordingId = activeRecording?.id ?? null

  // Whether the consent banner should be shown:
  // active recording present AND user has not yet responded.
  const showConsentNotice = !!activeRecordingId && recordingConsentGiven === null

  const performJoin = useCallback(
    (_audioEnabled: boolean, _videoEnabled: boolean) => {
      if (!meeting) return
      if (meeting.status === 'scheduled') {
        startMutation.mutate(meetingId, {
          onSuccess: (response) => {
            setActiveCall(
              { ...meeting, status: 'in_progress', room_name: meeting.room_name ?? meetingId } as never,
              response.token,
              response.ws_url,
            )
            onJoinMeeting?.()
          },
        })
      } else {
        onJoinMeeting?.()
      }
    },
    [meeting, meetingId, startMutation, setActiveCall, onJoinMeeting],
  )

  const handleJoin = useCallback(
    (audioEnabled: boolean, videoEnabled: boolean) => {
      // If a recording is active and the user has not yet given consent,
      // block the join until they respond to the consent notice.
      if (showConsentNotice) return
      performJoin(audioEnabled, videoEnabled)
    },
    [showConsentNotice, performJoin],
  )

  const handleConsentAccept = useCallback(async () => {
    if (!activeRecordingId) return
    try {
      await setConsent.mutateAsync({ recordingId: activeRecordingId, consented: true })
    } catch {
      // Non-fatal: the join proceeds regardless; consent will be re-requested
      // by the in-room consent dialog if the mutation failed.
    }
    setRecordingConsentGiven(true)
    performJoin(true, true)
  }, [activeRecordingId, setConsent, performJoin])

  const handleConsentDecline = useCallback(async () => {
    if (!activeRecordingId) return
    try {
      await setConsent.mutateAsync({ recordingId: activeRecordingId, consented: false })
    } catch {
      // Non-fatal: user joins anonymized; backend will treat missing consent as declined.
    }
    setRecordingConsentGiven(false)
    performJoin(true, true)
  }, [activeRecordingId, setConsent, performJoin])

  if (isLoading || !meeting) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <div className="animate-spin h-8 w-8 border-2 border-primary border-t-transparent rounded-full" />
      </div>
    )
  }

  const attendees = meeting.attendees ?? []

  return (
    <div className="flex-1 flex flex-col lg:flex-row h-full overflow-hidden">
      {/* Left: Camera Preview (60%) */}
      <div className="flex-[3] flex flex-col items-center justify-center bg-secondary/30 p-6 min-h-[400px]">
        <div className="w-full max-w-lg">
          <PreJoinScreen
            onJoin={handleJoin}
            onCancel={onCancel}
          />
        </div>
      </div>

      {/* Right: Meeting Context (40%) */}
      <div className="flex-[2] overflow-y-auto border-l border-border p-6">
        {/* Meeting Title and Time */}
        <div className="mb-6">
          <h2 className="text-lg font-semibold text-foreground mb-1">{meeting.title}</h2>
          <div className="flex items-center gap-3 text-sm text-muted-foreground">
            <span className="flex items-center gap-1">
              <Calendar className="h-3.5 w-3.5" />
              {formatDate(meeting.scheduled_start)}
            </span>
            <span className="flex items-center gap-1">
              <Clock className="h-3.5 w-3.5" />
              {formatTime(meeting.scheduled_start)} - {formatTime(meeting.scheduled_end)}
            </span>
          </div>
        </div>

        {/* Agenda */}
        {meeting.agenda && (
          <div className="mb-6">
            <h3 className="text-xs font-medium uppercase text-muted-foreground mb-2">
              <FileText className="mr-1 inline h-3.5 w-3.5" />
              {t('meetings.form.agenda')}
            </h3>
            <div className="rounded-lg border border-border bg-card p-3 text-sm text-foreground whitespace-pre-wrap">
              {meeting.agenda}
            </div>
          </div>
        )}

        {/* Attendees with RSVP */}
        <div className="mb-6">
          <h3 className="text-xs font-medium uppercase text-muted-foreground mb-2">
            <Users className="mr-1 inline h-3.5 w-3.5" />
            {t('meetings.form.participants')} ({attendees.length})
          </h3>
          <div className="space-y-2">
            {attendees.map((a) => (
              <div key={a.user_id} className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <span className="flex h-7 w-7 items-center justify-center rounded-full bg-primary/10 text-[10px] font-medium text-primary">
                    {(a.user_name ?? a.user_id).slice(0, 2).toUpperCase()}
                  </span>
                  <span className="text-sm text-foreground">
                    {a.user_name ?? a.user_id.slice(0, 8)}
                  </span>
                </div>
                <Badge
                  variant="outline"
                  className={`text-[10px] ${rsvpColors[a.rsvp_status]}`}
                >
                  {t(rsvpLabelKeys[a.rsvp_status])}
                </Badge>
              </div>
            ))}
          </div>
        </div>

        {/* Shared Documents placeholder */}
        <div className="mb-6">
          <h3 className="text-xs font-medium uppercase text-muted-foreground mb-2">
            <FileText className="mr-1 inline h-3.5 w-3.5" />
            {t('meetings.lobby.sharedDocuments')}
          </h3>
          <div className="rounded-lg border border-dashed border-border p-3 text-center text-sm text-muted-foreground">
            {t('meetings.lobby.fileManagerPlaceholder')}
          </div>
        </div>

        <Separator className="my-4" />

        {/* Previous Meeting Notes (for recurring meetings) */}
        {previousNotes && (
          <div className="mb-6">
            <button
              onClick={() => setPreviousNotesExpanded(!previousNotesExpanded)}
              className="flex items-center gap-1.5 text-xs font-medium uppercase text-muted-foreground mb-2 hover:text-foreground transition-colors"
            >
              {previousNotesExpanded ? (
                <ChevronUp className="h-3.5 w-3.5" />
              ) : (
                <ChevronDown className="h-3.5 w-3.5" />
              )}
              {t('meetings.lobby.previousNotes')}
            </button>
            {previousNotesExpanded && (
              <div className="rounded-lg border border-border bg-amber-50/50 p-3 text-sm text-foreground whitespace-pre-wrap">
                {previousNotes.content}
              </div>
            )}
          </div>
        )}

        {/* Recording consent notice — shown when the meeting is already being recorded.
            The user must explicitly accept or decline before joining.
            DSGVO Art. 6(1)(a): explicit consent required before joining a recorded session. */}
        {showConsentNotice && (
          <div className="mb-4 rounded-lg border border-red-200 bg-red-50/60 p-4 dark:border-red-900/40 dark:bg-red-900/10">
            <div className="flex items-start gap-3">
              {/* Recording indicator — pulsing dot */}
              <span className="mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center">
                <span className="h-2.5 w-2.5 animate-pulse rounded-full bg-red-600" />
              </span>
              <div className="flex-1 space-y-3">
                <p className="text-sm font-medium text-foreground">
                  {t('meetings.lobby.recordingActiveNotice')}
                </p>
                <p className="text-xs text-muted-foreground">
                  {t('meetings.lobby.recordingActiveInfo')}
                </p>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    className="flex-1"
                    onClick={() => void handleConsentDecline()}
                    disabled={setConsent.isPending || startMutation.isPending}
                  >
                    {t('meetings.lobby.recordingActiveDecline')}
                  </Button>
                  <Button
                    size="sm"
                    className="flex-1 bg-green-600 text-white hover:bg-green-700"
                    onClick={() => void handleConsentAccept()}
                    disabled={setConsent.isPending || startMutation.isPending}
                  >
                    {t('meetings.lobby.recordingActiveAccept')}
                  </Button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Join Button (large, prominent) — hidden when consent notice is shown */}
        {!showConsentNotice && (
          <Button
            className="w-full h-12 text-base"
            onClick={() => handleJoin(true, true)}
            disabled={startMutation.isPending}
          >
            <Phone className="mr-2 h-5 w-5" />
            {startMutation.isPending ? t('meetings.lobby.starting') : t('meetings.lobby.joinMeeting')}
          </Button>
        )}
      </div>
    </div>
  )
}

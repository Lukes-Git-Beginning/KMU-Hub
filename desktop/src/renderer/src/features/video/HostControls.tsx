/**
 * HostControls -- server-authoritative moderation controls for the meeting host.
 *
 * Wave 3: mute/kick per participant, mute-all, co-host promote/demote, room lock.
 *
 * Architecture: ALL moderation goes through backend routes (NOT direct LiveKit
 * RoomServiceClient calls). The backend enforces RBAC and mutes via LiveKit
 * server-side Room service.
 *
 * Visibility rules:
 * - Roster controls (mute/kick/co-host): only rendered when isHost=true
 * - Global controls (mute-all, lock): only rendered when isHost=true
 * - Co-host promote/demote: only the organizer (isOrganizer) may use these
 */
import { useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  useMuteParticipant,
  useMuteAllParticipants,
  useKickParticipant,
  usePromoteCoHost,
  useDemoteCoHost,
  useSetMeetingLock,
} from '@/api/hooks/useVideo'
import { cn } from '@/lib'

// ---------------------------------------------------------------------------
// SVG icons (no emoji per design rules)
// ---------------------------------------------------------------------------

function MicOffIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor"
      strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <line x1="2" x2="22" y1="2" y2="22" />
      <path d="M18.89 13.23A7.12 7.12 0 0 0 19 12v-2" />
      <path d="M5 10v2a7 7 0 0 0 12 5.29" />
      <path d="M15 9.34V5a3 3 0 0 0-5.68-1.33" />
      <path d="M9 9v3a3 3 0 0 0 5.12 2.12" />
      <line x1="12" x2="12" y1="19" y2="22" />
    </svg>
  )
}

function UserXIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor"
      strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
      <circle cx="9" cy="7" r="4" />
      <line x1="17" x2="22" y1="18" y2="23" />
      <line x1="22" x2="17" y1="18" y2="23" />
    </svg>
  )
}

function ShieldPlusIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor"
      strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10" />
      <line x1="9" y1="12" x2="15" y2="12" />
      <line x1="12" y1="9" x2="12" y2="15" />
    </svg>
  )
}

function ShieldMinusIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor"
      strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10" />
      <line x1="9" y1="12" x2="15" y2="12" />
    </svg>
  )
}

function UsersSlashIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor"
      strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
      <circle cx="9" cy="7" r="4" />
      <path d="M22 10H16" />
      <path d="M19 4v12" />
    </svg>
  )
}

function LockIcon({ small }: { small?: boolean }) {
  const size = small ? 12 : 16
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor"
      strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <rect width="18" height="11" x="3" y="11" rx="2" ry="2" />
      <path d="M7 11V7a5 5 0 0 1 10 0v4" />
    </svg>
  )
}

function UnlockIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor"
      strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <rect width="18" height="11" x="3" y="11" rx="2" ry="2" />
      <path d="M7 11V7a5 5 0 0 1 9.9-1" />
    </svg>
  )
}

// ---------------------------------------------------------------------------
// Per-participant moderation actions (rendered inside the roster row)
// ---------------------------------------------------------------------------

export interface ParticipantActionsProps {
  /** LiveKit participant identity (= User-ID used for API calls). */
  participantIdentity: string
  meetingId: string
  /** True when the current user is the meeting organizer. */
  isOrganizer: boolean
  /** True when participantIdentity is already a co-host. */
  isCoHost: boolean
}

export function ParticipantActions({
  participantIdentity,
  meetingId,
  isOrganizer,
  isCoHost,
}: ParticipantActionsProps) {
  const { t } = useTranslation()

  const muteParticipant = useMuteParticipant(meetingId)
  const kickParticipant = useKickParticipant(meetingId)
  const promoteCoHost = usePromoteCoHost(meetingId)
  const demoteCoHost = useDemoteCoHost(meetingId)

  const handleMute = useCallback(async () => {
    try {
      await muteParticipant.mutateAsync(participantIdentity)
      toast.success(t('features.video.hostControls.muteSuccess'))
    } catch (err) {
      const status = (err as { status?: number })?.status
      toast.error(
        status === 403
          ? t('features.video.hostControls.error403')
          : t('features.video.hostControls.muteError'),
      )
    }
  }, [muteParticipant, participantIdentity, t])

  const handleKick = useCallback(async () => {
    try {
      await kickParticipant.mutateAsync(participantIdentity)
      toast.success(t('features.video.hostControls.kickSuccess'))
    } catch (err) {
      const status = (err as { status?: number })?.status
      toast.error(
        status === 403
          ? t('features.video.hostControls.error403')
          : t('features.video.hostControls.kickError'),
      )
    }
  }, [kickParticipant, participantIdentity, t])

  const handleCoHostToggle = useCallback(async () => {
    try {
      if (isCoHost) {
        await demoteCoHost.mutateAsync(participantIdentity)
        toast.success(t('features.video.hostControls.coHostRemovedSuccess'))
      } else {
        await promoteCoHost.mutateAsync(participantIdentity)
        toast.success(t('features.video.hostControls.coHostAddedSuccess'))
      }
    } catch (err) {
      const status = (err as { status?: number })?.status
      toast.error(
        status === 403
          ? t('features.video.hostControls.error403')
          : t('features.video.hostControls.coHostError'),
      )
    }
  }, [promoteCoHost, demoteCoHost, isCoHost, participantIdentity, t])

  return (
    <div className="flex shrink-0 items-center gap-0.5">
      {/* Mute */}
      <button
        onClick={handleMute}
        disabled={muteParticipant.isPending}
        className={cn(
          'flex h-6 w-6 items-center justify-center rounded text-zinc-400 transition-opacity',
          'hover:bg-zinc-700 hover:text-orange-400',
          'disabled:cursor-not-allowed disabled:opacity-40',
        )}
        title={t('features.video.hostControls.muteParticipant')}
        aria-label={t('features.video.hostControls.muteParticipant')}
      >
        <MicOffIcon />
      </button>

      {/* Kick */}
      <button
        onClick={handleKick}
        disabled={kickParticipant.isPending}
        className={cn(
          'flex h-6 w-6 items-center justify-center rounded text-zinc-400 transition-opacity',
          'hover:bg-zinc-700 hover:text-red-400',
          'disabled:cursor-not-allowed disabled:opacity-40',
        )}
        title={t('features.video.hostControls.kickParticipant')}
        aria-label={t('features.video.hostControls.kickParticipant')}
      >
        <UserXIcon />
      </button>

      {/* Co-host toggle (organizer only) */}
      {isOrganizer && (
        <button
          onClick={handleCoHostToggle}
          disabled={promoteCoHost.isPending || demoteCoHost.isPending}
          className={cn(
            'flex h-6 w-6 items-center justify-center rounded transition-opacity',
            'disabled:cursor-not-allowed disabled:opacity-40',
            isCoHost
              ? 'text-violet-400 hover:bg-zinc-700 hover:text-zinc-300'
              : 'text-zinc-400 hover:bg-zinc-700 hover:text-violet-400',
          )}
          title={isCoHost
            ? t('features.video.hostControls.removeCoHost')
            : t('features.video.hostControls.makeCoHost')}
          aria-label={isCoHost
            ? t('features.video.hostControls.removeCoHost')
            : t('features.video.hostControls.makeCoHost')}
        >
          {isCoHost ? <ShieldMinusIcon /> : <ShieldPlusIcon />}
        </button>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Global host controls (mute-all + lock/unlock) rendered in the call header
// ---------------------------------------------------------------------------

export interface GlobalHostControlsProps {
  meetingId: string
  isLocked: boolean
}

export function GlobalHostControls({ meetingId, isLocked }: GlobalHostControlsProps) {
  const { t } = useTranslation()

  const muteAll = useMuteAllParticipants(meetingId)
  const setLock = useSetMeetingLock(meetingId)

  const handleMuteAll = useCallback(async () => {
    try {
      await muteAll.mutateAsync()
      toast.success(t('features.video.hostControls.muteAllSuccess'))
    } catch (err) {
      const status = (err as { status?: number })?.status
      toast.error(
        status === 403
          ? t('features.video.hostControls.error403')
          : t('features.video.hostControls.muteAllError'),
      )
    }
  }, [muteAll, t])

  const handleLockToggle = useCallback(async () => {
    try {
      await setLock.mutateAsync(!isLocked)
      toast.success(
        isLocked
          ? t('features.video.hostControls.unlockSuccess')
          : t('features.video.hostControls.lockSuccess'),
      )
    } catch (err) {
      const status = (err as { status?: number })?.status
      toast.error(
        status === 403
          ? t('features.video.hostControls.error403')
          : t('features.video.hostControls.lockError'),
      )
    }
  }, [setLock, isLocked, t])

  return (
    <div className="flex items-center gap-2">
      {/* Mute all */}
      <button
        onClick={handleMuteAll}
        disabled={muteAll.isPending}
        className={cn(
          'flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium transition-opacity',
          'bg-zinc-800 text-zinc-300 hover:bg-zinc-700 hover:text-orange-300',
          'disabled:cursor-not-allowed disabled:opacity-40',
        )}
        title={t('features.video.hostControls.muteAll')}
        aria-label={t('features.video.hostControls.muteAll')}
      >
        <UsersSlashIcon />
        <span>{t('features.video.hostControls.muteAll')}</span>
      </button>

      {/* Lock/unlock toggle */}
      <button
        onClick={handleLockToggle}
        disabled={setLock.isPending}
        className={cn(
          'flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium transition-opacity',
          'disabled:cursor-not-allowed disabled:opacity-40',
          isLocked
            ? 'bg-amber-500/20 text-amber-300 hover:bg-amber-500/30'
            : 'bg-zinc-800 text-zinc-300 hover:bg-zinc-700 hover:text-amber-300',
        )}
        title={isLocked ? t('features.video.hostControls.unlock') : t('features.video.hostControls.lock')}
        aria-label={isLocked ? t('features.video.hostControls.unlock') : t('features.video.hostControls.lock')}
      >
        {isLocked ? <LockIcon /> : <UnlockIcon />}
        <span>
          {isLocked
            ? t('features.video.hostControls.unlock')
            : t('features.video.hostControls.lock')}
        </span>
      </button>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Lock indicator badge (header pill, visible when room is locked)
// ---------------------------------------------------------------------------

export interface MeetingLockIndicatorProps {
  isLocked: boolean
  className?: string
}

export function MeetingLockIndicator({ isLocked, className }: MeetingLockIndicatorProps) {
  const { t } = useTranslation()

  if (!isLocked) return null

  return (
    <span
      className={cn(
        'flex items-center gap-1 rounded-full bg-amber-500/20 px-2 py-0.5 text-xs font-medium text-amber-300',
        className,
      )}
      title={t('features.video.hostControls.locked')}
      aria-label={t('features.video.hostControls.locked')}
    >
      <LockIcon small />
      <span>{t('features.video.hostControls.locked')}</span>
    </span>
  )
}
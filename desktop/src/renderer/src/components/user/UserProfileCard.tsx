/**
 * UserProfileCard — reusable popover overlay that appears on avatar/name click.
 *
 * UserProfileTrigger wraps any trigger element (avatar, name, etc.) and shows
 * a profile card with presence status and a "Nachricht senden" action.
 *
 * Data:
 *  - Own user: useAuthStore + useSettingsStore
 *  - Other users: useEmployee (lazy, enabled only when popover opens)
 *  - Presence: usePresenceStore
 *
 * Ping→Chat: useGetOrCreateDM → NavigationIntent { channelId, userId } →
 *   ChatLayout selects the DM channel directly.
 */
import { useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { MessageSquare, ExternalLink, Loader2, AlertCircle } from 'lucide-react'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { usePresenceStore } from '@/stores/presence'
import { useEmployee } from '@/api/hooks/hr-hooks'
import { useGetOrCreateDM } from '@/api/hooks/useChannels'
import { useNavigationStore } from '@/stores/navigation'
import type { PresenceLevel } from '@/api/video-types'
import { cn } from '@/lib/cn'

// ---------------------------------------------------------------------------
// Presence helpers
// ---------------------------------------------------------------------------

const PRESENCE_COLORS: Record<PresenceLevel, string> = {
  online: 'bg-green-500',
  away: 'bg-yellow-500',
  dnd: 'bg-red-500',
  in_call: 'bg-purple-500',
  offline: 'bg-gray-400',
}

/** Compute presence display label — mirrors PresenceIndicator approach to avoid typed-i18n overload crash. */
function usePresenceLabel(status: PresenceLevel): string {
  const { t } = useTranslation()
  const labels: Record<PresenceLevel, string> = {
    online: t('features.presence.status.online'),
    away: t('features.presence.status.away'),
    dnd: t('features.presence.status.dnd'),
    in_call: t('features.presence.status.inCall'),
    offline: t('features.presence.status.offline'),
  }
  return labels[status]
}

function getInitials(name: string): string {
  const parts = name.trim().split(/\s+/)
  if (parts.length >= 2) return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
  return (parts[0]?.[0] ?? '?').toUpperCase()
}

// ---------------------------------------------------------------------------
// Card content for other users (fetches HR data lazily)
// ---------------------------------------------------------------------------

interface OtherUserCardContentProps {
  userId: string
  name: string
  email?: string
  presence: PresenceLevel
  onSendMessage: () => void
  isSending: boolean
  sendError: boolean
}

function OtherUserCardContent({
  userId,
  name: fallbackName,
  email: fallbackEmail,
  presence,
  onSendMessage,
  isSending,
  sendError,
}: OtherUserCardContentProps) {
  const { t } = useTranslation()
  const { data: employeeData, isLoading, isError } = useEmployee(userId)
  const presenceLabel = usePresenceLabel(presence)

  const employee = employeeData
  const displayName = employee?.userName ?? fallbackName
  const displayEmail = employee?.userEmail ?? fallbackEmail
  const position = employee?.positionTitle
  const department = employee?.department

  const initials = getInitials(displayName)

  return (
    <div className="flex flex-col gap-4">
      {/* Header with avatar + basic info */}
      <div className="flex items-start gap-3">
        <div className="relative shrink-0">
          <Avatar className="h-12 w-12">
            <AvatarFallback className="text-sm font-medium bg-primary/10 text-primary">
              {initials}
            </AvatarFallback>
          </Avatar>
          <span
            className={cn(
              'absolute -bottom-0.5 -right-0.5 h-3 w-3 rounded-full border-2 border-popover',
              PRESENCE_COLORS[presence],
            )}
          />
        </div>

        <div className="min-w-0 flex-1">
          {isLoading ? (
            <div className="flex items-center gap-2">
              <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
              <span className="text-sm text-muted-foreground">{t('common.loading')}</span>
            </div>
          ) : isError ? (
            <div className="flex items-center gap-1.5 text-sm text-destructive">
              <AlertCircle className="h-4 w-4 shrink-0" />
              <span>{t('profil.card.loadError')}</span>
            </div>
          ) : (
            <>
              <p className="text-sm font-semibold text-foreground leading-tight">{displayName}</p>
              {position && (
                <p className="text-xs text-muted-foreground mt-0.5">{position}</p>
              )}
              {department && (
                <p className="text-xs text-muted-foreground">
                  {t('profil.info.department')}: {department}
                </p>
              )}
            </>
          )}
        </div>
      </div>

      {/* Presence badge */}
      <div
        className="flex items-center gap-2 rounded-md bg-secondary/50 px-3 py-2"
        aria-label={presenceLabel}
      >
        <span
          className={cn('h-2 w-2 rounded-full shrink-0', PRESENCE_COLORS[presence])}
          aria-hidden="true"
        />
        <span className="text-xs text-foreground">
          {presenceLabel}
        </span>
      </div>

      {/* Email if available */}
      {displayEmail && (
        <p className="truncate text-xs text-muted-foreground">{displayEmail}</p>
      )}

      {/* Action: Nachricht senden */}
      {sendError && (
        <p className="text-xs text-destructive">{t('profil.card.sendError')}</p>
      )}
      <Button
        size="sm"
        className="w-full gap-2"
        onClick={onSendMessage}
        disabled={isSending}
      >
        {isSending ? (
          <Loader2 className="h-4 w-4 animate-spin" />
        ) : (
          <MessageSquare className="h-4 w-4" />
        )}
        {isSending ? t('profil.card.sending') : t('profil.card.sendMessage')}
      </Button>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Card content for own user
// ---------------------------------------------------------------------------

function OwnUserCardContent() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const user = useAuthStore((s) => s.user)
  const profile = useSettingsStore((s) => s.profile)
  const myStatus = usePresenceStore((s) => s.myStatus)
  const presenceLabel = usePresenceLabel(myStatus)

  const displayName = user
    ? `${user.firstName} ${user.lastName}`.trim() || user.email
    : profile.firstName + ' ' + profile.lastName

  const displayEmail = user?.email ?? profile.email
  const position = profile.position
  const initials = getInitials(displayName)

  return (
    <div className="flex flex-col gap-4">
      {/* Header */}
      <div className="flex items-start gap-3">
        <div className="relative shrink-0">
          <Avatar className="h-12 w-12">
            <AvatarFallback className="text-sm font-medium bg-primary/10 text-primary">
              {initials}
            </AvatarFallback>
          </Avatar>
          <span
            className={cn(
              'absolute -bottom-0.5 -right-0.5 h-3 w-3 rounded-full border-2 border-popover',
              PRESENCE_COLORS[myStatus],
            )}
          />
        </div>

        <div className="min-w-0 flex-1">
          <p className="text-sm font-semibold text-foreground leading-tight">{displayName}</p>
          {position && (
            <p className="text-xs text-muted-foreground mt-0.5">{position}</p>
          )}
        </div>
      </div>

      {/* Presence badge */}
      <div
        className="flex items-center gap-2 rounded-md bg-secondary/50 px-3 py-2"
        aria-label={presenceLabel}
      >
        <span
          className={cn('h-2 w-2 rounded-full shrink-0', PRESENCE_COLORS[myStatus])}
          aria-hidden="true"
        />
        <span className="text-xs text-foreground">
          {presenceLabel}
        </span>
      </div>

      {/* Email */}
      {displayEmail && (
        <p className="truncate text-xs text-muted-foreground">{displayEmail}</p>
      )}

      {/* Action: open own profile */}
      <Button
        size="sm"
        variant="outline"
        className="w-full gap-2"
        onClick={() => navigate('/profil')}
      >
        <ExternalLink className="h-4 w-4" />
        {t('profil.card.openProfile')}
      </Button>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Public API: UserProfileTrigger
// ---------------------------------------------------------------------------

export interface UserProfileTriggerProps {
  userId: string
  /** Display name fallback (shown before HR data loads). */
  name: string
  /** Email fallback */
  email?: string
  /** Wrap any trigger element — rendered as-is with click handler via asChild */
  children: React.ReactNode
}

export function UserProfileTrigger({
  userId,
  name,
  email,
  children,
}: UserProfileTriggerProps) {
  const [open, setOpen] = useState(false)
  const currentUser = useAuthStore((s) => s.user)
  const isOwnProfile = !!currentUser && currentUser.id === userId

  const presence = usePresenceStore((s) => s.presenceMap[userId] ?? 'offline')

  const setIntent = useNavigationStore((s) => s.setIntent)
  const navigate = useNavigate()
  const getDM = useGetOrCreateDM()

  const [isSending, setIsSending] = useState(false)
  const [sendError, setSendError] = useState(false)

  const handleSendMessage = useCallback(async () => {
    setIsSending(true)
    setSendError(false)
    try {
      const result = await getDM.mutateAsync(userId)
      const channelId = result?.channel?.id
      setOpen(false)
      setIntent({
        type: 'send-message',
        data: { name, userId, ...(channelId ? { channelId } : {}) },
      })
      navigate('/kommunikation?bereich=team')
    } catch {
      setSendError(true)
    } finally {
      setIsSending(false)
    }
  }, [getDM, userId, name, setIntent, navigate])

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        asChild
        onClick={(e) => {
          // Prevent list row-click handlers from firing when clicking the avatar
          e.stopPropagation()
        }}
      >
        {children}
      </PopoverTrigger>

      <PopoverContent
        data-testid="user-profile-card"
        className="w-72 p-4"
        side="top"
        align="start"
        sideOffset={8}
      >
        {isOwnProfile ? (
          <OwnUserCardContent />
        ) : (
          <OtherUserCardContent
            userId={userId}
            name={name}
            email={email}
            presence={presence}
            onSendMessage={handleSendMessage}
            isSending={isSending}
            sendError={sendError}
          />
        )}
      </PopoverContent>
    </Popover>
  )
}

import { useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Camera, Save, X, Mail, ChevronDown, Bell, BellOff, Volume2, VolumeX, Palette, Shield, ExternalLink } from 'lucide-react'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useSettingsStore } from '@/stores/settings'
import { useAuthStore } from '@/stores/auth'
import { usePresenceStore } from '@/stores/presence'
import { useNotificationsStore } from '@/stores/notifications'
import type { PresenceLevel } from '@/api/video-types'
import { LazyRichTextEditor as RichTextEditor } from '@/components/shared/RichTextEditor'
import { toast } from 'sonner'
import { useNavigate } from 'react-router-dom'
import {
  useDNDStatus,
  useEnableDND,
  useDisableDND,
} from '@/api/hooks/useNotifications'

// Manually selectable presence states ('in_call' is system-driven). Dot colors
// match the kommunikation presence section.
const PRESENCE_OPTIONS = [
  { value: 'online', labelKey: 'profil.presence.online', dot: 'bg-emerald-500' },
  { value: 'away', labelKey: 'profil.presence.away', dot: 'bg-amber-400' },
  { value: 'dnd', labelKey: 'profil.presence.dnd', dot: 'bg-red-500' },
  { value: 'offline', labelKey: 'profil.presence.invisible', dot: 'bg-slate-400' },
] as const satisfies ReadonlyArray<{ value: PresenceLevel; labelKey: string; dot: string }>

// Keep stored avatars comfortably below the localStorage quota (mock-first —
// the real upload endpoint is pending, see backend-handover).
const MAX_AVATAR_BYTES = 1.5 * 1024 * 1024

// Role keys that map to i18n translations. Unknown roles are displayed raw.
const KNOWN_ROLE_KEYS = ['admin', 'manager', 'employee'] as const satisfies ReadonlyArray<string>

export default function ProfilTab() {
  const { t, i18n } = useTranslation()
  const profile = useSettingsStore((s) => s.profile)
  const updateProfile = useSettingsStore((s) => s.updateProfile)
  const mailSignature = useSettingsStore((s) => s.mail.signature)
  const updateMail = useSettingsStore((s) => s.updateMail)
  const user = useAuthStore((s) => s.user)
  const navigate = useNavigate()

  const [form, setForm] = useState({ ...profile })
  const [hasChanges, setHasChanges] = useState(false)
  const [signatureDraft, setSignatureDraft] = useState(mailSignature)
  const [signatureChanged, setSignatureChanged] = useState(false)

  // Presence (settings panel pattern): persisted in stores/presence.myStatus,
  // mirrored by the dot on the avatar and the status dots in team/kommunikation.
  const myStatus = usePresenceStore((s) => s.myStatus)
  const setMyStatus = usePresenceStore((s) => s.setMyStatus)
  const currentPresence =
    PRESENCE_OPTIONS.find((o) => o.value === myStatus) ?? PRESENCE_OPTIONS[0]

  // Sound toggle — purely local, always available.
  const soundEnabled = useNotificationsStore((s) => s.soundEnabled)
  const toggleSound = useNotificationsStore((s) => s.toggleSound)

  // DND — backend-driven via React Query. isLoading = backend unreachable in dev.
  const { data: dndStatus, isLoading: dndLoading, isError: dndError } = useDNDStatus()
  const enableDND = useEnableDND()
  const disableDND = useDisableDND()
  const dndBackendAvailable = !dndLoading && !dndError

  const handleToggleDND = () => {
    if (!dndBackendAvailable) return
    if (dndStatus?.is_active) {
      disableDND.mutate(undefined, {
        onSuccess: () => toast.success(t('settings.notifications.dnd.deactivated')),
      })
    } else {
      enableDND.mutate(undefined, {
        onSuccess: () => toast.success(t('settings.notifications.dnd.activated')),
      })
    }
  }

  // Avatar upload UI — local preview + mock-first persistence as data URL in
  // the settings store; the real upload endpoint is Luke's backend (handover).
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [avatarPreview, setAvatarPreview] = useState<string | null>(null)

  const handleAvatarFile = (file: File | undefined) => {
    if (!file) return
    if (!file.type.startsWith('image/')) {
      toast.error(t('profil.avatar.errorType'))
      return
    }
    if (file.size > MAX_AVATAR_BYTES) {
      toast.error(t('profil.avatar.errorSize'))
      return
    }
    setAvatarPreview(URL.createObjectURL(file))
    const reader = new FileReader()
    reader.onload = () => {
      if (typeof reader.result === 'string') {
        updateProfile({ avatarUrl: reader.result })
        toast.success(t('profil.avatar.saved'))
      }
    }
    reader.readAsDataURL(file)
  }

  const initials = `${form.firstName.charAt(0)}${form.lastName.charAt(0)}`.toUpperCase()

  // Derive role label: map known roles to i18n keys, fall back to first raw role.
  // user may be null in dev without backend (Electron stub + no login).
  const roleLabel: string = (() => {
    if (!user?.roles?.length) return '—'
    const firstRole = user.roles[0]
    if ((KNOWN_ROLE_KEYS as ReadonlyArray<string>).includes(firstRole)) {
      return t(`profil.role.${firstRole}` as Parameters<typeof t>[0])
    }
    return firstRole
  })()

  // Email from auth store (single source of truth), falls back to profile store.
  const displayEmail = user?.email ?? form.email

  // "Member since" — demo join date from the profile seed (real backend will
  // supply this via /auth/me). Formatted as month + year in the active locale.
  const memberSince = profile.joinedAt
    ? new Date(profile.joinedAt).toLocaleDateString(i18n.language, {
        year: 'numeric',
        month: 'long',
      })
    : null

  const handleChange = (field: string, value: string) => {
    setForm((prev) => ({ ...prev, [field]: value }))
    setHasChanges(true)
  }

  const handleSave = () => {
    updateProfile(form)
    setHasChanges(false)
    toast.success(t('profil.info.profileSaved'))
  }

  const handleCancel = () => {
    setForm({ ...profile })
    setHasChanges(false)
  }

  return (
    <div className="max-w-3xl mx-auto p-6 space-y-6">
      {/* Profile Header Card */}
      <div className="rounded-xl border border-border bg-gradient-to-br from-primary/10 via-card to-card p-6">
        <div className="flex items-start gap-5">
          <div className="relative">
            <Avatar className="h-20 w-20 ring-4 ring-primary/20">
              {(avatarPreview || profile.avatarUrl) && (
                <AvatarImage src={avatarPreview ?? profile.avatarUrl ?? undefined} alt="" />
              )}
              <AvatarFallback className="text-2xl font-bold bg-primary/10 text-primary">
                {initials}
              </AvatarFallback>
            </Avatar>
            {/* Presence dot mirrors the picked status */}
            <span
              className={`absolute top-0 right-0 h-4 w-4 rounded-full ring-2 ring-card ${currentPresence.dot}`}
              aria-hidden="true"
            />
            <input
              ref={fileInputRef}
              type="file"
              accept="image/*"
              className="hidden"
              onChange={(e) => {
                handleAvatarFile(e.target.files?.[0])
                e.target.value = ''
              }}
            />
            <button
              onClick={() => fileInputRef.current?.click()}
              aria-label={t('profil.avatar.change')}
              className="absolute -bottom-1 -right-1 flex h-8 w-8 items-center justify-center rounded-full bg-primary text-primary-foreground shadow-md hover:bg-primary/90 transition-colors"
            >
              <Camera className="h-4 w-4" />
            </button>
          </div>
          <div className="flex-1 space-y-1">
            <h2 className="text-xl font-bold text-foreground">
              {form.firstName} {form.lastName}
            </h2>
            <p className="text-sm text-muted-foreground">{form.position}</p>
            <div className="flex items-center gap-2 mt-2">
              <Badge variant="outline" className="border-primary/30 text-primary">
                {roleLabel}
              </Badge>
              {/* Presence picker — persisted via stores/presence (myStatus) */}
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <button
                    className="inline-flex items-center gap-1.5 rounded-full border border-border px-2.5 py-0.5 text-xs font-medium text-foreground hover:bg-secondary transition-colors"
                    aria-label={t('profil.presence.pick')}
                  >
                    <span className={`h-2 w-2 rounded-full ${currentPresence.dot}`} />
                    {t(currentPresence.labelKey)}
                    <ChevronDown className="h-3 w-3 text-muted-foreground" />
                  </button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start">
                  {PRESENCE_OPTIONS.map((opt) => (
                    <DropdownMenuItem key={opt.value} onClick={() => setMyStatus(opt.value)}>
                      <span className={`mr-2 h-2 w-2 rounded-full ${opt.dot}`} />
                      {t(opt.labelKey)}
                    </DropdownMenuItem>
                  ))}
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          </div>
        </div>
      </div>

      {/* Notification Quick-Card */}
      <div className="rounded-xl border border-border bg-card p-6 space-y-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Bell className="h-5 w-5 text-primary" />
            <h3 className="font-semibold text-foreground">{t('profil.notifications.title')}</h3>
          </div>
          <button
            onClick={() => navigate('/settings', { state: { tab: 'notifications' } })}
            className="inline-flex items-center gap-1 text-xs text-primary hover:underline"
            aria-label={t('profil.notifications.allSettings')}
          >
            {t('profil.notifications.allSettings')}
            <ExternalLink className="h-3 w-3" />
          </button>
        </div>

        <div className="space-y-3">
          {/* DND toggle */}
          <div className="flex items-center justify-between rounded-lg border border-border bg-background px-4 py-3">
            <div className="flex items-center gap-3">
              <BellOff className="h-4 w-4 text-muted-foreground shrink-0" />
              <div>
                <p className="text-sm font-medium text-foreground">{t('profil.notifications.dnd.label')}</p>
                <p className="text-xs text-muted-foreground">
                  {dndLoading
                    ? t('profil.notifications.dnd.loading')
                    : dndError
                      ? t('profil.notifications.dnd.unavailable')
                      : dndStatus?.is_active
                        ? t('settings.notifications.dnd.statusActive')
                        : t('settings.notifications.dnd.statusInactive')}
                </p>
              </div>
            </div>
            <Switch
              checked={dndBackendAvailable ? (dndStatus?.is_active ?? false) : false}
              onCheckedChange={handleToggleDND}
              disabled={!dndBackendAvailable || enableDND.isPending || disableDND.isPending}
              aria-label={t('profil.notifications.dnd.label')}
            />
          </div>

          {/* Sound toggle */}
          <div className="flex items-center justify-between rounded-lg border border-border bg-background px-4 py-3">
            <div className="flex items-center gap-3">
              {soundEnabled ? (
                <Volume2 className="h-4 w-4 text-muted-foreground shrink-0" />
              ) : (
                <VolumeX className="h-4 w-4 text-muted-foreground shrink-0" />
              )}
              <div>
                <p className="text-sm font-medium text-foreground">{t('profil.notifications.sound.label')}</p>
                <p className="text-xs text-muted-foreground">{t('profil.notifications.sound.desc')}</p>
              </div>
            </div>
            <Switch
              checked={soundEnabled}
              onCheckedChange={toggleSound}
              aria-label={t('profil.notifications.sound.label')}
            />
          </div>
        </div>
      </div>

      {/* Form */}
      <div className="rounded-xl border border-border bg-card p-6 space-y-5">
        <h3 className="font-semibold text-foreground">{t('profil.info.personalInfo')}</h3>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('profil.field.firstName')}</label>
            <Input
              value={form.firstName}
              onChange={(e) => handleChange('firstName', e.target.value)}
            />
          </div>
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('profil.field.lastName')}</label>
            <Input
              value={form.lastName}
              onChange={(e) => handleChange('lastName', e.target.value)}
            />
          </div>
        </div>

        <div className="space-y-1.5">
          <label className="text-sm font-medium text-foreground">{t('profil.field.email')}</label>
          <Input
            type="email"
            value={form.email}
            onChange={(e) => handleChange('email', e.target.value)}
          />
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('profil.field.phone')}</label>
            <Input
              value={form.phone}
              onChange={(e) => handleChange('phone', e.target.value)}
            />
          </div>
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('profil.field.position')}</label>
            <Input
              value={form.position}
              onChange={(e) => handleChange('position', e.target.value)}
            />
          </div>
        </div>

        <div className="space-y-1.5">
          <label className="text-sm font-medium text-foreground">{t('profil.field.bio')}</label>
          <textarea
            value={form.bio}
            onChange={(e) => handleChange('bio', e.target.value)}
            rows={3}
            className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          />
        </div>
      </div>

      {/* Account Info — real data from auth store */}
      <div className="rounded-xl border border-border bg-card p-6 space-y-4">
        <h3 className="font-semibold text-foreground">{t('profil.info.accountInfo')}</h3>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <p className="text-xs text-muted-foreground">{t('profil.info.role')}</p>
            <p className="text-sm font-medium text-foreground">{roleLabel}</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">{t('profil.field.email')}</p>
            <p className="text-sm font-medium text-foreground">{displayEmail || '—'}</p>
          </div>
          {memberSince && (
            <div>
              <p className="text-xs text-muted-foreground">{t('profil.info.memberSince')}</p>
              <p className="text-sm font-medium text-foreground">{memberSince}</p>
            </div>
          )}
        </div>
      </div>

      {/* Settings Shortcuts */}
      <div className="rounded-xl border border-border bg-card p-6 space-y-3">
        <h3 className="font-semibold text-foreground">{t('profil.shortcuts.title')}</h3>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <button
            onClick={() => navigate('/settings', { state: { tab: 'appearance' } })}
            className="flex items-center gap-3 rounded-lg border border-border bg-background px-4 py-3 text-left hover:bg-secondary transition-colors group"
            aria-label={t('profil.shortcuts.appearance')}
          >
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 group-hover:bg-primary/15 transition-colors">
              <Palette className="h-4 w-4 text-primary" />
            </div>
            <div>
              <p className="text-sm font-medium text-foreground">{t('profil.shortcuts.appearance')}</p>
              <p className="text-xs text-muted-foreground">{t('profil.shortcuts.appearanceDesc')}</p>
            </div>
          </button>
          <button
            onClick={() => navigate('/settings', { state: { tab: 'security' } })}
            className="flex items-center gap-3 rounded-lg border border-border bg-background px-4 py-3 text-left hover:bg-secondary transition-colors group"
            aria-label={t('profil.shortcuts.security')}
          >
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 group-hover:bg-primary/15 transition-colors">
              <Shield className="h-4 w-4 text-primary" />
            </div>
            <div>
              <p className="text-sm font-medium text-foreground">{t('profil.shortcuts.security')}</p>
              <p className="text-xs text-muted-foreground">{t('profil.shortcuts.securityDesc')}</p>
            </div>
          </button>
        </div>
      </div>

      {/* E-Mail-Signatur */}
      <div className="rounded-xl border border-border bg-card p-6 space-y-4">
        <div className="flex items-center gap-2">
          <Mail className="h-5 w-5 text-primary" />
          <h3 className="font-semibold text-foreground">{t('profil.signature.title')}</h3>
        </div>
        <p className="text-sm text-muted-foreground">
          {t('profil.signature.description')}
        </p>
        <RichTextEditor
          content={signatureDraft}
          onChange={(html) => {
            setSignatureDraft(html)
            setSignatureChanged(html !== mailSignature)
          }}
          placeholder={t('profil.signature.placeholder')}
          compact
          showFooter={false}
          minHeight="80px"
          maxHeight="200px"
        />
        {signatureChanged && (
          <div className="flex justify-end gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setSignatureDraft(mailSignature)
                setSignatureChanged(false)
              }}
            >
              {t('common.cancel')}
            </Button>
            <Button
              size="sm"
              onClick={() => {
                updateMail({ signature: signatureDraft })
                setSignatureChanged(false)
                toast.success(t('profil.signature.saved'))
              }}
            >
              <Save className="h-3.5 w-3.5 mr-1.5" />
              {t('profil.signature.save')}
            </Button>
          </div>
        )}
      </div>

      {/* Save/Cancel */}
      {hasChanges && (
        <div className="flex justify-end gap-3 sticky bottom-0 bg-background/80 backdrop-blur-sm py-4 border-t border-border -mx-6 px-6">
          <Button variant="outline" onClick={handleCancel}>
            <X className="h-4 w-4 mr-2" />
            {t('common.cancel')}
          </Button>
          <Button onClick={handleSave}>
            <Save className="h-4 w-4 mr-2" />
            {t('common.save')}
          </Button>
        </div>
      )}
    </div>
  )
}

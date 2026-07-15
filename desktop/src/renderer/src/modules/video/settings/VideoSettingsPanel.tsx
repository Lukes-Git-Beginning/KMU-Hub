import { useTranslation } from 'react-i18next'
import { Video, ShieldCheck, ChevronDown } from 'lucide-react'
import { ModuleSettingsShell, type ModuleSettingsSection } from '@/components/shared'
import { useVideoTenantStore, type RecordingPolicy } from '@/stores/videoTenant'
import { VideoDevicePrefsControls } from '../VideoDeviceSettings'

// ─── Persönlich ──────────────────────────────────────────────────

function VideoPersonalPrefs() {
  // Device defaults live in the videoPrefs store; controls are shared with the
  // in-page settings tab. The compact overlay omits the live camera preview.
  return <VideoDevicePrefsControls />
}

// ─── Für alle (Tenant) ───────────────────────────────────────────

function VideoTenantSettings() {
  const { t } = useTranslation()
  const recordingPolicy = useVideoTenantStore((s) => s.recordingPolicy)
  const defaultRoom = useVideoTenantStore((s) => s.defaultRoom)
  const setRecordingPolicy = useVideoTenantStore((s) => s.setRecordingPolicy)
  const setDefaultRoom = useVideoTenantStore((s) => s.setDefaultRoom)

  return (
    <div className="space-y-5">
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('video.settings.tenant.recordingPolicy')}</label>
        <div className="relative w-64 max-w-full">
          <select
            value={recordingPolicy}
            onChange={(e) => setRecordingPolicy(e.target.value as RecordingPolicy)}
            className="w-full appearance-none rounded-lg border border-border bg-card px-3 py-2 pr-8 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
          >
            <option value="consent">{t('video.settings.tenant.recording.consent')}</option>
            <option value="always">{t('video.settings.tenant.recording.always')}</option>
            <option value="off">{t('video.settings.tenant.recording.off')}</option>
          </select>
          <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        </div>
        <p className="text-xs text-muted-foreground">{t('video.settings.tenant.recordingPolicyHint')}</p>
      </div>

      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('video.settings.tenant.defaultRoom')}</label>
        <input
          type="text"
          value={defaultRoom}
          onChange={(e) => setDefaultRoom(e.target.value)}
          placeholder={t('video.settings.tenant.defaultRoomPlaceholder')}
          className="w-64 max-w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
        />
        <p className="text-xs text-muted-foreground">{t('video.settings.tenant.defaultRoomHint')}</p>
      </div>
    </div>
  )
}

// ─── Panel ───────────────────────────────────────────────────────

/**
 * VideoSettingsPanel — the "Video & Anrufe" entry of the module-settings overlay.
 * Personal device prefs (microphone/speaker/camera/background) apply per user;
 * the tenant block (recording policy, default room) is editable only by a
 * Meetings-Modul-Leiter / admin.
 */
export function VideoSettingsPanel() {
  const sections: ModuleSettingsSection[] = [
    {
      id: 'personal',
      titleKey: 'video.settings.personal.title',
      descriptionKey: 'video.settings.personal.desc',
      scope: 'personal',
      icon: Video,
      children: <VideoPersonalPrefs />,
    },
    {
      id: 'tenant',
      titleKey: 'video.settings.tenant.title',
      descriptionKey: 'video.settings.tenant.desc',
      scope: 'tenant',
      icon: ShieldCheck,
      children: <VideoTenantSettings />,
    },
  ]

  return (
    <ModuleSettingsShell
      moduleId="meetings"
      titleKey="video.settings.panel.title"
      descriptionKey="video.settings.panel.subtitle"
      sections={sections}
    />
  )
}

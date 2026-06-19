import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { TimePicker } from '@/components/shared/TimePicker'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Separator } from '@/components/ui/separator'
import {
  Save,
  Bell,
  Moon,
  BellOff,
  VolumeX,
  Volume2,
  Loader2,
  Trash2,
} from 'lucide-react'
import { toast } from 'sonner'
import { useSettingsStore, type NotificationModule, type NotificationPrefs } from '@/stores/settings'
import { useNotificationsStore } from '@/stores/notifications'
import { playNotificationSound } from '@/lib/notification-sound'
import {
  useQuietHours,
  useUpdateQuietHours,
  useDNDStatus,
  useEnableDND,
  useDisableDND,
  useMutedResources,
  useUnmuteResource,
} from '@/api/hooks/useNotifications'
import { formatDate, formatTime } from '@/lib/format'

// ---------------------------------------------------------------------------
// Module matrix (same as before, but in a dedicated component)
// ---------------------------------------------------------------------------

const NOTIFICATION_MODULE_KEYS: { key: NotificationModule; labelKey: string; descKey: string }[] = [
  { key: 'messages', labelKey: 'settings.notifications.module.messages.label', descKey: 'settings.notifications.module.messages.desc' },
  { key: 'tasks', labelKey: 'settings.notifications.module.tasks.label', descKey: 'settings.notifications.module.tasks.desc' },
  { key: 'meetings', labelKey: 'settings.notifications.module.meetings.label', descKey: 'settings.notifications.module.meetings.desc' },
  { key: 'mails', labelKey: 'settings.notifications.module.mails.label', descKey: 'settings.notifications.module.mails.desc' },
  { key: 'calendar', labelKey: 'settings.notifications.module.calendar.label', descKey: 'settings.notifications.module.calendar.desc' },
  { key: 'team', labelKey: 'settings.notifications.module.team.label', descKey: 'settings.notifications.module.team.desc' },
  { key: 'finance', labelKey: 'settings.notifications.module.finance.label', descKey: 'settings.notifications.module.finance.desc' },
]

const CHANNEL_KEYS: { key: keyof NotificationPrefs; labelKey: string }[] = [
  { key: 'email', labelKey: 'settings.notifications.channel.email' },
  { key: 'push', labelKey: 'settings.notifications.channel.push' },
  { key: 'inApp', labelKey: 'settings.notifications.channel.inApp' },
]

const DAY_LABEL_KEYS = [
  'settings.notifications.day.sun',
  'settings.notifications.day.mon',
  'settings.notifications.day.tue',
  'settings.notifications.day.wed',
  'settings.notifications.day.thu',
  'settings.notifications.day.fri',
  'settings.notifications.day.sat',
]

export function NotificationSettingsTab({ embedded = false }: { embedded?: boolean } = {}) {
  const { t } = useTranslation()
  const { notifications, updateNotification } = useSettingsStore()
  const soundEnabled = useNotificationsStore((s) => s.soundEnabled)
  const toggleSound = useNotificationsStore((s) => s.toggleSound)

  // Quiet Hours (from API)
  const { data: quietHours, isLoading: qhLoading } = useQuietHours()
  const qhMutation = useUpdateQuietHours()

  const [qhStart, setQhStart] = useState('22:00')
  const [qhEnd, setQhEnd] = useState('07:00')
  const [qhDays, setQhDays] = useState<number[]>([0, 1, 2, 3, 4, 5, 6])
  const [qhActive, setQhActive] = useState(true)

   
  useEffect(() => {
    if (!quietHours) return
    // eslint-disable-next-line react-hooks/set-state-in-effect -- sync form fields from prop/API data
    setQhStart(quietHours.start_time ?? '22:00')
    setQhEnd(quietHours.end_time ?? '07:00')
    setQhDays(quietHours.days ?? [0, 1, 2, 3, 4, 5, 6])
    setQhActive(quietHours.is_active ?? true)
  }, [quietHours])

  const toggleDay = (day: number) => {
    setQhDays((prev) =>
      prev.includes(day) ? prev.filter((d) => d !== day) : [...prev, day],
    )
  }

  const handleSaveQuietHours = () => {
    qhMutation.mutate(
      { start_time: qhStart, end_time: qhEnd, days: qhDays, is_active: qhActive },
      { onSuccess: () => toast.success(t('settings.notifications.quietHours.saved')) },
    )
  }

  // DND
  const { data: dndStatus } = useDNDStatus()
  const enableDND = useEnableDND()
  const disableDND = useDisableDND()

  const handleToggleDND = () => {
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

  // Muted resources
  const { data: mutedResources } = useMutedResources()
  const unmuteMutation = useUnmuteResource()

  return (
    <div className={embedded ? '' : 'max-w-2xl'}>
      {!embedded && (
        <>
          <h2 className="text-foreground mb-1">{t('settings.notifications.title')}</h2>
          <p className="text-sm text-muted-foreground mb-6">
            {t('settings.notifications.subtitle')}
          </p>
        </>
      )}

      {/* ── Module × Channel Matrix ──────────────────── */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Bell className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">{t('settings.notifications.matrix.title')}</h3>
        </div>

        <div className="rounded-lg border border-border bg-card overflow-hidden">
          {/* Header */}
          <div className="grid grid-cols-[1fr_70px_70px_70px] gap-2 px-4 py-3 text-xs font-medium text-muted-foreground border-b border-border bg-secondary/30">
            <span>{t('settings.notifications.matrix.moduleCol')}</span>
            {CHANNEL_KEYS.map((ch) => (
              <span key={ch.key} className="text-center">{t(ch.labelKey)}</span>
            ))}
          </div>

          {/* Rows */}
          {NOTIFICATION_MODULE_KEYS.map((mod) => (
            <div key={mod.key} className="grid grid-cols-[1fr_70px_70px_70px] gap-2 items-center px-4 py-3 border-b border-border-muted last:border-b-0">
              <div>
                <p className="text-sm text-foreground">{t(mod.labelKey)}</p>
                <p className="text-xs text-muted-foreground">{t(mod.descKey)}</p>
              </div>
              {CHANNEL_KEYS.map((ch) => (
                <div key={ch.key} className="flex justify-center">
                  <Switch
                    checked={notifications[mod.key][ch.key]}
                    onCheckedChange={(v) => updateNotification(mod.key, ch.key, v)}
                  />
                </div>
              ))}
            </div>
          ))}
        </div>

        <p className="text-xs text-muted-foreground mt-3">{t('settings.notifications.matrix.autoSave')}</p>
      </section>

      <Separator className="mb-8" />

      {/* ── Ton ──────────────────────────────────────── */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          {soundEnabled ? (
            <Volume2 className="h-4 w-4 text-muted-foreground" />
          ) : (
            <VolumeX className="h-4 w-4 text-muted-foreground" />
          )}
          <h3 className="text-sm font-medium text-foreground">{t('settings.notifications.sound.title')}</h3>
        </div>

        <div className="flex items-center justify-between rounded-lg border border-border bg-card p-4">
          <div>
            <p className="text-sm text-foreground">{t('settings.notifications.sound.label')}</p>
            <p className="text-xs text-muted-foreground">{t('settings.notifications.sound.desc')}</p>
          </div>
          <div className="flex items-center gap-3">
            <Button
              size="sm"
              variant="outline"
              onClick={() => playNotificationSound()}
              disabled={!soundEnabled}
            >
              {t('settings.notifications.sound.test')}
            </Button>
            <Switch checked={soundEnabled} onCheckedChange={toggleSound} />
          </div>
        </div>
      </section>

      <Separator className="mb-8" />

      {/* ── DND (Bitte nicht stören) ────────────────── */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <BellOff className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">{t('settings.notifications.dnd.title')}</h3>
        </div>

        <div className="flex items-center justify-between rounded-lg border border-border bg-card p-4">
          <div>
            <p className="text-sm text-foreground">
              {dndStatus?.is_active ? t('settings.notifications.dnd.statusActive') : t('settings.notifications.dnd.statusInactive')}
            </p>
            <p className="text-xs text-muted-foreground">
              {dndStatus?.is_active
                ? dndStatus.expires_at
                  ? t('settings.notifications.dnd.expiresAt', { time: formatTime(dndStatus.expires_at, { hour: '2-digit', minute: '2-digit' }) })
                  : t('settings.notifications.dnd.noExpiry')
                : t('settings.notifications.dnd.hint')}
            </p>
          </div>
          <Button
            size="sm"
            variant={dndStatus?.is_active ? 'outline' : 'default'}
            onClick={handleToggleDND}
            disabled={enableDND.isPending || disableDND.isPending}
          >
            {dndStatus?.is_active ? t('settings.notifications.dnd.disable') : t('settings.notifications.dnd.enable')}
          </Button>
        </div>
      </section>

      <Separator className="mb-8" />

      {/* ── Muted Resources ──────────────────────────── */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <VolumeX className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">{t('settings.notifications.muted.title')}</h3>
        </div>

        {(!mutedResources || mutedResources.length === 0) ? (
          <p className="text-xs text-muted-foreground italic">{t('settings.notifications.muted.empty')}</p>
        ) : (
          <div className="space-y-1.5">
            {mutedResources.map((mute) => (
              <div
                key={mute.id}
                className="flex items-center justify-between rounded-md border border-border px-3 py-2"
              >
                <div>
                  <p className="text-sm text-foreground">{mute.resource_label ?? mute.resource_id}</p>
                  <p className="text-[10px] text-muted-foreground">
                    {mute.resource_type} · {t('settings.notifications.muted.since', { date: formatDate(mute.muted_at) })}
                  </p>
                </div>
                <button
                  onClick={() => unmuteMutation.mutate(mute.id, {
                    onSuccess: () => toast.success(t('settings.notifications.muted.unmuteSuccess')),
                  })}
                  className="text-muted-foreground hover:text-error transition-colors"
                  disabled={unmuteMutation.isPending}
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
            ))}
          </div>
        )}
      </section>

      <Separator className="mb-8" />

      {/* ── Quiet Hours (last section — collapsing it must not push other content) ─ */}
      <section>
        <div className="flex items-center gap-2 mb-4">
          <Moon className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">{t('settings.notifications.quietHours.title')}</h3>
        </div>

        {qhLoading ? (
          <div className="flex items-center gap-2 text-muted-foreground py-4">
            <Loader2 className="h-4 w-4 animate-spin" />
            <span className="text-sm">{t('settings.notifications.quietHours.loading')}</span>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="flex items-center justify-between rounded-lg border border-border bg-card p-3">
              <div>
                <p className="text-sm text-foreground">{t('settings.notifications.quietHours.activeLabel')}</p>
                <p className="text-xs text-muted-foreground">{t('settings.notifications.quietHours.activeDesc')}</p>
              </div>
              <Switch checked={qhActive} onCheckedChange={setQhActive} />
            </div>

            {/* Options stay in place but greyed out + disabled when inactive — the
                height never changes, so toggling never shifts the page. */}
            <div
              className={`space-y-4 transition-opacity duration-200 ${
                qhActive ? '' : 'pointer-events-none select-none opacity-50'
              }`}
              aria-hidden={!qhActive}
            >
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <Label>{t('settings.notifications.quietHours.from')}</Label>
                  <TimePicker value={qhStart} onChange={setQhStart} disabled={!qhActive} ariaLabel={t('settings.notifications.quietHours.from')} />
                </div>
                <div className="space-y-1.5">
                  <Label>{t('settings.notifications.quietHours.to')}</Label>
                  <TimePicker value={qhEnd} onChange={setQhEnd} disabled={!qhActive} ariaLabel={t('settings.notifications.quietHours.to')} />
                </div>
              </div>

              <div className="space-y-1.5">
                <Label>{t('settings.notifications.quietHours.weekdays')}</Label>
                <div className="flex gap-2">
                  {DAY_LABEL_KEYS.map((key, idx) => (
                    <button
                      key={idx}
                      tabIndex={qhActive ? 0 : -1}
                      onClick={() => toggleDay(idx)}
                      className={`flex h-8 w-8 items-center justify-center rounded-full text-xs font-medium transition-colors ${
                        qhDays.includes(idx)
                          ? 'bg-primary text-primary-foreground'
                          : 'bg-secondary text-muted-foreground hover:bg-secondary/80'
                      }`}
                    >
                      {t(key)}
                    </button>
                  ))}
                </div>
              </div>

              <Button onClick={handleSaveQuietHours} size="sm" disabled={qhMutation.isPending || !qhActive}>
                <Save className="mr-1.5 h-4 w-4" />
                {t('settings.notifications.quietHours.saveButton')}
              </Button>
            </div>
          </div>
        )}
      </section>
    </div>
  )
}

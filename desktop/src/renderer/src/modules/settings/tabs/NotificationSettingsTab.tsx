import { useState, useEffect } from 'react'
import { Input } from '@/components/ui/input'
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
  Loader2,
  Trash2,
} from 'lucide-react'
import { toast } from 'sonner'
import { useSettingsStore, type NotificationModule, type NotificationPrefs } from '@/stores/settings'
import {
  useQuietHours,
  useUpdateQuietHours,
  useDNDStatus,
  useEnableDND,
  useDisableDND,
  useMutedResources,
  useUnmuteResource,
} from '@/api/hooks/useNotifications'

// ---------------------------------------------------------------------------
// Module matrix (same as before, but in a dedicated component)
// ---------------------------------------------------------------------------

const NOTIFICATION_MODULES: { key: NotificationModule; label: string; desc: string }[] = [
  { key: 'messages', label: 'Nachrichten', desc: 'Chat-Nachrichten und DMs' },
  { key: 'tasks', label: 'Aufgaben', desc: 'Zuweisung, Status-Aenderungen' },
  { key: 'meetings', label: 'Meetings', desc: 'Erinnerungen und Einladungen' },
  { key: 'mails', label: 'E-Mails', desc: 'Neue E-Mails und Antworten' },
  { key: 'calendar', label: 'Kalender', desc: 'Termine und Aenderungen' },
  { key: 'team', label: 'Team', desc: 'HR-Antraege und Mitglieder-Updates' },
  { key: 'finance', label: 'Buchhaltung', desc: 'Zahlungen und Faelligkeiten' },
]

const CHANNELS: { key: keyof NotificationPrefs; label: string }[] = [
  { key: 'email', label: 'E-Mail' },
  { key: 'push', label: 'Push' },
  { key: 'inApp', label: 'In-App' },
]

const DAY_LABELS = ['So', 'Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa']

export function NotificationSettingsTab() {
  const { notifications, updateNotification } = useSettingsStore()

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
      { onSuccess: () => toast.success('Ruhezeiten gespeichert') },
    )
  }

  // DND
  const { data: dndStatus } = useDNDStatus()
  const enableDND = useEnableDND()
  const disableDND = useDisableDND()

  const handleToggleDND = () => {
    if (dndStatus?.is_active) {
      disableDND.mutate(undefined, {
        onSuccess: () => toast.success('Bitte-nicht-stoeren deaktiviert'),
      })
    } else {
      enableDND.mutate(undefined, {
        onSuccess: () => toast.success('Bitte-nicht-stoeren aktiviert'),
      })
    }
  }

  // Muted resources
  const { data: mutedResources } = useMutedResources()
  const unmuteMutation = useUnmuteResource()

  return (
    <div className="max-w-2xl">
      <h2 className="text-foreground mb-1">Benachrichtigungen</h2>
      <p className="text-sm text-muted-foreground mb-6">
        Pro Modul einstellen, Ruhezeiten und Stummschaltungen verwalten
      </p>

      {/* ── Module × Channel Matrix ──────────────────── */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Bell className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">Benachrichtigungs-Matrix</h3>
        </div>

        <div className="rounded-lg border border-border bg-card overflow-hidden">
          {/* Header */}
          <div className="grid grid-cols-[1fr_70px_70px_70px] gap-2 px-4 py-3 text-xs font-medium text-muted-foreground border-b border-border bg-secondary/30">
            <span>Modul</span>
            {CHANNELS.map((ch) => (
              <span key={ch.key} className="text-center">{ch.label}</span>
            ))}
          </div>

          {/* Rows */}
          {NOTIFICATION_MODULES.map((mod) => (
            <div key={mod.key} className="grid grid-cols-[1fr_70px_70px_70px] gap-2 items-center px-4 py-3 border-b border-border-muted last:border-b-0">
              <div>
                <p className="text-sm text-foreground">{mod.label}</p>
                <p className="text-xs text-muted-foreground">{mod.desc}</p>
              </div>
              {CHANNELS.map((ch) => (
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

        <p className="text-xs text-muted-foreground mt-3">Aenderungen werden automatisch gespeichert.</p>
      </section>

      <Separator className="mb-8" />

      {/* ── DND (Bitte nicht stoeren) ────────────────── */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <BellOff className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">Bitte nicht stoeren</h3>
        </div>

        <div className="flex items-center justify-between rounded-lg border border-border bg-card p-4">
          <div>
            <p className="text-sm text-foreground">
              {dndStatus?.is_active ? 'Aktiv' : 'Deaktiviert'}
            </p>
            <p className="text-xs text-muted-foreground">
              {dndStatus?.is_active
                ? dndStatus.expires_at
                  ? `Bis ${new Date(dndStatus.expires_at).toLocaleTimeString('de-DE', { hour: '2-digit', minute: '2-digit' })}`
                  : 'Ohne Ablaufzeit'
                : 'Alle Benachrichtigungen sofort stummschalten'}
            </p>
          </div>
          <Button
            size="sm"
            variant={dndStatus?.is_active ? 'outline' : 'default'}
            onClick={handleToggleDND}
            disabled={enableDND.isPending || disableDND.isPending}
          >
            {dndStatus?.is_active ? 'Deaktivieren' : 'Aktivieren'}
          </Button>
        </div>
      </section>

      <Separator className="mb-8" />

      {/* ── Quiet Hours ──────────────────────────────── */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Moon className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">Ruhezeiten</h3>
        </div>

        {qhLoading ? (
          <div className="flex items-center gap-2 text-muted-foreground py-4">
            <Loader2 className="h-4 w-4 animate-spin" />
            <span className="text-sm">Ruhezeiten werden geladen...</span>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="flex items-center justify-between rounded-lg border border-border bg-card p-3">
              <div>
                <p className="text-sm text-foreground">Ruhezeiten aktiv</p>
                <p className="text-xs text-muted-foreground">Keine Benachrichtigungen in dieser Zeitspanne</p>
              </div>
              <Switch checked={qhActive} onCheckedChange={setQhActive} />
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <Label>Von</Label>
                <Input type="time" value={qhStart} onChange={(e) => setQhStart(e.target.value)} />
              </div>
              <div className="space-y-1.5">
                <Label>Bis</Label>
                <Input type="time" value={qhEnd} onChange={(e) => setQhEnd(e.target.value)} />
              </div>
            </div>

            <div className="space-y-1.5">
              <Label>Wochentage</Label>
              <div className="flex gap-2">
                {DAY_LABELS.map((label, idx) => (
                  <button
                    key={idx}
                    onClick={() => toggleDay(idx)}
                    className={`flex h-8 w-8 items-center justify-center rounded-full text-xs font-medium transition-colors ${
                      qhDays.includes(idx)
                        ? 'bg-primary text-primary-foreground'
                        : 'bg-secondary text-muted-foreground hover:bg-secondary/80'
                    }`}
                  >
                    {label}
                  </button>
                ))}
              </div>
            </div>

            <Button onClick={handleSaveQuietHours} size="sm" disabled={qhMutation.isPending}>
              <Save className="mr-1.5 h-4 w-4" />
              Ruhezeiten speichern
            </Button>
          </div>
        )}
      </section>

      <Separator className="mb-8" />

      {/* ── Muted Resources ──────────────────────────── */}
      <section>
        <div className="flex items-center gap-2 mb-4">
          <VolumeX className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">Stummgeschaltet</h3>
        </div>

        {(!mutedResources || mutedResources.length === 0) ? (
          <p className="text-xs text-muted-foreground italic">Keine Konversationen stummgeschaltet</p>
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
                    {mute.resource_type} · Seit {new Date(mute.muted_at).toLocaleDateString('de-DE')}
                  </p>
                </div>
                <button
                  onClick={() => unmuteMutation.mutate(mute.id, {
                    onSuccess: () => toast.success('Stummschaltung aufgehoben'),
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
    </div>
  )
}

/**
 * CalendarSettingsTab — Kalender-Einstellungen im Kalender-Modul.
 *
 * Zusammengeführt aus:
 *   - settings/tabs/CalendarSettingsTab.tsx (Default-View, Wochenstart, Feiertags-Region)
 *   - settings/tabs/CalDAVSettingsTab.tsx (CalDAV-Konten — persönliche App-Passwörter)
 *
 * Sektionen:
 *   - Standard-Werte (Default-View, Wochenstart, Arbeitszeiten, Erinnerung)
 *   - Feiertage (Region-Auswahl)
 *   - CalDAV-Konten (persönliche App-Passwörter für externe Clients)
 *
 * Gating: Jeder User sieht eigene Defaults. CalDAV-Konto-Status für alle.
 * Apple-Disziplin: klares Sections-Layout, kein Card-in-Card.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import {
  Save,
  Calendar,
  Globe,
  Key,
  Plus,
  Trash2,
  Copy,
} from 'lucide-react'
import { toast } from 'sonner'
import { useSettingsStore } from '@/stores/settings'
import {
  useAppPasswords,
  useCreateAppPassword,
  useRevokeAppPassword,
} from '@/api/hooks/useCaldav'
import { formatDate } from '@/lib/format'

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const HOLIDAY_REGIONS = [
  { id: 'DE-BW', label: 'Baden-Württemberg' },
  { id: 'DE-BY', label: 'Bayern' },
  { id: 'DE-BE', label: 'Berlin' },
  { id: 'DE-BB', label: 'Brandenburg' },
  { id: 'DE-HB', label: 'Bremen' },
  { id: 'DE-HH', label: 'Hamburg' },
  { id: 'DE-HE', label: 'Hessen' },
  { id: 'DE-MV', label: 'Mecklenburg-Vorpommern' },
  { id: 'DE-NI', label: 'Niedersachsen' },
  { id: 'DE-NW', label: 'Nordrhein-Westfalen' },
  { id: 'DE-RP', label: 'Rheinland-Pfalz' },
  { id: 'DE-SL', label: 'Saarland' },
  { id: 'DE-SN', label: 'Sachsen' },
  { id: 'DE-ST', label: 'Sachsen-Anhalt' },
  { id: 'DE-SH', label: 'Schleswig-Holstein' },
  { id: 'DE-TH', label: 'Thüringen' },
  { id: 'AT-W', label: 'Wien' },
  { id: 'AT-S', label: 'Salzburg' },
  { id: 'CH-ZH', label: 'Zürich' },
  { id: 'CH-BE', label: 'Bern' },
  { id: 'CH-GE', label: 'Genf' },
]

const REMINDER_KEYS = [
  { value: 0, key: 'settings.calendar.reminder.none' },
  { value: 5, key: 'settings.calendar.reminder.5min' },
  { value: 10, key: 'settings.calendar.reminder.10min' },
  { value: 15, key: 'settings.calendar.reminder.15min' },
  { value: 30, key: 'settings.calendar.reminder.30min' },
  { value: 60, key: 'settings.calendar.reminder.1hour' },
  { value: 1440, key: 'settings.calendar.reminder.1day' },
]

// ---------------------------------------------------------------------------
// Main Component
// ---------------------------------------------------------------------------

export function CalendarSettingsTab() {
  const { t } = useTranslation()
  const { calendar, updateCalendar } = useSettingsStore()

  const [defaultView, setDefaultView] = useState(calendar.defaultView)
  const [workStart, setWorkStart] = useState(calendar.workStartHour)
  const [workEnd, setWorkEnd] = useState(calendar.workEndHour)
  const [reminder, setReminder] = useState(calendar.defaultReminder)
  const [region, setRegion] = useState(calendar.holidayRegion)
  const [weekStart, setWeekStart] = useState(calendar.weekStartsOn)

  const handleSaveDefaults = () => {
    updateCalendar({ defaultView, workStartHour: workStart, workEndHour: workEnd, defaultReminder: reminder, holidayRegion: region, weekStartsOn: weekStart })
    toast.success(t('settings.calendar.saved'))
  }

  return (
    <div className="max-w-2xl space-y-0">
      <div className="mb-6">
        <h2 className="text-base font-semibold text-foreground">
          {t('kalender.einstellungen.title', { defaultValue: 'Kalender-Einstellungen' })}
        </h2>
        <p className="text-sm text-muted-foreground mt-0.5">
          {t('kalender.einstellungen.subtitle', { defaultValue: 'Standard-Werte und CalDAV-Verbindungen' })}
        </p>
      </div>

      {/* ── Standard-Werte ────────────────────────────────── */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Calendar className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">
            {t('kalender.einstellungen.defaults', { defaultValue: 'Standard-Werte' })}
          </h3>
        </div>

        <div className="space-y-5">
          {/* Default view */}
          <div className="space-y-1.5">
            <Label>{t('settings.calendar.defaultView', { defaultValue: 'Standard-Ansicht' })}</Label>
            <div className="flex gap-2">
              {(['week', 'day', 'month'] as const).map((v) => {
                const labels = {
                  week: t('settings.calendar.view.week', { defaultValue: 'Woche' }),
                  day: t('settings.calendar.view.day', { defaultValue: 'Tag' }),
                  month: t('settings.calendar.view.month', { defaultValue: 'Monat' }),
                }
                return (
                  <button
                    key={v}
                    onClick={() => setDefaultView(v)}
                    className={`flex-1 rounded-lg border py-2 text-sm transition-colors ${
                      defaultView === v
                        ? 'border-primary bg-primary/5 text-primary font-medium'
                        : 'border-border text-foreground hover:bg-secondary'
                    }`}
                  >
                    {labels[v]}
                  </button>
                )
              })}
            </div>
          </div>

          {/* Week start */}
          <div className="space-y-1.5">
            <Label>{t('settings.calendar.weekStartsOn', { defaultValue: 'Woche beginnt am' })}</Label>
            <div className="flex gap-2">
              {(['monday', 'sunday'] as const).map((d) => (
                <button
                  key={d}
                  onClick={() => setWeekStart(d)}
                  className={`flex-1 rounded-lg border py-2 text-sm transition-colors ${
                    weekStart === d
                      ? 'border-primary bg-primary/5 text-primary font-medium'
                      : 'border-border text-foreground hover:bg-secondary'
                  }`}
                >
                  {d === 'monday'
                    ? t('settings.calendar.weekDay.monday', { defaultValue: 'Montag' })
                    : t('settings.calendar.weekDay.sunday', { defaultValue: 'Sonntag' })
                  }
                </button>
              ))}
            </div>
          </div>

          {/* Work hours */}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label>{t('settings.calendar.workStart', { defaultValue: 'Arbeitsbeginn' })}</Label>
              <Select value={String(workStart)} onValueChange={(v) => setWorkStart(Number(v))}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {Array.from({ length: 12 }, (_, i) => i + 5).map((h) => (
                    <SelectItem key={h} value={String(h)}>{`${String(h).padStart(2, '0')}:00`}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>{t('settings.calendar.workEnd', { defaultValue: 'Arbeitsende' })}</Label>
              <Select value={String(workEnd)} onValueChange={(v) => setWorkEnd(Number(v))}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {Array.from({ length: 12 }, (_, i) => i + 12).map((h) => (
                    <SelectItem key={h} value={String(h)}>{`${String(h).padStart(2, '0')}:00`}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Default reminder */}
          <div className="space-y-1.5">
            <Label>{t('settings.calendar.defaultReminder', { defaultValue: 'Standard-Erinnerung' })}</Label>
            <Select value={String(reminder)} onValueChange={(v) => setReminder(Number(v))}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                {REMINDER_KEYS.map((r) => (
                  <SelectItem key={r.value} value={String(r.value)}>{t(r.key)}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>

        <Button onClick={handleSaveDefaults} className="mt-5" size="sm">
          <Save className="mr-1.5 h-4 w-4" />
          {t('settings.calendar.saveButton', { defaultValue: 'Einstellungen speichern' })}
        </Button>
      </section>

      <Separator className="mb-8" />

      {/* ── Feiertage ────────────────────────────────────── */}
      <section className="mb-8">
        <div className="flex items-center gap-2 mb-4">
          <Globe className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">
            {t('kalender.einstellungen.feiertage', { defaultValue: 'Feiertage' })}
          </h3>
        </div>

        <div className="space-y-1.5">
          <Label>{t('settings.calendar.holidayRegion', { defaultValue: 'Feiertags-Region' })}</Label>
          <Select value={region} onValueChange={setRegion}>
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent>
              {HOLIDAY_REGIONS.map((r) => (
                <SelectItem key={r.id} value={r.id}>{r.label}</SelectItem>
              ))}
            </SelectContent>
          </Select>
          <p className="text-[10px] text-muted-foreground">
            {t('settings.calendar.holidayRegionHint', { defaultValue: 'Legt fest welche gesetzlichen Feiertage im Kalender angezeigt werden.' })}
          </p>
        </div>

        <Button
          className="mt-4"
          size="sm"
          onClick={() => {
            updateCalendar({ holidayRegion: region })
            toast.success(t('kalender.einstellungen.feiertageGespeichert', { defaultValue: 'Feiertags-Region gespeichert' }))
          }}
        >
          <Save className="mr-1.5 h-4 w-4" />
          {t('kalender.einstellungen.saveFeiertage', { defaultValue: 'Region speichern' })}
        </Button>
      </section>

      <Separator className="mb-8" />

      {/* ── CalDAV-Konten ─────────────────────────────────── */}
      <section>
        <div className="flex items-center gap-2 mb-4">
          <Key className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium text-foreground">
            {t('kalender.einstellungen.caldav', { defaultValue: 'CalDAV-Konten' })}
          </h3>
        </div>
        <p className="text-xs text-muted-foreground mb-4">
          {t('kalender.einstellungen.caldavDesc', { defaultValue: 'App-spezifische Passwörter für Apple Kalender, Thunderbird oder andere CalDAV-Clients.' })}
        </p>
        <CalDAVPasswordsSection />
      </section>
    </div>
  )
}

// ---------------------------------------------------------------------------
// CalDAV Passwords sub-section
// ---------------------------------------------------------------------------

function CalDAVPasswordsSection() {
  const { t } = useTranslation()
  const { data: passwords, isLoading } = useAppPasswords()
  const createPassword = useCreateAppPassword()
  const revokePassword = useRevokeAppPassword()

  const [newLabel, setNewLabel] = useState('')
  const [showNewPassword, setShowNewPassword] = useState<string | null>(null)

  const handleCreate = async () => {
    if (!newLabel.trim()) {
      toast.error(t('settings.caldav.labelRequired', { defaultValue: 'Bitte Bezeichnung eingeben' }))
      return
    }
    const result = await createPassword.mutateAsync(newLabel.trim())
    setShowNewPassword(result.password)
    setNewLabel('')
    toast.success(t('settings.caldav.passwordCreated', { defaultValue: 'App-Passwort erstellt' }))
  }

  if (isLoading) {
    return <div className="h-12 animate-pulse rounded bg-secondary/50" />
  }

  return (
    <div className="space-y-4">
      {/* Password list */}
      {passwords && passwords.length > 0 ? (
        <div className="divide-y divide-border rounded-lg border border-border overflow-hidden">
          {passwords.map((pw) => (
            <div key={pw.id} className="flex items-center gap-3 px-4 py-3">
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-foreground truncate">{pw.label}</p>
                <p className="text-xs text-muted-foreground tabular-nums">
                  {pw.password_prefix}… · {t('settings.caldav.created', { defaultValue: 'Erstellt' })}: {formatDate(pw.created_at)}
                  {pw.last_used_at && ` · ${t('settings.caldav.lastUsed', { defaultValue: 'Zuletzt' })}: ${formatDate(pw.last_used_at)}`}
                </p>
              </div>
              {pw.revoked_at ? (
                <span className="text-xs text-error font-medium">
                  {t('settings.caldav.revoked', { defaultValue: 'Widerrufen' })}
                </span>
              ) : (
                <button
                  onClick={() => revokePassword.mutate(pw.id)}
                  className="rounded p-1.5 text-muted-foreground hover:text-destructive transition-colors"
                  aria-label={t('common.delete', { defaultValue: 'Löschen' })}
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              )}
            </div>
          ))}
        </div>
      ) : (
        <p className="text-sm text-muted-foreground">
          {t('settings.caldav.noPasswords', { defaultValue: 'Noch keine App-Passwörter erstellt.' })}
        </p>
      )}

      {/* New password — one-time display */}
      {showNewPassword && (
        <div className="rounded-lg border border-warning/30 bg-warning-light p-4">
          <p className="text-sm font-medium text-warning mb-2">
            {t('settings.caldav.passwordOnce', { defaultValue: 'Passwort nur jetzt sichtbar — bitte sofort kopieren.' })}
          </p>
          <div className="flex items-center gap-2">
            <code className="flex-1 rounded border border-border bg-card px-3 py-2 font-mono text-sm select-all break-all">
              {showNewPassword}
            </code>
            <button
              onClick={() => { navigator.clipboard.writeText(showNewPassword); toast.success(t('settings.caldav.copied', { label: 'Passwort', defaultValue: 'Kopiert' })) }}
              className="rounded-md border border-border p-2 text-muted-foreground hover:text-foreground transition-colors"
              aria-label={t('common.copy', { defaultValue: 'Kopieren' })}
            >
              <Copy className="h-4 w-4" />
            </button>
          </div>
          <button
            onClick={() => setShowNewPassword(null)}
            className="mt-2 text-xs text-muted-foreground hover:text-foreground"
          >
            {t('common.close', { defaultValue: 'Schließen' })}
          </button>
        </div>
      )}

      {/* Create new */}
      <div className="flex items-end gap-2">
        <div className="flex-1 space-y-1.5">
          <Label htmlFor="caldav-label-kal">
            {t('settings.caldav.labelField', { defaultValue: 'Bezeichnung' })}
          </Label>
          <Input
            id="caldav-label-kal"
            placeholder={t('settings.caldav.labelPlaceholder', { defaultValue: 'z. B. macOS Kalender' })}
            value={newLabel}
            onChange={(e) => setNewLabel(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleCreate()}
          />
        </div>
        <Button
          onClick={handleCreate}
          disabled={createPassword.isPending || !newLabel.trim()}
          size="sm"
        >
          <Plus className="h-3.5 w-3.5 mr-1" />
          {t('settings.caldav.createPassword', { defaultValue: 'Passwort erstellen' })}
        </Button>
      </div>
    </div>
  )
}

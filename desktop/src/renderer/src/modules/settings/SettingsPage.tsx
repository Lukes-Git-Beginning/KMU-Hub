import { useState } from 'react'
import {
  User,
  Globe,
  Palette,
  Shield,
  Bell,
  Info,
  Key,
  Monitor,
  Smartphone,
  Check,
  Mail,
  Calendar,
  Receipt,
  Lock,
  Copy,
  Eye,
  EyeOff,
  Landmark,
  Plug,
  Sparkles,
} from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { toast } from 'sonner'
import { ConfirmDialog } from '@/components/shared'
import { useSettingsStore } from '@/stores/settings'
import { useAuthStore } from '@/stores/auth'
import { useUIStore } from '@/stores/ui'
import { DESK_BACKGROUNDS } from '@/components/layout/DeskEnvironment'
import { canSeeSettingsTab } from '@/config/roles'
import { MailSettingsTab } from './tabs/MailSettingsTab'
import { CalendarSettingsTab } from './tabs/CalendarSettingsTab'
import { FinanceSettingsTab } from './tabs/FinanceSettingsTab'
import { PrivacySettingsTab } from './tabs/PrivacySettingsTab'
import { PaletteSwitcher } from '@/components/shared/PaletteSwitcher'
import { LayoutSwitcher } from '@/components/shared/LayoutSwitcher'
import { headerWidgetList } from '@/components/header/header-widgets'
import { CompanySettingsTab } from './tabs/CompanySettingsTab'
import { NotificationSettingsTab } from './tabs/NotificationSettingsTab'
import { IntegrationSettingsTab } from './tabs/IntegrationSettingsTab'
import { AIGovernanceTab } from './tabs/AIGovernanceTab'
import { ITAdminTab } from './tabs/ITAdminTab'
import { ThemePreview } from './ThemePreview'
import { useTourStore } from '@/stores/tour'

type TabKey = 'profile' | 'appearance' | 'language' | 'security' | 'notifications' | 'mail' | 'calendar' | 'finance' | 'company' | 'it-admin' | 'integrations' | 'privacy' | 'ai' | 'about'

interface TabConfig {
  key: TabKey
  label: string
  icon: typeof User
  group?: string
}

const ALL_TABS: TabConfig[] = [
  { key: 'profile', label: 'Profil', icon: User, group: 'Persönlich' },
  { key: 'appearance', label: 'Darstellung', icon: Palette, group: 'Persönlich' },
  { key: 'language', label: 'Sprache & Region', icon: Globe, group: 'Persönlich' },
  { key: 'security', label: 'Sicherheit', icon: Shield, group: 'Persönlich' },
  { key: 'notifications', label: 'Benachrichtigungen', icon: Bell, group: 'Persönlich' },
  { key: 'mail', label: 'E-Mail', icon: Mail, group: 'Module' },
  { key: 'calendar', label: 'Kalender', icon: Calendar, group: 'Module' },
  { key: 'finance', label: 'Buchhaltung', icon: Receipt, group: 'Module' },
  { key: 'company', label: 'Firma', icon: Landmark, group: 'Admin' },
  { key: 'it-admin', label: 'IT-Admin', icon: Monitor, group: 'Admin' },
  { key: 'integrations', label: 'Integrationen', icon: Plug, group: 'Admin' },
  { key: 'privacy', label: 'Datenschutz', icon: Lock, group: 'Admin' },
  { key: 'ai', label: 'KI-Assistent', icon: Sparkles, group: 'Admin' },
  { key: 'about', label: 'Über KMU Hub', icon: Info, group: 'Sonstiges' },
]

const TAB_GROUPS = ['Persönlich', 'Module', 'Admin', 'Sonstiges']

export default function SettingsPage() {
  const [activeTab, setActiveTab] = useState<TabKey>('profile')
  const user = useAuthStore((s) => s.user)

  // Filter tabs by role — restricted tabs are INVISIBLE, not greyed out
  const tabs = ALL_TABS.filter((tab) => canSeeSettingsTab(user, tab.key))

  // If the active tab got hidden (e.g. after role switch), fall back to profile
  const isActiveVisible = tabs.some((t) => t.key === activeTab)
  const effectiveTab = isActiveVisible ? activeTab : 'profile'

  return (
    <div className="flex h-full overflow-hidden animate-fade-up">
      {/* Settings sidebar */}
      <aside className="w-56 shrink-0 border-r border-border bg-card p-4 overflow-y-auto">
        <h3 className="text-sm font-medium text-foreground mb-4 px-2">Einstellungen</h3>
        <nav className="space-y-4">
          {TAB_GROUPS.map((group) => {
            const groupTabs = tabs.filter((t) => t.group === group)
            if (groupTabs.length === 0) return null
            return (
              <div key={group}>
                <p className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground px-3 mb-1">{group}</p>
                <div className="space-y-0.5">
                  {groupTabs.map((tab) => {
                    const Icon = tab.icon
                    return (
                      <button
                        key={tab.key}
                        onClick={() => setActiveTab(tab.key)}
                        className={`flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-sm transition-colors ${
                          effectiveTab === tab.key
                            ? 'bg-primary-light text-primary font-medium'
                            : 'text-foreground hover:bg-secondary'
                        }`}
                      >
                        <Icon className="h-4 w-4 shrink-0" />
                        {tab.label}
                      </button>
                    )
                  })}
                </div>
              </div>
            )
          })}
        </nav>
      </aside>

      {/* Content area */}
      <div className="flex-1 overflow-y-auto p-6">
        {effectiveTab === 'profile' && <ProfileTab />}
        {effectiveTab === 'appearance' && <AppearanceTab />}
        {effectiveTab === 'language' && <LanguageTab />}
        {effectiveTab === 'security' && <SecurityTab />}
        {effectiveTab === 'notifications' && <NotificationSettingsTab />}
        {effectiveTab === 'mail' && <MailSettingsTab />}
        {effectiveTab === 'calendar' && <CalendarSettingsTab />}
        {effectiveTab === 'finance' && <FinanceSettingsTab />}
        {effectiveTab === 'company' && <CompanySettingsTab />}
        {effectiveTab === 'it-admin' && <ITAdminTab />}
        {effectiveTab === 'integrations' && <IntegrationSettingsTab />}
        {effectiveTab === 'privacy' && <PrivacySettingsTab />}
        {effectiveTab === 'ai' && <AIGovernanceTab />}
        {effectiveTab === 'about' && <AboutTab />}
      </div>
    </div>
  )
}

// ============================================================
// Profile Tab — now wired to settings store
// ============================================================
function ProfileTab() {
  const { profile, updateProfile } = useSettingsStore()
  const [firstName, setFirstName] = useState(profile.firstName)
  const [lastName, setLastName] = useState(profile.lastName)
  const [email, setEmail] = useState(profile.email)
  const [phone, setPhone] = useState(profile.phone)
  const [position, setPosition] = useState(profile.position)
  const [bio, setBio] = useState(profile.bio)

  const handleSave = () => {
    updateProfile({ firstName, lastName, email, phone, position, bio })
    toast.success('Profil gespeichert')
  }

  const handlePhotoChange = () => {
    toast.success('Foto-Upload wird simuliert...')
    setTimeout(() => {
      updateProfile({ avatarUrl: 'mock-avatar.jpg' })
      toast.success('Profilbild aktualisiert')
    }, 1000)
  }

  const initials = `${firstName.charAt(0)}${lastName.charAt(0)}`.toUpperCase()

  return (
    <div className="max-w-2xl mx-auto">
      <h2 className="text-foreground mb-1">Profil</h2>
      <p className="text-sm text-muted-foreground mb-6">Verwalte deine persönlichen Informationen</p>

      {/* Avatar */}
      <div className="flex items-center gap-4 mb-8">
        <div className="flex h-20 w-20 items-center justify-center rounded-full bg-primary-light text-2xl font-medium text-primary">
          {initials}
        </div>
        <div>
          <Button variant="outline" size="sm" onClick={handlePhotoChange}>
            Foto ändern
          </Button>
          <p className="text-xs text-muted-foreground mt-1">JPG, PNG oder GIF, max. 5 MB</p>
        </div>
      </div>

      {/* Form fields */}
      <div className="space-y-4">
        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-1.5">
            <label className="block text-sm font-medium text-foreground">Vorname</label>
            <Input value={firstName} onChange={(e) => setFirstName(e.target.value)} />
          </div>
          <div className="space-y-1.5">
            <label className="block text-sm font-medium text-foreground">Nachname</label>
            <Input value={lastName} onChange={(e) => setLastName(e.target.value)} />
          </div>
        </div>
        <div className="space-y-1.5">
          <label className="block text-sm font-medium text-foreground">E-Mail</label>
          <Input type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
        </div>
        <div className="space-y-1.5">
          <label className="block text-sm font-medium text-foreground">Telefon</label>
          <Input type="tel" value={phone} onChange={(e) => setPhone(e.target.value)} />
        </div>
        <div className="space-y-1.5">
          <label className="block text-sm font-medium text-foreground">Position</label>
          <Input value={position} onChange={(e) => setPosition(e.target.value)} />
        </div>
        <div className="space-y-1.5">
          <label className="block text-sm font-medium text-foreground">Bio</label>
          <textarea
            value={bio}
            onChange={(e) => setBio(e.target.value)}
            rows={3}
            className="w-full rounded-lg border border-border bg-input-background px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring resize-none"
          />
        </div>
      </div>

      <div className="mt-6 flex gap-2">
        <Button onClick={handleSave}>Speichern</Button>
        <Button
          variant="outline"
          onClick={() => {
            setFirstName(profile.firstName)
            setLastName(profile.lastName)
            setEmail(profile.email)
            setPhone(profile.phone)
            setPosition(profile.position)
            setBio(profile.bio)
          }}
        >
          Abbrechen
        </Button>
      </div>
    </div>
  )
}

// ============================================================
// Appearance Tab — wired to store
// ============================================================
function AppearanceTab() {
  const { appearance, updateAppearance } = useSettingsStore()

  const theme = useUIStore((s) => s.theme)
  const setTheme = useUIStore((s) => s.setTheme)
  const accentIntensity = useUIStore((s) => s.accentIntensity)
  const setAccentIntensity = useUIStore((s) => s.setAccentIntensity)
  const windowStyle = useUIStore((s) => s.windowStyle)
  const setWindowStyle = useUIStore((s) => s.setWindowStyle)
  const uiLook = useUIStore((s) => s.uiLook)
  const setUILook = useUIStore((s) => s.setUILook)
  const deskBackground = useUIStore((s) => s.deskBackground)
  const setDeskBackground = useUIStore((s) => s.setDeskBackground)

  const themes = [
    { id: 'light' as const, label: 'Hell', desc: 'Warme, helle Oberfläche' },
    { id: 'dark' as const, label: 'Dunkel', desc: 'Augenfreundlich bei wenig Licht' },
    { id: 'auto' as const, label: 'System', desc: 'Folgt den Systemeinstellungen' },
  ]

  const handleThemeChange = (t: 'light' | 'dark' | 'auto') => {
    setTheme(t)
    updateAppearance({ theme: t })
  }

  return (
    <div className="max-w-2xl mx-auto">
      <h2 className="text-foreground mb-1">Darstellung</h2>
      <p className="text-sm text-muted-foreground mb-6">Passe das Erscheinungsbild der App an</p>

      <ThemePreview className="mb-8" />

      <h3 className="text-sm font-medium text-foreground mb-3">Farbschema</h3>
      <div className="grid grid-cols-3 gap-3 mb-8">
        {themes.map((t) => (
          <button
            key={t.id}
            onClick={() => handleThemeChange(t.id)}
            className={`relative rounded-lg border p-4 text-center transition-colors ${
              theme === t.id
                ? 'border-primary bg-primary-light'
                : 'border-border bg-card hover:bg-secondary'
            }`}
          >
            {theme === t.id && (
              <span className="absolute top-2 right-2">
                <Check className="h-4 w-4 text-primary" />
              </span>
            )}
            <div className={`mx-auto mb-2 h-8 w-8 rounded-full border ${
              t.id === 'light' ? 'bg-amber-100 border-amber-300' :
              t.id === 'dark' ? 'bg-slate-700 border-slate-500' :
              'bg-gradient-to-br from-amber-100 to-slate-700 border-gray-400'
            }`} />
            <p className="text-sm font-medium text-foreground">{t.label}</p>
            <p className="text-xs text-muted-foreground mt-0.5">{t.desc}</p>
          </button>
        ))}
      </div>

      {/* ── COLOR PALETTE PICKER ────────────────────── */}
      <h3 className="text-sm font-medium text-foreground mb-3">Farbpalette</h3>
      <p className="text-xs text-muted-foreground mb-3">Waehle die Akzentfarben fuer die gesamte App</p>
      <PaletteSwitcher className="mb-8" />

      {/* ── ACCENT INTENSITY ─────────────────────────── */}
      <h3 className="text-sm font-medium text-foreground mb-3">Akzentfarben-Intensitaet</h3>
      <p className="text-xs text-muted-foreground mb-3">Bestimme wie stark die Akzentfarben in der App eingesetzt werden</p>
      <div className="grid grid-cols-2 gap-3 mb-8">
        {([
          { id: 'subtle' as const, label: 'Dezent', desc: 'Akzente nur bei KPIs, Status-Badges und wichtigen CTAs' },
          { id: 'vivid' as const, label: 'Lebendig', desc: 'Akzente auch bei Tabs, Kategorien, Charts und Tags' },
        ]).map((opt) => (
          <button
            key={opt.id}
            onClick={() => setAccentIntensity(opt.id)}
            className={`relative rounded-lg border p-4 text-left transition-colors ${
              accentIntensity === opt.id
                ? 'border-primary bg-primary-light'
                : 'border-border bg-card hover:bg-secondary'
            }`}
          >
            {accentIntensity === opt.id && (
              <span className="absolute top-2 right-2">
                <Check className="h-4 w-4 text-primary" />
              </span>
            )}
            <div className={`mb-2 flex gap-1 ${opt.id === 'vivid' ? 'opacity-100' : 'opacity-60'}`}>
              <span className="h-3 w-3 rounded-full" style={{ background: 'var(--accent-1)' }} />
              <span className="h-3 w-3 rounded-full" style={{ background: 'var(--accent-2)' }} />
              <span className="h-3 w-3 rounded-full" style={{ background: 'var(--primary)' }} />
            </div>
            <p className="text-sm font-medium text-foreground">{opt.label}</p>
            <p className="text-xs text-muted-foreground mt-0.5">{opt.desc}</p>
          </button>
        ))}
      </div>

      {/* ── NAVIGATION LAYOUT ────────────────────────── */}
      <h3 className="text-sm font-medium text-foreground mb-3">Navigation</h3>
      <p className="text-xs text-muted-foreground mb-3">Waehle wie du durch die App navigierst</p>
      <LayoutSwitcher className="mb-8" />

      {/* ── FENSTER-STIL ──────────────────────────── */}
      <h3 className="text-sm font-medium text-foreground mb-3">Fenster-Stil</h3>
      <p className="text-xs text-muted-foreground mb-3">Vollbild oder abgerundetes Fenster mit Rand</p>
      <div className="grid grid-cols-2 gap-3 mb-8">
        {([
          { id: 'full' as const, label: 'Vollbild', desc: 'Fenster komplett ausgefuellt' },
          { id: 'bubble' as const, label: 'Bubble', desc: 'Abgerundeter Rahmen mit Rand' },
        ]).map((style) => (
          <button
            key={style.id}
            onClick={() => setWindowStyle(style.id)}
            className={`relative rounded-lg border p-4 text-center transition-colors ${
              windowStyle === style.id
                ? 'border-primary bg-primary-light'
                : 'border-border bg-card hover:bg-secondary'
            }`}
          >
            {windowStyle === style.id && (
              <span className="absolute top-2 right-2">
                <Check className="h-4 w-4 text-primary" />
              </span>
            )}
            <div className="mx-auto mb-2 h-12 w-20 rounded-md bg-muted/60 flex items-center justify-center p-0 overflow-hidden">
              {style.id === 'full' ? (
                /* Vollbild: fills the entire frame, sharp corners */
                <div className="h-full w-full bg-card flex flex-col">
                  <div className="h-1.5 bg-secondary border-b border-border" />
                  <div className="flex flex-1">
                    <div className="w-4 bg-secondary/60 border-r border-border" />
                    <div className="flex-1" />
                  </div>
                </div>
              ) : (
                /* Bubble: smaller window with rounded corners + visible gap */
                <div className="h-8 w-14 rounded-lg bg-card shadow-md flex flex-col overflow-hidden">
                  <div className="h-1.5 bg-secondary border-b border-border" />
                  <div className="flex flex-1">
                    <div className="w-3 bg-secondary/60 border-r border-border" />
                    <div className="flex-1" />
                  </div>
                </div>
              )}
            </div>
            <p className="text-sm font-medium text-foreground">{style.label}</p>
            <p className="text-xs text-muted-foreground mt-0.5">{style.desc}</p>
          </button>
        ))}
      </div>

      {/* ── OBERFLAECHE (SOLID / MILCHGLAS) ─────────── */}
      <h3 className="text-sm font-medium text-foreground mb-3">Oberflaeche</h3>
      <p className="text-xs text-muted-foreground mb-3">Standard oder Milchglas-Effekt fuer die gesamte App</p>
      <div className="grid grid-cols-2 gap-3 mb-8">
        {([
          { id: 'solid' as const, label: 'Standard', desc: 'Solide, deckende Oberflaechen' },
          { id: 'glass' as const, label: 'Milchglas', desc: 'Frosted Glass mit Hintergrund-Durchschein' },
        ]).map((opt) => (
          <button
            key={opt.id}
            onClick={() => setUILook(opt.id)}
            className={`relative rounded-lg border p-4 text-center transition-colors ${
              uiLook === opt.id
                ? 'border-primary bg-primary-light'
                : 'border-border bg-card hover:bg-secondary'
            }`}
          >
            {uiLook === opt.id && (
              <span className="absolute top-2 right-2">
                <Check className="h-4 w-4 text-primary" />
              </span>
            )}
            <div className="mx-auto mb-2 h-12 w-20 rounded-md overflow-hidden">
              {opt.id === 'solid' ? (
                /* Standard: Solid opaque card */
                <div className="h-full w-full bg-card border border-border flex flex-col">
                  <div className="h-2 bg-secondary border-b border-border" />
                  <div className="flex-1 p-1 space-y-0.5">
                    <div className="h-1 w-10 rounded-full bg-foreground/20" />
                    <div className="h-1 w-7 rounded-full bg-foreground/10" />
                  </div>
                </div>
              ) : (
                /* Milchglas: Frosted, translucent with gradient behind */
                <div className="h-full w-full bg-gradient-to-br from-primary/30 via-accent-1/20 to-primary/10 relative">
                  <div className="absolute inset-0 bg-white/50 backdrop-blur-[2px] flex flex-col">
                    <div className="h-2 bg-white/40 border-b border-white/20" />
                    <div className="flex-1 p-1 space-y-0.5">
                      <div className="h-1 w-10 rounded-full bg-foreground/15" />
                      <div className="h-1 w-7 rounded-full bg-foreground/10" />
                    </div>
                  </div>
                </div>
              )}
            </div>
            <p className="text-sm font-medium text-foreground">{opt.label}</p>
            <p className="text-xs text-muted-foreground mt-0.5">{opt.desc}</p>
          </button>
        ))}
      </div>

      {/* ── HINTERGRUND (nur bei Milchglas) ──────────── */}
      {uiLook === 'glass' && (
        <>
          <h3 className="text-sm font-medium text-foreground mb-3">Hintergrund</h3>
          <p className="text-xs text-muted-foreground mb-3">Waehle einen Hintergrund der durch die Milchglas-Oberflaeche scheint</p>
          <div className="grid grid-cols-4 gap-2 mb-8">
            <button
              onClick={() => setDeskBackground(null)}
              className={`relative h-16 rounded-lg border-2 transition-colors flex items-center justify-center ${
                !deskBackground
                  ? 'border-primary'
                  : 'border-border hover:border-primary/40'
              }`}
            >
              <span className="text-xs text-muted-foreground">Keiner</span>
              {!deskBackground && (
                <span className="absolute top-1 right-1">
                  <Check className="h-3 w-3 text-primary" />
                </span>
              )}
            </button>
            {Object.entries(DESK_BACKGROUNDS).map(([id, bg]) => (
              <button
                key={id}
                onClick={() => setDeskBackground(id)}
                className={`relative h-16 rounded-lg border-2 transition-colors overflow-hidden ${
                  deskBackground === id
                    ? 'border-primary'
                    : 'border-border hover:border-primary/40'
                }`}
                title={bg.label}
              >
                <div className="absolute inset-0" style={{ background: bg.css }} />
                <span className="relative z-10 text-[10px] font-medium text-white drop-shadow-md">{bg.label}</span>
                {deskBackground === id && (
                  <span className="absolute top-1 right-1 z-10">
                    <Check className="h-3 w-3 text-white drop-shadow-md" />
                  </span>
                )}
              </button>
            ))}
          </div>
        </>
      )}

      {/* ── HEADER WIDGETS ─────────────────────────── */}
      <h3 className="text-sm font-medium text-foreground mb-3">Header-Widgets</h3>
      <p className="text-xs text-muted-foreground mb-3">Waehle bis zu 3 Mini-Widgets fuer die Kopfleiste</p>
      <HeaderWidgetPicker className="mb-8" />

      <div className="mt-4 mb-4 border-t border-border" />

      <h3 className="text-sm font-medium text-foreground mb-3">Schriftgröße</h3>
      <div className="flex items-center gap-3 mb-2">
        <span className="text-xs text-muted-foreground">Klein</span>
        <input
          type="range"
          min={12}
          max={20}
          value={appearance.fontSize}
          onChange={(e) => updateAppearance({ fontSize: Number(e.target.value) })}
          className="flex-1 accent-[var(--primary)]"
        />
        <span className="text-xs text-muted-foreground">Gross</span>
      </div>
      <p className="text-xs text-muted-foreground mb-8">{appearance.fontSize}px</p>
    </div>
  )
}

// ============================================================
// Language Tab — wired to store
// ============================================================
function LanguageTab() {
  const { language, updateLanguage } = useSettingsStore()
  const [locale, setLocale] = useState(language.locale)
  const [timezone, setTimezone] = useState(language.timezone)
  const [dateFormat, setDateFormat] = useState(language.dateFormat)

  const languages = [
    { id: 'de' as const, label: 'Deutsch', flag: 'DE' },
    { id: 'en' as const, label: 'English', flag: 'EN' },
    { id: 'fr' as const, label: 'Francais', flag: 'FR' },
  ]

  const handleSave = () => {
    updateLanguage({ locale, timezone, dateFormat })
    toast.success('Sprache & Region gespeichert')
  }

  return (
    <div className="max-w-2xl mx-auto">
      <h2 className="text-foreground mb-1">Sprache & Region</h2>
      <p className="text-sm text-muted-foreground mb-6">Sprache, Zeitzone und Datumsformat einstellen</p>

      <h3 className="text-sm font-medium text-foreground mb-3">Sprache</h3>
      <div className="space-y-2 mb-8">
        {languages.map((lang) => (
          <button
            key={lang.id}
            onClick={() => setLocale(lang.id)}
            className={`flex w-full items-center gap-3 rounded-lg border p-3 transition-colors ${
              locale === lang.id ? 'border-primary bg-primary-light' : 'border-border bg-card hover:bg-secondary'
            }`}
          >
            <span className="flex h-8 w-8 items-center justify-center rounded-full bg-secondary text-xs font-medium text-foreground">{lang.flag}</span>
            <span className="text-sm text-foreground">{lang.label}</span>
            {locale === lang.id && <Check className="ml-auto h-4 w-4 text-primary" />}
          </button>
        ))}
      </div>

      <div className="grid grid-cols-2 gap-4 mb-6">
        <div>
          <label className="block text-sm font-medium text-foreground mb-1.5">Zeitzone</label>
          <select
            value={timezone}
            onChange={(e) => setTimezone(e.target.value)}
            className="w-full rounded-lg border border-border bg-input-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
          >
            <option value="Europe/Zurich">Europe/Zurich (GMT+1)</option>
            <option value="Europe/Berlin">Europe/Berlin (GMT+1)</option>
            <option value="Europe/Vienna">Europe/Vienna (GMT+1)</option>
            <option value="Europe/Paris">Europe/Paris (GMT+1)</option>
            <option value="Europe/London">Europe/London (GMT+0)</option>
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium text-foreground mb-1.5">Datumsformat</label>
          <select
            value={dateFormat}
            onChange={(e) => setDateFormat(e.target.value)}
            className="w-full rounded-lg border border-border bg-input-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
          >
            <option value="DD.MM.YYYY">DD.MM.YYYY</option>
            <option value="MM/DD/YYYY">MM/DD/YYYY</option>
            <option value="YYYY-MM-DD">YYYY-MM-DD</option>
          </select>
        </div>
      </div>

      <Button onClick={handleSave}>Speichern</Button>
    </div>
  )
}

// ============================================================
// Security Tab — 2FA setup, sessions, password
// ============================================================
function SecurityTab() {
  const { security, enable2FA, disable2FA, updateSecurity } = useSettingsStore()
  const [showCurrent, setShowCurrent] = useState(false)
  const [showNew, setShowNew] = useState(false)
  const [currentPw, setCurrentPw] = useState('')
  const [newPw, setNewPw] = useState('')
  const [confirmPw, setConfirmPw] = useState('')
  const [show2FASetup, setShow2FASetup] = useState(false)
  const [showDisable2FA, setShowDisable2FA] = useState(false)
  const [showRevokeSession, setShowRevokeSession] = useState<number | null>(null)

  // Password expiry calculation
  const lastChanged = security.passwordLastChanged ? new Date(security.passwordLastChanged) : null
  const expiryDays = security.passwordExpiryDays || 90
  // eslint-disable-next-line react-hooks/purity -- Date.now() needed for expiry calculation
  const daysSinceChange = lastChanged ? Math.floor((Date.now() - lastChanged.getTime()) / (1000 * 60 * 60 * 24)) : 0
  const daysUntilExpiry = Math.max(0, expiryDays - daysSinceChange)
  const isExpiringSoon = daysUntilExpiry <= 14 && daysUntilExpiry > 0
  const isExpired = daysUntilExpiry === 0

  const mockSessions = [
    { id: 1, device: 'Desktop — Windows', location: 'Zürich, CH', lastActive: 'Jetzt aktiv', isCurrent: true, icon: Monitor },
    { id: 2, device: 'iPhone 15 Pro', location: 'Zürich, CH', lastActive: 'Vor 2 Stunden', isCurrent: false, icon: Smartphone },
    { id: 3, device: 'MacBook Pro', location: 'Bern, CH', lastActive: 'Vor 3 Tagen', isCurrent: false, icon: Monitor },
  ]

  const handlePasswordChange = () => {
    if (!currentPw || !newPw) return
    if (newPw !== confirmPw) {
      toast.error('Passwörter stimmen nicht überein')
      return
    }
    if (newPw.length < 8) {
      toast.error('Passwort muss mindestens 8 Zeichen haben')
      return
    }
    updateSecurity({ passwordLastChanged: new Date().toISOString() })
    toast.success('Passwort geändert')
    setCurrentPw('')
    setNewPw('')
    setConfirmPw('')
  }

  const handle2FAEnable = () => {
    enable2FA()
    setShow2FASetup(false)
    toast.success('Zwei-Faktor-Authentifizierung aktiviert')
  }

  const handle2FADisable = () => {
    disable2FA()
    setShowDisable2FA(false)
    toast.success('Zwei-Faktor-Authentifizierung deaktiviert')
  }

  const handleRevokeSession = () => {
    toast.success('Sitzung beendet')
    setShowRevokeSession(null)
  }

  return (
    <div className="max-w-2xl mx-auto">
      <h2 className="text-foreground mb-1">Sicherheit</h2>
      <p className="text-sm text-muted-foreground mb-6">Passwort, 2FA und Sitzungen verwalten</p>

      {/* Password Expiry Info */}
      {expiryDays > 0 && (
        <section className="mb-6">
          <div className={`flex items-center gap-3 rounded-lg border p-4 ${
            isExpired ? 'border-error/50 bg-error/5' :
            isExpiringSoon ? 'border-warning/50 bg-warning/5' :
            'border-border bg-card'
          }`}>
            <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${
              isExpired ? 'bg-error/10' : isExpiringSoon ? 'bg-warning/10' : 'bg-secondary'
            }`}>
              <Shield className={`h-5 w-5 ${
                isExpired ? 'text-error' : isExpiringSoon ? 'text-warning' : 'text-muted-foreground'
              }`} />
            </div>
            <div className="flex-1">
              <p className="text-sm font-medium text-foreground">
                {isExpired ? 'Passwort abgelaufen' :
                 isExpiringSoon ? `Passwort laeuft in ${daysUntilExpiry} Tagen ab` :
                 `Naechste Aenderung in ${daysUntilExpiry} Tagen`}
              </p>
              <p className="text-xs text-muted-foreground">
                Zuletzt geaendert: {lastChanged?.toLocaleDateString('de-DE') ?? 'Unbekannt'}
                {' · '}Richtlinie: alle {expiryDays} Tage
              </p>
            </div>
            {(isExpired || isExpiringSoon) && (
              <span className={`rounded-full px-2.5 py-1 text-[10px] font-medium ${
                isExpired ? 'bg-error/15 text-error' : 'bg-warning/15 text-warning'
              }`}>
                {isExpired ? 'Sofort aendern' : 'Bald faellig'}
              </span>
            )}
          </div>
        </section>
      )}

      {/* Password */}
      <section className="mb-8">
        <h3 className="text-sm font-medium text-foreground mb-3">Passwort ändern</h3>
        <div className="space-y-3">
          <div className="space-y-1.5">
            <label className="block text-sm font-medium text-foreground">Aktuelles Passwort</label>
            <div className="relative">
              <Input
                type={showCurrent ? 'text' : 'password'}
                value={currentPw}
                onChange={(e) => setCurrentPw(e.target.value)}
              />
              <button
                onClick={() => setShowCurrent(!showCurrent)}
                className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              >
                {showCurrent ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
          </div>
          <div className="space-y-1.5">
            <label className="block text-sm font-medium text-foreground">Neues Passwort</label>
            <div className="relative">
              <Input
                type={showNew ? 'text' : 'password'}
                value={newPw}
                onChange={(e) => setNewPw(e.target.value)}
              />
              <button
                onClick={() => setShowNew(!showNew)}
                className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              >
                {showNew ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
          </div>
          <div className="space-y-1.5">
            <label className="block text-sm font-medium text-foreground">Passwort bestätigen</label>
            <Input type="password" value={confirmPw} onChange={(e) => setConfirmPw(e.target.value)} />
          </div>
        </div>
        <Button onClick={handlePasswordChange} className="mt-3" size="sm" disabled={!currentPw || !newPw || !confirmPw}>
          Passwort ändern
        </Button>
      </section>

      {/* 2FA */}
      <section className="mb-8">
        <h3 className="text-sm font-medium text-foreground mb-3">Zwei-Faktor-Authentifizierung</h3>

        {!security.twoFactorEnabled ? (
          <>
            <div className="flex items-center justify-between rounded-lg border border-border bg-card p-4">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-light">
                  <Key className="h-5 w-5 text-primary" />
                </div>
                <div>
                  <p className="text-sm font-medium text-foreground">2FA nicht aktiviert</p>
                  <p className="text-xs text-muted-foreground">Schuetze dein Konto mit einem zweiten Faktor</p>
                </div>
              </div>
              <Button size="sm" onClick={() => setShow2FASetup(true)}>
                Aktivieren
              </Button>
            </div>

            {/* 2FA setup flow */}
            {show2FASetup && (
              <div className="mt-4 rounded-lg border border-border bg-card p-4 space-y-4">
                <h4 className="text-sm font-medium text-foreground">2FA einrichten</h4>
                <p className="text-xs text-muted-foreground">
                  Scanne den QR-Code mit deiner Authenticator-App (z.B. Google Authenticator, Authy).
                </p>

                {/* QR code mockup */}
                <div className="flex justify-center">
                  <div className="h-40 w-40 rounded-lg border-2 border-dashed border-border bg-secondary flex items-center justify-center">
                    <div className="text-center">
                      <Key className="h-8 w-8 text-muted-foreground mx-auto mb-1" />
                      <p className="text-[10px] text-muted-foreground">QR-Code</p>
                      <p className="text-[10px] text-muted-foreground">(Simulation)</p>
                    </div>
                  </div>
                </div>

                <div className="text-center">
                  <p className="text-xs text-muted-foreground mb-1">Oder manuell eingeben:</p>
                  <code className="rounded bg-secondary px-2 py-1 text-xs font-mono text-foreground">JBSWY3DPEHPK3PXP</code>
                </div>

                <div className="flex gap-2 justify-end">
                  <Button variant="outline" size="sm" onClick={() => setShow2FASetup(false)}>
                    Abbrechen
                  </Button>
                  <Button size="sm" onClick={handle2FAEnable}>
                    2FA aktivieren
                  </Button>
                </div>
              </div>
            )}
          </>
        ) : (
          <div className="space-y-3">
            <div className="flex items-center justify-between rounded-lg border border-success/30 bg-success/5 p-4">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-success/10">
                  <Check className="h-5 w-5 text-success" />
                </div>
                <div>
                  <p className="text-sm font-medium text-foreground">2FA ist aktiv</p>
                  <p className="text-xs text-muted-foreground">Dein Konto ist durch einen zweiten Faktor geschuetzt</p>
                </div>
              </div>
              <Button variant="outline" size="sm" onClick={() => setShowDisable2FA(true)}>
                Deaktivieren
              </Button>
            </div>

            {/* Backup codes */}
            <div className="rounded-lg border border-border bg-card p-4">
              <h4 className="text-sm font-medium text-foreground mb-2">Backup-Codes</h4>
              <p className="text-xs text-muted-foreground mb-3">
                Bewahre diese Codes sicher auf. Jeder Code kann einmalig verwendet werden.
              </p>
              <div className="grid grid-cols-3 gap-2 mb-3">
                {security.backupCodes.map((code) => (
                  <code key={code} className="rounded bg-secondary px-2 py-1.5 text-center text-xs font-mono text-foreground">
                    {code}
                  </code>
                ))}
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => {
                  navigator.clipboard.writeText(security.backupCodes.join('\n'))
                  toast.success('Backup-Codes in Zwischenablage kopiert')
                }}
              >
                <Copy className="mr-1.5 h-3.5 w-3.5" />
                Codes kopieren
              </Button>
            </div>
          </div>
        )}
      </section>

      {/* Sessions */}
      <section>
        <h3 className="text-sm font-medium text-foreground mb-3">Aktive Sitzungen</h3>
        <div className="space-y-2">
          {mockSessions.map((session) => {
            const Icon = session.icon
            return (
              <div key={session.id} className="flex items-center gap-3 rounded-lg border border-border bg-card p-3">
                <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-secondary">
                  <Icon className="h-4 w-4 text-foreground" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm text-foreground">{session.device}</p>
                  <p className="text-xs text-muted-foreground">
                    {session.location} &middot; {session.lastActive}
                  </p>
                </div>
                {session.isCurrent ? (
                  <span className="rounded-full bg-success-light px-2 py-0.5 text-[10px] text-success font-medium">Aktuell</span>
                ) : (
                  <button
                    onClick={() => setShowRevokeSession(session.id)}
                    className="text-xs text-error hover:underline"
                  >
                    Abmelden
                  </button>
                )}
              </div>
            )
          })}
        </div>
      </section>

      {/* Confirm dialogs */}
      <ConfirmDialog
        open={showDisable2FA}
        onOpenChange={setShowDisable2FA}
        title="2FA deaktivieren?"
        description="Dein Konto wird nur noch durch dein Passwort geschuetzt. Dies wird nicht empfohlen."
        confirmLabel="2FA deaktivieren"
        variant="destructive"
        onConfirm={handle2FADisable}
      />

      <ConfirmDialog
        open={showRevokeSession !== null}
        onOpenChange={(open) => { if (!open) setShowRevokeSession(null) }}
        title="Sitzung beenden?"
        description="Das Gerät wird abgemeldet und muss sich erneut anmelden."
        confirmLabel="Abmelden"
        variant="destructive"
        onConfirm={handleRevokeSession}
      />
    </div>
  )
}

// ============================================================
// About Tab
// ============================================================
function AboutTab() {
  const startTour = useTourStore((s) => s.startTour)
  const tours = useTourStore((s) => s.tours)

  return (
    <div className="max-w-2xl mx-auto">
      {/* Hero */}
      <div className="rounded-xl bg-gradient-to-br from-primary to-primary-dark p-8 mb-6">
        <h2 className="text-primary-foreground text-xl font-semibold mb-1">KMU Hub</h2>
        <p className="text-primary-foreground/80 text-sm mb-3">
          All-in-One Business-Plattform fuer DACH-KMUs
        </p>
        <div className="flex items-center gap-3">
          <span className="rounded-full bg-white/15 px-3 py-1 text-xs text-primary-foreground">v0.1.0 Beta</span>
          <span className="rounded-full bg-white/15 px-3 py-1 text-xs text-primary-foreground">Enterprise</span>
        </div>
      </div>

      {/* Support & Contact */}
      <h3 className="text-sm font-medium text-foreground mb-3">Support & Kontakt</h3>
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 mb-8">
        <div className="rounded-lg border border-border bg-card p-4">
          <div className="flex items-center gap-3 mb-2">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary-light">
              <Mail className="h-4 w-4 text-primary" />
            </div>
            <div>
              <p className="text-sm font-medium text-foreground">E-Mail Support</p>
              <p className="text-xs text-muted-foreground">Mo-Fr, 08:00-18:00 Uhr</p>
            </div>
          </div>
          <p className="text-sm text-primary ml-12">support@kmuhub.ch</p>
        </div>
        <div className="rounded-lg border border-border bg-card p-4">
          <div className="flex items-center gap-3 mb-2">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary-light">
              <Calendar className="h-4 w-4 text-primary" />
            </div>
            <div>
              <p className="text-sm font-medium text-foreground">Telefon-Support</p>
              <p className="text-xs text-muted-foreground">Mo-Fr, 09:00-17:00 Uhr</p>
            </div>
          </div>
          <p className="text-sm text-primary ml-12">+41 44 000 00 00</p>
        </div>
        <div className="rounded-lg border border-border bg-card p-4">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-secondary">
              <Info className="h-4 w-4 text-muted-foreground" />
            </div>
            <div>
              <p className="text-sm font-medium text-foreground">Wissensdatenbank</p>
              <p className="text-xs text-muted-foreground">docs.kmuhub.ch</p>
            </div>
          </div>
        </div>
        <div className="rounded-lg border border-border bg-card p-4">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-secondary">
              <Info className="h-4 w-4 text-muted-foreground" />
            </div>
            <div>
              <p className="text-sm font-medium text-foreground">Website</p>
              <p className="text-xs text-muted-foreground">www.kmuhub.ch</p>
            </div>
          </div>
        </div>
      </div>

      {/* Interactive Tours */}
      <h3 className="text-sm font-medium text-foreground mb-1">Interaktive Touren</h3>
      <p className="text-xs text-muted-foreground mb-3">Starte eine gefuehrte Tour um die App oder einzelne Module kennenzulernen</p>
      <div className="space-y-2 mb-8">
        {tours.map((tour) => (
          <div
            key={tour.id}
            className="flex items-center justify-between rounded-lg border border-border bg-card p-3 hover:bg-secondary/50 transition-colors"
          >
            <div>
              <p className="text-sm font-medium text-foreground">{tour.name}</p>
              <p className="text-xs text-muted-foreground">{tour.description} · {tour.steps.length} Schritte</p>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => startTour(tour.id)}
            >
              Tour starten
            </Button>
          </div>
        ))}
      </div>

      {/* System info */}
      <h3 className="text-sm font-medium text-foreground mb-3">System</h3>
      <div className="rounded-lg border border-border bg-card p-4 text-xs text-muted-foreground space-y-1">
        <div className="flex justify-between"><span>Version</span><span className="text-foreground">0.1.0 (Beta)</span></div>
        <div className="flex justify-between"><span>Lizenz</span><span className="text-foreground">Enterprise</span></div>
        <div className="flex justify-between"><span>Hosting</span><span className="text-foreground">EU (Hetzner)</span></div>
        <div className="flex justify-between"><span>Datenschutz</span><span className="text-foreground">DSGVO-konform</span></div>
      </div>
    </div>
  )
}

// ============================================================
// Header Widget Picker — inline component for Appearance tab
// ============================================================
function HeaderWidgetPicker({ className }: { className?: string }) {
  const headerWidgets = useUIStore((s) => s.headerWidgets)
  const toggleHeaderWidget = useUIStore((s) => s.toggleHeaderWidget)

  return (
    <div className={`grid grid-cols-5 gap-2 ${className ?? ''}`}>
      {headerWidgetList.map((widget) => {
        const active = headerWidgets.includes(widget.id)
        const atMax = headerWidgets.length >= 3 && !active
        const Icon = widget.icon
        return (
          <button
            key={widget.id}
            onClick={() => toggleHeaderWidget(widget.id)}
            disabled={atMax}
            className={`relative flex flex-col items-center gap-1.5 rounded-lg border-2 p-3 text-center transition-all ${
              active
                ? 'border-primary bg-primary-light'
                : atMax
                  ? 'border-border bg-muted/50 opacity-50 cursor-not-allowed'
                  : 'border-border bg-card hover:border-muted-foreground/30'
            }`}
          >
            {active && (
              <span className="absolute top-1 right-1">
                <Check className="h-3 w-3 text-primary" />
              </span>
            )}
            <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${
              active ? 'bg-primary/10' : 'bg-secondary'
            }`}>
              <Icon className={`h-4 w-4 ${active ? 'text-primary' : 'text-muted-foreground'}`} />
            </div>
            <p className="text-xs font-medium text-foreground">{widget.name}</p>
            <p className="text-[9px] text-muted-foreground leading-tight">{widget.description}</p>
          </button>
        )
      })}
    </div>
  )
}


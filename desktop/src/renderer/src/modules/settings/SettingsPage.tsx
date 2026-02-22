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
  Users,
  Lock,
  Copy,
  Eye,
  EyeOff,
  X,
  Plus,
  Building2,
  Landmark,
  RefreshCw,
  Plug,
  Sparkles,
} from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { toast } from 'sonner'
import { ConfirmDialog } from '@/components/shared'
import { useSettingsStore } from '@/stores/settings'
import { useAuthStore } from '@/stores/auth'
import { useUIStore } from '@/stores/ui'
import { canSeeSettingsTab } from '@/config/roles'
import { MailSettingsTab } from './tabs/MailSettingsTab'
import { DESK_THEMES, DESK_THEME_ORDER } from '@/config/desk-themes'
import { getDecosForTheme, DECO_ASSETS } from '@/config/desk-asset-urls'
import { CalendarSettingsTab } from './tabs/CalendarSettingsTab'
import { FinanceSettingsTab } from './tabs/FinanceSettingsTab'
import { TeamSettingsTab } from './tabs/TeamSettingsTab'
import { PrivacySettingsTab } from './tabs/PrivacySettingsTab'
import { useProfileStore } from '@/stores/profile'
import { PaletteSwitcher } from '@/components/shared/PaletteSwitcher'
import { LayoutSwitcher } from '@/components/shared/LayoutSwitcher'
import { BUSINESS_PROFILES } from '@/config/business-profiles'
import { CalDAVSettingsTab } from './tabs/CalDAVSettingsTab'
import { CompanySettingsTab } from './tabs/CompanySettingsTab'
import { NotificationSettingsTab } from './tabs/NotificationSettingsTab'
import { IntegrationSettingsTab } from './tabs/IntegrationSettingsTab'
import { AIGovernanceTab } from './tabs/AIGovernanceTab'

type TabKey = 'profile' | 'appearance' | 'language' | 'security' | 'notifications' | 'mail' | 'calendar' | 'caldav' | 'finance' | 'company' | 'integrations' | 'team' | 'privacy' | 'ai' | 'business' | 'about'

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
  { key: 'caldav', label: 'CalDAV/CardDAV', icon: RefreshCw, group: 'Module' },
  { key: 'finance', label: 'Buchhaltung', icon: Receipt, group: 'Module' },
  { key: 'company', label: 'Firma', icon: Landmark, group: 'Admin' },
  { key: 'integrations', label: 'Integrationen', icon: Plug, group: 'Admin' },
  { key: 'business', label: 'Branchenprofil', icon: Building2, group: 'Admin' },
  { key: 'team', label: 'Team & HR', icon: Users, group: 'Admin' },
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
    <div className="flex h-full overflow-hidden">
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
        {effectiveTab === 'caldav' && <CalDAVSettingsTab />}
        {effectiveTab === 'finance' && <FinanceSettingsTab />}
        {effectiveTab === 'company' && <CompanySettingsTab />}
        {effectiveTab === 'integrations' && <IntegrationSettingsTab />}
        {effectiveTab === 'team' && <TeamSettingsTab />}
        {effectiveTab === 'privacy' && <PrivacySettingsTab />}
        {effectiveTab === 'ai' && <AIGovernanceTab />}
        {effectiveTab === 'business' && <BusinessProfileTab />}
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
    <div className="max-w-2xl">
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
  const [fontSize, setFontSize] = useState(appearance.fontSize)

  const theme = useUIStore((s) => s.theme)
  const setTheme = useUIStore((s) => s.setTheme)
  const uiLook = useUIStore((s) => s.uiLook)
  const setUILook = useUIStore((s) => s.setUILook)
  const deskThemeId = useUIStore((s) => s.deskThemeId)
  const setDeskThemeAndDecorations = useUIStore((s) => s.setDeskThemeAndDecorations)
  const deskDecorations = useUIStore((s) => s.deskDecorations)
  const setDeskDecoration = useUIStore((s) => s.setDeskDecoration)

  const themes = [
    { id: 'light' as const, label: 'Hell', desc: 'Warme, helle Oberfläche' },
    { id: 'dark' as const, label: 'Dunkel', desc: 'Augenfreundlich bei wenig Licht' },
    { id: 'auto' as const, label: 'System', desc: 'Folgt den Systemeinstellungen' },
  ]

  const handleThemeChange = (t: 'light' | 'dark' | 'auto') => {
    setTheme(t)
    updateAppearance({ theme: t })
  }

  const handleSave = () => {
    updateAppearance({ theme, fontSize })
    toast.success('Darstellung gespeichert')
  }

  // Theme-aware available decorations
  const availableDecos = getDecosForTheme(deskThemeId)
  const activeDecoIds = Object.values(deskDecorations)
    .filter((p) => p.type === 'image')
    .map((p) => p.variant)

  // Find first free image mount point for adding a decoration
  const currentTheme = DESK_THEMES[deskThemeId]
  const freeImageSlots = currentTheme?.mountPoints.filter(
    (mp) => mp.acceptTypes.includes('image') && !deskDecorations[mp.id]
  ) ?? []

  const handleAddDeco = (decoId: string) => {
    if (freeImageSlots.length === 0) {
      toast.error('Alle Plätze belegt — entferne erst eine Deko')
      return
    }
    const mp = freeImageSlots[0]
    const asset = DECO_ASSETS.find((d) => d.id === decoId)
    if (!asset) return
    setDeskDecoration(mp.id, {
      slotId: mp.id,
      type: 'image',
      variant: asset.id,
      imageUrl: asset.image,
      animationClass: asset.animation ?? undefined,
    })
  }

  const handleRemoveDeco = (slotId: string) => {
    setDeskDecoration(slotId, null)
  }

  return (
    <div className="max-w-2xl">
      <h2 className="text-foreground mb-1">Darstellung</h2>
      <p className="text-sm text-muted-foreground mb-6">Passe das Erscheinungsbild der App an</p>

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

      {/* ── NAVIGATION LAYOUT ────────────────────────── */}
      <h3 className="text-sm font-medium text-foreground mb-3">Navigation</h3>
      <p className="text-xs text-muted-foreground mb-3">Waehle wie du durch die App navigierst</p>
      <LayoutSwitcher className="mb-8" />

      {/* ── UI LOOK PICKER ──────────────────────────── */}
      <h3 className="text-sm font-medium text-foreground mb-3">Oberflächenstil</h3>
      <div className="grid grid-cols-3 gap-3 mb-8">
        {([
          { id: 'solid' as const, label: 'Standard', desc: 'Solide, opake Oberflächen' },
          { id: 'glass' as const, label: 'Milchglas', desc: 'Halbtransparent mit Blur' },
          { id: 'crystal' as const, label: 'Transparent', desc: 'Durchsichtig, Inhalte milchig' },
        ]).map((look) => (
          <button
            key={look.id}
            onClick={() => setUILook(look.id)}
            className={`relative rounded-lg border p-4 text-center transition-colors ${
              uiLook === look.id
                ? 'border-primary bg-primary-light'
                : 'border-border bg-card hover:bg-secondary'
            }`}
          >
            {uiLook === look.id && (
              <span className="absolute top-2 right-2">
                <Check className="h-4 w-4 text-primary" />
              </span>
            )}
            <div className={`mx-auto mb-2 h-10 w-16 rounded border ${
              look.id === 'solid'
                ? 'bg-card border-card-border'
                : 'border-card-border/40'
            }`} style={look.id === 'glass' ? {
              background: 'linear-gradient(135deg, rgba(245,239,232,0.5) 0%, rgba(232,227,221,0.3) 100%)',
              backdropFilter: 'blur(4px)',
            } : look.id === 'crystal' ? {
              background: 'linear-gradient(135deg, rgba(245,239,232,0.25) 0%, rgba(232,227,221,0.15) 100%)',
              backdropFilter: 'blur(6px)',
            } : undefined} />
            <p className="text-sm font-medium text-foreground">{look.label}</p>
            <p className="text-xs text-muted-foreground mt-0.5">{look.desc}</p>
          </button>
        ))}
      </div>

      {/* ── DESK THEME PICKER ─────────────────────────── */}
      <h3 className="text-sm font-medium text-foreground mb-3">Arbeitsplatz-Theme</h3>
      <p className="text-xs text-muted-foreground mb-3">Wähle das Ambiente deines virtuellen Schreibtischs</p>
      <div className="grid grid-cols-3 gap-3 mb-8">
        {DESK_THEME_ORDER.map((id) => {
          const dt = DESK_THEMES[id]
          const isActive = deskThemeId === id
          return (
            <button
              key={id}
              onClick={() => setDeskThemeAndDecorations(id)}
              className={`relative rounded-xl border-2 overflow-hidden transition-all ${
                isActive
                  ? 'border-primary ring-2 ring-primary/20 scale-[1.02]'
                  : 'border-border hover:border-muted-foreground/30'
              }`}
            >
              {/* CSS gradient preview from theme roomVariables */}
              <div
                className="aspect-[4/3]"
                style={{ background: dt.roomVariables['desk-room-bg'] ?? '#e5e5e5' }}
              />
              <div className="p-2 bg-card">
                <p className="text-xs font-medium text-foreground truncate">{dt.name}</p>
                <p className="text-[10px] text-muted-foreground truncate">{dt.description}</p>
              </div>
              {dt.isMinimal && (
                <span className="absolute top-1.5 left-1.5 bg-muted text-muted-foreground rounded-full px-2 py-0.5 text-[9px] font-medium">
                  Kein Schreibtisch
                </span>
              )}
              {isActive && (
                <span className="absolute top-1.5 right-1.5 bg-primary rounded-full p-0.5">
                  <Check className="h-3 w-3 text-white" />
                </span>
              )}
            </button>
          )
        })}
      </div>

      {/* ── DEKO-REGAL ────────────────────────────────── */}
      {!currentTheme?.isMinimal && (
        <>
          <h3 className="text-sm font-medium text-foreground mb-3">Dekorationen</h3>
          <p className="text-xs text-muted-foreground mb-3">Platziere Objekte auf deinem Schreibtisch</p>

          {/* Active decorations */}
          <div className="mb-4">
            <p className="text-xs text-muted-foreground mb-2">Auf dem Schreibtisch</p>
            <div className="flex flex-wrap gap-2 min-h-[60px] rounded-lg border border-border bg-card/50 p-3">
              {Object.entries(deskDecorations)
                .filter(([, p]) => p.type === 'image')
                .map(([slotId, placement]) => {
                  const asset = DECO_ASSETS.find((d) => d.id === placement.variant)
                  if (!asset) return null
                  return (
                    <div
                      key={slotId}
                      className="relative group flex flex-col items-center gap-1 rounded-lg border border-border bg-card p-2 w-[72px]"
                    >
                      <img
                        src={asset.image}
                        alt={asset.name}
                        className="h-10 w-10 object-contain"
                      />
                      <span className="text-[9px] text-muted-foreground text-center leading-tight truncate w-full">{asset.name}</span>
                      <button
                        onClick={() => handleRemoveDeco(slotId)}
                        className="absolute -top-1.5 -right-1.5 bg-destructive text-white rounded-full p-0.5 opacity-0 group-hover:opacity-100 transition-opacity"
                      >
                        <X className="h-3 w-3" />
                      </button>
                    </div>
                  )
                })}
              {Object.values(deskDecorations).filter((p) => p.type === 'image').length === 0 && (
                <p className="text-xs text-muted-foreground italic m-auto">Keine Dekorationen platziert</p>
              )}
            </div>
          </div>

          {/* Available decorations */}
          <div>
            <p className="text-xs text-muted-foreground mb-2">Verfuegbar ({availableDecos.length})</p>
            <div className="flex flex-wrap gap-2 rounded-lg border border-dashed border-border p-3">
              {availableDecos.map((deco) => {
                const isPlaced = activeDecoIds.includes(deco.id)
                return (
                  <button
                    key={deco.id}
                    onClick={() => {
                      if (!isPlaced) handleAddDeco(deco.id)
                    }}
                    disabled={isPlaced}
                    className={`flex flex-col items-center gap-1 rounded-lg border p-2 w-[72px] transition-all ${
                      isPlaced
                        ? 'border-border/50 opacity-40 cursor-not-allowed'
                        : 'border-border bg-card hover:border-primary hover:bg-primary-light cursor-pointer'
                    }`}
                  >
                    <img
                      src={deco.image}
                      alt={deco.name}
                      className="h-10 w-10 object-contain"
                    />
                    <span className="text-[9px] text-muted-foreground text-center leading-tight truncate w-full">{deco.name}</span>
                    {!isPlaced && (
                      <Plus className="h-3 w-3 text-muted-foreground" />
                    )}
                  </button>
                )
              })}
            </div>
          </div>
        </>
      )}

      <div className="mt-8 mb-4 border-t border-border" />

      <h3 className="text-sm font-medium text-foreground mb-3">Schriftgröße</h3>
      <div className="flex items-center gap-3 mb-2">
        <span className="text-xs text-muted-foreground">Klein</span>
        <input
          type="range"
          min={12}
          max={20}
          value={fontSize}
          onChange={(e) => setFontSize(Number(e.target.value))}
          className="flex-1 accent-[var(--primary)]"
        />
        <span className="text-xs text-muted-foreground">Gross</span>
      </div>
      <p className="text-xs text-muted-foreground mb-8">{fontSize}px</p>

      <Button onClick={handleSave}>Speichern</Button>
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
    <div className="max-w-2xl">
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
  const { security, enable2FA, disable2FA } = useSettingsStore()
  const [showCurrent, setShowCurrent] = useState(false)
  const [showNew, setShowNew] = useState(false)
  const [currentPw, setCurrentPw] = useState('')
  const [newPw, setNewPw] = useState('')
  const [confirmPw, setConfirmPw] = useState('')
  const [show2FASetup, setShow2FASetup] = useState(false)
  const [showDisable2FA, setShowDisable2FA] = useState(false)
  const [showRevokeSession, setShowRevokeSession] = useState<number | null>(null)

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
    <div className="max-w-2xl">
      <h2 className="text-foreground mb-1">Sicherheit</h2>
      <p className="text-sm text-muted-foreground mb-6">Passwort, 2FA und Sitzungen verwalten</p>

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
  return (
    <div className="max-w-2xl">
      <div className="rounded-lg bg-gradient-to-br from-primary to-primary-dark p-8 mb-6">
        <h2 className="text-primary-foreground text-xl mb-2">KMU Hub</h2>
        <p className="text-primary-foreground/80 text-sm">
          All-in-One CRM für DACH-KMUs mit EU-Datensouveränität
        </p>
      </div>

      <div className="grid grid-cols-2 gap-4 mb-6">
        <div className="rounded-lg border border-border bg-card p-4">
          <p className="text-xs text-muted-foreground mb-1">Version</p>
          <p className="text-sm font-medium text-foreground">0.1.0 (Beta)</p>
        </div>
        <div className="rounded-lg border border-border bg-card p-4">
          <p className="text-xs text-muted-foreground mb-1">Lizenz</p>
          <p className="text-sm font-medium text-foreground">Enterprise</p>
        </div>
      </div>

      <h3 className="text-sm font-medium text-foreground mb-3">Features</h3>
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 mb-6">
        {[
          { title: 'EU-Datensouveränität', desc: 'Hosting nur auf EU-Servern' },
          { title: 'KMU-optimiert', desc: 'Für 5-200 Mitarbeiter' },
          { title: 'Self-Hosted Option', desc: 'Volle Kontrolle über deine Daten' },
        ].map((f) => (
          <div key={f.title} className="rounded-lg border border-border bg-card p-3">
            <p className="text-sm font-medium text-foreground mb-0.5">{f.title}</p>
            <p className="text-xs text-muted-foreground">{f.desc}</p>
          </div>
        ))}
      </div>

      <h3 className="text-sm font-medium text-foreground mb-3">Kontakt</h3>
      <div className="space-y-2 text-sm text-muted-foreground">
        <p>Support: support@kmuhub.ch</p>
        <p>Website: www.kmuhub.ch</p>
      </div>
    </div>
  )
}

// ============================================================
// Business Profile Tab — industry profile + optional modules
// ============================================================
function BusinessProfileTab() {
  const { businessProfileId, enabledOptionalModules, setBusinessProfile, enableModule, disableModule } = useProfileStore()
  const selectedProfile = BUSINESS_PROFILES.find((p) => p.id === businessProfileId)

  return (
    <div className="max-w-3xl">
      <h2 className="text-lg font-semibold text-foreground mb-1">Branchenprofil</h2>
      <p className="text-sm text-muted-foreground mb-6">
        Wählen Sie das Profil das am besten zu Ihrem Unternehmen passt. Module werden entsprechend angezeigt.
      </p>

      {/* Profile grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 mb-8">
        {BUSINESS_PROFILES.map((profile) => {
          const isActive = businessProfileId === profile.id
          return (
            <button
              key={profile.id}
              onClick={() => setBusinessProfile(profile.id)}
              className={`flex items-start gap-3 rounded-lg border-2 p-4 text-left transition-all ${
                isActive
                  ? 'border-primary bg-primary/5'
                  : 'border-border hover:border-primary/40'
              }`}
            >
              <span className="text-2xl shrink-0">{profile.emoji}</span>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <p className={`text-sm font-medium ${isActive ? 'text-primary' : 'text-foreground'}`}>
                    {profile.name}
                  </p>
                  {isActive && <Check className="h-4 w-4 text-primary shrink-0" />}
                </div>
                <p className="text-xs text-muted-foreground mt-0.5">{profile.description}</p>
                <p className="text-[10px] text-muted-foreground mt-1">
                  z.B. {profile.examples.slice(0, 3).join(', ')}
                </p>
                <p className="text-[10px] text-muted-foreground mt-0.5">
                  {profile.defaultModules.length} Module · {profile.optionalModules.length} optional
                </p>
              </div>
            </button>
          )
        })}
      </div>

      {/* Optional modules */}
      {selectedProfile && selectedProfile.optionalModules.length > 0 && (
        <>
          <h3 className="text-sm font-medium text-foreground mb-1">Optionale Module</h3>
          <p className="text-xs text-muted-foreground mb-4">
            Erweitern Sie Ihr &quot;{selectedProfile.name}&quot;-Profil mit zusätzlichen Modulen.
          </p>
          <div className="space-y-2">
            {selectedProfile.optionalModules.map((moduleId) => {
              const isEnabled = enabledOptionalModules.includes(moduleId)
              const moduleLabel = MODULE_LABELS[moduleId] ?? moduleId
              return (
                <div
                  key={moduleId}
                  className="flex items-center justify-between rounded-lg border border-border p-3"
                >
                  <span className="text-sm text-foreground">{moduleLabel}</span>
                  <Switch
                    checked={isEnabled}
                    onCheckedChange={(checked: boolean) => {
                      if (checked) enableModule(moduleId)
                      else disableModule(moduleId)
                      toast.success(checked ? `${moduleLabel} aktiviert` : `${moduleLabel} deaktiviert`)
                    }}
                  />
                </div>
              )
            })}
          </div>
        </>
      )}
    </div>
  )
}

/** Label map for module IDs used in the optional modules list */
const MODULE_LABELS: Record<string, string> = {
  crm: 'CRM',
  projects: 'Projekte',
  tasks: 'Aufgaben',
  chat: 'Nachrichten',
  calendar: 'Kalender',
  meetings: 'Meetings',
  documents: 'Dokumente',
  mail: 'E-Mail',
  contacts: 'Kontakte',
  team: 'Team',
  finance: 'Buchhaltung',
  infrastructure: 'Infrastruktur',
  inventar: 'Inventar',
  schichten: 'Schichtplanung',
  einkauf: 'Einkauf',
  helpdesk: 'Helpdesk',
  fuhrpark: 'Fuhrpark',
  produktion: 'Produktion',
  berichte: 'Berichte',
  vertraege: 'Vertraege',
  rapporte: 'Rapporte',
  zeiterfassung: 'Zeiterfassung',
  formulare: 'Formulare',
  vermietung: 'Vermietung',
}

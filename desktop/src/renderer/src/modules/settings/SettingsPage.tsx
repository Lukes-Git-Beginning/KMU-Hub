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
  Lock,
  Copy,
  Eye,
  EyeOff,
  Landmark,
  Plug,
  Bot,
  CreditCard,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { toast } from 'sonner'
import { ConfirmDialog } from '@/components/shared'
import { useSettingsStore } from '@/stores/settings'
import { useAuthStore } from '@/stores/auth'
import { useUIStore } from '@/stores/ui'
import { DESK_BACKGROUNDS } from '@/components/layout/DeskEnvironment'
import { canSeeSettingsTab } from '@/config/roles'
import { PrivacySettingsTab } from './tabs/PrivacySettingsTab'
import { PaletteSwitcher } from '@/components/shared/PaletteSwitcher'
import { LayoutSwitcher } from '@/components/shared/LayoutSwitcher'
import { headerWidgetList } from '@/components/header/header-widgets'
import { CompanySettingsTab } from './tabs/CompanySettingsTab'
import { BillingSettingsTab } from './tabs/BillingSettingsTab'
import { NotificationSettingsTab } from './tabs/NotificationSettingsTab'
import { IntegrationSettingsTab } from './tabs/IntegrationSettingsTab'
import { AIGovernanceTab } from './tabs/AIGovernanceTab'
import { ITAdminTab } from './tabs/ITAdminTab'
import { ThemePreview } from './ThemePreview'
import { useTourStore } from '@/stores/tour'
import { branding } from '@/config/branding'
import { useNavigate, useLocation } from 'react-router-dom'
import { formatDate } from '@/lib/format'

type TabKey = 'profile' | 'appearance' | 'language' | 'security' | 'notifications' | 'company' | 'billing' | 'it-admin' | 'integrations' | 'privacy' | 'ai' | 'about'

interface TabConfig {
  key: TabKey
  labelKey: string
  icon: typeof User
  groupKey?: string
}

interface TabConfigExtended extends TabConfig {
  /** Tabs mit adminOnly=true werden im distributed-Modus ausgeblendet. */
  adminOnly?: boolean
}

const ALL_TABS: TabConfigExtended[] = [
  { key: 'profile', labelKey: 'settings.tabs.profile', icon: User, groupKey: 'settings.groups.personal' },
  { key: 'appearance', labelKey: 'settings.tabs.appearance', icon: Palette, groupKey: 'settings.groups.personal' },
  { key: 'language', labelKey: 'settings.tabs.language', icon: Globe, groupKey: 'settings.groups.personal' },
  { key: 'security', labelKey: 'settings.tabs.security', icon: Shield, groupKey: 'settings.groups.personal' },
  { key: 'notifications', labelKey: 'settings.tabs.notifications', icon: Bell, groupKey: 'settings.groups.personal' },
  { key: 'company', labelKey: 'settings.tabs.company', icon: Landmark, groupKey: 'settings.groups.admin', adminOnly: true },
  { key: 'billing', labelKey: 'settings.tabs.billing', icon: CreditCard, groupKey: 'settings.groups.admin', adminOnly: true },
  { key: 'it-admin', labelKey: 'settings.tabs.itAdmin', icon: Monitor, groupKey: 'settings.groups.admin', adminOnly: true },
  { key: 'integrations', labelKey: 'settings.tabs.integrations', icon: Plug, groupKey: 'settings.groups.admin', adminOnly: true },
  { key: 'privacy', labelKey: 'settings.tabs.privacy', icon: Lock, groupKey: 'settings.groups.personal' },
  { key: 'ai', labelKey: 'settings.tabs.ai', icon: Bot, groupKey: 'settings.groups.personal' },
  { key: 'about', labelKey: 'settings.tabs.about', icon: Info, groupKey: 'settings.groups.other' },
]

const TAB_GROUP_KEYS = [
  'settings.groups.personal',
  'settings.groups.modules',
  'settings.groups.admin',
  'settings.groups.other',
]

export default function SettingsPage() {
  const { t } = useTranslation()
  const location = useLocation()
  const user = useAuthStore((s) => s.user)
  const navigate = useNavigate()

  // Deep-link support: navigate('/settings', { state: { tab: 'notifications' } })
  // passes the desired tab via router state so the URL stays clean.
  const stateTab = (location.state as { tab?: string } | null)?.tab as TabKey | undefined
  const [activeTab, setActiveTab] = useState<TabKey>(stateTab ?? 'profile')

  // Filter tabs:
  // 1. RBAC — restricted tabs are INVISIBLE
  // 2. adminOnly Tabs — IMMER ausgeblendet (Admin-Funktionen leben im Admin-Modul links unten)
  const tabs = ALL_TABS.filter((tab) => {
    if (!canSeeSettingsTab(user, tab.key)) return false
    if (tab.adminOnly) return false
    return true
  })

  // If the active tab got hidden (e.g. after role switch or mode change), fall back to profile
  const isActiveVisible = tabs.some((t) => t.key === activeTab)
  const effectiveTab = isActiveVisible ? activeTab : 'profile'

  // Handle backwards-compat redirects: ?tab=it-admin → /admin/it etc.
  const searchParams = new URLSearchParams(window.location.hash.split('?')[1] ?? '')
  const tabParam = searchParams.get('tab')
  if (tabParam === 'it-admin') {
    navigate('/admin/it', { replace: true })
    return null
  }
  if (tabParam === 'billing') {
    navigate('/admin/billing', { replace: true })
    return null
  }

  return (
    <div className="flex h-full overflow-hidden animate-fade-up">
      {/* Settings sidebar */}
      <aside className="w-56 shrink-0 border-r border-border bg-card p-4 overflow-y-auto">
        <h3 className="text-sm font-medium text-foreground mb-4 px-2">{t('settings.title')}</h3>
        <nav className="space-y-4">
          {TAB_GROUP_KEYS.map((groupKey) => {
            const groupTabs = tabs.filter((tab) => tab.groupKey === groupKey)
            if (groupTabs.length === 0) return null
            return (
              <div key={groupKey}>
                <p className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground px-3 mb-1">{t(groupKey)}</p>
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
                        {t(tab.labelKey)}
                      </button>
                    )
                  })}
                </div>
              </div>
            )
          })}

          {/* Hinweis auf Admin-Modul (für admin/it_support) */}
          {user?.roles.some((r) => ['admin', 'it_support'].includes(r)) && (
            <div className="px-3 mt-2">
              <p className="text-[10px] text-muted-foreground leading-relaxed">
                {t('settings.adminBlock.distributedHint', { defaultValue: 'Tenant-weite Administration im Modul links unten.' })}
              </p>
            </div>
          )}
        </nav>
      </aside>

      {/* Content area */}
      <div className="flex-1 overflow-y-auto p-6">
        {effectiveTab === 'profile' && <ProfileTab />}
        {effectiveTab === 'appearance' && <AppearanceTab />}
        {effectiveTab === 'language' && <LanguageTab />}
        {effectiveTab === 'security' && <SecurityTab />}
        {effectiveTab === 'notifications' && <NotificationSettingsTab />}
        {effectiveTab === 'company' && <CompanySettingsTab />}
        {effectiveTab === 'billing' && <BillingSettingsTab />}
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
  const { t } = useTranslation()
  const { profile, updateProfile } = useSettingsStore()
  const [firstName, setFirstName] = useState(profile.firstName)
  const [lastName, setLastName] = useState(profile.lastName)
  const [email, setEmail] = useState(profile.email)
  const [phone, setPhone] = useState(profile.phone)
  const [position, setPosition] = useState(profile.position)
  const [bio, setBio] = useState(profile.bio)

  const handleSave = () => {
    updateProfile({ firstName, lastName, email, phone, position, bio })
    toast.success(t('settings.profile.saved'))
  }

  const handlePhotoChange = () => {
    toast.success(t('settings.profile.photoUploading'))
    setTimeout(() => {
      updateProfile({ avatarUrl: 'mock-avatar.jpg' })
      toast.success(t('settings.profile.photoUpdated'))
    }, 1000)
  }

  const initials = `${firstName.charAt(0)}${lastName.charAt(0)}`.toUpperCase()

  return (
    <div className="max-w-2xl mx-auto">
      <h2 className="text-foreground mb-1">{t('settings.tabs.profile')}</h2>
      <p className="text-sm text-muted-foreground mb-6">{t('settings.profile.subtitle')}</p>

      {/* Avatar */}
      <div className="flex items-center gap-4 mb-8">
        <div className="flex h-20 w-20 items-center justify-center rounded-full bg-primary-light text-2xl font-medium text-primary">
          {initials}
        </div>
        <div>
          <Button variant="outline" size="sm" onClick={handlePhotoChange}>
            {t('settings.profile.changePhoto')}
          </Button>
          <p className="text-xs text-muted-foreground mt-1">{t('settings.profile.photoHint')}</p>
        </div>
      </div>

      {/* Form fields */}
      <div className="space-y-4">
        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-1.5">
            <label className="block text-sm font-medium text-foreground">{t('settings.profile.firstName')}</label>
            <Input value={firstName} onChange={(e) => setFirstName(e.target.value)} />
          </div>
          <div className="space-y-1.5">
            <label className="block text-sm font-medium text-foreground">{t('settings.profile.lastName')}</label>
            <Input value={lastName} onChange={(e) => setLastName(e.target.value)} />
          </div>
        </div>
        <div className="space-y-1.5">
          <label className="block text-sm font-medium text-foreground">{t('settings.profile.email')}</label>
          <Input type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
        </div>
        <div className="space-y-1.5">
          <label className="block text-sm font-medium text-foreground">{t('settings.profile.phone')}</label>
          <Input type="tel" value={phone} onChange={(e) => setPhone(e.target.value)} />
        </div>
        <div className="space-y-1.5">
          <label className="block text-sm font-medium text-foreground">{t('settings.profile.position')}</label>
          <Input value={position} onChange={(e) => setPosition(e.target.value)} />
        </div>
        <div className="space-y-1.5">
          <label className="block text-sm font-medium text-foreground">{t('settings.profile.bio')}</label>
          <textarea
            value={bio}
            onChange={(e) => setBio(e.target.value)}
            rows={3}
            className="w-full rounded-lg border border-border bg-input-background px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring resize-none"
          />
        </div>
      </div>

      <div className="mt-6 flex gap-2">
        <Button onClick={handleSave}>{t('common.save')}</Button>
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
          {t('common.cancel')}
        </Button>
      </div>
    </div>
  )
}

// ============================================================
// Appearance Tab — wired to store
// ============================================================
function AppearanceTab() {
  const { t } = useTranslation()
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
    { id: 'light' as const, labelKey: 'settings.appearance.themeLight', descKey: 'settings.appearance.themeLightDesc' },
    { id: 'dark' as const, labelKey: 'settings.appearance.themeDark', descKey: 'settings.appearance.themeDarkDesc' },
    { id: 'auto' as const, labelKey: 'settings.appearance.themeSystem', descKey: 'settings.appearance.themeSystemDesc' },
  ]

  const handleThemeChange = (val: 'light' | 'dark' | 'auto') => {
    setTheme(val)
    updateAppearance({ theme: val })
  }

  return (
    <div className="max-w-2xl mx-auto">
      <h2 className="text-foreground mb-1">{t('settings.tabs.appearance')}</h2>
      <p className="text-sm text-muted-foreground mb-6">{t('settings.appearance.subtitle')}</p>

      <ThemePreview className="mb-8" />

      <h3 className="text-sm font-medium text-foreground mb-3">{t('settings.appearance.colorScheme')}</h3>
      <div className="grid grid-cols-3 gap-3 mb-8">
        {themes.map((themeItem) => (
          <button
            key={themeItem.id}
            onClick={() => handleThemeChange(themeItem.id)}
            className={`relative rounded-lg border p-4 text-center transition-colors ${
              theme === themeItem.id
                ? 'border-primary bg-primary-light'
                : 'border-border bg-card hover:bg-secondary'
            }`}
          >
            {theme === themeItem.id && (
              <span className="absolute top-2 right-2">
                <Check className="h-4 w-4 text-primary" />
              </span>
            )}
            <div className={`mx-auto mb-2 h-8 w-8 rounded-full border ${
              themeItem.id === 'light' ? 'bg-amber-100 border-amber-300' :
              themeItem.id === 'dark' ? 'bg-slate-700 border-slate-500' :
              'bg-gradient-to-br from-amber-100 to-slate-700 border-gray-400'
            }`} />
            <p className="text-sm font-medium text-foreground">{t(themeItem.labelKey)}</p>
            <p className="text-xs text-muted-foreground mt-0.5">{t(themeItem.descKey)}</p>
          </button>
        ))}
      </div>

      {/* ── COLOR PALETTE PICKER ────────────────────── */}
      <h3 className="text-sm font-medium text-foreground mb-3">{t('settings.appearance.colorPalette')}</h3>
      <p className="text-xs text-muted-foreground mb-3">{t('settings.appearance.colorPaletteDesc')}</p>
      <PaletteSwitcher className="mb-8" />

      {/* ── ACCENT INTENSITY ─────────────────────────── */}
      <h3 className="text-sm font-medium text-foreground mb-3">{t('settings.appearance.accentIntensity')}</h3>
      <p className="text-xs text-muted-foreground mb-3">{t('settings.appearance.accentIntensityDesc')}</p>
      <div className="grid grid-cols-2 gap-3 mb-8">
        {([
          { id: 'subtle' as const, labelKey: 'settings.appearance.accentSubtle', descKey: 'settings.appearance.accentSubtleDesc' },
          { id: 'vivid' as const, labelKey: 'settings.appearance.accentVivid', descKey: 'settings.appearance.accentVividDesc' },
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
            <p className="text-sm font-medium text-foreground">{t(opt.labelKey)}</p>
            <p className="text-xs text-muted-foreground mt-0.5">{t(opt.descKey)}</p>
          </button>
        ))}
      </div>

      {/* ── NAVIGATION LAYOUT ────────────────────────── */}
      <h3 className="text-sm font-medium text-foreground mb-3">{t('settings.appearance.navigation')}</h3>
      <p className="text-xs text-muted-foreground mb-3">{t('settings.appearance.navigationDesc')}</p>
      <LayoutSwitcher className="mb-8" />

      {/* ── FENSTER-STIL ──────────────────────────── */}
      <h3 className="text-sm font-medium text-foreground mb-3">{t('settings.appearance.windowStyle')}</h3>
      <p className="text-xs text-muted-foreground mb-3">{t('settings.appearance.windowStyleDesc')}</p>
      <div className="grid grid-cols-2 gap-3 mb-8">
        {([
          { id: 'full' as const, labelKey: 'settings.appearance.windowFull', descKey: 'settings.appearance.windowFullDesc' },
          { id: 'bubble' as const, labelKey: 'settings.appearance.windowBubble', descKey: 'settings.appearance.windowBubbleDesc' },
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
            <p className="text-sm font-medium text-foreground">{t(style.labelKey)}</p>
            <p className="text-xs text-muted-foreground mt-0.5">{t(style.descKey)}</p>
          </button>
        ))}
      </div>

      {/* ── OBERFLAECHE (SOLID / MILCHGLAS) ─────────── */}
      <h3 className="text-sm font-medium text-foreground mb-3">{t('settings.appearance.surface')}</h3>
      <p className="text-xs text-muted-foreground mb-3">{t('settings.appearance.surfaceDesc')}</p>
      <div className="grid grid-cols-2 gap-3 mb-8">
        {([
          { id: 'solid' as const, labelKey: 'settings.appearance.surfaceSolid', descKey: 'settings.appearance.surfaceSolidDesc' },
          { id: 'glass' as const, labelKey: 'settings.appearance.surfaceGlass', descKey: 'settings.appearance.surfaceGlassDesc' },
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
            <p className="text-sm font-medium text-foreground">{t(opt.labelKey)}</p>
            <p className="text-xs text-muted-foreground mt-0.5">{t(opt.descKey)}</p>
          </button>
        ))}
      </div>

      {/* ── HINTERGRUND (nur bei Milchglas) ──────────── */}
      {uiLook === 'glass' && (
        <>
          <h3 className="text-sm font-medium text-foreground mb-3">{t('settings.appearance.background')}</h3>
          <p className="text-xs text-muted-foreground mb-3">{t('settings.appearance.backgroundDesc')}</p>
          <div className="grid grid-cols-4 gap-2 mb-8">
            <button
              onClick={() => setDeskBackground(null)}
              className={`relative h-16 rounded-lg border-2 transition-colors flex items-center justify-center ${
                !deskBackground
                  ? 'border-primary'
                  : 'border-border hover:border-primary/40'
              }`}
            >
              <span className="text-xs text-muted-foreground">{t('settings.appearance.bgNone')}</span>
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
      <h3 className="text-sm font-medium text-foreground mb-3">{t('settings.appearance.headerWidgets')}</h3>
      <p className="text-xs text-muted-foreground mb-3">{t('settings.appearance.headerWidgetsDesc')}</p>
      <HeaderWidgetPicker className="mb-8" />

      <div className="mt-4 mb-4 border-t border-border" />

      <h3 className="text-sm font-medium text-foreground mb-3">{t('settings.appearance.fontSize')}</h3>
      <div className="flex items-center gap-3 mb-2">
        <span className="text-xs text-muted-foreground">{t('settings.appearance.fontSmall')}</span>
        <input
          type="range"
          min={12}
          max={20}
          value={appearance.fontSize}
          onChange={(e) => updateAppearance({ fontSize: Number(e.target.value) })}
          className="flex-1 accent-[var(--primary)]"
        />
        <span className="text-xs text-muted-foreground">{t('settings.appearance.fontLarge')}</span>
      </div>
      <p className="text-xs text-muted-foreground mb-8">{appearance.fontSize}px</p>
    </div>
  )
}

// ============================================================
// Language Tab — wired to store
// ============================================================
function LanguageTab() {
  const { t } = useTranslation()
  const { language, updateLanguage } = useSettingsStore()
  const [locale, setLocale] = useState(language.locale)
  const [timezone, setTimezone] = useState(language.timezone)
  const [dateFormat, setDateFormat] = useState(language.dateFormat)

  const languages = [
    { id: 'de' as const, label: 'Deutsch', flag: 'DE' },
    { id: 'en' as const, label: 'English', flag: 'EN' },
    { id: 'fr' as const, label: 'Français', flag: 'FR' },
  ]

  const handleSave = () => {
    updateLanguage({ locale, timezone, dateFormat })
    toast.success(t('settings.languageRegion.saved'))
  }

  return (
    <div className="max-w-2xl mx-auto">
      <h2 className="text-foreground mb-1">{t('settings.tabs.language')}</h2>
      <p className="text-sm text-muted-foreground mb-6">{t('settings.languageRegion.subtitle')}</p>

      <h3 className="text-sm font-medium text-foreground mb-3">{t('settings.languageRegion.language')}</h3>
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
          <label className="block text-sm font-medium text-foreground mb-1.5">{t('settings.languageRegion.timezone')}</label>
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
          <label className="block text-sm font-medium text-foreground mb-1.5">{t('settings.languageRegion.dateFormat')}</label>
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

      <Button onClick={handleSave}>{t('common.save')}</Button>
    </div>
  )
}

// ============================================================
// Security Tab — 2FA setup, sessions, password
// ============================================================
function SecurityTab() {
  const { t } = useTranslation()
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
    { id: 1, device: 'Desktop — Windows', location: 'Zürich, CH', lastActiveKey: 'settings.security.sessionNow', isCurrent: true, icon: Monitor },
    { id: 2, device: 'iPhone 15 Pro', location: 'Zürich, CH', lastActiveKey: 'settings.security.session2h', isCurrent: false, icon: Smartphone },
    { id: 3, device: 'MacBook Pro', location: 'Bern, CH', lastActiveKey: 'settings.security.session3d', isCurrent: false, icon: Monitor },
  ]

  const handlePasswordChange = () => {
    if (!currentPw || !newPw) return
    if (newPw !== confirmPw) {
      toast.error(t('settings.security.passwordMismatch'))
      return
    }
    if (newPw.length < 8) {
      toast.error(t('settings.security.passwordMinLength'))
      return
    }
    updateSecurity({ passwordLastChanged: new Date().toISOString() })
    toast.success(t('settings.security.passwordChanged'))
    setCurrentPw('')
    setNewPw('')
    setConfirmPw('')
  }

  const handle2FAEnable = () => {
    enable2FA()
    setShow2FASetup(false)
    toast.success(t('settings.security.twoFAEnabled'))
  }

  const handle2FADisable = () => {
    disable2FA()
    setShowDisable2FA(false)
    toast.success(t('settings.security.twoFADisabled'))
  }

  const handleRevokeSession = () => {
    toast.success(t('settings.security.sessionRevoked'))
    setShowRevokeSession(null)
  }

  return (
    <div className="max-w-2xl mx-auto">
      <h2 className="text-foreground mb-1">{t('settings.tabs.security')}</h2>
      <p className="text-sm text-muted-foreground mb-6">{t('settings.security.subtitle')}</p>

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
                {isExpired ? t('settings.security.passwordExpired') :
                 isExpiringSoon ? t('settings.security.passwordExpiresSoon', { days: daysUntilExpiry }) :
                 t('settings.security.passwordNextChange', { days: daysUntilExpiry })}
              </p>
              <p className="text-xs text-muted-foreground">
                {t('settings.security.lastChanged')}: {lastChanged ? formatDate(lastChanged) : t('settings.security.unknown')}
                {' · '}{t('settings.security.policy')}: {t('settings.security.everyDays', { days: expiryDays })}
              </p>
            </div>
            {(isExpired || isExpiringSoon) && (
              <span className={`rounded-full px-2.5 py-1 text-[10px] font-medium ${
                isExpired ? 'bg-error/15 text-error' : 'bg-warning/15 text-warning'
              }`}>
                {isExpired ? t('settings.security.changeNow') : t('settings.security.dueSoon')}
              </span>
            )}
          </div>
        </section>
      )}

      {/* Password */}
      <section className="mb-8">
        <h3 className="text-sm font-medium text-foreground mb-3">{t('settings.security.changePassword')}</h3>
        <div className="space-y-3">
          <div className="space-y-1.5">
            <label className="block text-sm font-medium text-foreground">{t('settings.security.currentPassword')}</label>
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
            <label className="block text-sm font-medium text-foreground">{t('settings.security.newPassword')}</label>
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
            <label className="block text-sm font-medium text-foreground">{t('settings.security.confirmPassword')}</label>
            <Input type="password" value={confirmPw} onChange={(e) => setConfirmPw(e.target.value)} />
          </div>
        </div>
        <Button onClick={handlePasswordChange} className="mt-3" size="sm" disabled={!currentPw || !newPw || !confirmPw}>
          {t('settings.security.changePassword')}
        </Button>
      </section>

      {/* 2FA */}
      <section className="mb-8">
        <h3 className="text-sm font-medium text-foreground mb-3">{t('settings.security.twoFA')}</h3>

        {!security.twoFactorEnabled ? (
          <>
            <div className="flex items-center justify-between rounded-lg border border-border bg-card p-4">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-light">
                  <Key className="h-5 w-5 text-primary" />
                </div>
                <div>
                  <p className="text-sm font-medium text-foreground">{t('settings.security.twoFANotActive')}</p>
                  <p className="text-xs text-muted-foreground">{t('settings.security.twoFAProtect')}</p>
                </div>
              </div>
              <Button size="sm" onClick={() => setShow2FASetup(true)}>
                {t('settings.security.enable')}
              </Button>
            </div>

            {/* 2FA setup flow */}
            {show2FASetup && (
              <div className="mt-4 rounded-lg border border-border bg-card p-4 space-y-4">
                <h4 className="text-sm font-medium text-foreground">{t('settings.security.setup2FA')}</h4>
                <p className="text-xs text-muted-foreground">
                  {t('settings.security.scanQR')}
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
                  <p className="text-xs text-muted-foreground mb-1">{t('settings.security.manualEntry')}:</p>
                  <code className="rounded bg-secondary px-2 py-1 text-xs font-mono text-foreground">JBSWY3DPEHPK3PXP</code>
                </div>

                <div className="flex gap-2 justify-end">
                  <Button variant="outline" size="sm" onClick={() => setShow2FASetup(false)}>
                    {t('common.cancel')}
                  </Button>
                  <Button size="sm" onClick={handle2FAEnable}>
                    {t('settings.security.activate2FA')}
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
                  <p className="text-sm font-medium text-foreground">{t('settings.security.twoFAActive')}</p>
                  <p className="text-xs text-muted-foreground">{t('settings.security.twoFAActiveDesc')}</p>
                </div>
              </div>
              <Button variant="outline" size="sm" onClick={() => setShowDisable2FA(true)}>
                {t('settings.security.disable')}
              </Button>
            </div>

            {/* Backup codes */}
            <div className="rounded-lg border border-border bg-card p-4">
              <h4 className="text-sm font-medium text-foreground mb-2">{t('settings.security.backupCodes')}</h4>
              <p className="text-xs text-muted-foreground mb-3">
                {t('settings.security.backupCodesDesc')}
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
                  toast.success(t('settings.security.codesCopied'))
                }}
              >
                <Copy className="mr-1.5 h-3.5 w-3.5" />
                {t('settings.security.copyCodes')}
              </Button>
            </div>
          </div>
        )}
      </section>

      {/* Sessions */}
      <section>
        <h3 className="text-sm font-medium text-foreground mb-3">{t('settings.security.activeSessions')}</h3>
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
                    {session.location} &middot; {t(session.lastActiveKey)}
                  </p>
                </div>
                {session.isCurrent ? (
                  <span className="rounded-full bg-success-light px-2 py-0.5 text-[10px] text-success font-medium">{t('settings.security.current')}</span>
                ) : (
                  <button
                    onClick={() => setShowRevokeSession(session.id)}
                    className="text-xs text-error hover:underline"
                  >
                    {t('settings.security.logout')}
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
        title={t('settings.security.disable2FATitle')}
        description={t('settings.security.disable2FADesc')}
        confirmLabel={t('settings.security.disable2FAConfirm')}
        variant="destructive"
        onConfirm={handle2FADisable}
      />

      <ConfirmDialog
        open={showRevokeSession !== null}
        onOpenChange={(open) => { if (!open) setShowRevokeSession(null) }}
        title={t('settings.security.endSessionTitle')}
        description={t('settings.security.endSessionDesc')}
        confirmLabel={t('settings.security.logout')}
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
  const { t } = useTranslation()
  const startTour = useTourStore((s) => s.startTour)
  const tours = useTourStore((s) => s.tours)

  return (
    <div className="max-w-2xl mx-auto">
      {/* Hero */}
      <div className="rounded-xl border border-border bg-card p-8 mb-6">
        <div className="flex items-center gap-4 mb-3">
          <img src={branding.cosmi.icon128} alt="Cosmi" className="max-h-14 w-auto object-contain" />
          <div>
            <h2 className="text-foreground text-xl font-semibold mb-0.5">Cosmi</h2>
            <p className="text-muted-foreground text-xs">by Zentria</p>
          </div>
        </div>
        <p className="text-muted-foreground text-sm mb-3">
          {t('settings.about.tagline')}
        </p>
        <div className="flex items-center gap-3">
          <span className="rounded-full bg-secondary px-3 py-1 text-xs text-muted-foreground">v0.1.0 Beta</span>
          <span className="rounded-full bg-secondary px-3 py-1 text-xs text-muted-foreground">Enterprise</span>
        </div>
      </div>

      {/* Product family */}
      <h3 className="text-sm font-medium text-foreground mb-3">{t('settings.about.productFamily')}</h3>
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 mb-8">
        <div className="rounded-lg border border-border bg-card p-4 flex flex-col items-center gap-2">
          <img src={branding.cosmi.logoTransparent} alt="Cosmi" className="max-h-12 w-auto object-contain" />
          <p className="text-xs text-muted-foreground">Desktop App</p>
        </div>
        <div className="rounded-lg border border-border bg-card p-4 flex flex-col items-center gap-2">
          <img src={branding.orbit.logoTransparent} alt="Orbit" className="max-h-12 w-auto object-contain" />
          <p className="text-xs text-muted-foreground">Cloud Platform</p>
        </div>
        <div className="rounded-lg border border-border bg-card p-4 flex flex-col items-center gap-2">
          <img src={branding.zentria.logoTransparent} alt="Zentria" className="max-h-12 w-auto object-contain" />
          <p className="text-xs text-muted-foreground">Unternehmen</p>
        </div>
      </div>

      {/* Support & Contact */}
      <h3 className="text-sm font-medium text-foreground mb-3">{t('settings.about.supportContact')}</h3>
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 mb-8">
        <div className="rounded-lg border border-border bg-card p-4">
          <div className="flex items-center gap-3 mb-2">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary-light">
              <Mail className="h-4 w-4 text-primary" />
            </div>
            <div>
              <p className="text-sm font-medium text-foreground">{t('settings.about.emailSupport')}</p>
              <p className="text-xs text-muted-foreground">{t('settings.about.emailSupportHours')}</p>
            </div>
          </div>
          <p className="text-sm text-primary ml-12">support@zentria.tech</p>
        </div>
        <div className="rounded-lg border border-border bg-card p-4">
          <div className="flex items-center gap-3 mb-2">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary-light">
              <Calendar className="h-4 w-4 text-primary" />
            </div>
            <div>
              <p className="text-sm font-medium text-foreground">{t('settings.about.phoneSupport')}</p>
              <p className="text-xs text-muted-foreground">{t('settings.about.phoneSupportHours')}</p>
            </div>
          </div>
          <p className="text-sm text-primary ml-12">+49 30 000 00 00</p>
        </div>
        <div className="rounded-lg border border-border bg-card p-4">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-secondary">
              <Info className="h-4 w-4 text-muted-foreground" />
            </div>
            <div>
              <p className="text-sm font-medium text-foreground">{t('settings.about.knowledgeBase')}</p>
              <p className="text-xs text-muted-foreground">docs.zentria.tech</p>
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
              <p className="text-xs text-muted-foreground">www.zentria.tech</p>
            </div>
          </div>
        </div>
      </div>

      {/* Interactive Tours */}
      <h3 className="text-sm font-medium text-foreground mb-1">{t('settings.about.tours')}</h3>
      <p className="text-xs text-muted-foreground mb-3">{t('settings.about.toursDesc')}</p>
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
              {t('settings.about.startTour')}
            </Button>
          </div>
        ))}
      </div>

      {/* System info */}
      <h3 className="text-sm font-medium text-foreground mb-3">{t('settings.about.system')}</h3>
      <div className="rounded-lg border border-border bg-card p-4 text-xs text-muted-foreground space-y-1">
        <div className="flex justify-between"><span>{t('settings.about.version')}</span><span className="text-foreground">0.1.0 (Beta)</span></div>
        <div className="flex justify-between"><span>{t('settings.about.license')}</span><span className="text-foreground">Enterprise</span></div>
        <div className="flex justify-between"><span>{t('settings.about.hosting')}</span><span className="text-foreground">EU (Hetzner)</span></div>
        <div className="flex justify-between"><span>{t('settings.about.privacy')}</span><span className="text-foreground">{t('settings.about.gdprCompliant')}</span></div>
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


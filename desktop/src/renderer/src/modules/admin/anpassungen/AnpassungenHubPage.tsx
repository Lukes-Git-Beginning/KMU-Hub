/**
 * AnpassungenTab — Sub-Tab-Inhalt für „Anpassungen" im AdminHubPage.
 *
 * Eingehängt als 9. Tab unter /admin/anpassungen (+ /admin/anpassungen/felder).
 * AdminHubPage liefert den übergeordneten „Verwaltung"-Header + die Top-Tab-Leiste.
 * Diese Komponente beginnt direkt mit der Sub-Tab-Leiste — identisch zu SecurityAdminHubTab.
 *
 * Gated auf `admin:customization:manage` — nur admin + it_admin haben diesen Key.
 * Das Gate wird von AdminHubPage bereits via TAB_CAPABILITY geprüft; hier nur
 * pessimistisches ready-Guard + Sub-Tab-Filterung.
 *
 * Sub-Tabs:
 *   v1.1 Felder (CustomFieldsTab)
 *   v1.2 Begriffe — hier als Kommentar-Slot vorbereitet
 *   v1.3 Wertelisten — hier als Kommentar-Slot vorbereitet
 *
 * URL-Sync: /admin/anpassungen/felder setzt activeSubTab='felder' via ?subtab-Param
 * oder Pathname-Matching (ROUTE_TO_SUBTAB). Weitere Sub-Tabs nach demselben Muster.
 */
import { useEffect, useRef, lazy, Suspense, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useLocation, useNavigate } from 'react-router-dom'
import { ModuleLoadingFallback } from '@/components/layout/ModuleShell'
import { useCapabilitySet } from '@/hooks/useCapability'

const CustomFieldsTab = lazy(() => import('./CustomFieldsTab'))
const BegriffeTab = lazy(() => import('./BegriffeTab'))

type AnpassungenSubTab = 'felder' | 'begriffe'
// v1.3: | 'wertelisten'

/** Pathname → Sub-Tab-Schlüssel (für Deep-Links via URL). */
const ROUTE_TO_SUBTAB: Record<string, AnpassungenSubTab> = {
  '/admin/anpassungen/felder': 'felder',
  '/admin/anpassungen/begriffe': 'begriffe',
}

/** Sub-Tab → Route (für navigate bei Tab-Wechsel). */
const SUBTAB_TO_ROUTE: Record<AnpassungenSubTab, string> = {
  felder: '/admin/anpassungen/felder',
  begriffe: '/admin/anpassungen/begriffe',
}

/** Capability gate per Sub-Tab. undefined = kein zusätzliches Gate. */
const SUBTAB_CAPABILITY: Partial<Record<AnpassungenSubTab, string>> = {
  felder: 'admin:customization:manage',
  begriffe: 'admin:customization:manage',
}

const ALL_SUB_TABS: { key: AnpassungenSubTab; labelKey: string }[] = [
  { key: 'felder', labelKey: 'customization.hub.tabs.felder' },
  { key: 'begriffe', labelKey: 'customization.hub.tabs.begriffe' },
  // v1.3: { key: 'wertelisten', labelKey: 'customization.hub.tabs.wertelisten' },
]

export default function AnpassungenTab() {
  const { t } = useTranslation()
  const { has, ready } = useCapabilitySet()
  const contentRef = useRef<HTMLDivElement>(null)
  const location = useLocation()
  const navigate = useNavigate()

  // Deep-Link-Support: Pathname-Matching für /admin/anpassungen/felder etc.
  const pathnameTab = ROUTE_TO_SUBTAB[location.pathname] ?? null

  // null = noch keine explizite Auswahl → erster erlaubter Tab nach ready.
  const [activeSubTab, setActiveSubTab] = useState<AnpassungenSubTab | null>(pathnameTab)

  const subTabs = ready
    ? ALL_SUB_TABS.filter((tab) => {
        const cap = SUBTAB_CAPABILITY[tab.key]
        return cap ? has(cap) : true
      })
    : []

  const isActiveAllowed = activeSubTab !== null && subTabs.some((tab) => tab.key === activeSubTab)
  const effectiveSubTab: AnpassungenSubTab =
    isActiveAllowed && activeSubTab !== null ? activeSubTab : (subTabs[0]?.key ?? 'felder')

  useEffect(() => {
    contentRef.current?.focus()
  }, [effectiveSubTab])

  const handleSubTabChange = (tab: AnpassungenSubTab) => {
    setActiveSubTab(tab)
    navigate(SUBTAB_TO_ROUTE[tab])
  }

  if (!ready) return <ModuleLoadingFallback />

  return (
    <div className="flex flex-col h-full">
      {/* Sub-Tab-Leiste — direkt beginnend, kein Doppel-Header */}
      <div
        role="tablist"
        aria-label={t('admin.hub.tabs.customization')}
        className="shrink-0 flex gap-0 px-6 pt-4 border-b border-border overflow-x-auto touch-manipulation"
      >
        {subTabs.map((tab) => (
          <button
            key={tab.key}
            id={`anpassungen-subtab-${tab.key}`}
            role="tab"
            aria-selected={effectiveSubTab === tab.key}
            aria-controls={`anpassungen-panel-${tab.key}`}
            onClick={() => handleSubTabChange(tab.key)}
            className={[
              'whitespace-nowrap px-3 py-2 text-sm border-b-2 -mb-px transition-colors',
              effectiveSubTab === tab.key
                ? 'border-primary text-primary font-medium'
                : 'border-transparent text-muted-foreground hover:text-foreground',
            ].join(' ')}
          >
            {t(tab.labelKey)}
          </button>
        ))}
      </div>

      {/* Sub-Tab-Inhalt */}
      <div
        ref={contentRef}
        tabIndex={-1}
        id={`anpassungen-panel-${effectiveSubTab}`}
        role="tabpanel"
        aria-labelledby={`anpassungen-subtab-${effectiveSubTab}`}
        className="flex-1 overflow-y-auto outline-none"
      >
        <Suspense fallback={<ModuleLoadingFallback />}>
          {effectiveSubTab === 'felder' && <CustomFieldsTab />}
          {effectiveSubTab === 'begriffe' && <BegriffeTab />}
        </Suspense>
      </div>
    </div>
  )
}

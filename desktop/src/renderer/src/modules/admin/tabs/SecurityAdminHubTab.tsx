/**
 * SecurityAdminHubTab — Sicherheit & Compliance im Admin-Hub.
 *
 * Sub-Tabs:
 *   Audit-Log | DSGVO | Sessions | IP-Whitelist | Vault | Datenschutz | KI-Governance
 *
 * Audit-Log, Sessions, Vault, IP-Whitelist, DSGVO-Sub-Pages: direkte Imports
 * aus modules/security/ (keine Code-Duplikation).
 * Datenschutz: tenant-seitige Tracking-Toggles, Retention, Cookie-Defaults (Phase 4).
 * KI-Governance: org-weiter Master-Toggle, Modul-Verbote, Activity-Log (Phase 4).
 */
import { useEffect, useRef, lazy, Suspense, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ModuleLoadingFallback } from '@/components/layout/ModuleShell'
import { useCapabilitySet } from '@/hooks/useCapability'
import { PrivacyAdminTab } from './security/PrivacyAdminTab'
import { AIGovernanceAdminTab } from './security/AIGovernanceAdminTab'

const AuditLogPage = lazy(() => import('@/modules/security/AuditLogPage'))
const SessionsPage = lazy(() => import('@/modules/security/SessionsPage'))
const VaultPage = lazy(() => import('@/modules/security/VaultPage'))
const IPAccessPage = lazy(() => import('@/modules/security/IPAccessPage'))
const GDPRExportPage = lazy(() => import('@/modules/security/GDPRExportPage'))
const GDPRErasurePage = lazy(() => import('@/modules/security/GDPRErasurePage'))
const DSARSearchPage = lazy(() => import('@/modules/security/DSARSearchPage'))
const RetentionPolicyPage = lazy(() => import('@/modules/security/RetentionPolicyPage'))
const PasswordPolicyPage = lazy(() => import('@/modules/security/PasswordPolicyPage'))

type SecuritySubTab =
  | 'audit'
  | 'gdpr'
  | 'dsar'
  | 'retention'
  | 'sessions'
  | 'password-policy'
  | 'ip-whitelist'
  | 'vault'
  | 'privacy'
  | 'ai'

/** Capability required per sub-tab. Tabs without an entry are ungated (self-service). */
const SUB_TAB_CAPABILITY: Partial<Record<SecuritySubTab, string>> = {
  audit: 'security:audit:read',
  gdpr: 'security:gdpr:execute',
  dsar: 'security:gdpr:execute',
  retention: 'security:policy:manage',
  'password-policy': 'security:policy:manage',
  'ip-whitelist': 'security:policy:manage',
  vault: 'security:policy:manage',
  privacy: 'security:policy:manage',
  ai: 'admin:ai:manage',
  // sessions: no gate — self-service (own sessions only)
}

const ALL_SUB_TABS: { key: SecuritySubTab; labelKey: string }[] = [
  { key: 'audit', labelKey: 'admin.security.tabs.audit' },
  { key: 'gdpr', labelKey: 'admin.security.tabs.gdpr' },
  { key: 'dsar', labelKey: 'admin.security.tabs.dsar' },
  { key: 'retention', labelKey: 'admin.security.tabs.retention' },
  { key: 'sessions', labelKey: 'admin.security.tabs.sessions' },
  { key: 'password-policy', labelKey: 'admin.security.tabs.passwordPolicy' },
  { key: 'ip-whitelist', labelKey: 'admin.security.tabs.ipWhitelist' },
  { key: 'vault', labelKey: 'admin.security.tabs.vault' },
  { key: 'privacy', labelKey: 'admin.security.tabs.privacy' },
  { key: 'ai', labelKey: 'admin.security.tabs.ai' },
]

export default function SecurityAdminHubTab() {
  const { t } = useTranslation()
  const { has, ready } = useCapabilitySet()
  const contentRef = useRef<HTMLDivElement>(null)

  // null = no explicit user choice yet → first allowed tab once ready.
  // (Initialising with a concrete tab before `ready` would freeze the
  // pessimistic pre-ready list — admin would land on "Sessions".)
  const [activeSubTab, setActiveSubTab] = useState<SecuritySubTab | null>(null)

  const subTabs = ready
    ? ALL_SUB_TABS.filter((tab) => {
        const cap = SUB_TAB_CAPABILITY[tab.key]
        return cap ? has(cap) : true
      })
    : []

  const isActiveAllowed = activeSubTab !== null && subTabs.some((tab) => tab.key === activeSubTab)
  const effectiveSubTab: SecuritySubTab =
    isActiveAllowed && activeSubTab !== null ? activeSubTab : (subTabs[0]?.key ?? 'sessions')

  useEffect(() => {
    contentRef.current?.focus()
  }, [effectiveSubTab])

  if (!ready) return <ModuleLoadingFallback />

  return (
    <div className="flex flex-col h-full">
      {/* Sub-Tab bar */}
      <div
        role="tablist"
        aria-label={t('admin.hub.tabs.security')}
        className="shrink-0 flex gap-0 px-6 pt-4 border-b border-border overflow-x-auto touch-manipulation"
      >
        {subTabs.map((tab) => (
          <button
            key={tab.key}
            id={`security-tab-${tab.key}`}
            role="tab"
            aria-selected={effectiveSubTab === tab.key}
            aria-controls={`security-panel-${tab.key}`}
            onClick={() => setActiveSubTab(tab.key)}
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

      {/* Sub-tab content — tabIndex={-1}: programmatic focus only, outline-none is intentional */}
      <div
        ref={contentRef}
        tabIndex={-1}
        id={`security-panel-${effectiveSubTab}`}
        role="tabpanel"
        aria-labelledby={`security-tab-${effectiveSubTab}`}
        className="flex-1 overflow-y-auto outline-none"
      >
        <Suspense fallback={<ModuleLoadingFallback />}>
          {effectiveSubTab === 'audit' && <AuditLogPage />}
          {effectiveSubTab === 'gdpr' && (
            <div className="px-6 py-6 space-y-6">
              <GDPRExportPage />
              <div className="border-t border-border pt-6">
                <GDPRErasurePage />
              </div>
            </div>
          )}
          {effectiveSubTab === 'dsar' && <DSARSearchPage />}
          {effectiveSubTab === 'retention' && <RetentionPolicyPage />}
          {effectiveSubTab === 'sessions' && <SessionsPage />}
          {effectiveSubTab === 'password-policy' && <PasswordPolicyPage />}
          {effectiveSubTab === 'ip-whitelist' && <IPAccessPage />}
          {effectiveSubTab === 'vault' && <VaultPage />}
          {effectiveSubTab === 'privacy' && <PrivacyAdminTab />}
          {effectiveSubTab === 'ai' && <AIGovernanceAdminTab />}
        </Suspense>
      </div>
    </div>
  )
}

// Platzhalter wurden in Phase 4 durch echte Implementierungen ersetzt:
// - PrivacyAdminTab: security/PrivacyAdminTab.tsx
// - AIGovernanceAdminTab: security/AIGovernanceAdminTab.tsx

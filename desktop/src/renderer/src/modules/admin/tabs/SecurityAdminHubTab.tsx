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

export default function SecurityAdminHubTab() {
  const { t } = useTranslation()
  const [activeSubTab, setActiveSubTab] = useState<SecuritySubTab>('audit')
  const contentRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    contentRef.current?.focus()
  }, [activeSubTab])

  const subTabs: { key: SecuritySubTab; labelKey: string }[] = [
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
            aria-selected={activeSubTab === tab.key}
            aria-controls={`security-panel-${tab.key}`}
            onClick={() => setActiveSubTab(tab.key)}
            className={[
              'whitespace-nowrap px-3 py-2 text-sm border-b-2 -mb-px transition-colors',
              activeSubTab === tab.key
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
        id={`security-panel-${activeSubTab}`}
        role="tabpanel"
        aria-labelledby={`security-tab-${activeSubTab}`}
        className="flex-1 overflow-y-auto outline-none"
      >
        <Suspense fallback={<ModuleLoadingFallback />}>
          {activeSubTab === 'audit' && <AuditLogPage />}
          {activeSubTab === 'gdpr' && (
            <div className="px-6 py-6 space-y-6">
              <GDPRExportPage />
              <div className="border-t border-border pt-6">
                <GDPRErasurePage />
              </div>
            </div>
          )}
          {activeSubTab === 'dsar' && <DSARSearchPage />}
          {activeSubTab === 'retention' && <RetentionPolicyPage />}
          {activeSubTab === 'sessions' && <SessionsPage />}
          {activeSubTab === 'password-policy' && <PasswordPolicyPage />}
          {activeSubTab === 'ip-whitelist' && <IPAccessPage />}
          {activeSubTab === 'vault' && <VaultPage />}
          {activeSubTab === 'privacy' && <PrivacyAdminTab />}
          {activeSubTab === 'ai' && <AIGovernanceAdminTab />}
        </Suspense>
      </div>
    </div>
  )
}

// Platzhalter wurden in Phase 4 durch echte Implementierungen ersetzt:
// - PrivacyAdminTab: security/PrivacyAdminTab.tsx
// - AIGovernanceAdminTab: security/AIGovernanceAdminTab.tsx

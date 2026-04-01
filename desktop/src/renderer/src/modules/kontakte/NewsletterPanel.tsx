/**
 * NewsletterPanel — Integration panel for newsletter services (Brevo/CleverReach).
 *
 * Shows connection status, subscriber lists, send history, sync button.
 * Mock data for design — backend swap: replace with API hooks.
 */
import { useState } from 'react'
import {
  Mail,
  RefreshCw,
  CheckCircle2,
  XCircle,
  ExternalLink,
  Users,
  Send,
  Calendar,
  Zap,
  BarChart3,
} from 'lucide-react'
import { toast } from 'sonner'

// ---------------------------------------------------------------------------
// Mock data
// ---------------------------------------------------------------------------

interface NewsletterService {
  id: string
  name: string
  connected: boolean
  account: string | null
  lastSync: string | null
  subscriberCount: number
  listCount: number
}

interface SendHistoryEntry {
  id: string
  subject: string
  sentAt: string
  recipients: number
  opens: number
  clicks: number
}

const mockServices: NewsletterService[] = [
  {
    id: 'brevo',
    name: 'Brevo (Sendinblue)',
    connected: true,
    account: 'marketing@zentria.tech',
    lastSync: '2026-02-20 14:30',
    subscriberCount: 847,
    listCount: 5,
  },
  {
    id: 'cleverreach',
    name: 'CleverReach',
    connected: false,
    account: null,
    lastSync: null,
    subscriberCount: 0,
    listCount: 0,
  },
]

const mockHistory: SendHistoryEntry[] = [
  { id: 'sh1', subject: 'Cosmi Newsletter - Februar 2026', sentAt: '2026-02-15', recipients: 812, opens: 423, clicks: 89 },
  { id: 'sh2', subject: 'Neues Feature: Kalender-Integration', sentAt: '2026-02-01', recipients: 798, opens: 512, clicks: 145 },
  { id: 'sh3', subject: 'Cosmi Newsletter - Januar 2026', sentAt: '2026-01-15', recipients: 780, opens: 398, clicks: 67 },
  { id: 'sh4', subject: 'Frohe Festtage & Ausblick 2026', sentAt: '2025-12-20', recipients: 756, opens: 534, clicks: 102 },
]

const mockLists = [
  { name: 'Alle Kunden', count: 423, active: true },
  { name: 'Newsletter-Abonnenten', count: 812, active: true },
  { name: 'Interessenten', count: 156, active: true },
  { name: 'Partner', count: 34, active: false },
  { name: 'Ehemalige Kunden', count: 89, active: false },
]

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function NewsletterPanel() {
  const [syncing, setSyncing] = useState(false)
  const [activeTab, setActiveTab] = useState<'services' | 'lists' | 'history'>('services')

  const handleSync = () => {
    setSyncing(true)
    setTimeout(() => {
      setSyncing(false)
      toast.success('Kontakte synchronisiert')
    }, 2000)
  }

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Mail className="h-4 w-4 text-primary" />
          <span className="text-sm font-medium text-foreground">Newsletter-Integration</span>
        </div>
        <button
          onClick={handleSync}
          disabled={syncing}
          className="flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 text-xs text-foreground hover:bg-accent transition-colors disabled:opacity-50"
        >
          <RefreshCw className={`h-3 w-3 ${syncing ? 'animate-spin' : ''}`} />
          {syncing ? 'Synchronisiere...' : 'Kontakte sync'}
        </button>
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-1 border-b border-border">
        {[
          { key: 'services' as const, label: 'Dienste' },
          { key: 'lists' as const, label: 'Listen' },
          { key: 'history' as const, label: 'Verlauf' },
        ].map((tab) => (
          <button
            key={tab.key}
            onClick={() => setActiveTab(tab.key)}
            className={`border-b-2 px-3 pb-2 text-xs transition-colors ${
              activeTab === tab.key
                ? 'border-primary text-primary font-medium'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Services tab */}
      {activeTab === 'services' && (
        <div className="space-y-2">
          {mockServices.map((svc) => (
            <div
              key={svc.id}
              className={`rounded-lg border p-3 ${
                svc.connected ? 'border-success/30 bg-success-light/10' : 'border-border'
              }`}
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  {svc.connected ? (
                    <CheckCircle2 className="h-4 w-4 text-success" />
                  ) : (
                    <XCircle className="h-4 w-4 text-muted-foreground" />
                  )}
                  <div>
                    <p className="text-sm font-medium text-foreground">{svc.name}</p>
                    {svc.account && (
                      <p className="text-[11px] text-muted-foreground">{svc.account}</p>
                    )}
                  </div>
                </div>
                <button
                  className={`rounded-md px-3 py-1.5 text-xs font-medium transition-colors ${
                    svc.connected
                      ? 'border border-border text-foreground hover:bg-accent'
                      : 'bg-primary text-primary-foreground hover:bg-primary/90'
                  }`}
                >
                  {svc.connected ? 'Einstellungen' : 'Verbinden'}
                </button>
              </div>

              {svc.connected && (
                <div className="mt-2 grid grid-cols-3 gap-3 pl-6">
                  <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                    <Users className="h-3 w-3" />
                    {svc.subscriberCount} Abonnenten
                  </div>
                  <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                    <Zap className="h-3 w-3" />
                    {svc.listCount} Listen
                  </div>
                  <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                    <Calendar className="h-3 w-3" />
                    {svc.lastSync}
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Lists tab */}
      {activeTab === 'lists' && (
        <div className="space-y-1.5">
          {mockLists.map((list) => (
            <div
              key={list.name}
              className="flex items-center justify-between rounded-md border border-border px-3 py-2"
            >
              <div className="flex items-center gap-2">
                <div className={`h-2 w-2 rounded-full ${list.active ? 'bg-success' : 'bg-muted-foreground/30'}`} />
                <span className="text-sm text-foreground">{list.name}</span>
              </div>
              <div className="flex items-center gap-3">
                <span className="text-xs text-muted-foreground">{list.count} Kontakte</span>
                <button className="rounded-md p-1 text-muted-foreground hover:text-primary transition-colors">
                  <ExternalLink className="h-3 w-3" />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* History tab */}
      {activeTab === 'history' && (
        <div className="space-y-2">
          {mockHistory.map((entry) => (
            <div key={entry.id} className="rounded-md border border-border p-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Send className="h-3.5 w-3.5 text-primary" />
                  <span className="text-sm font-medium text-foreground">{entry.subject}</span>
                </div>
                <span className="text-[10px] text-muted-foreground">{entry.sentAt}</span>
              </div>
              <div className="mt-1.5 flex items-center gap-4 pl-5">
                <span className="flex items-center gap-1 text-[10px] text-muted-foreground">
                  <Users className="h-2.5 w-2.5" />
                  {entry.recipients} Empfaenger
                </span>
                <span className="flex items-center gap-1 text-[10px] text-muted-foreground">
                  <Mail className="h-2.5 w-2.5" />
                  {entry.opens} Öffnungen ({Math.round((entry.opens / entry.recipients) * 100)}%)
                </span>
                <span className="flex items-center gap-1 text-[10px] text-muted-foreground">
                  <BarChart3 className="h-2.5 w-2.5" />
                  {entry.clicks} Klicks ({Math.round((entry.clicks / entry.recipients) * 100)}%)
                </span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

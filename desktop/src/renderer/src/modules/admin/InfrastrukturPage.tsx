/**
 * Server Infrastructure Admin Panel.
 *
 * Gives admin / IT-support users an overview of the self-hosted
 * server environment: health metrics, services, backups, storage,
 * security, updates, and system logs.
 *
 * All data is mock — the real backend integration comes in Luke's phases.
 */
import { useState } from 'react'
import {
  Server,
  HardDrive,
  Shield,
  Clock,
  Database,
  Wifi,
  RefreshCw,
  Download,
  Play,
  Pause,
  CheckCircle2,
  AlertTriangle,
  XCircle,
  ChevronDown,
  ChevronRight,
  FileText,
  Lock,
  Globe,
  Cpu,
  MemoryStick,
  ArrowUpCircle,
  Trash2,
  RotateCcw,
  Settings,
  Activity,
  Network,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { toast } from 'sonner'
import { ConfirmDialog, PageHeader } from '@/components/shared'

// ---- Types ----

interface ServiceStatus {
  id: string
  name: string
  status: 'running' | 'stopped' | 'warning'
  uptime: string
  memory: string
  cpu: string
}

interface Backup {
  id: string
  date: string
  time: string
  size: string
  type: 'automatic' | 'manual'
  status: 'success' | 'failed' | 'in-progress'
}

interface LogEntry {
  id: string
  timestamp: string
  level: 'info' | 'warning' | 'error'
  service: string
  message: string
}

// ---- Mock Data ----

const MOCK_SERVICES: ServiceStatus[] = [
  { id: 'api', name: 'API Gateway', status: 'running', uptime: '14d 6h 23m', memory: '256 MB', cpu: '3%' },
  { id: 'crm', name: 'CRM Service', status: 'running', uptime: '14d 6h 23m', memory: '384 MB', cpu: '5%' },
  { id: 'chat', name: 'Chat Service', status: 'running', uptime: '14d 6h 22m', memory: '192 MB', cpu: '2%' },
  { id: 'auth', name: 'Auth Service', status: 'running', uptime: '14d 6h 23m', memory: '128 MB', cpu: '1%' },
  { id: 'mail', name: 'Mail Service', status: 'running', uptime: '7d 12h 45m', memory: '96 MB', cpu: '1%' },
  { id: 'db', name: 'PostgreSQL', status: 'running', uptime: '14d 6h 23m', memory: '512 MB', cpu: '8%' },
  { id: 'redis', name: 'Redis Cache', status: 'running', uptime: '14d 6h 23m', memory: '64 MB', cpu: '<1%' },
  { id: 'livekit', name: 'LiveKit (Video)', status: 'warning', uptime: '2d 1h 15m', memory: '320 MB', cpu: '12%' },
]

const MOCK_BACKUPS: Backup[] = [
  { id: 'b1', date: '09.02.2026', time: '03:00', size: '2.4 GB', type: 'automatic', status: 'success' },
  { id: 'b2', date: '08.02.2026', time: '03:00', size: '2.3 GB', type: 'automatic', status: 'success' },
  { id: 'b3', date: '07.02.2026', time: '15:30', size: '2.3 GB', type: 'manual', status: 'success' },
  { id: 'b4', date: '07.02.2026', time: '03:00', size: '2.3 GB', type: 'automatic', status: 'success' },
  { id: 'b5', date: '06.02.2026', time: '03:00', size: '2.2 GB', type: 'automatic', status: 'failed' },
  { id: 'b6', date: '05.02.2026', time: '03:00', size: '2.2 GB', type: 'automatic', status: 'success' },
]

const MOCK_LOGS: LogEntry[] = [
  { id: 'l1', timestamp: '09.02.2026 14:23:01', level: 'info', service: 'API Gateway', message: 'Health check passed — all services responsive' },
  { id: 'l2', timestamp: '09.02.2026 14:15:44', level: 'warning', service: 'LiveKit', message: 'High memory usage detected (320 MB / 512 MB limit)' },
  { id: 'l3', timestamp: '09.02.2026 13:02:11', level: 'info', service: 'Auth Service', message: 'Token cleanup completed — 142 expired tokens removed' },
  { id: 'l4', timestamp: '09.02.2026 12:30:00', level: 'info', service: 'Backup', message: 'Automated daily backup completed successfully (2.4 GB)' },
  { id: 'l5', timestamp: '09.02.2026 09:45:22', level: 'info', service: 'CRM Service', message: 'Database connection pool resized: 10 → 15 connections' },
  { id: 'l6', timestamp: '08.02.2026 22:14:08', level: 'error', service: 'Mail Service', message: 'SMTP relay timeout — retrying in 30s' },
  { id: 'l7', timestamp: '08.02.2026 22:14:38', level: 'info', service: 'Mail Service', message: 'SMTP relay reconnected successfully' },
  { id: 'l8', timestamp: '08.02.2026 15:00:00', level: 'info', service: 'System', message: 'SSL certificate renewal check — valid until 15.06.2026' },
]

type Tab = 'overview' | 'services' | 'backups' | 'storage' | 'security' | 'updates' | 'logs'

// ---- Page Component ----

export default function InfrastrukturPage() {
  const [activeTab, setActiveTab] = useState<Tab>('overview')
  const [showRestoreConfirm, setShowRestoreConfirm] = useState<string | null>(null)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState<string | null>(null)
  const [showRestartConfirm, setShowRestartConfirm] = useState<string | null>(null)

  const tabs: { id: Tab; label: string; icon: typeof Server }[] = [
    { id: 'overview', label: 'Übersicht', icon: Activity },
    { id: 'services', label: 'Dienste', icon: Server },
    { id: 'backups', label: 'Backups', icon: Database },
    { id: 'storage', label: 'Speicher', icon: HardDrive },
    { id: 'security', label: 'Sicherheit', icon: Shield },
    { id: 'updates', label: 'Updates', icon: ArrowUpCircle },
    { id: 'logs', label: 'System-Logs', icon: FileText },
  ]

  return (
    <div className="h-full flex flex-col overflow-hidden">
      {/* Header */}
      <div className="shrink-0 border-b border-border bg-card px-6 py-4">
        <PageHeader
          title="Infrastruktur"
          description="Server-Verwaltung und Systemübersicht"
          icon={Network}
          moduleId="infrastructure"
          actions={
            <div className="flex items-center gap-2">
              <div className="flex items-center gap-2 rounded-full bg-success/10 px-3 py-1.5">
                <div className="h-2 w-2 rounded-full bg-success animate-pulse" />
                <span className="text-xs font-medium text-success">System Online</span>
              </div>
              <Button variant="outline" size="sm" onClick={() => toast.success('System-Status aktualisiert')}>
                <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
                Aktualisieren
              </Button>
            </div>
          }
        />
      </div>

      {/* Tab bar */}
      <div className="shrink-0 border-b border-border bg-card px-6">
        <div className="flex gap-1 overflow-x-auto">
          {tabs.map((tab) => {
            const Icon = tab.icon
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`flex items-center gap-1.5 whitespace-nowrap px-3 py-2.5 text-sm font-medium transition-colors border-b-2 -mb-px ${
                  activeTab === tab.id
                    ? 'border-primary text-primary tab-accent-active'
                    : 'border-transparent text-muted-foreground hover:text-foreground'
                }`}
              >
                <Icon className="h-4 w-4" />
                {tab.label}
              </button>
            )
          })}
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-6">
        {activeTab === 'overview' && <OverviewTab />}
        {activeTab === 'services' && (
          <ServicesTab
            onRestart={(id) => setShowRestartConfirm(id)}
          />
        )}
        {activeTab === 'backups' && (
          <BackupsTab
            onRestore={(id) => setShowRestoreConfirm(id)}
            onDelete={(id) => setShowDeleteConfirm(id)}
          />
        )}
        {activeTab === 'storage' && <StorageTab />}
        {activeTab === 'security' && <SecurityTab />}
        {activeTab === 'updates' && <UpdatesTab />}
        {activeTab === 'logs' && <LogsTab />}
      </div>

      {/* Confirm dialogs */}
      <ConfirmDialog
        open={showRestoreConfirm !== null}
        onOpenChange={(open) => { if (!open) setShowRestoreConfirm(null) }}
        title="Backup wiederherstellen?"
        description="Das aktuelle System wird mit dem ausgewählten Backup überschrieben. Dieser Vorgang kann nicht rückgängig gemacht werden."
        confirmLabel="Wiederherstellen"
        variant="destructive"
        onConfirm={() => {
          toast.success('Wiederherstellung gestartet — dies kann einige Minuten dauern...')
          setShowRestoreConfirm(null)
        }}
      />

      <ConfirmDialog
        open={showDeleteConfirm !== null}
        onOpenChange={(open) => { if (!open) setShowDeleteConfirm(null) }}
        title="Backup löschen?"
        description="Das ausgewählte Backup wird unwiderruflich gelöscht."
        confirmLabel="Löschen"
        variant="destructive"
        onConfirm={() => {
          toast.success('Backup gelöscht')
          setShowDeleteConfirm(null)
        }}
      />

      <ConfirmDialog
        open={showRestartConfirm !== null}
        onOpenChange={(open) => { if (!open) setShowRestartConfirm(null) }}
        title="Dienst neustarten?"
        description="Der Dienst wird kurzzeitig nicht verfügbar sein. Laufende Verbindungen werden unterbrochen."
        confirmLabel="Neustarten"
        onConfirm={() => {
          toast.success('Dienst wird neugestartet...')
          setShowRestartConfirm(null)
        }}
      />
    </div>
  )
}

// ============================================================
// Overview Tab — Health gauges, quick stats, recent events
// ============================================================
function OverviewTab() {
  const runningServices = MOCK_SERVICES.filter((s) => s.status === 'running').length
  const warningServices = MOCK_SERVICES.filter((s) => s.status === 'warning').length

  const stats = [
    { label: 'Betriebszeit', value: '14 Tage 6 Std', icon: Clock, color: 'text-primary' },
    { label: 'Dienste aktiv', value: `${runningServices}/${MOCK_SERVICES.length}`, icon: Server, color: 'text-success' },
    { label: 'Warnungen', value: String(warningServices), icon: AlertTriangle, color: warningServices > 0 ? 'text-warning' : 'text-success' },
    { label: 'Letztes Backup', value: 'Heute 03:00', icon: Database, color: 'text-primary' },
  ]

  return (
    <div className="space-y-6">
      {/* Quick stats */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        {stats.map((stat) => {
          const Icon = stat.icon
          return (
            <div key={stat.label} className="rounded-xl border border-border bg-card p-4">
              <div className="flex items-center gap-2 mb-2">
                <Icon className={`h-4 w-4 ${stat.color}`} />
                <span className="text-xs text-muted-foreground">{stat.label}</span>
              </div>
              <p className="text-lg font-semibold text-foreground">{stat.value}</p>
            </div>
          )
        })}
      </div>

      {/* Resource usage */}
      <div className="rounded-xl border border-border bg-card p-5">
        <h3 className="text-sm font-medium text-foreground mb-4">Ressourcen-Auslastung</h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <GaugeBar label="CPU" value={32} icon={Cpu} unit="%" />
          <GaugeBar label="Arbeitsspeicher" value={58} icon={MemoryStick} unit="%" detail="4.6 / 8 GB" />
          <GaugeBar label="Speicherplatz" value={41} icon={HardDrive} unit="%" detail="82 / 200 GB" />
        </div>
      </div>

      {/* Service overview */}
      <div className="rounded-xl border border-border bg-card p-5">
        <h3 className="text-sm font-medium text-foreground mb-3">Dienste-Status</h3>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          {MOCK_SERVICES.map((svc) => (
            <div key={svc.id} className="flex items-center gap-2 rounded-lg border border-border-muted p-3">
              <StatusDot status={svc.status} />
              <div className="min-w-0">
                <p className="text-sm text-foreground truncate">{svc.name}</p>
                <p className="text-[10px] text-muted-foreground">{svc.status === 'running' ? svc.uptime : svc.status === 'warning' ? 'Warnung' : 'Gestoppt'}</p>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Recent logs */}
      <div className="rounded-xl border border-border bg-card p-5">
        <h3 className="text-sm font-medium text-foreground mb-3">Letzte Ereignisse</h3>
        <div className="space-y-2">
          {MOCK_LOGS.slice(0, 5).map((log) => (
            <LogRow key={log.id} log={log} />
          ))}
        </div>
      </div>
    </div>
  )
}

// ============================================================
// Services Tab — Detailed service management
// ============================================================
function ServicesTab({ onRestart }: { onRestart: (id: string) => void }) {
  const [expandedService, setExpandedService] = useState<string | null>(null)

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-base font-medium text-foreground">Dienste-Verwaltung</h2>
          <p className="text-sm text-muted-foreground">Starte, stoppe und überwache einzelne Dienste</p>
        </div>
        <Button variant="outline" size="sm" onClick={() => toast.success('Alle Dienste werden neugestartet...')}>
          <RotateCcw className="mr-1.5 h-3.5 w-3.5" />
          Alle neustarten
        </Button>
      </div>

      <div className="space-y-2">
        {MOCK_SERVICES.map((svc) => {
          const isExpanded = expandedService === svc.id
          return (
            <div key={svc.id} className="rounded-xl border border-border bg-card overflow-hidden">
              <button
                onClick={() => setExpandedService(isExpanded ? null : svc.id)}
                className="flex w-full items-center gap-3 p-4 text-left hover:bg-secondary/30 transition-colors"
              >
                <StatusDot status={svc.status} />
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-foreground">{svc.name}</p>
                  <p className="text-xs text-muted-foreground">Betriebszeit: {svc.uptime}</p>
                </div>
                <div className="hidden md:flex items-center gap-4 text-xs text-muted-foreground">
                  <span>CPU: {svc.cpu}</span>
                  <span>RAM: {svc.memory}</span>
                </div>
                {isExpanded ? (
                  <ChevronDown className="h-4 w-4 text-muted-foreground" />
                ) : (
                  <ChevronRight className="h-4 w-4 text-muted-foreground" />
                )}
              </button>

              {isExpanded && (
                <div className="border-t border-border bg-secondary/10 px-4 py-3">
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-4">
                    <div>
                      <p className="text-[10px] text-muted-foreground uppercase tracking-wider">Status</p>
                      <p className="text-sm text-foreground capitalize">{svc.status === 'running' ? 'Aktiv' : svc.status === 'warning' ? 'Warnung' : 'Gestoppt'}</p>
                    </div>
                    <div>
                      <p className="text-[10px] text-muted-foreground uppercase tracking-wider">Betriebszeit</p>
                      <p className="text-sm text-foreground">{svc.uptime}</p>
                    </div>
                    <div>
                      <p className="text-[10px] text-muted-foreground uppercase tracking-wider">CPU</p>
                      <p className="text-sm text-foreground">{svc.cpu}</p>
                    </div>
                    <div>
                      <p className="text-[10px] text-muted-foreground uppercase tracking-wider">RAM</p>
                      <p className="text-sm text-foreground">{svc.memory}</p>
                    </div>
                  </div>
                  <div className="flex gap-2">
                    <Button size="sm" variant="outline" onClick={() => onRestart(svc.id)}>
                      <RotateCcw className="mr-1.5 h-3.5 w-3.5" />
                      Neustarten
                    </Button>
                    {svc.status === 'running' ? (
                      <Button size="sm" variant="outline" onClick={() => toast.info(`${svc.name} wird gestoppt...`)}>
                        <Pause className="mr-1.5 h-3.5 w-3.5" />
                        Stoppen
                      </Button>
                    ) : (
                      <Button size="sm" variant="outline" onClick={() => toast.success(`${svc.name} wird gestartet...`)}>
                        <Play className="mr-1.5 h-3.5 w-3.5" />
                        Starten
                      </Button>
                    )}
                    <Button size="sm" variant="outline" onClick={() => toast.info(`Logs für ${svc.name} werden geladen...`)}>
                      <FileText className="mr-1.5 h-3.5 w-3.5" />
                      Logs
                    </Button>
                  </div>
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}

// ============================================================
// Backups Tab — Schedule, history, restore
// ============================================================
function BackupsTab({
  onRestore,
  onDelete,
}: {
  onRestore: (id: string) => void
  onDelete: (id: string) => void
}) {
  const [autoBackup, setAutoBackup] = useState(true)

  return (
    <div className="space-y-6">
      {/* Backup config */}
      <div className="rounded-xl border border-border bg-card p-5">
        <h3 className="text-sm font-medium text-foreground mb-4">Backup-Konfiguration</h3>
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-foreground">Automatisches Backup</p>
              <p className="text-xs text-muted-foreground">Täglich um 03:00 Uhr</p>
            </div>
            <Switch checked={autoBackup} onCheckedChange={setAutoBackup} />
          </div>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-foreground">Aufbewahrung</p>
              <p className="text-xs text-muted-foreground">Backups werden 30 Tage aufbewahrt</p>
            </div>
            <span className="text-sm text-muted-foreground">30 Tage</span>
          </div>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-foreground">Speicherort</p>
              <p className="text-xs text-muted-foreground">/var/backups/cosmi/</p>
            </div>
            <Button variant="outline" size="sm" onClick={() => toast.info('Speicherort-Konfiguration wird geöffnet...')}>
              <Settings className="mr-1.5 h-3.5 w-3.5" />
              Ändern
            </Button>
          </div>
        </div>
      </div>

      {/* Manual backup button */}
      <div className="flex items-center gap-3">
        <Button onClick={() => toast.success('Manuelles Backup wird erstellt — dies kann einige Minuten dauern...')}>
          <Database className="mr-1.5 h-4 w-4" />
          Backup jetzt erstellen
        </Button>
        <Button variant="outline" onClick={() => toast.info('Backup wird heruntergeladen...')}>
          <Download className="mr-1.5 h-4 w-4" />
          Letztes Backup herunterladen
        </Button>
      </div>

      {/* Backup history */}
      <div className="rounded-xl border border-border bg-card overflow-hidden">
        <div className="px-5 py-3 border-b border-border bg-secondary/30">
          <h3 className="text-sm font-medium text-foreground">Backup-Verlauf</h3>
        </div>
        <div className="divide-y divide-border-muted">
          {MOCK_BACKUPS.map((backup) => (
            <div key={backup.id} className="flex items-center gap-3 px-5 py-3">
              <BackupStatusIcon status={backup.status} />
              <div className="flex-1 min-w-0">
                <p className="text-sm text-foreground">
                  {backup.date} um {backup.time}
                </p>
                <p className="text-xs text-muted-foreground">
                  {backup.size} &middot; {backup.type === 'automatic' ? 'Automatisch' : 'Manuell'}
                </p>
              </div>
              <div className="flex items-center gap-1">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => onRestore(backup.id)}
                  disabled={backup.status === 'failed'}
                  title="Wiederherstellen"
                >
                  <RotateCcw className="h-3.5 w-3.5" />
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => toast.info('Backup wird heruntergeladen...')}
                  disabled={backup.status === 'failed'}
                  title="Herunterladen"
                >
                  <Download className="h-3.5 w-3.5" />
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => onDelete(backup.id)}
                  title="Löschen"
                  className="text-muted-foreground hover:text-error"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

// ============================================================
// Storage Tab — Disk usage breakdown
// ============================================================
function StorageTab() {
  const storageItems = [
    { label: 'Datenbank', used: '34.2 GB', percent: 42, color: 'bg-primary' },
    { label: 'Dokumente & Dateien', used: '28.7 GB', percent: 35, color: 'bg-blue-500' },
    { label: 'E-Mail-Archiv', used: '12.1 GB', percent: 15, color: 'bg-amber-500' },
    { label: 'Backups', used: '4.8 GB', percent: 6, color: 'bg-purple-500' },
    { label: 'System & Logs', used: '2.2 GB', percent: 2, color: 'bg-gray-400' },
  ]

  return (
    <div className="space-y-6">
      {/* Overview */}
      <div className="rounded-xl border border-border bg-card p-5">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-sm font-medium text-foreground">Speicherplatz</h3>
            <p className="text-xs text-muted-foreground">82.0 GB von 200 GB verwendet (41%)</p>
          </div>
          <Button variant="outline" size="sm" onClick={() => toast.info('Speicher-Analyse wird durchgefuehrt...')}>
            Analyse starten
          </Button>
        </div>

        {/* Full bar */}
        <div className="h-6 rounded-full bg-secondary overflow-hidden flex mb-6">
          {storageItems.map((item) => (
            <div
              key={item.label}
              className={`${item.color} transition-all`}
              style={{ width: `${item.percent}%` }}
              title={`${item.label}: ${item.used}`}
            />
          ))}
        </div>

        {/* Breakdown */}
        <div className="space-y-3">
          {storageItems.map((item) => (
            <div key={item.label} className="flex items-center gap-3">
              <div className={`h-3 w-3 rounded-full ${item.color} shrink-0`} />
              <span className="text-sm text-foreground flex-1">{item.label}</span>
              <span className="text-sm text-muted-foreground">{item.used}</span>
              <span className="text-xs text-muted-foreground w-10 text-right">{item.percent}%</span>
            </div>
          ))}
        </div>
      </div>

      {/* Database info */}
      <div className="rounded-xl border border-border bg-card p-5">
        <h3 className="text-sm font-medium text-foreground mb-4">Datenbank-Details</h3>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {[
            { label: 'Kontakte', value: '2,847' },
            { label: 'Dokumente', value: '15,234' },
            { label: 'E-Mails', value: '48,912' },
            { label: 'Chat-Nachrichten', value: '124,567' },
          ].map((item) => (
            <div key={item.label} className="text-center p-3 rounded-lg bg-secondary/30">
              <p className="text-lg font-semibold text-foreground">{item.value}</p>
              <p className="text-xs text-muted-foreground">{item.label}</p>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

// ============================================================
// Security Tab — SSL, firewall, encryption
// ============================================================
function SecurityTab() {
  const [firewallEnabled, setFirewallEnabled] = useState(true)
  const [encryptionEnabled, setEncryptionEnabled] = useState(true)
  const [bruteForceProtection, setBruteForceProtection] = useState(true)

  return (
    <div className="space-y-6">
      {/* SSL Certificate */}
      <div className="rounded-xl border border-border bg-card p-5">
        <div className="flex items-center gap-3 mb-4">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-success/10">
            <Lock className="h-5 w-5 text-success" />
          </div>
          <div>
            <h3 className="text-sm font-medium text-foreground">SSL/TLS-Zertifikat</h3>
            <p className="text-xs text-success font-medium">Gültig bis 15.06.2026</p>
          </div>
        </div>
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <p className="text-xs text-muted-foreground">Aussteller</p>
            <p className="text-foreground">Let's Encrypt</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Domain</p>
            <p className="text-foreground">crm.firma.ch</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Typ</p>
            <p className="text-foreground">TLS 1.3</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Auto-Erneuerung</p>
            <p className="text-success">Aktiv</p>
          </div>
        </div>
        <Button variant="outline" size="sm" className="mt-4" onClick={() => toast.success('Zertifikat-Erneuerung angestossen...')}>
          <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
          Jetzt erneuern
        </Button>
      </div>

      {/* Security toggles */}
      <div className="rounded-xl border border-border bg-card p-5 space-y-4">
        <h3 className="text-sm font-medium text-foreground">Sicherheits-Einstellungen</h3>

        <div className="flex items-center justify-between py-2">
          <div>
            <p className="text-sm text-foreground">Firewall</p>
            <p className="text-xs text-muted-foreground">Nur erlaubte Ports und IPs zulassen</p>
          </div>
          <Switch checked={firewallEnabled} onCheckedChange={(v) => { setFirewallEnabled(v); toast.success(v ? 'Firewall aktiviert' : 'Firewall deaktiviert') }} />
        </div>

        <div className="flex items-center justify-between py-2">
          <div>
            <p className="text-sm text-foreground">Verschlüsselung (at rest)</p>
            <p className="text-xs text-muted-foreground">Datenbank und Dateien verschlüsselt speichern</p>
          </div>
          <Switch checked={encryptionEnabled} onCheckedChange={(v) => { setEncryptionEnabled(v); toast.success(v ? 'Verschlüsselung aktiviert' : 'Verschlüsselung deaktiviert') }} />
        </div>

        <div className="flex items-center justify-between py-2">
          <div>
            <p className="text-sm text-foreground">Brute-Force-Schutz</p>
            <p className="text-xs text-muted-foreground">Konto nach 5 fehlgeschlagenen Versuchen sperren</p>
          </div>
          <Switch checked={bruteForceProtection} onCheckedChange={(v) => { setBruteForceProtection(v); toast.success(v ? 'Brute-Force-Schutz aktiviert' : 'Brute-Force-Schutz deaktiviert') }} />
        </div>
      </div>

      {/* Network info */}
      <div className="rounded-xl border border-border bg-card p-5">
        <h3 className="text-sm font-medium text-foreground mb-3">Netzwerk</h3>
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div className="flex items-center gap-2">
            <Globe className="h-4 w-4 text-muted-foreground" />
            <div>
              <p className="text-xs text-muted-foreground">Externe IP</p>
              <p className="text-foreground font-mono text-xs">185.212.71.42</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Wifi className="h-4 w-4 text-muted-foreground" />
            <div>
              <p className="text-xs text-muted-foreground">Interne IP</p>
              <p className="text-foreground font-mono text-xs">10.0.1.10</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Server className="h-4 w-4 text-muted-foreground" />
            <div>
              <p className="text-xs text-muted-foreground">Standort</p>
              <p className="text-foreground">Hetzner, Falkenstein (DE)</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Shield className="h-4 w-4 text-muted-foreground" />
            <div>
              <p className="text-xs text-muted-foreground">DDoS-Schutz</p>
              <p className="text-success">Aktiv</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

// ============================================================
// Updates Tab — Version management
// ============================================================
function UpdatesTab() {
  return (
    <div className="space-y-6">
      {/* Current version */}
      <div className="rounded-xl border border-border bg-card p-5">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-sm font-medium text-foreground">Aktuelle Version</h3>
            <p className="text-2xl font-bold text-foreground mt-1">v0.1.0</p>
            <p className="text-xs text-muted-foreground">Installiert am 01.02.2026</p>
          </div>
          <div className="flex h-14 w-14 items-center justify-center rounded-xl bg-success/10">
            <CheckCircle2 className="h-7 w-7 text-success" />
          </div>
        </div>
        <div className="rounded-lg bg-success/5 border border-success/20 p-3">
          <p className="text-sm text-success font-medium">System ist auf dem neuesten Stand</p>
          <p className="text-xs text-muted-foreground mt-0.5">Letzte Prüfung: Heute um 06:00</p>
        </div>
      </div>

      {/* Update settings */}
      <div className="rounded-xl border border-border bg-card p-5 space-y-4">
        <h3 className="text-sm font-medium text-foreground">Update-Einstellungen</h3>

        <div className="flex items-center justify-between py-2">
          <div>
            <p className="text-sm text-foreground">Automatische Updates</p>
            <p className="text-xs text-muted-foreground">Sicherheits-Patches automatisch installieren</p>
          </div>
          <Switch defaultChecked />
        </div>

        <div className="flex items-center justify-between py-2">
          <div>
            <p className="text-sm text-foreground">Update-Benachrichtigungen</p>
            <p className="text-xs text-muted-foreground">Per E-Mail über neue Versionen informieren</p>
          </div>
          <Switch defaultChecked />
        </div>

        <div className="flex items-center justify-between py-2">
          <div>
            <p className="text-sm text-foreground">Beta-Versionen</p>
            <p className="text-xs text-muted-foreground">Vorab-Versionen zum Testen erhalten</p>
          </div>
          <Switch />
        </div>
      </div>

      {/* Update check */}
      <Button onClick={() => toast.info('Nach Updates wird gesucht...')}>
        <RefreshCw className="mr-1.5 h-4 w-4" />
        Jetzt nach Updates suchen
      </Button>

      {/* Changelog */}
      <div className="rounded-xl border border-border bg-card p-5">
        <h3 className="text-sm font-medium text-foreground mb-3">Änderungsprotokoll</h3>
        <div className="space-y-4">
          {[
            { version: 'v0.1.0', date: '01.02.2026', changes: ['Erstinstallation', 'CRM, Chat, Projekte, Auth', 'Desktop-App (Electron)', 'Self-Hosted Deployment'] },
          ].map((release) => (
            <div key={release.version}>
              <div className="flex items-center gap-2 mb-1">
                <span className="text-sm font-medium text-foreground">{release.version}</span>
                <span className="text-xs text-muted-foreground">{release.date}</span>
              </div>
              <ul className="space-y-0.5 ml-4">
                {release.changes.map((change) => (
                  <li key={change} className="text-sm text-muted-foreground list-disc">{change}</li>
                ))}
              </ul>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

// ============================================================
// Logs Tab — Filterable system log
// ============================================================
function LogsTab() {
  const [filter, setFilter] = useState<'all' | 'info' | 'warning' | 'error'>('all')

  const filteredLogs = filter === 'all'
    ? MOCK_LOGS
    : MOCK_LOGS.filter((l) => l.level === filter)

  const filterOptions: { id: typeof filter; label: string; count: number }[] = [
    { id: 'all', label: 'Alle', count: MOCK_LOGS.length },
    { id: 'info', label: 'Info', count: MOCK_LOGS.filter((l) => l.level === 'info').length },
    { id: 'warning', label: 'Warnungen', count: MOCK_LOGS.filter((l) => l.level === 'warning').length },
    { id: 'error', label: 'Fehler', count: MOCK_LOGS.filter((l) => l.level === 'error').length },
  ]

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-base font-medium text-foreground">System-Logs</h2>
          <p className="text-sm text-muted-foreground">Ereignisse und Fehlermeldungen aller Dienste</p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => toast.info('Logs werden exportiert...')}>
            <Download className="mr-1.5 h-3.5 w-3.5" />
            Exportieren
          </Button>
          <Button variant="outline" size="sm" onClick={() => toast.success('Logs aktualisiert')}>
            <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
          </Button>
        </div>
      </div>

      {/* Filter chips */}
      <div className="flex gap-2">
        {filterOptions.map((opt) => (
          <button
            key={opt.id}
            onClick={() => setFilter(opt.id)}
            className={`rounded-full px-3 py-1.5 text-xs font-medium transition-colors ${
              filter === opt.id
                ? 'bg-primary text-primary-foreground'
                : 'bg-secondary text-muted-foreground hover:bg-secondary/80'
            }`}
          >
            {opt.label} ({opt.count})
          </button>
        ))}
      </div>

      {/* Log list */}
      <div className="rounded-xl border border-border bg-card overflow-hidden">
        <div className="divide-y divide-border-muted">
          {filteredLogs.map((log) => (
            <LogRow key={log.id} log={log} showFull />
          ))}
          {filteredLogs.length === 0 && (
            <div className="p-8 text-center text-sm text-muted-foreground">
              Keine Logs für diesen Filter gefunden.
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

// ============================================================
// Shared Components
// ============================================================

function GaugeBar({ label, value, icon: Icon, unit, detail }: {
  label: string
  value: number
  icon: typeof Cpu
  unit: string
  detail?: string
}) {
  const color = value > 80 ? 'bg-error' : value > 60 ? 'bg-warning' : 'bg-primary'

  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-1.5">
          <Icon className="h-4 w-4 text-muted-foreground" />
          <span className="text-sm text-foreground">{label}</span>
        </div>
        <span className="text-sm font-medium text-foreground">{value}{unit}</span>
      </div>
      <div className="h-2.5 rounded-full bg-secondary overflow-hidden">
        <div
          className={`h-full rounded-full ${color} transition-all`}
          style={{ width: `${value}%` }}
        />
      </div>
      {detail && <p className="text-xs text-muted-foreground mt-1">{detail}</p>}
    </div>
  )
}

function StatusDot({ status }: { status: 'running' | 'stopped' | 'warning' }) {
  const color = status === 'running'
    ? 'bg-success'
    : status === 'warning'
      ? 'bg-warning'
      : 'bg-error'

  return (
    <div className={`h-2.5 w-2.5 rounded-full ${color} shrink-0`} />
  )
}

function BackupStatusIcon({ status }: { status: 'success' | 'failed' | 'in-progress' }) {
  if (status === 'success') return <CheckCircle2 className="h-5 w-5 text-success shrink-0" />
  if (status === 'failed') return <XCircle className="h-5 w-5 text-error shrink-0" />
  return <RefreshCw className="h-5 w-5 text-primary animate-spin shrink-0" />
}

function LogRow({ log, showFull }: { log: LogEntry; showFull?: boolean }) {
  const levelColor = log.level === 'error'
    ? 'text-error bg-error/10'
    : log.level === 'warning'
      ? 'text-warning bg-warning/10'
      : 'text-primary bg-primary/10'

  return (
    <div className="flex items-start gap-3 px-4 py-2.5">
      <span className={`mt-0.5 shrink-0 rounded px-1.5 py-0.5 text-[10px] font-mono font-medium uppercase ${levelColor}`}>
        {log.level}
      </span>
      <div className="flex-1 min-w-0">
        <p className="text-sm text-foreground">{log.message}</p>
        <div className="flex items-center gap-2 mt-0.5">
          <span className="text-[10px] text-muted-foreground font-mono">{log.timestamp}</span>
          {showFull && <span className="text-[10px] text-muted-foreground">&middot; {log.service}</span>}
        </div>
      </div>
    </div>
  )
}

import { useState } from 'react'
import {
  User,
  Calendar,
  FileText,
  Download,
  Clock,
  Briefcase,
  Mail,
  Phone,
  MapPin,
  Building2,
  Send,
  CheckCircle2,
  AlertCircle,
  ChevronRight,
} from 'lucide-react'
import { toast } from 'sonner'

// ============================================================
// Types
// ============================================================

type SelfServiceTab = 'profil' | 'antraege' | 'dokumente' | 'zeitkonto'

interface LeaveBalance {
  type: string
  total: number
  used: number
  remaining: number
  color: string
}

interface SalaryStatement {
  id: string
  month: string
  label: string
  gross: number
  net: number
}

interface TimeAccount {
  label: string
  hours: number
  type: 'positive' | 'negative' | 'neutral'
}

// ============================================================
// Mock Data
// ============================================================

const CURRENT_USER = {
  name: 'Jonas Diaz',
  initials: 'JD',
  role: 'Full-Stack Developer',
  department: 'Entwicklung',
  email: 'jonas.diaz@zentria.tech',
  phone: '+49 151 456 78 90',
  location: 'Berlin',
  joinDate: '2024-09-15',
  manager: 'Michael Berg',
  contractType: 'Vollzeit',
  workload: 100,
  employeeId: 'DE-2024-0042',
}

const LEAVE_BALANCES: LeaveBalance[] = [
  { type: 'Urlaub', total: 30, used: 8, remaining: 22, color: 'text-primary' },
  { type: 'Krankheit', total: 0, used: 3, remaining: 0, color: 'text-error' },
  { type: 'Sonderurlaub', total: 5, used: 1, remaining: 4, color: 'text-info' },
  { type: 'Homeoffice-Tage', total: 52, used: 12, remaining: 40, color: 'text-warning' },
]

const SALARY_STATEMENTS: SalaryStatement[] = [
  { id: 'ss-1', month: '2026-01', label: 'Januar 2026', gross: 7200, net: 4856 },
  { id: 'ss-2', month: '2025-12', label: 'Dezember 2025', gross: 7200, net: 4856 },
  { id: 'ss-3', month: '2025-11', label: 'November 2025', gross: 7200, net: 4856 },
  { id: 'ss-4', month: '2025-10', label: 'Oktober 2025', gross: 7200, net: 4856 },
  { id: 'ss-5', month: '2025-09', label: 'September 2025', gross: 7200, net: 4856 },
  { id: 'ss-6', month: '2025-08', label: 'August 2025', gross: 7200, net: 4856 },
]

const TIME_ACCOUNTS: TimeAccount[] = [
  { label: 'Ueberstunden (Monat)', hours: 4.5, type: 'positive' },
  { label: 'Ueberstunden (gesamt)', hours: 18.25, type: 'positive' },
  { label: 'Gleitzeit-Saldo', hours: 2.75, type: 'positive' },
  { label: 'Resturlaub (Tage)', hours: 22, type: 'neutral' },
]

const MY_REQUESTS = [
  { id: 'r-1', type: 'Urlaub', startDate: '2026-03-15', endDate: '2026-03-20', days: 4, status: 'pending' as const, reason: 'Fruehlings-Kurzurlaub' },
  { id: 'r-2', type: 'Homeoffice', startDate: '2026-02-24', endDate: '2026-02-28', days: 5, status: 'approved' as const, reason: 'Konzentrations-Woche' },
  { id: 'r-3', type: 'Arzttermin', startDate: '2026-02-12', endDate: '2026-02-12', days: 0.5, status: 'approved' as const, reason: 'Zahnarzt nachmittags' },
]

const requestStatusConfig = {
  pending: { label: 'Ausstehend', color: 'bg-warning-light text-warning' },
  approved: { label: 'Genehmigt', color: 'bg-success-light text-success' },
  rejected: { label: 'Abgelehnt', color: 'bg-error-light text-error' },
}

const formatEUR = (amount: number) =>
  new Intl.NumberFormat('de-DE', { style: 'currency', currency: 'EUR', minimumFractionDigits: 0, maximumFractionDigits: 0 }).format(amount)

// ============================================================
// Component
// ============================================================

export function SelfServiceView() {
  const [tab, setTab] = useState<SelfServiceTab>('profil')

  return (
    <div className="space-y-4">
      {/* Banner */}
      <div className="rounded-lg border border-primary/20 bg-primary-light/30 p-4">
        <div className="flex items-center gap-4">
          <div className="flex h-14 w-14 items-center justify-center rounded-full bg-primary text-lg font-bold text-primary-foreground">
            {CURRENT_USER.initials}
          </div>
          <div>
            <h2 className="text-lg font-semibold text-foreground">{CURRENT_USER.name}</h2>
            <p className="text-sm text-muted-foreground">{CURRENT_USER.role} · {CURRENT_USER.department}</p>
            <p className="text-xs text-muted-foreground">Personal-Nr: {CURRENT_USER.employeeId}</p>
          </div>
          <div className="ml-auto flex items-center gap-3">
            {LEAVE_BALANCES.slice(0, 2).map((lb) => (
              <div key={lb.type} className="text-center">
                <p className={`text-lg font-bold ${lb.color}`}>{lb.remaining}</p>
                <p className="text-[10px] text-muted-foreground">{lb.type} übrig</p>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-4 border-b border-border">
        {([
          { key: 'profil' as const, label: 'Mein Profil', icon: User },
          { key: 'antraege' as const, label: 'Meine Antraege', icon: Send },
          { key: 'dokumente' as const, label: 'Gehaltsabrechnungen', icon: FileText },
          { key: 'zeitkonto' as const, label: 'Zeitkonto', icon: Clock },
        ]).map((t) => {
          const Icon = t.icon
          return (
            <button
              key={t.key}
              onClick={() => setTab(t.key)}
              className={`flex items-center gap-1.5 border-b-2 px-1 pb-2 text-sm transition-colors ${
                tab === t.key ? 'border-primary text-primary font-medium' : 'border-transparent text-muted-foreground hover:text-foreground'
              }`}
            >
              <Icon className="h-3.5 w-3.5" />
              {t.label}
            </button>
          )
        })}
      </div>

      {/* Profile Tab */}
      {tab === 'profil' && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="rounded-lg border border-border bg-card p-5">
            <h3 className="text-sm font-medium text-foreground mb-4">Persönliche Daten</h3>
            <div className="space-y-3">
              {[
                { icon: User, label: 'Name', value: CURRENT_USER.name },
                { icon: Mail, label: 'E-Mail', value: CURRENT_USER.email },
                { icon: Phone, label: 'Telefon', value: CURRENT_USER.phone },
                { icon: MapPin, label: 'Standort', value: CURRENT_USER.location },
                { icon: Building2, label: 'Abteilung', value: CURRENT_USER.department },
                { icon: Briefcase, label: 'Vertragsart', value: `${CURRENT_USER.contractType} (${CURRENT_USER.workload}%)` },
                { icon: Calendar, label: 'Eintrittsdatum', value: new Date(CURRENT_USER.joinDate).toLocaleDateString('de-DE') },
                { icon: User, label: 'Vorgesetzte/r', value: CURRENT_USER.manager },
              ].map((item) => {
                const Icon = item.icon
                return (
                  <div key={item.label} className="flex items-center gap-3">
                    <Icon className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                    <span className="text-xs text-muted-foreground w-28">{item.label}</span>
                    <span className="text-sm text-foreground">{item.value}</span>
                  </div>
                )
              })}
            </div>
            <button
              onClick={() => toast.info('Änderungsantrag gestellt (Mock)')}
              className="mt-4 flex items-center gap-1.5 text-xs text-primary hover:underline"
            >
              Änderung beantragen
              <ChevronRight className="h-3 w-3" />
            </button>
          </div>

          <div className="space-y-4">
            <div className="rounded-lg border border-border bg-card p-5">
              <h3 className="text-sm font-medium text-foreground mb-4">Urlaubskonto</h3>
              <div className="space-y-3">
                {LEAVE_BALANCES.map((lb) => (
                  <div key={lb.type}>
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-xs text-muted-foreground">{lb.type}</span>
                      <span className={`text-xs font-medium ${lb.color}`}>
                        {lb.remaining}{lb.total > 0 ? `/${lb.total}` : ''} {lb.type === 'Krankheit' ? 'Tage genommen' : 'Tage übrig'}
                      </span>
                    </div>
                    {lb.total > 0 && (
                      <div className="h-1.5 rounded-full bg-secondary">
                        <div
                          className="h-full rounded-full bg-primary transition-all"
                          style={{ width: `${Math.min(100, (lb.used / lb.total) * 100)}%` }}
                        />
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>

            <div className="rounded-lg border border-border bg-card p-5">
              <h3 className="text-sm font-medium text-foreground mb-3">Schnellaktionen</h3>
              <div className="grid grid-cols-2 gap-2">
                {[
                  { label: 'Urlaub beantragen', icon: Calendar, onClick: () => toast.info('Urlaubsantrag-Dialog (Mock)') },
                  { label: 'Homeoffice melden', icon: MapPin, onClick: () => toast.info('Homeoffice-Meldung (Mock)') },
                  { label: 'Krank melden', icon: AlertCircle, onClick: () => toast.info('Krankmeldung (Mock)') },
                  { label: 'Ueberstunden melden', icon: Clock, onClick: () => toast.info('Ueberstunden-Meldung (Mock)') },
                ].map((action) => {
                  const Icon = action.icon
                  return (
                    <button
                      key={action.label}
                      onClick={action.onClick}
                      className="flex items-center gap-2 rounded-lg border border-border p-3 text-sm text-foreground hover:bg-secondary transition-colors text-left"
                    >
                      <Icon className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                      <span className="text-xs">{action.label}</span>
                    </button>
                  )
                })}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Requests Tab */}
      {tab === 'antraege' && (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <p className="text-sm text-muted-foreground">{MY_REQUESTS.length} Antraege</p>
            <button
              onClick={() => toast.info('Neuer Antrag (Mock)')}
              className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
            >
              <Send className="h-3.5 w-3.5" />
              Neuer Antrag
            </button>
          </div>
          <div className="space-y-3">
            {MY_REQUESTS.map((req) => {
              const st = requestStatusConfig[req.status]
              return (
                <div key={req.id} className="rounded-lg border border-border bg-card p-4">
                  <div className="flex items-start justify-between">
                    <div>
                      <div className="flex items-center gap-2 mb-1">
                        <span className="text-sm font-medium text-foreground">{req.type}</span>
                        <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${st.color}`}>{st.label}</span>
                      </div>
                      <div className="flex items-center gap-2 text-xs text-muted-foreground">
                        <Calendar className="h-3 w-3" />
                        <span>
                          {new Date(req.startDate).toLocaleDateString('de-DE')}
                          {req.startDate !== req.endDate && ` – ${new Date(req.endDate).toLocaleDateString('de-DE')}`}
                          {' '}({req.days} {req.days === 1 ? 'Tag' : 'Tage'})
                        </span>
                      </div>
                      <p className="text-xs text-muted-foreground mt-1">{req.reason}</p>
                    </div>
                    {req.status === 'approved' && <CheckCircle2 className="h-5 w-5 text-success flex-shrink-0" />}
                    {req.status === 'pending' && <Clock className="h-5 w-5 text-warning flex-shrink-0" />}
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      )}

      {/* Documents Tab */}
      {tab === 'dokumente' && (
        <div className="space-y-3">
          <p className="text-sm text-muted-foreground">Letzte Gehaltsabrechnungen zum Download</p>
          <div className="rounded-lg border border-border overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-secondary/50">
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">Monat</th>
                    <th className="px-4 py-3 text-right font-medium text-muted-foreground">Brutto</th>
                    <th className="px-4 py-3 text-right font-medium text-muted-foreground">Netto</th>
                    <th className="px-4 py-3 text-right font-medium text-muted-foreground">Aktion</th>
                  </tr>
                </thead>
                <tbody>
                  {SALARY_STATEMENTS.map((ss) => (
                    <tr key={ss.id} className="border-b border-border-muted hover:bg-secondary/20">
                      <td className="px-4 py-3 font-medium text-foreground">{ss.label}</td>
                      <td className="px-4 py-3 text-right text-muted-foreground tabular-nums">{formatEUR(ss.gross)}</td>
                      <td className="px-4 py-3 text-right font-medium text-foreground tabular-nums">{formatEUR(ss.net)}</td>
                      <td className="px-4 py-3 text-right">
                        <button
                          onClick={() => toast.success(`Download: Gehaltsabrechnung ${ss.label}`)}
                          className="inline-flex items-center gap-1 rounded-md border border-border px-2 py-1 text-xs text-foreground hover:bg-secondary transition-colors"
                        >
                          <Download className="h-3 w-3" />
                          PDF
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}

      {/* Time Account Tab */}
      {tab === 'zeitkonto' && (
        <div className="space-y-4">
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
            {TIME_ACCOUNTS.map((ta) => (
              <div key={ta.label} className="rounded-lg border border-border bg-card p-4">
                <p className="text-xs text-muted-foreground mb-1">{ta.label}</p>
                <p className={`text-lg font-semibold ${
                  ta.type === 'positive' ? 'text-success' : ta.type === 'negative' ? 'text-error' : 'text-foreground'
                }`}>
                  {ta.type === 'positive' && '+'}{ta.hours}{ta.label.includes('Tage') ? '' : 'h'}
                </p>
              </div>
            ))}
          </div>

          <div className="rounded-lg border border-border bg-card p-5">
            <h3 className="text-sm font-medium text-foreground mb-3">Arbeitszeitkonto (letzte 5 Tage)</h3>
            <div className="space-y-2">
              {[
                { day: 'Mo, 17.02.', soll: '8:00', ist: '8:30', diff: '+0:30' },
                { day: 'Di, 18.02.', soll: '8:00', ist: '8:45', diff: '+0:45' },
                { day: 'Mi, 19.02.', soll: '8:00', ist: '7:30', diff: '-0:30' },
                { day: 'Do, 20.02.', soll: '8:00', ist: '9:00', diff: '+1:00' },
                { day: 'Fr, 21.02.', soll: '8:00', ist: '7:45', diff: '-0:15' },
              ].map((entry) => (
                <div key={entry.day} className="flex items-center justify-between py-2 border-b border-border-muted last:border-0">
                  <span className="text-sm text-foreground w-24">{entry.day}</span>
                  <span className="text-xs text-muted-foreground">Soll: {entry.soll}</span>
                  <span className="text-xs text-muted-foreground">Ist: {entry.ist}</span>
                  <span className={`text-xs font-medium tabular-nums ${
                    entry.diff.startsWith('+') ? 'text-success' : 'text-error'
                  }`}>
                    {entry.diff}
                  </span>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

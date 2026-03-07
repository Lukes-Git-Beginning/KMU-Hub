/**
 * DSAR (Data Subject Access Request) search page — Art. 15 DSGVO.
 *
 * Global cross-module search for a person's data. Shows all stored data
 * grouped by module with record counts. Supports JSON/CSV mock export.
 */
import { useState, useCallback } from 'react'
import { Navigate } from 'react-router-dom'
import {
  Search,
  User,
  ChevronDown,
  ChevronRight,
  FileJson,
  FileSpreadsheet,
  Mail,
  MessageSquare,
  Calendar,
  FileText,
  Headphones,
  Receipt,
  FolderKanban,
  ClipboardList,
  BarChart3,
} from 'lucide-react'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth'

interface ModuleData {
  module: string
  icon: typeof Mail
  records: Array<Record<string, string>>
  columns: string[]
}

interface PersonResult {
  id: string
  name: string
  email: string
  company: string
  avatar: string
  modules: ModuleData[]
}

const MOCK_PERSONS: Record<string, PersonResult> = {
  'max': {
    id: 'user-1',
    name: 'Max Mustermann',
    email: 'max@example.com',
    company: 'Mustermann GmbH',
    avatar: 'MM',
    modules: [
      {
        module: 'CRM Kontakte',
        icon: User,
        columns: ['Feld', 'Wert'],
        records: [
          { Feld: 'Name', Wert: 'Max Mustermann' },
          { Feld: 'E-Mail', Wert: 'max@example.com' },
          { Feld: 'Telefon', Wert: '+49 170 1234567' },
          { Feld: 'Firma', Wert: 'Mustermann GmbH' },
          { Feld: 'Position', Wert: 'Geschaeftsfuehrer' },
          { Feld: 'Erstellt am', Wert: '2025-03-12' },
        ],
      },
      {
        module: 'CRM Deals',
        icon: BarChart3,
        columns: ['Deal', 'Status', 'Wert'],
        records: [
          { Deal: 'Website Redesign', Status: 'Gewonnen', Wert: '12.500 EUR' },
          { Deal: 'SEO-Paket 2026', Status: 'Verhandlung', Wert: '4.800 EUR' },
          { Deal: 'App-Entwicklung', Status: 'Angebot', Wert: '35.000 EUR' },
        ],
      },
      {
        module: 'E-Mails',
        icon: Mail,
        columns: ['Betreff', 'Datum', 'Richtung'],
        records: [
          { Betreff: 'Angebot Website Redesign', Datum: '2026-02-18', Richtung: 'Gesendet' },
          { Betreff: 'Re: Vertragsentwurf', Datum: '2026-02-15', Richtung: 'Empfangen' },
          { Betreff: 'Terminbestaetigung', Datum: '2026-02-10', Richtung: 'Gesendet' },
          { Betreff: 'Anfrage SEO-Paket', Datum: '2026-01-28', Richtung: 'Empfangen' },
          { Betreff: 'Rechnung RE-2025-089', Datum: '2025-12-15', Richtung: 'Gesendet' },
        ],
      },
      {
        module: 'Chat-Nachrichten',
        icon: MessageSquare,
        columns: ['Kanal', 'Nachrichten', 'Zeitraum'],
        records: [
          { Kanal: '#projekt-mustermann', Nachrichten: '127', Zeitraum: 'Marz 2025 – Feb 2026' },
          { Kanal: 'Direktnachricht', Nachrichten: '43', Zeitraum: 'Jan 2026 – Feb 2026' },
        ],
      },
      {
        module: 'Kalender',
        icon: Calendar,
        columns: ['Termin', 'Datum', 'Teilnehmer'],
        records: [
          { Termin: 'Kick-off Meeting', Datum: '2025-04-01', Teilnehmer: '4' },
          { Termin: 'Sprint Review', Datum: '2025-06-15', Teilnehmer: '6' },
          { Termin: 'Abnahme Website', Datum: '2025-09-20', Teilnehmer: '3' },
          { Termin: 'Jahresgespraech 2026', Datum: '2026-01-10', Teilnehmer: '2' },
        ],
      },
      {
        module: 'Dokumente',
        icon: FileText,
        columns: ['Dateiname', 'Hochgeladen', 'Groesse'],
        records: [
          { Dateiname: 'Vertrag_Mustermann_2025.pdf', Hochgeladen: '2025-03-15', Groesse: '245 KB' },
          { Dateiname: 'Angebot_Redesign_v2.pdf', Hochgeladen: '2025-04-02', Groesse: '1.2 MB' },
          { Dateiname: 'Logo_Mustermann.png', Hochgeladen: '2025-04-10', Groesse: '89 KB' },
        ],
      },
      {
        module: 'Helpdesk-Tickets',
        icon: Headphones,
        columns: ['Ticket', 'Betreff', 'Status'],
        records: [
          { Ticket: 'TK-0847', Betreff: 'Login-Problem Portal', Status: 'Geschlossen' },
          { Ticket: 'TK-1102', Betreff: 'PDF-Export fehlerhaft', Status: 'Offen' },
        ],
      },
      {
        module: 'Rechnungen',
        icon: Receipt,
        columns: ['Nummer', 'Betrag', 'Datum', 'Status'],
        records: [
          { Nummer: 'RE-2025-089', Betrag: '12.500,00 EUR', Datum: '2025-12-15', Status: 'Bezahlt' },
          { Nummer: 'RE-2026-012', Betrag: '4.800,00 EUR', Datum: '2026-02-01', Status: 'Offen' },
        ],
      },
      {
        module: 'Projekte',
        icon: FolderKanban,
        columns: ['Projekt', 'Rolle', 'Zeitraum'],
        records: [
          { Projekt: 'Website Redesign Mustermann', Rolle: 'Auftraggeber', Zeitraum: 'Apr 2025 – Okt 2025' },
          { Projekt: 'SEO Optimierung', Rolle: 'Ansprechpartner', Zeitraum: 'Feb 2026 – laufend' },
        ],
      },
      {
        module: 'Formulare',
        icon: ClipboardList,
        columns: ['Formular', 'Eingereicht', 'Status'],
        records: [
          { Formular: 'Kontaktformular Website', Eingereicht: '2025-02-28', Status: 'Verarbeitet' },
        ],
      },
    ],
  },
  'anna': {
    id: 'user-2',
    name: 'Anna Schmidt',
    email: 'anna@example.com',
    company: 'Schmidt & Partner',
    avatar: 'AS',
    modules: [
      {
        module: 'CRM Kontakte',
        icon: User,
        columns: ['Feld', 'Wert'],
        records: [
          { Feld: 'Name', Wert: 'Anna Schmidt' },
          { Feld: 'E-Mail', Wert: 'anna@example.com' },
          { Feld: 'Telefon', Wert: '+49 151 9876543' },
          { Feld: 'Firma', Wert: 'Schmidt & Partner' },
        ],
      },
      {
        module: 'E-Mails',
        icon: Mail,
        columns: ['Betreff', 'Datum', 'Richtung'],
        records: [
          { Betreff: 'Partnervertrag 2026', Datum: '2026-01-20', Richtung: 'Empfangen' },
          { Betreff: 'Re: Partnervertrag', Datum: '2026-01-22', Richtung: 'Gesendet' },
        ],
      },
      {
        module: 'Rechnungen',
        icon: Receipt,
        columns: ['Nummer', 'Betrag', 'Datum', 'Status'],
        records: [
          { Nummer: 'RE-2025-045', Betrag: '8.900,00 EUR', Datum: '2025-08-10', Status: 'Bezahlt' },
        ],
      },
    ],
  },
  'peter': {
    id: 'user-3',
    name: 'Peter Mueller',
    email: 'peter@example.com',
    company: 'Mueller Technik AG',
    avatar: 'PM',
    modules: [
      {
        module: 'CRM Kontakte',
        icon: User,
        columns: ['Feld', 'Wert'],
        records: [
          { Feld: 'Name', Wert: 'Peter Mueller' },
          { Feld: 'E-Mail', Wert: 'peter@example.com' },
          { Feld: 'Firma', Wert: 'Mueller Technik AG' },
        ],
      },
      {
        module: 'Helpdesk-Tickets',
        icon: Headphones,
        columns: ['Ticket', 'Betreff', 'Status'],
        records: [
          { Ticket: 'TK-0523', Betreff: 'Server-Migration Fragen', Status: 'Geschlossen' },
          { Ticket: 'TK-0890', Betreff: 'API Zugang beantragen', Status: 'Geschlossen' },
          { Ticket: 'TK-1205', Betreff: 'SSL Zertifikat erneuern', Status: 'Offen' },
        ],
      },
      {
        module: 'Projekte',
        icon: FolderKanban,
        columns: ['Projekt', 'Rolle', 'Zeitraum'],
        records: [
          { Projekt: 'Cloud-Migration Mueller', Rolle: 'Technischer Leiter', Zeitraum: 'Nov 2025 – laufend' },
        ],
      },
    ],
  },
}

export default function DSARSearchPage() {
  const user = useAuthStore((s) => s.user)
  const isAdmin = user?.roles.includes('admin')

  const [searchQuery, setSearchQuery] = useState('')
  const [isSearching, setIsSearching] = useState(false)
  const [result, setResult] = useState<PersonResult | null>(null)
  const [expandedModules, setExpandedModules] = useState<Set<string>>(new Set())

  const handleSearch = useCallback(() => {
    if (searchQuery.length < 2) return
    setIsSearching(true)
    setResult(null)
    setExpandedModules(new Set())

    setTimeout(() => {
      const q = searchQuery.toLowerCase()
      const found = Object.values(MOCK_PERSONS).find(
        (p) => p.name.toLowerCase().includes(q) || p.email.toLowerCase().includes(q),
      )
      setResult(found ?? null)
      if (found) {
        setExpandedModules(new Set([found.modules[0]?.module]))
      }
      setIsSearching(false)
    }, 600)
  }, [searchQuery])

  const toggleModule = (mod: string) => {
    setExpandedModules((prev) => {
      const next = new Set(prev)
      if (next.has(mod)) next.delete(mod)
      else next.add(mod)
      return next
    })
  }

  const totalRecords = result
    ? result.modules.reduce((sum, m) => sum + m.records.length, 0)
    : 0

  const handleExport = (format: string) => {
    toast.success(`Datenpaket als ${format} wird erstellt...`)
    setTimeout(() => toast.success(`${format}-Export fuer ${result?.name} bereit zum Download`), 1500)
  }

  if (!isAdmin) {
    return <Navigate to="/" replace />
  }

  return (
    <div className="flex-1 overflow-y-auto p-6">
      <div className="mx-auto max-w-3xl">
        {/* Header */}
        <div className="flex items-center gap-4 mb-6">
          <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-info-light">
            <Search className="h-6 w-6 text-info" />
          </div>
          <div>
            <h1 className="text-foreground">Auskunft (Art. 15 DSGVO)</h1>
            <p className="text-sm text-muted-foreground mt-0.5">
              Durchsuche alle Module nach personenbezogenen Daten einer betroffenen Person.
            </p>
          </div>
        </div>

        {/* Search */}
        <div className="rounded-lg border border-border bg-card p-5 glass-surface mb-6">
          <h3 className="text-sm font-semibold text-foreground mb-3">Person suchen</h3>
          <div className="flex gap-2">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <input
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
                placeholder="Name oder E-Mail-Adresse eingeben..."
                className="w-full rounded-lg border border-border bg-card pl-10 pr-3 py-2.5 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <button
              onClick={handleSearch}
              disabled={searchQuery.length < 2 || isSearching}
              className="rounded-lg bg-primary px-5 py-2.5 text-sm font-medium text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-40"
            >
              {isSearching ? (
                <span className="flex items-center gap-2">
                  <span className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
                  Suche...
                </span>
              ) : (
                'Suchen'
              )}
            </button>
          </div>
          <p className="text-[11px] text-muted-foreground mt-2">
            Testdaten: Max Mustermann, Anna Schmidt, Peter Mueller
          </p>
        </div>

        {/* No result */}
        {!isSearching && searchQuery.length >= 2 && result === null && (
          <div className="rounded-lg border border-border bg-card p-8 text-center glass-surface mb-6">
            <Search className="mx-auto h-10 w-10 text-muted-foreground/30 mb-2" />
            <p className="text-sm text-muted-foreground">
              Keine Person gefunden. Bitte ueberpruefen Sie die Suchkriterien.
            </p>
          </div>
        )}

        {/* Person card + results */}
        {result && (
          <>
            {/* Person card */}
            <div className="rounded-lg border border-primary/20 bg-primary-light/20 p-4 mb-6">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="flex h-11 w-11 items-center justify-center rounded-full bg-primary text-sm font-medium text-primary-foreground">
                    {result.avatar}
                  </div>
                  <div>
                    <p className="text-sm font-medium text-foreground">{result.name}</p>
                    <p className="text-xs text-muted-foreground">{result.email} &middot; {result.company}</p>
                  </div>
                </div>
                <div className="flex items-center gap-4 text-xs text-muted-foreground">
                  <span>{result.modules.length} Module</span>
                  <span className="font-medium text-foreground">{totalRecords} Datensaetze</span>
                </div>
              </div>
            </div>

            {/* Module accordions */}
            <div className="space-y-2 mb-6">
              {result.modules.map((mod) => {
                const Icon = mod.icon
                const isExpanded = expandedModules.has(mod.module)
                return (
                  <div key={mod.module} className="rounded-lg border border-border bg-card glass-surface overflow-hidden">
                    <button
                      onClick={() => toggleModule(mod.module)}
                      className="flex w-full items-center justify-between p-3 hover:bg-secondary/30 transition-colors"
                    >
                      <div className="flex items-center gap-2.5">
                        {isExpanded ? (
                          <ChevronDown className="h-4 w-4 text-muted-foreground" />
                        ) : (
                          <ChevronRight className="h-4 w-4 text-muted-foreground" />
                        )}
                        <Icon className="h-4 w-4 text-primary" />
                        <span className="text-sm font-medium text-foreground">{mod.module}</span>
                      </div>
                      <span className="rounded-full bg-secondary px-2 py-0.5 text-[10px] font-medium text-muted-foreground tabular-nums">
                        {mod.records.length} {mod.records.length === 1 ? 'Eintrag' : 'Eintraege'}
                      </span>
                    </button>

                    {isExpanded && (
                      <div className="border-t border-border">
                        <table className="w-full text-sm">
                          <thead>
                            <tr className="border-b border-border bg-secondary/20">
                              {mod.columns.map((col) => (
                                <th key={col} className="px-4 py-2 text-left text-[10px] font-medium text-muted-foreground uppercase">
                                  {col}
                                </th>
                              ))}
                            </tr>
                          </thead>
                          <tbody>
                            {mod.records.map((record, i) => (
                              <tr key={i} className="border-b border-border-muted last:border-0">
                                {mod.columns.map((col) => (
                                  <td key={col} className="px-4 py-2 text-xs text-foreground">
                                    {record[col] ?? '-'}
                                  </td>
                                ))}
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    )}
                  </div>
                )
              })}
            </div>

            {/* Export buttons */}
            <div className="flex items-center gap-3">
              <button
                onClick={() => handleExport('JSON')}
                className="flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
              >
                <FileJson className="h-4 w-4" />
                Als JSON exportieren
              </button>
              <button
                onClick={() => handleExport('CSV')}
                className="flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
              >
                <FileSpreadsheet className="h-4 w-4" />
                Als CSV exportieren
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  )
}

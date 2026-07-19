/**
 * DSAR (Data Subject Access Request) search page — Art. 15 DSGVO.
 *
 * Global cross-module search for a person's data. Shows all stored data
 * grouped by module with record counts. Supports JSON/CSV mock export.
 */
import { useState, useCallback } from 'react'
import { Navigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
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
import { useHasCapability } from '@/hooks/useCapability'
import { authenticatedRequest } from '@/api/utils/authenticatedFetch'

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

// Backend DSAR response shapes (see gateway HandleDSARSearch).
interface BackendModule {
  module: string
  columns: string[]
  records: Array<Record<string, string>>
}

interface BackendPerson {
  id: string
  name: string
  email: string
  company: string
  avatar: string
  modules: BackendModule[]
}

// The backend returns module names only; the icon is a pure UI concern. Map the
// full module catalogue so newly surfaced modules render with a sensible icon.
const MODULE_ICONS: Record<string, typeof Mail> = {
  'CRM Kontakte': User,
  'CRM Deals': BarChart3,
  Benutzerkonto: User,
  Einwilligungen: ClipboardList,
  Anrufe: Headphones,
  'E-Mails': Mail,
  Chat: MessageSquare,
  Kalender: Calendar,
  Dokumente: FileText,
  Helpdesk: Headphones,
  Rechnungen: Receipt,
  Projekte: FolderKanban,
  Formulare: ClipboardList,
  Berichte: BarChart3,
}

function toPersonResult(p: BackendPerson): PersonResult {
  return {
    id: p.id,
    name: p.name,
    email: p.email,
    company: p.company,
    avatar: p.avatar,
    modules: p.modules.map((m) => ({
      module: m.module,
      icon: MODULE_ICONS[m.module] ?? FileText,
      columns: m.columns,
      records: m.records,
    })),
  }
}

export default function DSARSearchPage() {
  const { t } = useTranslation()
  const canExecute = useHasCapability('security:gdpr:execute')

  const [searchQuery, setSearchQuery] = useState('')
  const [isSearching, setIsSearching] = useState(false)
  const [result, setResult] = useState<PersonResult | null>(null)
  const [expandedModules, setExpandedModules] = useState<Set<string>>(new Set())

  const handleSearch = useCallback(async () => {
    if (searchQuery.length < 2) return
    setIsSearching(true)
    setResult(null)
    setExpandedModules(new Set())

    try {
      const data = await authenticatedRequest<{ results: BackendPerson[] }>({
        method: 'GET',
        path: '/api/v1/security/dsar/search',
        params: { q: searchQuery },
      })
      const first = data.results?.[0]
      if (first) {
        const mapped = toPersonResult(first)
        setResult(mapped)
        setExpandedModules(new Set([mapped.modules[0]?.module]))
      } else {
        setResult(null)
      }
    } catch {
      setResult(null)
      toast.error(t('security.dsar.search.error'))
    } finally {
      setIsSearching(false)
    }
  }, [searchQuery, t])

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
    if (!result) return
    toast.success(t('security.dsar.export.preparing', { format }))
    let content: string
    let mime: string
    let ext: string
    if (format === 'JSON') {
      content = JSON.stringify(
        {
          subject: { name: result.name, email: result.email, company: result.company },
          generated_at: new Date().toISOString(),
          modules: result.modules.map((m) => ({ module: m.module, records: m.records })),
        },
        null,
        2,
      )
      mime = 'application/json'
      ext = 'json'
    } else {
      const lines: string[] = []
      for (const m of result.modules) {
        lines.push(`# ${m.module}`)
        lines.push(m.columns.join(','))
        for (const rec of m.records) {
          lines.push(m.columns.map((c) => `"${(rec[c] ?? '').replace(/"/g, '""')}"`).join(','))
        }
        lines.push('')
      }
      content = lines.join('\n')
      mime = 'text/csv'
      ext = 'csv'
    }
    const blob = new Blob([content], { type: mime })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `dsar-${result.name.replace(/\s+/g, '_').toLowerCase()}.${ext}`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    toast.success(t('security.dsar.export.ready', { format, name: result.name }))
  }

  if (!canExecute) {
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
            <h1 className="text-foreground">{t('security.dsar.title')}</h1>
            <p className="text-sm text-muted-foreground mt-0.5">
              {t('security.dsar.subtitle')}
            </p>
          </div>
        </div>

        {/* Search */}
        <div className="rounded-lg border border-border bg-card p-5 glass-surface mb-6">
          <h3 className="text-sm font-semibold text-foreground mb-3">{t('security.dsar.search.heading')}</h3>
          <div className="flex gap-2">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <input
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onKeyDown={(e) => { if (e.key === 'Enter') void handleSearch() }}
                placeholder={t('security.dsar.search.placeholder')}
                className="w-full rounded-lg border border-border bg-card pl-10 pr-3 py-2.5 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <button
              onClick={() => void handleSearch()}
              disabled={searchQuery.length < 2 || isSearching}
              className="rounded-lg bg-primary px-5 py-2.5 text-sm font-medium text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-40"
            >
              {isSearching ? (
                <span className="flex items-center gap-2">
                  <span className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
                  {t('security.dsar.search.searching')}
                </span>
              ) : (
                t('common.search')
              )}
            </button>
          </div>
          <p className="text-[11px] text-muted-foreground mt-2">
            {t('security.dsar.search.hint')}
          </p>
        </div>

        {/* No result */}
        {!isSearching && searchQuery.length >= 2 && result === null && (
          <div className="rounded-lg border border-border bg-card p-8 text-center glass-surface mb-6">
            <Search className="mx-auto h-10 w-10 text-muted-foreground/30 mb-2" />
            <p className="text-sm text-muted-foreground">
              {t('security.dsar.search.noResult')}
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
                  <span>{t('security.dsar.result.moduleCount', { count: result.modules.length })}</span>
                  <span className="font-medium text-foreground">{t('security.dsar.result.recordCount', { count: totalRecords })}</span>
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
                        {t('security.dsar.result.entryCount', { count: mod.records.length })}
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
                {t('security.dsar.export.json')}
              </button>
              <button
                onClick={() => handleExport('CSV')}
                className="flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
              >
                <FileSpreadsheet className="h-4 w-4" />
                {t('security.dsar.export.csv')}
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  )
}

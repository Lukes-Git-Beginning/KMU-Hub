import { useState } from 'react'
import {
  FileText, Upload, Download, Trash2, Eye, FolderOpen,
  Receipt, Briefcase, FileCheck, Files, Search,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/cn'
import { toast } from 'sonner'

// ── Types ──────────────────────────────────────────────

interface Document {
  id: string
  name: string
  category: DocumentCategory
  date: string
  size: string
  type: 'pdf' | 'docx' | 'xlsx' | 'png' | 'jpg'
  uploadedBy: 'self' | 'hr'
}

type DocumentCategory = 'payslips' | 'contracts' | 'certificates' | 'applications' | 'other'

const CATEGORIES: { key: DocumentCategory; label: string; icon: typeof FileText; description: string }[] = [
  { key: 'payslips', label: 'Lohnabrechnungen', icon: Receipt, description: 'Monatliche Gehaltsabrechnungen' },
  { key: 'contracts', label: 'Arbeitsverträge', icon: FileCheck, description: 'Verträge, Nachträge, Vereinbarungen' },
  { key: 'certificates', label: 'Zeugnisse & Zertifikate', icon: Briefcase, description: 'Arbeitszeugnisse, Schulungen, Zertifikate' },
  { key: 'applications', label: 'Bewerbungsunterlagen', icon: Files, description: 'Lebenslauf, Anschreiben, Portfolio' },
  { key: 'other', label: 'Sonstige Dokumente', icon: FolderOpen, description: 'Weitere arbeitsrelevante Unterlagen' },
]

// ── Mock Data ──────────────────────────────────────────

const MOCK_DOCUMENTS: Document[] = [
  // Payslips
  { id: 'd1', name: 'Lohnabrechnung_2026_01.pdf', category: 'payslips', date: '2026-01-31', size: '245 KB', type: 'pdf', uploadedBy: 'hr' },
  { id: 'd2', name: 'Lohnabrechnung_2025_12.pdf', category: 'payslips', date: '2025-12-31', size: '238 KB', type: 'pdf', uploadedBy: 'hr' },
  { id: 'd3', name: 'Lohnabrechnung_2025_11.pdf', category: 'payslips', date: '2025-11-30', size: '241 KB', type: 'pdf', uploadedBy: 'hr' },
  { id: 'd4', name: 'Lohnabrechnung_2025_10.pdf', category: 'payslips', date: '2025-10-31', size: '239 KB', type: 'pdf', uploadedBy: 'hr' },
  { id: 'd5', name: 'Lohnabrechnung_2025_09.pdf', category: 'payslips', date: '2025-09-30', size: '242 KB', type: 'pdf', uploadedBy: 'hr' },
  { id: 'd6', name: 'Lohnabrechnung_2025_08.pdf', category: 'payslips', date: '2025-08-31', size: '237 KB', type: 'pdf', uploadedBy: 'hr' },
  // Contracts
  { id: 'd7', name: 'Arbeitsvertrag_KMU_Hub_AG.pdf', category: 'contracts', date: '2025-03-01', size: '1.2 MB', type: 'pdf', uploadedBy: 'hr' },
  { id: 'd8', name: 'Nachtrag_Gehaltserhöhung_2026.pdf', category: 'contracts', date: '2026-01-01', size: '156 KB', type: 'pdf', uploadedBy: 'hr' },
  { id: 'd9', name: 'Datenschutzvereinbarung.pdf', category: 'contracts', date: '2025-03-01', size: '89 KB', type: 'pdf', uploadedBy: 'hr' },
  { id: 'd10', name: 'Homeoffice_Vereinbarung.pdf', category: 'contracts', date: '2025-06-15', size: '134 KB', type: 'pdf', uploadedBy: 'hr' },
  // Certificates
  { id: 'd11', name: 'Zwischenzeugnis_2025.pdf', category: 'certificates', date: '2025-12-15', size: '312 KB', type: 'pdf', uploadedBy: 'hr' },
  { id: 'd12', name: 'Zertifikat_React_Advanced.pdf', category: 'certificates', date: '2025-09-20', size: '567 KB', type: 'pdf', uploadedBy: 'self' },
  { id: 'd13', name: 'Zertifikat_AWS_Solutions_Architect.pdf', category: 'certificates', date: '2025-07-10', size: '489 KB', type: 'pdf', uploadedBy: 'self' },
  // Applications
  { id: 'd14', name: 'Lebenslauf_Markus_Weber.pdf', category: 'applications', date: '2025-02-15', size: '456 KB', type: 'pdf', uploadedBy: 'self' },
  { id: 'd15', name: 'Bewerbungsschreiben_KMU_Hub.pdf', category: 'applications', date: '2025-02-15', size: '178 KB', type: 'pdf', uploadedBy: 'self' },
  { id: 'd16', name: 'Portfolio_Webentwicklung.pdf', category: 'applications', date: '2025-02-10', size: '3.4 MB', type: 'pdf', uploadedBy: 'self' },
  // Other
  { id: 'd17', name: 'Firmenausweis_Antrag.pdf', category: 'other', date: '2025-03-05', size: '67 KB', type: 'pdf', uploadedBy: 'hr' },
  { id: 'd18', name: 'Parkplatz_Zuweisung.pdf', category: 'other', date: '2025-04-01', size: '45 KB', type: 'pdf', uploadedBy: 'hr' },
  { id: 'd19', name: 'Lohnausweis_2025.pdf', category: 'other', date: '2026-01-15', size: '198 KB', type: 'pdf', uploadedBy: 'hr' },
]

// ── Component ──────────────────────────────────────────

export default function DokumenteTab() {
  const [documents] = useState<Document[]>(MOCK_DOCUMENTS)
  const [activeCategory, setActiveCategory] = useState<DocumentCategory | 'all'>('all')
  const [searchQuery, setSearchQuery] = useState('')

  const filtered = documents
    .filter((d) => activeCategory === 'all' || d.category === activeCategory)
    .filter((d) => searchQuery === '' || d.name.toLowerCase().includes(searchQuery.toLowerCase()))
    .sort((a, b) => b.date.localeCompare(a.date))

  const categoryCounts = CATEGORIES.reduce((acc, cat) => {
    acc[cat.key] = documents.filter((d) => d.category === cat.key).length
    return acc
  }, {} as Record<DocumentCategory, number>)

  const formatDate = (dateStr: string) => {
    const d = new Date(dateStr)
    return `${d.getDate().toString().padStart(2, '0')}.${(d.getMonth() + 1).toString().padStart(2, '0')}.${d.getFullYear()}`
  }

  const getTypeColor = (type: Document['type']) => {
    switch (type) {
      case 'pdf': return 'text-red-600 dark:text-red-400 bg-red-100 dark:bg-red-900/30'
      case 'docx': return 'text-blue-600 dark:text-blue-400 bg-blue-100 dark:bg-blue-900/30'
      case 'xlsx': return 'text-emerald-600 dark:text-emerald-400 bg-emerald-100 dark:bg-emerald-900/30'
      default: return 'text-gray-600 dark:text-gray-400 bg-gray-100 dark:bg-gray-900/30'
    }
  }

  return (
    <div className="h-full flex">
      {/* Sidebar */}
      <div className="w-64 shrink-0 border-r border-border bg-card/30 p-4 space-y-1">
        <button
          onClick={() => setActiveCategory('all')}
          className={cn(
            'w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm transition-colors',
            activeCategory === 'all'
              ? 'bg-primary/10 text-primary font-medium'
              : 'text-muted-foreground hover:bg-secondary hover:text-foreground',
          )}
        >
          <FolderOpen className="h-4 w-4 shrink-0" />
          <span className="flex-1 text-left">Alle Dokumente</span>
          <span className="text-xs">{documents.length}</span>
        </button>

        <div className="pt-2 pb-1 px-3">
          <span className="text-[10px] font-semibold text-muted-foreground uppercase tracking-wider">Kategorien</span>
        </div>

        {CATEGORIES.map((cat) => {
          const Icon = cat.icon
          return (
            <button
              key={cat.key}
              onClick={() => setActiveCategory(cat.key)}
              className={cn(
                'w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm transition-colors',
                activeCategory === cat.key
                  ? 'bg-primary/10 text-primary font-medium'
                  : 'text-muted-foreground hover:bg-secondary hover:text-foreground',
              )}
            >
              <Icon className="h-4 w-4 shrink-0" />
              <span className="flex-1 text-left truncate">{cat.label}</span>
              <span className="text-xs">{categoryCounts[cat.key]}</span>
            </button>
          )
        })}

        {/* Storage info */}
        <div className="pt-4 mt-4 border-t border-border">
          <div className="px-3">
            <p className="text-xs text-muted-foreground mb-2">Speicherplatz</p>
            <div className="w-full h-2 rounded-full bg-secondary overflow-hidden">
              <div className="h-full rounded-full bg-primary" style={{ width: '23%' }} />
            </div>
            <p className="text-[10px] text-muted-foreground mt-1">234 MB / 1 GB verwendet</p>
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Toolbar */}
        <div className="flex items-center gap-3 px-6 py-3 border-b border-border">
          <div className="relative flex-1 max-w-sm">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Dokument suchen..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-9"
            />
          </div>
          <div className="ml-auto flex items-center gap-2">
            <Button
              size="sm"
              variant="outline"
              className="gap-2"
              onClick={() => toast.success('Upload-Funktion wird später aktiviert')}
            >
              <Upload className="h-4 w-4" />
              Hochladen
            </Button>
          </div>
        </div>

        {/* Category Header */}
        {activeCategory !== 'all' && (
          <div className="px-6 py-3 bg-card/30 border-b border-border">
            <div className="flex items-center gap-2">
              {(() => {
                const cat = CATEGORIES.find((c) => c.key === activeCategory)!
                const Icon = cat.icon
                return (
                  <>
                    <Icon className="h-5 w-5 text-primary" />
                    <div>
                      <h3 className="text-sm font-semibold text-foreground">{cat.label}</h3>
                      <p className="text-xs text-muted-foreground">{cat.description}</p>
                    </div>
                  </>
                )
              })()}
            </div>
          </div>
        )}

        {/* Document List */}
        <div className="flex-1 overflow-auto p-6">
          {filtered.length === 0 ? (
            <div className="text-center py-16 text-muted-foreground">
              <FileText className="h-12 w-12 mx-auto mb-3 opacity-30" />
              <p className="font-medium">Keine Dokumente gefunden</p>
              <p className="text-sm mt-1">
                {searchQuery ? 'Versuche einen anderen Suchbegriff' : 'In dieser Kategorie sind noch keine Dokumente'}
              </p>
            </div>
          ) : (
            <div className="space-y-2">
              {filtered.map((doc) => (
                <div
                  key={doc.id}
                  className="flex items-center gap-3 p-3 rounded-lg border border-border bg-card hover:border-primary/30 transition-colors group"
                >
                  {/* File type badge */}
                  <span className={cn(
                    'px-2 py-1 rounded text-[10px] font-bold uppercase shrink-0',
                    getTypeColor(doc.type),
                  )}>
                    {doc.type}
                  </span>

                  {/* File info */}
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-foreground truncate">{doc.name}</p>
                    <div className="flex items-center gap-2 text-xs text-muted-foreground mt-0.5">
                      <span>{formatDate(doc.date)}</span>
                      <span className="text-border">|</span>
                      <span>{doc.size}</span>
                      {doc.uploadedBy === 'hr' && (
                        <>
                          <span className="text-border">|</span>
                          <span className="text-primary">Von HR bereitgestellt</span>
                        </>
                      )}
                    </div>
                  </div>

                  {/* Actions */}
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                    <button
                      onClick={() => toast.info('Vorschau wird geöffnet...')}
                      className="p-1.5 rounded hover:bg-secondary text-muted-foreground hover:text-foreground transition-colors"
                      title="Vorschau"
                    >
                      <Eye className="h-4 w-4" />
                    </button>
                    <button
                      onClick={() => toast.success(`${doc.name} wird heruntergeladen...`)}
                      className="p-1.5 rounded hover:bg-secondary text-muted-foreground hover:text-foreground transition-colors"
                      title="Herunterladen"
                    >
                      <Download className="h-4 w-4" />
                    </button>
                    {doc.uploadedBy === 'self' && (
                      <button
                        onClick={() => toast.success('Dokument gelöscht')}
                        className="p-1.5 rounded hover:bg-destructive/10 text-muted-foreground hover:text-destructive transition-colors"
                        title="Löschen"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

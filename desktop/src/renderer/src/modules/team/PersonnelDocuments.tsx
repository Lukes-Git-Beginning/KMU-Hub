import { useState, useMemo } from 'react'
import {
  FileText,
  Upload,
  Download,
  Search,
  Clock,
  AlertTriangle,
  CheckCircle2,
  FolderOpen,
  Eye,
  File,
  FileCheck,
  FileLock2,
} from 'lucide-react'
import { toast } from 'sonner'
import { EmptyState } from '@/components/shared'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'

// ============================================================
// Types
// ============================================================

type DocCategory = 'vertrag' | 'zeugnis' | 'zertifikat' | 'ausweis' | 'sonstiges'
type DocStatus = 'aktuell' | 'bald_ablaufend' | 'abgelaufen'

interface PersonnelDocument {
  id: string
  employeeId: string
  employeeName: string
  title: string
  category: DocCategory
  fileName: string
  fileSize: string
  uploadedAt: string
  uploadedBy: string
  expiresAt?: string
  status: DocStatus
  notes?: string
}

// ============================================================
// Constants
// ============================================================

const categoryLabels: Record<DocCategory, string> = {
  vertrag: 'Arbeitsvertrag',
  zeugnis: 'Zeugnis',
  zertifikat: 'Zertifikat',
  ausweis: 'Ausweis',
  sonstiges: 'Sonstiges',
}

const categoryIcons: Record<DocCategory, typeof FileText> = {
  vertrag: FileLock2,
  zeugnis: FileCheck,
  zertifikat: FileCheck,
  ausweis: File,
  sonstiges: FileText,
}

const categoryColors: Record<DocCategory, string> = {
  vertrag: 'bg-primary-light text-primary',
  zeugnis: 'bg-info-light text-info',
  zertifikat: 'bg-success-light text-success',
  ausweis: 'bg-warning-light text-warning',
  sonstiges: 'bg-secondary text-muted-foreground',
}

const statusConfig: Record<DocStatus, { label: string; color: string; icon: typeof CheckCircle2 }> = {
  aktuell: { label: 'Aktuell', color: 'text-success', icon: CheckCircle2 },
  bald_ablaufend: { label: 'Bald ablaufend', color: 'text-warning', icon: Clock },
  abgelaufen: { label: 'Abgelaufen', color: 'text-error', icon: AlertTriangle },
}

// ============================================================
// Mock Data
// ============================================================

const MOCK_DOCUMENTS: PersonnelDocument[] = [
  { id: 'doc-1', employeeId: 'm1', employeeName: 'Anna Müller', title: 'Arbeitsvertrag', category: 'vertrag', fileName: 'AV_Müller_2024.pdf', fileSize: '245 KB', uploadedAt: '2024-03-15', uploadedBy: 'Laura Weber', status: 'aktuell' },
  { id: 'doc-2', employeeId: 'm1', employeeName: 'Anna Müller', title: 'Zwischenzeugnis 2025', category: 'zeugnis', fileName: 'ZZ_Müller_2025.pdf', fileSize: '180 KB', uploadedAt: '2025-12-20', uploadedBy: 'Laura Weber', status: 'aktuell' },
  { id: 'doc-3', employeeId: 'm2', employeeName: 'Michael Berg', title: 'Arbeitsvertrag', category: 'vertrag', fileName: 'AV_Berg_2024.pdf', fileSize: '260 KB', uploadedAt: '2024-01-10', uploadedBy: 'Laura Weber', status: 'aktuell' },
  { id: 'doc-4', employeeId: 'm2', employeeName: 'Michael Berg', title: 'AWS Solutions Architect', category: 'zertifikat', fileName: 'Cert_AWS_Berg.pdf', fileSize: '120 KB', uploadedAt: '2025-06-15', uploadedBy: 'Michael Berg', expiresAt: '2028-06-15', status: 'aktuell' },
  { id: 'doc-5', employeeId: 'm3', employeeName: 'Sarah Klein', title: 'Arbeitsvertrag (Teilzeit)', category: 'vertrag', fileName: 'AV_Klein_2024.pdf', fileSize: '230 KB', uploadedAt: '2024-06-01', uploadedBy: 'Laura Weber', status: 'aktuell' },
  { id: 'doc-6', employeeId: 'm4', employeeName: 'Jonas Diaz', title: 'React Certification', category: 'zertifikat', fileName: 'Cert_React_Diaz.pdf', fileSize: '95 KB', uploadedAt: '2025-03-10', uploadedBy: 'Jonas Diaz', expiresAt: '2026-03-10', status: 'bald_ablaufend', notes: 'Erneuerung im März fällig' },
  { id: 'doc-7', employeeId: 'm6', employeeName: 'Peter Koch', title: 'OSCP Zertifikat', category: 'zertifikat', fileName: 'Cert_OSCP_Koch.pdf', fileSize: '110 KB', uploadedAt: '2024-08-20', uploadedBy: 'Peter Koch', expiresAt: '2025-08-20', status: 'abgelaufen', notes: 'Zertifikat seit Aug 2025 abgelaufen!' },
  { id: 'doc-8', employeeId: 'm8', employeeName: 'Tom Brunner', title: 'Arbeitsvertrag', category: 'vertrag', fileName: 'AV_Brunner_2025.pdf', fileSize: '240 KB', uploadedAt: '2025-06-01', uploadedBy: 'Laura Weber', status: 'aktuell' },
  { id: 'doc-9', employeeId: 'm10', employeeName: 'Markus Steiner', title: 'Fuehrerschein Kopie', category: 'ausweis', fileName: 'FS_Steiner.pdf', fileSize: '85 KB', uploadedAt: '2025-01-15', uploadedBy: 'Markus Steiner', expiresAt: '2030-01-15', status: 'aktuell' },
  { id: 'doc-10', employeeId: 'm11', employeeName: 'Laura Weber', title: 'Arbeitsvertrag', category: 'vertrag', fileName: 'AV_Weber_2024.pdf', fileSize: '255 KB', uploadedAt: '2024-05-15', uploadedBy: 'Anna Müller', status: 'aktuell' },
  { id: 'doc-11', employeeId: 'm5', employeeName: 'Lisa Schmidt', title: 'Personalausweis Kopie', category: 'ausweis', fileName: 'PA_Schmidt.pdf', fileSize: '72 KB', uploadedAt: '2024-09-10', uploadedBy: 'Lisa Schmidt', expiresAt: '2026-04-15', status: 'bald_ablaufend' },
  { id: 'doc-12', employeeId: 'm12', employeeName: 'Nils Hofer', title: 'Kubernetes CKA', category: 'zertifikat', fileName: 'Cert_CKA_Hofer.pdf', fileSize: '105 KB', uploadedAt: '2025-09-01', uploadedBy: 'Nils Hofer', expiresAt: '2028-09-01', status: 'aktuell' },
]

// ============================================================
// Component
// ============================================================

export function PersonnelDocuments() {
  const [search, setSearch] = useState('')
  const [categoryFilter, setCategoryFilter] = useState<DocCategory | 'all'>('all')
  const [statusFilter, setStatusFilter] = useState<DocStatus | 'all'>('all')
  const [showUpload, setShowUpload] = useState(false)

  const filteredDocs = useMemo(() => {
    return MOCK_DOCUMENTS.filter((doc) => {
      if (categoryFilter !== 'all' && doc.category !== categoryFilter) return false
      if (statusFilter !== 'all' && doc.status !== statusFilter) return false
      if (search) {
        const q = search.toLowerCase()
        return doc.title.toLowerCase().includes(q) || doc.employeeName.toLowerCase().includes(q) || doc.fileName.toLowerCase().includes(q)
      }
      return true
    })
  }, [search, categoryFilter, statusFilter])

  const docsByEmployee = useMemo(() => {
    const map = new Map<string, PersonnelDocument[]>()
    for (const doc of filteredDocs) {
      const key = doc.employeeName
      if (!map.has(key)) map.set(key, [])
      map.get(key)!.push(doc)
    }
    return [...map.entries()].sort(([a], [b]) => a.localeCompare(b))
  }, [filteredDocs])

  const totalDocs = MOCK_DOCUMENTS.length
  const expiredCount = MOCK_DOCUMENTS.filter((d) => d.status === 'abgelaufen').length
  const expiringCount = MOCK_DOCUMENTS.filter((d) => d.status === 'bald_ablaufend').length

  const handleDownload = (doc: PersonnelDocument) => {
    toast.success(`Download gestartet: ${doc.fileName}`)
  }

  const handleUpload = () => {
    toast.success('Dokument hochgeladen (Mock)')
    setShowUpload(false)
  }

  return (
    <div className="space-y-4">
      {/* KPI Row */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <div className="rounded-lg border border-border bg-card p-4">
          <p className="text-xs text-muted-foreground mb-1">Dokumente gesamt</p>
          <p className="text-lg font-semibold text-foreground">{totalDocs}</p>
        </div>
        <div className="rounded-lg border border-border bg-card p-4">
          <p className="text-xs text-muted-foreground mb-1">Mitarbeiter</p>
          <p className="text-lg font-semibold text-foreground">{new Set(MOCK_DOCUMENTS.map(d => d.employeeId)).size}</p>
        </div>
        <div className="rounded-lg border border-border bg-card p-4">
          <p className="text-xs text-muted-foreground mb-1">Bald ablaufend</p>
          <div className="flex items-center gap-2">
            <p className="text-lg font-semibold text-warning">{expiringCount}</p>
            {expiringCount > 0 && <Clock className="h-4 w-4 text-warning" />}
          </div>
        </div>
        <div className="rounded-lg border border-border bg-card p-4">
          <p className="text-xs text-muted-foreground mb-1">Abgelaufen</p>
          <div className="flex items-center gap-2">
            <p className="text-lg font-semibold text-error">{expiredCount}</p>
            {expiredCount > 0 && <AlertTriangle className="h-4 w-4 text-error" />}
          </div>
        </div>
      </div>

      {/* Toolbar */}
      <div className="flex flex-wrap items-center gap-3">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <input
            type="text"
            placeholder="Dokument oder Mitarbeiter suchen..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
          />
        </div>

        <select
          value={categoryFilter}
          onChange={(e) => setCategoryFilter(e.target.value as DocCategory | 'all')}
          className="rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
        >
          <option value="all">Alle Kategorien</option>
          {Object.entries(categoryLabels).map(([k, v]) => <option key={k} value={k}>{v}</option>)}
        </select>

        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value as DocStatus | 'all')}
          className="rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
        >
          <option value="all">Alle Status</option>
          <option value="aktuell">Aktuell</option>
          <option value="bald_ablaufend">Bald ablaufend</option>
          <option value="abgelaufen">Abgelaufen</option>
        </select>

        <button
          onClick={() => setShowUpload(true)}
          className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
        >
          <Upload className="h-4 w-4" />
          Hochladen
        </button>
      </div>

      {/* Document List grouped by Employee */}
      {docsByEmployee.length === 0 ? (
        <EmptyState
          icon={FolderOpen}
          title="Keine Dokumente gefunden"
          description={search || categoryFilter !== 'all' ? 'Passe deine Filter an' : 'Lade Personaldokumente hoch, um die digitale Akte zu starten'}
        />
      ) : (
        <div className="space-y-4">
          {docsByEmployee.map(([name, docs]) => (
            <div key={name} className="rounded-lg border border-border bg-card overflow-hidden">
              <div className="flex items-center gap-2 px-4 py-2.5 border-b border-border bg-secondary/30">
                <FolderOpen className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm font-medium text-foreground">{name}</span>
                <span className="text-xs text-muted-foreground">({docs.length} Dokumente)</span>
              </div>
              <div className="divide-y divide-border-muted">
                {docs.map((doc) => {
                  const CatIcon = categoryIcons[doc.category]
                  const st = statusConfig[doc.status]
                  const StIcon = st.icon
                  return (
                    <div key={doc.id} className="flex items-center gap-3 px-4 py-3 hover:bg-secondary/20 transition-colors">
                      <div className={`flex h-9 w-9 items-center justify-center rounded-lg ${categoryColors[doc.category]}`}>
                        <CatIcon className="h-4 w-4" />
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="text-sm font-medium text-foreground truncate">{doc.title}</span>
                          <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${categoryColors[doc.category]}`}>
                            {categoryLabels[doc.category]}
                          </span>
                        </div>
                        <div className="flex items-center gap-3 text-xs text-muted-foreground mt-0.5">
                          <span>{doc.fileName} ({doc.fileSize})</span>
                          <span>Hochgeladen: {new Date(doc.uploadedAt).toLocaleDateString('de-DE')}</span>
                          {doc.expiresAt && (
                            <span className={`flex items-center gap-1 ${st.color}`}>
                              <StIcon className="h-3 w-3" />
                              Ablauf: {new Date(doc.expiresAt).toLocaleDateString('de-DE')}
                            </span>
                          )}
                        </div>
                        {doc.notes && (
                          <p className="text-xs text-warning mt-1">{doc.notes}</p>
                        )}
                      </div>
                      <div className="flex items-center gap-1 flex-shrink-0">
                        <button
                          onClick={() => toast.info(`Vorschau: ${doc.fileName}`)}
                          className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary transition-colors"
                          title="Vorschau"
                        >
                          <Eye className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => handleDownload(doc)}
                          className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary transition-colors"
                          title="Download"
                        >
                          <Download className="h-4 w-4" />
                        </button>
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Upload Dialog */}
      <Dialog open={showUpload} onOpenChange={(o) => { if (!o) setShowUpload(false) }}>
        <DialogContent className="gap-0 p-0 max-w-md">
          <div className="p-6">
            <DialogHeader className="mb-4">
              <DialogTitle className="text-base font-semibold text-foreground">Dokument hochladen</DialogTitle>
              <DialogDescription className="sr-only">Personaldokument hochladen</DialogDescription>
            </DialogHeader>
            <div className="space-y-4">
              <div className="rounded-lg border-2 border-dashed border-border bg-secondary/20 p-8 text-center">
                <Upload className="h-8 w-8 text-muted-foreground mx-auto mb-2" />
                <p className="text-sm text-muted-foreground">Datei hierher ziehen oder klicken</p>
                <p className="text-xs text-muted-foreground mt-1">PDF, DOC, DOCX, JPG (max. 10 MB)</p>
              </div>
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5">Kategorie</label>
                <select className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring">
                  {Object.entries(categoryLabels).map(([k, v]) => <option key={k} value={k}>{v}</option>)}
                </select>
              </div>
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5">Ablaufdatum (optional)</label>
                <input type="date" className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring" />
              </div>
            </div>
            <DialogFooter className="mt-6">
              <button onClick={() => setShowUpload(false)} className="rounded-lg border border-border px-4 py-2 text-sm text-foreground hover:bg-secondary transition-colors">
                Abbrechen
              </button>
              <button onClick={handleUpload} className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors">
                Hochladen
              </button>
            </DialogFooter>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}

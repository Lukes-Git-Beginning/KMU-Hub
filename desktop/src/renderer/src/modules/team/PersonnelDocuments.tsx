import { useState, useMemo, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { useQuery, useQueryClient } from '@tanstack/react-query'
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
  X,
} from 'lucide-react'
import { toast } from 'sonner'
import { EmptyState, InlineStat } from '@/components/shared'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { formatDate } from '@/lib/format'
import { API_BASE_URL } from '@/lib/constants'

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
// Component
// ============================================================

export function PersonnelDocuments() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [search, setSearch] = useState('')
  const [categoryFilter, setCategoryFilter] = useState<DocCategory | 'all'>('all')
  const [statusFilter, setStatusFilter] = useState<DocStatus | 'all'>('all')
  const [showUpload, setShowUpload] = useState(false)
  const [previewDoc, setPreviewDoc] = useState<PersonnelDocument | null>(null)
  const [uploadFile, setUploadFile] = useState<File | null>(null)
  const [uploadCategory, setUploadCategory] = useState<DocCategory>('vertrag')
  const [uploadExpiry, setUploadExpiry] = useState('')
  const [uploading, setUploading] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const { data: documents = [] } = useQuery({
    queryKey: ['hr', 'personnel-documents'],
    queryFn: async () => {
      const res = await fetch(`${API_BASE_URL}/api/v1/hr/personnel-documents`)
      const json = (await res.json()) as { documents: PersonnelDocument[] }
      return json.documents
    },
  })

  const filteredDocs = useMemo(() => {
    return documents.filter((doc) => {
      if (categoryFilter !== 'all' && doc.category !== categoryFilter) return false
      if (statusFilter !== 'all' && doc.status !== statusFilter) return false
      if (search) {
        const q = search.toLowerCase()
        return doc.title.toLowerCase().includes(q) || doc.employeeName.toLowerCase().includes(q) || doc.fileName.toLowerCase().includes(q)
      }
      return true
    })
  }, [documents, search, categoryFilter, statusFilter])

  const docsByEmployee = useMemo(() => {
    const map = new Map<string, PersonnelDocument[]>()
    for (const doc of filteredDocs) {
      const key = doc.employeeName
      if (!map.has(key)) map.set(key, [])
      map.get(key)!.push(doc)
    }
    return [...map.entries()].sort(([a], [b]) => a.localeCompare(b))
  }, [filteredDocs])

  const totalDocs = documents.length
  const expiredCount = documents.filter((d) => d.status === 'abgelaufen').length
  const expiringCount = documents.filter((d) => d.status === 'bald_ablaufend').length

  // Download a personnel document as a real (demo) blob.
  const handleDownload = (doc: PersonnelDocument) => {
    const content =
      `${doc.title}\n${doc.employeeName}\n` +
      `${t('team.documents.category')}: ${categoryLabels[doc.category]}\n` +
      `${t('team.personnelDocs.uploaded')}: ${doc.uploadedAt} · ${doc.uploadedBy}\n` +
      `${doc.expiresAt ? `${t('team.personnelDocs.expires')}: ${doc.expiresAt}\n` : ''}` +
      `\n[Cosmi — ${t('team.personnelDocs.title', { defaultValue: 'Personalakte' })} (Demo)]\n`
    const blob = new Blob([content], { type: 'text/plain;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = doc.fileName.replace(/\.pdf$/i, '.txt')
    a.click()
    URL.revokeObjectURL(url)
    toast.success(t('team.personnelDocs.downloadStarted', { name: doc.fileName }))
  }

  const handleUpload = async () => {
    setUploading(true)
    try {
      const sizeKb = uploadFile ? Math.max(1, Math.round(uploadFile.size / 1024)) : 120
      await fetch(`${API_BASE_URL}/api/v1/hr/personnel-documents`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          title: uploadFile ? uploadFile.name.replace(/\.[^.]+$/, '') : t('team.personnelDocs.uploadDocument'),
          category: uploadCategory,
          fileName: uploadFile?.name ?? 'dokument.pdf',
          fileSize: `${sizeKb} KB`,
          expiresAt: uploadExpiry || undefined,
        }),
      })
      await qc.invalidateQueries({ queryKey: ['hr', 'personnel-documents'] })
      toast.success(t('team.personnelDocs.uploadSuccess'))
      setShowUpload(false)
      setUploadFile(null)
      setUploadExpiry('')
    } catch {
      toast.error(t('team.personnelDocs.uploadError', { defaultValue: 'Upload fehlgeschlagen' }))
    } finally {
      setUploading(false)
    }
  }

  return (
    <div className="space-y-4">
      {/* KPI Row */}
      <dl className="mb-4 flex flex-wrap items-end gap-x-10 gap-y-3 border-b border-border pb-4">
        <InlineStat label={t('team.personnelDocs.totalDocuments')} value={totalDocs} />
        <InlineStat label={t('team.personnelDocs.employees')} value={new Set(documents.map((d) => d.employeeId)).size} />
        <InlineStat
          label={t('team.personnelDocs.expiringSoon')}
          value={expiringCount}
          accent={expiringCount > 0 ? 'warning' : 'default'}
          icon={expiringCount > 0 ? Clock : undefined}
        />
        <InlineStat
          label={t('team.personnelDocs.expired')}
          value={expiredCount}
          accent={expiredCount > 0 ? 'error' : 'default'}
          icon={expiredCount > 0 ? AlertTriangle : undefined}
        />
      </dl>

      {/* Toolbar */}
      <div className="flex flex-wrap items-center gap-3">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <input
            type="text"
            placeholder={t('team.personnelDocs.searchPlaceholder')}
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
          <option value="all">{t('team.personnelDocs.allCategories')}</option>
          {Object.entries(categoryLabels).map(([k, v]) => <option key={k} value={k}>{v}</option>)}
        </select>

        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value as DocStatus | 'all')}
          className="rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
        >
          <option value="all">{t('team.personnelDocs.allStatuses')}</option>
          <option value="aktuell">{t('team.personnelDocs.statusCurrent')}</option>
          <option value="bald_ablaufend">{t('team.personnelDocs.statusExpiringSoon')}</option>
          <option value="abgelaufen">{t('team.personnelDocs.statusExpired')}</option>
        </select>

        <button
          onClick={() => setShowUpload(true)}
          className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
        >
          <Upload className="h-4 w-4" />
          {t('common.upload')}
        </button>
      </div>

      {/* Document List grouped by Employee */}
      {docsByEmployee.length === 0 ? (
        <EmptyState
          icon={FolderOpen}
          title={t('team.personnelDocs.noDocuments')}
          description={search || categoryFilter !== 'all' ? t('team.personnelDocs.adjustFilters') : t('team.personnelDocs.uploadToStart')}
        />
      ) : (
        <div className="space-y-4">
          {docsByEmployee.map(([name, docs]) => (
            <div key={name} className="rounded-lg border border-border bg-card overflow-hidden">
              <div className="flex items-center gap-2 px-4 py-2.5 border-b border-border bg-secondary/30">
                <FolderOpen className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm font-medium text-foreground">{name}</span>
                <span className="text-xs text-muted-foreground">({docs.length} {t('team.personnelDocs.documentsCount')})</span>
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
                          <span>{t('team.personnelDocs.uploaded')}: {formatDate(doc.uploadedAt)}</span>
                          {doc.expiresAt && (
                            <span className={`flex items-center gap-1 ${st.color}`}>
                              <StIcon className="h-3 w-3" />
                              {t('team.personnelDocs.expires')}: {formatDate(doc.expiresAt)}
                            </span>
                          )}
                        </div>
                        {doc.notes && (
                          <p className="text-xs text-warning mt-1">{doc.notes}</p>
                        )}
                      </div>
                      <div className="flex items-center gap-1 flex-shrink-0">
                        <button
                          onClick={() => setPreviewDoc(doc)}
                          className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary transition-colors"
                          title={t('team.personnelDocs.preview')}
                        >
                          <Eye className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => handleDownload(doc)}
                          className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary transition-colors"
                          title={t('team.personnelDocs.download', { defaultValue: 'Download' })}
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

      {/* Preview Dialog */}
      <Dialog open={!!previewDoc} onOpenChange={(o) => { if (!o) setPreviewDoc(null) }}>
        <DialogContent className="max-w-lg gap-0 p-0">
          {previewDoc && (
            <div className="flex flex-col">
              <div className="flex items-center justify-between border-b border-border px-5 py-3">
                <div className="flex items-center gap-2 min-w-0">
                  <span className="rounded-full bg-primary-light px-2 py-0.5 text-[10px] font-medium text-primary">{t('team.personnelDocs.previewBadge', { defaultValue: 'Demo-Vorschau' })}</span>
                  <span className="text-sm font-medium text-foreground truncate">{previewDoc.fileName}</span>
                </div>
                <button onClick={() => setPreviewDoc(null)} className="rounded-md p-1 text-muted-foreground hover:bg-secondary transition-colors">
                  <X className="h-4 w-4" />
                </button>
              </div>
              <div className="px-6 py-8">
                <div className="mx-auto max-w-sm rounded-lg border border-border bg-secondary/20 p-8">
                  <div className={`mb-4 flex h-12 w-12 items-center justify-center rounded-lg ${categoryColors[previewDoc.category]}`}>
                    {(() => { const I = categoryIcons[previewDoc.category]; return <I className="h-6 w-6" /> })()}
                  </div>
                  <h2 className="text-lg font-semibold text-foreground">{previewDoc.title}</h2>
                  <p className="text-sm text-muted-foreground">{previewDoc.employeeName} · {categoryLabels[previewDoc.category]}</p>
                  <dl className="mt-4 space-y-1.5 text-xs">
                    <div className="flex justify-between"><dt className="text-muted-foreground">{t('team.personnelDocs.uploaded')}</dt><dd className="text-foreground">{formatDate(previewDoc.uploadedAt)} · {previewDoc.uploadedBy}</dd></div>
                    {previewDoc.expiresAt && <div className="flex justify-between"><dt className="text-muted-foreground">{t('team.personnelDocs.expires')}</dt><dd className={statusConfig[previewDoc.status].color}>{formatDate(previewDoc.expiresAt)}</dd></div>}
                    <div className="flex justify-between"><dt className="text-muted-foreground">{t('team.personnelDocs.fileSize', { defaultValue: 'Größe' })}</dt><dd className="text-foreground">{previewDoc.fileSize}</dd></div>
                  </dl>
                  <p className="mt-5 border-t border-border pt-4 text-xs text-muted-foreground">{t('team.personnelDocs.previewHint', { defaultValue: 'Im Produktivbetrieb wird hier das hinterlegte PDF angezeigt.' })}</p>
                </div>
              </div>
              <div className="flex justify-end border-t border-border px-5 py-3">
                <button onClick={() => { handleDownload(previewDoc); }} className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors">
                  <Download className="h-3.5 w-3.5" />
                  {t('team.personnelDocs.download', { defaultValue: 'Download' })}
                </button>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>

      {/* Upload Dialog */}
      <Dialog open={showUpload} onOpenChange={(o) => { if (!o) { setShowUpload(false); setUploadFile(null) } }}>
        <DialogContent className="gap-0 p-0 max-w-md">
          <div className="p-6">
            <DialogHeader className="mb-4">
              <DialogTitle className="text-base font-semibold text-foreground">{t('team.personnelDocs.uploadDocument')}</DialogTitle>
              <DialogDescription className="sr-only">{t('team.personnelDocs.uploadDocument')}</DialogDescription>
            </DialogHeader>
            <div className="space-y-4">
              <input
                ref={fileInputRef}
                type="file"
                accept=".pdf,.png,.jpg,.jpeg,.docx"
                className="hidden"
                onChange={(e) => setUploadFile(e.target.files?.[0] ?? null)}
              />
              <button
                type="button"
                onClick={() => fileInputRef.current?.click()}
                className="w-full rounded-lg border-2 border-dashed border-border bg-secondary/20 p-8 text-center hover:border-primary/50 transition-colors"
              >
                <Upload className="h-8 w-8 text-muted-foreground mx-auto mb-2" />
                {uploadFile ? (
                  <p className="text-sm font-medium text-foreground">{uploadFile.name}</p>
                ) : (
                  <>
                    <p className="text-sm text-muted-foreground">{t('team.personnelDocs.dropOrClick')}</p>
                    <p className="text-xs text-muted-foreground mt-1">{t('team.personnelDocs.allowedFormats')}</p>
                  </>
                )}
              </button>
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5">{t('team.documents.category')}</label>
                <select
                  value={uploadCategory}
                  onChange={(e) => setUploadCategory(e.target.value as DocCategory)}
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                >
                  {Object.entries(categoryLabels).map(([k, v]) => <option key={k} value={k}>{v}</option>)}
                </select>
              </div>
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5">{t('team.personnelDocs.expiryDate')}</label>
                <input
                  type="date"
                  value={uploadExpiry}
                  onChange={(e) => setUploadExpiry(e.target.value)}
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                />
              </div>
            </div>
            <DialogFooter className="mt-6">
              <button onClick={() => { setShowUpload(false); setUploadFile(null) }} className="rounded-lg border border-border px-4 py-2 text-sm text-foreground hover:bg-secondary transition-colors">
                {t('common.cancel')}
              </button>
              <button onClick={handleUpload} disabled={uploading} className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-60">
                {t('common.upload')}
              </button>
            </DialogFooter>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}

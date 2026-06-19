import { useState, useMemo, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import {
  FileText, Upload, Download, FolderOpen, Search, Loader2,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter,
} from '@/components/ui/dialog'
import { DetailModal } from '@/components/shared/DetailModal'
import { cn } from '@/lib/cn'
import { toast } from 'sonner'
import {
  useEmployeeDocuments,
  useDocumentCategories,
  useUploadEmployeeDocument,
  useSelfProfile,
} from '@/api/hooks/hr-hooks'
import type { EmployeeDocument } from '@/api/hr-types'

export default function DokumenteTab() {
  const { t } = useTranslation()
  const [activeCategory, setActiveCategory] = useState<string>('all')
  const [searchQuery, setSearchQuery] = useState('')
  const [previewDoc, setPreviewDoc] = useState<EmployeeDocument | null>(null)

  // Upload dialog state
  const [showUpload, setShowUpload] = useState(false)
  const [uploadFile, setUploadFile] = useState<File | null>(null)
  const [uploadCategoryId, setUploadCategoryId] = useState('')
  const [uploadNotes, setUploadNotes] = useState('')
  const fileInputRef = useRef<HTMLInputElement>(null)

  // Get current employee profile for the employee ID
  const { data: selfProfile } = useSelfProfile()
  const employeeId = selfProfile?.id ?? ''

  // TanStack Query hooks
  const { data: documents, isLoading: docsLoading } = useEmployeeDocuments(employeeId)
  const { data: categories } = useDocumentCategories(employeeId)
  const uploadMutation = useUploadEmployeeDocument()

  const allDocuments = useMemo(() => documents ?? [], [documents])
  const allCategories = categories ?? []

  // Filter and search
  const filtered = useMemo(() => {
    return allDocuments
      .filter((d) => activeCategory === 'all' || d.categoryId === activeCategory)
      .filter((d) => {
        if (!searchQuery) return true
        const q = searchQuery.toLowerCase()
        return (
          (d.fileName ?? '').toLowerCase().includes(q) ||
          (d.categoryName ?? '').toLowerCase().includes(q) ||
          (d.notes ?? '').toLowerCase().includes(q)
        )
      })
      .sort((a, b) => b.createdAt.localeCompare(a.createdAt))
  }, [allDocuments, activeCategory, searchQuery])

  // Category counts
  const categoryCounts = useMemo(() => {
    const counts = new Map<string, number>()
    for (const doc of allDocuments) {
      counts.set(doc.categoryId, (counts.get(doc.categoryId) ?? 0) + 1)
    }
    return counts
  }, [allDocuments])

  const formatDate = (dateStr: string) => {
    const d = new Date(dateStr)
    return `${d.getDate().toString().padStart(2, '0')}.${(d.getMonth() + 1).toString().padStart(2, '0')}.${d.getFullYear()}`
  }

  const getFileExtension = (fileName: string): string => {
    const parts = fileName.split('.')
    return parts.length > 1 ? parts[parts.length - 1].toLowerCase() : ''
  }

  const getTypeColor = (ext: string) => {
    switch (ext) {
      case 'pdf': return 'text-destructive bg-error-light'
      case 'docx': case 'doc': return 'text-blue-600 dark:text-blue-400 bg-blue-100 dark:bg-blue-900/30'
      case 'xlsx': case 'xls': return 'text-success bg-success-light'
      case 'png': case 'jpg': case 'jpeg': return 'text-purple-600 dark:text-purple-400 bg-purple-100 dark:bg-purple-900/30'
      default: return 'text-gray-600 dark:text-gray-400 bg-gray-100 dark:bg-gray-900/30'
    }
  }

  // Download a document as a real (demo) text blob — the production backend will
  // stream the stored file; here we synthesise a placeholder from the metadata.
  const handleDownload = (doc: EmployeeDocument) => {
    const lines = [
      doc.fileName ?? t('profil.documents.unnamed'),
      doc.categoryName ?? '',
      `${t('profil.documents.uploaded')}: ${formatDate(doc.createdAt)}${doc.uploaderName ? ` · ${doc.uploaderName}` : ''}`,
      doc.fileSize ? `${t('profil.documents.fileSize')}: ${doc.fileSize}` : '',
      doc.notes ? `\n${doc.notes}` : '',
      `\n[Cosmi — ${t('profil.tabs.documents')} (Demo)]`,
    ].filter(Boolean)
    const blob = new Blob([lines.join('\n')], { type: 'text/plain;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = (doc.fileName ?? 'dokument.pdf').replace(/\.[^.]+$/i, '.txt')
    a.click()
    URL.revokeObjectURL(url)
    toast.success(t('profil.documents.downloading', { name: doc.fileName ?? '' }))
  }

  const openUpload = () => {
    setUploadFile(null)
    setUploadNotes('')
    setUploadCategoryId(allCategories[0]?.id ?? '')
    setShowUpload(true)
  }

  const handleUploadSubmit = () => {
    if (!uploadFile) {
      toast.error(t('profil.documents.selectFileFirst'))
      return
    }
    const sizeKb = Math.max(1, Math.round(uploadFile.size / 1024))
    uploadMutation.mutate(
      {
        employeeId,
        data: {
          categoryId: uploadCategoryId || allCategories[0]?.id || 'hrcat-other',
          fileId: `file-${Date.now()}`,
          fileName: uploadFile.name,
          fileSize: `${sizeKb} KB`,
          notes: uploadNotes.trim() || undefined,
        },
      },
      {
        onSuccess: () => {
          setShowUpload(false)
          setUploadFile(null)
          setUploadNotes('')
        },
      },
    )
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
          <span className="flex-1 text-left">{t('profil.documents.allDocuments')}</span>
          <span className="text-xs">{allDocuments.length}</span>
        </button>

        <div className="pt-2 pb-1 px-3">
          <span className="text-[10px] font-semibold text-muted-foreground uppercase tracking-wider">{t('profil.documents.categories')}</span>
        </div>

        {allCategories.map((cat) => (
          <button
            key={cat.id}
            onClick={() => setActiveCategory(cat.id)}
            className={cn(
              'w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm transition-colors',
              activeCategory === cat.id
                ? 'bg-primary/10 text-primary font-medium'
                : 'text-muted-foreground hover:bg-secondary hover:text-foreground',
            )}
          >
            <FolderOpen className="h-4 w-4 shrink-0" />
            <span className="flex-1 text-left truncate">{cat.name}</span>
            <span className="text-xs">{categoryCounts.get(cat.id) ?? 0}</span>
          </button>
        ))}

        {/* Visibility legend */}
        <div className="pt-4 mt-4 border-t border-border px-3">
          <p className="text-[10px] text-muted-foreground">
            {t('profil.documents.visibilityNote')}
          </p>
        </div>
      </div>

      {/* Main Content */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Toolbar */}
        <div className="flex items-center gap-3 px-6 py-3 border-b border-border">
          <div className="relative flex-1 max-w-sm">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder={t('profil.documents.searchPlaceholder')}
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
              onClick={openUpload}
            >
              <Upload className="h-4 w-4" />
              {t('common.upload')}
            </Button>
          </div>
        </div>

        {/* Category Header */}
        {activeCategory !== 'all' && (() => {
          const cat = allCategories.find((c) => c.id === activeCategory)
          if (!cat) return null
          return (
            <div className="px-6 py-3 bg-card/30 border-b border-border">
              <div className="flex items-center gap-2">
                <FolderOpen className="h-5 w-5 text-primary" />
                <div>
                  <h3 className="text-sm font-semibold text-foreground">{cat.name}</h3>
                  <p className="text-xs text-muted-foreground">
                    {t('profil.documents.visibility')}: {cat.visibility === 'hr_only' ? t('profil.documents.visibilityHrOnly') : cat.visibility === 'manager' ? t('profil.documents.visibilityManager') : t('profil.documents.visibilityEmployee')}
                  </p>
                </div>
              </div>
            </div>
          )
        })()}

        {/* Document List */}
        <div className="flex-1 overflow-auto p-6">
          {docsLoading ? (
            <div className="flex items-center justify-center py-16">
              <Loader2 className="h-6 w-6 animate-spin text-primary" />
            </div>
          ) : filtered.length === 0 ? (
            <div className="text-center py-16 text-muted-foreground">
              <FileText className="h-12 w-12 mx-auto mb-3 opacity-30" />
              <p className="font-medium">{t('profil.documents.noDocuments')}</p>
              <p className="text-sm mt-1">
                {searchQuery ? t('profil.documents.tryDifferentSearch') : t('profil.documents.noCategoryDocuments')}
              </p>
            </div>
          ) : (
            <div className="space-y-2">
              {filtered.map((doc) => {
                const ext = getFileExtension(doc.fileName ?? '')
                return (
                  <div
                    key={doc.id}
                    role="button"
                    tabIndex={0}
                    onClick={() => setPreviewDoc(doc)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault()
                        setPreviewDoc(doc)
                      }
                    }}
                    aria-label={t('profil.documents.openDocument', { name: doc.fileName ?? t('profil.documents.unnamed') })}
                    className="flex items-center gap-3 p-3 rounded-lg border border-border bg-card hover:border-primary/30 hover:bg-secondary/30 transition-colors group cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  >
                    {/* File type badge */}
                    <span className={cn(
                      'px-2 py-1 rounded text-[10px] font-bold uppercase shrink-0',
                      getTypeColor(ext),
                    )}>
                      {ext || '?'}
                    </span>

                    {/* File info */}
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-foreground truncate">{doc.fileName ?? t('profil.documents.unnamed')}</p>
                      <div className="flex items-center gap-2 text-xs text-muted-foreground mt-0.5">
                        <span>{formatDate(doc.createdAt)}</span>
                        {doc.fileSize && (
                          <>
                            <span className="text-border">|</span>
                            <span>{doc.fileSize}</span>
                          </>
                        )}
                        {doc.categoryName && (
                          <>
                            <span className="text-border">|</span>
                            <span>{doc.categoryName}</span>
                          </>
                        )}
                        {doc.uploaderName && (
                          <>
                            <span className="text-border">|</span>
                            <span className="text-primary">{t('profil.documents.uploadedBy', { name: doc.uploaderName })}</span>
                          </>
                        )}
                      </div>
                      {doc.notes && (
                        <p className="text-xs text-muted-foreground mt-0.5 italic">{doc.notes}</p>
                      )}
                    </div>

                    {/* Actions */}
                    <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-opacity">
                      <button
                        onClick={(e) => { e.stopPropagation(); handleDownload(doc) }}
                        className="p-1.5 rounded hover:bg-secondary text-muted-foreground hover:text-foreground transition-colors"
                        title={t('common.download')}
                        aria-label={t('common.download')}
                      >
                        <Download className="h-4 w-4" />
                      </button>
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </div>
      </div>

      {/* Preview — centered DetailModal (metadata + placeholder preview) */}
      <DetailModal
        open={!!previewDoc}
        onClose={() => setPreviewDoc(null)}
        title={previewDoc?.fileName ?? t('profil.documents.unnamed')}
        subtitle={previewDoc?.categoryName}
        badge={
          <span className="rounded-full bg-primary/10 px-2 py-0.5 text-[10px] font-medium text-primary">
            {t('profil.documents.previewBadge')}
          </span>
        }
        maxWidth="max-w-lg"
        footer={
          previewDoc && (
            <div className="flex justify-end">
              <Button size="sm" variant="outline" className="gap-1.5" onClick={() => handleDownload(previewDoc)}>
                <Download className="h-3.5 w-3.5" />
                {t('common.download')}
              </Button>
            </div>
          )
        }
      >
        {previewDoc && (
          <div className="space-y-5">
            <div className="mx-auto max-w-sm rounded-lg border border-border bg-secondary/20 p-8 text-center">
              <div className={cn(
                'mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-lg',
                getTypeColor(getFileExtension(previewDoc.fileName ?? '')),
              )}>
                <FileText className="h-6 w-6" />
              </div>
              <p className="text-sm font-semibold text-foreground break-all">{previewDoc.fileName}</p>
              <p className="mt-1 text-xs text-muted-foreground">{previewDoc.categoryName}</p>
            </div>
            <dl className="space-y-2 text-xs">
              <div className="flex justify-between gap-4">
                <dt className="text-muted-foreground">{t('profil.documents.uploaded')}</dt>
                <dd className="text-foreground text-right">{formatDate(previewDoc.createdAt)}{previewDoc.uploaderName ? ` · ${previewDoc.uploaderName}` : ''}</dd>
              </div>
              {previewDoc.fileSize && (
                <div className="flex justify-between gap-4">
                  <dt className="text-muted-foreground">{t('profil.documents.fileSize')}</dt>
                  <dd className="text-foreground">{previewDoc.fileSize}</dd>
                </div>
              )}
              {previewDoc.notes && (
                <div className="flex justify-between gap-4">
                  <dt className="text-muted-foreground">{t('profil.documents.note')}</dt>
                  <dd className="text-foreground text-right">{previewDoc.notes}</dd>
                </div>
              )}
            </dl>
            <p className="border-t border-border pt-4 text-xs text-muted-foreground">
              {t('profil.documents.previewHint')}
            </p>
          </div>
        )}
      </DetailModal>

      {/* Upload dialog */}
      <Dialog open={showUpload} onOpenChange={(o) => { if (!o) { setShowUpload(false); setUploadFile(null) } }}>
        <DialogContent className="gap-0 p-0 max-w-md">
          <div className="p-6">
            <DialogHeader className="mb-4">
              <DialogTitle className="text-base font-semibold text-foreground">{t('profil.documents.uploadTitle')}</DialogTitle>
              <DialogDescription className="sr-only">{t('profil.documents.uploadTitle')}</DialogDescription>
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
                  <p className="text-sm font-medium text-foreground break-all">{uploadFile.name}</p>
                ) : (
                  <>
                    <p className="text-sm text-muted-foreground">{t('profil.documents.uploadDropOrClick')}</p>
                    <p className="text-xs text-muted-foreground mt-1">{t('profil.documents.uploadAllowedFormats')}</p>
                  </>
                )}
              </button>
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5">{t('profil.documents.uploadCategory')}</label>
                <select
                  value={uploadCategoryId}
                  onChange={(e) => setUploadCategoryId(e.target.value)}
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                >
                  {allCategories.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
                </select>
              </div>
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5">{t('profil.documents.uploadNotes')}</label>
                <textarea
                  value={uploadNotes}
                  onChange={(e) => setUploadNotes(e.target.value)}
                  rows={2}
                  placeholder={t('profil.documents.uploadNotesPlaceholder')}
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                />
              </div>
            </div>
            <DialogFooter className="mt-6">
              <Button variant="outline" size="sm" onClick={() => { setShowUpload(false); setUploadFile(null) }}>
                {t('common.cancel')}
              </Button>
              <Button size="sm" onClick={handleUploadSubmit} disabled={uploadMutation.isPending}>
                {uploadMutation.isPending && <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />}
                {t('common.upload')}
              </Button>
            </DialogFooter>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}

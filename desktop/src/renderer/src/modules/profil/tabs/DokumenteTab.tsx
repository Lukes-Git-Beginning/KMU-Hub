import { useState, useMemo } from 'react'
import {
  FileText, Upload, Download, Trash2, Eye, FolderOpen, Search, Loader2,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/cn'
import { toast } from 'sonner'
import {
  useEmployeeDocuments,
  useDocumentCategories,
  useUploadEmployeeDocument,
  useSelfProfile,
} from '@/api/hooks/hr-hooks'
import type { EmployeeDocument, HRDocumentCategory } from '@/api/hr-types'

export default function DokumenteTab() {
  const [activeCategory, setActiveCategory] = useState<string>('all')
  const [searchQuery, setSearchQuery] = useState('')

  // Get current employee profile for the employee ID
  const { data: selfProfile } = useSelfProfile()
  const employeeId = selfProfile?.id ?? ''

  // TanStack Query hooks
  const { data: documents, isLoading: docsLoading } = useEmployeeDocuments(employeeId)
  const { data: categories } = useDocumentCategories(employeeId)
  const uploadMutation = useUploadEmployeeDocument()

  const allDocuments = documents ?? []
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
      case 'pdf': return 'text-red-600 dark:text-red-400 bg-red-100 dark:bg-red-900/30'
      case 'docx': case 'doc': return 'text-blue-600 dark:text-blue-400 bg-blue-100 dark:bg-blue-900/30'
      case 'xlsx': case 'xls': return 'text-emerald-600 dark:text-emerald-400 bg-emerald-100 dark:bg-emerald-900/30'
      case 'png': case 'jpg': case 'jpeg': return 'text-purple-600 dark:text-purple-400 bg-purple-100 dark:bg-purple-900/30'
      default: return 'text-gray-600 dark:text-gray-400 bg-gray-100 dark:bg-gray-900/30'
    }
  }

  const handleUpload = () => {
    // Placeholder -- integrate with document service upload flow
    toast.info('Upload-Funktion wird mit dem Dokumenten-Service verbunden')
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
          <span className="text-xs">{allDocuments.length}</span>
        </button>

        <div className="pt-2 pb-1 px-3">
          <span className="text-[10px] font-semibold text-muted-foreground uppercase tracking-wider">Kategorien</span>
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
            Sichtbarkeit wird serverseitig gesteuert. Nur freigegebene Dokumente werden angezeigt.
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
              onClick={handleUpload}
            >
              <Upload className="h-4 w-4" />
              Hochladen
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
                    Sichtbarkeit: {cat.visibility === 'hr_only' ? 'Nur HR' : cat.visibility === 'manager' ? 'Manager' : 'Mitarbeiter'}
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
              <p className="font-medium">Keine Dokumente gefunden</p>
              <p className="text-sm mt-1">
                {searchQuery ? 'Versuche einen anderen Suchbegriff' : 'In dieser Kategorie sind noch keine Dokumente'}
              </p>
            </div>
          ) : (
            <div className="space-y-2">
              {filtered.map((doc) => {
                const ext = getFileExtension(doc.fileName ?? '')
                return (
                  <div
                    key={doc.id}
                    className="flex items-center gap-3 p-3 rounded-lg border border-border bg-card hover:border-primary/30 transition-colors group"
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
                      <p className="text-sm font-medium text-foreground truncate">{doc.fileName ?? 'Unbenannt'}</p>
                      <div className="flex items-center gap-2 text-xs text-muted-foreground mt-0.5">
                        <span>{formatDate(doc.createdAt)}</span>
                        {doc.categoryName && (
                          <>
                            <span className="text-border">|</span>
                            <span>{doc.categoryName}</span>
                          </>
                        )}
                        {doc.uploaderName && (
                          <>
                            <span className="text-border">|</span>
                            <span className="text-primary">Von {doc.uploaderName}</span>
                          </>
                        )}
                      </div>
                      {doc.notes && (
                        <p className="text-xs text-muted-foreground mt-0.5 italic">{doc.notes}</p>
                      )}
                    </div>

                    {/* Actions */}
                    <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                      <button
                        onClick={() => toast.info('Vorschau wird geoeffnet...')}
                        className="p-1.5 rounded hover:bg-secondary text-muted-foreground hover:text-foreground transition-colors"
                        title="Vorschau"
                      >
                        <Eye className="h-4 w-4" />
                      </button>
                      <button
                        onClick={() => toast.success(`${doc.fileName} wird heruntergeladen...`)}
                        className="p-1.5 rounded hover:bg-secondary text-muted-foreground hover:text-foreground transition-colors"
                        title="Herunterladen"
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
    </div>
  )
}

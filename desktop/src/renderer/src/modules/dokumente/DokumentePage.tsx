import { useState, useRef, useMemo } from 'react'
import {
  FileText,
  FolderOpen,
  Upload,
  Search,
  Grid3X3,
  List,
  Star,
  Download,
  ChevronRight,
  ChevronDown,
  Folder,
  Image,
  FileSpreadsheet,
  File,
  Film,
  Archive,
  Lock,
  Share2,
  Eye,
  Trash2,
  Plus,
  Pencil,
  BookOpen,
  ArrowLeft,
  Clock,
  Users,
  Monitor,
  GitBranch,
  Tag,
  Hash,
} from 'lucide-react'
import { toast } from 'sonner'
import { useDocumentsStore, type DocFile, type DocFolder, type WikiArticle } from '@/stores/documents'
import { ItemActions, ConfirmDialog, EmptyState, type ActionItem } from '@/components/shared'
import { FilePreviewModal } from './FilePreviewModal'
import { FileDetailPanel } from './FileDetailPanel'
import { FolderCreateDialog } from './FolderCreateDialog'
import { RenameDialog } from './RenameDialog'
import { ShareDialog } from './ShareDialog'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

type TabKey = 'dateien' | 'wiki'

const fileTypeIcons: Record<string, typeof FileText> = {
  pdf: FileText,
  word: FileText,
  excel: FileSpreadsheet,
  image: Image,
  video: Film,
  archive: Archive,
  other: File,
}

const fileTypeColors: Record<string, string> = {
  pdf: 'bg-file-pdf-light text-file-pdf',
  word: 'bg-file-word-light text-file-word',
  excel: 'bg-file-excel-light text-file-excel',
  image: 'bg-file-image-light text-file-image',
  video: 'bg-file-video-light text-file-video',
  archive: 'bg-file-archive-light text-file-archive',
  other: 'bg-file-other-light text-file-other',
}

const typeLabels: Record<string, string> = {
  pdf: 'PDF',
  word: 'Word',
  excel: 'Excel',
  image: 'Bild',
  video: 'Video',
  archive: 'Archiv',
  other: 'Datei',
}

function formatBytes(bytes: number): string {
  if (bytes >= 1073741824) return `${(bytes / 1073741824).toFixed(1)} GB`
  if (bytes >= 1048576) return `${(bytes / 1048576).toFixed(1)} MB`
  if (bytes >= 1024) return `${(bytes / 1024).toFixed(0)} KB`
  return `${bytes} B`
}

const wikiCategoryIcons: Record<string, typeof BookOpen> = {
  BookOpen: BookOpen,
  Monitor: Monitor,
  Users: Users,
  GitBranch: GitBranch,
  FileText: FileText,
}

export default function DokumentePage() {
  const {
    files, folders,
    addFile, removeFile, renameFile, moveFile, toggleFavorite, toggleShare, updateFileTags,
    addFolder, renameFolder, deleteFolder, totalStorageUsed,
  } = useDocumentsStore()

  const [activeTab, setActiveTab] = useState<TabKey>('dateien')
  const [view, setView] = useState<'grid' | 'list'>('grid')
  const [search, setSearch] = useState('')
  const [activeFolder, setActiveFolder] = useState('root')
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const [isDragOver, setIsDragOver] = useState(false)

  // Dialog/Panel state
  const [previewFile, setPreviewFile] = useState<DocFile | null>(null)
  const [detailFile, setDetailFile] = useState<DocFile | null>(null)
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null)
  const [deleteFolderConfirmId, setDeleteFolderConfirmId] = useState<string | null>(null)
  const [folderCreateOpen, setFolderCreateOpen] = useState(false)
  const [folderCreateParentId, setFolderCreateParentId] = useState<string | null>(null)
  const [renameTarget, setRenameTarget] = useState<{ id: string; name: string; type: 'file' | 'folder' } | null>(null)
  const [shareTarget, setShareTarget] = useState<DocFile | null>(null)
  const [moveTarget, setMoveTarget] = useState<DocFile | null>(null)

  const fileInputRef = useRef<HTMLInputElement>(null)

  const filtered = files.filter((f) => {
    if (search && !f.name.toLowerCase().includes(search.toLowerCase())) return false
    if (activeFolder === 'favorites') return f.isFavorite
    if (activeFolder === 'shared') return f.isShared
    if (activeFolder === 'vault') return f.isVault
    if (activeFolder === 'root') return true
    return f.folderId === activeFolder
  })

  const storageUsed = totalStorageUsed()
  const storageTotal = 10737418240 // 10 GB
  const storagePercent = Math.round((storageUsed / storageTotal) * 100)

  const rootFolders = folders.filter((f) => f.parentId === null)
  const deleteTarget = files.find((f) => f.id === deleteConfirmId)
  const deleteFolderTarget = folders.find((f) => f.id === deleteFolderConfirmId)

  const activeFolderObj = folders.find((f) => f.id === activeFolder)
  const activeFolderName = activeFolderObj?.name || 'Alle Dateien'

  const handleUpload = () => {
    fileInputRef.current?.click()
  }

  const handleFileSelect = () => {
    // Mock file upload
    const mockNames = [
      'Dokument_Neu.pdf', 'Tabelle_Import.xlsx', 'Bild_Upload.png',
      'Präsentation_Draft.pptx', 'Notizen_Meeting.docx',
    ]
    const mockTypes: Record<string, DocFile['type']> = {
      pdf: 'pdf', xlsx: 'excel', png: 'image', pptx: 'other', docx: 'word',
    }
    const name = mockNames[Math.floor(Math.random() * mockNames.length)]
    const ext = name.split('.').pop() || 'pdf'
    const type = mockTypes[ext] || 'other'
    const sizeBytes = Math.floor(Math.random() * 5000000) + 100000

    addFile({
      name,
      type,
      size: formatBytes(sizeBytes),
      sizeBytes,
      date: new Date().toISOString().split('T')[0],
      folderId: activeFolder === 'favorites' || activeFolder === 'shared' || activeFolder === 'vault' ? 'root' : activeFolder,
      tags: [],
      createdBy: 'Du',
      isFavorite: false,
      isShared: false,
      isVault: activeFolder === 'vault' || activeFolder.startsWith('vault-'),
      sharedWith: [],
    })
    toast.success(`"${name}" hochgeladen`)
  }

  const handleDeleteFile = () => {
    if (deleteConfirmId) {
      removeFile(deleteConfirmId)
      toast.success('Datei gelöscht')
      if (detailFile?.id === deleteConfirmId) setDetailFile(null)
      setDeleteConfirmId(null)
    }
  }

  const handleDeleteFolder = () => {
    if (deleteFolderConfirmId) {
      deleteFolder(deleteFolderConfirmId)
      toast.success('Ordner gelöscht')
      if (activeFolder === deleteFolderConfirmId) setActiveFolder('root')
      setDeleteFolderConfirmId(null)
    }
  }

  const handleRenameSubmit = (name: string) => {
    if (!renameTarget) return
    if (renameTarget.type === 'file') {
      renameFile(renameTarget.id, name)
      toast.success('Datei umbenannt')
    } else {
      renameFolder(renameTarget.id, name)
      toast.success('Ordner umbenannt')
    }
  }

  const getFileActions = (f: DocFile): ActionItem[] => [
    {
      label: 'Vorschau',
      icon: Eye,
      onClick: () => setPreviewFile(f),
    },
    {
      label: 'Herunterladen',
      icon: Download,
      onClick: () => toast.success(`"${f.name}" wird heruntergeladen`),
    },
    {
      label: 'Umbenennen',
      icon: Pencil,
      onClick: () => setRenameTarget({ id: f.id, name: f.name, type: 'file' }),
      separator: true,
    },
    {
      label: 'Teilen',
      icon: Share2,
      onClick: () => setShareTarget(f),
    },
    {
      label: 'Verschieben',
      icon: FolderOpen,
      onClick: () => setMoveTarget(f),
    },
    {
      label: f.isFavorite ? 'Aus Favoriten' : 'Favorisieren',
      icon: Star,
      onClick: () => {
        toggleFavorite(f.id)
        toast.success(f.isFavorite ? 'Aus Favoriten entfernt' : 'Zu Favoriten')
      },
    },
    {
      label: 'Löschen',
      icon: Trash2,
      variant: 'destructive',
      onClick: () => setDeleteConfirmId(f.id),
      separator: true,
    },
  ]

  const getFolderActions = (folder: DocFolder): ActionItem[] => {
    if (folder.isSystem) return []
    return [
      {
        label: 'Hochladen hierhin',
        icon: Upload,
        onClick: () => {
          setActiveFolder(folder.id)
          handleUpload()
        },
      },
      {
        label: 'Neuer Unterordner',
        icon: Plus,
        onClick: () => {
          setFolderCreateParentId(folder.id)
          setFolderCreateOpen(true)
        },
      },
      {
        label: 'Umbenennen',
        icon: Pencil,
        onClick: () => setRenameTarget({ id: folder.id, name: folder.name, type: 'folder' }),
        separator: true,
      },
      {
        label: 'Löschen',
        icon: Trash2,
        variant: 'destructive',
        onClick: () => setDeleteFolderConfirmId(folder.id),
        separator: true,
      },
    ]
  }

  return (
    <div className="flex h-full flex-col overflow-hidden">
      {/* Tab bar */}
      <div className="flex items-center gap-4 border-b border-border px-6 pt-3">
        {([
          { key: 'dateien' as const, label: 'Dateien', icon: FileText },
          { key: 'wiki' as const, label: 'Wiki', icon: BookOpen },
        ]).map((t) => {
          const Icon = t.icon
          return (
            <button
              key={t.key}
              onClick={() => setActiveTab(t.key)}
              className={`flex items-center gap-1.5 border-b-2 px-1 pb-2 text-sm whitespace-nowrap transition-colors ${
                activeTab === t.key ? 'border-primary text-primary font-medium' : 'border-transparent text-muted-foreground hover:text-foreground'
              }`}
            >
              <Icon className="h-4 w-4" />
              {t.label}
            </button>
          )
        })}
      </div>

      {/* Dateien tab */}
      {activeTab === 'dateien' && (
        <div className="flex flex-1 overflow-hidden">
          {/* Hidden file input */}
          <input
            ref={fileInputRef}
            type="file"
            className="hidden"
            onChange={handleFileSelect}
            multiple
          />

          {/* Sidebar */}
          {sidebarOpen && (
            <aside className="w-56 shrink-0 border-r border-border bg-card p-4 overflow-y-auto">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-medium text-foreground">Ordner</h3>
                <button
                  onClick={() => {
                    setFolderCreateParentId(null)
                    setFolderCreateOpen(true)
                  }}
                  className="rounded-md p-1 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
                  title="Neuer Ordner"
                >
                  <Plus className="h-4 w-4" />
                </button>
              </div>
              <nav className="space-y-0.5">
                {rootFolders.map((folder) => (
                  <FolderTreeItem
                    key={folder.id}
                    folder={folder}
                    allFolders={folders}
                    activeFolder={activeFolder}
                    onSelect={setActiveFolder}
                    actions={getFolderActions(folder)}
                    depth={0}
                  />
                ))}
              </nav>

              {/* Storage */}
              <div className="mt-6 rounded-lg border border-border p-3">
                <p className="text-xs font-medium text-foreground mb-1">Speicher</p>
                <div className="h-1.5 rounded-full bg-secondary mb-1">
                  <div
                    className="h-full rounded-full bg-primary transition-all"
                    style={{ width: `${Math.min(storagePercent, 100)}%` }}
                  />
                </div>
                <p className="text-[10px] text-muted-foreground">
                  {formatBytes(storageUsed)} von {formatBytes(storageTotal)} verwendet
                </p>
              </div>
            </aside>
          )}

          {/* Main */}
          <div className="flex-1 flex flex-col overflow-hidden">
            {/* Toolbar */}
            <div className="flex items-center gap-3 border-b border-border px-6 py-3">
              <button
                onClick={() => setSidebarOpen(!sidebarOpen)}
                className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary"
              >
                <FolderOpen className="h-4 w-4" />
              </button>
              <div className="relative flex-1 max-w-sm">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <input
                  type="text"
                  placeholder="Dateien suchen..."
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-1.5 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                />
              </div>
              <span className="text-xs text-muted-foreground hidden sm:block">
                {filtered.length} Dateien
              </span>
              <div className="flex items-center gap-1 ml-auto">
                <button
                  onClick={() => setView('grid')}
                  className={`rounded-md p-1.5 transition-colors ${view === 'grid' ? 'bg-secondary text-foreground' : 'text-muted-foreground hover:bg-secondary'}`}
                >
                  <Grid3X3 className="h-4 w-4" />
                </button>
                <button
                  onClick={() => setView('list')}
                  className={`rounded-md p-1.5 transition-colors ${view === 'list' ? 'bg-secondary text-foreground' : 'text-muted-foreground hover:bg-secondary'}`}
                >
                  <List className="h-4 w-4" />
                </button>
              </div>
              <button
                onClick={handleUpload}
                className="flex items-center gap-2 rounded-lg bg-primary px-3 py-1.5 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                <Upload className="h-4 w-4" />
                Hochladen
              </button>
            </div>

            {/* Content with drag overlay */}
            <div
              className="flex-1 overflow-y-auto p-6 relative"
              onDragEnter={(e) => { e.preventDefault(); setIsDragOver(true) }}
              onDragOver={(e) => { e.preventDefault(); setIsDragOver(true) }}
              onDragLeave={(e) => { e.preventDefault(); setIsDragOver(false) }}
              onDrop={(e) => {
                e.preventDefault()
                setIsDragOver(false)
                handleFileSelect()
              }}
            >
              {/* Drag overlay */}
              {isDragOver && (
                <div className="absolute inset-0 z-10 flex items-center justify-center rounded-lg border-2 border-dashed border-primary bg-primary/5 backdrop-blur-sm">
                  <div className="text-center">
                    <Upload className="h-12 w-12 mx-auto mb-3 text-primary opacity-60" />
                    <p className="text-sm font-medium text-primary">Dateien hierhin ziehen</p>
                    <p className="text-xs text-muted-foreground mt-1">in "{activeFolderName}" hochladen</p>
                  </div>
                </div>
              )}

              {view === 'grid' ? (
                <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4">
                  {filtered.map((file) => (
                    <FileGridCard
                      key={file.id}
                      file={file}
                      actions={getFileActions(file)}
                      onDoubleClick={() => setPreviewFile(file)}
                      onClick={() => setDetailFile(file)}
                    />
                  ))}
                </div>
              ) : (
                <div className="space-y-1">
                  <div className="grid grid-cols-[1fr_100px_100px_120px_40px] gap-3 px-3 py-2 text-xs font-medium text-muted-foreground border-b border-border">
                    <span>Name</span>
                    <span>Größe</span>
                    <span>Typ</span>
                    <span>Datum</span>
                    <span />
                  </div>
                  {filtered.map((file) => (
                    <FileListRow
                      key={file.id}
                      file={file}
                      actions={getFileActions(file)}
                      onDoubleClick={() => setPreviewFile(file)}
                      onClick={() => setDetailFile(file)}
                    />
                  ))}
                </div>
              )}

              {filtered.length === 0 && (
                <EmptyState
                  icon={FolderOpen}
                  title="Keine Dateien gefunden"
                  description={search ? 'Versuche einen anderen Suchbegriff' : 'Lade deine erste Datei hoch'}
                  action={
                    !search
                      ? { label: 'Datei hochladen', onClick: handleUpload }
                      : undefined
                  }
                />
              )}
            </div>
          </div>

          {/* Preview Modal */}
          <FilePreviewModal
            file={previewFile}
            open={!!previewFile}
            onOpenChange={(open) => !open && setPreviewFile(null)}
          />

          {/* Detail Panel */}
          <FileDetailPanel
            file={detailFile}
            open={!!detailFile}
            onClose={() => setDetailFile(null)}
            onPreview={setPreviewFile}
            onRename={(f) => setRenameTarget({ id: f.id, name: f.name, type: 'file' })}
            onShare={setShareTarget}
            onDelete={setDeleteConfirmId}
            onToggleFavorite={(id) => {
              toggleFavorite(id)
              toast.success('Favoriten aktualisiert')
            }}
            onUpdateTags={(id, tags) => {
              updateFileTags(id, tags)
              toast.success('Tags aktualisiert')
            }}
            onMove={setMoveTarget}
          />

          {/* Folder Create Dialog */}
          <FolderCreateDialog
            open={folderCreateOpen}
            onOpenChange={setFolderCreateOpen}
            parentName={folderCreateParentId ? folders.find((f) => f.id === folderCreateParentId)?.name : undefined}
            onSubmit={(name) => {
              addFolder(name, folderCreateParentId)
              toast.success(`Ordner "${name}" erstellt`)
            }}
          />

          {/* Rename Dialog */}
          <RenameDialog
            open={!!renameTarget}
            onOpenChange={(open) => !open && setRenameTarget(null)}
            currentName={renameTarget?.name || ''}
            itemType={renameTarget?.type || 'file'}
            onSubmit={handleRenameSubmit}
          />

          {/* Share Dialog */}
          <ShareDialog
            open={!!shareTarget}
            onOpenChange={(open) => !open && setShareTarget(null)}
            fileName={shareTarget?.name || ''}
            currentShares={shareTarget?.sharedWith || []}
            onSave={(shares) => {
              if (shareTarget) {
                const wasShared = shareTarget.isShared
                const isNowShared = shares.length > 0
                if (wasShared !== isNowShared) {
                  toggleShare(shareTarget.id)
                }
                toast.success(isNowShared ? 'Freigabe gespeichert' : 'Freigabe entfernt')
              }
            }}
          />

          {/* Delete File Confirm */}
          <ConfirmDialog
            open={!!deleteConfirmId}
            onOpenChange={(open) => !open && setDeleteConfirmId(null)}
            title="Datei löschen?"
            description={`"${deleteTarget?.name}" wird unwiderruflich gelöscht.`}
            confirmLabel="Löschen"
            variant="destructive"
            onConfirm={handleDeleteFile}
          />

          {/* Delete Folder Confirm */}
          <ConfirmDialog
            open={!!deleteFolderConfirmId}
            onOpenChange={(open) => !open && setDeleteFolderConfirmId(null)}
            title="Ordner löschen?"
            description={`"${deleteFolderTarget?.name}" und alle enthaltenen Dateien werden verschoben nach "Alle Dateien".`}
            confirmLabel="Löschen"
            variant="destructive"
            onConfirm={handleDeleteFolder}
          />

          {/* Move File Dialog */}
          <MoveFileDialog
            open={!!moveTarget}
            onOpenChange={(open) => !open && setMoveTarget(null)}
            fileName={moveTarget?.name || ''}
            currentFolderId={moveTarget?.folderId || 'root'}
            folders={folders.filter((f) => !f.isSystem || f.id === 'root')}
            onMove={(folderId) => {
              if (moveTarget) {
                moveFile(moveTarget.id, folderId)
                const targetFolder = folders.find((f) => f.id === folderId)
                toast.success(`"${moveTarget.name}" nach "${targetFolder?.name || 'Alle Dateien'}" verschoben`)
                setMoveTarget(null)
              }
            }}
          />
        </div>
      )}

      {/* Wiki tab */}
      {activeTab === 'wiki' && <WikiTab />}
    </div>
  )
}

/* ===== Wiki Tab ===== */

function WikiTab() {
  const {
    wikiArticles, wikiCategories,
    addWikiArticle, updateWikiArticle, deleteWikiArticle,
  } = useDocumentsStore()

  const [wikiSearch, setWikiSearch] = useState('')
  const [activeCategory, setActiveCategory] = useState<string | null>(null)
  const [selectedArticle, setSelectedArticle] = useState<WikiArticle | null>(null)
  const [activeTag, setActiveTag] = useState<string | null>(null)

  // Dialogs
  const [showArticleForm, setShowArticleForm] = useState(false)
  const [editArticle, setEditArticle] = useState<WikiArticle | null>(null)
  const [deleteArticleConfirm, setDeleteArticleConfirm] = useState<WikiArticle | null>(null)

  // Form state
  const [formTitle, setFormTitle] = useState('')
  const [formCategory, setFormCategory] = useState('')
  const [formContent, setFormContent] = useState('')
  const [formTags, setFormTags] = useState('')

  // Derived
  const filteredArticles = useMemo(() => {
    let result = [...wikiArticles]
    if (activeCategory) {
      result = result.filter((a) => a.categoryId === activeCategory)
    }
    if (activeTag) {
      result = result.filter((a) => a.tags.includes(activeTag))
    }
    if (wikiSearch) {
      const q = wikiSearch.toLowerCase()
      result = result.filter(
        (a) =>
          a.title.toLowerCase().includes(q) ||
          a.content.toLowerCase().includes(q) ||
          a.tags.some((t) => t.toLowerCase().includes(q))
      )
    }
    return result.sort((a, b) => b.updatedAt.localeCompare(a.updatedAt))
  }, [wikiArticles, activeCategory, activeTag, wikiSearch])

  const allTags = useMemo(() => {
    const tagMap: Record<string, number> = {}
    wikiArticles.forEach((a) => a.tags.forEach((t) => { tagMap[t] = (tagMap[t] || 0) + 1 }))
    return Object.entries(tagMap).sort((a, b) => b[1] - a[1])
  }, [wikiArticles])

  const recentlyUpdated = useMemo(
    () => [...wikiArticles].sort((a, b) => b.updatedAt.localeCompare(a.updatedAt)).slice(0, 5),
    [wikiArticles]
  )

  const latestUpdate = recentlyUpdated[0]?.updatedAt
    ? new Date(recentlyUpdated[0].updatedAt).toLocaleDateString('de-CH')
    : '-'

  const getCategoryName = (id: string) => wikiCategories.find((c) => c.id === id)?.name || 'Unbekannt'

  const openNewArticle = () => {
    setEditArticle(null)
    setFormTitle('')
    setFormCategory(wikiCategories[0]?.id || '')
    setFormContent('')
    setFormTags('')
    setShowArticleForm(true)
  }

  const openEditArticle = (article: WikiArticle) => {
    setEditArticle(article)
    setFormTitle(article.title)
    setFormCategory(article.categoryId)
    setFormContent(article.content)
    setFormTags(article.tags.join(', '))
    setShowArticleForm(true)
  }

  const handleArticleSubmit = () => {
    if (!formTitle.trim() || !formCategory) return
    const tags = formTags.split(',').map((t) => t.trim()).filter(Boolean)
    const now = new Date().toISOString().split('T')[0]

    if (editArticle) {
      updateWikiArticle(editArticle.id, {
        title: formTitle.trim(),
        categoryId: formCategory,
        content: formContent.trim(),
        tags,
        updatedAt: now,
      })
      // Update the selected article view if it's the same
      if (selectedArticle?.id === editArticle.id) {
        setSelectedArticle({
          ...editArticle,
          title: formTitle.trim(),
          categoryId: formCategory,
          content: formContent.trim(),
          tags,
          updatedAt: now,
        })
      }
      toast.success('Artikel aktualisiert')
    } else {
      addWikiArticle({
        title: formTitle.trim(),
        categoryId: formCategory,
        content: formContent.trim(),
        tags,
        author: 'Du',
        createdAt: now,
        updatedAt: now,
        views: 0,
      })
      toast.success('Artikel erstellt')
    }
    setShowArticleForm(false)
  }

  const handleDeleteArticle = () => {
    if (!deleteArticleConfirm) return
    deleteWikiArticle(deleteArticleConfirm.id)
    if (selectedArticle?.id === deleteArticleConfirm.id) setSelectedArticle(null)
    setDeleteArticleConfirm(null)
    toast.success('Artikel gelöscht')
  }

  return (
    <div className="flex flex-1 overflow-hidden">
      {/* Left Sidebar — Categories */}
      <aside className="w-56 shrink-0 border-r border-border bg-card p-4 overflow-y-auto">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-sm font-medium text-foreground">Kategorien</h3>
        </div>
        <nav className="space-y-0.5">
          {/* All articles */}
          <button
            onClick={() => { setActiveCategory(null); setActiveTag(null); setSelectedArticle(null) }}
            className={`flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors ${
              !activeCategory && !activeTag
                ? 'bg-primary-light text-primary font-medium'
                : 'text-foreground hover:bg-secondary'
            }`}
          >
            <BookOpen className="h-4 w-4 shrink-0" />
            <span className="truncate">Alle Artikel</span>
            <span className="ml-auto text-xs text-muted-foreground">{wikiArticles.length}</span>
          </button>

          {/* Category list */}
          {wikiCategories
            .sort((a, b) => a.order - b.order)
            .map((cat) => {
              const CatIcon = wikiCategoryIcons[cat.icon] || BookOpen
              const count = wikiArticles.filter((a) => a.categoryId === cat.id).length
              return (
                <button
                  key={cat.id}
                  onClick={() => { setActiveCategory(cat.id); setActiveTag(null); setSelectedArticle(null) }}
                  className={`flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors ${
                    activeCategory === cat.id
                      ? 'bg-primary-light text-primary font-medium'
                      : 'text-foreground hover:bg-secondary'
                  }`}
                >
                  <CatIcon className="h-4 w-4 shrink-0" />
                  <span className="truncate">{cat.name}</span>
                  <span className="ml-auto text-xs text-muted-foreground">{count}</span>
                </button>
              )
            })}
        </nav>

        {/* Stats */}
        <div className="mt-6 rounded-lg border border-border p-3 space-y-2">
          <div className="flex items-center justify-between">
            <span className="text-[10px] text-muted-foreground">Artikel gesamt</span>
            <span className="text-xs font-medium text-foreground">{wikiArticles.length}</span>
          </div>
          <div className="flex items-center justify-between">
            <span className="text-[10px] text-muted-foreground">Kategorien</span>
            <span className="text-xs font-medium text-foreground">{wikiCategories.length}</span>
          </div>
          <div className="flex items-center justify-between">
            <span className="text-[10px] text-muted-foreground">Letzte Änderung</span>
            <span className="text-xs font-medium text-foreground">{latestUpdate}</span>
          </div>
        </div>
      </aside>

      {/* Main content */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Search bar + actions */}
        <div className="flex items-center gap-3 border-b border-border px-6 py-3">
          <div className="relative flex-1 max-w-sm">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <input
              type="text"
              placeholder="Wiki durchsuchen..."
              value={wikiSearch}
              onChange={(e) => setWikiSearch(e.target.value)}
              className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-1.5 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>
          {activeTag && (
            <button
              onClick={() => setActiveTag(null)}
              className="flex items-center gap-1 rounded-full bg-primary-light px-2.5 py-1 text-xs text-primary"
            >
              <Tag className="h-3 w-3" />
              {activeTag}
              <span className="ml-1 font-medium">&times;</span>
            </button>
          )}
          <span className="text-xs text-muted-foreground hidden sm:block">
            {filteredArticles.length} Artikel
          </span>
          <button
            onClick={openNewArticle}
            className="ml-auto flex items-center gap-2 rounded-lg bg-primary px-3 py-1.5 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            <Plus className="h-4 w-4" />
            Neuer Artikel
          </button>
        </div>

        {/* Content area */}
        <div className="flex-1 overflow-y-auto p-6">
          {selectedArticle ? (
            <WikiArticleDetail
              article={selectedArticle}
              categoryName={getCategoryName(selectedArticle.categoryId)}
              onBack={() => setSelectedArticle(null)}
              onEdit={() => openEditArticle(selectedArticle)}
              onDelete={() => setDeleteArticleConfirm(selectedArticle)}
            />
          ) : filteredArticles.length === 0 ? (
            <EmptyState
              icon={BookOpen}
              title="Keine Artikel gefunden"
              description={wikiSearch ? 'Versuche einen anderen Suchbegriff' : 'Erstelle deinen ersten Wiki-Artikel'}
              action={!wikiSearch ? { label: 'Neuer Artikel', onClick: openNewArticle } : undefined}
            />
          ) : (
            <div className="grid gap-3">
              {filteredArticles.map((article) => (
                <WikiArticleCard
                  key={article.id}
                  article={article}
                  categoryName={getCategoryName(article.categoryId)}
                  onClick={() => setSelectedArticle(article)}
                />
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Right Sidebar — Recent + Tags */}
      <aside className="w-48 shrink-0 border-l border-border bg-card p-4 overflow-y-auto hidden lg:block">
        {/* Recent changes */}
        <h4 className="text-xs font-medium text-foreground mb-3">Letzte Änderungen</h4>
        <div className="space-y-2 mb-6">
          {recentlyUpdated.map((a) => (
            <button
              key={a.id}
              onClick={() => setSelectedArticle(a)}
              className="block w-full text-left group"
            >
              <p className="text-xs text-foreground truncate group-hover:text-primary transition-colors">{a.title}</p>
              <p className="text-[10px] text-muted-foreground flex items-center gap-1">
                <Clock className="h-2.5 w-2.5" />
                {new Date(a.updatedAt).toLocaleDateString('de-CH')}
              </p>
            </button>
          ))}
        </div>

        {/* Tags cloud */}
        <h4 className="text-xs font-medium text-foreground mb-3">Tags</h4>
        <div className="flex flex-wrap gap-1.5">
          {allTags.map(([tag, count]) => (
            <button
              key={tag}
              onClick={() => {
                setActiveTag(activeTag === tag ? null : tag)
                setActiveCategory(null)
                setSelectedArticle(null)
              }}
              className={`rounded-full px-2 py-0.5 text-[10px] transition-colors ${
                activeTag === tag
                  ? 'bg-primary text-primary-foreground'
                  : 'bg-secondary text-muted-foreground hover:text-foreground'
              }`}
            >
              {tag} ({count})
            </button>
          ))}
        </div>
      </aside>

      {/* Article Form Dialog */}
      <Dialog open={showArticleForm} onOpenChange={setShowArticleForm}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{editArticle ? 'Artikel bearbeiten' : 'Neuer Artikel'}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-1.5">
              <Label>Titel</Label>
              <Input
                placeholder="Artikelname"
                value={formTitle}
                onChange={(e) => setFormTitle(e.target.value)}
                autoFocus
              />
            </div>
            <div className="space-y-1.5">
              <Label>Kategorie</Label>
              <Select value={formCategory} onValueChange={setFormCategory}>
                <SelectTrigger>
                  <SelectValue placeholder="Kategorie wählen" />
                </SelectTrigger>
                <SelectContent>
                  {wikiCategories.map((c) => (
                    <SelectItem key={c.id} value={c.id}>{c.name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>Inhalt</Label>
              <Textarea
                placeholder="Artikelinhalt..."
                value={formContent}
                onChange={(e) => setFormContent(e.target.value)}
                rows={6}
              />
            </div>
            <div className="space-y-1.5">
              <Label>Tags (kommagetrennt)</Label>
              <Input
                placeholder="z.B. Anleitung, IT, Wichtig"
                value={formTags}
                onChange={(e) => setFormTags(e.target.value)}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowArticleForm(false)}>Abbrechen</Button>
            <Button onClick={handleArticleSubmit} disabled={!formTitle.trim() || !formCategory}>
              {editArticle ? 'Speichern' : 'Erstellen'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete article confirm */}
      <ConfirmDialog
        open={!!deleteArticleConfirm}
        onOpenChange={(open) => !open && setDeleteArticleConfirm(null)}
        title="Artikel löschen?"
        description={`"${deleteArticleConfirm?.title}" wird unwiderruflich gelöscht.`}
        confirmLabel="Löschen"
        variant="destructive"
        onConfirm={handleDeleteArticle}
      />
    </div>
  )
}

/* ===== Wiki Article Card (list item) ===== */

function WikiArticleCard({
  article,
  categoryName,
  onClick,
}: {
  article: WikiArticle
  categoryName: string
  onClick: () => void
}) {
  const firstLine = article.content.split('\n')[0].slice(0, 120)

  return (
    <div
      onClick={onClick}
      className="group rounded-lg border border-border bg-card p-4 transition-shadow hover:shadow-[var(--shadow-card-hover)] cursor-pointer"
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2 mb-1">
            <h3 className="text-sm font-medium text-foreground truncate group-hover:text-primary transition-colors">
              {article.title}
            </h3>
            <span className="shrink-0 rounded-full bg-primary-light px-2 py-0.5 text-[10px] font-medium text-primary">
              {categoryName}
            </span>
          </div>
          <p className="text-xs text-muted-foreground line-clamp-1 mb-2">{firstLine}</p>
          <div className="flex items-center gap-3 text-[10px] text-muted-foreground">
            <span className="flex items-center gap-1">
              <Users className="h-3 w-3" />
              {article.author}
            </span>
            <span className="flex items-center gap-1">
              <Clock className="h-3 w-3" />
              {new Date(article.updatedAt).toLocaleDateString('de-CH')}
            </span>
            <span className="flex items-center gap-1">
              <Eye className="h-3 w-3" />
              {article.views}
            </span>
          </div>
        </div>
      </div>
      {article.tags.length > 0 && (
        <div className="flex flex-wrap gap-1 mt-2.5">
          {article.tags.slice(0, 4).map((tag) => (
            <span key={tag} className="rounded-full bg-secondary px-2 py-0.5 text-[10px] text-muted-foreground">
              {tag}
            </span>
          ))}
          {article.tags.length > 4 && (
            <span className="text-[10px] text-muted-foreground">+{article.tags.length - 4}</span>
          )}
        </div>
      )}
    </div>
  )
}

/* ===== Wiki Article Detail View ===== */

function WikiArticleDetail({
  article,
  categoryName,
  onBack,
  onEdit,
  onDelete,
}: {
  article: WikiArticle
  categoryName: string
  onBack: () => void
  onEdit: () => void
  onDelete: () => void
}) {
  return (
    <div className="max-w-3xl">
      {/* Back button */}
      <button
        onClick={onBack}
        className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors mb-4"
      >
        <ArrowLeft className="h-4 w-4" />
        Zurück zur Übersicht
      </button>

      {/* Title */}
      <h1 className="text-xl font-semibold text-foreground mb-3">{article.title}</h1>

      {/* Meta */}
      <div className="flex flex-wrap items-center gap-3 mb-4 text-xs text-muted-foreground">
        <span className="rounded-full bg-primary-light px-2.5 py-0.5 text-[11px] font-medium text-primary">
          {categoryName}
        </span>
        <span className="flex items-center gap-1">
          <Users className="h-3.5 w-3.5" />
          {article.author}
        </span>
        <span className="flex items-center gap-1">
          <Clock className="h-3.5 w-3.5" />
          Erstellt: {new Date(article.createdAt).toLocaleDateString('de-CH')}
        </span>
        <span className="flex items-center gap-1">
          <Clock className="h-3.5 w-3.5" />
          Aktualisiert: {new Date(article.updatedAt).toLocaleDateString('de-CH')}
        </span>
        <span className="flex items-center gap-1">
          <Eye className="h-3.5 w-3.5" />
          {article.views} Aufrufe
        </span>
      </div>

      {/* Tags */}
      {article.tags.length > 0 && (
        <div className="flex flex-wrap gap-1.5 mb-6">
          {article.tags.map((tag) => (
            <span key={tag} className="rounded-full bg-secondary px-2.5 py-0.5 text-[11px] text-muted-foreground flex items-center gap-1">
              <Hash className="h-3 w-3" />
              {tag}
            </span>
          ))}
        </div>
      )}

      {/* Content */}
      <div className="rounded-lg border border-border bg-card p-5 mb-6">
        {article.content.split('\n').map((paragraph, i) => (
          <p
            key={i}
            className={`text-sm text-foreground leading-relaxed ${paragraph.trim() === '' ? 'h-3' : i > 0 ? 'mt-2' : ''}`}
          >
            {paragraph}
          </p>
        ))}
      </div>

      {/* Actions */}
      <div className="flex items-center gap-2">
        <button
          onClick={onEdit}
          className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
        >
          <Pencil className="h-4 w-4" />
          Bearbeiten
        </button>
        <button
          onClick={onDelete}
          className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm text-error hover:bg-error-light transition-colors"
        >
          <Trash2 className="h-4 w-4" />
          Löschen
        </button>
      </div>
    </div>
  )
}

/* ===== File Manager Sub-Components (unchanged) ===== */

function FolderTreeItem({
  folder,
  allFolders,
  activeFolder,
  onSelect,
  actions,
  depth,
}: {
  folder: DocFolder
  allFolders: DocFolder[]
  activeFolder: string
  onSelect: (id: string) => void
  actions: ActionItem[]
  depth: number
}) {
  const [expanded, setExpanded] = useState(folder.id === 'root' || folder.id === 'vault')
  const isActive = activeFolder === folder.id
  const children = allFolders.filter((f) => f.parentId === folder.id)
  const hasChildren = children.length > 0

  const iconMap: Record<string, typeof Folder> = {
    folder: Folder,
    share: Share2,
    star: Star,
    lock: Lock,
  }
  const Icon = iconMap[folder.icon] || Folder

  return (
    <div>
      <div className="group flex items-center">
        <button
          onClick={() => {
            onSelect(folder.id)
            if (hasChildren) setExpanded(!expanded)
          }}
          className={`flex flex-1 items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors ${
            isActive
              ? 'bg-primary-light text-primary font-medium'
              : 'text-foreground hover:bg-secondary'
          }`}
          style={{ paddingLeft: `${8 + depth * 16}px` }}
        >
          {hasChildren ? (
            expanded ? (
              <ChevronDown className="h-3 w-3 shrink-0 text-muted-foreground" />
            ) : (
              <ChevronRight className="h-3 w-3 shrink-0 text-muted-foreground" />
            )
          ) : (
            <span className="w-3" />
          )}
          <Icon className={`h-4 w-4 shrink-0 ${folder.icon === 'lock' ? 'text-success' : ''}`} />
          <span className="truncate">{folder.name}</span>
        </button>
        {actions.length > 0 && (
          <div className="opacity-0 group-hover:opacity-100 transition-opacity pr-1">
            <ItemActions items={actions} />
          </div>
        )}
      </div>
      {expanded &&
        hasChildren &&
        children.map((child) => (
          <FolderTreeItem
            key={child.id}
            folder={child}
            allFolders={allFolders}
            activeFolder={activeFolder}
            onSelect={onSelect}
            actions={child.isSystem ? [] : [
              {
                label: 'Umbenennen',
                icon: Pencil,
                onClick: () => {},
              },
              {
                label: 'Löschen',
                icon: Trash2,
                variant: 'destructive',
                onClick: () => {},
              },
            ]}
            depth={depth + 1}
          />
        ))}
    </div>
  )
}

function FileGridCard({
  file,
  actions,
  onDoubleClick,
  onClick,
}: {
  file: DocFile
  actions: ActionItem[]
  onDoubleClick: () => void
  onClick: () => void
}) {
  const Icon = fileTypeIcons[file.type] || File
  const colorClass = fileTypeColors[file.type] || fileTypeColors.other

  return (
    <div
      className="group rounded-lg border border-border bg-card p-3 transition-shadow hover:shadow-[var(--shadow-card-hover)] cursor-pointer"
      onDoubleClick={onDoubleClick}
      onClick={onClick}
    >
      <div className={`flex h-20 items-center justify-center rounded-md ${colorClass} mb-3 relative`}>
        <Icon className="h-8 w-8" />
        <div className="absolute top-1.5 right-1.5 opacity-0 group-hover:opacity-100 transition-opacity" onClick={(e) => e.stopPropagation()}>
          <ItemActions items={actions} />
        </div>
      </div>
      <div className="flex items-start justify-between gap-1">
        <div className="min-w-0">
          <p className="text-sm font-medium text-foreground truncate" title={file.name}>
            {file.name}
          </p>
          <p className="text-xs text-muted-foreground mt-0.5">
            {file.size} &middot; {new Date(file.date).toLocaleDateString('de-CH')}
          </p>
        </div>
        <div className="flex items-center gap-0.5 shrink-0">
          {file.isFavorite && <Star className="h-3 w-3 fill-yellow-400 text-yellow-400" />}
          {file.isVault && <Lock className="h-3 w-3 text-success" />}
          {file.isShared && <Share2 className="h-3 w-3 text-info" />}
        </div>
      </div>
      {file.tags.length > 0 && (
        <div className="flex flex-wrap gap-1 mt-2">
          {file.tags.slice(0, 2).map((tag) => (
            <span key={tag} className="rounded-full bg-secondary px-2 py-0.5 text-[10px] text-muted-foreground">
              {tag}
            </span>
          ))}
          {file.tags.length > 2 && (
            <span className="text-[10px] text-muted-foreground">+{file.tags.length - 2}</span>
          )}
        </div>
      )}
    </div>
  )
}

function FileListRow({
  file,
  actions,
  onDoubleClick,
  onClick,
}: {
  file: DocFile
  actions: ActionItem[]
  onDoubleClick: () => void
  onClick: () => void
}) {
  const Icon = fileTypeIcons[file.type] || File
  const colorClass = fileTypeColors[file.type] || fileTypeColors.other

  return (
    <div
      className="group grid grid-cols-[1fr_100px_100px_120px_40px] items-center gap-3 rounded-md px-3 py-2 hover:bg-secondary/50 transition-colors cursor-pointer"
      onDoubleClick={onDoubleClick}
      onClick={onClick}
    >
      <div className="flex items-center gap-3 min-w-0">
        <div className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-md ${colorClass}`}>
          <Icon className="h-4 w-4" />
        </div>
        <div className="min-w-0">
          <p className="text-sm text-foreground truncate">{file.name}</p>
          <p className="text-[10px] text-muted-foreground">{file.createdBy}</p>
        </div>
        {file.isFavorite && <Star className="h-3 w-3 shrink-0 fill-yellow-400 text-yellow-400" />}
        {file.isVault && <Lock className="h-3 w-3 shrink-0 text-success" />}
        {file.isShared && <Share2 className="h-3 w-3 shrink-0 text-info" />}
      </div>
      <span className="text-xs text-muted-foreground">{file.size}</span>
      <span className="text-xs text-muted-foreground">{typeLabels[file.type]}</span>
      <span className="text-xs text-muted-foreground">{new Date(file.date).toLocaleDateString('de-CH')}</span>
      <div className="opacity-0 group-hover:opacity-100 transition-opacity" onClick={(e) => e.stopPropagation()}>
        <ItemActions items={actions} />
      </div>
    </div>
  )
}

/* ===== Move File Dialog ===== */

function MoveFileDialog({
  open,
  onOpenChange,
  fileName,
  currentFolderId,
  folders,
  onMove,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  fileName: string
  currentFolderId: string
  folders: DocFolder[]
  onMove: (folderId: string) => void
}) {
  const [selected, setSelected] = useState(currentFolderId)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Folder className="h-5 w-5" />
            Datei verschieben
          </DialogTitle>
        </DialogHeader>

        <p className="text-sm text-muted-foreground truncate">{fileName}</p>

        <div className="max-h-56 overflow-y-auto rounded-md border border-border p-1 space-y-0.5">
          {folders.map((folder) => {
            const isCurrent = folder.id === currentFolderId
            const isSelected = folder.id === selected
            const indent = folder.parentId && folder.parentId !== 'root'
              ? folders.some((f) => f.id === folder.parentId && f.parentId !== null) ? 32 : 16
              : 0

            return (
              <button
                key={folder.id}
                onClick={() => setSelected(folder.id)}
                disabled={isCurrent}
                className={`flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm transition-colors ${
                  isSelected && !isCurrent
                    ? 'bg-primary-light text-primary font-medium'
                    : isCurrent
                      ? 'text-muted-foreground cursor-not-allowed'
                      : 'text-foreground hover:bg-secondary'
                }`}
                style={{ paddingLeft: `${8 + indent}px` }}
              >
                <Folder className="h-4 w-4 shrink-0" />
                <span className="truncate">{folder.name}</span>
                {isCurrent && (
                  <span className="ml-auto text-[10px] text-muted-foreground">(aktuell)</span>
                )}
              </button>
            )
          })}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Abbrechen</Button>
          <Button
            disabled={selected === currentFolderId}
            onClick={() => onMove(selected)}
          >
            Verschieben
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

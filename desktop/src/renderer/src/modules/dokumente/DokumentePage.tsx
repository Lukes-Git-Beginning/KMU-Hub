import { useState, useRef } from 'react'
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
} from 'lucide-react'
import { toast } from 'sonner'
import { useDocumentsStore, type DocFile, type DocFolder } from '@/stores/documents'
import { ItemActions, ConfirmDialog, EmptyState, type ActionItem } from '@/components/shared'
import { FilePreviewModal } from './FilePreviewModal'
import { FileDetailPanel } from './FileDetailPanel'
import { FolderCreateDialog } from './FolderCreateDialog'
import { RenameDialog } from './RenameDialog'
import { ShareDialog } from './ShareDialog'

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

export default function DokumentePage() {
  const {
    files, folders,
    addFile, removeFile, renameFile, moveFile, toggleFavorite, toggleShare, updateFileTags,
    addFolder, renameFolder, deleteFolder, totalStorageUsed,
  } = useDocumentsStore()

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
    <div className="flex h-full overflow-hidden">
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
    </div>
  )
}

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

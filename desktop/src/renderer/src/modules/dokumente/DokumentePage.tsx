import { useState, useRef, useMemo, useCallback } from 'react'
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
  Share2,
  Trash2,
  Plus,
  Pencil,
  BookOpen,
  Users,
  Monitor,
  GitBranch,
  HardDrive,
  Paperclip,
  MessageSquare,
  Mail,
  CheckSquare,
  Heart,
  ArrowLeft,
  Clock,
  Eye,
  Hash,
  Tag,
} from 'lucide-react'
import { toast } from 'sonner'
import { ItemActions, ConfirmDialog, EmptyState, type ActionItem } from '@/components/shared'
import { FilePreviewModal } from './FilePreviewModal'
import { FileDetailPanel } from './FileDetailPanel'
import { FolderCreateDialog } from './FolderCreateDialog'
import { RenameDialog } from './RenameDialog'
import { ShareDialog } from './ShareDialog'
import { FileContextMenu, FolderContextMenu } from './FileContextMenu'
import { VersionHistoryPanel } from './VersionHistoryPanel'
import { OnlyOfficeEditor, isOnlyOfficeEditable } from './OnlyOfficeEditor'
import {
  useDocumentFolders,
  useDocumentFiles,
  useFolderPath,
  useUpdateFile,
  useDeleteFile,
  useDeleteFolder,
  useMoveFile,
  useVirtualFiles,
  useSharedWithMe,
  useWOPIToken,
} from '@/api/hooks/useDocuments'
import { useDocumentUpload, type UploadFileStatus } from '@/api/hooks/useDocumentUpload'
import type {
  DocumentFile,
  DocumentFolder,
  FolderSpaceType,
  FileSortField,
  SortDirection,
} from '@/api/types/document-types'
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

// Wiki imports -- keep Zustand for wiki as per plan
import { useDocumentsStore, type WikiArticle } from '@/stores/documents'

type TabKey = 'dateien' | 'wiki'

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

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

function formatBytes(bytes: number): string {
  if (bytes >= 1073741824) return `${(bytes / 1073741824).toFixed(1)} GB`
  if (bytes >= 1048576) return `${(bytes / 1048576).toFixed(1)} MB`
  if (bytes >= 1024) return `${(bytes / 1024).toFixed(0)} KB`
  return `${bytes} B`
}

function getMimeCategory(mimeType: string): string {
  if (mimeType.startsWith('image/')) return 'image'
  if (mimeType === 'application/pdf') return 'pdf'
  if (mimeType.startsWith('video/')) return 'video'
  if (mimeType.includes('spreadsheet') || mimeType.includes('excel')) return 'excel'
  if (mimeType.includes('word') || mimeType.includes('document')) return 'word'
  if (mimeType.includes('zip') || mimeType.includes('archive')) return 'archive'
  return 'other'
}

function getViewPref(folderId: string): 'grid' | 'list' {
  try {
    const stored = localStorage.getItem(`view-pref-${folderId}`)
    return stored === 'list' ? 'list' : 'grid'
  } catch {
    return 'grid'
  }
}

function setViewPref(folderId: string, view: 'grid' | 'list') {
  try {
    localStorage.setItem(`view-pref-${folderId}`, view)
  } catch {
    // silently fail
  }
}

// Virtual sidebar item keys
const SIDEBAR_FAVORITES = '__favorites__'
const SIDEBAR_SHARED = '__shared__'
const SIDEBAR_VIRTUAL_CHAT = '__virtual_chat__'
const SIDEBAR_VIRTUAL_EMAIL = '__virtual_email__'
const SIDEBAR_VIRTUAL_TASK = '__virtual_task__'

// Wiki constants (kept from original)
const wikiCategoryIcons: Record<string, typeof BookOpen> = {
  BookOpen: BookOpen,
  Monitor: Monitor,
  Users: Users,
  GitBranch: GitBranch,
  FileText: FileText,
}

// ---------------------------------------------------------------------------
// Main Component
// ---------------------------------------------------------------------------

export default function DokumentePage() {
  const [activeTab, setActiveTab] = useState<TabKey>('dateien')
  const [activeFolderId, setActiveFolderId] = useState<string | null>(null)
  const [activeSpecialView, setActiveSpecialView] = useState<string | null>(null)
  const [view, setView] = useState<'grid' | 'list'>(() => getViewPref('default'))
  const [search, setSearch] = useState('')
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const [isDragOver, setIsDragOver] = useState(false)
  const [sortField] = useState<FileSortField>('date')
  const [sortDir] = useState<SortDirection>('desc')

  // Multi-select
  const [selectedFiles, setSelectedFiles] = useState<Set<string>>(new Set())

  // Dialog/Panel state
  const [previewFile, setPreviewFile] = useState<DocumentFile | null>(null)
  const [detailFile, setDetailFile] = useState<DocumentFile | null>(null)
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null)
  const [deleteFolderConfirmId, setDeleteFolderConfirmId] = useState<string | null>(null)
  const [folderCreateOpen, setFolderCreateOpen] = useState(false)
  const [folderCreateParentId, setFolderCreateParentId] = useState<string | null>(null)
  const [renameTarget, setRenameTarget] = useState<{
    id: string
    name: string
    type: 'file' | 'folder'
  } | null>(null)
  const [shareTarget, setShareTarget] = useState<{
    type: 'file' | 'folder'
    id: string
    name: string
  } | null>(null)
  const [versionHistoryTarget, setVersionHistoryTarget] = useState<{
    id: string
    name: string
  } | null>(null)

  // OnlyOffice editor state
  const [onlyOfficeEditor, setOnlyOfficeEditor] = useState<{
    fileId: string
    fileName: string
    token: string
    ttl: number
  } | null>(null)

  // Upload tracking
  const [uploadFiles, setUploadFiles] = useState<UploadFileStatus[]>([])
  const fileInputRef = useRef<HTMLInputElement>(null)

  // Current space context for folder creation
  const [currentSpaceType] = useState<FolderSpaceType>('personal')
  const [currentSpaceId] = useState<string>('me')

  // API hooks
  const { data: personalFolders } = useDocumentFolders({
    space_type: 'personal',
  })
  const { data: teamFolders } = useDocumentFolders({
    space_type: 'team',
  })
  const { data: projectFolders } = useDocumentFolders({
    space_type: 'project',
  })
  const { data: subFolders } = useDocumentFolders(
    activeFolderId ? { parent_id: activeFolderId } : undefined,
  )
  const { data: filesData, isLoading: filesLoading } = useDocumentFiles(
    activeSpecialView === SIDEBAR_FAVORITES
      ? { is_favorite: true, sort_field: sortField, sort_dir: sortDir }
      : activeFolderId
        ? { folder_id: activeFolderId, sort_field: sortField, sort_dir: sortDir }
        : { sort_field: sortField, sort_dir: sortDir },
  )
  const { data: pathData } = useFolderPath(activeFolderId ?? '')
  const { data: _sharedData } = useSharedWithMe(
    activeSpecialView === SIDEBAR_SHARED ? 'file' : undefined,
  )
  const { data: _virtualChatFiles } = useVirtualFiles(
    activeSpecialView === SIDEBAR_VIRTUAL_CHAT ? 'chat' : undefined,
  )
  const { data: _virtualEmailFiles } = useVirtualFiles(
    activeSpecialView === SIDEBAR_VIRTUAL_EMAIL ? 'email' : undefined,
  )
  const { data: _virtualTaskFiles } = useVirtualFiles(
    activeSpecialView === SIDEBAR_VIRTUAL_TASK ? 'task' : undefined,
  )

  const updateFile = useUpdateFile()
  const deleteFile = useDeleteFile()
  const deleteFolder = useDeleteFolder()
  const moveFile = useMoveFile()
  const upload = useDocumentUpload()
  const wopiToken = useWOPIToken()

  // Derived data
  const files = filesData?.files ?? []
  const breadcrumbs = pathData?.segments ?? []

  const filtered = useMemo(() => {
    if (!search) return files
    const q = search.toLowerCase()
    return files.filter((f) => f.filename.toLowerCase().includes(q))
  }, [files, search])

  const contentSubfolders = subFolders?.folders ?? []
  const deleteTarget = files.find((f) => f.id === deleteConfirmId)
  const folderList = [
    ...(personalFolders?.folders ?? []),
    ...(teamFolders?.folders ?? []),
    ...(projectFolders?.folders ?? []),
  ]
  const deleteFolderTarget = folderList.find(
    (f) => f.id === deleteFolderConfirmId,
  )

  // View preference per folder
  const handleViewChange = (newView: 'grid' | 'list') => {
    setView(newView)
    setViewPref(activeFolderId ?? 'default', newView)
  }

  // Navigate to a folder
  const navigateToFolder = (folderId: string | null) => {
    setActiveFolderId(folderId)
    setActiveSpecialView(null)
    setSelectedFiles(new Set())
    if (folderId) {
      setView(getViewPref(folderId))
    }
  }

  const navigateToSpecial = (key: string) => {
    setActiveSpecialView(key)
    setActiveFolderId(null)
    setSelectedFiles(new Set())
  }

  // Upload handlers
  const handleUpload = () => {
    fileInputRef.current?.click()
  }

  const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const fileList = e.target.files
    if (!fileList || fileList.length === 0) return

    const targetFolder = activeFolderId ?? ''
    const newUploads: UploadFileStatus[] = Array.from(fileList).map((f) => ({
      file: f,
      progress: 0,
      status: 'pending' as const,
    }))

    setUploadFiles((prev) => [...prev, ...newUploads])

    for (let i = 0; i < newUploads.length; i++) {
      const uploadItem = newUploads[i]
      setUploadFiles((prev) =>
        prev.map((u) =>
          u.file === uploadItem.file ? { ...u, status: 'uploading' } : u,
        ),
      )

      try {
        await upload.mutateAsync({
          folderId: targetFolder,
          file: uploadItem.file,
          onProgress: (pct) => {
            setUploadFiles((prev) =>
              prev.map((u) =>
                u.file === uploadItem.file ? { ...u, progress: pct } : u,
              ),
            )
          },
        })
        setUploadFiles((prev) =>
          prev.map((u) =>
            u.file === uploadItem.file
              ? { ...u, status: 'complete', progress: 100 }
              : u,
          ),
        )
      } catch (err) {
        setUploadFiles((prev) =>
          prev.map((u) =>
            u.file === uploadItem.file
              ? {
                  ...u,
                  status: 'error',
                  error: err instanceof Error ? err.message : 'Upload fehlgeschlagen',
                }
              : u,
          ),
        )
      }
    }

    // Clear completed uploads after 3s
    setTimeout(() => {
      setUploadFiles((prev) => prev.filter((u) => u.status !== 'complete'))
    }, 3000)

    // Reset input
    e.target.value = ''
  }

  // Desktop drop handler
  const handleDrop = async (e: React.DragEvent) => {
    e.preventDefault()
    setIsDragOver(false)

    // Check if it's an internal file drag (file reorganization)
    const draggedFileId = e.dataTransfer.getData('application/x-document-file')
    if (draggedFileId && activeFolderId) {
      moveFile.mutate(
        { id: draggedFileId, targetFolderId: activeFolderId },
        {
          onSuccess: () => toast.success('Datei verschoben'),
          onError: (err) => toast.error(`Fehler: ${err.message}`),
        },
      )
      return
    }

    // Desktop file upload
    const droppedFiles = e.dataTransfer.files
    if (!droppedFiles || droppedFiles.length === 0) return

    const targetFolder = activeFolderId ?? ''
    const newUploads: UploadFileStatus[] = Array.from(droppedFiles).map(
      (f) => ({
        file: f,
        progress: 0,
        status: 'pending' as const,
      }),
    )

    setUploadFiles((prev) => [...prev, ...newUploads])

    for (const uploadItem of newUploads) {
      setUploadFiles((prev) =>
        prev.map((u) =>
          u.file === uploadItem.file ? { ...u, status: 'uploading' } : u,
        ),
      )

      try {
        await upload.mutateAsync({
          folderId: targetFolder,
          file: uploadItem.file,
          onProgress: (pct) => {
            setUploadFiles((prev) =>
              prev.map((u) =>
                u.file === uploadItem.file ? { ...u, progress: pct } : u,
              ),
            )
          },
        })
        setUploadFiles((prev) =>
          prev.map((u) =>
            u.file === uploadItem.file
              ? { ...u, status: 'complete', progress: 100 }
              : u,
          ),
        )
        toast.success(`"${uploadItem.file.name}" hochgeladen`)
      } catch (err) {
        setUploadFiles((prev) =>
          prev.map((u) =>
            u.file === uploadItem.file
              ? {
                  ...u,
                  status: 'error',
                  error:
                    err instanceof Error
                      ? err.message
                      : 'Upload fehlgeschlagen',
                }
              : u,
          ),
        )
      }
    }

    setTimeout(() => {
      setUploadFiles((prev) => prev.filter((u) => u.status !== 'complete'))
    }, 3000)
  }

  // Folder drop handler for sidebar items
  const handleFolderDrop = useCallback(
    (folderId: string, e: React.DragEvent) => {
      e.preventDefault()
      e.stopPropagation()
      const draggedFileId = e.dataTransfer.getData(
        'application/x-document-file',
      )
      if (draggedFileId) {
        moveFile.mutate(
          { id: draggedFileId, targetFolderId: folderId },
          {
            onSuccess: () => toast.success('Datei verschoben'),
            onError: (err) => toast.error(`Fehler: ${err.message}`),
          },
        )
      }
    },
    [moveFile],
  )

  // Delete handlers
  const handleDeleteFile = () => {
    if (deleteConfirmId) {
      deleteFile.mutate(deleteConfirmId, {
        onSuccess: () => {
          toast.success('Datei geloescht')
          if (detailFile?.id === deleteConfirmId) setDetailFile(null)
          setDeleteConfirmId(null)
        },
        onError: (err) => {
          toast.error(`Fehler: ${err.message}`)
          setDeleteConfirmId(null)
        },
      })
    }
  }

  const handleDeleteFolder = () => {
    if (deleteFolderConfirmId) {
      deleteFolder.mutate(deleteFolderConfirmId, {
        onSuccess: () => {
          toast.success('Ordner geloescht')
          if (activeFolderId === deleteFolderConfirmId) {
            navigateToFolder(null)
          }
          setDeleteFolderConfirmId(null)
        },
        onError: (err) => {
          toast.error(`Fehler: ${err.message}`)
          setDeleteFolderConfirmId(null)
        },
      })
    }
  }

  const handleToggleFavorite = (fileId: string) => {
    const file = files.find((f) => f.id === fileId)
    if (!file) return
    updateFile.mutate(
      { id: fileId, is_favorite: !file.is_favorite },
      {
        onSuccess: () =>
          toast.success(
            file.is_favorite ? 'Aus Favoriten entfernt' : 'Zu Favoriten hinzugefuegt',
          ),
      },
    )
  }

  // Multi-select handler
  const handleFileClick = (
    file: DocumentFile,
    e: React.MouseEvent,
  ) => {
    if (e.ctrlKey || e.metaKey) {
      setSelectedFiles((prev) => {
        const next = new Set(prev)
        if (next.has(file.id)) next.delete(file.id)
        else next.add(file.id)
        return next
      })
    } else if (e.shiftKey && selectedFiles.size > 0) {
      // Range select
      const lastSelected = Array.from(selectedFiles).pop()
      const lastIdx = filtered.findIndex((f) => f.id === lastSelected)
      const currentIdx = filtered.findIndex((f) => f.id === file.id)
      if (lastIdx >= 0 && currentIdx >= 0) {
        const start = Math.min(lastIdx, currentIdx)
        const end = Math.max(lastIdx, currentIdx)
        const range = filtered.slice(start, end + 1).map((f) => f.id)
        setSelectedFiles(new Set([...selectedFiles, ...range]))
      }
    } else {
      setSelectedFiles(new Set())
      setDetailFile(file)
    }
  }

  // Internal drag start for file reorganization
  const handleFileDragStart = (
    e: React.DragEvent,
    fileId: string,
  ) => {
    e.dataTransfer.setData('application/x-document-file', fileId)
    e.dataTransfer.effectAllowed = 'move'
  }

  // Get active section label
  const getActiveSectionName = (): string => {
    if (activeSpecialView === SIDEBAR_FAVORITES) return 'Favoriten'
    if (activeSpecialView === SIDEBAR_SHARED) return 'Geteilt mit mir'
    if (activeSpecialView === SIDEBAR_VIRTUAL_CHAT) return 'Chat-Anhaenge'
    if (activeSpecialView === SIDEBAR_VIRTUAL_EMAIL) return 'E-Mail-Anhaenge'
    if (activeSpecialView === SIDEBAR_VIRTUAL_TASK) return 'Aufgaben-Anhaenge'
    if (breadcrumbs.length > 0) return breadcrumbs[breadcrumbs.length - 1].name
    return 'Alle Dateien'
  }

  // OnlyOffice editor handler
  const handleEditInOnlyOffice = async (file: DocumentFile) => {
    try {
      const result = await wopiToken.mutateAsync(file.id)
      setOnlyOfficeEditor({
        fileId: file.id,
        fileName: file.filename,
        token: result.access_token,
        ttl: result.access_token_ttl,
      })
    } catch (err) {
      toast.error(
        `Editor konnte nicht geoeffnet werden: ${err instanceof Error ? err.message : 'Unbekannter Fehler'}`,
      )
    }
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
                activeTab === t.key
                  ? 'border-primary text-primary font-medium'
                  : 'border-transparent text-muted-foreground hover:text-foreground'
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
              {/* Personal Space */}
              <div className="mb-4">
                <div className="flex items-center justify-between mb-2">
                  <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                    Meine Dateien
                  </h3>
                  <button
                    onClick={() => {
                      setFolderCreateParentId(null)
                      setFolderCreateOpen(true)
                    }}
                    className="rounded-md p-1 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
                    title="Neuer Ordner"
                  >
                    <Plus className="h-3.5 w-3.5" />
                  </button>
                </div>
                <nav className="space-y-0.5">
                  {(personalFolders?.folders ?? [])
                    .filter((f) => !f.parent_id)
                    .map((folder) => (
                      <SidebarFolderItem
                        key={folder.id}
                        folder={folder}
                        allFolders={personalFolders?.folders ?? []}
                        activeFolderId={activeFolderId}
                        onSelect={navigateToFolder}
                        onFolderDrop={handleFolderDrop}
                        onNewSubfolder={(id) => {
                          setFolderCreateParentId(id)
                          setFolderCreateOpen(true)
                        }}
                        onRename={(f) =>
                          setRenameTarget({
                            id: f.id,
                            name: f.name,
                            type: 'folder',
                          })
                        }
                        onDelete={(id) => setDeleteFolderConfirmId(id)}
                        depth={0}
                      />
                    ))}
                </nav>
              </div>

              {/* Team Spaces */}
              {(teamFolders?.folders ?? []).length > 0 && (
                <div className="mb-4">
                  <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground mb-2">
                    Team
                  </h3>
                  <nav className="space-y-0.5">
                    {(teamFolders?.folders ?? [])
                      .filter((f) => !f.parent_id)
                      .map((folder) => (
                        <SidebarFolderItem
                          key={folder.id}
                          folder={folder}
                          allFolders={teamFolders?.folders ?? []}
                          activeFolderId={activeFolderId}
                          onSelect={navigateToFolder}
                          onFolderDrop={handleFolderDrop}
                          onNewSubfolder={(id) => {
                            setFolderCreateParentId(id)
                            setFolderCreateOpen(true)
                          }}
                          onRename={(f) =>
                            setRenameTarget({
                              id: f.id,
                              name: f.name,
                              type: 'folder',
                            })
                          }
                          onDelete={(id) => setDeleteFolderConfirmId(id)}
                          depth={0}
                        />
                      ))}
                  </nav>
                </div>
              )}

              {/* Project Spaces */}
              {(projectFolders?.folders ?? []).length > 0 && (
                <div className="mb-4">
                  <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground mb-2">
                    Projekte
                  </h3>
                  <nav className="space-y-0.5">
                    {(projectFolders?.folders ?? [])
                      .filter((f) => !f.parent_id)
                      .map((folder) => (
                        <SidebarFolderItem
                          key={folder.id}
                          folder={folder}
                          allFolders={projectFolders?.folders ?? []}
                          activeFolderId={activeFolderId}
                          onSelect={navigateToFolder}
                          onFolderDrop={handleFolderDrop}
                          onNewSubfolder={(id) => {
                            setFolderCreateParentId(id)
                            setFolderCreateOpen(true)
                          }}
                          onRename={(f) =>
                            setRenameTarget({
                              id: f.id,
                              name: f.name,
                              type: 'folder',
                            })
                          }
                          onDelete={(id) => setDeleteFolderConfirmId(id)}
                          depth={0}
                        />
                      ))}
                  </nav>
                </div>
              )}

              {/* Quick access */}
              <div className="mb-4 border-t border-border pt-4">
                <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground mb-2">
                  Schnellzugriff
                </h3>
                <nav className="space-y-0.5">
                  <button
                    onClick={() => navigateToSpecial(SIDEBAR_FAVORITES)}
                    className={`flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors ${
                      activeSpecialView === SIDEBAR_FAVORITES
                        ? 'bg-primary-light text-primary font-medium'
                        : 'text-foreground hover:bg-secondary'
                    }`}
                  >
                    <Star className="h-4 w-4 shrink-0" />
                    <span className="truncate">Favoriten</span>
                  </button>
                  <button
                    onClick={() => navigateToSpecial(SIDEBAR_SHARED)}
                    className={`flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors ${
                      activeSpecialView === SIDEBAR_SHARED
                        ? 'bg-primary-light text-primary font-medium'
                        : 'text-foreground hover:bg-secondary'
                    }`}
                  >
                    <Share2 className="h-4 w-4 shrink-0" />
                    <span className="truncate">Geteilt mit mir</span>
                  </button>
                </nav>
              </div>

              {/* Virtual folders */}
              <div className="border-t border-border pt-4">
                <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground mb-2">
                  Anhaenge
                </h3>
                <nav className="space-y-0.5">
                  <button
                    onClick={() => navigateToSpecial(SIDEBAR_VIRTUAL_CHAT)}
                    className={`flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors ${
                      activeSpecialView === SIDEBAR_VIRTUAL_CHAT
                        ? 'bg-primary-light text-primary font-medium'
                        : 'text-foreground hover:bg-secondary'
                    }`}
                  >
                    <MessageSquare className="h-4 w-4 shrink-0" />
                    <span className="truncate">Chat-Anhaenge</span>
                  </button>
                  <button
                    onClick={() => navigateToSpecial(SIDEBAR_VIRTUAL_EMAIL)}
                    className={`flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors ${
                      activeSpecialView === SIDEBAR_VIRTUAL_EMAIL
                        ? 'bg-primary-light text-primary font-medium'
                        : 'text-foreground hover:bg-secondary'
                    }`}
                  >
                    <Mail className="h-4 w-4 shrink-0" />
                    <span className="truncate">E-Mail-Anhaenge</span>
                  </button>
                  <button
                    onClick={() => navigateToSpecial(SIDEBAR_VIRTUAL_TASK)}
                    className={`flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors ${
                      activeSpecialView === SIDEBAR_VIRTUAL_TASK
                        ? 'bg-primary-light text-primary font-medium'
                        : 'text-foreground hover:bg-secondary'
                    }`}
                  >
                    <CheckSquare className="h-4 w-4 shrink-0" />
                    <span className="truncate">Aufgaben-Anhaenge</span>
                  </button>
                </nav>
              </div>
            </aside>
          )}

          {/* Main content area */}
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
                  onClick={() => handleViewChange('grid')}
                  className={`rounded-md p-1.5 transition-colors ${
                    view === 'grid'
                      ? 'bg-secondary text-foreground'
                      : 'text-muted-foreground hover:bg-secondary'
                  }`}
                >
                  <Grid3X3 className="h-4 w-4" />
                </button>
                <button
                  onClick={() => handleViewChange('list')}
                  className={`rounded-md p-1.5 transition-colors ${
                    view === 'list'
                      ? 'bg-secondary text-foreground'
                      : 'text-muted-foreground hover:bg-secondary'
                  }`}
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

            {/* Breadcrumbs */}
            {breadcrumbs.length > 0 && !activeSpecialView && (
              <div className="flex items-center gap-1 border-b border-border px-6 py-2 text-sm">
                <button
                  onClick={() => navigateToFolder(null)}
                  className="text-muted-foreground hover:text-foreground transition-colors"
                >
                  Alle Dateien
                </button>
                {breadcrumbs.map((seg) => (
                  <div key={seg.id} className="flex items-center gap-1">
                    <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
                    <button
                      onClick={() => navigateToFolder(seg.id)}
                      className={`transition-colors ${
                        seg.id === activeFolderId
                          ? 'text-foreground font-medium'
                          : 'text-muted-foreground hover:text-foreground'
                      }`}
                    >
                      {seg.name}
                    </button>
                  </div>
                ))}
              </div>
            )}

            {/* Content with drag overlay */}
            <div
              className="flex-1 overflow-y-auto p-6 relative"
              onDragEnter={(e) => {
                e.preventDefault()
                setIsDragOver(true)
              }}
              onDragOver={(e) => {
                e.preventDefault()
                setIsDragOver(true)
              }}
              onDragLeave={(e) => {
                e.preventDefault()
                setIsDragOver(false)
              }}
              onDrop={handleDrop}
            >
              {/* Drag overlay */}
              {isDragOver && (
                <div className="absolute inset-0 z-10 flex items-center justify-center rounded-lg border-2 border-dashed border-primary bg-primary/5 backdrop-blur-sm">
                  <div className="text-center">
                    <Upload className="h-12 w-12 mx-auto mb-3 text-primary opacity-60" />
                    <p className="text-sm font-medium text-primary">
                      Dateien hierhin ziehen
                    </p>
                    <p className="text-xs text-muted-foreground mt-1">
                      in &quot;{getActiveSectionName()}&quot; hochladen
                    </p>
                  </div>
                </div>
              )}

              {/* Subfolders (shown as cards above files) */}
              {contentSubfolders.length > 0 && !activeSpecialView && (
                <div className="mb-4">
                  <div
                    className={
                      view === 'grid'
                        ? 'grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-3'
                        : 'space-y-1'
                    }
                  >
                    {contentSubfolders.map((folder) => (
                      <FolderContextMenu
                        key={folder.id}
                        folder={folder}
                        onOpen={() => navigateToFolder(folder.id)}
                        onNewSubfolder={() => {
                          setFolderCreateParentId(folder.id)
                          setFolderCreateOpen(true)
                        }}
                        onUploadHere={() => {
                          navigateToFolder(folder.id)
                          setTimeout(handleUpload, 100)
                        }}
                        onRename={() =>
                          setRenameTarget({
                            id: folder.id,
                            name: folder.name,
                            type: 'folder',
                          })
                        }
                        onDelete={() =>
                          setDeleteFolderConfirmId(folder.id)
                        }
                      >
                        <div
                          className="group rounded-lg border border-border bg-card p-3 cursor-pointer hover:shadow-[var(--shadow-card-hover)] transition-shadow"
                          onDoubleClick={() =>
                            navigateToFolder(folder.id)
                          }
                          onDragOver={(e) => {
                            e.preventDefault()
                            e.currentTarget.classList.add(
                              'ring-2',
                              'ring-primary',
                            )
                          }}
                          onDragLeave={(e) => {
                            e.currentTarget.classList.remove(
                              'ring-2',
                              'ring-primary',
                            )
                          }}
                          onDrop={(e) => {
                            e.currentTarget.classList.remove(
                              'ring-2',
                              'ring-primary',
                            )
                            handleFolderDrop(folder.id, e)
                          }}
                        >
                          <div className="flex items-center gap-2">
                            <Folder className="h-5 w-5 text-primary shrink-0" />
                            <span className="text-sm font-medium text-foreground truncate">
                              {folder.name}
                            </span>
                            <span className="text-xs text-muted-foreground ml-auto">
                              {folder.file_count} Dateien
                            </span>
                          </div>
                        </div>
                      </FolderContextMenu>
                    ))}
                  </div>
                </div>
              )}

              {/* Files */}
              {view === 'grid' ? (
                <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4">
                  {filtered.map((file) => (
                    <FileContextMenu
                      key={file.id}
                      file={file}
                      onOpen={() => setPreviewFile(file)}
                      onDownload={() =>
                        toast.success(
                          `"${file.filename}" wird heruntergeladen`,
                        )
                      }
                      onRename={() =>
                        setRenameTarget({
                          id: file.id,
                          name: file.filename,
                          type: 'file',
                        })
                      }
                      onMove={() =>
                        toast.info('Verschieben-Dialog kommt in Kuerze')
                      }
                      onCopy={() =>
                        toast.info('Kopieren-Dialog kommt in Kuerze')
                      }
                      onShare={() =>
                        setShareTarget({
                          type: 'file',
                          id: file.id,
                          name: file.filename,
                        })
                      }
                      onVersionHistory={() =>
                        setVersionHistoryTarget({
                          id: file.id,
                          name: file.filename,
                        })
                      }
                      onDelete={() => setDeleteConfirmId(file.id)}
                      onProperties={() => setDetailFile(file)}
                      onEditInOnlyOffice={
                        isOnlyOfficeEditable(file.mime_type)
                          ? () => handleEditInOnlyOffice(file)
                          : undefined
                      }
                    >
                      <FileGridCard
                        file={file}
                        isSelected={selectedFiles.has(file.id)}
                        onDoubleClick={() => setPreviewFile(file)}
                        onClick={(e) => handleFileClick(file, e)}
                        onDragStart={(e) =>
                          handleFileDragStart(e, file.id)
                        }
                      />
                    </FileContextMenu>
                  ))}
                </div>
              ) : (
                <div className="space-y-1">
                  <div className="grid grid-cols-[1fr_100px_100px_120px] gap-3 px-3 py-2 text-xs font-medium text-muted-foreground border-b border-border">
                    <span>Name</span>
                    <span>Groesse</span>
                    <span>Typ</span>
                    <span>Datum</span>
                  </div>
                  {filtered.map((file) => (
                    <FileContextMenu
                      key={file.id}
                      file={file}
                      onOpen={() => setPreviewFile(file)}
                      onDownload={() =>
                        toast.success(
                          `"${file.filename}" wird heruntergeladen`,
                        )
                      }
                      onRename={() =>
                        setRenameTarget({
                          id: file.id,
                          name: file.filename,
                          type: 'file',
                        })
                      }
                      onMove={() =>
                        toast.info('Verschieben-Dialog kommt in Kuerze')
                      }
                      onCopy={() =>
                        toast.info('Kopieren-Dialog kommt in Kuerze')
                      }
                      onShare={() =>
                        setShareTarget({
                          type: 'file',
                          id: file.id,
                          name: file.filename,
                        })
                      }
                      onVersionHistory={() =>
                        setVersionHistoryTarget({
                          id: file.id,
                          name: file.filename,
                        })
                      }
                      onDelete={() => setDeleteConfirmId(file.id)}
                      onProperties={() => setDetailFile(file)}
                      onEditInOnlyOffice={
                        isOnlyOfficeEditable(file.mime_type)
                          ? () => handleEditInOnlyOffice(file)
                          : undefined
                      }
                    >
                      <FileListRow
                        file={file}
                        isSelected={selectedFiles.has(file.id)}
                        onDoubleClick={() => setPreviewFile(file)}
                        onClick={(e) => handleFileClick(file, e)}
                        onDragStart={(e) =>
                          handleFileDragStart(e, file.id)
                        }
                      />
                    </FileContextMenu>
                  ))}
                </div>
              )}

              {filtered.length === 0 && !filesLoading && (
                <EmptyState
                  icon={FolderOpen}
                  title="Keine Dateien gefunden"
                  description={
                    search
                      ? 'Versuche einen anderen Suchbegriff'
                      : 'Lade deine erste Datei hoch'
                  }
                  action={
                    !search
                      ? { label: 'Datei hochladen', onClick: handleUpload }
                      : undefined
                  }
                />
              )}
            </div>

            {/* Upload progress overlay */}
            {uploadFiles.length > 0 && (
              <div className="fixed bottom-4 right-4 z-50 w-80 space-y-2">
                {uploadFiles.map((u, i) => (
                  <div
                    key={`${u.file.name}-${i}`}
                    className="rounded-lg border border-border bg-card p-3 shadow-lg"
                  >
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-xs text-foreground truncate max-w-[200px]">
                        {u.file.name}
                      </span>
                      <span className="text-[10px] text-muted-foreground">
                        {u.status === 'complete'
                          ? 'Fertig'
                          : u.status === 'error'
                            ? 'Fehler'
                            : `${u.progress}%`}
                      </span>
                    </div>
                    <div className="h-1.5 rounded-full bg-secondary">
                      <div
                        className={`h-full rounded-full transition-all ${
                          u.status === 'error'
                            ? 'bg-destructive'
                            : u.status === 'complete'
                              ? 'bg-green-500'
                              : 'bg-primary'
                        }`}
                        style={{ width: `${u.progress}%` }}
                      />
                    </div>
                    {u.error && (
                      <p className="text-[10px] text-destructive mt-1">
                        {u.error}
                      </p>
                    )}
                  </div>
                ))}
              </div>
            )}
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
            onRename={(f) =>
              setRenameTarget({
                id: f.id,
                name: f.filename,
                type: 'file',
              })
            }
            onShare={(f) =>
              setShareTarget({
                type: 'file',
                id: f.id,
                name: f.filename,
              })
            }
            onDelete={setDeleteConfirmId}
            onToggleFavorite={handleToggleFavorite}
            onVersionHistory={(id) => {
              const f = files.find((file) => file.id === id)
              if (f) setVersionHistoryTarget({ id: f.id, name: f.filename })
            }}
          />

          {/* Version History Panel */}
          <VersionHistoryPanel
            fileId={versionHistoryTarget?.id ?? ''}
            fileName={versionHistoryTarget?.name ?? ''}
            open={!!versionHistoryTarget}
            onClose={() => setVersionHistoryTarget(null)}
          />

          {/* Folder Create Dialog */}
          <FolderCreateDialog
            open={folderCreateOpen}
            onOpenChange={setFolderCreateOpen}
            parentId={folderCreateParentId}
            parentName={
              folderCreateParentId
                ? folderList.find((f) => f.id === folderCreateParentId)?.name
                : undefined
            }
            spaceType={currentSpaceType}
            spaceId={currentSpaceId}
          />

          {/* Rename Dialog */}
          <RenameDialog
            open={!!renameTarget}
            onOpenChange={(open) => !open && setRenameTarget(null)}
            targetId={renameTarget?.id ?? ''}
            currentName={renameTarget?.name ?? ''}
            itemType={renameTarget?.type ?? 'file'}
          />

          {/* Share Dialog */}
          <ShareDialog
            open={!!shareTarget}
            onOpenChange={(open) => !open && setShareTarget(null)}
            entityType={shareTarget?.type ?? 'file'}
            entityId={shareTarget?.id ?? ''}
            entityName={shareTarget?.name ?? ''}
          />

          {/* Delete File Confirm */}
          <ConfirmDialog
            open={!!deleteConfirmId}
            onOpenChange={(open) => !open && setDeleteConfirmId(null)}
            title="Datei loeschen?"
            description={`"${deleteTarget?.filename}" wird unwiderruflich geloescht.`}
            confirmLabel="Loeschen"
            variant="destructive"
            onConfirm={handleDeleteFile}
          />

          {/* Delete Folder Confirm */}
          <ConfirmDialog
            open={!!deleteFolderConfirmId}
            onOpenChange={(open) =>
              !open && setDeleteFolderConfirmId(null)
            }
            title="Ordner loeschen?"
            description={`"${deleteFolderTarget?.name}" und alle enthaltenen Dateien werden geloescht.`}
            confirmLabel="Loeschen"
            variant="destructive"
            onConfirm={handleDeleteFolder}
          />

          {/* OnlyOffice Editor Overlay */}
          {onlyOfficeEditor && (
            <OnlyOfficeEditor
              fileId={onlyOfficeEditor.fileId}
              fileName={onlyOfficeEditor.fileName}
              wopiToken={onlyOfficeEditor.token}
              wopiTokenTTL={onlyOfficeEditor.ttl}
              onClose={() => setOnlyOfficeEditor(null)}
              onVersionCreated={() => {
                // Invalidate file queries since version may have changed
                setOnlyOfficeEditor(null)
              }}
            />
          )}
        </div>
      )}

      {/* Wiki tab */}
      {activeTab === 'wiki' && <WikiTab />}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Sidebar Folder Item (recursive tree)
// ---------------------------------------------------------------------------

function SidebarFolderItem({
  folder,
  allFolders,
  activeFolderId,
  onSelect,
  onFolderDrop,
  onNewSubfolder,
  onRename,
  onDelete,
  depth,
}: {
  folder: DocumentFolder
  allFolders: DocumentFolder[]
  activeFolderId: string | null
  onSelect: (id: string) => void
  onFolderDrop: (folderId: string, e: React.DragEvent) => void
  onNewSubfolder: (parentId: string) => void
  onRename: (folder: DocumentFolder) => void
  onDelete: (id: string) => void
  depth: number
}) {
  const [expanded, setExpanded] = useState(depth === 0)
  const [isDragTarget, setIsDragTarget] = useState(false)
  const isActive = activeFolderId === folder.id
  const children = allFolders.filter((f) => f.parent_id === folder.id)
  const hasChildren = children.length > 0

  return (
    <div>
      <FolderContextMenu
        folder={folder}
        onOpen={() => onSelect(folder.id)}
        onNewSubfolder={() => onNewSubfolder(folder.id)}
        onUploadHere={() => onSelect(folder.id)}
        onRename={() => onRename(folder)}
        onDelete={() => onDelete(folder.id)}
      >
        <div
          className={`group flex items-center ${
            isDragTarget ? 'ring-2 ring-primary rounded-md' : ''
          }`}
          onDragOver={(e) => {
            e.preventDefault()
            setIsDragTarget(true)
          }}
          onDragLeave={() => setIsDragTarget(false)}
          onDrop={(e) => {
            setIsDragTarget(false)
            onFolderDrop(folder.id, e)
          }}
        >
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
            <Folder
              className={`h-4 w-4 shrink-0 ${
                folder.icon === 'lock' ? 'text-success' : ''
              }`}
            />
            <span className="truncate">{folder.name}</span>
          </button>
        </div>
      </FolderContextMenu>
      {expanded &&
        hasChildren &&
        children.map((child) => (
          <SidebarFolderItem
            key={child.id}
            folder={child}
            allFolders={allFolders}
            activeFolderId={activeFolderId}
            onSelect={onSelect}
            onFolderDrop={onFolderDrop}
            onNewSubfolder={onNewSubfolder}
            onRename={onRename}
            onDelete={onDelete}
            depth={depth + 1}
          />
        ))}
    </div>
  )
}

// ---------------------------------------------------------------------------
// File Grid Card
// ---------------------------------------------------------------------------

function FileGridCard({
  file,
  isSelected,
  onDoubleClick,
  onClick,
  onDragStart,
}: {
  file: DocumentFile
  isSelected: boolean
  onDoubleClick: () => void
  onClick: (e: React.MouseEvent) => void
  onDragStart: (e: React.DragEvent) => void
}) {
  const cat = getMimeCategory(file.mime_type)
  const Icon = fileTypeIcons[cat] || File
  const colorClass = fileTypeColors[cat] || fileTypeColors.other

  return (
    <div
      className={`group rounded-lg border bg-card p-3 transition-shadow hover:shadow-[var(--shadow-card-hover)] cursor-pointer ${
        isSelected
          ? 'border-primary ring-2 ring-primary/20'
          : 'border-border'
      }`}
      onDoubleClick={onDoubleClick}
      onClick={onClick}
      draggable
      onDragStart={onDragStart}
    >
      <div
        className={`flex h-20 items-center justify-center rounded-md ${colorClass} mb-3 relative`}
      >
        <Icon className="h-8 w-8" />
      </div>
      <div className="flex items-start justify-between gap-1">
        <div className="min-w-0">
          <p
            className="text-sm font-medium text-foreground truncate"
            title={file.filename}
          >
            {file.filename}
          </p>
          <p className="text-xs text-muted-foreground mt-0.5">
            {formatBytes(file.file_size)} &middot;{' '}
            {new Date(file.updated_at).toLocaleDateString('de-CH')}
          </p>
        </div>
        <div className="flex items-center gap-0.5 shrink-0">
          {file.is_favorite && (
            <Star className="h-3 w-3 fill-yellow-400 text-yellow-400" />
          )}
        </div>
      </div>
      {file.tags && file.tags.length > 0 && (
        <div className="flex flex-wrap gap-1 mt-2">
          {file.tags.slice(0, 2).map((tag) => (
            <span
              key={tag.id}
              className="rounded-full bg-secondary px-2 py-0.5 text-[10px] text-muted-foreground"
            >
              {tag.name}
            </span>
          ))}
          {file.tags.length > 2 && (
            <span className="text-[10px] text-muted-foreground">
              +{file.tags.length - 2}
            </span>
          )}
        </div>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// File List Row
// ---------------------------------------------------------------------------

function FileListRow({
  file,
  isSelected,
  onDoubleClick,
  onClick,
  onDragStart,
}: {
  file: DocumentFile
  isSelected: boolean
  onDoubleClick: () => void
  onClick: (e: React.MouseEvent) => void
  onDragStart: (e: React.DragEvent) => void
}) {
  const cat = getMimeCategory(file.mime_type)
  const Icon = fileTypeIcons[cat] || File
  const colorClass = fileTypeColors[cat] || fileTypeColors.other

  const typeLabels: Record<string, string> = {
    pdf: 'PDF',
    word: 'Word',
    excel: 'Excel',
    image: 'Bild',
    video: 'Video',
    archive: 'Archiv',
    other: 'Datei',
  }

  return (
    <div
      className={`group grid grid-cols-[1fr_100px_100px_120px] items-center gap-3 rounded-md px-3 py-2 transition-colors cursor-pointer ${
        isSelected
          ? 'bg-primary/5 ring-1 ring-primary/20'
          : 'hover:bg-secondary/50'
      }`}
      onDoubleClick={onDoubleClick}
      onClick={onClick}
      draggable
      onDragStart={onDragStart}
    >
      <div className="flex items-center gap-3 min-w-0">
        <div
          className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-md ${colorClass}`}
        >
          <Icon className="h-4 w-4" />
        </div>
        <div className="min-w-0">
          <p className="text-sm text-foreground truncate">{file.filename}</p>
        </div>
        {file.is_favorite && (
          <Star className="h-3 w-3 shrink-0 fill-yellow-400 text-yellow-400" />
        )}
      </div>
      <span className="text-xs text-muted-foreground">
        {formatBytes(file.file_size)}
      </span>
      <span className="text-xs text-muted-foreground">
        {typeLabels[cat] ?? cat}
      </span>
      <span className="text-xs text-muted-foreground">
        {new Date(file.updated_at).toLocaleDateString('de-CH')}
      </span>
    </div>
  )
}

// =====================================================================
// Wiki Tab (kept from original -- uses Zustand store)
// =====================================================================

function WikiTab() {
  const {
    wikiArticles,
    wikiCategories,
    addWikiArticle,
    updateWikiArticle,
    deleteWikiArticle,
  } = useDocumentsStore()

  const [wikiSearch, setWikiSearch] = useState('')
  const [activeCategory, setActiveCategory] = useState<string | null>(null)
  const [selectedArticle, setSelectedArticle] = useState<WikiArticle | null>(
    null,
  )
  const [activeTag, setActiveTag] = useState<string | null>(null)

  // Dialogs
  const [showArticleForm, setShowArticleForm] = useState(false)
  const [editArticle, setEditArticle] = useState<WikiArticle | null>(null)
  const [deleteArticleConfirm, setDeleteArticleConfirm] =
    useState<WikiArticle | null>(null)

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
          a.tags.some((t) => t.toLowerCase().includes(q)),
      )
    }
    return result.sort((a, b) => b.updatedAt.localeCompare(a.updatedAt))
  }, [wikiArticles, activeCategory, activeTag, wikiSearch])

  const allTags = useMemo(() => {
    const tagMap: Record<string, number> = {}
    wikiArticles.forEach((a) =>
      a.tags.forEach((t) => {
        tagMap[t] = (tagMap[t] || 0) + 1
      }),
    )
    return Object.entries(tagMap).sort((a, b) => b[1] - a[1])
  }, [wikiArticles])

  const recentlyUpdated = useMemo(
    () =>
      [...wikiArticles]
        .sort((a, b) => b.updatedAt.localeCompare(a.updatedAt))
        .slice(0, 5),
    [wikiArticles],
  )

  const latestUpdate = recentlyUpdated[0]?.updatedAt
    ? new Date(recentlyUpdated[0].updatedAt).toLocaleDateString('de-CH')
    : '-'

  const getCategoryName = (id: string) =>
    wikiCategories.find((c) => c.id === id)?.name || 'Unbekannt'

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
    const tags = formTags
      .split(',')
      .map((t) => t.trim())
      .filter(Boolean)
    const now = new Date().toISOString().split('T')[0]

    if (editArticle) {
      updateWikiArticle(editArticle.id, {
        title: formTitle.trim(),
        categoryId: formCategory,
        content: formContent.trim(),
        tags,
        updatedAt: now,
      })
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
    if (selectedArticle?.id === deleteArticleConfirm.id)
      setSelectedArticle(null)
    setDeleteArticleConfirm(null)
    toast.success('Artikel geloescht')
  }

  return (
    <div className="flex flex-1 overflow-hidden">
      {/* Left Sidebar -- Categories */}
      <aside className="w-56 shrink-0 border-r border-border bg-card p-4 overflow-y-auto">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-sm font-medium text-foreground">Kategorien</h3>
        </div>
        <nav className="space-y-0.5">
          <button
            onClick={() => {
              setActiveCategory(null)
              setActiveTag(null)
              setSelectedArticle(null)
            }}
            className={`flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors ${
              !activeCategory && !activeTag
                ? 'bg-primary-light text-primary font-medium'
                : 'text-foreground hover:bg-secondary'
            }`}
          >
            <BookOpen className="h-4 w-4 shrink-0" />
            <span className="truncate">Alle Artikel</span>
            <span className="ml-auto text-xs text-muted-foreground">
              {wikiArticles.length}
            </span>
          </button>

          {wikiCategories
            .sort((a, b) => a.order - b.order)
            .map((cat) => {
              const CatIcon = wikiCategoryIcons[cat.icon] || BookOpen
              const count = wikiArticles.filter(
                (a) => a.categoryId === cat.id,
              ).length
              return (
                <button
                  key={cat.id}
                  onClick={() => {
                    setActiveCategory(cat.id)
                    setActiveTag(null)
                    setSelectedArticle(null)
                  }}
                  className={`flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors ${
                    activeCategory === cat.id
                      ? 'bg-primary-light text-primary font-medium'
                      : 'text-foreground hover:bg-secondary'
                  }`}
                >
                  <CatIcon className="h-4 w-4 shrink-0" />
                  <span className="truncate">{cat.name}</span>
                  <span className="ml-auto text-xs text-muted-foreground">
                    {count}
                  </span>
                </button>
              )
            })}
        </nav>

        <div className="mt-6 rounded-lg border border-border p-3 space-y-2">
          <div className="flex items-center justify-between">
            <span className="text-[10px] text-muted-foreground">
              Artikel gesamt
            </span>
            <span className="text-xs font-medium text-foreground">
              {wikiArticles.length}
            </span>
          </div>
          <div className="flex items-center justify-between">
            <span className="text-[10px] text-muted-foreground">
              Kategorien
            </span>
            <span className="text-xs font-medium text-foreground">
              {wikiCategories.length}
            </span>
          </div>
          <div className="flex items-center justify-between">
            <span className="text-[10px] text-muted-foreground">
              Letzte Aenderung
            </span>
            <span className="text-xs font-medium text-foreground">
              {latestUpdate}
            </span>
          </div>
        </div>
      </aside>

      {/* Main content */}
      <div className="flex-1 flex flex-col overflow-hidden">
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
              description={
                wikiSearch
                  ? 'Versuche einen anderen Suchbegriff'
                  : 'Erstelle deinen ersten Wiki-Artikel'
              }
              action={
                !wikiSearch
                  ? { label: 'Neuer Artikel', onClick: openNewArticle }
                  : undefined
              }
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

      {/* Right Sidebar -- Recent + Tags */}
      <aside className="w-48 shrink-0 border-l border-border bg-card p-4 overflow-y-auto hidden lg:block">
        <h4 className="text-xs font-medium text-foreground mb-3">
          Letzte Aenderungen
        </h4>
        <div className="space-y-2 mb-6">
          {recentlyUpdated.map((a) => (
            <button
              key={a.id}
              onClick={() => setSelectedArticle(a)}
              className="block w-full text-left group"
            >
              <p className="text-xs text-foreground truncate group-hover:text-primary transition-colors">
                {a.title}
              </p>
              <p className="text-[10px] text-muted-foreground flex items-center gap-1">
                <Clock className="h-2.5 w-2.5" />
                {new Date(a.updatedAt).toLocaleDateString('de-CH')}
              </p>
            </button>
          ))}
        </div>

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
            <DialogTitle>
              {editArticle ? 'Artikel bearbeiten' : 'Neuer Artikel'}
            </DialogTitle>
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
              <Select
                value={formCategory}
                onValueChange={setFormCategory}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Kategorie waehlen" />
                </SelectTrigger>
                <SelectContent>
                  {wikiCategories.map((c) => (
                    <SelectItem key={c.id} value={c.id}>
                      {c.name}
                    </SelectItem>
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
            <Button
              variant="outline"
              onClick={() => setShowArticleForm(false)}
            >
              Abbrechen
            </Button>
            <Button
              onClick={handleArticleSubmit}
              disabled={!formTitle.trim() || !formCategory}
            >
              {editArticle ? 'Speichern' : 'Erstellen'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={!!deleteArticleConfirm}
        onOpenChange={(open) => !open && setDeleteArticleConfirm(null)}
        title="Artikel loeschen?"
        description={`"${deleteArticleConfirm?.title}" wird unwiderruflich geloescht.`}
        confirmLabel="Loeschen"
        variant="destructive"
        onConfirm={handleDeleteArticle}
      />
    </div>
  )
}

// =====================================================================
// Wiki sub-components (unchanged from original)
// =====================================================================

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
          <p className="text-xs text-muted-foreground line-clamp-1 mb-2">
            {firstLine}
          </p>
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
            <span
              key={tag}
              className="rounded-full bg-secondary px-2 py-0.5 text-[10px] text-muted-foreground"
            >
              {tag}
            </span>
          ))}
          {article.tags.length > 4 && (
            <span className="text-[10px] text-muted-foreground">
              +{article.tags.length - 4}
            </span>
          )}
        </div>
      )}
    </div>
  )
}

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
      <button
        onClick={onBack}
        className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors mb-4"
      >
        <ArrowLeft className="h-4 w-4" />
        Zurueck zur Uebersicht
      </button>

      <h1 className="text-xl font-semibold text-foreground mb-3">
        {article.title}
      </h1>

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
          Erstellt:{' '}
          {new Date(article.createdAt).toLocaleDateString('de-CH')}
        </span>
        <span className="flex items-center gap-1">
          <Clock className="h-3.5 w-3.5" />
          Aktualisiert:{' '}
          {new Date(article.updatedAt).toLocaleDateString('de-CH')}
        </span>
        <span className="flex items-center gap-1">
          <Eye className="h-3.5 w-3.5" />
          {article.views} Aufrufe
        </span>
      </div>

      {article.tags.length > 0 && (
        <div className="flex flex-wrap gap-1.5 mb-6">
          {article.tags.map((tag) => (
            <span
              key={tag}
              className="rounded-full bg-secondary px-2.5 py-0.5 text-[11px] text-muted-foreground flex items-center gap-1"
            >
              <Hash className="h-3 w-3" />
              {tag}
            </span>
          ))}
        </div>
      )}

      <div className="rounded-lg border border-border bg-card p-5 mb-6">
        {article.content.split('\n').map((paragraph, i) => (
          <p
            key={i}
            className={`text-sm text-foreground leading-relaxed ${
              paragraph.trim() === '' ? 'h-3' : i > 0 ? 'mt-2' : ''
            }`}
          >
            {paragraph}
          </p>
        ))}
      </div>

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
          Loeschen
        </button>
      </div>
    </div>
  )
}

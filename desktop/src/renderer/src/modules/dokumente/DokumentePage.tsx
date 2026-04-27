import { useState, useRef, useMemo, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import {
  FileText,
  FolderOpen,
  Upload,
  Search,
  Grid3X3,
  List,
  Star,
  ChevronRight,
  ChevronDown,
  Folder,
  Image,
  FileSpreadsheet,
  File,
  Film,
  Archive,
  Share2,
  Plus,
  MessageSquare,
  Mail,
  CheckSquare,
} from 'lucide-react'
import { toast } from 'sonner'
import { ConfirmDialog, EmptyState } from '@/components/shared'
import { FilePreviewModal } from './FilePreviewModal'
import { FileDetailPanel } from './FileDetailPanel'
import { FolderCreateDialog } from './FolderCreateDialog'
import { RenameDialog } from './RenameDialog'
import { ShareDialog } from './ShareDialog'
import { FileContextMenu, FolderContextMenu } from './FileContextMenu'
import { VersionHistoryPanel } from './VersionHistoryPanel'
import { OnlyOfficeEditor, isOnlyOfficeEditable } from './OnlyOfficeEditor'
import { TemplateGalleryDialog } from './TemplateGalleryDialog'
import { ShareLinkDialog } from './ShareLinkDialog'
import { useAIStore, type ClassificationLevel } from '@/stores/ai'
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
import { formatDate } from '@/lib/format'






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
    const stored = localStorage.getItem(`cosmi-view-pref-${folderId}`)
    return stored === 'list' ? 'list' : 'grid'
  } catch {
    return 'grid'
  }
}

function setViewPref(folderId: string, view: 'grid' | 'list') {
  try {
    localStorage.setItem(`cosmi-view-pref-${folderId}`, view)
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

// ---------------------------------------------------------------------------
// Main Component
// ---------------------------------------------------------------------------

export default function DokumentePage() {
  const { t } = useTranslation()
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

  // Template gallery + Share link
  const [showTemplateGallery, setShowTemplateGallery] = useState(false)
  const [shareLinkTarget, setShareLinkTarget] = useState<{ id: string; name: string } | null>(null)

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
  const files = useMemo(() => filesData?.files ?? [], [filesData?.files])
  const breadcrumbs = pathData?.segments ?? []

  const filtered = useMemo(() => {
    if (!search) return files
    const q = search.toLowerCase()
    return files.filter((f) => f.filename?.toLowerCase().includes(q))
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
                  error: err instanceof Error ? err.message : t('dokumente.upload.failed'),
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
          onSuccess: () => toast.success(t('dokumente.fileMoved')),
          onError: (err) => toast.error(`${t('common.error')}: ${err.message}`),
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
        toast.success(t('dokumente.upload.fileUploaded', { name: uploadItem.file.name }))
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
                      : t('dokumente.upload.failed'),
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
            onSuccess: () => toast.success(t('dokumente.fileMoved')),
            onError: (err) => toast.error(`${t('common.error')}: ${err.message}`),
          },
        )
      }
    },
    [moveFile, t],
  )

  // Delete handlers
  const handleDeleteFile = () => {
    if (deleteConfirmId) {
      deleteFile.mutate(deleteConfirmId, {
        onSuccess: () => {
          toast.success(t('dokumente.fileDeleted'))
          if (detailFile?.id === deleteConfirmId) setDetailFile(null)
          setDeleteConfirmId(null)
        },
        onError: (err) => {
          toast.error(`${t('common.error')}: ${err.message}`)
          setDeleteConfirmId(null)
        },
      })
    }
  }

  const handleDeleteFolder = () => {
    if (deleteFolderConfirmId) {
      deleteFolder.mutate(deleteFolderConfirmId, {
        onSuccess: () => {
          toast.success(t('dokumente.folderDeleted'))
          if (activeFolderId === deleteFolderConfirmId) {
            navigateToFolder(null)
          }
          setDeleteFolderConfirmId(null)
        },
        onError: (err) => {
          toast.error(`${t('common.error')}: ${err.message}`)
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
            file.is_favorite ? t('dokumente.favorites.removed') : t('dokumente.favorites.added'),
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
    if (activeSpecialView === SIDEBAR_FAVORITES) return t('dokumente.sidebar.favorites')
    if (activeSpecialView === SIDEBAR_SHARED) return t('dokumente.sidebar.sharedWithMe')
    if (activeSpecialView === SIDEBAR_VIRTUAL_CHAT) return t('dokumente.sidebar.chatAttachments')
    if (activeSpecialView === SIDEBAR_VIRTUAL_EMAIL) return t('dokumente.sidebar.emailAttachments')
    if (activeSpecialView === SIDEBAR_VIRTUAL_TASK) return t('dokumente.sidebar.taskAttachments')
    if (breadcrumbs.length > 0) return breadcrumbs[breadcrumbs.length - 1].name
    return t('dokumente.allFiles')
  }

  // "In Office öffnen" handler (Electron only)
  const isElectron = !!window.electronAPI
  const handleOpenInOffice = (file: DocumentFile) => {
    if (isElectron) {
      toast.success(t('dokumente.openInOffice.opening', { name: file.filename }))
    } else {
      toast.info(t('dokumente.openInOffice.desktopOnly'))
    }
  }

  const isOfficeFile = (mimeType: string): boolean => {
    return (
      mimeType.includes('word') ||
      mimeType.includes('document') ||
      mimeType.includes('spreadsheet') ||
      mimeType.includes('excel') ||
      mimeType.includes('presentation') ||
      mimeType.includes('powerpoint')
    )
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
        `${t('dokumente.editor.openError')}: ${err instanceof Error ? err.message : t('dokumente.unknownError')}`,
      )
    }
  }

  return (
    <div className="flex h-full flex-col overflow-hidden animate-fade-up">

        <div className="flex flex-1 min-h-0 overflow-hidden">
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
            <aside className="h-full w-56 shrink-0 border-r border-border bg-card p-4 overflow-y-auto">
              {/* Personal Space */}
              <div className="mb-4">
                <div className="flex items-center justify-between mb-2">
                  <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                    {t('dokumente.sidebar.myFiles')}
                  </h3>
                  <button
                    onClick={() => {
                      setFolderCreateParentId(null)
                      setFolderCreateOpen(true)
                    }}
                    className="rounded-md p-1 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
                    title={t('dokumente.folder.new')}
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
                    {t('dokumente.sidebar.team')}
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
                    {t('dokumente.sidebar.projects')}
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
                  {t('dokumente.sidebar.quickAccess')}
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
                    <span className="truncate">{t('dokumente.sidebar.favorites')}</span>
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
                    <span className="truncate">{t('dokumente.sidebar.sharedWithMe')}</span>
                  </button>
                </nav>
              </div>

              {/* Virtual folders */}
              <div className="border-t border-border pt-4">
                <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground mb-2">
                  {t('dokumente.sidebar.attachments')}
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
                    <span className="truncate">{t('dokumente.sidebar.chatAttachments')}</span>
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
                    <span className="truncate">{t('dokumente.sidebar.emailAttachments')}</span>
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
                    <span className="truncate">{t('dokumente.sidebar.taskAttachments')}</span>
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
                  placeholder={t('dokumente.searchPlaceholder')}
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-1.5 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                />
              </div>
              <span className="text-xs text-muted-foreground hidden sm:block">
                {t('dokumente.fileCount', { count: filtered.length })}
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
                onClick={() => setShowTemplateGallery(true)}
                className="flex items-center gap-2 rounded-lg border border-border px-3 py-1.5 text-sm text-foreground hover:bg-muted transition-colors"
              >
                <Plus className="h-4 w-4" />
                {t('dokumente.fromTemplate')}
              </button>
              <button
                onClick={handleUpload}
                className="flex items-center gap-2 rounded-lg bg-primary px-3 py-1.5 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                <Upload className="h-4 w-4" />
                {t('common.upload')}
              </button>
            </div>

            {/* Breadcrumbs */}
            {breadcrumbs.length > 0 && !activeSpecialView && (
              <div className="flex items-center gap-1 border-b border-border px-6 py-2 text-sm">
                <button
                  onClick={() => navigateToFolder(null)}
                  className="text-muted-foreground hover:text-foreground transition-colors"
                >
                  {t('dokumente.allFiles')}
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
                      {t('dokumente.dragOverlay.title')}
                    </p>
                    <p className="text-xs text-muted-foreground mt-1">
                      {t('dokumente.dragOverlay.subtitle', { folder: getActiveSectionName() })}
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
                          className="group rounded-xl border border-border bg-card p-3 cursor-pointer hover:shadow-[var(--shadow-card-hover)] hover:-translate-y-0.5 transition-all duration-200"
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
                              {t('dokumente.fileCount', { count: folder.file_count })}
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
                          t('dokumente.downloading', { name: file.filename }),
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
                        toast.info(t('dokumente.moveComingSoon'))
                      }
                      onCopy={() =>
                        toast.info(t('dokumente.copyComingSoon'))
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
                      onShareLink={() =>
                        setShareLinkTarget({ id: file.id, name: file.filename })
                      }
                      onOpenInOffice={
                        isOfficeFile(file.mime_type)
                          ? () => handleOpenInOffice(file)
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
                    <span>{t('dokumente.list.name')}</span>
                    <span>{t('dokumente.list.size')}</span>
                    <span>{t('dokumente.list.type')}</span>
                    <span>{t('dokumente.list.date')}</span>
                  </div>
                  {filtered.map((file) => (
                    <FileContextMenu
                      key={file.id}
                      file={file}
                      onOpen={() => setPreviewFile(file)}
                      onDownload={() =>
                        toast.success(
                          t('dokumente.downloading', { name: file.filename }),
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
                        toast.info(t('dokumente.moveComingSoon'))
                      }
                      onCopy={() =>
                        toast.info(t('dokumente.copyComingSoon'))
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
                      onShareLink={() =>
                        setShareLinkTarget({ id: file.id, name: file.filename })
                      }
                      onOpenInOffice={
                        isOfficeFile(file.mime_type)
                          ? () => handleOpenInOffice(file)
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
                  title={t('dokumente.empty.title')}
                  description={
                    search
                      ? t('dokumente.empty.searchHint')
                      : t('dokumente.empty.uploadHint')
                  }
                  action={
                    !search
                      ? { label: t('dokumente.empty.uploadAction'), onClick: handleUpload }
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
                          ? t('dokumente.upload.complete')
                          : u.status === 'error'
                            ? t('common.error')
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
            title={t('dokumente.deleteFile.title')}
            description={t('dokumente.deleteFile.description', { name: deleteTarget?.filename })}
            confirmLabel={t('common.delete')}
            variant="destructive"
            onConfirm={handleDeleteFile}
          />

          {/* Delete Folder Confirm */}
          <ConfirmDialog
            open={!!deleteFolderConfirmId}
            onOpenChange={(open) =>
              !open && setDeleteFolderConfirmId(null)
            }
            title={t('dokumente.deleteFolder.title')}
            description={t('dokumente.deleteFolder.description', { name: deleteFolderTarget?.name })}
            confirmLabel={t('common.delete')}
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

          {/* Template Gallery Dialog */}
          <TemplateGalleryDialog
            open={showTemplateGallery}
            onClose={() => setShowTemplateGallery(false)}
          />

          {/* Share Link Dialog */}
          {shareLinkTarget && (
            <ShareLinkDialog
              open
              onClose={() => setShareLinkTarget(null)}
              fileName={shareLinkTarget.name}
              fileId={shareLinkTarget.id}
            />
          )}
        </div>

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

// ---------------------------------------------------------------------------
// AI Classification Badge
// ---------------------------------------------------------------------------

const CLASSIFICATION_COLORS: Record<ClassificationLevel, string> = {
  öffentlich: 'bg-success-light text-success',
  intern: 'bg-info-light text-info',
  vertraulich: 'bg-error-light text-error',
}

// Classification labels are resolved via t() in the component below
const CLASSIFICATION_LABEL_KEYS: Record<ClassificationLevel, string> = {
  öffentlich: 'dokumente.classification.public',
  intern: 'dokumente.classification.internal',
  vertraulich: 'dokumente.classification.confidential',
}

function classifyByFilename(filename: string): ClassificationLevel {
  const lower = filename.toLowerCase()
  if (/vertrag|gehalt|personal|lohn|kündigung|zeugnis|steuer|sozialversicherung/.test(lower)) return 'vertraulich'
  if (/angebot|bericht|budget|intern|entwurf|protokoll|strategie|planung/.test(lower)) return 'intern'
  return 'öffentlich'
}

function ClassificationBadge({ fileId, filename }: { fileId: string; filename: string }) {
  const { t } = useTranslation()
  const aiDocsEnabled = useAIStore((s) => s.isModuleEnabled('docs'))
  const classifications = useAIStore((s) => s.fileClassifications)
  const setClassification = useAIStore((s) => s.setFileClassification)

  if (!aiDocsEnabled) return null

  let level = classifications[fileId]
  if (!level) {
    level = classifyByFilename(filename)
    // Defer to avoid set-during-render
    setTimeout(() => setClassification(fileId, level!), 0)
  }

  return (
    <span className={`rounded-full px-1.5 py-0 text-[9px] font-medium ${CLASSIFICATION_COLORS[level]}`}>
      {t(CLASSIFICATION_LABEL_KEYS[level])}
    </span>
  )
}

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
      className={`group rounded-xl border bg-card p-3 transition-all duration-200 hover:shadow-[var(--shadow-card-hover)] hover:-translate-y-0.5 cursor-pointer ${
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
        <div className="absolute top-1.5 right-1.5">
          <ClassificationBadge fileId={file.id} filename={file.filename} />
        </div>
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
            {formatDate(file.updated_at)}
          </p>
        </div>
        <div className="flex items-center gap-0.5 shrink-0">
          {file.is_favorite && (
            <Star className="h-3 w-3 fill-warning text-warning" />
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
  const { t } = useTranslation()
  const cat = getMimeCategory(file.mime_type)
  const Icon = fileTypeIcons[cat] || File
  const colorClass = fileTypeColors[cat] || fileTypeColors.other

  const typeLabels: Record<string, string> = {
    pdf: 'PDF',
    word: 'Word',
    excel: 'Excel',
    image: t('dokumente.fileType.image'),
    video: 'Video',
    archive: t('dokumente.fileType.archive'),
    other: t('dokumente.fileType.file'),
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
        <ClassificationBadge fileId={file.id} filename={file.filename} />
        {file.is_favorite && (
          <Star className="h-3 w-3 shrink-0 fill-warning text-warning" />
        )}
      </div>
      <span className="text-xs text-muted-foreground">
        {formatBytes(file.file_size)}
      </span>
      <span className="text-xs text-muted-foreground">
        {typeLabels[cat] ?? cat}
      </span>
      <span className="text-xs text-muted-foreground">
        {formatDate(file.updated_at)}
      </span>
    </div>
  )
}

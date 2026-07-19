/**
 * Context menu for files and folders in the document manager.
 *
 * Uses a custom implementation with portal-rendered menu since
 * @radix-ui/react-context-menu is not installed. Positions the
 * menu at the right-click coordinates and closes on outside click.
 */
import { useState, useEffect, useCallback, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { createPortal } from 'react-dom'
import {
  ExternalLink,
  Download,
  Pencil,
  FolderInput,
  Copy,
  Share2,
  History,
  Trash2,
  Info,
  FolderPlus,
  Upload,
  FileEdit,
  Link2,
  AppWindow,
} from 'lucide-react'
import type { DocumentFile, DocumentFolder } from '@/api/types/document-types'
import { useHasCapability, useScopedCapability } from '@/hooks/useCapability'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface MenuPosition {
  x: number
  y: number
}

interface FileContextMenuProps {
  file: DocumentFile
  children: React.ReactNode
  onOpen: () => void
  onDownload: () => void
  onRename: () => void
  onMove: () => void
  onCopy: () => void
  onShare: () => void
  onVersionHistory: () => void
  onDelete: () => void
  onProperties: () => void
  onEditInOnlyOffice?: () => void
  onShareLink?: () => void
  onOpenInOffice?: () => void
}

interface FolderContextMenuProps {
  folder: DocumentFolder
  children: React.ReactNode
  onOpen: () => void
  onNewSubfolder: () => void
  onUploadHere: () => void
  onRename: () => void
  onDelete: () => void
}

interface MenuItem {
  label: string
  icon: React.ElementType
  onClick: () => void
  variant?: 'default' | 'destructive'
  separator?: boolean
  /** If true: item renders visually disabled (aria-disabled) with a title tooltip. */
  disabled?: boolean
  disabledTitle?: string
}

// ---------------------------------------------------------------------------
// Shared menu renderer
// ---------------------------------------------------------------------------

function ContextMenuContent({
  items,
  position,
  onClose,
}: {
  items: MenuItem[]
  position: MenuPosition
  onClose: () => void
}) {
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        onClose()
      }
    }
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    const handleScroll = () => onClose()

    document.addEventListener('mousedown', handleClickOutside)
    document.addEventListener('keydown', handleEscape)
    window.addEventListener('scroll', handleScroll, true)
    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
      document.removeEventListener('keydown', handleEscape)
      window.removeEventListener('scroll', handleScroll, true)
    }
  }, [onClose])

  // Clamp position so menu doesn't overflow viewport
  useEffect(() => {
    if (!menuRef.current) return
    const rect = menuRef.current.getBoundingClientRect()
    const vw = window.innerWidth
    const vh = window.innerHeight
    if (rect.right > vw) {
      menuRef.current.style.left = `${vw - rect.width - 8}px`
    }
    if (rect.bottom > vh) {
      menuRef.current.style.top = `${vh - rect.height - 8}px`
    }
  }, [position])

  return createPortal(
    <div
      ref={menuRef}
      className="fixed z-[200] min-w-[180px] overflow-hidden rounded-md border bg-popover p-1 text-popover-foreground shadow-md animate-in fade-in-0 zoom-in-95"
      style={{ left: position.x, top: position.y }}
    >
      {items.map((item, i) => {
        const Icon = item.icon
        return (
          <div key={item.label}>
            {item.separator && i > 0 && (
              <div className="-mx-1 my-1 h-px bg-muted" />
            )}
            <button
              aria-disabled={item.disabled || undefined}
              title={item.disabled ? item.disabledTitle : undefined}
              onClick={(e) => {
                e.stopPropagation()
                if (item.disabled) {
                  e.preventDefault()
                  return
                }
                item.onClick()
                onClose()
              }}
              className={`relative flex w-full cursor-default select-none items-center gap-2 rounded-sm px-2 py-1.5 text-sm outline-none transition-colors hover:bg-accent hover:text-accent-foreground ${
                item.variant === 'destructive'
                  ? 'text-destructive hover:text-destructive'
                  : ''
              } ${item.disabled ? 'opacity-50 cursor-not-allowed hover:bg-transparent hover:text-inherit' : ''}`}
            >
              <Icon className="h-4 w-4 shrink-0" />
              {item.label}
            </button>
          </div>
        )
      })}
    </div>,
    document.body,
  )
}

// ---------------------------------------------------------------------------
// File context menu
// ---------------------------------------------------------------------------

export function FileContextMenu({
  file,
  children,
  onOpen,
  onDownload,
  onRename,
  onMove,
  onCopy,
  onShare,
  onVersionHistory,
  onDelete,
  onProperties,
  onEditInOnlyOffice,
  onShareLink,
  onOpenInOffice,
}: FileContextMenuProps) {
  const { t } = useTranslation()
  const [menuPos, setMenuPos] = useState<MenuPosition | null>(null)

  // RBAC capability checks (hooks must be at component top level)
  const canDownload = useHasCapability('documents:file:download')
  const canEdit = useScopedCapability('documents:file:edit', file.owner_id)
  const canDelete = useScopedCapability('documents:file:delete', file.owner_id)
  const canShare = useHasCapability('documents:share:manage')
  const canShareLink = useHasCapability('documents:share_link:create')
  const canVersionRestore = useHasCapability('documents:version:restore')

  const handleContextMenu = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault()
      e.stopPropagation()
      setMenuPos({ x: e.clientX, y: e.clientY })
    },
    [],
  )

  const downloadDisabledTitle = t('rbac.gate.downloadDisabled')

  const items: MenuItem[] = [
    { label: t('dokumente.context.open'), icon: ExternalLink, onClick: onOpen },
    ...(onEditInOnlyOffice && canEdit
      ? [{ label: t('dokumente.context.editInOnlyOffice'), icon: FileEdit, onClick: onEditInOnlyOffice }]
      : []),
    ...(onOpenInOffice
      ? [{ label: t('dokumente.context.openInOffice'), icon: AppWindow, onClick: onOpenInOffice }]
      : []),
    // Download: always visible, disabled without right (Ausnahme-Muster)
    {
      label: t('common.download'),
      icon: Download,
      onClick: onDownload,
      disabled: !canDownload,
      disabledTitle: downloadDisabledTitle,
    },
    // Rename + Move: only with edit right on own files
    ...(canEdit
      ? [
          { label: t('dokumente.context.rename'), icon: Pencil, onClick: onRename, separator: true },
          { label: t('dokumente.context.move'), icon: FolderInput, onClick: onMove },
        ]
      : []),
    // Copy is read-action — always visible
    { label: t('dokumente.context.copy'), icon: Copy, onClick: onCopy, ...(!canEdit ? { separator: true } : {}) },
    // Share: only with share:manage
    ...(canShare
      ? [{ label: t('dokumente.context.share'), icon: Share2, onClick: onShare, separator: canEdit }]
      : []),
    // Share link: only with share_link:create
    ...(canShareLink && onShareLink
      ? [{ label: t('dokumente.context.shareLink'), icon: Link2, onClick: onShareLink }]
      : []),
    // Version history: only with version:restore
    ...(canVersionRestore
      ? [{ label: t('dokumente.context.versionHistory'), icon: History, onClick: onVersionHistory }]
      : []),
    // Delete: only with delete right on own files
    ...(canDelete
      ? [
          {
            label: t('common.delete'),
            icon: Trash2,
            onClick: onDelete,
            variant: 'destructive' as const,
            separator: true,
          },
        ]
      : []),
    {
      label: t('dokumente.context.properties'),
      icon: Info,
      onClick: onProperties,
      separator: true,
    },
  ]

  return (
    <div onContextMenu={handleContextMenu}>
      {children}
      {menuPos && (
        <ContextMenuContent
          items={items}
          position={menuPos}
          onClose={() => setMenuPos(null)}
        />
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Folder context menu
// ---------------------------------------------------------------------------

export function FolderContextMenu({
  folder,
  children,
  onOpen,
  onNewSubfolder,
  onUploadHere,
  onRename,
  onDelete,
}: FolderContextMenuProps) {
  const { t } = useTranslation()
  const [menuPos, setMenuPos] = useState<MenuPosition | null>(null)

  // RBAC capability checks (hooks must be at component top level)
  const canUpload = useHasCapability('documents:file:upload')
  // Ordner-Owner = owner_id
  const canEditFolder = useScopedCapability('documents:file:edit', folder.created_by)
  const canDeleteFolder = useScopedCapability('documents:file:delete', folder.created_by)

  const handleContextMenu = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault()
      e.stopPropagation()
      setMenuPos({ x: e.clientX, y: e.clientY })
    },
    [],
  )

  const items: MenuItem[] = [
    { label: t('dokumente.context.open'), icon: ExternalLink, onClick: onOpen },
    // Neuer Unterordner: requires edit right
    ...(canEditFolder
      ? [
          {
            label: t('dokumente.context.newSubfolder'),
            icon: FolderPlus,
            onClick: onNewSubfolder,
          },
        ]
      : []),
    // Hier hochladen: requires upload right
    ...(canUpload
      ? [{ label: t('dokumente.context.uploadHere'), icon: Upload, onClick: onUploadHere }]
      : []),
    // Umbenennen: requires edit right on this folder's owner
    ...(canEditFolder
      ? [
          {
            label: t('dokumente.context.rename'),
            icon: Pencil,
            onClick: onRename,
            separator: true,
          },
        ]
      : []),
    // Löschen: requires delete right on this folder's owner
    ...(canDeleteFolder
      ? [
          {
            label: t('common.delete'),
            icon: Trash2,
            onClick: onDelete,
            variant: 'destructive' as const,
            separator: true,
          },
        ]
      : []),
  ]

  return (
    <div onContextMenu={handleContextMenu}>
      {children}
      {menuPos && (
        <ContextMenuContent
          items={items}
          position={menuPos}
          onClose={() => setMenuPos(null)}
        />
      )}
    </div>
  )
}

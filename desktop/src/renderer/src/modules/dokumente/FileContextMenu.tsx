/**
 * Context menu for files and folders in the document manager.
 *
 * Uses a custom implementation with portal-rendered menu since
 * @radix-ui/react-context-menu is not installed. Positions the
 * menu at the right-click coordinates and closes on outside click.
 */
import { useState, useEffect, useCallback, useRef } from 'react'
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
              onClick={(e) => {
                e.stopPropagation()
                item.onClick()
                onClose()
              }}
              className={`relative flex w-full cursor-default select-none items-center gap-2 rounded-sm px-2 py-1.5 text-sm outline-none transition-colors hover:bg-accent hover:text-accent-foreground ${
                item.variant === 'destructive'
                  ? 'text-destructive hover:text-destructive'
                  : ''
              }`}
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
  file: _file,
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
  const [menuPos, setMenuPos] = useState<MenuPosition | null>(null)

  const handleContextMenu = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault()
      e.stopPropagation()
      setMenuPos({ x: e.clientX, y: e.clientY })
    },
    [],
  )

  const items: MenuItem[] = [
    { label: 'Oeffnen', icon: ExternalLink, onClick: onOpen },
    ...(onEditInOnlyOffice
      ? [{ label: 'In OnlyOffice bearbeiten', icon: FileEdit, onClick: onEditInOnlyOffice }]
      : []),
    ...(onOpenInOffice
      ? [{ label: 'In Office oeffnen', icon: AppWindow, onClick: onOpenInOffice }]
      : []),
    { label: 'Herunterladen', icon: Download, onClick: onDownload },
    { label: 'Umbenennen', icon: Pencil, onClick: onRename, separator: true },
    { label: 'Verschieben...', icon: FolderInput, onClick: onMove },
    { label: 'Kopieren...', icon: Copy, onClick: onCopy },
    { label: 'Teilen', icon: Share2, onClick: onShare, separator: true },
    ...(onShareLink
      ? [{ label: 'Link teilen', icon: Link2, onClick: onShareLink }]
      : []),
    { label: 'Versionsverlauf', icon: History, onClick: onVersionHistory },
    {
      label: 'Loeschen',
      icon: Trash2,
      onClick: onDelete,
      variant: 'destructive',
      separator: true,
    },
    {
      label: 'Eigenschaften',
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
  folder: _folder,
  children,
  onOpen,
  onNewSubfolder,
  onUploadHere,
  onRename,
  onDelete,
}: FolderContextMenuProps) {
  const [menuPos, setMenuPos] = useState<MenuPosition | null>(null)

  const handleContextMenu = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault()
      e.stopPropagation()
      setMenuPos({ x: e.clientX, y: e.clientY })
    },
    [],
  )

  const items: MenuItem[] = [
    { label: 'Oeffnen', icon: ExternalLink, onClick: onOpen },
    {
      label: 'Neuer Unterordner',
      icon: FolderPlus,
      onClick: onNewSubfolder,
    },
    { label: 'Hochladen hierhin', icon: Upload, onClick: onUploadHere },
    {
      label: 'Umbenennen',
      icon: Pencil,
      onClick: onRename,
      separator: true,
    },
    {
      label: 'Loeschen',
      icon: Trash2,
      onClick: onDelete,
      variant: 'destructive',
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

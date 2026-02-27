/**
 * Drag-and-drop file zone wrapping the chat message input area.
 *
 * Shows a visual overlay when files are dragged over.
 * On drop, creates file attachment objects for preview.
 */
import { useState, useCallback, type DragEvent } from 'react'
import { Upload } from 'lucide-react'

export interface AttachedFile {
  id: string
  name: string
  size: number
  type: string
  isImage: boolean
  previewUrl?: string
}

interface FileDropZoneProps {
  children: React.ReactNode
  onFilesAdded: (files: AttachedFile[]) => void
  disabled?: boolean
}

let fileIdCounter = 0

function fileToAttachment(file: File): AttachedFile {
  const isImage = file.type.startsWith('image/')
  return {
    id: `file-${++fileIdCounter}`,
    name: file.name,
    size: file.size,
    type: file.type,
    isImage,
    previewUrl: isImage ? URL.createObjectURL(file) : undefined,
  }
}

export function FileDropZone({ children, onFilesAdded, disabled }: FileDropZoneProps) {
  const [isDragging, setIsDragging] = useState(false)

  const handleDragEnter = useCallback(
    (e: DragEvent) => {
      e.preventDefault()
      e.stopPropagation()
      if (!disabled) setIsDragging(true)
    },
    [disabled]
  )

  const handleDragLeave = useCallback((e: DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (e.currentTarget === e.target) setIsDragging(false)
  }, [])

  const handleDragOver = useCallback((e: DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
  }, [])

  const handleDrop = useCallback(
    (e: DragEvent) => {
      e.preventDefault()
      e.stopPropagation()
      setIsDragging(false)
      if (disabled) return

      const files = Array.from(e.dataTransfer.files)
      if (files.length > 0) {
        onFilesAdded(files.map(fileToAttachment))
      }
    },
    [disabled, onFilesAdded]
  )

  return (
    <div
      className="relative"
      onDragEnter={handleDragEnter}
      onDragLeave={handleDragLeave}
      onDragOver={handleDragOver}
      onDrop={handleDrop}
    >
      {children}

      {isDragging && (
        <div className="absolute inset-0 z-10 flex items-center justify-center rounded-lg border-2 border-dashed border-primary bg-primary/5 backdrop-blur-[1px]">
          <div className="flex flex-col items-center gap-2 text-primary">
            <Upload className="h-8 w-8" />
            <p className="text-sm font-medium">Dateien hier ablegen</p>
          </div>
        </div>
      )}
    </div>
  )
}

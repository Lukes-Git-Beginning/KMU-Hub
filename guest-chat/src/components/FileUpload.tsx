import { useCallback, useRef, useState } from 'react'
import { apiClient } from '../api/client'
import type { GuestChannelConfig } from '../api/types'

interface FileUploadProps {
  channelId: string
  config: GuestChannelConfig
  disabled: boolean
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function buildAcceptString(allowedTypes: string): string {
  // Parse "image/*, application/pdf" or similar
  return allowedTypes || 'image/*,application/pdf'
}

export function FileUpload({ channelId, config, disabled }: FileUploadProps) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [preview, setPreview] = useState<string | null>(null)
  const [uploading, setUploading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleClick = useCallback(() => {
    inputRef.current?.click()
  }, [])

  const handleFileSelect = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0]
      if (!file) return

      // Reset input so same file can be re-selected
      e.target.value = ''

      // Validate size
      const maxBytes = config.max_file_size_mb * 1024 * 1024
      if (file.size > maxBytes) {
        setError(`Datei zu gross (max ${config.max_file_size_mb} MB)`)
        return
      }

      // Validate type
      const allowed = config.allowed_file_types.split(',').map((t) => t.trim())
      const typeOk = allowed.some((type) => {
        if (type.endsWith('/*')) {
          return file.type.startsWith(type.replace('/*', '/'))
        }
        return file.type === type
      })
      if (!typeOk) {
        setError('Nur Bilder und PDF erlaubt')
        return
      }

      setError(null)
      setSelectedFile(file)

      // Generate preview for images
      if (file.type.startsWith('image/')) {
        const reader = new FileReader()
        reader.onload = (ev) => setPreview(ev.target?.result as string)
        reader.readAsDataURL(file)
      } else {
        setPreview(null)
      }
    },
    [config],
  )

  const handleCancel = useCallback(() => {
    setSelectedFile(null)
    setPreview(null)
    setError(null)
  }, [])

  const handleUpload = useCallback(async () => {
    if (!selectedFile) return
    setUploading(true)
    try {
      await apiClient.uploadFile(channelId, selectedFile)
      setSelectedFile(null)
      setPreview(null)
    } catch {
      setError('Upload fehlgeschlagen')
    } finally {
      setUploading(false)
    }
  }, [selectedFile, channelId])

  return (
    <>
      <input
        ref={inputRef}
        type="file"
        accept={buildAcceptString(config.allowed_file_types)}
        onChange={handleFileSelect}
        hidden
      />
      <button className="icon-btn" onClick={handleClick} disabled={disabled} title="Datei anhaengen">
        {/* Paperclip SVG */}
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <path d="M21.44 11.05l-9.19 9.19a6 6 0 01-8.49-8.49l9.19-9.19a4 4 0 015.66 5.66l-9.2 9.19a2 2 0 01-2.83-2.83l8.49-8.48" />
        </svg>
      </button>

      {error && !selectedFile && (
        <div className="file-preview">
          <span style={{ color: 'var(--error)', fontSize: '0.875rem' }}>{error}</span>
          <button className="file-preview-cancel" onClick={handleCancel}>&#10005;</button>
        </div>
      )}

      {selectedFile && (
        <div className="file-preview">
          {preview ? (
            <img src={preview} alt={selectedFile.name} />
          ) : (
            <span>&#128196;</span>
          )}
          <div className="file-preview-info">
            <div className="file-preview-name">{selectedFile.name}</div>
            <div className="file-preview-size">{formatSize(selectedFile.size)}</div>
            {error && <div style={{ color: 'var(--error)', fontSize: '0.75rem' }}>{error}</div>}
          </div>
          {uploading ? (
            <div className="spinner" style={{ width: 20, height: 20 }} />
          ) : (
            <>
              <button className="icon-btn send" onClick={handleUpload} title="Hochladen" style={{ width: 32, height: 32 }}>
                &#10003;
              </button>
              <button className="file-preview-cancel" onClick={handleCancel}>&#10005;</button>
            </>
          )}
        </div>
      )}
    </>
  )
}

/* eslint-disable react-refresh/only-export-components */
/**
 * OnlyOffice WOPI editor iframe wrapper.
 *
 * Renders a full-screen overlay with an embedded OnlyOffice editor via WOPI
 * protocol. Collaboration presence (colored cursors, user avatars) is provided
 * natively by OnlyOffice through WOPI CheckFileInfo UserId/UserFriendlyName.
 */
import { useState, useEffect, useCallback, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { X, FileText, AlertTriangle, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { API_BASE_URL } from '@/lib/constants'

// ---------------------------------------------------------------------------
// WOPI-editable MIME types
// ---------------------------------------------------------------------------

const ONLYOFFICE_EDITABLE = new Set([
  'application/vnd.openxmlformats-officedocument.wordprocessingml.document', // docx
  'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',      // xlsx
  'application/vnd.openxmlformats-officedocument.presentationml.presentation', // pptx
  'application/msword',                                                       // doc
  'application/vnd.ms-excel',                                                  // xls
  'application/vnd.ms-powerpoint',                                             // ppt
  'application/vnd.oasis.opendocument.text',                                   // odt
  'application/vnd.oasis.opendocument.spreadsheet',                            // ods
  'application/vnd.oasis.opendocument.presentation',                           // odp
  'text/csv',
  'text/plain',
])

/**
 * Check whether a file's MIME type is editable in OnlyOffice.
 */
export function isOnlyOfficeEditable(mimeType: string): boolean {
  return ONLYOFFICE_EDITABLE.has(mimeType)
}

/**
 * Check whether a filename extension is editable in OnlyOffice.
 */
export function isOnlyOfficeEditableByExtension(filename: string): boolean {
  const ext = filename.split('.').pop()?.toLowerCase()
  if (!ext) return false
  return EDITABLE_EXTENSIONS.has(ext)
}

const EDITABLE_EXTENSIONS = new Set([
  'docx', 'doc', 'odt',
  'xlsx', 'xls', 'ods',
  'pptx', 'ppt', 'odp',
  'csv', 'txt',
])

// ---------------------------------------------------------------------------
// Editor type mapping
// ---------------------------------------------------------------------------

type EditorType = 'word' | 'cell' | 'slide'

function getEditorType(fileName: string): EditorType {
  const ext = fileName.split('.').pop()?.toLowerCase() ?? ''
  switch (ext) {
    case 'xlsx':
    case 'xls':
    case 'ods':
    case 'csv':
      return 'cell'
    case 'pptx':
    case 'ppt':
    case 'odp':
      return 'slide'
    default:
      return 'word'
  }
}

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface OnlyOfficeEditorProps {
  fileId: string
  fileName: string
  wopiToken: string
  wopiTokenTTL: number
  onClose: () => void
  onVersionCreated?: () => void
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

const ONLYOFFICE_URL =
  import.meta.env.VITE_ONLYOFFICE_URL || 'http://localhost:8088'
/** Same resolved address the API client uses -- re-reading the env var here
 *  would point WOPI callbacks at localhost in the web build. */
const API_URL = API_BASE_URL

/**
 * Full-screen overlay hosting the OnlyOffice editor via WOPI iframe.
 *
 * Collaboration features (colored cursors, user avatars) are provided natively
 * by OnlyOffice. This works because the WOPI CheckFileInfo response (built in
 * Plan 11-04) returns correct UserId and UserFriendlyName fields from the
 * WOPI token. No additional frontend code is needed for cursor/avatar rendering.
 */
export function OnlyOfficeEditor({
  fileId,
  fileName,
  wopiToken,
  wopiTokenTTL,
  onClose,
  onVersionCreated,
}: OnlyOfficeEditorProps) {
  const { t } = useTranslation()
  const [isLoading, setIsLoading] = useState(true)
  const [showTokenWarning, setShowTokenWarning] = useState(false)
  const tokenWarningTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const iframeRef = useRef<HTMLIFrameElement>(null)

  // Construct WOPI iframe URL
  const editorType = getEditorType(fileName)
  const wopiSrc = encodeURIComponent(`${API_URL}/wopi/files/${fileId}`)
  const iframeUrl = `${ONLYOFFICE_URL}/hosting/wopi/${editorType}/edit?WOPISrc=${wopiSrc}&access_token=${encodeURIComponent(wopiToken)}&access_token_ttl=${wopiTokenTTL}`

  // Token expiry warning (30 min before expiration)
  useEffect(() => {
    const now = Math.floor(Date.now() / 1000)
    const expiresAt = wopiTokenTTL
    const timeUntilWarning = (expiresAt - now - 30 * 60) * 1000 // 30 min before

    if (timeUntilWarning > 0) {
      tokenWarningTimerRef.current = setTimeout(() => {
        setShowTokenWarning(true)
      }, timeUntilWarning)
    } else if (expiresAt - now > 0) {
      // Less than 30 min remaining, show warning immediately
      setShowTokenWarning(true)
    }

    return () => {
      if (tokenWarningTimerRef.current) {
        clearTimeout(tokenWarningTimerRef.current)
      }
    }
  }, [wopiTokenTTL])

  // PostMessage listener for OnlyOffice
  useEffect(() => {
    function handleMessage(event: MessageEvent) {
      // Verify origin
      try {
        const onlyOfficeOrigin = new URL(ONLYOFFICE_URL).origin
        if (event.origin !== onlyOfficeOrigin) return
      } catch {
        return
      }

      const data = event.data
      if (typeof data === 'object' && data !== null) {
        // Handle UI_Close from OnlyOffice
        if (data.MessageId === 'UI_Close' || data.type === 'UI_Close') {
          // eslint-disable-next-line react-hooks/immutability -- calling handler from event listener
          handleClose()
        }
      }
    }

    window.addEventListener('message', handleMessage)
    return () => window.removeEventListener('message', handleMessage)
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // Escape key to close
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        handleClose()
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const handleIframeLoad = useCallback(() => {
    setIsLoading(false)
  }, [])

  const handleClose = useCallback(() => {
    // Notify parent that a version may have been created
    // (auto-version is created when last editor closes)
    onVersionCreated?.()
    onClose()
  }, [onClose, onVersionCreated])

  return (
    <Dialog open onOpenChange={(open) => { if (!open) handleClose() }}>
      <DialogContent className="gap-0 p-0 max-w-full w-screen h-screen rounded-none border-0 [&>button:last-child]:hidden">
    <div className="flex flex-col h-full">
      {/* Top bar */}
      <div className="flex items-center justify-between border-b border-border bg-card px-4 py-2 shrink-0">
        <div className="flex items-center gap-3 min-w-0">
          <FileText className="h-5 w-5 text-primary shrink-0" />
          <DialogTitle className="text-sm font-medium text-foreground truncate">
            {fileName}
          </DialogTitle>
          <DialogDescription className="sr-only">
            {t('dokumente.editor.description', { name: fileName })}
          </DialogDescription>
          <span className="rounded-full bg-primary/10 px-2.5 py-0.5 text-[11px] font-medium text-primary shrink-0">
            {t('dokumente.editor.editingActive')}
          </span>
        </div>

        <div className="flex items-center gap-2 shrink-0">
          {/* Token expiry warning */}
          {showTokenWarning && (
            <div className="flex items-center gap-1.5 rounded-md bg-warning/10 px-3 py-1.5 text-xs text-warning">
              <AlertTriangle className="h-3.5 w-3.5" />
              <span>{t('dokumente.editor.sessionExpiring')}</span>
            </div>
          )}

          <Button
            variant="outline"
            size="sm"
            onClick={handleClose}
          >
            <X className="mr-1.5 h-4 w-4" />
            {t('common.close')}
          </Button>
        </div>
      </div>

      {/* Editor iframe */}
      <div className="flex-1 relative">
        {/* Loading state */}
        {isLoading && (
          <div className="absolute inset-0 flex items-center justify-center bg-background z-10">
            <div className="text-center">
              <Loader2 className="h-8 w-8 mx-auto mb-3 text-primary animate-spin" />
              <p className="text-sm text-muted-foreground">
                {t('dokumente.editor.loading')}
              </p>
            </div>
          </div>
        )}

        <iframe
          ref={iframeRef}
          src={iframeUrl}
          className="h-full w-full border-0"
          onLoad={handleIframeLoad}
          allow="clipboard-read; clipboard-write"
          sandbox="allow-scripts allow-same-origin allow-forms allow-popups allow-downloads"
          title={`OnlyOffice Editor - ${fileName}`}
        />
      </div>
    </div>
      </DialogContent>
    </Dialog>
  )
}

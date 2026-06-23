/**
 * ScreenSourcePicker (Wave 4.1) — Electron screen-source selection modal.
 *
 * Shows a grid of available screens and windows (with thumbnails) so the user
 * can choose what to share before LiveKit calls getDisplayMedia().
 *
 * Flow:
 *   1. Caller opens this modal (showPicker=true).
 *   2. Component calls window.electronAPI.screenshare.getSources() via IPC.
 *   3. User selects a source → onSelect(sourceId) is called.
 *   4. Caller passes the sourceId to LiveKit's setScreenShareEnabled via
 *      ScreenShareCaptureOptions.sourceId.
 *   5. Electron's setDisplayMediaRequestHandler intercepts getDisplayMedia()
 *      and returns the correct DesktopCapturerSource.
 *
 * Non-Electron fallback: when window.electronAPI is not present (e.g. web)
 * the component calls onSelect(null) immediately so LiveKit falls back to the
 * browser's native getDisplayMedia() UI.
 *
 * Audio (4.2): System/loopback audio is enabled automatically on Windows via
 * setDisplayMediaRequestHandler (audio: 'loopback'). On macOS audio loopback
 * requires a kernel extension we don't bundle — a notice is shown.
 */
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface ScreenSource {
  id: string
  name: string
  thumbnailDataUrl: string
  type: 'screen' | 'window'
}

export interface ScreenSourcePickerProps {
  open: boolean
  onSelect: (sourceId: string) => void
  onCancel: () => void
}

// ---------------------------------------------------------------------------
// Helper: detect Electron context
// ---------------------------------------------------------------------------

function isElectron(): boolean {
  return (
    typeof window !== 'undefined' &&
    typeof window.electronAPI !== 'undefined' &&
    typeof window.electronAPI.screenshare !== 'undefined'
  )
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function ScreenSourcePicker({ open, onSelect, onCancel }: ScreenSourcePickerProps) {
  const { t } = useTranslation()
  const [sources, setSources] = useState<ScreenSource[]>([])
  const [loading, setLoading] = useState(false)
  const [selected, setSelected] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  // Fetch sources when modal opens
  useEffect(() => {
    if (!open) {
      setSources([])
      setSelected(null)
      setError(null)
      return
    }

    if (!isElectron()) {
      // Non-Electron path: immediately call back with no sourceId so the
      // browser's native getDisplayMedia picker is shown instead.
      onSelect('')
      return
    }

    setLoading(true)
    setError(null)

    window.electronAPI.screenshare.getSources()
      .then((srcs) => {
        setSources(srcs)
        // Auto-select primary screen
        const primary = srcs.find((s) => s.type === 'screen')
        if (primary) setSelected(primary.id)
      })
      .catch(() => {
        setError(t('features.video.screenPicker.error'))
      })
      .finally(() => {
        setLoading(false)
      })
  // onSelect is stable (useCallback at call site), no dep needed
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, t])

  const handleConfirm = useCallback(() => {
    if (selected) {
      onSelect(selected)
    }
  }, [selected, onSelect])

  if (!open) return null

  const screens = sources.filter((s) => s.type === 'screen')
  const windows = sources.filter((s) => s.type === 'window')

  const isWindows = typeof navigator !== 'undefined' && navigator.userAgent.toLowerCase().includes('win')

  return (
    /* Backdrop */
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
      role="dialog"
      aria-modal="true"
      aria-label={t('features.video.screenPicker.title')}
    >
      <div className="relative flex h-[80vh] w-[680px] max-w-[95vw] flex-col overflow-hidden rounded-2xl border border-zinc-700 bg-zinc-900 shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-zinc-800 px-5 py-4">
          <h2 className="text-base font-semibold text-zinc-100">
            {t('features.video.screenPicker.title')}
          </h2>
          <button
            onClick={onCancel}
            className="flex h-7 w-7 items-center justify-center rounded-lg text-zinc-400 transition-opacity hover:text-zinc-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-zinc-400"
            aria-label={t('features.video.screenPicker.cancel')}
          >
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
              <line x1="1" y1="1" x2="13" y2="13" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" />
              <line x1="13" y1="1" x2="1" y2="13" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" />
            </svg>
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto px-5 py-4">
          {loading && (
            <div className="flex h-48 items-center justify-center">
              <span className="text-sm text-zinc-400">{t('features.video.screenPicker.loading')}</span>
            </div>
          )}

          {error && (
            <div className="flex h-48 items-center justify-center">
              <span className="text-sm text-red-400">{error}</span>
            </div>
          )}

          {!loading && !error && (
            <div className="space-y-5">
              {/* Screens group */}
              {screens.length > 0 && (
                <div>
                  <p className="mb-2.5 text-xs font-medium uppercase tracking-wide text-zinc-500">
                    {t('features.video.screenPicker.groupScreens')}
                  </p>
                  <div className="grid grid-cols-2 gap-3">
                    {screens.map((src) => (
                      <SourceTile
                        key={src.id}
                        source={src}
                        selected={selected === src.id}
                        onSelect={() => setSelected(src.id)}
                      />
                    ))}
                  </div>
                </div>
              )}

              {/* Windows group */}
              {windows.length > 0 && (
                <div>
                  <p className="mb-2.5 text-xs font-medium uppercase tracking-wide text-zinc-500">
                    {t('features.video.screenPicker.groupWindows')}
                  </p>
                  <div className="grid grid-cols-3 gap-3">
                    {windows.map((src) => (
                      <SourceTile
                        key={src.id}
                        source={src}
                        selected={selected === src.id}
                        onSelect={() => setSelected(src.id)}
                      />
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Audio notice (4.2) */}
        <div className="border-t border-zinc-800 px-5 py-2.5">
          <p className="text-xs text-zinc-500">
            {isWindows
              ? t('features.video.screenPicker.audioNoteWindows')
              : t('features.video.screenPicker.audioNoteMac')}
          </p>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-2.5 border-t border-zinc-800 px-5 py-3.5">
          <button
            onClick={onCancel}
            className="rounded-lg border border-zinc-700 bg-zinc-800 px-4 py-2 text-sm font-medium text-zinc-300 transition-colors hover:bg-zinc-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-zinc-400"
          >
            {t('features.video.screenPicker.cancel')}
          </button>
          <button
            onClick={handleConfirm}
            disabled={!selected}
            className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400 disabled:cursor-not-allowed disabled:opacity-40"
          >
            {t('features.video.screenPicker.share')}
          </button>
        </div>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// SourceTile subcomponent
// ---------------------------------------------------------------------------

interface SourceTileProps {
  source: ScreenSource
  selected: boolean
  onSelect: () => void
}

function SourceTile({ source, selected, onSelect }: SourceTileProps) {
  return (
    <button
      onClick={onSelect}
      onDoubleClick={onSelect}
      className={cn(
        'group relative flex flex-col overflow-hidden rounded-xl border-2 text-left transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400',
        selected
          ? 'border-blue-500 bg-blue-500/10'
          : 'border-zinc-700 bg-zinc-800 hover:border-zinc-600',
      )}
    >
      {/* Thumbnail */}
      <div className="aspect-video w-full overflow-hidden bg-zinc-950">
        {source.thumbnailDataUrl ? (
          <img
            src={source.thumbnailDataUrl}
            alt=""
            className="h-full w-full object-cover"
            draggable={false}
          />
        ) : (
          <div className="flex h-full w-full items-center justify-center">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" className="text-zinc-600" aria-hidden="true">
              <rect x="2" y="3" width="20" height="14" rx="2" />
              <path d="M8 21h8M12 17v4" />
            </svg>
          </div>
        )}
      </div>

      {/* Label */}
      <div className="px-2.5 py-2">
        <p className="truncate text-xs font-medium text-zinc-200">{source.name}</p>
      </div>

      {/* Selected ring overlay */}
      {selected && (
        <div className="absolute right-2 top-2 flex h-5 w-5 items-center justify-center rounded-full bg-blue-500">
          <svg width="10" height="10" viewBox="0 0 10 10" fill="none" aria-hidden="true">
            <polyline points="1.5,5 4,7.5 8.5,2.5" stroke="white" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        </div>
      )}
    </button>
  )
}

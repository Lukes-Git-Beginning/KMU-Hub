/**
 * Screenshare IPC handlers (Wave 4.1)
 *
 * Provides:
 *  - screenshare:get-sources  → returns array of DesktopCapturerSource objects
 *                                (id, name, thumbnail as data-URL)
 *  - setDisplayMediaRequestHandler on the default session so Electron passes
 *    the user-selected sourceId back through getUserMedia when the renderer
 *    calls getDisplayMedia().
 *
 * Architecture:
 *   Renderer calls window.electronAPI.screenshare.getSources() via contextBridge.
 *   Renderer shows the custom picker modal, user picks a source.
 *   Renderer calls setScreenShareEnabled(true) → LiveKit calls getDisplayMedia.
 *   LiveKit passes sourceId in the video constraint (chromeMediaSourceId) which
 *   Electron routes through setDisplayMediaRequestHandler.
 *
 * Security: contextIsolation stays ON, nodeIntegration stays OFF.
 * The handler only allows the renderer to see source metadata (name + thumbnail).
 * The actual media stream is obtained through the standard getDisplayMedia path.
 *
 * Audio sharing (4.2):
 *   On Windows, 'loopback' captures system audio. On macOS it requires a
 *   third-party kernel extension (e.g. BlackHole) which we do not bundle —
 *   audio: false is used there. Linux: PulseAudio/PipeWire loopback depends
 *   on system config; we set audio: 'loopback' and let Electron decide.
 */
import { ipcMain, desktopCapturer, session } from 'electron'

export interface ScreenSourceInfo {
  id: string
  name: string
  /** PNG data URL of the thumbnail (192×108). Empty string when unavailable. */
  thumbnailDataUrl: string
  /** 'screen' or 'window' */
  type: 'screen' | 'window'
}

// lean: module-level cache for the renderer-chosen sourceId; cleared on use.
// The renderer sends 'screenshare:set-source' with the picked id immediately
// before calling setScreenShareEnabled() so the handler reads it synchronously.
// Upgrade: per-webContents map if multi-window screenshare is needed.
let _pendingSourceId: string | null = null

export function registerScreenshareHandlers(): void {
  // -------------------------------------------------------------------------
  // IPC: renderer asks for the available capture sources
  // -------------------------------------------------------------------------
  ipcMain.handle('screenshare:get-sources', async (): Promise<ScreenSourceInfo[]> => {
    const sources = await desktopCapturer.getSources({
      types: ['screen', 'window'],
      thumbnailSize: { width: 192, height: 108 },
      fetchWindowIcons: false,
    })

    return sources.map((src) => ({
      id: src.id,
      name: src.name,
      thumbnailDataUrl: src.thumbnail ? src.thumbnail.toDataURL() : '',
      type: src.id.startsWith('screen:') ? 'screen' : 'window',
    }))
  })

  // -------------------------------------------------------------------------
  // IPC: renderer stores the chosen sourceId before calling LiveKit.
  // The setDisplayMediaRequestHandler reads this synchronously to avoid
  // an async desktopCapturer.getSources() race inside the handler callback.
  // -------------------------------------------------------------------------
  ipcMain.handle('screenshare:set-source', (_event, sourceId: string) => {
    _pendingSourceId = sourceId || null
  })

  // -------------------------------------------------------------------------
  // setDisplayMediaRequestHandler: intercepts getDisplayMedia() calls from the
  // renderer and returns the source the user already selected in our modal.
  //
  // Priority:
  //   1. _pendingSourceId (set via screenshare:set-source just before LiveKit call) — fast path
  //   2. chromeMediaSourceId in request constraints — LiveKit-embedded path
  //   3. Fall back to primary screen
  //
  // Electron's desktopCapturer bypass path:
  //   When LiveKit sees sourceId in ScreenShareCaptureOptions it passes:
  //   video: { mandatory: { chromeMediaSource: 'desktop', chromeMediaSourceId: id } }
  // -------------------------------------------------------------------------
  session.defaultSession.setDisplayMediaRequestHandler(
    (request, callback) => {
      // Prefer the explicitly cached sourceId (renderer called screenshare:set-source).
      const cachedId = _pendingSourceId
      _pendingSourceId = null // consume once

      // Also check constraints for the chromeMediaSourceId LiveKit embeds.
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const videoConstraints = request.videoRequested ? (request as any).constraints?.video : undefined
      const mandatory = videoConstraints?.mandatory as Record<string, string> | undefined
      const constraintSourceId = mandatory?.chromeMediaSourceId

      const requestedSourceId = cachedId ?? constraintSourceId ?? null

      // Determine whether to include audio (loopback on Windows/Linux).
      // macOS loopback audio requires a kernel ext — we skip it.
      const audioCapture = process.platform !== 'darwin' ? 'loopback' : undefined

      if (requestedSourceId) {
        desktopCapturer
          .getSources({ types: ['screen', 'window'], thumbnailSize: { width: 1, height: 1 } })
          .then((sources) => {
            const chosen = sources.find((s) => s.id === requestedSourceId)
            if (chosen) {
              callback({ video: chosen, audio: audioCapture })
            } else {
              // sourceId not found (e.g. window closed) → fall back to primary screen
              const primaryScreen = sources.find((s) => s.id.startsWith('screen:'))
              callback({ video: primaryScreen ?? sources[0], audio: audioCapture })
            }
          })
          .catch(() => callback({}))
        return
      }

      // No sourceId — fall back to the primary screen (e.g. non-Electron browser path)
      desktopCapturer
        .getSources({ types: ['screen'], thumbnailSize: { width: 1, height: 1 } })
        .then((sources) => {
          callback({ video: sources[0] ?? undefined, audio: audioCapture })
        })
        .catch(() => callback({}))
    },
    { useSystemPicker: false },
  )
}

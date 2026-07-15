/**
 * Shared video-device UI: the live camera preview and the device-preference
 * controls (microphone / speaker / camera / virtual background). Both are backed
 * by the personal `videoPrefs` store so the in-page Settings tab and the module
 * settings overlay stay in sync.
 */
import { useCallback, useEffect, useRef, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { VideoOff, Camera, CameraOff } from 'lucide-react'
import { cn } from '@/lib'
import { useVideoPrefsStore, type VideoBackground } from '@/stores/videoPrefs'

// ---------------------------------------------------------------------------
// CameraPreview — real getUserMedia preview with a graceful placeholder.
// ---------------------------------------------------------------------------

type PreviewStatus = 'idle' | 'active' | 'error' | 'unsupported'

export function CameraPreview() {
  const { t } = useTranslation()
  const videoRef = useRef<HTMLVideoElement>(null)
  const streamRef = useRef<MediaStream | null>(null)
  const [status, setStatus] = useState<PreviewStatus>('idle')

  const stop = useCallback(() => {
    streamRef.current?.getTracks().forEach((track) => track.stop())
    streamRef.current = null
    if (videoRef.current) videoRef.current.srcObject = null
    setStatus('idle')
  }, [])

  const start = useCallback(async () => {
    if (!navigator.mediaDevices?.getUserMedia) {
      setStatus('unsupported')
      return
    }
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ video: true, audio: false })
      streamRef.current = stream
      // The <video> is always mounted, so the ref is available here.
      if (videoRef.current) videoRef.current.srcObject = stream
      setStatus('active')
    } catch {
      setStatus('error')
    }
  }, [])

  // Always release the camera when the component unmounts.
  useEffect(() => () => stop(), [stop])

  const active = status === 'active'

  return (
    <div className="relative flex aspect-video items-center justify-center overflow-hidden rounded-xl border border-border bg-muted/30">
      <video
        ref={videoRef}
        autoPlay
        playsInline
        muted
        className={cn('h-full w-full object-cover', !active && 'invisible')}
      />

      {!active && (
        <div className="absolute inset-0 flex flex-col items-center justify-center p-4 text-center text-muted-foreground">
          <VideoOff className="mb-2 h-8 w-8 opacity-50" />
          <p className="mb-3 text-xs">
            {status === 'error'
              ? t('video.settings.previewError')
              : status === 'unsupported'
                ? t('video.settings.previewUnsupported')
                : t('video.settings.kameraVorschau')}
          </p>
          {status !== 'unsupported' && (
            <button
              type="button"
              onClick={start}
              className="inline-flex items-center gap-1.5 rounded-lg border border-border bg-card px-3 py-1.5 text-xs font-medium text-foreground transition-colors hover:bg-muted"
            >
              <Camera className="h-3.5 w-3.5" />
              {status === 'error' ? t('video.settings.previewRetry') : t('video.settings.previewStart')}
            </button>
          )}
        </div>
      )}

      {active && (
        <button
          type="button"
          onClick={stop}
          className="absolute bottom-2 right-2 inline-flex items-center gap-1.5 rounded-lg bg-background/80 px-2.5 py-1 text-xs font-medium text-foreground backdrop-blur transition-colors hover:bg-background"
        >
          <CameraOff className="h-3.5 w-3.5" />
          {t('video.settings.previewStop')}
        </button>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Device controls — microphone / speaker / camera / background.
// ---------------------------------------------------------------------------

function SelectField({
  label,
  value,
  onChange,
  children,
}: {
  label: string
  value: string
  onChange: (v: string) => void
  children: ReactNode
}) {
  return (
    <div>
      <label className="mb-1 block text-xs text-muted-foreground">{label}</label>
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
      >
        {children}
      </select>
    </div>
  )
}

/**
 * Device preference controls. Pass a `preview` node (the CameraPreview) to render
 * it under the camera selector — used in the in-page tab where there is room; the
 * overlay panel renders the compact controls without a preview.
 */
export function VideoDevicePrefsControls({ preview }: { preview?: ReactNode }) {
  const { t } = useTranslation()
  const audioInput = useVideoPrefsStore((s) => s.audioInput)
  const audioOutput = useVideoPrefsStore((s) => s.audioOutput)
  const videoInput = useVideoPrefsStore((s) => s.videoInput)
  const background = useVideoPrefsStore((s) => s.background)
  const setAudioInput = useVideoPrefsStore((s) => s.setAudioInput)
  const setAudioOutput = useVideoPrefsStore((s) => s.setAudioOutput)
  const setVideoInput = useVideoPrefsStore((s) => s.setVideoInput)
  const setBackground = useVideoPrefsStore((s) => s.setBackground)

  const backgrounds: { id: VideoBackground; labelKey: string }[] = [
    { id: 'none', labelKey: 'video.settings.bg.none' },
    { id: 'blur', labelKey: 'video.settings.bg.blur' },
    { id: 'office', labelKey: 'video.settings.bg.office' },
    { id: 'nature', labelKey: 'video.settings.bg.nature' },
  ]

  return (
    <div className="space-y-6">
      <div>
        <h3 className="mb-3 text-sm font-medium text-foreground">{t('video.settings.audio')}</h3>
        <div className="space-y-3">
          <SelectField label={t('video.settings.mikrofon')} value={audioInput} onChange={setAudioInput}>
            <option value="default">{t('video.settings.audio.default')}</option>
            <option value="headset">{t('video.settings.audio.headset')}</option>
            <option value="webcam">{t('video.settings.audio.webcam')}</option>
          </SelectField>
          <SelectField label={t('video.settings.lautsprecher')} value={audioOutput} onChange={setAudioOutput}>
            <option value="default">{t('video.settings.output.default')}</option>
            <option value="headset">{t('video.settings.output.headset')}</option>
            <option value="speaker">{t('video.settings.output.speaker')}</option>
          </SelectField>
        </div>
      </div>

      <div>
        <h3 className="mb-3 text-sm font-medium text-foreground">{t('video.settings.video')}</h3>
        <div className="space-y-3">
          <SelectField label={t('video.settings.kamera')} value={videoInput} onChange={setVideoInput}>
            <option value="default">{t('video.settings.video.default')}</option>
            <option value="hd">{t('video.settings.video.hd')}</option>
            <option value="external">{t('video.settings.video.external')}</option>
          </SelectField>
          {preview}
        </div>
      </div>

      <div>
        <h3 className="mb-3 text-sm font-medium text-foreground">{t('video.settings.hintergrund')}</h3>
        <div className="grid grid-cols-4 gap-2">
          {backgrounds.map((bg) => (
            <button
              key={bg.id}
              type="button"
              onClick={() => setBackground(bg.id)}
              className={cn(
                'rounded-lg border p-3 text-center text-xs transition-colors',
                background === bg.id
                  ? 'border-primary bg-primary/10 font-medium text-primary'
                  : 'border-border text-muted-foreground hover:bg-muted',
              )}
            >
              {t(bg.labelKey)}
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}

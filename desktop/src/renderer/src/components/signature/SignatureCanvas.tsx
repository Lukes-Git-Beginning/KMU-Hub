import { useRef, useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { Eraser, Check } from 'lucide-react'

interface SignatureCanvasProps {
  width?: number
  height?: number
  onSave?: (dataUrl: string) => void
  initialData?: string
  className?: string
  /** i18n key for the hint text (default: 'rapporte.signature.hint') */
  hintKey?: string
  /** i18n key for the clear button (default: 'rapporte.signature.clear') */
  clearKey?: string
}

export default function SignatureCanvas({
  width = 400,
  height = 200,
  onSave,
  initialData,
  className = '',
  hintKey = 'rapporte.signature.hint',
  clearKey = 'rapporte.signature.clear',
}: SignatureCanvasProps) {
  const { t } = useTranslation()
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const [isDrawing, setIsDrawing] = useState(false)
  const [hasStrokes, setHasStrokes] = useState(false)
  const lastPoint = useRef<{ x: number; y: number } | null>(null)

  // Set up canvas with DPR scaling
  const setupCanvas = useCallback(() => {
    const canvas = canvasRef.current
    if (!canvas) return

    const dpr = window.devicePixelRatio || 1
    canvas.width = width * dpr
    canvas.height = height * dpr
    canvas.style.width = `${width}px`
    canvas.style.height = `${height}px`

    const ctx = canvas.getContext('2d')
    if (!ctx) return

    ctx.scale(dpr, dpr)
    ctx.lineCap = 'round'
    ctx.lineJoin = 'round'
    ctx.lineWidth = 2
    ctx.strokeStyle = 'currentColor'
  }, [width, height])

  useEffect(() => {
    setupCanvas()

    if (initialData) {
      const canvas = canvasRef.current
      if (!canvas) return

      const ctx = canvas.getContext('2d')
      if (!ctx) return

      const img = new Image()
      img.onload = () => {
        ctx.drawImage(img, 0, 0, width, height)
        setHasStrokes(true)
      }
      img.src = initialData
    }
  }, [setupCanvas, initialData, width, height])

  // Get coordinates from mouse or touch event
  const getPoint = useCallback(
    (e: React.MouseEvent | React.TouchEvent) => {
      const canvas = canvasRef.current
      if (!canvas) return null

      const rect = canvas.getBoundingClientRect()
      const scaleX = width / rect.width
      const scaleY = height / rect.height

      if ('touches' in e) {
        const touch = e.touches[0]
        if (!touch) return null
        return {
          x: (touch.clientX - rect.left) * scaleX,
          y: (touch.clientY - rect.top) * scaleY,
        }
      }

      return {
        x: (e.clientX - rect.left) * scaleX,
        y: (e.clientY - rect.top) * scaleY,
      }
    },
    [width, height]
  )

  const startDrawing = useCallback(
    (e: React.MouseEvent | React.TouchEvent) => {
      // Prevent scrolling on touch devices
      if ('touches' in e) e.preventDefault()

      const point = getPoint(e)
      if (!point) return

      setIsDrawing(true)
      setHasStrokes(true)
      lastPoint.current = point

      // Draw a dot for single taps/clicks
      const ctx = canvasRef.current?.getContext('2d')
      if (!ctx) return

      ctx.beginPath()
      ctx.arc(point.x, point.y, 1, 0, Math.PI * 2)
      ctx.fill()
    },
    [getPoint]
  )

  const draw = useCallback(
    (e: React.MouseEvent | React.TouchEvent) => {
      if (!isDrawing) return
      if ('touches' in e) e.preventDefault()

      const point = getPoint(e)
      if (!point || !lastPoint.current) return

      const ctx = canvasRef.current?.getContext('2d')
      if (!ctx) return

      ctx.beginPath()
      ctx.moveTo(lastPoint.current.x, lastPoint.current.y)
      ctx.lineTo(point.x, point.y)
      ctx.stroke()

      lastPoint.current = point
    },
    [isDrawing, getPoint]
  )

  const stopDrawing = useCallback(() => {
    setIsDrawing(false)
    lastPoint.current = null
  }, [])

  const handleClear = useCallback(() => {
    const canvas = canvasRef.current
    if (!canvas) return

    const ctx = canvas.getContext('2d')
    if (!ctx) return

    const dpr = window.devicePixelRatio || 1
    ctx.clearRect(0, 0, canvas.width / dpr, canvas.height / dpr)
    setHasStrokes(false)
  }, [])

  const handleSave = useCallback(() => {
    const canvas = canvasRef.current
    if (!canvas || !onSave) return

    onSave(canvas.toDataURL('image/png'))
  }, [onSave])

  return (
    <div className={`flex flex-col gap-2 ${className}`}>
      <div className="relative rounded-lg border border-border bg-white dark:bg-card overflow-hidden">
        {/* Hint text overlay */}
        {!hasStrokes && (
          <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
            <span className="text-sm text-muted-foreground/50 select-none">
              {t(hintKey)}
            </span>
          </div>
        )}

        <canvas
          ref={canvasRef}
          className="block cursor-crosshair"
          style={{ touchAction: 'none', width, height }}
          onMouseDown={startDrawing}
          onMouseMove={draw}
          onMouseUp={stopDrawing}
          onMouseLeave={stopDrawing}
          onTouchStart={startDrawing}
          onTouchMove={draw}
          onTouchEnd={stopDrawing}
        />
      </div>

      {/* Action buttons */}
      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={handleClear}
          className="inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground transition-colors hover:bg-secondary"
        >
          <Eraser className="h-3.5 w-3.5" />
          {t(clearKey)}
        </button>
        <button
          type="button"
          onClick={handleSave}
          disabled={!hasStrokes}
          className="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground transition-colors hover:bg-button-primary-hover disabled:opacity-50 disabled:pointer-events-none"
        >
          <Check className="h-3.5 w-3.5" />
          {t('common.save')}
        </button>
      </div>
    </div>
  )
}

import { useRef, useState, useEffect, useCallback } from 'react'
import { Pencil, Minus, Square, Type, Ruler, Undo2, Trash2, Save } from 'lucide-react'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type ToolType = 'freehand' | 'line' | 'rectangle' | 'text' | 'mass'

interface Point {
  x: number
  y: number
}

interface StrokeBase {
  color: string
  width: number
}

interface FreehandStroke extends StrokeBase {
  tool: 'freehand'
  points: Point[]
}

interface LineStroke extends StrokeBase {
  tool: 'line'
  start: Point
  end: Point
}

interface RectangleStroke extends StrokeBase {
  tool: 'rectangle'
  start: Point
  end: Point
}

interface TextStroke extends StrokeBase {
  tool: 'text'
  position: Point
  text: string
  fontSize: number
}

interface MassStroke extends StrokeBase {
  tool: 'mass'
  start: Point
  end: Point
  label: string
}

type Stroke = FreehandStroke | LineStroke | RectangleStroke | TextStroke | MassStroke

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const PRESET_COLORS = [
  { value: '#000000', label: 'Schwarz' },
  { value: '#dc2626', label: 'Rot' },
  { value: '#2563eb', label: 'Blau' },
  { value: '#16a34a', label: 'Gruen' },
]

const LINE_WIDTHS = [
  { value: 1, label: 'Duenn' },
  { value: 2, label: 'Mittel' },
  { value: 4, label: 'Dick' },
]

const TOOLS: { type: ToolType; label: string; icon: typeof Pencil }[] = [
  { type: 'freehand', label: 'Freihand', icon: Pencil },
  { type: 'line', label: 'Linie', icon: Minus },
  { type: 'rectangle', label: 'Rechteck', icon: Square },
  { type: 'text', label: 'Text', icon: Type },
  { type: 'mass', label: 'Mass', icon: Ruler },
]

const TEXT_FONT_SIZE = 16

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function getCanvasPoint(
  canvas: HTMLCanvasElement,
  clientX: number,
  clientY: number,
): Point {
  const rect = canvas.getBoundingClientRect()
  const dpr = window.devicePixelRatio || 1
  return {
    x: ((clientX - rect.left) / rect.width) * (canvas.width / dpr),
    y: ((clientY - rect.top) / rect.height) * (canvas.height / dpr),
  }
}

function pixelsToMeters(px: number): string {
  // 1 px = 1 cm = 0.01 m
  return (px * 0.01).toFixed(2)
}

function distance(a: Point, b: Point): number {
  return Math.sqrt((b.x - a.x) ** 2 + (b.y - a.y) ** 2)
}

function midpoint(a: Point, b: Point): Point {
  return { x: (a.x + b.x) / 2, y: (a.y + b.y) / 2 }
}

// ---------------------------------------------------------------------------
// Drawing helpers
// ---------------------------------------------------------------------------

function drawStroke(ctx: CanvasRenderingContext2D, stroke: Stroke) {
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'

  switch (stroke.tool) {
    case 'freehand': {
      if (stroke.points.length < 2) return
      ctx.strokeStyle = stroke.color
      ctx.lineWidth = stroke.width
      ctx.beginPath()
      ctx.moveTo(stroke.points[0].x, stroke.points[0].y)
      for (let i = 1; i < stroke.points.length; i++) {
        ctx.lineTo(stroke.points[i].x, stroke.points[i].y)
      }
      ctx.stroke()
      break
    }
    case 'line': {
      ctx.strokeStyle = stroke.color
      ctx.lineWidth = stroke.width
      ctx.beginPath()
      ctx.moveTo(stroke.start.x, stroke.start.y)
      ctx.lineTo(stroke.end.x, stroke.end.y)
      ctx.stroke()
      break
    }
    case 'rectangle': {
      ctx.strokeStyle = stroke.color
      ctx.lineWidth = stroke.width
      const w = stroke.end.x - stroke.start.x
      const h = stroke.end.y - stroke.start.y
      ctx.strokeRect(stroke.start.x, stroke.start.y, w, h)
      break
    }
    case 'text': {
      ctx.fillStyle = stroke.color
      ctx.font = `${stroke.fontSize}px sans-serif`
      ctx.textBaseline = 'top'
      ctx.fillText(stroke.text, stroke.position.x, stroke.position.y)
      break
    }
    case 'mass': {
      // Draw the line
      ctx.strokeStyle = stroke.color
      ctx.lineWidth = stroke.width
      ctx.beginPath()
      ctx.moveTo(stroke.start.x, stroke.start.y)
      ctx.lineTo(stroke.end.x, stroke.end.y)
      ctx.stroke()

      // Draw small perpendicular end caps
      const angle = Math.atan2(
        stroke.end.y - stroke.start.y,
        stroke.end.x - stroke.start.x,
      )
      const capLen = 6
      const perpX = Math.cos(angle + Math.PI / 2) * capLen
      const perpY = Math.sin(angle + Math.PI / 2) * capLen

      ctx.beginPath()
      ctx.moveTo(stroke.start.x - perpX, stroke.start.y - perpY)
      ctx.lineTo(stroke.start.x + perpX, stroke.start.y + perpY)
      ctx.stroke()

      ctx.beginPath()
      ctx.moveTo(stroke.end.x - perpX, stroke.end.y - perpY)
      ctx.lineTo(stroke.end.x + perpX, stroke.end.y + perpY)
      ctx.stroke()

      // Draw measurement label at midpoint
      const mid = midpoint(stroke.start, stroke.end)
      ctx.fillStyle = stroke.color
      ctx.font = 'bold 12px sans-serif'
      ctx.textAlign = 'center'
      ctx.textBaseline = 'bottom'
      ctx.fillText(stroke.label, mid.x, mid.y - 4)
      ctx.textAlign = 'start'
      ctx.textBaseline = 'alphabetic'
      break
    }
  }
}

function redrawCanvas(
  ctx: CanvasRenderingContext2D,
  width: number,
  height: number,
  strokes: Stroke[],
) {
  ctx.clearRect(0, 0, width, height)
  for (const stroke of strokes) {
    drawStroke(ctx, stroke)
  }
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface SketchCanvasProps {
  width?: number
  height?: number
  className?: string
  onSave?: (dataUrl: string) => void
}

export default function SketchCanvas({
  width = 600,
  height = 400,
  className = '',
  onSave,
}: SketchCanvasProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  // Drawing state
  const [strokes, setStrokes] = useState<Stroke[]>([])
  const [activeTool, setActiveTool] = useState<ToolType>('freehand')
  const [activeColor, setActiveColor] = useState(PRESET_COLORS[0].value)
  const [activeWidth, setActiveWidth] = useState(LINE_WIDTHS[1].value)

  // In-progress drawing state
  const isDrawingRef = useRef(false)
  const currentPointsRef = useRef<Point[]>([])
  const startPointRef = useRef<Point | null>(null)

  // Text input overlay
  const [textInput, setTextInput] = useState<{
    visible: boolean
    position: Point
    value: string
  }>({ visible: false, position: { x: 0, y: 0 }, value: '' })

  // ---------------------------------------------------------------------------
  // Canvas setup with device pixel ratio
  // ---------------------------------------------------------------------------

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const dpr = window.devicePixelRatio || 1
    canvas.width = width * dpr
    canvas.height = height * dpr
    canvas.style.width = `${width}px`
    canvas.style.height = `${height}px`
    const ctx = canvas.getContext('2d')
    if (ctx) {
      ctx.scale(dpr, dpr)
      redrawCanvas(ctx, width, height, strokes)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [width, height])

  // ---------------------------------------------------------------------------
  // Redraw whenever strokes change
  // ---------------------------------------------------------------------------

  const redraw = useCallback(
    (extraStroke?: Stroke) => {
      const canvas = canvasRef.current
      if (!canvas) return
      const ctx = canvas.getContext('2d')
      if (!ctx) return
      const all = extraStroke ? [...strokes, extraStroke] : strokes
      redrawCanvas(ctx, width, height, all)
    },
    [strokes, width, height],
  )

  useEffect(() => {
    redraw()
  }, [redraw])

  // ---------------------------------------------------------------------------
  // Pointer event handlers
  // ---------------------------------------------------------------------------

  const handlePointerDown = useCallback(
    (clientX: number, clientY: number) => {
      const canvas = canvasRef.current
      if (!canvas) return

      // If text input is visible, ignore canvas clicks (let input handle itself)
      if (textInput.visible) return

      const point = getCanvasPoint(canvas, clientX, clientY)

      if (activeTool === 'text') {
        // Show text input overlay at click position
        const _rect = canvas.getBoundingClientRect()
        setTextInput({
          visible: true,
          position: point,
          value: '',
        })
        return
      }

      isDrawingRef.current = true
      startPointRef.current = point

      if (activeTool === 'freehand') {
        currentPointsRef.current = [point]
      }
    },
    [activeTool, textInput.visible],
  )

  const handlePointerMove = useCallback(
    (clientX: number, clientY: number) => {
      if (!isDrawingRef.current) return
      const canvas = canvasRef.current
      if (!canvas) return

      const point = getCanvasPoint(canvas, clientX, clientY)

      if (activeTool === 'freehand') {
        currentPointsRef.current.push(point)
        // Draw incrementally for smooth freehand
        const ctx = canvas.getContext('2d')
        if (!ctx) return
        const pts = currentPointsRef.current
        if (pts.length < 2) return
        ctx.strokeStyle = activeColor
        ctx.lineWidth = activeWidth
        ctx.lineCap = 'round'
        ctx.lineJoin = 'round'
        ctx.beginPath()
        ctx.moveTo(pts[pts.length - 2].x, pts[pts.length - 2].y)
        ctx.lineTo(pts[pts.length - 1].x, pts[pts.length - 1].y)
        ctx.stroke()
      } else if (
        activeTool === 'line' ||
        activeTool === 'rectangle' ||
        activeTool === 'mass'
      ) {
        // Show live preview by redrawing history + preview shape
        const start = startPointRef.current
        if (!start) return

        let previewStroke: Stroke
        if (activeTool === 'line') {
          previewStroke = {
            tool: 'line',
            start,
            end: point,
            color: activeColor,
            width: activeWidth,
          }
        } else if (activeTool === 'rectangle') {
          previewStroke = {
            tool: 'rectangle',
            start,
            end: point,
            color: activeColor,
            width: activeWidth,
          }
        } else {
          const dist = distance(start, point)
          previewStroke = {
            tool: 'mass',
            start,
            end: point,
            color: activeColor,
            width: activeWidth,
            label: `${pixelsToMeters(dist)} m`,
          }
        }
        redraw(previewStroke)
      }
    },
    [activeTool, activeColor, activeWidth, redraw],
  )

  const handlePointerUp = useCallback(
    (clientX: number, clientY: number) => {
      if (!isDrawingRef.current) return
      isDrawingRef.current = false

      const canvas = canvasRef.current
      if (!canvas) return
      const point = getCanvasPoint(canvas, clientX, clientY)
      const start = startPointRef.current

      if (activeTool === 'freehand') {
        const pts = [...currentPointsRef.current]
        if (pts.length >= 2) {
          setStrokes((prev) => [
            ...prev,
            {
              tool: 'freehand',
              points: pts,
              color: activeColor,
              width: activeWidth,
            },
          ])
        }
        currentPointsRef.current = []
      } else if (activeTool === 'line' && start) {
        setStrokes((prev) => [
          ...prev,
          {
            tool: 'line',
            start,
            end: point,
            color: activeColor,
            width: activeWidth,
          },
        ])
      } else if (activeTool === 'rectangle' && start) {
        setStrokes((prev) => [
          ...prev,
          {
            tool: 'rectangle',
            start,
            end: point,
            color: activeColor,
            width: activeWidth,
          },
        ])
      } else if (activeTool === 'mass' && start) {
        const dist = distance(start, point)
        setStrokes((prev) => [
          ...prev,
          {
            tool: 'mass',
            start,
            end: point,
            color: activeColor,
            width: activeWidth,
            label: `${pixelsToMeters(dist)} m`,
          },
        ])
      }

      startPointRef.current = null
    },
    [activeTool, activeColor, activeWidth],
  )

  // ---------------------------------------------------------------------------
  // Mouse event wrappers
  // ---------------------------------------------------------------------------

  const onMouseDown = useCallback(
    (e: React.MouseEvent) => handlePointerDown(e.clientX, e.clientY),
    [handlePointerDown],
  )
  const onMouseMove = useCallback(
    (e: React.MouseEvent) => handlePointerMove(e.clientX, e.clientY),
    [handlePointerMove],
  )
  const onMouseUp = useCallback(
    (e: React.MouseEvent) => handlePointerUp(e.clientX, e.clientY),
    [handlePointerUp],
  )
  const onMouseLeave = useCallback(
    (e: React.MouseEvent) => {
      if (isDrawingRef.current) handlePointerUp(e.clientX, e.clientY)
    },
    [handlePointerUp],
  )

  // ---------------------------------------------------------------------------
  // Touch event wrappers
  // ---------------------------------------------------------------------------

  const onTouchStart = useCallback(
    (e: React.TouchEvent) => {
      if (e.touches.length !== 1) return
      const t = e.touches[0]
      handlePointerDown(t.clientX, t.clientY)
    },
    [handlePointerDown],
  )
  const onTouchMove = useCallback(
    (e: React.TouchEvent) => {
      if (e.touches.length !== 1) return
      const t = e.touches[0]
      handlePointerMove(t.clientX, t.clientY)
    },
    [handlePointerMove],
  )
  const onTouchEnd = useCallback(
    (e: React.TouchEvent) => {
      if (e.changedTouches.length !== 1) return
      const t = e.changedTouches[0]
      handlePointerUp(t.clientX, t.clientY)
    },
    [handlePointerUp],
  )

  // ---------------------------------------------------------------------------
  // Text input commit
  // ---------------------------------------------------------------------------

  const commitText = useCallback(() => {
    if (!textInput.value.trim()) {
      setTextInput((prev) => ({ ...prev, visible: false }))
      return
    }
    setStrokes((prev) => [
      ...prev,
      {
        tool: 'text',
        position: textInput.position,
        text: textInput.value,
        color: activeColor,
        width: activeWidth,
        fontSize: TEXT_FONT_SIZE,
      },
    ])
    setTextInput({ visible: false, position: { x: 0, y: 0 }, value: '' })
  }, [textInput, activeColor, activeWidth])

  const onTextKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter') {
        e.preventDefault()
        commitText()
      } else if (e.key === 'Escape') {
        setTextInput({ visible: false, position: { x: 0, y: 0 }, value: '' })
      }
    },
    [commitText],
  )

  // ---------------------------------------------------------------------------
  // Actions
  // ---------------------------------------------------------------------------

  const handleUndo = useCallback(() => {
    setStrokes((prev) => {
      if (prev.length === 0) return prev
      return prev.slice(0, -1)
    })
  }, [])

  const handleClear = useCallback(() => {
    setStrokes([])
    setTextInput({ visible: false, position: { x: 0, y: 0 }, value: '' })
  }, [])

  const handleSave = useCallback(() => {
    const canvas = canvasRef.current
    if (!canvas || !onSave) return
    const dataUrl = canvas.toDataURL('image/png')
    onSave(dataUrl)
  }, [onSave])

  // ---------------------------------------------------------------------------
  // Compute text input overlay position relative to canvas container
  // ---------------------------------------------------------------------------

  const textOverlayStyle: React.CSSProperties | undefined = textInput.visible
    ? {
        position: 'absolute' as const,
        left: textInput.position.x,
        top: textInput.position.y,
        zIndex: 10,
      }
    : undefined

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------

  return (
    <div className={className}>
      {/* Toolbar */}
      <div className="flex flex-wrap items-center gap-2 rounded-lg border border-border bg-card p-2 mb-2">
        {/* Tool buttons */}
        <div className="flex items-center gap-1">
          {TOOLS.map(({ type, label, icon: Icon }) => (
            <button
              key={type}
              type="button"
              title={label}
              onClick={() => {
                setActiveTool(type)
                if (textInput.visible) commitText()
              }}
              className={`flex items-center gap-1 rounded-lg px-2 py-1.5 text-xs transition-colors ${
                activeTool === type
                  ? 'bg-primary text-primary-foreground'
                  : 'hover:bg-muted'
              }`}
            >
              <Icon className="h-3.5 w-3.5" />
              <span className="hidden sm:inline">{label}</span>
            </button>
          ))}
        </div>

        {/* Divider */}
        <div className="h-6 w-px bg-border" />

        {/* Color picker */}
        <div className="flex items-center gap-1.5">
          {PRESET_COLORS.map(({ value, label }) => (
            <button
              key={value}
              type="button"
              title={label}
              onClick={() => setActiveColor(value)}
              className={`h-5 w-5 rounded-full border-2 transition-shadow ${
                activeColor === value
                  ? 'ring-2 ring-primary ring-offset-1 border-transparent'
                  : 'border-border'
              }`}
              style={{ backgroundColor: value }}
            />
          ))}
        </div>

        {/* Divider */}
        <div className="h-6 w-px bg-border" />

        {/* Line width */}
        <div className="flex items-center gap-1">
          {LINE_WIDTHS.map(({ value, label }) => (
            <button
              key={value}
              type="button"
              title={label}
              onClick={() => setActiveWidth(value)}
              className={`rounded-lg px-2 py-1.5 text-xs transition-colors ${
                activeWidth === value
                  ? 'bg-primary text-primary-foreground'
                  : 'hover:bg-muted'
              }`}
            >
              {label}
            </button>
          ))}
        </div>

        {/* Spacer */}
        <div className="flex-1" />

        {/* Action buttons */}
        <div className="flex items-center gap-1">
          <button
            type="button"
            title="Rueckgaengig"
            onClick={handleUndo}
            disabled={strokes.length === 0}
            className="flex items-center gap-1 rounded-lg px-2 py-1.5 text-xs hover:bg-muted disabled:opacity-40 disabled:pointer-events-none"
          >
            <Undo2 className="h-3.5 w-3.5" />
            <span className="hidden sm:inline">Rueckgaengig</span>
          </button>
          <button
            type="button"
            title="Alles loeschen"
            onClick={handleClear}
            disabled={strokes.length === 0}
            className="flex items-center gap-1 rounded-lg px-2 py-1.5 text-xs text-destructive hover:bg-destructive/10 disabled:opacity-40 disabled:pointer-events-none"
          >
            <Trash2 className="h-3.5 w-3.5" />
            <span className="hidden sm:inline">Loeschen</span>
          </button>
          {onSave && (
            <button
              type="button"
              title="Speichern"
              onClick={handleSave}
              className="flex items-center gap-1 rounded-lg bg-primary px-2 py-1.5 text-xs text-primary-foreground hover:bg-primary/90"
            >
              <Save className="h-3.5 w-3.5" />
              <span className="hidden sm:inline">Speichern</span>
            </button>
          )}
        </div>
      </div>

      {/* Canvas container */}
      <div ref={containerRef} className="relative inline-block">
        <canvas
          ref={canvasRef}
          className="rounded-lg border border-border bg-white dark:bg-gray-900 cursor-crosshair"
          style={{ touchAction: 'none' }}
          onMouseDown={onMouseDown}
          onMouseMove={onMouseMove}
          onMouseUp={onMouseUp}
          onMouseLeave={onMouseLeave}
          onTouchStart={onTouchStart}
          onTouchMove={onTouchMove}
          onTouchEnd={onTouchEnd}
        />

        {/* Text input overlay */}
        {textInput.visible && textOverlayStyle && (
          <input
            type="text"
            autoFocus
            value={textInput.value}
            onChange={(e) =>
              setTextInput((prev) => ({ ...prev, value: e.target.value }))
            }
            onKeyDown={onTextKeyDown}
            onBlur={commitText}
            placeholder="Text eingeben..."
            className="absolute rounded border border-primary bg-white px-1 py-0.5 text-sm text-black shadow-sm outline-none dark:bg-gray-800 dark:text-white"
            style={{
              left: textInput.position.x,
              top: textInput.position.y,
              zIndex: 10,
              minWidth: 120,
              fontSize: TEXT_FONT_SIZE,
            }}
          />
        )}
      </div>
    </div>
  )
}

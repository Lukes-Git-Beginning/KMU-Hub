import { useState } from 'react'
import { SWATCH_COLORS } from '@/lib/swatch-colors'

interface ColorSwatchPickerProps {
  value: string
  onChange: (color: string) => void
  colors?: string[]
  /** Diameter of the trigger dot in px. */
  size?: number
}

/**
 * ColorSwatchPicker — a small colour dot that opens a swatch palette popover.
 * Reused across pipeline-stage and tag editors.
 */
export function ColorSwatchPicker({ value, onChange, colors = SWATCH_COLORS, size = 20 }: ColorSwatchPickerProps) {
  const [open, setOpen] = useState(false)
  return (
    <div className="relative shrink-0">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-label="Farbe"
        className="rounded-full border border-border"
        style={{ backgroundColor: value, width: size, height: size }}
      />
      {open && (
        <>
          <div className="fixed inset-0 z-10" onClick={() => setOpen(false)} />
          <div className="absolute left-0 top-7 z-20 grid grid-cols-5 gap-1.5 rounded-lg border border-border bg-card p-2 shadow-lg">
            {colors.map((c) => (
              <button
                key={c}
                type="button"
                onClick={() => { onChange(c); setOpen(false) }}
                className="h-5 w-5 rounded-full border border-border transition-transform hover:scale-110"
                style={{ backgroundColor: c }}
              />
            ))}
          </div>
        </>
      )}
    </div>
  )
}

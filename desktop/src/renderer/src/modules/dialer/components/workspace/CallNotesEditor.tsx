import { useEffect, useRef } from 'react'
import { cn } from '@/lib/cn'

interface CallNotesEditorProps {
  value: string
  onChange: (value: string) => void
  onAutoSave?: (value: string) => void
  className?: string
}

export default function CallNotesEditor({
  value,
  onChange,
  onAutoSave,
  className,
}: CallNotesEditorProps) {
  const debounceRef = useRef<ReturnType<typeof setTimeout>>()

  useEffect(() => {
    if (!onAutoSave) return

    if (debounceRef.current) clearTimeout(debounceRef.current)
    debounceRef.current = setTimeout(() => {
      if (value.trim()) onAutoSave(value)
    }, 1500)

    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current)
    }
  }, [value, onAutoSave])

  return (
    <div className={cn('space-y-1', className)}>
      <textarea
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder="Notizen zum Gespräch..."
        className="w-full min-h-[120px] resize-y rounded-lg border border-border bg-card px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/30 transition-shadow"
      />
      <div className="flex items-center justify-end">
        <span className="text-xs text-muted-foreground tabular-nums">
          {value.length} Zeichen
        </span>
      </div>
    </div>
  )
}

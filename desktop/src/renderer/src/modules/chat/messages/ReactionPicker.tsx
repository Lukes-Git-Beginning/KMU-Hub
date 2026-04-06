/**
 * Small floating emoji grid for adding reactions.
 *
 * Shows commonly used emojis grouped by category.
 * Click selects the emoji and closes the picker.
 */
import { useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'

interface ReactionPickerProps {
  onSelect: (emoji: string) => void
  onClose: () => void
}

const EMOJI_CATEGORIES = [
  {
    labelKey: 'chat.reactions.frequent',
    emojis: [
      '\u{1F44D}', '\u{1F44E}', '\u{2764}\u{FE0F}', '\u{1F604}',
      '\u{1F389}', '\u{1F4AF}', '\u{1F525}', '\u{1F44F}',
    ],
  },
  {
    labelKey: 'chat.reactions.smileys',
    emojis: [
      '\u{1F600}', '\u{1F602}', '\u{1F60D}', '\u{1F914}',
      '\u{1F631}', '\u{1F622}', '\u{1F60E}', '\u{1F64F}',
    ],
  },
  {
    labelKey: 'chat.reactions.gestures',
    emojis: [
      '\u{1F44C}', '\u{270C}\u{FE0F}', '\u{1F4AA}', '\u{1F91D}',
      '\u{1F64C}', '\u{1F91E}', '\u{1F448}', '\u{1F449}',
    ],
  },
] as const

export function ReactionPicker({ onSelect, onClose }: ReactionPickerProps) {
  const { t } = useTranslation()
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        onClose()
      }
    }
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('mousedown', handleClickOutside)
    document.addEventListener('keydown', handleEscape)
    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
      document.removeEventListener('keydown', handleEscape)
    }
  }, [onClose])

  return (
    <div
      ref={ref}
      className="absolute bottom-full right-0 z-50 mb-1 w-[240px] rounded-lg border border-border bg-card p-2 shadow-lg"
    >
      {EMOJI_CATEGORIES.map((cat) => (
        <div key={cat.labelKey} className="mb-1 last:mb-0">
          <p className="mb-1 px-1 text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
            {t(cat.labelKey)}
          </p>
          <div className="grid grid-cols-8 gap-0.5">
            {cat.emojis.map((emoji) => (
              <button
                key={emoji}
                onClick={() => onSelect(emoji)}
                className="flex h-7 w-7 items-center justify-center rounded text-base transition-colors hover:bg-secondary"
              >
                {emoji}
              </button>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}

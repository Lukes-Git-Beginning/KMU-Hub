/**
 * Per-message emoji reaction picker using frimousse.
 *
 * Each message gets its own Popover instance (no global singleton).
 * Wraps the frimousse EmojiPicker in a Radix Popover, triggered by
 * a smiley face icon. Includes a quick-reactions bar for the most
 * commonly used emoji without needing to open the full picker.
 */
import { useState } from 'react'
import { Smile } from 'lucide-react'
import { EmojiPicker, type Emoji } from 'frimousse'
import { useToggleReaction } from '@/api/hooks/useReactions'
import { Button } from '@/components/ui/button'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

interface ReactionPickerProps {
  messageId: string
}

/** Frequently used emoji for the quick-reaction bar. */
const QUICK_REACTIONS = [
  { emoji: '\uD83D\uDC4D', label: 'Daumen hoch' },
  { emoji: '\u2764\uFE0F', label: 'Herz' },
  { emoji: '\uD83D\uDE02', label: 'Traenen lachend' },
  { emoji: '\uD83D\uDC4F', label: 'Klatschen' },
  { emoji: '\uD83E\uDD14', label: 'Nachdenklich' },
  { emoji: '\uD83D\uDE80', label: 'Rakete' },
]

export function ReactionPicker({ messageId }: ReactionPickerProps) {
  const [open, setOpen] = useState(false)
  const toggleReaction = useToggleReaction()

  const handleSelect = (emoji: string) => {
    toggleReaction.mutate({ message_id: messageId, emoji })
    setOpen(false)
  }

  const handleEmojiSelect = (emojiData: Emoji) => {
    handleSelect(emojiData.emoji)
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <Tooltip>
        <TooltipTrigger asChild>
          <PopoverTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="h-6 w-6"
              aria-label="Reaktion hinzufuegen"
            >
              <Smile className="h-3.5 w-3.5" />
            </Button>
          </PopoverTrigger>
        </TooltipTrigger>
        <TooltipContent>Reaktion</TooltipContent>
      </Tooltip>

      <PopoverContent
        className="w-[320px] p-0"
        side="top"
        align="end"
        sideOffset={4}
      >
        {/* Quick reactions bar */}
        <div className="flex items-center gap-1 border-b border-border px-2 py-1.5">
          {QUICK_REACTIONS.map((qr) => (
            <button
              key={qr.emoji}
              className="rounded p-1 text-lg transition-colors hover:bg-accent"
              onClick={() => handleSelect(qr.emoji)}
              title={qr.label}
              aria-label={qr.label}
            >
              {qr.emoji}
            </button>
          ))}
        </div>

        {/* Full emoji picker */}
        <EmojiPicker.Root
          onEmojiSelect={handleEmojiSelect}
          locale="de"
          columns={8}
          className="h-[280px]"
        >
          <EmojiPicker.Search
            className="mx-2 mt-2 mb-1 h-8 w-[calc(100%-1rem)] rounded-md border border-input bg-transparent px-3 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring"
            placeholder="Emoji suchen..."
            aria-label="Emoji suchen"
            autoFocus
          />
          <EmojiPicker.Viewport className="relative flex-1 overflow-y-auto px-1">
            <EmojiPicker.Loading>
              <span className="flex h-full items-center justify-center text-sm text-muted-foreground">
                Laden...
              </span>
            </EmojiPicker.Loading>
            <EmojiPicker.Empty>
              <span className="flex h-full items-center justify-center text-sm text-muted-foreground">
                Kein Emoji gefunden.
              </span>
            </EmojiPicker.Empty>
            <EmojiPicker.List
              components={{
                CategoryHeader: ({ category, ...props }) => (
                  <div
                    {...props}
                    className="sticky top-0 z-10 bg-popover px-1 py-1 text-xs font-semibold text-muted-foreground"
                  >
                    {category.label}
                  </div>
                ),
                Row: ({ children, ...props }) => (
                  <div {...props} className="flex">
                    {children}
                  </div>
                ),
                Emoji: ({ emoji, ...props }) => (
                  <button
                    {...props}
                    className="flex h-8 w-8 items-center justify-center rounded text-lg transition-colors hover:bg-accent data-[active]:bg-accent"
                    data-active={emoji.isActive || undefined}
                  >
                    {emoji.emoji}
                  </button>
                ),
              }}
            />
          </EmojiPicker.Viewport>
        </EmojiPicker.Root>
      </PopoverContent>
    </Popover>
  )
}

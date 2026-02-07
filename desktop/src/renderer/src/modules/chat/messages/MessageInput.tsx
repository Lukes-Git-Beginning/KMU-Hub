/**
 * Message composition input for chat channels.
 *
 * Features: auto-growing textarea, Enter to send, Shift+Enter for newline,
 * typing indicator emission via WebSocket, and send button.
 */
import { useState, useRef, useCallback } from 'react'
import { Send, Paperclip } from 'lucide-react'
import { useSendMessage, useSendTypingIndicator } from '@/api/hooks/useMessages'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

interface MessageInputProps {
  channelId: string
  parentMessageId?: string
  placeholder?: string
}

export function MessageInput({ channelId, parentMessageId, placeholder }: MessageInputProps) {
  const [content, setContent] = useState('')
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const sendMessage = useSendMessage()
  const sendTyping = useSendTypingIndicator(channelId)

  const handleSend = useCallback(async () => {
    const trimmed = content.trim()
    if (!trimmed) return

    try {
      await sendMessage.mutateAsync({
        channelId,
        content: trimmed,
        parentMessageId,
      })
      setContent('')

      // Reset textarea height
      if (textareaRef.current) {
        textareaRef.current.style.height = 'auto'
      }
    } catch {
      // Error is handled by useMutation
    }
  }, [content, channelId, parentMessageId, sendMessage])

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault()
        handleSend()
      }
    },
    [handleSend]
  )

  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLTextAreaElement>) => {
      setContent(e.target.value)
      sendTyping()

      // Auto-resize textarea (max 6 lines ~ 144px)
      const textarea = e.target
      textarea.style.height = 'auto'
      textarea.style.height = `${Math.min(textarea.scrollHeight, 144)}px`
    },
    [sendTyping]
  )

  return (
    <div className="border-t border-border bg-card p-3">
      <div className="flex items-end gap-2">
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="h-9 w-9 shrink-0"
              disabled
            >
              <Paperclip className="h-4 w-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>Datei anhaengen (bald verfuegbar)</TooltipContent>
        </Tooltip>

        <Textarea
          ref={textareaRef}
          value={content}
          onChange={handleChange}
          onKeyDown={handleKeyDown}
          placeholder={placeholder ?? 'Nachricht schreiben...'}
          className="min-h-[36px] max-h-[144px] resize-none"
          rows={1}
        />

        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              size="icon"
              className="h-9 w-9 shrink-0"
              disabled={!content.trim() || sendMessage.isPending}
              onClick={handleSend}
            >
              <Send className="h-4 w-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>Senden (Enter)</TooltipContent>
        </Tooltip>
      </div>
    </div>
  )
}

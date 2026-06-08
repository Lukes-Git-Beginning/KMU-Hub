/**
 * Message composition input for chat channels.
 *
 * Features: auto-growing textarea, Enter to send, Shift+Enter for newline,
 * typing indicator emission via WebSocket, and send button.
 */
import { useState, useRef, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { Send, Paperclip } from 'lucide-react'
import { toast } from 'sonner'
import { useSendMessage, useSendTypingIndicator } from '@/api/hooks/useMessages'
import { useChatFileUpload } from '@/api/hooks/useChatFileUpload'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { MentionAutocomplete } from './MentionAutocomplete'
import { FileDropZone, fileToAttachment, type AttachedFile } from './FileDropZone'
import { FileAttachmentCard } from './FileAttachmentCard'

interface MessageInputProps {
  channelId: string
  parentMessageId?: string
  placeholder?: string
}

export function MessageInput({ channelId, parentMessageId, placeholder }: MessageInputProps) {
  const { t } = useTranslation()
  const [content, setContent] = useState('')
  const [mentionQuery, setMentionQuery] = useState<string | null>(null)
  const [pendingFiles, setPendingFiles] = useState<AttachedFile[]>([])
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const sendMessage = useSendMessage()
  const uploadFile = useChatFileUpload()
  const sendTyping = useSendTypingIndicator(channelId)

  const handleSend = useCallback(async () => {
    const trimmed = content.trim()
    if (!trimmed && pendingFiles.length === 0) return

    const filesToUpload = pendingFiles
    try {
      const result = await sendMessage.mutateAsync({
        channelId,
        content: trimmed || ' ',
        parentMessageId,
      })
      setContent('')
      setPendingFiles([])
      if (textareaRef.current) {
        textareaRef.current.style.height = 'auto'
      }

      // Attach files to the just-created message (real upload endpoint).
      const messageId = (result as { message?: { id?: string } } | undefined)?.message?.id
      for (const f of filesToUpload) {
        if (f.raw) {
          await uploadFile.mutateAsync({ channelId, file: f.raw, messageId }).catch((err) => {
            toast.error(t('chat.input.uploadFailed', { name: f.name }))
            throw err
          })
        }
      }
    } catch {
      // Error is handled by useMutation / toast above
    }
  }, [content, pendingFiles, channelId, parentMessageId, sendMessage, uploadFile, t])

  const handleFilePick = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files ?? [])
    if (files.length > 0) {
      setPendingFiles((prev) => [...prev, ...files.map(fileToAttachment)])
    }
    e.target.value = ''
  }, [])

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
      const val = e.target.value
      setContent(val)
      sendTyping()

      // Detect @mention trigger
      const cursorPos = e.target.selectionStart
      const textUpToCursor = val.slice(0, cursorPos)
      const mentionMatch = textUpToCursor.match(/@(\w*)$/)
      setMentionQuery(mentionMatch ? mentionMatch[1] : null)

      // Auto-resize textarea (max 6 lines ~ 144px)
      const textarea = e.target
      textarea.style.height = 'auto'
      textarea.style.height = `${Math.min(textarea.scrollHeight, 144)}px`
    },
    [sendTyping]
  )

  const handleMentionSelect = useCallback(
    (displayName: string) => {
      const cursorPos = textareaRef.current?.selectionStart ?? content.length
      const textUpToCursor = content.slice(0, cursorPos)
      const beforeMention = textUpToCursor.replace(/@\w*$/, '')
      const afterCursor = content.slice(cursorPos)
      const newContent = `${beforeMention}@${displayName} ${afterCursor}`
      setContent(newContent)
      setMentionQuery(null)

      // Refocus textarea
      requestAnimationFrame(() => {
        if (textareaRef.current) {
          const pos = beforeMention.length + displayName.length + 2
          textareaRef.current.focus()
          textareaRef.current.setSelectionRange(pos, pos)
        }
      })
    },
    [content]
  )

  const handleAttachClick = useCallback(() => {
    fileInputRef.current?.click()
  }, [])

  const handleFilesAdded = useCallback((files: AttachedFile[]) => {
    setPendingFiles((prev) => [...prev, ...files])
  }, [])

  const handleRemoveFile = useCallback((fileId: string) => {
    setPendingFiles((prev) => prev.filter((f) => f.id !== fileId))
  }, [])

  return (
    <FileDropZone onFilesAdded={handleFilesAdded}>
      <div className="border-t border-border bg-card p-3">
        {/* Pending file attachments */}
        {pendingFiles.length > 0 && (
          <div className="mb-2 flex flex-wrap gap-2">
            {pendingFiles.map((file) => (
              <FileAttachmentCard
                key={file.id}
                file={file}
                compact
                onRemove={() => handleRemoveFile(file.id)}
              />
            ))}
          </div>
        )}

        <div className="relative flex items-end gap-2">
          {/* @Mention autocomplete */}
          {mentionQuery !== null && (
            <MentionAutocomplete
              query={mentionQuery}
              onSelect={handleMentionSelect}
              onClose={() => setMentionQuery(null)}
            />
          )}

          <input
            ref={fileInputRef}
            type="file"
            multiple
            className="hidden"
            onChange={handleFilePick}
          />
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="h-9 w-9 shrink-0"
                onClick={handleAttachClick}
              >
                <Paperclip className="h-4 w-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>{t('chat.input.attachFile')}</TooltipContent>
          </Tooltip>

          <Textarea
            ref={textareaRef}
            value={content}
            onChange={handleChange}
            onKeyDown={handleKeyDown}
            placeholder={placeholder ?? t('chat.input.placeholder')}
            className="min-h-[36px] max-h-[144px] resize-none"
            rows={1}
          />

          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                size="icon"
                className="h-9 w-9 shrink-0"
                disabled={(!content.trim() && pendingFiles.length === 0) || sendMessage.isPending}
                onClick={handleSend}
              >
                <Send className="h-4 w-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>{t('chat.input.send')}</TooltipContent>
          </Tooltip>
        </div>
      </div>
    </FileDropZone>
  )
}

import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Send, Paperclip, ChevronDown } from 'lucide-react'
import { toast } from 'sonner'
import type { CommunicationChannel } from '@/types/communication'
import { CannedResponsePicker } from './CannedResponsePicker'
import { InternalNoteComposer } from './InternalNoteComposer'
import { MentionTextarea } from './MentionTextarea'
import { SlashCommandPalette, type SlashCommand } from './SlashCommandPalette'

// ---------------------------------------------------------------------------
// Channel config
// ---------------------------------------------------------------------------

const channelLabelKeys: Record<CommunicationChannel, string> = {
  email: 'kommunikation.reply.channelEmail',
  teams: 'kommunikation.reply.channelTeams',
  whatsapp: 'kommunikation.reply.channelWhatsapp',
  widget: 'kommunikation.reply.channelWidget',
  portal: 'kommunikation.reply.channelPortal',
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface ReplyComposerProps {
  channel: CommunicationChannel
  onSendReply: (content: string) => void
  onSendInternalNote: (content: string) => void
  /** Handle a chosen slash command (poll/reminder are real; giphy is a stub). */
  onSlashCommand?: (cmd: SlashCommand) => void
}

export function ReplyComposer({
  channel,
  onSendReply,
  onSendInternalNote,
  onSlashCommand,
}: ReplyComposerProps) {
  const { t } = useTranslation()
  const [text, setText] = useState('')

  const handleSend = () => {
    if (!text.trim()) return
    onSendReply(text.trim())
    setText('')
  }

  const handleCannedResponse = (content: string) => {
    setText((prev) => (prev ? prev + '\n' + content : content))
  }

  // Slash-command palette opens while the whole input is a `/command` being typed
  const slashMatch = text.match(/^\/(\w*)$/)
  const slashQuery = slashMatch ? slashMatch[1] : null

  const handleSlashSelect = (cmd: SlashCommand) => {
    setText('')
    if (onSlashCommand) {
      onSlashCommand(cmd)
    } else {
      toast.info(t('kommunikation.slash.executed', { command: cmd.name }))
    }
  }

  return (
    <div className="border-t border-border px-4 py-3 space-y-2">
      {/* Reply area */}
      <div className="rounded-lg border border-border bg-background focus-within:border-primary transition-colors">
        {/* Channel indicator */}
        <div className="flex items-center gap-2 px-3 py-1.5 border-b border-border/50">
          <span className="text-[10px] font-medium text-muted-foreground">
            {t(channelLabelKeys[channel])}
          </span>
          <ChevronDown className="h-3 w-3 text-muted-foreground" />
        </div>

        {/* Text input (with @mention + /slash support) */}
        <div className="relative px-3 py-2">
          {slashQuery !== null && (
            <SlashCommandPalette
              query={slashQuery}
              onSelect={handleSlashSelect}
              onClose={() => setText('')}
            />
          )}
          <MentionTextarea
            value={text}
            onChange={setText}
            onSubmit={handleSend}
            placeholder={t('kommunikation.reply.placeholder')}
            rows={3}
            className="w-full bg-transparent text-sm text-foreground outline-none placeholder:text-muted-foreground resize-none"
          />
        </div>

        {/* Toolbar */}
        <div className="flex items-center justify-between px-2 py-1.5 border-t border-border/50">
          <div className="flex items-center gap-0.5">
            {/* Canned responses */}
            <CannedResponsePicker onSelect={handleCannedResponse} />

            {/* Attachment */}
            <button
              className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
              title={t('kommunikation.reply.attachFile')}
            >
              <Paperclip className="h-4 w-4" />
            </button>
          </div>

          {/* Send */}
          <button
            onClick={handleSend}
            disabled={!text.trim()}
            className="flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
          >
            <Send className="h-3.5 w-3.5" />
            {t('kommunikation.reply.send')}
          </button>
        </div>
      </div>

      {/* Internal note toggle */}
      <InternalNoteComposer onSend={onSendInternalNote} />
    </div>
  )
}
